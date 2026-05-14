package cgo

import "testing"

func TestMemory(t *testing.T) {
	p := Malloc(3)
	defer Free(p)
	Memset(p, 67, 3)
	t.Log(GoStringNTemp(p, 3))
}
