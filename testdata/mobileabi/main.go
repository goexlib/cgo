// 本例程由独立原生宿主调用，不直接运行。构建、覆盖范围和预期见 README.md。
package main

import (
	"example.com/cgoabi/bridge"
	"fmt"
	ffi "github.com/goexlib/cgo"
	"runtime"
)

type Point struct{ X, Y float64 }
type Mixed struct {
	N uint64
	X float64
}
type Big struct {
	A, B, C uint64
	D       float64
}
type Int3 struct{ V [3]uint32 }
type Float3 struct{ V [3]float32 }
type Float5 struct{ V [5]float64 }
type NestedHFA struct{ A, B Point }
type OnlyFloat struct{ X float64 }
type NestedMixed struct {
	N uint64
	P OnlyFloat
}

const variant int64 = 7

//go:noinline
func grow(n int) int64 {
	var block [256]int64
	block[n%256] = int64(n)
	if n == 0 {
		return block[0]
	}
	return block[n%256] + grow(n-1)
}
func require(ok bool, detail any) {
	if !ok {
		panic(detail)
	}
}
func init() {
	names := []string{"fixture_float", "fixture_point", "fixture_mixed", "fixture_big", "fixture_cb_float", "fixture_cb_point", "fixture_cb_mixed", "fixture_cb_big", "fixture_cb_packed", "fixture_cb_spill", "fixture_packed", "fixture_reverse", "fixture_nested", "fixture_array", "fixture_mixed_spill", "fixture_hfa_spill", "fixture_packed_hfa", "fixture_big_after_float", "fixture_big_last_reg", "fixture_float5", "fixture_nested_mixed", "fixture_keepalive", "fixture_cb_mixed_spill", "fixture_cb_packed_hfa", "fixture_cb_big_stack", "fixture_int3_stack", "fixture_cb_int3_stack"}
	symbols := make([]uintptr, len(names))
	for i, name := range names {
		v, err := ffi.Dlsym(ffi.RTLD_DEFAULT, name)
		if err != nil {
			panic(err)
		}
		symbols[i] = v
	}
	cbFloat := ffi.NewCallback(func(a float64, b float32, c uint64) float64 { return a + float64(b) + float64(c) })
	cbPoint := ffi.NewCallback(func(p Point) Point { return Point{p.X + 1, p.Y + 2} })
	cbMixed := ffi.NewCallback(func(p Mixed) Mixed { return Mixed{p.N + 10, p.X + 0.5} })
	cbBig := ffi.NewCallback(func(p Big) Big { return Big{p.A + 10, p.B + 20, p.C + 30, p.D + 0.5} })
	cbPacked := ffi.NewCallback(func(a, b, c, d, e, f, g, h int64, i int8, j uint8, k int16, l uint16, m int32, n uint32, o int64) int64 {
		return a + b + c + d + e + f + g + h + int64(i) + int64(j) + int64(k) + int64(l) + int64(m) + int64(n) + o
	})
	cbSpill := ffi.NewCallback(func(a, b, c, d, e, f, g, h, i, j int64, k, l, m, n, o, p, q, r, s, t float64) float64 {
		return float64(a+b+c+d+e+f+g+h+i+j) + k + l + m + n + o + p + q + r + s + t
	})
	cbMixedSpill := ffi.NewCallback(func(a, b, c, d, e, f, g uint64, p Mixed, tail uint64, z float64) float64 {
		return float64(a+b+c+d+e+f+g+p.N+tail) + p.X + z
	})
	cbGC := ffi.NewCallback(func() { runtime.GC(); require(grow(64) == 2080, "callback stack checksum") })
	cbPackedHFA := ffi.NewCallback(func(a, b, c, d, e, f, g, h uint64, i, j, k, l, m, n, o, p float64, before int8, value Float3, after int8) float64 {
		return float64(a+b+c+d+e+f+g+h) + i + j + k + l + m + n + o + p + float64(before) + float64(value.V[0]+value.V[1]+value.V[2]) + float64(after)
	})
	cbBigStack := ffi.NewCallback(func(a, b, c, d, e, f, g, h uint64, before int8, value Big, after uint64) float64 {
		return float64(a+b+c+d+e+f+g+h) + float64(before) + float64(value.A+value.B+value.C) + value.D + float64(after)
	})
	cbInt3Stack := ffi.NewCallback(func(a, b, c, d, e, f, g, h uint64, before int8, value Int3, after int8) float64 {
		return float64(a+b+c+d+e+f+g+h) + float64(before) + float64(value.V[0]+value.V[1]+value.V[2]) + float64(after)
	})
	bridge.Register(func(mode int64) (result int64) {
		defer func() {
			if e := recover(); e != nil {
				fmt.Printf("CASE %d FAIL: %v\n", mode, e)
				result = -1
			}
		}()
		if mode == 0 {
			return variant
		}
		switch mode {
		case 1:
			v := ffi.CallRet[float64](symbols[0], float64(1.25), float32(2.5), uint64(7))
			require(v == 10.75, v)
			var typed func(float64, float32, uint64) float64
			require(ffi.RegisterFunc(&typed, symbols[0]) == nil, "typed float registration")
			require(typed(1.25, 2.5, 7) == 10.75, "typed native float call")
			var expanded func(...any) float64
			require(ffi.RegisterFunc(&expanded, symbols[0]) == nil, "expanded registration")
			require(expanded(float64(1.25), float32(2.5), uint64(7)) == 10.75, "expanded native call")
		case 2:
			v := ffi.CallRet[Point](symbols[1], Point{1.5, 2.25}, float64(2))
			require(v == (Point{3, 4.5}), v)
			var typed func(Point, float64) Point
			require(ffi.RegisterFunc(&typed, symbols[1]) == nil, "HFA registration")
			require(typed(Point{1.5, 2.25}, 2) == (Point{3, 4.5}), "typed native HFA call")
		case 3:
			v := ffi.CallRet[Mixed](symbols[2], Mixed{7, 1.25}, uint64(10), float64(.5))
			require(v == (Mixed{17, 1.75}), v)
			var typed func(Mixed, uint64, float64) Mixed
			require(ffi.RegisterFunc(&typed, symbols[2]) == nil, "mixed registration")
			require(typed(Mixed{7, 1.25}, 10, .5) == (Mixed{17, 1.75}), "typed native mixed call")
		case 4:
			v := ffi.CallRet[Big](symbols[3], Big{1, 2, 3, 4.5}, uint64(10))
			require(v == (Big{11, 22, 33, 5}), v)
			var typed func(Big, uint64) Big
			require(ffi.RegisterFunc(&typed, symbols[3]) == nil, "indirect registration")
			require(typed(Big{1, 2, 3, 4.5}, 10) == (Big{11, 22, 33, 5}), "typed native indirect call")
		case 5:
			v := ffi.CallRet[float64](symbols[4], cbFloat)
			require(v == 10.75, v)
		case 6:
			v := ffi.CallRet[float64](symbols[5], cbPoint)
			require(v == 11, v)
		case 7:
			v := ffi.CallRet[float64](symbols[6], cbMixed)
			require(v == 18.75, v)
		case 8:
			v := ffi.CallRet[float64](symbols[7], cbBig)
			require(v == 71, v)
		case 9:
			v := ffi.CallRet[int64](symbols[8], cbPacked)
			require(v == 54, v)
		case 10:
			v := ffi.CallRet[float64](symbols[9], cbSpill)
			require(v == 105, v)
		case 11:
			v := ffi.CallRet[int64](symbols[10], int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8), int8(-9), uint8(10), int16(-11), uint16(12), int32(-13), uint32(14), int64(15))
			require(v == 54, v)
		case 12:
			b := make([]byte, 8192+int(mode))
			b[0] = 123
			runtime.GC()
			require(grow(64) == 2080, "stack checksum")
			require(b[0] == 123, "memory checksum")
			runtime.KeepAlive(b)
		case 13:
			type Reverse struct {
				X float64
				N uint64
			}
			v := ffi.CallRet[Reverse](symbols[11], Reverse{1.25, 7})
			require(v == (Reverse{1.75, 17}), v)
		case 14:
			v := ffi.CallRet[NestedHFA](symbols[12], NestedHFA{Point{1, 2}, Point{3, 4}})
			require(v == (NestedHFA{Point{2, 4}, Point{6, 8}}), v)
		case 15:
			v := ffi.CallRet[Float3](symbols[13], Float3{[3]float32{1.5, 2.5, 3.5}})
			require(v == (Float3{[3]float32{3, 5, 7}}), v)
		case 16:
			v := ffi.CallRet[float64](symbols[14], uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint64(7), Mixed{9, 1.25}, uint64(11), float64(.5))
			require(v == 49.75, v)
		case 17:
			v := ffi.CallRet[float64](symbols[15], float64(1), float64(2), float64(3), float64(4), float64(5), float64(6), Float3{[3]float32{1.5, 2.5, 3.5}}, float32(.25))
			require(v == 28.75, v)
		case 18:
			v := ffi.CallRet[float64](symbols[16], uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint64(7), uint64(8), float64(1), float64(2), float64(3), float64(4), float64(5), float64(6), float64(7), float64(8), int8(9), Float3{[3]float32{1.5, 2.5, 3.5}}, int8(11))
			require(v == 99.5, v)
		case 19:
			v := ffi.CallRet[float64](symbols[17], float64(1), float64(2), float64(3), float64(4), float64(5), float64(6), float64(7), float64(8), float64(9), float64(10), Big{1, 2, 3, 4.5}, uint64(7))
			require(v == 72.5, v)
		case 20:
			v := ffi.CallRet[float64](symbols[18], uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint64(7), Big{1, 2, 3, 4.5}, uint64(11))
			require(v == 49.5, v)
		case 21:
			v := ffi.CallRet[Float5](symbols[19], Float5{[5]float64{1, 2, 3, 4, 5}})
			require(v == (Float5{[5]float64{2, 4, 6, 8, 10}}), v)
		case 22:
			v := ffi.CallRet[NestedMixed](symbols[20], NestedMixed{7, OnlyFloat{1.25}})
			require(v == (NestedMixed{17, OnlyFloat{1.75}}), v)
		case 23:
			data := []byte("abc")
			v := ffi.CallRet[int64](symbols[21], uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint64(7), uint64(8), "goui ffi", data, uint64(len(data)), cbGC)
			require(v == 1, v)
			runtime.KeepAlive(data)
		case 24:
			v := ffi.CallRet[float64](symbols[22], cbMixedSpill)
			require(v == 49.75, v)
		case 25:
			v := ffi.CallRet[float64](symbols[23], cbPackedHFA)
			require(v == 99.5, v)
		case 26:
			v := ffi.CallRet[float64](symbols[24], cbBigStack)
			require(v == 66.5, v)
		case 27:
			v := ffi.CallRet[float64](symbols[25], uint64(1), uint64(2), uint64(3), uint64(4), uint64(5), uint64(6), uint64(7), uint64(8), int8(9), Int3{[3]uint32{1, 2, 3}}, int8(11))
			require(v == 62, v)
		case 28:
			v := ffi.CallRet[float64](symbols[26], cbInt3Stack)
			require(v == 62, v)
		default:
			panic("unknown mode")
		}
		return 1
	})
}
func main() {}
