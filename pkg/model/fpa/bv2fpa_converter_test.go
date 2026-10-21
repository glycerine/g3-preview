package fpa

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	astfpa "github.com/glycerine/g3/pkg/ast/fpa"
	"github.com/glycerine/g3/pkg/model"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func testMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func testModelCore(t *testing.T, m *ast.ASTManager) *model.ModelCore {
	t.Helper()
	core, err := model.NewModelCore(m)
	testMust(t, err)
	t.Cleanup(func() { testMust(t, core.Close()) })
	return core
}

func testFixture(t *testing.T) (*ast.ASTManager, *astfpa.Converter, *Converter) {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	testMust(t, err)
	numerals, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	testMust(t, err)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget})
	testMust(t, err)
	source, err := astfpa.NewConverter(manager)
	testMust(t, err)
	converter, err := NewConverter(manager, source)
	testMust(t, err)
	t.Cleanup(func() {
		testMust(t, converter.Close())
		testMust(t, source.Close())
		testMust(t, manager.Close())
		testMust(t, numerals.Close())
		testMust(t, symbols.Close())
		if budget.Used() != 0 {
			t.Fatalf("retained %d bytes", budget.Used())
		}
	})
	return manager, source, converter
}

func TestBV2FPAConverterNumeralsAndRoundingModes(t *testing.T) {
	m, _, c := testFixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	testMust(t, err)
	sign, err := m.BitVectorNumeralUnsigned(0, 1)
	testMust(t, err)
	exponent, err := m.BitVectorNumeralUnsigned(127, 8)
	testMust(t, err)
	significand, err := m.BitVectorNumeralUnsigned(1<<22, 23)
	testMust(t, err)
	got, err := c.ConvertBV2FP(sort, sign, exponent, significand)
	testMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	testMust(t, err)
	components, err := m.MkFamilyApp(family, ast.FPAOpFP, nil, []ast.ASTHandle{sign, exponent, significand})
	testMust(t, err)
	declaration, err := m.GetDecl(components)
	testMust(t, err)
	want, found, err := m.RewriteFPAFPComponentsNumerals(declaration, sign, exponent, significand)
	testMust(t, err)
	if !found || got != want {
		t.Fatal("component conversion", got, want, found)
	}

	packed, err := m.BitVectorNumeralUnsigned(0x3fc00000, 32)
	testMust(t, err)
	if rebuilt, e := c.ConvertPackedBV(nil, sort, packed); e != nil || rebuilt != want {
		t.Fatal("packed conversion", rebuilt, want, e)
	}

	for code, kind := range []int32{
		ast.FPAOpRMNearestEven, ast.FPAOpRMNearestAway,
		ast.FPAOpRMTowardPositive, ast.FPAOpRMTowardNegative,
		ast.FPAOpRMTowardZero, ast.FPAOpRMTowardZero,
	} {
		bits, e := m.BitVectorNumeralUnsigned(uint32(code), 3)
		testMust(t, e)
		mode, e := c.ConvertBV2RM(bits)
		testMust(t, e)
		if gotKind, e := m.GetDeclKind(mode); e != nil || gotKind != kind {
			t.Fatal("rounding mode conversion", code, gotKind, kind, e)
		}
	}
}

func TestBV2FPAConverterRetainsSourceSnapshot(t *testing.T) {
	m, source, _ := testFixture(t)
	sort, err := m.FloatingPointSort(8, 24)
	testMust(t, err)
	name, err := m.Symbols.Intern("snapshot-float")
	testMust(t, err)
	declaration, err := m.MkFuncDecl(name, nil, sort, ast.ASTFuncDeclInfo{})
	testMust(t, err)
	_, err = source.MkConst(declaration)
	testMust(t, err)
	before, err := m.GetRefCount(declaration)
	testMust(t, err)
	retained, err := NewConverter(m, source)
	testMust(t, err)
	after, err := m.GetRefCount(declaration)
	testMust(t, err)
	if after != before+1 {
		t.Fatal("snapshot declaration reference", before, after)
	}
	testMust(t, source.Reset())
	testMust(t, retained.Close())
}

func TestBV2FPAConverterConvertsFloatAndRoundingConstants(t *testing.T) {
	m, sourceConverter, _ := testFixture(t)
	sourceModel := testModelCore(t, m)
	targetModel := testModelCore(t, m)
	sort, err := m.FloatingPointSort(8, 24)
	testMust(t, err)
	name, err := m.Symbols.Intern("converted-float")
	testMust(t, err)
	floatDecl, err := m.MkFuncDecl(name, nil, sort, ast.ASTFuncDeclInfo{})
	testMust(t, err)
	encoded, err := sourceConverter.MkConst(floatDecl)
	testMust(t, err)
	sign, err := m.GetArg(encoded, 0)
	testMust(t, err)
	packed, err := m.GetArg(sign, 0)
	testMust(t, err)
	packedDecl, err := m.GetDecl(packed)
	testMust(t, err)
	packedValue, err := m.BitVectorNumeralUnsigned(0x3fc00000, 32)
	testMust(t, err)
	testMust(t, sourceModel.RegisterConstant(packedDecl, packedValue))

	family, err := m.RegisterFloatingPointPlugin()
	testMust(t, err)
	rmSort, err := m.MkFamilySort(family, 1, nil)
	testMust(t, err)
	rmName, err := m.Symbols.Intern("converted-rm")
	testMust(t, err)
	rmDecl, err := m.MkFuncDecl(rmName, nil, rmSort, ast.ASTFuncDeclInfo{})
	testMust(t, err)
	wrappedRM, err := sourceConverter.MkRMConst(rmDecl)
	testMust(t, err)
	rmBits, err := m.GetArg(wrappedRM, 0)
	testMust(t, err)
	rmBitsDecl, err := m.GetDecl(rmBits)
	testMust(t, err)
	rmValue, err := m.BitVectorNumeralUnsigned(3, 3)
	testMust(t, err)
	testMust(t, sourceModel.RegisterConstant(rmBitsDecl, rmValue))

	converter, err := NewConverter(m, sourceConverter)
	testMust(t, err)
	defer func() { testMust(t, converter.Close()) }()
	seen, err := converter.ConvertModel(sourceModel, targetModel)
	testMust(t, err)
	if _, ok := seen[packedDecl]; !ok {
		t.Fatal("packed declaration was not consumed")
	}
	if _, ok := seen[rmBitsDecl]; !ok {
		t.Fatal("rounding declaration was not consumed")
	}
	gotFloat, err := targetModel.ConstantInterpretation(floatDecl)
	testMust(t, err)
	wantFloat, err := converter.ConvertPackedBV(nil, sort, packedValue)
	testMust(t, err)
	if gotFloat != wantFloat {
		t.Fatal("converted float constant", gotFloat, wantFloat)
	}
	gotRM, err := targetModel.ConstantInterpretation(rmDecl)
	testMust(t, err)
	if kind, e := m.GetDeclKind(gotRM); e != nil || kind != ast.FPAOpRMTowardNegative {
		t.Fatal("converted rounding constant", kind, e)
	}
}
