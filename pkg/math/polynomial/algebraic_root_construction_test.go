package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
	usexpr "github.com/glycerine/g3/pkg/util/sexpr"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func algebraicSexprRootFixture(t *testing.T) (*AlgebraicRootCore, *UPolynomialManager, *usexpr.SexprManager, *usymbol.Manager, *unumeral.RationalService, *algebraicTestControl) {
	t.Helper()
	sx, sy, q := sxOwner(t, arena.NewBudget(1<<26))
	control := &algebraicTestControl{limit: 1000000}
	up, e := NewUPolynomialManagerFromRationalService(q, UPolynomialConfig{Checkpoint: control.checkpoint, Cooperation: control.cooperate})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(q.Manager.Budget())
	serviceMust(t, e)
	factors, e := NewUPolynomialFactorizationService(up, UPolynomialFactorizationConfig{Primes: primes})
	serviceMust(t, e)
	binary, e := unumeral.NewBinaryRationalManager(q.Manager.Integers(), unumeral.BinaryRationalConfig{Diagnostics: q})
	serviceMust(t, e)
	intervals, e := unumeral.NewBinaryIntervalManager(binary)
	serviceMust(t, e)
	allocator, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Manager.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	c, e := NewAlgebraicRootCore(AlgebraicRootConfig{Rationals: q.Manager, Polynomials: up, Factors: factors, Binary: binary, Intervals: intervals, Allocator: allocator, Checkpoint: control.checkpoint, Cooperation: control.cooperate})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, c.Close())
		serviceMust(t, allocator.Close())
		intervals.Close()
		serviceMust(t, binary.CloseChecked())
		serviceMust(t, factors.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, up.Close())
		if control.visits != control.cooperations+control.refused {
			t.Fatal("checkpoint/cooperation pairing", control)
		}
	})
	return c, up, sx, sy, q, control
}

func TestAlgebraicPolynomialRootPersistentConversionAndIndices(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{3, [3]uint32{0, 0, 2}}, {-6, [3]uint32{}}})
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	for i := uint32(1); i <= 2; i++ {
		serviceMust(t, c.MkPolynomialRoot(pm, p, i, &out))
		if i == 1 {
			algebraicExpectBetween(t, c, &out, -2, -1)
		} else {
			algebraicExpectBetween(t, c, &out, 1, 2)
		}
		checkAlgebraicResultantPolynomial(t, up, c.isolateTmp1, -6, 0, 3)
		if pm.factorNumeralVector.outputManager != up {
			t.Fatal("conversion used an intermediate numerical owner")
		}
		if pm.algebraicIsolationBorrowers != 0 || c.rootConstruction.active {
			t.Fatal("successful root construction retained caller")
		}
	}
	old := out
	e := c.MkPolynomialRoot(pm, p, 3, &out)
	if e == nil || e.Error() != "invalid root object, polynomial does have sufficient roots" || out != old {
		t.Fatal("out-of-range root", e)
	}
	var specific AlgebraicException
	var base z3exception.DefaultException
	if !errors.As(e, &specific) || !errors.As(e, &base) || specific.HasErrorCode() {
		t.Fatal("source algebraic/default exception class", e)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.MkPolynomialRoot(pm, p, 2, &out))
}
func TestAlgebraicSexprRootArithmeticAndPersistentReuse(t *testing.T) {
	c, up, sx, sy, q, _ := algebraicSexprRootFixture(t)
	x := sxPolySym(t, sx, sy, "x")
	two := sxPolyNum(t, sx, q, "2")
	expr := sxPolyApp(t, sx, sy, "-", sxPolyApp(t, sx, sy, "^", x, two), two)
	var out, square Algebraic
	defer func() { serviceMust(t, c.Del(&square)); serviceMust(t, c.Del(&out)) }()
	serviceMust(t, c.MkSexprRoot(sx, expr, 2, &out))
	algebraicExpectBetween(t, c, &out, 1, 2)
	serviceMust(t, c.Power(&out, 2, &square))
	n := unumeral.NewInt(2)
	var expected Algebraic
	serviceMust(t, c.SetInteger(&expected, &n))
	cmp, e := c.Compare(&square, &expected)
	serviceMust(t, e)
	serviceMust(t, c.Del(&expected))
	if cmp != 0 {
		t.Fatal("sqrt(2) squared", cmp)
	}
	workspace := c.isolateTmp1
	checkAlgebraicResultantPolynomial(t, up, workspace, -2, 0, 1)
	// Repeated factors exercise parser arithmetic and one-based distinct roots.
	one := sxPolyNum(t, sx, q, "1")
	expr = sxPolyApp(t, sx, sy, "*", sxPolyApp(t, sx, sy, "-", x, one), sxPolyApp(t, sx, sy, "^", sxPolyApp(t, sx, sy, "-", x, two), two))
	serviceMust(t, c.MkSexprRoot(sx, expr, 2, &out))
	algebraicExpectBetween(t, c, &out, 1, 3)
	if c.isolateTmp1 != workspace {
		t.Fatal("persistent isolation descriptor was replaced")
	}
	checkAlgebraicResultantPolynomial(t, up, workspace, -4, 8, -5, 1)
	if c.rootConstruction.active || c.rootConstruction.parser != nil {
		t.Fatal("successful expression conversion retained owner")
	}
}
func TestAlgebraicSexprRootParsePrefixAndOutputPreservation(t *testing.T) {
	c, up, sx, sy, q, _ := algebraicSexprRootFixture(t)
	x := sxPolySym(t, sx, sy, "x")
	invalid := sxPolyApp(t, sx, sy, "+", x, sxPolySym(t, sx, sy, "y"))
	n := unumeral.NewInt(17)
	var out Algebraic
	serviceMust(t, c.SetInteger(&out, &n))
	defer func() { serviceMust(t, c.Del(&out)) }()
	before := out
	e := c.MkSexprRoot(sx, invalid, 1, &out)
	var parse *SexprPolynomialError
	if !errors.As(e, &parse) || parse.Message != "invalid univariate polynomial, variable 'x' expected" || parse.Line != 13 || parse.Pos != 17 || out != before {
		t.Fatal("parse error", e, parse, out, before)
	}
	checkAlgebraicResultantPolynomial(t, up, c.isolateTmp1, 0, 1)
	serviceMust(t, c.CleanupRetained())
	zero := sxPolyNum(t, sx, q, "0")
	e = c.MkSexprRoot(sx, zero, 1, &out)
	if e == nil || e.Error() != "invalid root object, polynomial must not be the zero polynomial" || out != before {
		t.Fatal("zero expression", e)
	}
	checkAlgebraicResultantPolynomial(t, up, c.isolateTmp1)
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.MkSexprRoot(sx, x, 1, &out))
	if out != (Algebraic{}) {
		t.Fatal("identity root not null zero")
	}
}

func TestAlgebraicSexprRootBigCoefficientUsesBorrowedNumericService(t *testing.T) {
	c, up, sx, sy, q, _ := algebraicSexprRootFixture(t)
	x := sxPolySym(t, sx, sy, "x")
	expr := sxPolyApp(t, sx, sy, "-", x, sxPolyNum(t, sx, q, "1208925819614629174706177"))
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	serviceMust(t, c.MkSexprRoot(sx, expr, 1, &out))
	n := unumeral.NewInt(0)
	serviceMust(t, q.Manager.Integers().SetLimbs(&n, []uint32{1, 0, 1 << 16}))
	expected := unumeral.NewRational()
	serviceMust(t, q.Manager.SetInteger(&expected, &n))
	cmp, e := c.CompareRational(&out, &expected)
	serviceMust(t, e)
	serviceMust(t, q.Manager.Del(&expected))
	serviceMust(t, q.Manager.Integers().Del(&n))
	if cmp != 0 {
		t.Fatal("BIG linear root", cmp)
	}
	size, e := up.Size(c.isolateTmp1)
	serviceMust(t, e)
	if size != 2 {
		t.Fatal("BIG source degree", size)
	}
	if q.Lexical.Pending() {
		t.Fatal("completed converter retained numeric source locals")
	}
}
