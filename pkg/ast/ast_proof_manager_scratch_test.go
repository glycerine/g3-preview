package ast

import (
	"testing"
)

func TestASTProofInlineArgumentsDoNotAllocate(t *testing.T) {
	m := astTestManager(t)
	h := m.external(m.trueApp)
	before := m.BudgetField.Used()
	allocs := testing.AllocsPerRun(100, func() {
		var args astProofArguments
		for i := 0; i < 16; i++ {
			if e := args.push(m, h); e != nil {
				panic(e)
			}
		}
		if len(args.slice()) != 16 || args.slice()[15] != h {
			panic("inline contents")
		}
		args.close(m)
	})
	if allocs != 0 {
		t.Fatalf("source inline ptr_buffer heap allocations: %g", allocs)
	}
	if after := m.BudgetField.Used(); after != before {
		t.Fatal("inline scratch charged", before, after)
	}
}
