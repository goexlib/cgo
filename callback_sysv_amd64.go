//go:build linux || darwin

package cgo

import (
	"reflect"
	"runtime"

	"github.com/ebitengine/purego"
)

// cbEntrySize is the size of one entry in goexlib_cgo_callbackasm
// (a CALL rel32 instruction).
const cbEntrySize = 5

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

// buildCbArgPlan assigns the arguments following the System V AMD64 ABI:
// integer-like arguments take RDI,RSI,RDX,RCX,R8,R9, float arguments take
// XMM0-XMM7 and the rest goes onto the stack in 8-byte slots.
func buildCbArgPlan(ty reflect.Type, sret bool) []cbArgLoc {
	const intRegs, fltRegs = 6, 8
	ints, floats := 0, 0
	if sret {
		ints = 1 // hidden struct return pointer in RDI
	}
	var stackOff uintptr
	locs := make([]cbArgLoc, ty.NumIn())
	for i := range locs {
		t := ty.In(i)
		locs[i].typ = t
		switch t.Kind() {
		case reflect.Float32, reflect.Float64:
			if floats < fltRegs {
				locs[i].src = cbSrcFloatReg
				locs[i].reg = floats
				floats++
			} else {
				locs[i].src = cbSrcStack
				locs[i].off = stackOff
				stackOff += 8
			}
		default:
			if ints < intRegs {
				locs[i].src = cbSrcIntReg
				locs[i].reg = ints
				ints++
			} else {
				locs[i].src = cbSrcStack
				locs[i].off = stackOff
				stackOff += 8
			}
		}
	}
	return locs
}

// buildCbRetPlan classifies the return value following the System V AMD64
// ABI: floats return in XMM0, structs up to 16 bytes return in up to two
// registers picked by per-8-byte-chunk classification (INTEGER -> RAX,RDX;
// SSE -> XMM0,XMM1), larger structs return through a hidden pointer passed
// in RDI which must be handed back in RAX.
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
		} else if plan.size > 16 {
			plan.kind = cbRetStructMem
			sret = true
		} else {
			plan.kind = cbRetStructSysV
			plan.sse = [2]bool{true, true}
			cbClassifySysV(t, 0, &plan.sse)
		}
	default:
		plan.kind = cbRetInt
	}
	return
}

// cbClassifySysV marks every 8-byte chunk that contains a non-float field
// as INTEGER class (sse[chunk] = false).
func cbClassifySysV(t reflect.Type, base uintptr, sse *[2]bool) {
	for i := 0; i < t.NumField(); i++ {
		cbClassifySysVField(t.Field(i).Type, base+t.Field(i).Offset, sse)
	}
}

func cbClassifySysVField(t reflect.Type, off uintptr, sse *[2]bool) {
	switch t.Kind() {
	case reflect.Struct:
		cbClassifySysV(t, off, sse)
	case reflect.Array:
		elem := t.Elem()
		for i := 0; i < t.Len(); i++ {
			cbClassifySysVField(elem, off+uintptr(i)*elem.Size(), sse)
		}
	case reflect.Float32, reflect.Float64:
		// float-only chunks stay SSE class
	default:
		sse[off/8] = false
	}
}

func encodeCbStructRet(entry *cbEntry, v reflect.Value, f *cbFrame) {
	p, pin := cbStructImage(v)
	size := entry.ret.size
	switch entry.ret.kind {
	case cbRetStructSysV:
		intN, fltN := 0, 0
		for c := uintptr(0); c*8 < size; c++ {
			w := cbChunk(p, c*8, size)
			if entry.ret.sse[c] {
				f.retFlt[fltN] = w
				fltN++
			} else {
				f.retInt[intN] = w
				intN++
			}
		}
	case cbRetStructMem:
		// The hidden result pointer arrived in RDI (ints[0]) and must be
		// returned in RAX.
		Memcpy(cbPointer(&f.ints[0]), p, Sizet(size))
		f.retInt[0] = f.ints[0]
	}
	runtime.KeepAlive(pin)
}
