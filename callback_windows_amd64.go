package cgo

import (
	"reflect"
	"runtime"
	"syscall"
)

// cbEntrySize is the size of one entry in goexlib_cgo_callbackasm
// (a CALL rel32 instruction).
const cbEntrySize = 5

// ptrSize is the size of a pointer on windows/amd64.
const ptrSize = 8

// cbBackendSupported reports whether syscall.NewCallback can handle the
// signature.
//
// The Go runtime's syscall.NewCallback is far stricter than
// purego.NewCallback: it requires exactly one result whose size is exactly
// the pointer size (8 bytes) and which is not a float, and it rejects float
// arguments as well as any argument wider than a pointer. See
// runtime.compileCallback and abiDesc.assignArg. In particular it rejects
// int32/int16/bool results and functions with no result at all, which are
// perfectly ordinary C callbacks.
//
// Rather than mirror those internal rules (and risk drifting out of sync
// with them), we only forward the cases that are unambiguously accepted and
// let this package's own thunk handle everything else. The native thunk is
// correct for the forwarded cases too, so a false negative here only costs a
// callback table slot, never correctness.
func cbBackendSupported(ty reflect.Type) bool {
	for i := 0; i < ty.NumIn(); i++ {
		in := ty.In(i)
		switch in.Kind() {
		case reflect.Float32, reflect.Float64:
			return false
		}
		if in.Size() > ptrSize {
			return false
		}
	}
	if ty.NumOut() != 1 || ty.Out(0).Size() != ptrSize {
		return false
	}
	switch ty.Out(0).Kind() {
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64,
		reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
		return true
	}
	return false
}

func cbBackendNewCallback(fn any) uintptr {
	return syscall.NewCallback(fn)
}

func cbNewDispatchTrampoline() uintptr {
	return syscall.NewCallback(cbDispatch)
}

// buildCbArgPlan assigns the arguments following the Microsoft x64 calling
// convention: argument slots are positional, slot i (i < 4) lives in XMM_i
// for floats or in RCX,RDX,R8,R9 for everything else, and the remaining
// slots go onto the stack in 8-byte cells.
func buildCbArgPlan(ty reflect.Type, sret bool) []cbArgLoc {
	pos := 0
	if sret {
		pos = 1 // hidden struct return pointer in RCX
	}
	locs := make([]cbArgLoc, ty.NumIn())
	for i := range locs {
		t := ty.In(i)
		locs[i].typ = t
		isFloat := t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64
		if pos < 4 {
			if isFloat {
				locs[i].src = cbSrcFloatReg
			} else {
				locs[i].src = cbSrcIntReg
			}
			locs[i].reg = pos
		} else {
			locs[i].src = cbSrcStack
			locs[i].off = uintptr(pos-4) * 8
		}
		pos++
	}
	return locs
}

// buildCbRetPlan classifies the return value following the Microsoft x64
// calling convention: floats return in XMM0, structs of size 1, 2, 4 or 8
// bytes return in RAX and all other structs are copied through a hidden
// pointer passed in RCX which must be handed back in RAX.
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
		switch plan.size {
		case 0:
			plan.kind = cbRetNone
		case 1, 2, 4, 8:
			plan.kind = cbRetStructInt
		default:
			plan.kind = cbRetStructMem
			sret = true
		}
	default:
		plan.kind = cbRetInt
	}
	return
}

func encodeCbStructRet(entry *cbEntry, v reflect.Value, f *cbFrame) {
	p, pin := cbStructImage(v)
	size := entry.ret.size
	switch entry.ret.kind {
	case cbRetStructInt:
		f.retInt[0] = cbChunk(p, 0, size)
	case cbRetStructMem:
		// The hidden result pointer arrived in RCX (ints[0]) and must be
		// returned in RAX.
		Memcpy(cbPointer(&f.ints[0]), p, Sizet(size))
		f.retInt[0] = f.ints[0]
	}
	runtime.KeepAlive(pin)
}
