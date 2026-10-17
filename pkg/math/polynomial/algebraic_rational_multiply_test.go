package polynomial

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicRationalMultiplyRootsSignsAndAlias(t *testing.T) {
	for _, degree := range []uint32{2, 3} {
		for _, negative := range []bool{false, true} {
			for _, alias := range []bool{false, true} {
				for _, tc := range []struct {
					n int64
					d uint64
				}{{1, 1}, {-1, 1}, {3, 2}, {-3, 2}, {2, 3}, {-2, 3}} {
					t.Run(fmt.Sprint(degree, negative, alias, tc.n, tc.d), func(t *testing.T) {
						c, p := algebraicFixture(t)
						coeff := make([]int32, degree+1)
						coeff[0], coeff[degree] = -2, 1
						if negative && degree%2 != 0 {
							coeff[0] = 2
						}
						poly := algebraicPolynomial(t, p, coeff...)
						v, e := p.wholeView(poly)
						serviceMust(t, e)
						bm, qm := c.config.Binary, c.config.Rationals
						var lower, upper unumeral.BinaryRational
						lo, hi := int32(1), int32(2)
						if negative {
							lo, hi = -2, -1
						}
						serviceMust(t, bm.SetInt32(&lower, lo))
						serviceMust(t, bm.SetInt32(&upper, hi))
						var a, original, result Algebraic
						serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
						serviceMust(t, c.Copy(&original, &a))
						q := unumeral.NewRational()
						serviceMust(t, qm.SetFraction(&q, tc.n, tc.d))
						defer func() {
							serviceMust(t, c.Del(&result))
							serviceMust(t, c.Del(&original))
							serviceMust(t, c.Del(&a))
							serviceMust(t, qm.Del(&q))
							serviceMust(t, bm.Del(&upper))
							serviceMust(t, bm.Del(&lower))
						}()
						out := &result
						if alias {
							out = &a
						}
						serviceMust(t, c.MulRational(&a, &q, out))
						workspace := c.addTmp
						l, e := c.location(out)
						serviceMust(t, e)
						r, e := c.load(l)
						serviceMust(t, e)
						scaled, e := c.sourcePolynomial(&r)
						serviceMust(t, e)
						// If alpha^degree=C and beta=(n/d)*alpha, then
						// d^degree*beta^degree=C*n^degree.
						n, d := int32(1), int32(1)
						for i := uint32(0); i < degree; i++ {
							n *= int32(tc.n)
							d *= int32(tc.d)
						}
						constant := coeff[0] * n
						abs := constant
						if abs < 0 {
							abs = -abs
						}
						g, h := abs, d
						for h != 0 {
							g, h = h, g%h
						}
						for i := uint32(0); i <= degree; i++ {
							want := int32(0)
							if i == 0 {
								want = constant / g
							}
							if i == degree {
								want = d / g
							}
							value, e := p.read(scaled, i)
							serviceMust(t, e)
							if value.NativeScalar() != want {
								t.Fatal("scaled coefficient", i, value.NativeScalar(), want)
							}
						}
						positive, e := c.IsPos(out)
						serviceMust(t, e)
						if positive != (negative == (tc.n < 0)) || r.flags&1 == 0 {
							t.Fatal("scaled interval sign or minimality", positive, r.flags)
						}
						// Addition and multiplication must reuse and overwrite the same
						// persistent source workspace without corrupting either result.
						serviceMust(t, c.AddRational(out, &q, out))
						serviceMust(t, c.SubRational(out, &q, out))
						serviceMust(t, qm.InvInPlace(&q))
						serviceMust(t, c.MulRational(out, &q, out))
						if c.addTmp != workspace {
							t.Fatal("mixed operations replaced the persistent polynomial owner")
						}
						cmp, e := c.Compare(out, &original)
						serviceMust(t, e)
						if cmp != 0 {
							t.Fatal("scale and inverse scale changed the value", cmp)
						}
					})
				}
			}
		}
	}
}

func TestAlgebraicRationalMultiplyBasicAndZeroShortCircuit(t *testing.T) {
	c, _ := algebraicFixture(t)
	qm := c.config.Rationals
	q := unumeral.NewRational()
	var a Algebraic
	serviceMust(t, qm.SetFraction(&q, 3, 2))
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, qm.SetFraction(&q, 2, 3))
	serviceMust(t, c.MulRational(&a, &q, &a))
	serviceMust(t, qm.SetInt32(&q, 1))
	cmp, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("basic rational product")
	}
	serviceMust(t, qm.SetInt32(&q, 0))
	serviceMust(t, c.MulRational(&a, &q, &a))
	if a != (Algebraic{}) {
		t.Fatal("zero product did not reset the cell")
	}
	qm.Budget().FailAfter(0)
	e = c.MulRational(&a, nil, &a)
	qm.Budget().FailAfter(-1)
	serviceMust(t, e)
	serviceMust(t, qm.Del(&q))
}

func TestAlgebraicRationalMultiplySourceAssertionBeforeLocals(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion is erased")
	}
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	bm, qm := c.config.Binary, c.config.Rationals
	var lower, upper unumeral.BinaryRational
	serviceMust(t, bm.SetInt32(&lower, 1))
	serviceMust(t, bm.SetInt32(&upper, 2))
	var a, out Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	q := unumeral.NewRational()
	serviceMust(t, qm.SetInt32(&q, 7))
	serviceMust(t, c.SetRational(&out, &q))
	serviceMust(t, qm.SetInt32(&q, 2))
	defer func() {
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&a))
		serviceMust(t, qm.Del(&q))
		serviceMust(t, bm.Del(&upper))
		serviceMust(t, bm.Del(&lower))
	}()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	serviceMust(t, d.Close())
	body := c.MulRational(&a, &q, &out)
	_, e = diagnostic.BindProcessDebugContext(previous)
	serviceMust(t, e)
	serviceMust(t, region.Close())
	if !errors.Is(body, arena.ErrClosed) || c.arithmetic.active || c.addTmp != (UPolynomial{}) {
		t.Fatal("reached switch did not precede arithmetic locals", body)
	}
	polynomialThrowAssertionFixture(t)
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	serviceMust(t, bm.Copy(r.interval.BorrowUpper(), r.interval.BorrowLower()))
	serviceMust(t, c.store(l, &r))
	body = c.MulRational(&a, &q, &out)
	var exception z3exception.DefaultException
	if !errors.As(body, &exception) || c.arithmetic.active || c.addTmp != (UPolynomial{}) {
		t.Fatal("source T action did not precede arithmetic locals", body)
	}
	serviceMust(t, qm.SetInt32(&q, 7))
	cmp, e := c.CompareRational(&out, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("failed entry assertion changed the output")
	}
}

func TestAlgebraicRationalMultiplyBigScalarAndInverse(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	bm, qm := c.config.Binary, c.config.Rationals
	var lower, upper unumeral.BinaryRational
	serviceMust(t, bm.SetInt32(&lower, 1))
	serviceMust(t, bm.SetInt32(&upper, 2))
	var a, original Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	serviceMust(t, c.Copy(&original, &a))
	q := unumeral.NewRational()
	serviceMust(t, qm.SetString(&q, "18446744073709551617/3"))
	defer func() {
		serviceMust(t, c.Del(&original))
		serviceMust(t, c.Del(&a))
		serviceMust(t, qm.Del(&q))
		serviceMust(t, bm.Del(&upper))
		serviceMust(t, bm.Del(&lower))
	}()
	serviceMust(t, c.MulRational(&a, &q, &a))
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	scaled, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	constant, e := p.read(scaled, 0)
	serviceMust(t, e)
	text, e := qm.Integers().String(constant)
	serviceMust(t, e)
	n := new(big.Int).Lsh(big.NewInt(1), 64)
	n.Add(n, big.NewInt(1))
	want := new(big.Int).Mul(n, n)
	want.Mul(want, big.NewInt(-2))
	if text != want.String() {
		t.Fatal("BIG rational scaling coefficient", text, want)
	}
	serviceMust(t, qm.InvInPlace(&q))
	serviceMust(t, c.MulRational(&a, &q, &a))
	cmp, e := c.Compare(&a, &original)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("BIG inverse scale changed the algebraic value", cmp)
	}
}
