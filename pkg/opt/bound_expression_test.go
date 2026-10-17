package opt

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestOptBoundExpressionSourceBranchesAndOwnedResult(t *testing.T) {
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
	arith, err := m.RegisterArithmeticPlugin()
	must(err)
	realSort, err := m.MkFamilySort(arith, 0, nil)
	must(err)
	intSort, err := m.MkFamilySort(arith, 1, nil)
	must(err)
	makeRational := func(text string) numeral.OwnedRational {
		v, err := s.NewValue()
		must(err)
		must(s.SetString(&v, text))
		return v
	}
	makeBound := func(infinity, rational, epsilon string) numeral.OwnedInfEpsInfRational {
		inf, r, eps := makeRational(infinity), makeRational(rational), makeRational(epsilon)
		nested, err := s.InfFromPair(&r, &eps)
		must(err)
		bound, err := s.InfEpsInfFromPair(&inf, &nested)
		must(err)
		for _, v := range []*numeral.OwnedRational{&inf, &r, &eps} {
			must(s.Del(v))
		}
		must(s.InfDel(&nested))
		return bound
	}
	nameOf := func(h ast.ASTHandle) string {
		decl, err := m.GetDecl(h)
		must(err)
		name, err := m.GetASTName(decl)
		must(err)
		text, err := symbols.String(name)
		must(err)
		return text
	}
	for _, tc := range []struct {
		name, inf, r, eps string
		sort              ast.ASTHandle
		kind              int32
		count             uint32
	}{
		{"zero", "0", "0", "0", intSort, 0, 0},
		{"integer", "0", "4", "0", intSort, 0, 0},
		{"infinity-one", "1", "0", "0", intSort, -1, 0},
		{"fractional-infinity", "1/2", "0", "0", intSort, 9, 2},
		{"epsilon-one", "0", "0", "1", realSort, -1, 0},
		{"real-promotion", "0", "4", "1", realSort, 6, 2},
		{"all-components", "2", "3/2", "-1/3", realSort, 6, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			bound := makeBound(tc.inf, tc.r, tc.eps)
			ref, err := BoundExpression(m, &bound)
			if tc.name == "fractional-infinity" {
				if err == nil || !strings.Contains(err.Error(), "invalid rational value passed as an integer") {
					t.Fatal("source fractional infinity on Int-sort oo must refuse", err)
				}
				must(s.InfEpsInfDel(&bound))
				return
			}
			must(err)
			must(s.InfEpsInfDel(&bound))
			h, err := m.RefGet(ref)
			must(err)
			sort, err := m.GetSort(h)
			must(err)
			if sort != tc.sort {
				t.Fatal("source Int/Real sort", tc.name, sort, tc.sort)
			}
			kind, err := m.GetDeclKind(h)
			must(err)
			if tc.kind >= 0 && kind != tc.kind {
				t.Fatal("source expression operation", tc.name, kind, tc.kind)
			}
			switch tc.name {
			case "zero", "integer":
				text, integer, matched, err := m.ArithmeticNumeralText(h)
				must(err)
				want := "0"
				if tc.name == "integer" {
					want = "4"
				}
				if !matched || !integer || text != want {
					t.Fatal("source singleton numeral", text, integer, matched, want)
				}
			case "infinity-one":
				if name := nameOf(h); name != "oo" {
					t.Fatal("source infinity constant", name)
				}
			case "epsilon-one":
				if name := nameOf(h); name != "epsilon" {
					t.Fatal("source epsilon constant", name)
				}
			case "all-components":
				args, err := m.AppArgumentView(h)
				must(err)
				if args.Count() != tc.count {
					t.Fatal("source add argument count", args.Count())
				}
				for _, at := range []uint32{0, 2} {
					arg, err := args.At(at)
					must(err)
					kind, err := m.GetDeclKind(arg)
					must(err)
					if kind != 9 {
						t.Fatal("source coefficient product order", at, kind)
					}
				}
				middle, err := args.At(1)
				must(err)
				text, integer, matched, err := m.ArithmeticNumeralText(middle)
				must(err)
				if !matched || integer || text != "3/2" {
					t.Fatal("source rational term", text, integer, matched)
				}
			}
			must(m.DelRef(&ref))
		})
	}
	must(m.Close())
	must(s.Close())
	must(symbols.Close())
	if budget.Used() != 0 {
		t.Fatal("retained owner graph", budget.Used())
	}
}
