//go:build g3debug

package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFPABV2RMReachedSourceAssertionSwitch(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	bv3, err := m.BitVectorSort(3)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBV2RM, nil, []ast.ASTHandle{bv3}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	numeral, err := m.BitVectorNumeralUnsigned(1, 3)
	astTestMust(t, err)
	symbolic, err := m.MkVar(0, bv3)
	astTestMust(t, err)
	astTestPin(t, m, symbolic)
	out := astTestPin(t, m, symbolic)
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	astTestMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	astTestMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		astTestMust(t, e)
		astTestMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal("debug owner leak", budget.Used())
		}
	})
	astTestMust(t, debug.EnableAssertions(false))
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{numeral}), out)
	astTestMust(t, err)
	if status != RewriteDone {
		t.Fatal("disabled assertion switch prevented numeral conversion", status)
	}
	before, err := m.RefGet(out)
	astTestMust(t, err)
	astTestMust(t, debug.Close())
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{symbolic}), out)
	if err != nil || status != RewriteFailed {
		t.Fatal("symbolic input reached assertion switch", status, err)
	}
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{numeral}), out)
	if !errors.Is(err, arena.ErrClosed) || status != RewriteFailed {
		t.Fatal("numeral did not read closed process debug owner", status, err)
	}
	after, err := m.RefGet(out)
	astTestMust(t, err)
	if after != before {
		t.Fatal("stopped assertion switch changed output")
	}
}
