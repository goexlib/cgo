package cgo

import (
	"math"
	"reflect"
	"runtime"
	"syscall"
	"unsafe"
)

func callRet[T any](fn uintptr, args ...any) (ret T) {
	call(fn, &ret, args...)
	return
}

func call(fn uintptr, ret any, args ...any) {
	rawArgs := make([]uintptr, 0, len(args)+1)
	var retType reflect.Type
	if ret != nil {
		retType = reflect.TypeOf(ret).Elem()
		checkReturnType(retType)

		if retType.Kind() != reflect.String && retType.Size() > unsafe.Sizeof(uintptr(0)) {
			rp := (*iface)(unsafe.Pointer(&ret)).val
			rawArgs = append(rawArgs, rp)
		}
	}

	for i := range args {
		checkCgoType(reflect.TypeOf(args[i]))

		switch val := args[i].(type) {
		case string:
			p := CString(val)
			defer runtime.KeepAlive(p)
			rawArgs = append(rawArgs, uintptr(p))
		case bool:
			rawArgs = append(rawArgs, uintptr(CBool(val)))
		case int8:
			rawArgs = append(rawArgs, uintptr(val))
		case uint8:
			rawArgs = append(rawArgs, uintptr(val))
		case int16:
			rawArgs = append(rawArgs, uintptr(val))
		case uint16:
			rawArgs = append(rawArgs, uintptr(val))
		case int32:
			rawArgs = append(rawArgs, uintptr(val))
		case uint32:
			rawArgs = append(rawArgs, uintptr(val))
		case int64:
			rawArgs = append(rawArgs, uintptr(val))
		case uint64:
			rawArgs = append(rawArgs, uintptr(val))
		case int:
			rawArgs = append(rawArgs, uintptr(val))
		case uint:
			rawArgs = append(rawArgs, uintptr(val))
		case float32:
			rawArgs = append(rawArgs, uintptr(math.Float32bits(val)))
		case float64:
			rawArgs = append(rawArgs, uintptr(math.Float64bits(val)))
		case uintptr:
			rawArgs = append(rawArgs, val)
		default:
			v := unsafe.Pointer((*iface)(unsafe.Pointer(&args[i])).val)
			if reflect.TypeOf(args[i]).Size() > unsafe.Sizeof(uintptr(0)) {
				rawArgs = append(rawArgs, uintptr(v))
			} else {
				rawArgs = append(rawArgs, *(*uintptr)(v))
			}
		}
	}

	r1, r2, _ := syscall.SyscallN(fn, rawArgs...)
	if ret != nil {
		switch val := ret.(type) {
		case *string:
			*val = GoString(unsafe.Pointer(r1))
		case *bool:
			*val = r1 != 0
		case *int8:
			*val = int8(r1)
		case *uint8:
			*val = uint8(r1)
		case *int16:
			*val = int16(r1)
		case *uint16:
			*val = uint16(r1)
		case *int32:
			*val = int32(r1)
		case *uint32:
			*val = uint32(r1)
		case *int64:
			*val = int64(r1)
		case *uint64:
			*val = uint64(r1)
		case *int:
			*val = int(r1)
		case *uint:
			*val = uint(r1)
		case *uintptr:
			*val = r1
		case *float32:
			*val = math.Float32frombits(uint32(r2))
		case *float64:
			*val = math.Float64frombits(uint64(r2))
		default:
			if size := retType.Size(); size <= unsafe.Sizeof(uintptr(0)) {
				v := (*iface)(unsafe.Pointer(&ret)).val
				*(*uintptr)(unsafe.Pointer(v)) = r1
			}
		}
	}
}

func checkReturnType(retType reflect.Type) {
	switch retType.Kind() {
	case reflect.Array:
		panic("cgo.Call: unsupported return type")
	default:
		checkCgoType(retType)
	}
}

func checkCgoType(typ reflect.Type) {
	switch typ.Kind() {
	case reflect.Slice, reflect.Interface, reflect.Map, reflect.Chan, reflect.Func:
		panic("cgo.Call: unsupported cgo type")
	case reflect.Array:
		checkCgoType(typ.Elem())
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			checkCgoType(typ.Field(i).Type)
		}
	}
}

type iface struct {
	typ uintptr
	val uintptr
}
