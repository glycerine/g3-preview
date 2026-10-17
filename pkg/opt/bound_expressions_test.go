package opt

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestOptBoundExpressionsRetainSourceTripleAfterBoundRelease(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	budget := arena.NewBudget(1 << 27)
	symbols, err := symbol.NewManager(symbol.Config{Budget: budget})
	must(err)
	s, err := numeral.NewRationalService(numeral.RationalServiceConfig{
		Integers: numeral.IntegerConfig{
			Cells: arena.Config{Budget: budget},
			Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
		},
	})
	must(err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: s, Budget: budget})
	must(err)
	_, err = m.RegisterArithmeticPlugin()
	must(err)
	makeRational := func(text string) numeral.OwnedRational {
		v, err := s.NewValue()
		must(err)
		must(s.SetString(&v, text))
		return v
	}
	inf, rational, eps := makeRational("2"), makeRational("3/2"), makeRational("-1/3")
	nested, err := s.InfFromPair(&rational, &eps)
	must(err)
	bound, err := s.InfEpsInfFromPair(&inf, &nested)
	must(err)
	for _, v := range []*numeral.OwnedRational{&inf, &rational, &eps} {
		must(s.Del(v))
	}
	must(s.InfDel(&nested))
	prior := makeRational("7")
	priorNode, err := m.ArithmeticNumeral(&prior, true)
	must(err)
	priorRef, err := m.NewRef(priorNode)
	must(err)
	must(s.Del(&prior))
	refs := []ast.ASTRef{priorRef}
	must(AppendBoundExpressions(m, &bound, &refs))
	if len(refs) != 4 || refs[0] != priorRef {
		t.Fatal("source triple length", len(refs))
	}
	must(s.InfEpsInfDel(&bound))
	for i, want := range []struct {
		text    string
		integer bool
	}{{"2", true}, {"3/2", false}, {"-1/3", false}} {
		h, err := m.RefGet(refs[i+1])
		must(err)
		got, integer, matched, err := m.ArithmeticNumeralText(h)
		must(err)
		if !matched || got != want.text || integer != want.integer {
			t.Fatal("source triple order and sort", i, got, integer, matched, want)
		}
	}
	for i := range refs {
		must(m.DelRef(&refs[i]))
	}
	must(m.Close())
	must(s.Close())
	must(symbols.Close())
	if budget.Used() != 0 {
		t.Fatal("retained owner graph", budget.Used())
	}
}
