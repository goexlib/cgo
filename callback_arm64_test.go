//go:build linux || darwin

package cgo

import (
	"reflect"
	"runtime"
	"testing"
)

func TestAAPCS64StackOffsets(t *testing.T) {
	type triple struct{ V [3]float32 }
	type large struct{ V [5]uint64 }
	u64, f64 := reflect.TypeOf(uint64(0)), reflect.TypeOf(float64(0))
	prefix := make([]reflect.Type, 8)
	for i := range prefix {
		prefix[i] = u64
	}
	cases := []struct {
		name string
		args []reflect.Type
		want []uintptr
	}{
		{"narrow", []reflect.Type{reflect.TypeOf(int8(0)), reflect.TypeOf(uint8(0)), reflect.TypeOf(int16(0)), reflect.TypeOf(uint16(0)), reflect.TypeOf(int32(0)), reflect.TypeOf(uint32(0)), u64}, []uintptr{0, 8, 16, 24, 32, 40, 48}},
		{"ordinary-after-narrow", []reflect.Type{reflect.TypeOf(int8(0)), reflect.TypeOf(struct{ V [3]uint32 }{}), reflect.TypeOf(int8(0))}, []uintptr{0, 8, 24}},
		{"indirect-after-narrow", []reflect.Type{reflect.TypeOf(int8(0)), reflect.TypeOf(large{}), u64}, []uintptr{0, 8, 16}},
	}
	apple := runtime.GOOS == "darwin" || runtime.GOOS == "ios"
	if apple {
		cases[0].want = []uintptr{0, 1, 2, 4, 8, 12, 16}
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := append(append([]reflect.Type(nil), prefix...), c.args...)
			plan := buildCbArgPlan(reflect.FuncOf(args, nil, false), false)
			for i, want := range c.want {
				loc := plan[len(prefix)+i]
				if loc.src != cbSrcStack || loc.soff != want {
					t.Errorf("argument %d: src=%v offset=%d, want stack offset=%d", i, loc.src, loc.soff, want)
				}
			}
		})
	}
	t.Run("hfa-after-narrow", func(t *testing.T) {
		args := append([]reflect.Type(nil), prefix...)
		for i := 0; i < 8; i++ {
			args = append(args, f64)
		}
		args = append(args, reflect.TypeOf(int8(0)), reflect.TypeOf(triple{}), reflect.TypeOf(int8(0)))
		plan := buildCbArgPlan(reflect.FuncOf(args, nil, false), false)
		want := []uintptr{0, 8, 24}
		if apple {
			want = []uintptr{0, 4, 16}
		}
		for i, offset := range want {
			if loc := plan[16+i]; loc.src != cbSrcStack || loc.soff != offset {
				t.Errorf("argument %d: got src=%v offset=%d, want stack offset=%d", i, loc.src, loc.soff, offset)
			}
		}
	})
}
