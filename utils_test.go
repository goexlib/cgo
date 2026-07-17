package cgo

import "testing"

func TestMemory(t *testing.T) {
	p := Malloc(3)
	defer Free(p)
	Memset(p, 67, 3)
	if got := GoStringNTemp(p, 3); got != "CCC" {
		t.Fatalf("Memset filled %q, want \"CCC\"", got)
	}
}
