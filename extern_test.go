package cgo

import "testing"

func Test_Extern(t *testing.T) {
	lib, err := LoadLibrary("testdata/test.dll")
	if err != nil {
		t.Fatal(err)
	}
	count, err := GetExternVariant[Int](lib, "count")
	if err != nil {
		t.Fatal(err)
	}

	fnIncrement, err := lib.GetSymbol("increment")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("count = ", *count)
	Call(fnIncrement)
	t.Log("count = ", *count)
	*count = 999
	Call(fnIncrement)
	t.Log("count = ", *count)
}
