package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	astfpa "github.com/glycerine/g3/pkg/ast/fpa"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type fpa2BVFixture struct {
	m         *ast.ASTManager
	converter *astfpa.Converter
	rewriter  *FPA2BVRewriter
	family    ast.FamilyID
	sort      ast.ASTHandle
	out       ast.ASTRef
}

func newFPA2BVFixture(t *testing.T) fpa2BVFixture {
	t.Helper()
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	sort, err := m.FloatingPointSort(8, 24)
	astTestMust(t, err)
	converter, err := astfpa.NewConverter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, converter.Close()) })
	rw, err := NewFPA2BVRewriter(m, converter)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, rw.Close()) })
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&out)) })
	return fpa2BVFixture{m: m, converter: converter, rewriter: rw, family: family, sort: sort, out: out}
}

func (f fpa2BVFixture) rewrite(t *testing.T, value ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	astTestMust(t, f.rewriter.Rewrite(value, f.out))
	result, err := f.m.RefGet(f.out)
	astTestMust(t, err)
	return result
}

func TestFPA2BVRewriterSpecialsComparisonsAndSMTEquality(t *testing.T) {
	f := newFPA2BVFixture(t)
	positive, err := f.m.MkFPAZeroOfSort(f.sort, false)
	astTestMust(t, err)
	negative, err := f.m.MkFPAZeroOfSort(f.sort, true)
	astTestMust(t, err)

	converted := f.rewrite(t, positive)
	if kind, e := f.m.GetDeclKind(converted); e != nil || kind != ast.FPAOpFP {
		t.Fatal("special value was not lowered to FP components", kind, e)
	}

	fpEquality, err := f.m.MkFamilyApp(f.family, ast.FPAOpEq, nil, []ast.ASTHandle{positive, negative})
	astTestMust(t, err)
	if result := f.rewrite(t, fpEquality); !f.m.IsTrue(result) {
		t.Fatal("IEEE fp.eq must identify signed zeros", result)
	}

	smtEquality, err := f.m.MkEq(positive, negative)
	astTestMust(t, err)
	if result := f.rewrite(t, smtEquality); !f.m.IsFalse(result) {
		t.Fatal("SMT equality must distinguish signed zeros", result)
	}

	toIEEE, err := f.m.MkFamilyApp(f.family, ast.FPAOpToIEEEBV, nil, []ast.ASTHandle{negative})
	astTestMust(t, err)
	bits := f.rewrite(t, toIEEE)
	bitsSort, err := f.m.GetSort(bits)
	astTestMust(t, err)
	if width, e := f.m.BitVectorSize(bitsSort); e != nil || width != 32 {
		t.Fatal("IEEE conversion width", width, e)
	}
}

func TestFPA2BVRewriterQuantifiedVariablesAndUninterpretedFunctions(t *testing.T) {
	f := newFPA2BVFixture(t)
	variable, err := f.m.MkVar(0, f.sort)
	astTestMust(t, err)
	isZero, err := f.m.MkFamilyApp(f.family, int32(ast.FPAIsZero), nil, []ast.ASTHandle{variable})
	astTestMust(t, err)
	name := astTestSymbol(t, f.m, "x")
	quantifier, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true,
		Sorts:  []ast.ASTHandle{f.sort},
		Names:  []usymbol.Symbol{name},
		Body:   isZero,
	})
	astTestMust(t, err)
	rewritten := f.rewrite(t, quantifier)
	boundSort, err := f.m.GetDeclarationSort(rewritten, 0)
	astTestMust(t, err)
	if width, e := f.m.BitVectorSize(boundSort); e != nil || width != 32 {
		t.Fatal("quantified FP variable was not packed", width, e)
	}
	boundName, err := f.m.GetDeclarationName(rewritten, 0)
	astTestMust(t, err)
	text, err := f.m.Symbols.Display(boundName)
	astTestMust(t, err)
	if text != "x.bv" {
		t.Fatal("converted binder name", text)
	}

	functionName := astTestSymbol(t, f.m, "uf")
	declaration, err := f.m.MkFuncDecl(functionName, []ast.ASTHandle{f.sort}, f.sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	zero, err := f.m.MkFPAZeroOfSort(f.sort, false)
	astTestMust(t, err)
	application, err := f.m.MkApp(declaration, []ast.ASTHandle{zero})
	astTestMust(t, err)
	convertedUF := f.rewrite(t, application)
	if kind, e := f.m.GetDeclKind(convertedUF); e != nil || kind != ast.FPAOpFP {
		t.Fatal("FPA-valued UF was not lowered", kind, e)
	}
	if len(f.converter.ExtraAssertions()) != 1 {
		t.Fatal("UF connection assertion", len(f.converter.ExtraAssertions()))
	}
}

func TestFPA2BVRewriterArithmeticDispatch(t *testing.T) {
	f := newFPA2BVFixture(t)
	rounding, err := f.m.MkFamilyApp(f.family, ast.FPAOpRMNearestEven, nil, nil)
	astTestMust(t, err)
	positive, err := f.m.MkFPAInfinityOfSort(f.sort, false)
	astTestMust(t, err)
	negative, err := f.m.MkFPAInfinityOfSort(f.sort, true)
	astTestMust(t, err)
	addition, err := f.m.MkFamilyApp(f.family, ast.FPAOpAdd, nil, []ast.ASTHandle{rounding, positive, negative})
	astTestMust(t, err)
	got := f.rewrite(t, addition)
	want, err := f.converter.MkSpecial(f.sort, astfpa.NaN)
	astTestMust(t, err)
	if got != want {
		t.Fatal("opposite infinities did not lower to NaN", got, want)
	}
	zero, err := f.m.MkFPAZeroOfSort(f.sort, false)
	astTestMust(t, err)
	multiplication, err := f.m.MkFamilyApp(f.family, ast.FPAOpMul, nil, []ast.ASTHandle{rounding, zero, positive})
	astTestMust(t, err)
	if got = f.rewrite(t, multiplication); got != want {
		t.Fatal("zero times infinity did not lower to NaN", got, want)
	}
	division, err := f.m.MkFamilyApp(f.family, ast.FPAOpDiv, nil, []ast.ASTHandle{rounding, zero, zero})
	astTestMust(t, err)
	if got = f.rewrite(t, division); got != want {
		t.Fatal("zero divided by zero did not lower to NaN", got, want)
	}
	remainder, err := f.m.MkFamilyApp(f.family, ast.FPAOpRem, nil, []ast.ASTHandle{positive, negative})
	astTestMust(t, err)
	if got = f.rewrite(t, remainder); got != want {
		t.Fatal("infinity remainder did not lower to NaN", got, want)
	}
	negativeZero, err := f.m.MkFPAZeroOfSort(f.sort, true)
	astTestMust(t, err)
	integral, err := f.m.MkFamilyApp(f.family, ast.FPAOpRoundToIntegral, nil, []ast.ASTHandle{rounding, negativeZero})
	astTestMust(t, err)
	wantNegativeZero, err := f.converter.MkSpecial(f.sort, astfpa.NegativeZero)
	astTestMust(t, err)
	if got = f.rewrite(t, integral); got != wantNegativeZero {
		t.Fatal("round-to-integral did not preserve negative zero", got, wantNegativeZero)
	}
	squareRoot, err := f.m.MkFamilyApp(f.family, ast.FPAOpSqrt, nil, []ast.ASTHandle{rounding, negative})
	astTestMust(t, err)
	got = f.rewrite(t, squareRoot)
	isNaN, err := f.converter.Classify(got, ast.FPAIsNaN)
	astTestMust(t, err)
	if !f.m.IsTrue(isNaN) {
		t.Fatal("negative-infinity square root did not lower to NaN", got, isNaN)
	}
	fused, err := f.m.MkFamilyApp(f.family, ast.FPAOpFMA, nil, []ast.ASTHandle{rounding, positive, positive, negative})
	astTestMust(t, err)
	got = f.rewrite(t, fused)
	isNaN, err = f.converter.Classify(got, ast.FPAIsNaN)
	astTestMust(t, err)
	if !f.m.IsTrue(isNaN) {
		t.Fatal("opposed infinite FMA did not lower to NaN", got, isNaN)
	}

	left := astTestSymbol(t, f.m, "add-x")
	right := astTestSymbol(t, f.m, "add-y")
	leftValue, err := f.m.MkConst(left, f.sort)
	astTestMust(t, err)
	rightValue, err := f.m.MkConst(right, f.sort)
	astTestMust(t, err)
	addition, err = f.m.MkFamilyApp(f.family, ast.FPAOpAdd, nil, []ast.ASTHandle{rounding, leftValue, rightValue})
	astTestMust(t, err)
	got = f.rewrite(t, addition)
	if kind, e := f.m.GetDeclKind(got); e != nil || kind != ast.FPAOpFP {
		t.Fatal("symbolic addition did not lower to FP components", kind, e)
	}
}

func TestFPA2BVRewriterToFPFormatDispatch(t *testing.T) {
	f := newFPA2BVFixture(t)
	targetSort, err := f.m.FloatingPointSort(5, 11)
	astTestMust(t, err)
	rounding, err := f.m.MkFamilyApp(f.family, ast.FPAOpRMNearestEven, nil, nil)
	astTestMust(t, err)
	ebits, err := f.m.NewParameterInt(5)
	astTestMust(t, err)
	sbits, err := f.m.NewParameterInt(11)
	astTestMust(t, err)
	t.Cleanup(func() {
		astTestMust(t, f.m.DelParameter(&sbits))
		astTestMust(t, f.m.DelParameter(&ebits))
	})
	rmSort, err := f.m.GetSort(rounding)
	astTestMust(t, err)
	declaration, err := f.m.MkFamilyFuncDecl(
		f.family, ast.FPAOpToFP,
		[]ast.ASTParameter{ebits, sbits},
		[]ast.ASTHandle{rmSort, f.sort}, ast.ASTHandle{},
	)
	astTestMust(t, err)
	positiveInfinity, err := f.m.MkFPAInfinityOfSort(f.sort, false)
	astTestMust(t, err)
	conversion, err := f.m.MkApp(declaration, []ast.ASTHandle{rounding, positiveInfinity})
	astTestMust(t, err)
	got := f.rewrite(t, conversion)
	want, err := f.converter.MkSpecial(targetSort, astfpa.PositiveInfinity)
	astTestMust(t, err)
	if got != want {
		t.Fatal("format conversion dispatch", got, want)
	}
	bvSort, err := f.m.BitVectorSort(32)
	astTestMust(t, err)
	zero, err := f.m.BitVectorNumeralUnsigned(0, 32)
	astTestMust(t, err)
	for _, kind := range []int32{ast.FPAOpToFP, ast.FPAOpToFPUnsigned} {
		integerDeclaration, e := f.m.MkFamilyFuncDecl(
			f.family, kind,
			[]ast.ASTParameter{ebits, sbits},
			[]ast.ASTHandle{rmSort, bvSort}, ast.ASTHandle{},
		)
		astTestMust(t, e)
		integerConversion, e := f.m.MkApp(integerDeclaration, []ast.ASTHandle{rounding, zero})
		astTestMust(t, e)
		got = f.rewrite(t, integerConversion)
		want, e = f.converter.MkSpecial(targetSort, astfpa.PositiveZero)
		astTestMust(t, e)
		if got != want {
			t.Fatal("integer conversion dispatch", kind, got, want)
		}
	}
}
