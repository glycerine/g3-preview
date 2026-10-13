package tactic

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestArbitraryExprConverterTermsAndClause(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{Memory: runtime, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	family, err := m.RegisterArithmeticPlugin()
	if err != nil {
		t.Fatal(err)
	}
	sort, err := m.MkFamilySort(family, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, q, params.ParamsConfig{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := sy.Intern("mpf")
	if err != nil {
		t.Fatal(err)
	}
	key, depth := "numeral", "max_depth"
	if err := pc.SetSymbolName(&p, &key, selected); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&p, &depth, 0); err != nil {
		t.Fatal(err)
	}
	qm, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewArbitraryEngine(&control.Reslimit{Limit: 1000}, numbers, pc, &p)
	if err != nil {
		t.Fatal(err)
	}
	converter, err := NewArbitraryExprConverter(m, engine, qm)
	if err != nil {
		t.Fatal(err)
	}
	var n, d numeral.Int
	value, err := q.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{converter.Close(), engine.Close(), qm.Z.Del(&n), qm.Z.Del(&d), qm.CloseChecked(), pc.Del(&p), pc.Close(), q.Del(&value), m.Close(), q.Close(), sy.Close(), numbers.Close(), core.Close(), runtime.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("retained owner bytes", b.Used())
		}
	})
	name, err := sy.Intern("x")
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.MkConst(name, sort)
	if err != nil {
		t.Fatal(err)
	}
	yName, err := sy.Intern("y")
	if err != nil {
		t.Fatal(err)
	}
	y, err := m.MkConst(yName, sort)
	if err != nil {
		t.Fatal(err)
	}
	first, err := converter.InternalizeTerm(x, &n, &d)
	if err != nil || first == math.MaxUint32 {
		t.Fatal("variable", first, err)
	}
	if err := q.SetString(&value, "1"); err != nil {
		t.Fatal(err)
	}
	one, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{x, y, one})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(sum); err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(sum); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.DecRef(sum); err != nil {
			t.Error(err)
		}
		if err := m.DecRef(sum); err != nil {
			t.Error(err)
		}
	}()
	variable, err := converter.InternalizeTerm(sum, &n, &d)
	if err != nil || variable == first || variable == math.MaxUint32 {
		t.Fatal("sum", variable, err)
	}
	if got, err := qm.Z.String(&d); err != nil || got != "1" {
		t.Fatal("sum denominator", got, err)
	}
	if again, err := converter.InternalizeTerm(sum, &n, &d); err != nil || again != variable {
		t.Fatal("cached sum", again, err)
	}
	if refs, err := m.GetRefCount(sum); err != nil || refs != 3 {
		t.Fatal("cached AST ownership", refs, err)
	}
	if err := q.SetString(&value, "2"); err != nil {
		t.Fatal(err)
	}
	two, err := m.ArithmeticNumeral(&value, true)
	if err != nil {
		t.Fatal(err)
	}
	power, err := m.MkFamilyApp(family, 18, nil, []ast.ASTHandle{x, two})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := converter.InternalizeTerm(power, &n, &d); err != nil || v == math.MaxUint32 {
		t.Fatal("power", v, err)
	}
	ineq, err := m.MkFamilyApp(family, 2, nil, []ast.ASTHandle{sum, two})
	if err != nil {
		t.Fatal(err)
	}
	if err := converter.InternalizeClause(ineq); err != nil {
		t.Fatal("clause", err)
	}
	shell, err := engine.Shell()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := shell.DisplayConstraints(&out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "2") {
		t.Fatal("missing bound", out.String())
	}
	zeroConstant, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{x, y})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := converter.InternalizeTerm(zeroConstant, &n, &d); !errors.Is(err, numeral.ErrArbitraryFloatNonRegular) {
		t.Fatal("MPF f2n zero-constant refusal", err)
	}
}
