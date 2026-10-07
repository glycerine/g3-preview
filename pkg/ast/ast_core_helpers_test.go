package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func astTestMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func astTestOwners(t *testing.T) (*util.SymbolManager, *util.RationalService, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	t.Cleanup(func() {
		astTestMust(t, q.Close())
		astTestMust(t, s.Close())
		if b.Used() != 0 {
			t.Errorf("owner graph retained %d bytes", b.Used())
		}
	})
	return s, q, b
}
func astTestManager(t *testing.T) *ASTManager {
	t.Helper()
	s, q, b := astTestOwners(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	return m
}
func astTestSymbol(t *testing.T, m *ASTManager, text string) util.Symbol {
	t.Helper()
	s, e := m.Symbols.Intern(text)
	astTestMust(t, e)
	return s
}
func astTestPin(t *testing.T, m *ASTManager, h ASTHandle) ASTRef {
	t.Helper()
	r, e := m.NewRef(h)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&r)) })
	return r
}
