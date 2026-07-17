//go:build (linux || darwin) && (amd64 || arm64)

package cgo

import "testing"

// The tests in this file round-trip callbacks through CallRet
// (purego.RegisterFunc), which drives them following the C calling convention.

func TestCallbackRetFloat64(t *testing.T) {
	cb := NewCallback(func(a, b Double) Double {
		return a + b
	})
	if got := CallRet[Double](cb, Double(1.5), Double(2.25)); got != 3.75 {
		t.Fatalf("got %v, want 3.75", got)
	}
}

func TestCallbackRetFloat32(t *testing.T) {
	cb := NewCallback(func(a, b Float) Float {
		return a * b
	})
	if got := CallRet[Float](cb, Float(1.5), Float(3.0)); got != 4.5 {
		t.Fatalf("got %v, want 4.5", got)
	}
}

// TestCallbackFloatArgsIntRet exercises float arguments with an integer return.
func TestCallbackFloatArgsIntRet(t *testing.T) {
	cb := NewCallback(func(a Double, b Float, c Int) Int {
		return Int(a) + Int(b) + c
	})
	if got := CallRet[Int](cb, Double(100), Float(20), Int(3)); got != 123 {
		t.Fatalf("got %v, want 123", got)
	}
}

// TestCallbackStackArgs passes more arguments than there are registers so
// both integer and float values spill onto the C stack.
func TestCallbackStackArgs(t *testing.T) {
	requirePuregoStackLayout(t)
	cb := NewCallback(func(i1, i2, i3, i4, i5, i6, i7, i8, i9, i10 Int,
		f1, f2, f3, f4, f5, f6, f7, f8, f9, f10 Double) Double {
		is := i1 + i2 + i3 + i4 + i5 + i6 + i7 + i8 + i9 + i10
		fs := f1 + f2 + f3 + f4 + f5 + f6 + f7 + f8 + f9 + f10
		return Double(is) + fs
	})
	got := CallRet[Double](cb,
		Int(1), Int(2), Int(3), Int(4), Int(5), Int(6), Int(7), Int(8), Int(9), Int(10),
		Double(0.5), Double(1.5), Double(2.5), Double(3.5), Double(4.5),
		Double(5.5), Double(6.5), Double(7.5), Double(8.5), Double(9.5))
	if got != 55+50 {
		t.Fatalf("got %v, want 105", got)
	}
}

// TestCallbackStackFloat32Args exercises float32 stack arguments.
func TestCallbackStackFloat32Args(t *testing.T) {
	requirePuregoStackLayout(t)
	cb := NewCallback(func(f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11 Float) Float {
		return f1 + f2 + f3 + f4 + f5 + f6 + f7 + f8 + f9 + f10 + f11
	})
	got := CallRet[Float](cb,
		Float(0.5), Float(1.5), Float(2.5), Float(3.5), Float(4.5), Float(5.5),
		Float(6.5), Float(7.5), Float(8.5), Float(9.5), Float(10.5))
	if got != 60.5 {
		t.Fatalf("got %v, want 60.5", got)
	}
}

func TestCallbackRetStructTwoFloat32(t *testing.T) {
	cb := NewCallback(func(x, y Float) testPointF {
		return testPointF{X: x * 2, Y: y * 2}
	})
	got := CallRet[testPointF](cb, Float(1.5), Float(2.5))
	if got.X != 3 || got.Y != 5 {
		t.Fatalf("got %+v, want {3 5}", got)
	}
}

func TestCallbackRetStructThreeFloat32(t *testing.T) {
	cb := NewCallback(func(x Float) testVec3 {
		return testVec3{X: x, Y: x + 1, Z: x + 2}
	})
	got := CallRet[testVec3](cb, Float(1.5))
	if got.X != 1.5 || got.Y != 2.5 || got.Z != 3.5 {
		t.Fatalf("got %+v, want {1.5 2.5 3.5}", got)
	}
}

// TestCallbackRetStructFourFloat64 returns a 32-byte homogeneous float aggregate.
func TestCallbackRetStructFourFloat64(t *testing.T) {
	cb := NewCallback(func(a Double) testQuad {
		return testQuad{A: a, B: a * 2, C: a * 3, D: a * 4}
	})
	got := CallRet[testQuad](cb, Double(1.5))
	want := testQuad{A: 1.5, B: 3, C: 4.5, D: 6}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Structs with array fields are not round-tripped here (purego's call
// direction does not flatten arrays); they are covered in callback_clib_test.go.

// TestCallbackRetStructMixed returns a 16-byte struct with mixed int/float fields.
func TestCallbackRetStructMixed(t *testing.T) {
	cb := NewCallback(func(a Int, b Float, c Double) testMixed {
		return testMixed{A: a + 1, B: b + 0.5, C: c + 0.25}
	})
	got := CallRet[testMixed](cb, Int(10), Float(1.5), Double(2.5))
	want := testMixed{A: 11, B: 2, C: 2.75}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructIntPair returns a 16-byte integer struct.
func TestCallbackRetStructIntPair(t *testing.T) {
	cb := NewCallback(func(a, b LongLong) testIntPair {
		return testIntPair{A: a * 2, B: b * 3}
	})
	got := CallRet[testIntPair](cb, LongLong(11), LongLong(22))
	want := testIntPair{A: 22, B: 66}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructSmallInts returns a 12-byte struct (two partially
// filled register chunks).
func TestCallbackRetStructSmallInts(t *testing.T) {
	cb := NewCallback(func(a Int) testSmallInts {
		return testSmallInts{A: a, B: a + 1, C: a + 2}
	})
	got := CallRet[testSmallInts](cb, Int(7))
	want := testSmallInts{A: 7, B: 8, C: 9}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructBig returns a 24-byte struct through the hidden result pointer.
func TestCallbackRetStructBig(t *testing.T) {
	cb := NewCallback(func(a LongLong) testBig {
		return testBig{A: a, B: a * 2, C: a * 3}
	})
	got := CallRet[testBig](cb, LongLong(5))
	want := testBig{A: 5, B: 10, C: 15}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructBigWithArgs verifies the hidden result pointer does
// not shift or corrupt the visible arguments.
func TestCallbackRetStructBigWithArgs(t *testing.T) {
	cb := NewCallback(func(a, b, c, d, e, f, g Int, x Double) testBig {
		return testBig{
			A: LongLong(a + b + c + d),
			B: LongLong(e + f + g),
			C: LongLong(x),
		}
	})
	got := CallRet[testBig](cb, Int(1), Int(2), Int(3), Int(4), Int(5), Int(6), Int(7), Double(42.0))
	want := testBig{A: 10, B: 18, C: 42}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructFloatIntFloat returns a 12-byte struct that is not an HFA.
func TestCallbackRetStructFloatIntFloat(t *testing.T) {
	cb := NewCallback(func() testFloatIntFloat {
		return testFloatIntFloat{A: 1.5, B: 2, C: 3.5}
	})
	got := CallRet[testFloatIntFloat](cb)
	want := testFloatIntFloat{A: 1.5, B: 2, C: 3.5}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestCallbackRetStructOneFloat returns a 4-byte single-float struct.
func TestCallbackRetStructOneFloat(t *testing.T) {
	cb := NewCallback(func(v Float) testOneFloat {
		return testOneFloat{V: v + 0.25}
	})
	got := CallRet[testOneFloat](cb, Float(1.5))
	if got.V != 1.75 {
		t.Fatalf("got %+v, want {1.75}", got)
	}
}

// TestCallbackReentrant calls a callback from inside another callback.
func TestCallbackReentrant(t *testing.T) {
	inner := NewCallback(func(x Double) Double { return x * 2 })
	outer := NewCallback(func(x Double) Double {
		return CallRet[Double](inner, x+1) + 0.5
	})
	if got := CallRet[Double](outer, Double(2)); got != 6.5 {
		t.Fatalf("got %v, want 6.5", got)
	}
}

// The struct-argument round-trips below rely on purego's caller placing the
// struct where the callback reads it, so they only cover the cases where
// purego agrees with the C ABI (all-integer, HFA, and by-reference structs).
// Mixed int/float structs and other cases are covered against a real C
// compiler in callback_clib_test.go.

func TestCallbackArgStructIntPair(t *testing.T) {
	cb := NewCallback(func(p testIntPair) LongLong {
		return p.A*10 + p.B
	})
	if got := CallRet[LongLong](cb, testIntPair{A: 3, B: 4}); got != 34 {
		t.Fatalf("got %v, want 34", got)
	}
}

func TestCallbackArgStructHFA(t *testing.T) {
	cb := NewCallback(func(v testVec3) Float {
		return v.X + v.Y + v.Z
	})
	if got := CallRet[Float](cb, testVec3{X: 1.5, Y: 2.5, Z: 3.5}); got != 7.5 {
		t.Fatalf("got %v, want 7.5", got)
	}
}

func TestCallbackArgStructBig(t *testing.T) {
	cb := NewCallback(func(b testBig) LongLong {
		return b.A + b.B + b.C
	})
	if got := CallRet[LongLong](cb, testBig{A: 10, B: 20, C: 30}); got != 60 {
		t.Fatalf("got %v, want 60", got)
	}
}
