package pdr

// Source: scaler and closure::operator() in src/muz/pdr/pdr_closure.cpp.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestConvexGeneralizerRestoresPropSolverModes(t *testing.T) {
	_, owner, transformer, _, closeBase := closureFixture(t)
	closureMust(t, transformer.solver.SetUseFarkas(false))
	closureMust(t, transformer.solver.SetSubsetBasedCore(true))
	want := errors.New("convex check failure")
	_, err := withFarkasCore(transformer, func() (PropCheckResult, error) {
		farkas, checkErr := transformer.solver.UseFarkas()
		if checkErr != nil {
			return PropCheckResult{}, checkErr
		}
		if !farkas || transformer.solver.subsetCore {
			t.Fatal("convex check did not force Farkas mode", farkas, transformer.solver.subsetCore)
		}
		return PropCheckResult{}, want
	})
	if !errors.Is(err, want) {
		t.Fatal("convex check error", err)
	}
	farkas, err := transformer.solver.UseFarkas()
	closureMust(t, err)
	if farkas || !transformer.solver.subsetCore {
		t.Fatal("convex check did not restore solver modes", farkas, transformer.solver.subsetCore)
	}
	closureMust(t, transformer.Close())
	closureMust(t, owner.Close())
	closeBase()
}

func closureMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func closureFixture(t *testing.T) (*ast.ASTManager, *Manager, *PredicateTransformer, ast.ASTHandle, func()) {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	closureMust(t, err)
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	closureMust(t, err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	closureMust(t, err)
	closureMust(t, manager.RegisterDeclarationPlugins())
	parameters, err := uparams.NewParamsContext(symbols, numerals, uparams.ParamsConfig{Budget: budget})
	closureMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	closureMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, parameters, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	closureMust(t, err)
	closureMust(t, globals.InitializeSource())
	closeBase := func() {
		closureMust(t, manager.Close())
		closureMust(t, globals.Close())
		closureMust(t, catalog.Close())
		closureMust(t, parameters.Close())
		closureMust(t, numerals.Close())
		closureMust(t, symbols.Close())
		if used := budget.Used(); used != 0 {
			t.Fatalf("closure fixture retained %d bytes", used)
		}
	}
	owner, err := NewManager(manager, nil, 4, smtcore.SMTSolverConfig{Params: parameters, Globals: globals})
	closureMust(t, err)
	arithmetic, err := manager.RegisterArithmeticPlugin()
	closureMust(t, err)
	realSort, err := manager.MkFamilySort(arithmetic, 0, nil)
	closureMust(t, err)
	boolean, err := manager.MkBoolSort()
	closureMust(t, err)
	name, err := manager.Symbols.Intern("closure_predicate")
	closureMust(t, err)
	head, err := manager.MkFuncDecl(name, []ast.ASTHandle{realSort}, boolean, ast.ASTFuncDeclInfo{})
	closureMust(t, err)
	transformer, err := NewPredicateTransformer(owner, head, ReachableNoCache)
	closureMust(t, err)
	signature, err := transformer.Signature()
	closureMust(t, err)
	if len(signature) != 1 {
		t.Fatal("closure signature size", len(signature))
	}
	newDeclaration, err := owner.OldToNew(signature[0], 0)
	closureMust(t, err)
	x, err := manager.MkApp(newDeclaration, nil)
	closureMust(t, err)
	return manager, owner, transformer, x, closeBase
}

func closureOperation(t *testing.T, manager *ast.ASTManager, expression ast.ASTHandle, want int32) ast.AppArguments {
	t.Helper()
	arithmetic, err := manager.ArithmeticFamilyID()
	closureMust(t, err)
	family, err := manager.GetFamilyID(expression)
	closureMust(t, err)
	operator, err := manager.GetDeclKind(expression)
	closureMust(t, err)
	if family != arithmetic || operator != want {
		t.Fatalf("arithmetic operation got family=%d op=%d want family=%d op=%d", family, operator, arithmetic, want)
	}
	arguments, err := manager.AppArgumentView(expression)
	closureMust(t, err)
	return arguments
}

func TestClosureScalerTranslatesVariablesAndScalesOnlyFreeNumerals(t *testing.T) {
	manager, owner, transformer, x, closeBase := closureFixture(t)
	realSort, err := manager.GetSort(x)
	closureMust(t, err)
	y, err := manager.MkFreshConst("closure_y", realSort)
	closureMust(t, err)
	k, err := manager.MkFreshConst("closure_k", realSort)
	closureMust(t, err)
	two, err := manager.ArithmeticRealNumeralText("2")
	closureMust(t, err)
	product, err := manager.MkArithmeticMul(two, x)
	closureMust(t, err)
	expression, err := manager.MkArithmeticAdd(product, two)
	closureMust(t, err)
	xDeclaration, err := manager.GetDecl(x)
	closureMust(t, err)
	scaled, err := scaleClosureExpression(manager, expression, k, map[ast.ASTHandle]ast.ASTHandle{xDeclaration: y})
	closureMust(t, err)
	scaledHandle, err := manager.RefGet(scaled)
	closureMust(t, err)
	addition := closureOperation(t, manager, scaledHandle, closureArithAdd)
	if addition.Count() != 2 {
		t.Fatal("scaled addition arity", addition.Count())
	}
	left, err := addition.At(0)
	closureMust(t, err)
	right, err := addition.At(1)
	closureMust(t, err)
	leftProduct := closureOperation(t, manager, left, closureArithMul)
	left0, err := leftProduct.At(0)
	closureMust(t, err)
	left1, err := leftProduct.At(1)
	closureMust(t, err)
	if left0 != two || left1 != y {
		t.Fatal("multiplication boundary or declaration translation", left0, left1, two, y)
	}
	rightProduct := closureOperation(t, manager, right, closureArithMul)
	right0, err := rightProduct.At(0)
	closureMust(t, err)
	right1, err := rightProduct.At(1)
	closureMust(t, err)
	if right0 != k || right1 != two {
		t.Fatal("free numeral scaling", right0, right1, k, two)
	}
	closureMust(t, manager.DelRef(&scaled))
	closureMust(t, transformer.Close())
	closureMust(t, owner.Close())
	closeBase()
}

func closureStateSAT(t *testing.T, owner *Manager, manager *ast.ASTManager, state, x ast.ASTHandle, value string) bool {
	t.Helper()
	context, err := owner.contexts.Fresh()
	closureMust(t, err)
	defer func() { closureMust(t, context.Close()) }()
	closureMust(t, context.Assert(state))
	numeral, err := manager.ArithmeticRealNumeralText(value)
	closureMust(t, err)
	equality, err := manager.MkEq(x, numeral)
	closureMust(t, err)
	closureMust(t, context.Assert(equality))
	status, err := context.Check(nil)
	closureMust(t, err)
	if status == lbool.Undef {
		t.Fatal("closure satisfiability was unknown", value)
	}
	return status == lbool.True
}

func TestConvexClosureAndInteriorOwnExpectedEndpointRegions(t *testing.T) {
	manager, owner, transformer, x, closeBase := closureFixture(t)
	zero, err := manager.ArithmeticRealNumeralText("0")
	closureMust(t, err)
	two, err := manager.ArithmeticRealNumeralText("2")
	closureMust(t, err)
	left, err := manager.MkEq(x, zero)
	closureMust(t, err)
	right, err := manager.MkEq(x, two)
	closureMust(t, err)

	closedBuilder, err := newConvexClosure(transformer, true)
	closureMust(t, err)
	closedState, err := closedBuilder.Apply([]ast.ASTHandle{left, right})
	closureMust(t, err)
	closureMust(t, closedBuilder.Close())
	closedHandle, err := manager.RefGet(closedState)
	closureMust(t, err)
	for _, row := range []struct {
		value string
		want  bool
	}{{"-1", false}, {"0", true}, {"1", true}, {"2", true}, {"3", false}} {
		if got := closureStateSAT(t, owner, manager, closedHandle, x, row.value); got != row.want {
			t.Fatalf("closed convex hull at %s: got %v want %v", row.value, got, row.want)
		}
	}

	interiorBuilder, err := newConvexClosure(transformer, false)
	closureMust(t, err)
	interiorState, err := interiorBuilder.Apply([]ast.ASTHandle{left, right})
	closureMust(t, err)
	closureMust(t, interiorBuilder.Close())
	interiorHandle, err := manager.RefGet(interiorState)
	closureMust(t, err)
	for _, row := range []struct {
		value string
		want  bool
	}{{"0", false}, {"1", true}, {"2", false}} {
		if got := closureStateSAT(t, owner, manager, interiorHandle, x, row.value); got != row.want {
			t.Fatalf("convex interior at %s: got %v want %v", row.value, got, row.want)
		}
	}

	closureMust(t, manager.DelRef(&interiorState))
	closureMust(t, manager.DelRef(&closedState))
	closureMust(t, transformer.Close())
	closureMust(t, owner.Close())
	closeBase()
}

func TestConvexClosureRelaxesStrictAndUnsupportedAtoms(t *testing.T) {
	manager, owner, transformer, x, closeBase := closureFixture(t)
	two, err := manager.ArithmeticRealNumeralText("2")
	closureMust(t, err)
	arithmetic, err := manager.ArithmeticFamilyID()
	closureMust(t, err)
	strict, err := manager.MkFamilyApp(arithmetic, closureArithLT, nil, []ast.ASTHandle{x, two})
	closureMust(t, err)
	builder, err := newConvexClosure(transformer, true)
	closureMust(t, err)
	relaxed, err := builder.closeFormula(strict)
	closureMust(t, err)
	closureOperation(t, manager, relaxed, closureArithLE)
	boolean, err := manager.MkBoolSort()
	closureMust(t, err)
	unsupported, err := manager.MkFreshConst("closure_bool", boolean)
	closureMust(t, err)
	weakened, err := builder.closeFormula(unsupported)
	closureMust(t, err)
	if !manager.IsTrue(weakened) {
		t.Fatal("unsupported closure atom was not weakened to true")
	}
	closureMust(t, builder.Close())
	closureMust(t, transformer.Close())
	closureMust(t, owner.Close())
	closeBase()
}
