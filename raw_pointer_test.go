package cgo

import (
	"runtime"
	"testing"
	"unsafe"
)

// Fresh goroutines and different caller frame depths exercise stack growth
// between pointer conversion and the native call, including lazy resolution.
// The native memset result is checked against independent byte/canary values.
func TestRawCallPointerStability(t *testing.T) {
	for _, mode := range []string{"raw", "symbol", "lazy", "memset"} {
		t.Run(mode, func(t *testing.T) {
			lib := NewLazyLibrary("libc.so.6", "msvcrt.dll", "/usr/lib/libSystem.B.dylib")
			sym := lib.NewSymbol("memset")
			addr := sym.Addr()
			for depth := 0; depth < 96; depth++ {
				result := make(chan [258]byte)
				go func() { result <- rawAtDepth(mode, sym, addr, depth) }()
				got := <-result
				if got[0] != 0x11 || got[257] != 0x22 {
					t.Fatalf("depth=%d native write damaged canary", depth)
				}
				for i := 1; i < 257; i++ {
					if got[i] != 0xa5 {
						t.Fatalf("depth=%d byte=%d got=%#x want=0xa5", depth, i, got[i])
					}
				}
			}
		})
	}
}

//go:noinline
func rawAtDepth(mode string, symbol *LazySymbol, addr uintptr, depth int) (out [258]byte) {
	var padding [256]byte
	padding[0] = byte(depth)
	if depth > 0 {
		out = rawAtDepth(mode, symbol, addr, depth-1)
	} else {
		out = rawFill(mode, symbol, addr)
	}
	runtime.KeepAlive(padding)
	return
}

//go:noinline
func rawFill(mode string, symbol *LazySymbol, addr uintptr) (out [258]byte) {
	out[0], out[257] = 0x11, 0x22
	switch mode {
	case "raw":
		CallRaw(addr, uintptr(unsafe.Pointer(&out[1])), 0xa5, 256)
	case "symbol":
		Symbol(addr).CallRaw(uintptr(unsafe.Pointer(&out[1])), 0xa5, 256)
	case "lazy":
		symbol.CallRaw(uintptr(unsafe.Pointer(&out[1])), 0xa5, 256)
	case "memset":
		Memset(unsafe.Pointer(&out[1]), 0xa5, 256)
	}
	return
}
