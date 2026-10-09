package ast

import (
	"testing"
)

func TestASTProofArgumentsDoNotChargeASTBudget(t *testing.T) {
	m := astTestManager(t)
	h := m.external(m.trueApp)
	before := m.BudgetField.Used()
	var args astProofArguments
	for i := 0; i < 16; i++ {
		if e := args.push(m, h); e != nil {
			t.Fatal(e)
		}
	}
	if len(args.slice()) != 16 || args.slice()[15] != h {
		t.Fatal("contents")
	}
	args.close(m)
	if after := m.BudgetField.Used(); after != before {
		t.Fatal("inline scratch charged", before, after)
	}
}
