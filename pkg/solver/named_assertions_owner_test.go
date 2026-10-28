package solver

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type namedOwnerCore struct {
	namedCoreFixture
	assert func(ast.ASTHandle) error
}

func (c *namedOwnerCore) AssertFormula(h ast.ASTHandle) error { return c.assert(h) }
func namedOwnerConst(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	b, e := m.MkBoolSort()
	solverMust(t, e)
	symbol, e := m.Symbols.Intern(name)
	solverMust(t, e)
	h, e := m.MkConst(symbol, b)
	solverMust(t, e)
	ref, e := m.NewRef(h)
	solverMust(t, e)
	t.Cleanup(func() { solverMust(t, m.DelRef(&ref)) })
	return h
}
func TestNamedAssertionImplicationOwnedAcrossCoreAndErrors(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			m := solverTestManager(t)
			formula := namedOwnerConst(t, m, "owned-na-formula")
			tag := namedOwnerConst(t, m, "owned-na-tag")
			stop := errors.New("core assertion stopped")
			core := &namedOwnerCore{}
			n, e := NewNamedAssertions(m, core)
			solverMust(t, e)
			t.Cleanup(func() { solverMust(t, n.Close()) })
			var implication ast.ASTHandle
			var rootRefs, formulaRefs, tagRefs uint32
			core.assert = func(h ast.ASTHandle) error {
				implication = h
				var err error
				rootRefs, err = m.GetRefCount(h)
				solverMust(t, err)
				formulaRefs, err = m.GetRefCount(formula)
				solverMust(t, err)
				tagRefs, err = m.GetRefCount(tag)
				solverMust(t, err)
				if len(n.assumptions) != 1 {
					t.Fatal("assumption not published before core call")
				}
				if fail {
					return stop
				}
				return nil
			}
			err := n.AssertTracked(formula, tag)
			if fail {
				if !errors.Is(err, stop) {
					t.Fatal("lost core error", err)
				}
			} else {
				solverMust(t, err)
			}
			if rootRefs != 1 || formulaRefs != 2 || tagRefs != 3 {
				t.Fatal("source new_t ownership missing at core call", rootRefs, formulaRefs, tagRefs)
			}
			if _, err = m.GetKind(implication); !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("implication local survived core call", err)
			}
			refs, e := m.GetRefCount(formula)
			solverMust(t, e)
			if refs != 1 {
				t.Fatal("formula retained by ended local", refs)
			}
			refs, e = m.GetRefCount(tag)
			solverMust(t, e)
			if refs != 2 || len(n.assumptions) != 1 {
				t.Fatal("published assumption rolled back", refs)
			}
			solverMust(t, n.Close())
			refs, e = m.GetRefCount(tag)
			solverMust(t, e)
			if refs != 1 {
				t.Fatal("assumption member not retired", refs)
			}
		})
	}
}
func TestNamedAssertionNullTagBorrowsInput(t *testing.T) {
	m := solverTestManager(t)
	formula := namedOwnerConst(t, m, "borrowed-na-formula")
	core := &namedOwnerCore{}
	n, e := NewNamedAssertions(m, core)
	solverMust(t, e)
	t.Cleanup(func() { solverMust(t, n.Close()) })
	var got uint32
	core.assert = func(h ast.ASTHandle) error {
		if h != formula {
			t.Fatal("null-tag argument changed")
		}
		var e error
		got, e = m.GetRefCount(h)
		return e
	}
	m.Budget().FailAfter(0)
	err := n.AssertTracked(formula, ast.ASTHandle{})
	m.Budget().FailAfter(-1)
	solverMust(t, err)
	if got != 1 || len(n.assumptions) != 0 || len(n.pending) != 0 {
		t.Fatal("null-tag branch acquired source locals", got)
	}
}

type namedOwnerStopWriter struct {
	action func()
	called bool
}

func (w *namedOwnerStopWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		w.action()
	}
	return len(p), nil
}
func TestNamedAssertionStoppedLocalPrecedesMemberRetirement(t *testing.T) {
	base := solverTestManager(t)
	writer := &namedOwnerStopWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: base.Symbols, Numerals: base.NumeralService(), Budget: base.Budget(), TraceTags: tags})
	solverMust(t, e)
	t.Cleanup(func() { solverMust(t, m.Close()) })
	solverMust(t, m.RegisterDeclarationPlugins())
	n, e := NewNamedAssertions(m, &namedCoreFixture{})
	solverMust(t, e)
	t.Cleanup(func() { solverMust(t, n.Close()) })
	formula := namedOwnerConst(t, m, "stopped-na-formula")
	tag := namedOwnerConst(t, m, "stopped-na-tag")
	_, e = n.appendTemporary([]ast.ASTHandle{tag})
	solverMust(t, e)
	h, e := m.MkImplies(tag, formula)
	solverMust(t, e)
	ref, e := m.NewRef(h)
	solverMust(t, e)
	local := &namedAssertionScope{manager: m, implies: ref}
	var lexical, closeErr error
	sort, e := m.FloatingPointSort(3, 4)
	solverMust(t, e)
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		solverMust(t, local.close())
		t.Skip("requires source trace build")
	}
	writer.action = func() { n.finishAssertion(local, &lexical); closeErr = n.Close() }
	_, e = m.MkFPAZeroOfSort(sort, false)
	solverMust(t, e)
	if !writer.called || !errors.Is(lexical, ast.ErrASTPending) || !errors.Is(closeErr, ast.ErrASTPending) {
		t.Fatal("fixture did not reach actual numerical mutation guard", writer.called, lexical, closeErr)
	}
	if len(n.pending) != 1 || n.pending[0] != local || local.implies != ref || len(n.assumptions) != 1 || n.assumptions[0] == (ast.ASTRef{}) || n.closed {
		t.Fatal("stopped local entered member destruction")
	}
	if _, err := n.NumAssumptions(); !errors.Is(err, ast.ErrASTPending) {
		t.Fatal("stopped owner still executable", err)
	}
	refs, e := m.GetRefCount(tag)
	solverMust(t, e)
	if refs != 3 {
		t.Fatal("failed cleanup consumed an unrelated member", refs)
	}
	memberAlive := false
	writer.called = false
	writer.action = func() { memberAlive = len(n.assumptions) == 1 && n.assumptions[0] != (ast.ASTRef{}) }
	solverMust(t, n.Close())
	if !writer.called || !memberAlive {
		t.Fatal("member assumptions retired before the stopped local")
	}
	solverMust(t, n.Close())
	if len(n.pending) != 0 || local.implies != (ast.ASTRef{}) {
		t.Fatal("retry kept local owner")
	}
	refs, e = m.GetRefCount(tag)
	solverMust(t, e)
	if refs != 1 {
		t.Fatal("retry replayed or retained pin", refs)
	}
}

func TestNamedAssertionInterruptedRangeRetirementRequiresClose(t *testing.T) {
	for _, mode := range []string{"check", "pop"} {
		t.Run(mode, func(t *testing.T) {
			base := solverTestManager(t)
			writer := &namedOwnerStopWriter{}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: base.Symbols, Numerals: base.NumeralService(), Budget: base.Budget(), TraceTags: tags})
			solverMust(t, e)
			t.Cleanup(func() { solverMust(t, m.Close()) })
			solverMust(t, m.RegisterDeclarationPlugins())
			core := &namedCoreFixture{}
			n, e := NewNamedAssertions(m, core)
			solverMust(t, e)
			t.Cleanup(func() { solverMust(t, n.Close()) })
			tag := namedOwnerConst(t, m, "interrupted-na-tag-"+mode)
			if mode == "pop" {
				solverMust(t, n.Push())
				solverMust(t, n.AssertTracked(tag, tag))
			}
			sort, e := m.FloatingPointSort(3, 4)
			solverMust(t, e)
			tags.Enable("ast")
			if !tags.ShouldTrace("ast") {
				t.Skip("requires source trace build")
			}
			var operationErr error
			writer.action = func() {
				if mode == "check" {
					_, operationErr = n.CheckSat([]ast.ASTHandle{tag})
				} else {
					operationErr = n.Pop(1)
				}
			}
			_, e = m.MkFPAZeroOfSort(sort, false)
			solverMust(t, e)
			if !writer.called || !errors.Is(operationErr, ast.ErrASTPending) || !n.closing {
				t.Fatal("retirement interruption did not close operation gate", writer.called, operationErr, n.closing)
			}
			if _, e = n.NumAssumptions(); !errors.Is(e, ast.ErrASTPending) {
				t.Fatal("partially retired adapter remained callable", e)
			}
			if mode == "pop" && core.pops != 1 {
				t.Fatal("core pop was not completed exactly once", core.pops)
			}
			solverMust(t, n.Close())
			if mode == "pop" && core.pops != 1 {
				t.Fatal("cleanup replayed core pop", core.pops)
			}
			refs, e := m.GetRefCount(tag)
			solverMust(t, e)
			if refs != 1 {
				t.Fatal("cleanup retained or replayed assumption", refs)
			}
		})
	}
}
