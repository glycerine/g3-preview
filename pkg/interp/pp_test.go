package interp

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func interpASTFixture(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := symbol.NewManager(symbol.Config{Budget: b})
	interpMust(t, err)
	n, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	interpMust(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b})
	interpMust(t, err)
	t.Cleanup(func() {
		interpMust(t, m.Close())
		interpMust(t, n.Close())
		interpMust(t, s.Close())
		if b.Used() != 0 {
			t.Errorf("interpolation AST fixture retained %d bytes", b.Used())
		}
	})
	return m
}

func interpPin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	r, err := m.NewRef(h)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, m.DelRef(&r)) })
	return h
}

func TestInterpolationPrettyPrintProblemExactTreeAndNames(t *testing.T) {
	m := interpASTFixture(t)
	boolSort, err := m.MkBoolSort()
	interpMust(t, err)
	pName, err := m.Symbols.Intern("p")
	interpMust(t, err)
	qName, err := m.Symbols.Intern("q")
	interpMust(t, err)
	p, err := m.MkConst(pName, boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := m.MkConst(qName, boolSort)
	interpMust(t, err)
	q = interpPin(t, m, q)
	marked, err := m.MkInterp(q)
	interpMust(t, err)
	marked = interpPin(t, m, marked)
	tree, err := m.MkAnd(p, marked)
	interpMust(t, err)
	tree = interpPin(t, m, tree)
	var out bytes.Buffer
	interpMust(t, PrettyPrintProblem(m, []ast.ASTHandle{p, q}, tree, &out))
	want := "(set-option :produce-interpolants true)\n" +
		"(declare-fun p () Bool)\n(declare-fun q () Bool)\n" +
		"(assert (! p :named f!1))\n(assert (! q :named f!2))\n" +
		"(check-sat)\n(get-interpolant (and f!1 (interp f!2)))\n"
	if out.String() != want {
		t.Fatalf("pretty interpolation problem:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestInterpolationPrettyPrintDuplicateAndBadTree(t *testing.T) {
	m := interpASTFixture(t)
	p, err := m.MkTrue()
	interpMust(t, err)
	p = interpPin(t, m, p)
	var out bytes.Buffer
	interpMust(t, PrettyPrintProblem(m, []ast.ASTHandle{p, p}, p, &out))
	if !strings.HasSuffix(out.String(), "(get-interpolant f!2)\n") {
		t.Fatal(out.String())
	}
	bad, err := m.MkNot(p)
	interpMust(t, err)
	bad = interpPin(t, m, bad)
	out.Reset()
	err = PrettyPrintProblem(m, []ast.ASTHandle{p}, bad, &out)
	var classified BadTreeError
	if !errors.As(err, &classified) || classified.Error() != "iz3pp_bad_tree" {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out.String(), "(get-interpolant ") {
		t.Fatal("source prefix before bad tree was not preserved", out.String())
	}
}
