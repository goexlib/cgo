package cgo

import "testing"

func Test_Call(t *testing.T) {
	lib, err := LoadLibrary("testdata/test.dll")
	if err != nil {
		t.Fatal(err)
	}

	fnAdd, err := lib.GetSymbol("add")
	if err != nil {
		t.Fatal(err)
	}
	if got := CallRet[int](uintptr(fnAdd), 1, 2); got != 3 {
		t.Fatalf("add(1, 2) = %d, want 3", got)
	}

	fnGetFloat, err := lib.GetSymbol("getFloat")
	if err != nil {
		t.Fatal(err)
	}
	if got := CallRet[Float](fnGetFloat); got != Float(3.14) {
		t.Fatalf("getFloat() = %v, want 3.14", got)
	}

	fnGetDouble, err := lib.GetSymbol("getDouble")
	if err != nil {
		t.Fatal(err)
	}
	if got := CallRet[Double](fnGetDouble); got != Double(3.1415) {
		t.Fatalf("getDouble() = %v, want 3.1415", got)
	}

	fnPrint, err := lib.GetSymbol("print")
	if err != nil {
		t.Fatal(err)
	}
	Call(uintptr(fnPrint), Int(1), Float(1.2), Double(3.4))

	fnPrintString, err := lib.GetSymbol("printString")
	if err != nil {
		t.Fatal(err)
	}
	Call(fnPrintString.Addr(), "Hello cgo!")

	type Point struct {
		X, Y Float
	}

	fnAddPos, err := lib.GetSymbol("addPos")
	if err != nil {
		t.Fatal(err)
	}
	pos := CallRet[Point](uintptr(fnAddPos), Float(1.2), Float(3.4))
	if want := (Point{X: Float(1.2) * 2, Y: Float(3.4) * 2}); pos != want {
		t.Fatalf("addPos(1.2, 3.4) = %+v, want %+v", pos, want)
	}

	fnGetName, err := lib.GetSymbol("getName")
	if err != nil {
		t.Fatal(err)
	}
	if got := CallRet[string](uintptr(fnGetName)); got != "test" {
		t.Fatalf("getName() = %q, want \"test\"", got)
	}

	type Data struct {
		A Int
		B Float
		C Double
	}

	fnGetData, err := lib.GetSymbol("getData")
	if err != nil {
		t.Fatal(err)
	}
	data := CallRet[Data](uintptr(fnGetData), Int(10), Float(11.11), Double(12.12))
	if want := (Data{A: 10, B: Float(11.11), C: Double(12.12)}); data != want {
		t.Fatalf("getData() = %+v, want %+v", data, want)
	}
}
