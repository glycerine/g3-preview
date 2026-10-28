package polynomial

// Source: algebraic_numbers.cpp:1498-1539.

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicRationalMultiplyDiagnosticRecords(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, den := range []uint64{2, 3} {
			t.Run(fmt.Sprint(negative, "/", den), func(t *testing.T) {
				c, p, _ := algebraicCompareFixtureProfile(t, true)
				poly := algebraicPolynomial(t, p, -2, 0, 1)
				v, e := p.wholeView(poly)
				serviceMust(t, e)
				lo, hi := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
				var a, out Algebraic
				serviceMust(t, c.SetRoot(&a, v, &lo, &hi, true))
				q := unumeral.NewRational()
				n := int64(1)
				if negative {
					n = -1
				}
				serviceMust(t, c.config.Rationals.SetFraction(&q, n, den))
				var text strings.Builder
				tags := diagnostic.NewTraceTagsWithWriter(&text)
				tags.Enable("algebraic")
				serviceMust(t, c.config.Rationals.BindTrace(tags))
				serviceMust(t, c.MulRational(&a, &q, &out))
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				if tags.ShouldTrace("algebraic") {
					sign := ""
					if negative {
						sign = "-"
					}
					if !strings.Contains(text.String(), "mult algebraic and basic cells:\na: x^2 - 2 (1, 2)\nb: "+sign+"1/"+fmt.Sprint(den)+"\n") {
						t.Fatal("multiplication input record", text.String())
					}
					if den == 3 {
						if !strings.Contains(text.String(), "nbv: "+sign+"1/3\nil: 1, iu: 2\n") || !strings.Contains(text.String(), "9 x^2 - 2, l: ") {
							t.Fatal("nonbinary multiplication record", text.String())
						}
					} else {
						want := "4 x^2 - 2, l: 1/2, u: 1\nl_sign: -1\nu_sign: 1\n"
						if negative {
							want = "4 x^2 - 2, l: -1, u: -1/2\nl_sign: 1\nu_sign: -1\n"
						}
						if strings.Contains(text.String(), "nbv: ") || !strings.Contains(text.String(), want) {
							t.Fatal("dyadic multiplication record", text.String(), want)
						}
					}
					if c.arithmetic.evaluations[0].owner != p || c.arithmetic.evaluations[1].owner != p {
						t.Fatal("missing diagnostic Horner invocations")
					}
				} else if text.Len() != 0 || c.arithmetic.evaluations[0].owner != nil || c.arithmetic.evaluations[1].owner != nil {
					t.Fatal("erased TRACE evaluated operands")
				}
				enabled, e := diagnostic.SourceAssertionsEnabled(diagnostic.BorrowProcessDebugContext())
				serviceMust(t, e)
				for i := 2; i < 4; i++ {
					if (c.arithmetic.evaluations[i].owner == p) != (polynomialSourceDebug && enabled) {
						t.Fatal("assertion evaluation gate", i)
					}
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Del(&out))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
			})
		}
	}
}

func TestAlgebraicRationalMultiplyDisplayDeclarationFrontiers(t *testing.T) {
	for _, marker := range []string{":1498 ", ":1535 "} {
		t.Run(marker, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicLargeDyadicPromotionRoot(t, c)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
			w := &rationalTranslationInputWriter{core: c, stop: true, marker: marker}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			var out Algebraic
			body := c.MulRational(&a, &q, &out)
			if tags.ShouldTrace("algebraic") {
				qCount, bCount := uint8(0), uint8(0)
				if marker == ":1535 " {
					qCount, bCount = 1, 2
				}
				if !errors.Is(body, arena.ErrBounds) || w.ref == 0 || !c.rootTracePending() || !c.arithmetic.active || c.arithmetic.qCount != qCount || c.arithmetic.bCount != bCount || out.slot != 0 {
					t.Fatal("display crossed multiplication source declaration", body)
				}
				if c.arithmetic.sourceStop == nil || c.arithmetic.sourceStop != c.rootTraceSourceCause() {
					t.Fatal("lost exact formatter source stop")
				}
				if marker == ":1498 " && (c.addTmp.slot != 0 || c.arithmetic.evaluations[2].owner != nil || c.arithmetic.evaluations[3].owner != nil) {
					t.Fatal("debug assertion crossed input formatter")
				}
				if c.arithmetic.evaluations[0].owner != nil || c.arithmetic.evaluations[1].owner != nil {
					t.Fatal("final diagnostic evaluated past stopped formatter")
				}
				c.config.Rationals.Integers().Store(w.ref, w.original)
			} else {
				serviceMust(t, body)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.arithmetic.active || c.rootTracePending() {
				t.Fatal("multiplication formatter retained after acknowledgement")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		})
	}
}

func TestAlgebraicRationalMultiplyAssertionActualHornerOwnership(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	enabled, e := diagnostic.SourceAssertionsEnabled(diagnostic.BorrowProcessDebugContext())
	serviceMust(t, e)
	if !enabled {
		t.Skip("source assertions disabled")
	}
	reached := [2]bool{}
	done := false
	for prefix := int64(0); prefix < 512 && !done; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicLargeDyadicPromotionRoot(t, c)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			var out Algebraic
			body := c.MulRational(&a, &q, &out)
			budget.FailAfter(-1)
			if c.arithmetic.qCount != 0 {
				done = true
			} else {
				if body == nil || !numericPhysicalReservationRefusal(body) || c.arithmetic.bCount != 0 || c.addTmp.slot != 0 || out.slot != 0 {
					t.Fatal("assertion failure crossed nbv declaration", body)
				}
				for i := 2; i < 4; i++ {
					r := &c.arithmetic.evaluations[i]
					if p.matchesBinaryEvaluationReceipt(r) && p.binaryEvaluation.active {
						reached[i-2] = true
						if !c.arithmetic.active || c.arithmetic.sourceStop == nil && c.arithmetic.stage != 5 {
							t.Fatal("lost actual assertion child owner", body)
						}
						if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("pending arithmetic Horner child admitted ordinary deletion", e)
						}
					}
				}
			}
			serviceMust(t, c.CleanupRetained())
			algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
			if c.arithmetic.active || p.binaryEvaluation.active {
				t.Fatal("actual assertion Horner cleanup pending", body)
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}) {
			return
		}
	}
	if !done || !reached[0] || !reached[1] {
		t.Fatal("assertion sweep missed lower/upper actual child", done, reached)
	}
}

func TestAlgebraicRationalMultiplyStickyStreamStillEvaluatesSigns(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	serviceMust(t, c.MulRational(&a, &q, &out))
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if tags.ShouldTrace("algebraic") {
		if w.calls != 1 || c.arithmetic.evaluations[0].owner != p || c.arithmetic.evaluations[1].owner != p {
			t.Fatal("sticky stream skipped sign operands", w.calls)
		}
	} else if w.calls != 0 {
		t.Fatal("erased TRACE wrote to stream")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicRationalMultiplyAssertionTransportProtectsOwner(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	var c *AlgebraicRootCore
	var a, out Algebraic
	var q unumeral.Rational
	armed, calls := false, 0
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		calls++
		before := c.arithmetic
		original := a
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{
			func() error { return c.Del(&a) },
			func() error { return c.MulRational(&a, &q, &out) },
			func() error { return c.CleanupRetained() },
			func() error { return c.Close() },
		} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("actual assertion transport acquired arithmetic owner", e)
			}
		}
		if !reflect.DeepEqual(before, c.arithmetic) || a != original || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused assertion callback changed source owner or allocation graph")
		}
		return f
	}
	var p *UPolynomialManager
	var rt *umemory.Runtime
	c, p, rt = algebraicNegTLSFixture(t, transport)
	a = algebraicLargeDyadicPromotionRoot(t, c)
	q = unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	body := c.MulRational(&a, &q, &out)
	armed = false
	if body == nil || calls == 0 || !c.arithmetic.active || c.arithmetic.qCount != 0 || !p.matchesBinaryEvaluationReceipt(&c.arithmetic.evaluations[2]) {
		t.Fatal("actual assertion transport not owned before nbv", body, calls)
	}
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
