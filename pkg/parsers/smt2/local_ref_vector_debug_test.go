//go:build g3debug

package smt2

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSMT2LocalRefVectorSourceAssertionsBeforeRelease(t *testing.T) {
	m := localLookupManager(t)
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	term, err := m.MkVar(0, sort)
	if err != nil {
		t.Fatal(err)
	}
	var sorts localSortStack
	if err := sorts.push(m, sort); err != nil {
		t.Fatal(err)
	}
	ref, err := m.NewRef(term)
	if err != nil {
		t.Fatal(err)
	}
	var exprs localExprStack
	exprs.pushOwned(ref)

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
	for _, call := range []func() error{
		func() error { return sorts.shrink(m, 0) },
		func() error { return exprs.shrink(m, 0) },
		func() error { return sorts.pop(m) },
	} {
		if err := call(); !errors.Is(err, arena.ErrClosed) || len(sorts.refs) != 1 || len(exprs.refs) != 1 {
			t.Fatal("source ref-vector assertion must precede reference release", err, len(sorts.refs), len(exprs.refs))
		}
	}
	if err := sorts.reset(m); err != nil {
		t.Fatal("source reset must bypass shrink assertion", err)
	}
	if err := exprs.reset(m); err != nil {
		t.Fatal("source reset must bypass shrink assertion", err)
	}
	if len(sorts.refs) != 0 || len(exprs.refs) != 0 {
		t.Fatal("reset retained references after a stopped shrink")
	}
}
