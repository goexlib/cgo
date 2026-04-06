package cgo

import "unsafe"

func Memset(p unsafe.Pointer, val byte, size uintptr) {
	dst := GoSliceNTemp[byte](p, int(size))
	for i := range dst {
		dst[i] = val
	}
}

func Memcpy(dst, src unsafe.Pointer, size uintptr) {
	copy(GoSliceNTemp[byte](dst, int(size)), GoSliceNTemp[byte](src, int(size)))
}
