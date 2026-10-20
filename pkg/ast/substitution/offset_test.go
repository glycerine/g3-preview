package substitution

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	uhash "github.com/glycerine/g3/pkg/util/hash"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func offsetManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	syms, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	nums, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Budget: b, Symbols: syms, Numerals: nums})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
		if err := nums.Close(); err != nil {
			t.Error(err)
		}
		if err := syms.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Fatal("substitution manager retained bytes", b.Used())
		}
	})
	return m
}

func TestExprOffsetIdentityHashAndConcreteMaps(t *testing.T) {
	m := offsetManager(t)
	s, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.MkVar(3, s)
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.NewRef(x)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.DelRef(&r); err != nil {
			t.Error(err)
		}
	}()
	a, b := NewExprOffset(x, 2), NewExprOffset(x, 3)
	if a == b || a.Expr() != x || a.Offset() != 2 {
		t.Fatal("expression-offset identity")
	}
	id, err := m.GetASTID(x)
	if err != nil {
		t.Fatal(err)
	}
	h, err := a.Hash(m)
	if err != nil || h != uhash.Mix(id, 2, 17) {
		t.Fatal("source hash", h, err)
	}
	if _, err = (ExprOffset{}).Hash(m); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("null hash guard", err)
	}

	var vars varOffsetExprMap
	if err = vars.reserve(4, 8); err != nil {
		t.Fatal(err)
	}
	if err = vars.insert(3, 1, a); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := vars.find(3, 1); err != nil || !ok || got != a {
		t.Fatal("variable-offset lookup", got, ok, err)
	}
	vars.reset()
	if _, ok, err := vars.find(3, 1); err != nil || ok {
		t.Fatal("variable-offset reset", ok, err)
	}
	if err = vars.insert(8, 1, a); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("variable capacity guard", err)
	}

	exprs := make(exprOffsetExprMap)
	offsets := make(exprOffsetOffsetMap)
	uints := make(exprOffsetUintMap)
	exprs.insert(a, x)
	offsets.insert(a, b)
	uints.insert(a, 7)
	if got, ok := exprs.find(a); !ok || got != x || !exprs.contains(a) {
		t.Fatal("expression map lookup", got, ok)
	}
	if got, ok := offsets.find(a); !ok || got != b || !offsets.contains(a) {
		t.Fatal("offset map lookup", got, ok)
	}
	if got, ok := uints.find(a); !ok || got != 7 || !uints.contains(a) {
		t.Fatal("unsigned map lookup", got, ok)
	}
	exprs.reset()
	offsets.reset()
	uints.reset()
	if len(exprs)+len(offsets)+len(uints) != 0 {
		t.Fatal("expression-offset reset")
	}
}
