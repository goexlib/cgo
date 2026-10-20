package cgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

// A real C caller preserves an address while its Go callback grows the stack
// and runs GC. This covers reentry after C has already received the address,
// separately from raw_pointer_test's pre-call stack growth.
func TestPointersAcrossNativeReentry(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler unavailable for native reentry fixture")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "reentry.c")
	code := `
#include <stdint.h>
#ifdef _WIN32
#define EXPORT __declspec(dllexport)
#else
#define EXPORT
#endif
typedef void (*callback)(void);
typedef struct { uint64_t a, b, c; } triple;
EXPORT void write_after_callback(unsigned char *out, callback grow) {
    grow(); out[0] = 0xa5; out[1] = 0x5a;
}
EXPORT triple value_after_callback(triple value, callback grow) {
    grow(); value.a += 1; value.b += 2; value.c += 3; return value;
}
`
	if err := os.WriteFile(src, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	name, flags := "reentry.so", []string{"-shared", "-fPIC"}
	if runtime.GOOS == "darwin" {
		name, flags = "reentry.dylib", []string{"-dynamiclib"}
	}
	if runtime.GOOS == "windows" {
		name, flags = "reentry.dll", []string{"-shared"}
	}
	file := filepath.Join(dir, name)
	args := append(flags, "-O2", "-o", file, src)
	if output, err := exec.Command(cc, args...).CombinedOutput(); err != nil {
		t.Fatalf("build native fixture: %v\n%s", err, output)
	}
	lib, err := LoadLibrary(file)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	write, err := lib.GetSymbol("write_after_callback")
	if err != nil {
		t.Fatal(err)
	}
	value, err := lib.GetSymbol("value_after_callback")
	if err != nil {
		t.Fatal(err)
	}
	grow := NewCallback(func() { reentryGrow(64); runtime.GC() })
	for _, mode := range []string{"raw", "symbol", "typed"} {
		t.Run(mode, func(t *testing.T) {
			var out [4]byte
			out[0], out[3] = 0x11, 0x22
			switch mode {
			case "raw":
				CallRaw(write.Addr(), uintptr(unsafe.Pointer(&out[1])), grow)
			case "symbol":
				write.CallRaw(uintptr(unsafe.Pointer(&out[1])), grow)
			case "typed":
				Call(write, unsafe.Pointer(&out[1]), grow)
			}
			if out != [4]byte{0x11, 0xa5, 0x5a, 0x22} {
				t.Fatalf("native callback write: %#v", out)
			}
		})
	}
	type triple struct{ A, B, C uint64 }
	if got := CallRet[triple](value, triple{10, 20, 30}, grow); got != (triple{11, 22, 33}) {
		t.Fatalf("indirect argument/return after callback: %+v", got)
	}
}

//go:noinline
func reentryGrow(depth int) {
	var padding [1024]byte
	padding[0] = byte(depth)
	if depth > 0 {
		reentryGrow(depth - 1)
	}
	runtime.KeepAlive(padding)
}
