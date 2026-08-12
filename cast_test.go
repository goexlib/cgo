package cgo

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestCString(t *testing.T) {
	s := "hello"
	p := CString(s)
	if got := GoString(p); got != s {
		t.Fatalf("CString content = %q, want %q", got, s)
	}
	if *(*byte)(unsafe.Add(p, len(s))) != 0 {
		t.Fatal("CString missing NUL terminator")
	}
	if p == unsafe.Pointer(unsafe.StringData(s)) {
		t.Fatal("CString must allocate an independent copy")
	}
	Free(p)
}

func TestCStringNulTerminated(t *testing.T) {
	s := "hi\x00"
	p := CString(s)
	if got := GoStringN(p, len(s)); got != s {
		t.Fatalf("CString content = %q, want %q", got, s)
	}
	if *(*byte)(unsafe.Add(p, len(s))) != 0 {
		t.Fatal("CString missing NUL terminator")
	}
	if p == unsafe.Pointer(unsafe.StringData(s)) {
		t.Fatal("CString must allocate an independent copy even when already NUL-terminated")
	}
	Free(p)
}

func TestCStringEmpty(t *testing.T) {
	p := CString("")
	if p == nil {
		t.Fatal(`CString("") must not be nil`)
	}
	if *(*byte)(p) != 0 {
		t.Fatal(`CString("") must point at a NUL byte`)
	}
	Free(p)
}

func TestCStringTemp(t *testing.T) {
	s := "hi\x00"
	if p := CStringTemp(s); p != unsafe.Pointer(unsafe.StringData(s)) {
		t.Fatal("CStringTemp must alias s when already NUL-terminated")
	}
	runtime.KeepAlive(s)

	s2 := "hello"
	p2 := CStringTemp(s2)
	if got := GoString(p2); got != s2 {
		t.Fatalf("CStringTemp content = %q, want %q", got, s2)
	}
	runtime.KeepAlive(p2)

	if p3 := CStringTemp(""); p3 != nil {
		t.Fatal(`CStringTemp("") must be nil`)
	}
}
