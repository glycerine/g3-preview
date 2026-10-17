//go:build g3debug

package smt2

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2SortedVariablesSourceAssertionReadsProcessDebugAndRollsBack(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	if err != nil {
		t.Fatal(err)
	}
	debug, err := diagnostic.NewDebugContext(region, budget)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := diagnostic.BindProcessDebugContext(debug)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := diagnostic.BindProcessDebugContext(previous); err != nil {
			t.Fatal(err)
		}
		if err := region.Close(); err != nil {
			t.Fatal(err)
		}
		if budget.Used() != 0 {
			t.Fatalf("debug owner retained %d bytes", budget.Used())
		}
	}()
	if err := debug.Close(); err != nil {
		t.Fatal(err)
	}
	var env smt2LocalEnv
	var exprs localExprStack
	baseline := m.Stats().LiveReferences
	level, err := env.bindSortedVariables(m, []symbol.Symbol{symbol.Numerical(71)}, []ast.ASTHandle{sort}, 4, &exprs)
	if !errors.Is(err, arena.ErrClosed) || level != 4 {
		t.Fatalf("closed process debug context: level=%d err=%v", level, err)
	}
	if len(env.limits) != 0 || len(env.trail) != 0 || env.contains(symbol.Numerical(71)) || len(exprs.refs) != 0 || m.Stats().LiveReferences != baseline {
		t.Fatal("source assertion failure left a sorted-variable scope or owning reference")
	}
}
