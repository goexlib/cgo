//go:build linux || darwin

package cgo

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestARM64RegisterTypedAndExpanded(t *testing.T) {
	type point struct{ X, Y float64 }
	address := NewCallback(func(p point, scale float64) point { return point{p.X * scale, p.Y * scale} })
	var fixed func(point, float64) point
	if err := RegisterFunc(&fixed, address); err != nil {
		t.Fatal(err)
	}
	if got := fixed(point{1.5, 2.5}, 2); got != (point{3, 5}) {
		t.Fatalf("固定签名返回 %v", got)
	}
	var expanded func(...any) point
	if err := RegisterFunc(&expanded, address); err != nil {
		t.Fatal(err)
	}
	if got := expanded(point{2, 3}, float64(3)); got != (point{6, 9}) {
		t.Fatalf("展开签名返回 %v", got)
	}
	if got := CallRet[point](address, point{3, 4}, float64(4)); got != (point{12, 16}) {
		t.Fatalf("CallRet 返回 %v", got)
	}
}

func TestARM64RegisteredCallSurvivesReentry(t *testing.T) {
	address := NewCallback(func(value *int64) int64 {
		runtime.GC()
		reentryGrow(32)
		*value += 7
		return *value
	})
	var function func(*int64) int64
	if err := RegisterFunc(&function, address); err != nil {
		t.Fatal(err)
	}
	value := int64(10)
	if got := function(&value); got != 17 || value != 17 {
		t.Fatalf("重入后结果 %d/%d，预期 17", got, value)
	}
}

func TestARM64RegisterRejectsInvalidTypes(t *testing.T) {
	address := NewCallback(func() int { return 3 })
	var bad func(map[string]int)
	var many func() (int, int)
	var missing func()
	for _, tc := range []struct {
		name        string
		destination any
		address     uintptr
	}{
		{"空输出", nil, address}, {"非指针", func() {}, address},
		{"不支持的参数", &bad, address}, {"多个返回", &many, address}, {"空入口", &missing, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := RegisterFunc(tc.destination, tc.address); err == nil {
				t.Fatal("非法注册没有返回错误")
			}
		})
	}
	if bad != nil || many != nil || missing != nil {
		t.Fatal("注册失败改变了输出函数")
	}
}

func TestARM64CallPlanBoundsIndirectValues(t *testing.T) {
	// 九个整数后的大结构体只占一个栈指针槽，不能按整个结构体大小拒绝。
	types := make([]reflect.Type, 9)
	for i := range types {
		types[i] = reflect.TypeOf(uint64(0))
	}
	types = append(types, reflect.TypeOf(struct{ Data [1024]byte }{}))
	plan := newARM64CallPlan(types, nil)
	loc := plan.args[len(plan.args)-1]
	if loc.kind != cbArgIndirect || loc.src != cbSrcStack || loc.soff != 8 {
		t.Fatalf("间接参数位置错误：%+v", loc)
	}
	large := make([]reflect.Type, 8+arm64CallStackBytes/8+1)
	for i := range large {
		large[i] = reflect.TypeOf(uint64(0))
	}
	defer func() {
		p := recover()
		if p == nil || !strings.Contains(p.(string), "stack arguments") {
			t.Fatalf("栈越界没有明确拒绝：%v", p)
		}
	}()
	newARM64CallPlan(large, nil)
}
