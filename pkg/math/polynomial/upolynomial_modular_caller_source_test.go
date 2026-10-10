// Source: src/math/polynomial/upolynomial.cpp:1108-1225,1734-1820,1906-1951,2029-2074;
// src/math/polynomial/algebraic_numbers.cpp:1565-1577;
// src/util/mpzzp.h:97-105. Actual returning-transport owner guard fixture.
package polynomial

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestUPolynomialActualModularCallerCloseBeforeReturn(t *testing.T) {
	for _, mode := range []string{"knuth", "upper", "integer", "binary", "rational", "negation", "display"} {
		t.Run(mode, func(t *testing.T) {
			var p *UPolynomialManager
			var foreign *util.IntegerModularManager
			armed, reached := false, false
			transport := func(f umemory.Failure) error {
				if !armed {
					return f
				}
				armed = false
				reached = true
				closed, header, used := p.modular.Closed(), p.modular.Header, p.config.Integer.Budget().Used()
				physical := p.config.Integer.RuntimeOwner().Stats()
				e := p.modular.Close()
				if e != ErrPolynomialPending || p.modular.Closed() != closed || p.modular.Header != header || p.config.Integer.Budget().Used() != used || p.config.Integer.RuntimeOwner().Stats() != physical {
					t.Errorf("%s exact modular owner retired during actual caller: %v closed %v/%v header %d/%d", mode, e, closed, p.modular.Closed(), header, p.modular.Header)
				}
				foreignHeader := foreign.Header
				if e := foreign.Close(); e != nil || !foreign.Closed() || p.config.Integer.Budget().Used()+foreignHeader != used || p.config.Integer.RuntimeOwner().Stats() != physical {
					t.Error("unrelated exact modular receiver was blocked or changed physical graph", e)
				}
				return f
			}
			c, actual, rt := algebraicNegTLSFixture(t, transport)
			p = actual
			z := c.config.Rationals.Integers()
			foreign, e := util.NewIntegerModularManager(z)
			serviceMust(t, e)
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			values := []util.Int{util.NewInt(1), util.NewInt(math.MaxInt32), util.NewInt(1)}
			if mode == "knuth" {
				values = []util.Int{util.NewInt(1), util.NewInt(math.MinInt32), util.NewInt(1)}
			}
			var big util.Int
			if mode == "upper" || mode == "display" {
				serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
				values = []util.Int{big, util.NewInt(1)}
			}
			serviceMust(t, p.SetInts(poly, values))
			serviceMust(t, z.Del(&big))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			var integer, out util.Int
			var binary util.BinaryRational
			var rational util.Rational
			var algebraic Algebraic
			if mode == "integer" {
				serviceMust(t, z.SetInt32(&integer, math.MaxInt32))
			}
			if mode == "binary" {
				serviceMust(t, c.config.Binary.SetInt32(&binary, math.MaxInt32))
			}
			if mode == "rational" {
				serviceMust(t, c.config.Rationals.SetInt32(&rational, math.MaxInt32))
			}
			if mode == "negation" {
				serviceMust(t, p.SetInts(poly, []util.Int{util.NewInt(1), util.NewInt(math.MinInt32), util.NewInt(1)}))
				v, e = p.wholeView(poly)
				serviceMust(t, e)
				var lower, upper util.BinaryRational
				var upperInt util.Int
				serviceMust(t, c.config.Binary.SetInt32(&lower, math.MaxInt32))
				serviceMust(t, z.SetUint32(&upperInt, uint32(math.MaxInt32)+3))
				serviceMust(t, c.config.Binary.SetInteger(&upper, &upperInt))
				serviceMust(t, c.SetRoot(&algebraic, v, &lower, &upper, true))
				serviceMust(t, c.config.Binary.Del(&upper))
				serviceMust(t, c.config.Binary.Del(&lower))
				serviceMust(t, z.Del(&upperInt))
			}
			held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			var body error
			switch mode {
			case "knuth":
				_, body = p.KnuthNegativeRootUpperBound(v)
			case "upper":
				body = p.RootUpperBound(v, &out)
			case "integer":
				_, body = p.EvalSignInt(v, &integer)
			case "binary":
				_, body = p.EvalSignBinary(v, &binary)
			case "rational":
				_, body = p.EvalSignRational(v, &rational)
			case "negation":
				body = c.Neg(&algebraic)
			case "display":
				body = p.DisplayInto(v, &upolyDisplaySink{}, "x", false)
			}
			armed = false
			if !reached || body == nil {
				t.Fatal("actual allocating callback absent", mode, reached, body)
			}
			extra := []*util.Int{&integer, &out, &(*binary.BorrowNumerator()), &(*rational.BorrowNumerator()), &(*rational.BorrowDenominator())}
			if mode == "upper" {
				for i := range p.rootUpperBound.values {
					extra = append(extra, &p.rootUpperBound.values[i])
				}
			}
			if mode == "integer" {
				extra = append(extra, &p.integerEvaluation.values[0])
				if r := p.integerEvaluation.addMul.Frame; r != nil {
					extra = append(extra, &r.Tmp)
				}
			}
			if mode == "binary" {
				for i := range p.binaryEvaluation.values {
					extra = append(extra, &p.binaryEvaluation.values[i])
				}
				if r := p.binaryEvaluation.addMul.Frame; r != nil {
					extra = append(extra, &r.Tmp)
				}
			}
			if mode == "rational" {
				for i := range p.rationalEvaluation.values {
					extra = append(extra, &p.rationalEvaluation.values[i])
				}
				if r := p.rationalEvaluation.addMul.Frame; r != nil {
					extra = append(extra, &r.Tmp)
				}
			}
			if mode == "display" {
				for q := p.displayFrames; q != nil; q = q.next {
					extra = append(extra, &q.numeral)
				}
				extra = append(extra, &p.display.numeral)
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, extra, []umemory.Block{held}, &algebraic)
			serviceMust(t, rt.SetMaxSize(0))
			switch mode {
			case "knuth":
				serviceMust(t, p.cleanupKnuth(false))
			case "upper":
				serviceMust(t, p.cleanupRootUpperBound(false))
			case "integer":
				serviceMust(t, p.cleanupIntegerEvaluation(false))
			case "binary":
				serviceMust(t, p.cleanupBinaryEvaluation(false))
			case "rational":
				serviceMust(t, p.cleanupRationalEvaluation(false))
			case "negation":
				r := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
				serviceMust(t, c.cleanupNegation(&r, false))
			case "display":
				serviceMust(t, p.CleanupDisplay())
			}
			serviceMust(t, z.Del(&integer))
			serviceMust(t, z.Del(&out))
			serviceMust(t, c.config.Binary.Del(&binary))
			serviceMust(t, c.config.Rationals.Del(&rational))
			serviceMust(t, c.Del(&algebraic))
			serviceMust(t, p.DelPolynomial(&poly))
			switch mode {
			case "knuth":
				serviceMust(t, p.cleanupKnuth(true))
			case "upper":
				serviceMust(t, p.cleanupRootUpperBound(true))
			case "integer":
				serviceMust(t, p.cleanupIntegerEvaluation(true))
			case "binary":
				serviceMust(t, p.cleanupBinaryEvaluation(true))
			case "rational":
				serviceMust(t, p.cleanupRationalEvaluation(true))
			case "negation":
				r := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
				serviceMust(t, c.cleanupNegation(&r, true))
			}
			serviceMust(t, rt.Deallocate(&held))
		})
	}
}

// The basic rational Neg branch does not borrow the UP modular owner.
func TestAlgebraicBasicNegDoesNotBorrowUPModularOwner(t *testing.T) {
	var c *AlgebraicRootCore
	var p *UPolynomialManager
	armed, reached := false, false
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		reached = true
		if c.negation.phase != 1 {
			t.Fatal("expected actual basic rational branch", c.negation.phase)
		}
		used, physical := p.config.Integer.Budget().Used(), p.config.Integer.RuntimeOwner().Stats()
		if numericIntegerModularManager_upolynomialCallerExecuting(p.modular) || p.config.Integer.Budget().Used() != used || p.config.Integer.RuntimeOwner().Stats() != physical {
			t.Error("basic rational branch falsely borrowed unrelated UP modular receiver")
		}
		return f
	}
	var rt *umemory.Runtime
	c, p, rt = algebraicNegTLSFixture(t, transport)
	var r util.Rational
	serviceMust(t, c.config.Rationals.SetInt32(&r, math.MinInt32))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &r))
	serviceMust(t, c.config.Rationals.Del(&r))
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := c.Neg(&a)
	armed = false
	if !reached || body == nil {
		t.Fatal("actual basic MIN allocation callback absent", reached, body)
	}
	algebraicCellSourcePhysical(t, c, nil, nil, []umemory.Block{held}, &a)
	serviceMust(t, rt.SetMaxSize(0))
	receipt := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
	serviceMust(t, c.cleanupNegation(&receipt, false))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.cleanupNegation(&receipt, true))
	serviceMust(t, rt.Deallocate(&held))
}
