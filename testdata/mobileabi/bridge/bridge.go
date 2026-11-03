package bridge

/*
#include <stdint.h>
*/
import "C"

var handler func(int64) int64

func Register(f func(int64) int64) { handler = f }

//export goui_probe
func goui_probe(value C.int64_t) C.int64_t {
	if handler == nil {
		return -999
	}
	return C.int64_t(handler(int64(value)))
}
