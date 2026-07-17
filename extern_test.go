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

	if *count != 0 {
		t.Fatalf("initial count = %d, want 0", *count)
	}
	Call(fnIncrement)
	if *count != 1 {
		t.Fatalf("count after increment = %d, want 1", *count)
	}
	*count = 999
	Call(fnIncrement)
	if *count != 1000 {
		t.Fatalf("count after write and increment = %d, want 1000", *count)
	}
}
