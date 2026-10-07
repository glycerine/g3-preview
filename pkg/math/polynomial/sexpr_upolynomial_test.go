package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func sxPolyOwner(t *testing.T) (*SexprPolynomialConverter, *util.SexprManager, *util.SymbolManager, *util.RationalService, *UPolynomialManager, UPolynomial) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	sx, sy, q := sxOwner(t, b)
	limit, e := util.NewReslimit(b)
	sxMust(t, e)
	coop, e := util.NewCooperation(b)
	sxMust(t, e)
	p, e := NewUPolynomialManagerFromRationalService(q, UPolynomialConfig{Checkpoint: limit.Inc, Cooperation: func() error { return coop.Cooperate("upolynomial") }})
	sxMust(t, e)
	out, e := p.NewPolynomial()
	sxMust(t, e)
	c, e := NewSexprPolynomialConverter(sx, p)
	sxMust(t, e)
	t.Cleanup(func() {
		sxMust(t, c.Close())
		sxMust(t, p.DelPolynomial(&out))
		sxMust(t, p.Close())
		sxMust(t, coop.Close())
		limit.Close()
	})
	return c, sx, sy, q, p, out
}
func sxPolySym(t *testing.T, sx *util.SexprManager, sy *util.SymbolManager, name string) util.SexprHandle {
	t.Helper()
	h, e := sx.MkSymbol(internSymbol(t, sy, name), 13, 17)
	sxMust(t, e)
	return h
}
func sxPolyNum(t *testing.T, sx *util.SexprManager, q *util.RationalService, text string) util.SexprHandle {
	t.Helper()
	v := sxNumber(t, q, text)
	h, e := sx.MkNumeral(&v, 23, 29)
	sxMust(t, e)
	sxMust(t, q.Del(&v))
	return h
}
func sxPolyApp(t *testing.T, sx *util.SexprManager, sy *util.SymbolManager, op string, args ...util.SexprHandle) util.SexprHandle {
	t.Helper()
	children := append([]util.SexprHandle{sxPolySym(t, sx, sy, op)}, args...)
	h, e := sx.MkComposite(children, 31, 37)
	sxMust(t, e)
	return h
}
func sxPolyCoefficients(t *testing.T, m *UPolynomialManager, p UPolynomial, want []int32) {
	t.Helper()
	n, e := m.Size(p)
	sxMust(t, e)
	if n != uint32(len(want)) {
		t.Fatal(n, want)
	}
	for i, w := range want {
		var a util.Int
		sxMust(t, m.ReadCoefficient(p, uint32(i), &a))
		got, e := m.IntegerManager().Int32(&a)
		sxMust(t, e)
		sxMust(t, m.IntegerManager().Del(&a))
		if got != w {
			t.Fatal(i, got, w)
		}
	}
}
func TestSexprPolynomialSourceArithmetic(t *testing.T) {
	c, sx, sy, q, p, out := sxPolyOwner(t)
	x := sxPolySym(t, sx, sy, "x")
	one := sxPolyNum(t, sx, q, "1")
	two := sxPolyNum(t, sx, q, "2")
	// Independent expansion (x-1)*(x-2)^2 = x^3-5x^2+8x-4.
	expr := sxPolyApp(t, sx, sy, "*", sxPolyApp(t, sx, sy, "-", x, one), sxPolyApp(t, sx, sy, "^", sxPolyApp(t, sx, sy, "-", x, two), two))
	sxMust(t, c.Convert(expr, out))
	sxPolyCoefficients(t, p, out, []int32{-4, 8, -5, 1})
	expr = sxPolyApp(t, sx, sy, "-", sxPolyApp(t, sx, sy, "+", one, x, two))
	sxMust(t, c.Convert(expr, out))
	sxPolyCoefficients(t, p, out, []int32{-3, -1})
	expr = sxPolyApp(t, sx, sy, "-", x, one, two)
	sxMust(t, c.Convert(expr, out))
	sxPolyCoefficients(t, p, out, []int32{-3, 1})
	for _, op := range []string{"+", "-", "*"} {
		sxMust(t, c.Convert(sxPolyApp(t, sx, sy, op, two), out))
		want := int32(2)
		if op == "-" {
			want = -2
		}
		sxPolyCoefficients(t, p, out, []int32{want})
	}
}
func TestSexprPolynomialErrorsAndCompletedPrefix(t *testing.T) {
	c, sx, sy, q, p, out := sxPolyOwner(t)
	x := sxPolySym(t, sx, sy, "x")
	cases := []struct {
		h         util.SexprHandle
		message   string
		line, pos int32
	}{
		{sxPolyApp(t, sx, sy, "+"), "invalid univariate polynomial, '+' operator expects at least one argument", 31, 37},
		{sxPolyApp(t, sx, sy, "-"), "invalid univariate polynomial, '-' operator expects at least one argument", 31, 37},
		{sxPolyApp(t, sx, sy, "*"), "invalid univariate polynomial, '*' operator expects at least one argument", 31, 37},
		{sxPolyApp(t, sx, sy, "^", x), "invalid univariate polynomial, '^' operator expects two arguments", 31, 37},
		{sxPolyApp(t, sx, sy, "/", x), "invalid univariate polynomial, '+', '-', '^' or '*' expected", 31, 37},
		{sxPolyNum(t, sx, q, "1/2"), "invalid univariate polynomial, integer coefficient expected", 23, 29},
		{sxPolySym(t, sx, sy, "y"), "invalid univariate polynomial, variable 'x' expected", 13, 17},
		{sxPolyApp(t, sx, sy, "^", x, sxPolyNum(t, sx, q, "-1")), "invalid univariate polynomial, exponent must be an unsigned integer", 23, 29},
		{sxPolyApp(t, sx, sy, "^", x, sxPolyNum(t, sx, q, "4294967296")), "invalid univariate polynomial, exponent must be an unsigned integer", 23, 29},
	}
	for _, a := range cases {
		sxMust(t, p.SetInts(out, []util.Int{util.NewInt(41)}))
		err := c.Convert(a.h, out)
		var e *SexprPolynomialError
		if !errors.As(err, &e) || e.Message != a.message || e.Line != a.line || e.Pos != a.pos {
			t.Fatal(err, a)
		}
	}
	// First argument publishes before invalid exponent or later invalid argument.
	sxPolyCoefficients(t, p, out, []int32{0, 1})
	sxMust(t, p.SetInts(out, []util.Int{util.NewInt(41)}))
	e := c.Convert(sxPolyApp(t, sx, sy, "+", x, sxPolySym(t, sx, sy, "y")), out)
	if e == nil {
		t.Fatal("accepted y")
	}
	sxPolyCoefficients(t, p, out, []int32{0, 1})
}
func TestSexprPolynomialDepthBoundary(t *testing.T) {
	c, sx, sy, _, p, out := sxPolyOwner(t)
	x := sxPolySym(t, sx, sy, "x")
	sxMust(t, c.convertAtDepth(x, out, 1<<16))
	sxPolyCoefficients(t, p, out, []int32{0, 1})
	e := c.convertAtDepth(x, out, (1<<16)+1)
	var pe *SexprPolynomialError
	if !errors.As(e, &pe) || pe.Message != "invalid univariate polynomial, too complex" || pe.Line != 13 || pe.Pos != 17 {
		t.Fatal(e)
	}
	sxPolyCoefficients(t, p, out, []int32{0, 1})
}

func TestSexprPolynomialActualFaultPrefixes(t *testing.T) {
	failures, success := 0, 0
	for fault := int64(0); fault < 256; fault++ {
		done := false
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			c, sx, sy, q, p, out := sxPolyOwner(t)
			x := sxPolySym(t, sx, sy, "x")
			big := sxPolyNum(t, sx, q, "123456789012345678901234567890123456789")
			expr := sxPolyApp(t, sx, sy, "+", big, sxPolyApp(t, sx, sy, "*", x, sxPolyApp(t, sx, sy, "+", x, big)))
			b := sx.Config.Budget
			b.FailAfter(fault)
			e := c.Convert(expr, out)
			// Explicit source owners name node constants, service scopes/private locals,
			// caller output and all converter scoped vectors BEFORE fault disabling.
			upolyHenselAuditPhysical(t, []*UPolynomialManager{p}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, SexprManagers: []*util.SexprManager{sx}})
			if e != nil {
				if !upolyHenselAllocationRefusal(p.IntegerManager(), e) {
					t.Fatal(e)
				}
				failures++
			} else {
				success++
				done = true
			}
			b.FailAfter(-1)
			sxMust(t, c.Cleanup())
			if e == nil {
				if n, e := p.Size(out); e != nil || n != 3 {
					t.Fatal(n, e)
				}
			}
		})
		if done {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual refusal prefixes and success", failures, success)
}
func TestSexprPolynomialWarmAndRemainingKinds(t *testing.T) {
	c, sx, sy, q, _, out := sxPolyOwner(t)
	x := sxPolySym(t, sx, sy, "x")
	two := sxPolyNum(t, sx, q, "2")
	expr := sxPolyApp(t, sx, sy, "+", x, two)
	sxMust(t, c.Convert(expr, out))
	before := sx.Config.Budget.Used()
	allocations := testing.AllocsPerRun(20, func() {
		if e := c.Convert(expr, out); e != nil {
			panic(e)
		}
	})
	// Source scoped arg backing is allocated and destroyed once per call.
	if allocations != 1 || sx.Config.Budget.Used() != before {
		t.Fatal("warm concrete backing", allocations, before, sx.Config.Budget.Used())
	}
	atomAllocations := testing.AllocsPerRun(20, func() {
		if e := c.Convert(x, out); e != nil {
			panic(e)
		}
	})
	if atomAllocations != 0 {
		t.Fatal("warm atom allocation", atomAllocations)
	}
	empty, e := sx.MkComposite(nil, 41, 43)
	sxMust(t, e)
	nonhead, e := sx.MkComposite([]util.SexprHandle{two}, 47, 53)
	sxMust(t, e)
	str, e := sx.MkString("x", 59, 61)
	sxMust(t, e)
	for _, a := range []struct {
		h   util.SexprHandle
		msg string
	}{{empty, "invalid univariate polynomial, symbol expected"}, {nonhead, "invalid univariate polynomial, symbol expected"}, {str, "invalid univariate polynomial, unexpected "}, {sxPolyApp(t, sx, sy, "^", x, str), "invalid univariate polynomial, exponent must be an unsigned integer"}} {
		err := c.Convert(a.h, out)
		var pe *SexprPolynomialError
		if !errors.As(err, &pe) || pe.Message != a.msg {
			t.Fatal(err)
		}
	}
}
