//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && amd64)

package cgo

import (
	"runtime"
	"testing"
)

// requirePuregoStackLayout skips stack-spill tests on darwin/arm64, where
// purego's caller uses 8-byte stack slots instead of Apple's packed layout.
// Those callbacks are covered against a real C compiler in TestCallbackFromC.
func requirePuregoStackLayout(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		t.Skip("stack-argument callbacks are verified in TestCallbackFromC on darwin/arm64")
	}
}

// TestNewCallbackForwarded exercises the backend-forwarded path.
func TestNewCallbackForwarded(t *testing.T) {
	cb := NewCallback(func(a, b Int) Int {
		return a*10 + b
	})
	r1, _, _ := CallRaw(cb, 3, 4)
	if Int(r1) != 34 {
		t.Fatalf("forwarded callback returned %d, want 34", int32(r1))
	}
}

// TestNewCallbackNativeInts forces the native thunk with integer register
// and stack arguments.
func TestNewCallbackNativeInts(t *testing.T) {
	requirePuregoStackLayout(t)
	cb := newCallbackNative(func(a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 Int) Int {
		return a1*1 + a2*2 + a3*3 + a4*4 + a5*5 + a6*6 + a7*7 + a8*8 + a9*9 + a10*10 + a11*11 + a12*12
	})
	r1, _, _ := CallRaw(cb, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	want := Int(0)
	for i := Int(1); i <= 12; i++ {
		want += i * i
	}
	if Int(r1) != want {
		t.Fatalf("native callback returned %d, want %d", int32(r1), want)
	}
}

// testCbPtrTarget is package-level so its address stays stable while the raw
// pointer travels through the C calling convention.
var testCbPtrTarget Int

// TestNewCallbackNativeNarrowArgs forces the native thunk with narrow
// integer types, bool and pointer arguments.
func TestNewCallbackNativeNarrowArgs(t *testing.T) {
	var captured Int
	cb := newCallbackNative(func(a Char, b Ushort, c bool, p *Int) Uint {
		captured = *p
		r := Uint(0)
		if a == -5 {
			r += 1
		}
		if b == 40000 {
			r += 2
		}
		if c {
			r += 4
		}
		return r
	})
	testCbPtrTarget = 12345
	a := Char(-5)
	r1, _, _ := CallRaw(cb, uintptr(uint8(a)), 40000, 1, uintptr(Pointer(&testCbPtrTarget)))
	if Uint(uint32(r1)) != 7 {
		t.Fatalf("native callback returned %d, want 7", uint32(r1))
	}
	if captured != 12345 {
		t.Fatalf("pointer argument dereferenced to %d, want 12345", captured)
	}
}

// TestNewCallbackNativeNoReturn forces the native thunk with no return value.
func TestNewCallbackNativeNoReturn(t *testing.T) {
	var got Int
	cb := newCallbackNative(func(a, b Int) {
		got = a + b
	})
	CallRaw(cb, 20, 22)
	if got != 42 {
		t.Fatalf("native callback got %d, want 42", got)
	}
}

// TestNewCallbackMany ensures distinct callbacks keep distinct indices.
func TestNewCallbackMany(t *testing.T) {
	for i := Int(0); i < 8; i++ {
		i := i
		cb := newCallbackNative(func(x Int) Int { return x + i })
		r1, _, _ := CallRaw(cb, 100)
		if Int(r1) != 100+i {
			t.Fatalf("callback %d returned %d, want %d", i, int32(r1), 100+i)
		}
	}
}

// Struct types shared by the round-trip and C-library callback tests.
// Field layout matches the C structs in testdata/libcallback.c.
type (
	testPointF struct {
		X, Y Float
	}
	testVec3 struct {
		X, Y, Z Float
	}
	testArrayHFA struct {
		V [2]Float
		W Float
	}
	testQuad struct {
		A, B, C, D Double
	}
	testMixed struct {
		A Int
		B Float
		C Double
	}
	testIntPair struct {
		A, B LongLong
	}
	testSmallInts struct {
		A, B, C Int
	}
	testBig struct {
		A, B, C LongLong
	}
	testFloatIntFloat struct {
		A Float
		B Int
		C Float
	}
	testOneFloat struct {
		V Float
	}
)

func TestNewCallbackPanics(t *testing.T) {
	expectPanic := func(name string, f func()) {
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		f()
	}
	expectPanic("nil", func() { NewCallback(nil) })
	expectPanic("not a function", func() { NewCallback(42) })
	expectPanic("variadic", func() { NewCallback(func(...Int) {}) })
	expectPanic("multiple returns", func() { NewCallback(func() (Int, Int) { return 0, 0 }) })
	expectPanic("struct argument", func() { NewCallback(func(struct{ X Int }) {}) })
	expectPanic("string argument", func() { NewCallback(func(string) {}) })
	expectPanic("string return", func() { NewCallback(func() string { return "" }) })
	expectPanic("slice field", func() { NewCallback(func() struct{ S []Int } { return struct{ S []Int }{} }) })
}
