package example_test

import (
	"testing"

	"github.com/9506hqwy/template-go-module/pkg/example"
)

//revive:disable:add-constant

func Test_add(t *testing.T) {
	ret := example.Add(1, 2)
	if ret != 3 {
		t.Error("Failed.", ret)
	}
}

func Test_sub(t *testing.T) {
	ret := example.Sub(2, 1)
	if ret != 1 {
		t.Error("Failed.", ret)
	}
}

func Benchmark_add(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = example.Add(1, 2)
	}
}

func Benchmark_sub(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = example.Sub(2, 1)
	}
}

//revive:enable:add-constant
