package interp

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestFullTranslationArithmeticHelpers(t *testing.T) {
	_, manager := interpRewritingManagerFixture(t)
	integer, err := manager.IntType()
	interpMust(t, err)
	x, err := manager.MakeVariable("translation-x", integer)
	interpMust(t, err)
	y, err := manager.MakeVariable("translation-y", integer)
	interpMust(t, err)
	base, err := NewBase(manager, []ast.ASTHandle{x, y}, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, base.Close()) })
	translation, err := NewFullTranslation(base)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, translation.Close()) })
	three, err := manager.MakeIntText("3")
	interpMust(t, err)
	right, err := manager.Make(OperatorPlus, three, y)
	interpMust(t, err)
	inequality, err := manager.Make(OperatorLE, x, right)
	interpMust(t, err)
	normalized, err := translation.NormalizeInequality(inequality)
	interpMust(t, err)
	normalizedRight, err := manager.Arg(normalized, 1)
	interpMust(t, err)
	value, numeral, err := manager.Numeral(normalizedRight)
	if err != nil || !numeral || value.Cmp(new(big.Rat).SetInt64(3)) != 0 {
		t.Fatal("normalized affine right side", value, numeral, err)
	}
	valid, err := translation.CheckFarkas([]ast.ASTHandle{inequality}, inequality)
	if err != nil || !valid {
		t.Fatal("Farkas identity", valid, err)
	}
}

func interpRewritingManagerFixture(t *testing.T) (*ast.ASTManager, *Manager) {
	t.Helper()
	m := interpASTFixture(t)
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	interpMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	interpMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	interpMust(t, err)
	interpMust(t, globals.InitializeSource())
	t.Cleanup(func() {
		interpMust(t, globals.Close())
		interpMust(t, catalog.Close())
		interpMust(t, context.Close())
	})
	w, err := NewManagerWithParams(m, context, globals)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, w.Close()) })
	return m, w
}

func TestInterpolationManagerTheorySimplifiers(t *testing.T) {
	m, w := interpRewritingManagerFixture(t)
	intSort, err := w.IntType()
	interpMust(t, err)
	x, err := w.MakeVariable("rewrite-x", intSort)
	interpMust(t, err)
	x = interpPin(t, m, x)
	zero, err := w.MakeIntText("0")
	interpMust(t, err)
	sum, err := w.Make(OperatorPlus, x, zero)
	interpMust(t, err)
	simplified, err := w.Simplify(sum)
	interpMust(t, err)
	defer func() { interpMust(t, m.DelRef(&simplified)) }()
	if got, e := m.RefGet(simplified); e != nil || got != x {
		t.Fatal("default simplifier", got, e)
	}

	one, err := w.MakeIntText("1")
	interpMust(t, err)
	two, err := w.MakeIntText("2")
	interpMust(t, err)
	left, err := w.Make(OperatorPlus, x, one)
	interpMust(t, err)
	right, err := w.Make(OperatorPlus, two, x)
	interpMust(t, err)
	polynomial, err := w.Make(OperatorPlus, left, right)
	interpMust(t, err)
	strongRef, err := w.ReallySimplify(polynomial)
	interpMust(t, err)
	defer func() { interpMust(t, m.DelRef(&strongRef)) }()
	strong, err := m.RefGet(strongRef)
	interpMust(t, err)
	if op, e := w.Operator(strong); e != nil || op != OperatorPlus {
		t.Fatal("strong simplifier", op, e)
	}
}

func TestInterpolationManagerSimplifierConfigurationGuard(t *testing.T) {
	_, w := interpManagerFixture(t)
	truth, err := w.Make(OperatorTrue)
	interpMust(t, err)
	if _, err = w.Simplify(truth); err == nil || err.Error() != "interpolation manager has no theory rewriter" {
		t.Fatal(err)
	}
}
