package tactic

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestHardwareExprConverterNumeralsAndVariableLeaves(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	sy, err := symbol.NewManager(symbol.Config{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	family, err := m.RegisterArithmeticPlugin()
	if err != nil {
		t.Fatal(err)
	}
	integerSort, err := m.MkFamilySort(family, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	hwf, err := sy.Intern("hwf")
	if err != nil {
		t.Fatal(err)
	}
	key := "numeral"
	if err := pc.SetSymbolName(&p, &key, hwf); err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewHardwareEngine(&control.Reslimit{Limit: 1000}, qm, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	converter, err := NewHardwareExprConverter(m, engine)
	if err != nil {
		t.Fatal(err)
	}
	var n, d numeral.Int
	value, err := q.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{converter.Close(), engine.Close(), qm.Z.Del(&n), qm.Z.Del(&d), qm.CloseChecked(), pc.Del(&p), pc.Close(), q.Del(&value), m.Close(), q.Close(), sy.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("retained owner bytes", budget.Used())
		}
	})
	for _, tc := range []struct {
		text         string
		integer      bool
		wantN, wantD string
	}{
		{"42", true, "42", "1"},
		{"18446744073709551617/3", false, "18446744073709551617", "3"},
	} {
		if err := q.SetString(&value, tc.text); err != nil {
			t.Fatal(err)
		}
		h, err := m.ArithmeticNumeral(&value, tc.integer)
		if err != nil {
			t.Fatal(err)
		}
		v, err := converter.InternalizeTerm(h, &n, &d)
		if err != nil || v != math.MaxUint32 {
			t.Fatalf("numeral %s: var %d, %v", tc.text, v, err)
		}
		gotN, err := qm.Z.String(&n)
		if err != nil {
			t.Fatal(err)
		}
		gotD, err := qm.Z.String(&d)
		if err != nil {
			t.Fatal(err)
		}
		if gotN != tc.wantN || gotD != tc.wantD {
			t.Fatalf("numeral %s: %s/%s", tc.text, gotN, gotD)
		}
	}
	name, err := sy.Intern("x")
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.MkConst(name, integerSort)
	if err != nil {
		t.Fatal(err)
	}
	first, err := converter.InternalizeTerm(x, &n, &d)
	if err != nil {
		t.Fatal(err)
	}
	second, err := converter.InternalizeTerm(x, &n, &d)
	if err != nil || first != second {
		t.Fatalf("variable reuse: %d %d %v", first, second, err)
	}
	gotN, err := qm.Z.String(&n)
	if err != nil {
		t.Fatal(err)
	}
	gotD, err := qm.Z.String(&d)
	if err != nil {
		t.Fatal(err)
	}
	if gotN != "1" || gotD != "1" {
		t.Fatalf("variable coefficient: %s/%s", gotN, gotD)
	}
	isVar, err := converter.IsVar(x)
	if err != nil || !isVar {
		t.Fatalf("mapped variable: %v %v", isVar, err)
	}
	add, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{x, x})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := converter.InternalizeTerm(add, &n, &d)
	if err != nil || sum == math.MaxUint32 || sum == first {
		t.Fatalf("x+x must define a polynomial variable: %d, %v", sum, err)
	}
	gotN, err = qm.Z.String(&n)
	if err != nil {
		t.Fatal(err)
	}
	gotD, err = qm.Z.String(&d)
	if err != nil {
		t.Fatal(err)
	}
	if gotN != "1" || gotD != "1" {
		t.Fatalf("sum coefficient: %s/%s", gotN, gotD)
	}
	if _, err := converter.InternalizeTerm(add, &n, &d); err != nil {
		t.Fatal(err)
	}
	if err := q.SetString(&value, "1/2"); err != nil {
		t.Fatal(err)
	}
	half, err := m.ArithmeticNumeral(&value, false)
	if err != nil {
		t.Fatal(err)
	}
	constantAdd, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{half, half})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := converter.InternalizeTerm(constantAdd, &n, &d); err != nil || v != math.MaxUint32 {
		t.Fatalf("half plus half: %d, %v", v, err)
	}
	gotN, err = qm.Z.String(&n)
	if err != nil {
		t.Fatal(err)
	}
	gotD, err = qm.Z.String(&d)
	if err != nil {
		t.Fatal(err)
	}
	if gotN != "1" || gotD != "1" {
		t.Fatalf("constant sum: %s/%s", gotN, gotD)
	}
	realSort, err := m.MkFamilySort(family, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	realName, err := sy.Intern("y")
	if err != nil {
		t.Fatal(err)
	}
	y, err := m.MkConst(realName, realSort)
	if err != nil {
		t.Fatal(err)
	}
	mixedAdd, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{y, half})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := converter.InternalizeTerm(mixedAdd, &n, &d); err != nil || v == math.MaxUint32 {
		t.Fatalf("y plus half: %d, %v", v, err)
	}
	gotN, err = qm.Z.String(&n)
	if err != nil {
		t.Fatal(err)
	}
	gotD, err = qm.Z.String(&d)
	if err != nil {
		t.Fatal(err)
	}
	if gotN != "1" || gotD != "2" {
		t.Fatalf("mixed sum: %s/%s", gotN, gotD)
	}
	mul, err := m.MkFamilyApp(family, 9, nil, []ast.ASTHandle{x, x})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := converter.InternalizeTerm(mul, &n, &d); !errors.Is(err, ErrExprConverterUnsupported) {
		t.Fatalf("unported multiplication: %v", err)
	}
	if err := converter.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := converter.InternalizeTerm(x, &n, &d); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed converter: %v", err)
	}
}
