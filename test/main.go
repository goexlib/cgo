package main

import (
	"fmt"
	"github.com/goexlib/cgo"
	"runtime"
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var libPath string
	switch runtime.GOOS {
	case "windows":
		libPath = "testdata/test.dll"
	case "linux":
		libPath = "testdata/libtest.so"
	case "darwin":
		libPath = "testdata/libtest.dylib"
	}

	lib, err := cgo.LoadLibrary(libPath)
	panicIf(err)
	defer lib.Close()

	fnAdd, err := lib.GetSymbol("add")
	panicIf(err)

	fmt.Println("add(1, 2) = ", cgo.CallRet[int](fnAdd, 1, 2))

	fnGetFloat, err := lib.GetSymbol("getFloat")
	panicIf(err)

	f32 := cgo.CallRet[cgo.Float](fnGetFloat)
	fmt.Println("getFloat() = ", f32)

	fnGetDouble, err := lib.GetSymbol("getDouble")
	panicIf(err)

	f64 := cgo.CallRet[cgo.Double](fnGetDouble)
	fmt.Println("getDouble() = ", f64)

	fnPrint, err := lib.GetSymbol("print")
	panicIf(err)

	cgo.Call(fnPrint.Addr(), cgo.Int(1), cgo.Float(1.2), cgo.Double(3.4))

	fnPrintString, err := lib.GetSymbol("printString")
	panicIf(err)

	cgo.Call(fnPrintString.Addr(), "Hello cgo!")

	type Point struct {
		X, Y cgo.Float
	}

	fnAddPos, err := lib.GetSymbol("addPos")
	panicIf(err)

	pos := cgo.CallRet[Point](uintptr(fnAddPos), cgo.Float(1.2), cgo.Float(3.4))
	fmt.Println("addPos(1.2, 3.4) = ", pos)

	fnGetName, err := lib.GetSymbol("getName")
	panicIf(err)

	fmt.Println("getName() = ", cgo.CallRet[string](fnGetName.Addr()))

	type Data struct {
		A cgo.Int
		B cgo.Float
		C cgo.Double
	}

	fnGetData, err := lib.GetSymbol("getData")
	panicIf(err)

	data := cgo.CallRet[Data](uintptr(fnGetData), cgo.Int(10), cgo.Float(11.11), cgo.Double(12.12))
	fmt.Println("getData(10, 11.11, 12.12) = ", data)

	fnPrintData, err := lib.GetSymbol("printData")
	panicIf(err)

	cgo.Call(uintptr(fnPrintData), data)

	count, err := cgo.GetExternVariant[cgo.Int](lib, "count")
	panicIf(err)

	fnIncrement, err := lib.GetSymbol("increment")
	panicIf(err)

	fmt.Println("count = ", *count)
	cgo.Call(fnIncrement)
	fmt.Println("count = ", *count)
	*count = 999
	cgo.Call(fnIncrement)
	fmt.Println("count = ", *count)
}
