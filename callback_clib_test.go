//go:build ((linux || darwin) && (amd64 || arm64)) || (windows && amd64)

package cgo

import (
	"runtime"
	"testing"
)

// The tests in this file verify the callbacks against real C code compiled
// by a real C compiler (the authoritative ABI implementation). Every
// test_cb_* function in testdata/libcallback.c invokes the callback with
// hard-coded arguments, checks the result in C and returns 0 on success.
//
// Build the library first (see testdata/libcallback.c for the exact
// commands), the tests are skipped when it is missing.

func loadCallbackTestLib(t *testing.T) Library {
	t.Helper()
	var name string
	switch runtime.GOOS {
	case "windows":
		name = "testdata/libcallback.dll"
	case "darwin":
		name = "testdata/libcallback.dylib"
	default:
		name = "testdata/libcallback.so"
	}
	lib, err := LoadLibrary(name)
	if err != nil {
		t.Skipf("%s not built (see testdata/libcallback.c): %v", name, err)
	}
	return lib
}

// runCbTest invokes one C-side test driver and reports its error code.
func runCbTest(t *testing.T, lib Library, name string, cbs ...uintptr) {
	t.Helper()
	sym, err := lib.GetSymbol(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	r1, _, _ := CallRaw(sym.Addr(), cbs...)
	if r1 != 0 {
		t.Errorf("%s failed with code %d", name, int64(r1))
	}
}

func TestCallbackFromC(t *testing.T) {
	lib := loadCallbackTestLib(t)
	defer lib.Close()

	runCbTest(t, lib, "test_cb_int", NewCallback(func(a, b Int) Int {
		return a*10 + b
	}))

	runCbTest(t, lib, "test_cb_double", NewCallback(func(a, b Double) Double {
		return a + b
	}))

	runCbTest(t, lib, "test_cb_float", NewCallback(func(a, b Float) Float {
		return a * b
	}))

	runCbTest(t, lib, "test_cb_float_int_mix", NewCallback(func(a Int, b Float, c Double, d Int) Double {
		return Double(a) + Double(b) + c + Double(d)
	}))

	runCbTest(t, lib, "test_cb_stack", NewCallback(func(
		i1, i2, i3, i4, i5, i6, i7, i8, i9, i10 Int,
		f1, f2, f3, f4, f5, f6, f7, f8, f9, f10 Double) Double {
		is := i1 + i2 + i3 + i4 + i5 + i6 + i7 + i8 + i9 + i10
		fs := f1 + f2 + f3 + f4 + f5 + f6 + f7 + f8 + f9 + f10
		return Double(is) + fs
	}))

	runCbTest(t, lib, "test_cb_pointf", NewCallback(func(x, y Float) testPointF {
		return testPointF{X: x * 2, Y: y * 2}
	}))

	runCbTest(t, lib, "test_cb_vec3", NewCallback(func(x Float) testVec3 {
		return testVec3{X: x, Y: x + 1, Z: x + 2}
	}))

	runCbTest(t, lib, "test_cb_array_hfa", NewCallback(func() testArrayHFA {
		return testArrayHFA{V: [2]Float{1.5, 2.5}, W: 3.5}
	}))

	runCbTest(t, lib, "test_cb_quad", NewCallback(func(a Double) testQuad {
		return testQuad{A: a, B: a * 2, C: a * 3, D: a * 4}
	}))

	runCbTest(t, lib, "test_cb_mixed", NewCallback(func(a Int, b Float, c Double) testMixed {
		return testMixed{A: a + 1, B: b + 0.5, C: c + 0.25}
	}))

	runCbTest(t, lib, "test_cb_big", NewCallback(func(a LongLong) testBig {
		return testBig{A: a, B: a * 2, C: a * 3}
	}))

	runCbTest(t, lib, "test_cb_big_args", NewCallback(func(a, b, c, d, e, f, g Int, x Double) testBig {
		return testBig{A: LongLong(a + b + c + d), B: LongLong(e + f + g), C: LongLong(x)}
	}))

	runCbTest(t, lib, "test_cb_float_int_float", NewCallback(func() testFloatIntFloat {
		return testFloatIntFloat{A: 1.5, B: 2, C: 3.5}
	}))

	runCbTest(t, lib, "test_cb_one_float", NewCallback(func(v Float) testOneFloat {
		return testOneFloat{V: v + 0.25}
	}))

	runCbTest(t, lib, "test_cb_int_pair", NewCallback(func(a, b LongLong) testIntPair {
		return testIntPair{A: a * 2, B: b * 3}
	}))

	runCbTest(t, lib, "test_cb_bool_ret", NewCallback(func(v Int) bool {
		return v != 0
	}))

	var voidGot Int
	runCbTest(t, lib, "test_cb_void",
		NewCallback(func(v Int) { voidGot = v }),
		NewCallback(func() Int { return voidGot }))

	// Invoke a float-returning callback from a foreign C thread.
	runCbTest(t, lib, "test_cb_thread", NewCallback(func(a, b Double) Double {
		return a + b
	}))
}
