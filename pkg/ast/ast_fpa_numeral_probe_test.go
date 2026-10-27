package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestASTFloatingPointClassificationDestructorFailureClearsResult(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	f, e := p.beginNumeralProbe()
	astTestMust(t, e)
	z, e := p.numeric.BorrowIntegerManager()
	astTestMust(t, e)
	var sig unumeral.Int
	astTestMust(t, z.SetLimbs(&sig, []uint32{1, 2, 3, 4}))
	astTestMust(t, p.numeric.SetComponents(&f.value, 11, 5000, false, 3, &sig))
	astTestMust(t, z.Del(&sig))
	borrowed, e := p.numeric.BorrowSignificand(&f.value)
	astTestMust(t, e)
	if borrowed.Cell == 0 {
		t.Fatal("fixture did not retain physical significand storage")
	}
	cell := borrowed.Cell
	borrowed.Cell = ^unumeral.NumericCellRef(0)
	matched := true // the classification body completed before source destruction
	var result error
	p.finishNumeralProbe(f, &result, &matched)
	if matched || !errors.Is(result, arena.ErrInvalidSlot) || p.numeralProbeFrames != f || f.retired {
		t.Fatal("failed scoped_mpf destructor published a classification", matched, result, f)
	}
	borrowed.Cell = cell
	astTestMust(t, m.Close())
}

func TestASTFloatingPointConstructorDestructorFailureClearsResult(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	f, e := p.beginNumeralProbe()
	astTestMust(t, e)
	z, e := p.numeric.BorrowIntegerManager()
	astTestMust(t, e)
	var sig unumeral.Int
	astTestMust(t, z.SetLimbs(&sig, []uint32{1, 2, 3, 4}))
	astTestMust(t, p.numeric.SetComponents(&f.value, 11, 5000, false, 3, &sig))
	astTestMust(t, z.Del(&sig))
	out, e := p.mkNumeral(&f.value, nil)
	astTestMust(t, e)
	if out.slot == 0 {
		t.Fatal("constructor body did not publish a numeral")
	}
	borrowed, e := p.numeric.BorrowSignificand(&f.value)
	astTestMust(t, e)
	if borrowed.Cell == 0 {
		t.Fatal("fixture did not retain physical significand storage")
	}
	cell := borrowed.Cell
	borrowed.Cell = ^unumeral.NumericCellRef(0)
	var result error
	p.finishNumeralConstruction(f, &result, &out)
	if out.slot != 0 || !errors.Is(result, arena.ErrInvalidSlot) || p.numeralProbeFrames != f || f.retired {
		t.Fatal("failed scoped_mpf destructor published a constructor result", out, result, f)
	}
	borrowed.Cell = cell
	astTestMust(t, m.Close())
}

func TestASTFloatingPointBooleanNumeralScopedValue(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, sigValue := range []int32{0, 1, 19} {
		sig := unumeral.NewInt(sigValue)
		var value unumeral.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&value, 8, 24, false, 3, &sig))
		n, e := p.mkNumeral(&value, nil)
		astTestMust(t, e)
		before := m.BudgetField.Used()
		found, e := p.isNumeralOnly(n)
		astTestMust(t, e)
		if !found || p.numeralProbeFrames != nil || m.BudgetField.Used() != before {
			t.Fatal("source boolean numeral and scoped value", found, p.numeralProbeFrames, m.BudgetField.Used(), before)
		}
		astTestMust(t, p.numeric.Del(&value))
	}
	h, e := m.MkTrue()
	astTestMust(t, e)
	n, e := m.internal(h)
	astTestMust(t, e)
	before := m.BudgetField.Used()
	found, e := p.isNumeralOnly(n)
	astTestMust(t, e)
	if found || p.numeralProbeFrames != nil || m.BudgetField.Used() != before {
		t.Fatal("unmatched numeral scoped value", found, p.numeralProbeFrames, m.BudgetField.Used(), before)
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointBooleanNumeralRetainsFailedScalarCopy(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	var sig unumeral.Int
	z, e := p.numeric.BorrowScopedInteger(&sig)
	astTestMust(t, e)
	digits := make([]uint32, 128)
	for i := range digits {
		digits[i] = 0xffffffff
	}
	digits[len(digits)-1] = 0x7fffffff
	astTestMust(t, z.SetLimbs(&sig, digits))
	var source unumeral.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&source, 11, 5000, false, 3, &sig))
	n, e := p.mkNumeral(&source, nil)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	found, body := p.isNumeralOnly(n)
	if body == nil || found || p.numeralProbeFrames == nil || p.numeralProbeFrames.retired {
		t.Fatal("failed source scalar copy did not retain exact owner", found, body, p.numeralProbeFrames)
	}
	m.BudgetField.FailAfter(-1)
	older := p.numeralProbeFrames
	h, e := m.MkTrue()
	astTestMust(t, e)
	other, e := m.internal(h)
	astTestMust(t, e)
	found, e = p.isNumeralOnly(other)
	astTestMust(t, e)
	if found || p.numeralProbeFrames != older || older.retired || older.header != 0 {
		t.Fatal("later query displaced stopped scalar owner", found, p.numeralProbeFrames, older)
	}
	astTestMust(t, p.numeric.Del(&source))
	astTestMust(t, z.Del(&sig))
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, m.Close())
}

func TestASTFloatingPointScopedClassificationSourcePredicates(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	classes := []astFloatClass{
		astFloatAny, astFloatNaN, astFloatInf, astFloatPInf, astFloatNInf,
		astFloatZero, astFloatPZero, astFloatNZero, astFloatNormal,
		astFloatSubnormal, astFloatPositive, astFloatNegative,
	}
	for _, tc := range []struct {
		name     string
		sig      int32
		exponent int64
		sign     bool
		trueFor  []astFloatClass
	}{
		{"nan", 1, 128, false, []astFloatClass{astFloatAny, astFloatNaN}},
		{"positive infinity", 0, 128, false, []astFloatClass{astFloatAny, astFloatInf, astFloatPInf, astFloatPositive}},
		{"negative infinity", 0, 128, true, []astFloatClass{astFloatAny, astFloatInf, astFloatNInf, astFloatNegative}},
		{"positive zero", 0, -127, false, []astFloatClass{astFloatAny, astFloatZero, astFloatPZero, astFloatPositive}},
		{"negative zero", 0, -127, true, []astFloatClass{astFloatAny, astFloatZero, astFloatNZero, astFloatNegative}},
		{"normal", 19, 3, false, []astFloatClass{astFloatAny, astFloatNormal, astFloatPositive}},
		{"negative normal", 19, 3, true, []astFloatClass{astFloatAny, astFloatNormal, astFloatNegative}},
		{"subnormal", 1, -127, false, []astFloatClass{astFloatAny, astFloatSubnormal, astFloatPositive}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sig := unumeral.NewInt(tc.sig)
			var value unumeral.ArbitraryFloat
			astTestMust(t, p.numeric.SetComponents(&value, 8, 24, tc.sign, tc.exponent, &sig))
			n, e := p.mkNumeral(&value, nil)
			astTestMust(t, e)
			for _, class := range classes {
				got, e := p.isNumeralClass(n, class)
				astTestMust(t, e)
				want := false
				for _, expected := range tc.trueFor {
					want = want || class == expected
				}
				if got != want || p.numeralProbeFrames != nil {
					t.Fatal("source class and scope", class, got, want, p.numeralProbeFrames)
				}
			}
			astTestMust(t, p.numeric.Del(&value))
		})
	}
	h, e := m.MkTrue()
	astTestMust(t, e)
	n, e := m.internal(h)
	astTestMust(t, e)
	for _, class := range classes {
		got, e := p.isNumeralClass(n, class)
		astTestMust(t, e)
		if got {
			t.Fatal("unmatched expression classified", class)
		}
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointSpecialConstructorsScopedValue(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, tc := range []struct {
		class astFloatClass
		kind  int32
	}{
		{astFloatNaN, astFPANaN}, {astFloatPInf, astFPAPlusInf},
		{astFloatNInf, astFPAMinusInf}, {astFloatPZero, astFPAPlusZero},
		{astFloatNZero, astFPAMinusZero},
	} {
		n, e := p.mkSpecialNumeral(8, 24, tc.class)
		astTestMust(t, e)
		decl := m.appAt(n.slot).decl
		family, kind, e := m.declIdentity(decl)
		astTestMust(t, e)
		if family != p.family || kind != tc.kind || p.numeralProbeFrames != nil {
			t.Fatal("source special constructor", tc, family, kind, p.numeralProbeFrames)
		}
		matched, e := p.isNumeralClass(n, tc.class)
		astTestMust(t, e)
		if !matched {
			t.Fatal("constructed special value did not classify", tc)
		}
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointSpecialNaNFailureRetainsScalar(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	m.BudgetField.FailAfter(0)
	n, body := p.mkSpecialNumeral(11, 5000, astFloatNaN)
	if body == nil || n.slot != 0 || p.numeralProbeFrames == nil || p.numeralProbeFrames.retired {
		t.Fatal("stopped NaN constructor lost scalar scope", n, body, p.numeralProbeFrames)
	}
	m.BudgetField.FailAfter(-1)
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, m.Close())
}

func TestASTFloatingPointSignedZeroSortOverloads(t *testing.T) {
	_, _, d, _, _ := astDebugActionFixture(t, "T")
	previous, err := diagnostic.BindProcessDebugContext(d)
	astTestMust(t, err)
	t.Cleanup(func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) })
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	sort, e := m.mkFloatSort(8, 24)
	astTestMust(t, e)
	for _, negative := range []bool{false, true} {
		n, e := p.mkSignedZeroFromSort(sort, negative)
		astTestMust(t, e)
		kind := astFPAPlusZero
		class := astFloatPZero
		if negative {
			kind = astFPAMinusZero
			class = astFloatNZero
		}
		family, got, e := m.declIdentity(m.appAt(n.slot).decl)
		astTestMust(t, e)
		if family != p.family || got != kind || p.numeralProbeFrames != nil {
			t.Fatal("sort overload result and scope", negative, family, got, p.numeralProbeFrames)
		}
		matched, e := p.isNumeralClass(n, class)
		astTestMust(t, e)
		if !matched {
			t.Fatal("sort overload lost sign", negative)
		}
	}
	n, e := p.mkSignedZeroFromSort(m.boolSort, false)
	if diagnostic.DebugBuild {
		var ex z3exception.DefaultException
		if !errors.As(e, &ex) || ex.Message() != "assertion violation" {
			t.Fatal("nonfloating source width assertion", e)
		}
	} else if e != ErrASTDomain {
		t.Fatal("nonfloating source parameter read", e)
	}
	if n.slot != 0 || p.numeralProbeFrames != nil {
		t.Fatal("nonfloating sort entered local MPF constructor", n, p.numeralProbeFrames)
	}

	astTestMust(t, m.Close())
}
