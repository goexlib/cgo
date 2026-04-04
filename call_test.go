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

	t.Log(CallRet[int](uintptr(fnAdd), 1, 2))

	fnGetFloat, err := lib.GetSymbol("getFloat")
	if err != nil {
		t.Fatal(err)
	}

	f32 := CallRet[Float](fnGetFloat)
	t.Log(f32)

	fnGetDouble, err := lib.GetSymbol("getDouble")
	if err != nil {
		t.Fatal(err)
	}

	f64 := CallRet[Double](fnGetDouble)
	t.Log(f64)

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
	t.Log(pos)

	fnGetName, err := lib.GetSymbol("getName")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(CallRet[string](uintptr(fnGetName)))

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
	t.Log(data)

	//fnPrintData, err := lib.GetSymbol("printData")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//Call(uintptr(fnPrintData), data)
}
