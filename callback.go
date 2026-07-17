//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && amd64)

package cgo

import (
	"math"
	"reflect"
	"sync"
	"unsafe"
)

//go:generate go run gen_callback.go

// cbFrame is the data exchanged between the assembly thunk
// (goexlib_cgo_callbackasm1 in callback_*.s) and the Go dispatcher
// cbDispatch. It lives on the C thread's stack. The field offsets are
// exported to the assembly through go_asm.h, so the assembly stays in sync
// automatically.
type cbFrame struct {
	// floats holds the callee float registers:
	// XMM0-XMM7 (SysV amd64), XMM0-XMM3 (win64), D0-D7 (arm64).
	floats [8]uintptr
	// ints holds the callee integer registers:
	// DI,SI,DX,CX,R8,R9 (SysV amd64), CX,DX,R8,R9 (win64), R0-R7 (arm64).
	ints [8]uintptr
	// x8 is the arm64 indirect result location register (unused on amd64).
	x8 uintptr
	// stack points at the first stack argument of the C caller.
	stack unsafe.Pointer
	// index is the callback index recorded by the entry table.
	index uintptr
	// retInt receives the integer return registers: RAX,RDX / X0,X1.
	retInt [2]uintptr
	// retFlt receives the float return registers: XMM0,XMM1 / D0-D3.
	retFlt [4]uintptr
}

// cbMaxCallbacks is the size of the generated entry table.
// Must be kept in sync with maxCallback in gen_callback.go.
const cbMaxCallbacks = 1024

// cbasmBaseAddr returns the address of goexlib_cgo_callbackasm.
// Implemented in zcallback_GOARCH.s.
func cbasmBaseAddr() uintptr

// cbTrampoline holds a C function pointer to cbDispatch created with the
// backend NewCallback (purego on unix, syscall on windows). It is read by
// the assembly thunk, and must be initialized before any native callback
// pointer is handed out.
var cbTrampoline uintptr

var cbTrampolineOnce sync.Once

func ensureCbTrampoline() {
	cbTrampolineOnce.Do(func() {
		cbTrampoline = cbNewDispatchTrampoline()
	})
}

type cbArgSrc uint8

const (
	cbSrcIntReg cbArgSrc = iota
	cbSrcFloatReg
	cbSrcStack
)

// cbArgLoc describes where one callback argument comes from.
type cbArgLoc struct {
	typ reflect.Type
	src cbArgSrc
	reg int     // register index when src != cbSrcStack
	off uintptr // byte offset into the stack arguments when src == cbSrcStack
}

type cbRetKind uint8

const (
	cbRetNone       cbRetKind = iota
	cbRetInt                  // integer-like value in retInt[0]
	cbRetFloat                // float32/float64 bits in retFlt[0]
	cbRetStructInt            // struct image in retInt[0..1] (arm64 <=16B, win64 1/2/4/8B)
	cbRetStructSysV           // SysV amd64 <=16B struct, 8-byte chunks classified INTEGER/SSE
	cbRetStructHFA            // arm64 homogeneous float aggregate in retFlt[0..3]
	cbRetStructMem            // large struct copied through the hidden result pointer
)

// cbRetPlan describes how the callback return value is encoded.
type cbRetPlan struct {
	kind cbRetKind
	size uintptr
	// sse reports whether the SysV 8-byte chunk is SSE class (amd64 only).
	sse [2]bool
	// hfa element info (arm64 only).
	hfaN      int
	hfaElem64 bool
}

type cbEntry struct {
	fn   reflect.Value
	args []cbArgLoc
	ret  cbRetPlan
	// sret reports whether the C caller passes a hidden pointer for the
	// struct return value in the first integer register (amd64) / X8 (arm64).
	sret bool
}

var cbs struct {
	sync.Mutex
	n       int
	entries [cbMaxCallbacks]*cbEntry
}

// newCallback converts a Go function to a C function pointer.
// If the backend (purego on unix, syscall on windows) supports the
// signature, it is forwarded there. Otherwise (float/struct return values,
// float arguments on windows) the callback is backed by this package's own
// assembly thunk.
func newCallback(fn any) uintptr {
	if fn == nil {
		panic("cgo.NewCallback: fn must not be nil")
	}
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("cgo.NewCallback: fn must be a function")
	}
	if val.IsNil() {
		panic("cgo.NewCallback: fn must not be nil")
	}
	ty := val.Type()
	if ty.IsVariadic() {
		panic("cgo.NewCallback: variadic functions are not supported")
	}
	if ty.NumOut() > 1 {
		panic("cgo.NewCallback: function can only return zero or one value")
	}
	for i := 0; i < ty.NumIn(); i++ {
		checkCbArgType(ty.In(i))
	}
	if ty.NumOut() == 1 {
		checkCbRetType(ty.Out(0))
	}
	if cbBackendSupported(ty) {
		return cbBackendNewCallback(fn)
	}
	return newCallbackNative(fn)
}

// newCallbackNative always uses this package's own callback machinery,
// bypassing the backend even for signatures it could handle.
func newCallbackNative(fn any) uintptr {
	ensureCbTrampoline()

	val := reflect.ValueOf(fn)
	ty := val.Type()
	entry := &cbEntry{fn: val}
	var retType reflect.Type
	if ty.NumOut() == 1 {
		retType = ty.Out(0)
	}
	entry.ret, entry.sret = buildCbRetPlan(retType)
	entry.args = buildCbArgPlan(ty, entry.sret)

	cbs.Lock()
	defer cbs.Unlock()
	if cbs.n >= cbMaxCallbacks {
		panic("cgo.NewCallback: the maximum number of callbacks has been reached")
	}
	cbs.entries[cbs.n] = entry
	cbs.n++
	return cbasmBaseAddr() + uintptr((cbs.n-1)*cbEntrySize)
}

func checkCbArgType(t reflect.Type) {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Ptr, reflect.UnsafePointer:
	case reflect.Struct:
		panic("cgo.NewCallback: struct arguments are not supported")
	default:
		panic("cgo.NewCallback: unsupported argument type: " + t.Kind().String())
	}
}

func checkCbRetType(t reflect.Type) {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Ptr, reflect.UnsafePointer:
	case reflect.Struct:
		checkCbRetStructFields(t)
	default:
		panic("cgo.NewCallback: unsupported return type: " + t.Kind().String())
	}
}

func checkCbRetStructFields(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i).Type
		for ft.Kind() == reflect.Array {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uintptr, reflect.Float32, reflect.Float64,
			reflect.Ptr, reflect.UnsafePointer:
		case reflect.Struct:
			checkCbRetStructFields(ft)
		default:
			panic("cgo.NewCallback: unsupported struct field type: " + ft.Kind().String())
		}
	}
}

// cbDispatch is called (through the backend trampoline) by the assembly
// thunk whenever C code invokes a native callback. It decodes the C
// arguments, calls the Go function and encodes the return value back into
// the frame, from which the assembly thunk restores the return registers.
func cbDispatch(f *cbFrame) uintptr {
	cbs.Lock()
	entry := cbs.entries[f.index]
	cbs.Unlock()

	args := make([]reflect.Value, len(entry.args))
	for i := range entry.args {
		loc := &entry.args[i]
		var p unsafe.Pointer
		switch loc.src {
		case cbSrcIntReg:
			p = unsafe.Pointer(&f.ints[loc.reg])
		case cbSrcFloatReg:
			p = unsafe.Pointer(&f.floats[loc.reg])
		default:
			p = unsafe.Add(f.stack, loc.off)
		}
		args[i] = reflect.NewAt(loc.typ, p).Elem()
	}

	rets := entry.fn.Call(args)

	switch entry.ret.kind {
	case cbRetNone:
	case cbRetInt:
		f.retInt[0] = cbIntBits(rets[0])
	case cbRetFloat:
		if rets[0].Kind() == reflect.Float32 {
			f.retFlt[0] = uintptr(math.Float32bits(float32(rets[0].Float())))
		} else {
			f.retFlt[0] = uintptr(math.Float64bits(rets[0].Float()))
		}
	default:
		encodeCbStructRet(entry, rets[0], f)
	}
	return 0
}

func cbIntBits(v reflect.Value) uintptr {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uintptr(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintptr(v.Uint())
	case reflect.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	case reflect.UnsafePointer:
		return uintptr(v.UnsafePointer())
	default: // reflect.Ptr
		return v.Pointer()
	}
}

// cbStructImage returns a pointer to an addressable copy of the returned
// struct value. The returned pointer stays alive while v does through the
// pinned reflect value; callers must finish copying before returning.
func cbStructImage(v reflect.Value) (unsafe.Pointer, reflect.Value) {
	pv := reflect.New(v.Type())
	pv.Elem().Set(v)
	return pv.UnsafePointer(), pv
}

// cbChunk loads the 8-byte chunk of the struct image starting at off,
// zero-padding past size.
func cbChunk(p unsafe.Pointer, off, size uintptr) uintptr {
	var w uintptr
	n := size - off
	if n > 8 {
		n = 8
	}
	Memcpy(Pointer(&w), unsafe.Add(p, off), Sizet(n))
	return w
}

// cbPointer converts an integer register slot holding a C pointer into an
// unsafe.Pointer. We take the address and then dereference it to keep
// go vet happy (same trick as purego).
func cbPointer(v *uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(v))
}
