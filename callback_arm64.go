//go:build linux || darwin

package cgo

import (
	"reflect"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// cbEntrySize is the size of one entry in goexlib_cgo_callbackasm
// (a MOVD plus a B instruction).
const cbEntrySize = 8

// cbBackendSupported reports whether purego.NewCallback can handle the
// signature. purego supports integer-like and float arguments, but only
// integer-like (or no) return values.
func cbBackendSupported(ty reflect.Type) bool {
	if ty.NumOut() == 0 {
		return true
	}
	switch ty.Out(0).Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
		return true
	}
	return false
}

func cbBackendNewCallback(fn any) uintptr {
	return purego.NewCallback(fn)
}

func cbNewDispatchTrampoline() uintptr {
	return purego.NewCallback(cbDispatch)
}

// buildCbArgPlan assigns the arguments following the AAPCS64 calling
// convention: integer-like arguments take X0-X7, float arguments take
// D0-D7 and the rest goes onto the stack. Linux uses 8-byte stack slots
// while Darwin packs stack arguments to their natural alignment.
func buildCbArgPlan(ty reflect.Type, sret bool) []cbArgLoc {
	const intRegs, fltRegs = 8, 8
	ints, floats := 0, 0
	// The hidden struct return pointer uses the dedicated X8 register on
	// arm64, so sret does not consume an argument register.
	_ = sret
	var stackOff uintptr
	locs := make([]cbArgLoc, ty.NumIn())
	for i := range locs {
		t := ty.In(i)
		locs[i].typ = t
		isFloat := t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64
		if isFloat && floats < fltRegs {
			locs[i].src = cbSrcFloatReg
			locs[i].reg = floats
			floats++
			continue
		}
		if !isFloat && ints < intRegs {
			locs[i].src = cbSrcIntReg
			locs[i].reg = ints
			ints++
			continue
		}
		locs[i].src = cbSrcStack
		if runtime.GOOS == "darwin" {
			// Darwin packs stack arguments to their natural alignment.
			align := uintptr(t.Align())
			stackOff = (stackOff + align - 1) &^ (align - 1)
			locs[i].off = stackOff
			stackOff += t.Size()
		} else {
			locs[i].off = stackOff
			stackOff += 8
		}
	}
	return locs
}

// buildCbRetPlan classifies the return value following AAPCS64:
// floats return in D0, homogeneous float aggregates with up to four
// elements return in S0-S3/D0-D3, other structs up to 16 bytes return in
// X0,X1 and larger structs are copied through the pointer passed in X8.
func buildCbRetPlan(t reflect.Type) (plan cbRetPlan, sret bool) {
	if t == nil {
		plan.kind = cbRetNone
		return
	}
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		plan.kind = cbRetFloat
	case reflect.Struct:
		plan.size = t.Size()
		if plan.size == 0 {
			plan.kind = cbRetNone
		} else if ok, n, elem64 := cbIsHFA(t); ok {
			plan.kind = cbRetStructHFA
			plan.hfaN = n
			plan.hfaElem64 = elem64
		} else if plan.size <= 16 {
			plan.kind = cbRetStructInt
		} else {
			plan.kind = cbRetStructMem
			sret = true
		}
	default:
		plan.kind = cbRetInt
	}
	return
}

// cbIsHFA reports whether t is a homogeneous float aggregate: up to four
// fields (flattened through nested structs and arrays) that are all
// float32 or all float64.
func cbIsHFA(t reflect.Type) (ok bool, n int, elem64 bool) {
	kind := reflect.Invalid
	ok = true
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		if !ok {
			return
		}
		switch t.Kind() {
		case reflect.Float32, reflect.Float64:
			if kind == reflect.Invalid {
				kind = t.Kind()
			}
			if t.Kind() != kind {
				ok = false
			}
			n++
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				walk(t.Field(i).Type)
			}
		case reflect.Array:
			for i := 0; i < t.Len(); i++ {
				walk(t.Elem())
			}
		default:
			ok = false
		}
	}
	walk(t)
	if !ok || n == 0 || n > 4 {
		return false, 0, false
	}
	return true, n, kind == reflect.Float64
}

func encodeCbStructRet(entry *cbEntry, v reflect.Value, f *cbFrame) {
	p, pin := cbStructImage(v)
	size := entry.ret.size
	switch entry.ret.kind {
	case cbRetStructHFA:
		// One element per float register: S0-S3 or D0-D3.
		for i := 0; i < entry.ret.hfaN; i++ {
			if entry.ret.hfaElem64 {
				f.retFlt[i] = uintptr(*(*uint64)(unsafe.Add(p, i*8)))
			} else {
				f.retFlt[i] = uintptr(*(*uint32)(unsafe.Add(p, i*4)))
			}
		}
	case cbRetStructInt:
		f.retInt[0] = cbChunk(p, 0, size)
		if size > 8 {
			f.retInt[1] = cbChunk(p, 8, size)
		}
	case cbRetStructMem:
		// The hidden result pointer arrived in X8.
		Memcpy(cbPointer(&f.x8), p, Sizet(size))
	}
	runtime.KeepAlive(pin)
}
