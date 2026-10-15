package numeral

import (
	"math"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerArrayRuntimeAliasRejectsWrappedAndUnreachedOffsets(t *testing.T) {
	owner := &umemory.Runtime{}
	view := IntegerArrayView{Runtime: owner, Count: 2, Offset: math.MaxUint64 - 1, Stride: 4}
	if view.SameRuntimeElement(owner, umemory.Block{}, 2, 1) {
		t.Fatal("wrapped offset was accepted as a borrowed element")
	}
	view.Offset, view.Stride = 12, 12
	if !view.SameRuntimeElement(owner, umemory.Block{}, 24, 1) {
		t.Fatal("valid borrowed element was rejected")
	}
	if view.SameRuntimeElement(owner, umemory.Block{}, 36, 2) {
		t.Fatal("element past source count was accepted")
	}
}
