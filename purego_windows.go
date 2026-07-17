package cgo

import (
	"errors"
	"path/filepath"
	"syscall"
)

// Dummy RTLD flags
const (
	RTLD_DEFAULT = 0
	RTLD_LAZY    = 0
	RTLD_NOW     = 0
	RTLD_LOCAL   = 0
	RTLD_GLOBAL  = 0
)

// Dlopen examines the dynamic library or bundle file specified by path.
//
// The mode was ignored on Windows.
func Dlopen(path string, mode int) (uintptr, error) {
	if len(path) == 0 {
		return 0, errors.New("invalid library path")
	}
	handle, err := syscall.LoadLibrary(filepath.ToSlash(path))
	if err != nil {
		return 0, err
	}
	return uintptr(handle), nil
}

// Dlsym takes a "handle" of a dynamic library returned by Dlopen and the symbol name.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	if handle == 0 {
		return 0, errors.New("invalid handle")
	}
	return syscall.GetProcAddress(syscall.Handle(handle), name)
}

// Dlclose decrements the reference count on the dynamic library handle.
func Dlclose(handle uintptr) error {
	return syscall.FreeLibrary(syscall.Handle(handle))
}

// NewCallback converts a Go function to a C function pointer that can be
// passed to C code as a callback.
//
// On windows/amd64 it additionally supports float32/float64 arguments as
// well as float32/float64 and struct return values (which syscall.NewCallback
// does not): such callbacks are backed by this package's own assembly thunk,
// everything else is forwarded to syscall.NewCallback. Struct arguments are
// not supported. Struct return types must have the same memory layout as
// their C counterpart (add explicit padding fields if needed).
//
// Only a limited number of callbacks may be created in a single Go process,
// and any memory allocated for these callbacks is never released.
func NewCallback(fn any) uintptr {
	return newCallback(fn)
}

func CallRaw(fn uintptr, args ...uintptr) (r1, r2 uintptr, err syscall.Errno) {
	return syscall.SyscallN(fn, args...)
}
