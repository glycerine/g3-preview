package api

import (
	"testing"

	"github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/util/params"
)

func TestAPIEstimatedAllocSizeReadsProcessCounterWithoutInitializing(t *testing.T) {
	if params.BorrowProcessGlobalParamsService() != nil {
		t.Fatal("unexpected generated process before memory getter")
	}
	if got := EstimatedAllocSize(); got != 0 || params.BorrowProcessGlobalParamsService() != nil {
		t.Fatal("memory getter initialized process", got)
	}
	if err := engine.InitializeDefaultProcess(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := engine.FinalizeDefaultProcess(); err != nil {
			t.Error(err)
		}
	})
	g := params.BorrowProcessGlobalParamsService()
	runtime := g.Context.Numerals().RuntimeOwner()
	p := runtime.ProcessOwner()
	before := p.AllocationSize()
	if got := EstimatedAllocSize(); got != before {
		t.Fatal("initial process counter", got, before)
	}
	block, err := runtime.Allocate(24)
	if err != nil {
		t.Fatal(err)
	}
	after := p.AllocationSize()
	if got := EstimatedAllocSize(); got != after || after <= before {
		t.Fatal("allocated process counter", got, before, after)
	}
	if err := runtime.Deallocate(&block); err != nil {
		t.Fatal(err)
	}
	if got := EstimatedAllocSize(); got != p.AllocationSize() {
		t.Fatal("released process counter", got, p.AllocationSize())
	}
}
