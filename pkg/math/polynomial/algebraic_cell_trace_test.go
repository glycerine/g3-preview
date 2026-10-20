// Source: src/math/polynomial/algebraic_numbers.cpp:476-506,1479-1493.
package polynomial

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCellSetterTraceLiteralAndConstructorBoundary(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	linear := algebraicPolynomial(t, p, -1, 1)
	quadratic := algebraicPolynomial(t, p, -2, 0, 1)
	lv, e := p.wholeView(linear)
	serviceMust(t, e)
	qv, e := p.wholeView(quadratic)
	serviceMust(t, e)
	lo, hi := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	a, e := c.mkAlgebraicCell(qv, &lo, &hi, true)
	serviceMust(t, e)
	if text.Len() != 0 {
		t.Fatal("constructor emitted setter record", text.String())
	}
	for _, nonlinear := range []bool{false, true, true, false} {
		text.Reset()
		v, want := lv, "a: (# - 1, 1)\n"
		if nonlinear {
			v, want = qv, "a: (#^2 - 2, 2)\n"
		}
		serviceMust(t, c.SetRoot(&a, v, &lo, &hi, true))
		if tags.ShouldTrace("algebraic") {
			expected := "-------- [algebraic] set src/math/polynomial/algebraic_numbers.cpp:506 ---------\n" + want + "------------------------------------------------\n"
			if text.String() != expected {
				t.Fatal("setter source record", text.String(), expected)
			}
		} else if text.Len() != 0 {
			t.Fatal("erased setter record", text.String())
		}
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicCellSetterTraceDisabledSkipsOperand(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	serviceMust(t, c.traceSetRoot(nil))
}

func TestAlgebraicCellSetterTraceCallbackAfterLinearDestructor(t *testing.T) {
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	poly := algebraicPolynomial(t, p, -1, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var out Algebraic
	calls := 0
	writer := &linearOperationProbeWriter{probe: func() {
		calls++
		if out.slot == 0 || c.localActive {
			t.Error("setter trace preceded assignment or scoped_mpq destruction")
		}
		before, callers := out, c.callers
		z := c.config.Rationals.Integers()
		used, physical := z.Budget().Used(), z.RuntimeOwner().Stats()
		for _, call := range []func() error{c.Close, c.CleanupRetained, func() error { return c.Del(&out) }} {
			if e := call(); !errors.Is(e, ErrAlgebraicPending) {
				t.Error("setter trace callback acquired core", e)
			}
		}
		if e := p.Close(); !errors.Is(e, ErrUPolynomialCleanupPending) {
			t.Error("setter trace callback acquired UP manager", e)
		}
		if out != before || callers != c.callers || used != z.Budget().Used() || physical != z.RuntimeOwner().Stats() {
			t.Error("refused setter callback changed graph")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	serviceMust(t, c.SetRoot(&out, v, nil, nil, false))
	if tags.ShouldTrace("algebraic") && calls == 0 {
		t.Fatal("setter trace callback was not reached")
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.Del(&out))
}

func TestAlgebraicCellSetterTraceSourceStopKeepsPublishedCaller(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "linear", true: "mixed-add"}[mixed], func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			var input, out Algebraic
			var v UPolynomialView
			if mixed {
				input = algebraicLargeDyadicPromotionRoot(t, c)
			} else {
				z := c.config.Rationals.Integers()
				var n unumeral.Int
				serviceMust(t, z.SetString(&n, "-18446744073709551619"))
				poly, e := p.NewPolynomial()
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, p.DelPolynomial(&poly)) })
				serviceMust(t, p.SetInts(poly, []unumeral.Int{n, unumeral.NewInt(1)}))
				serviceMust(t, z.Del(&n))
				v, e = p.wholeView(poly)
				serviceMust(t, e)
			}
			writer := &algebraicBinaryDisplayStopWriter{core: c, marker: ":506 "}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			var body error
			if mixed {
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetInt64(&q, 1))
				body = c.AddRational(&input, &q, &out)
				serviceMust(t, c.config.Rationals.Del(&q))
			} else {
				body = c.SetRoot(&out, v, nil, nil, false)
			}
			if tags.ShouldTrace("algebraic") {
				if !errors.Is(body, arena.ErrBounds) || writer.ref == 0 || out.slot == 0 || !c.rootTracePending() || c.localActive {
					t.Fatal("setter display crossed publication or local destructor", body, out, writer.ref)
				}
				stop := c.rootDisplay.sourceStop
				if stop == nil {
					t.Fatal("actual display source stop not retained")
				}
				if mixed && (!c.arithmetic.active || c.arithmetic.sourceStop != stop || c.arithmetic.stage != 0 || c.arithmetic.bCount != 2) {
					t.Fatal("setter source stop consumed older arithmetic locals", c.arithmetic.sourceStop)
				}
				if e := c.Del(&out); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending setter display admitted deletion", e)
				}
				c.config.Rationals.Integers().Store(writer.ref, writer.original)
			} else {
				serviceMust(t, body)
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootTracePending() || c.arithmetic.active {
				t.Fatal("owning cleanup kept setter display or arithmetic")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&input))
		})
	}
}

func TestAlgebraicCellSetterTraceActualReservationPrefixes(t *testing.T) {
	succeeded, reached, traced := false, false, false
	for prefix := int64(0); prefix < 2048 && !succeeded; prefix++ {
		if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			seed := algebraicTinyBinaryBoundsRoot(t, c)
			loc, e := c.location(&seed)
			serviceMust(t, e)
			record, e := c.load(loc)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&record)
			serviceMust(t, e)
			var text strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			tags.Enable("algebraic")
			traced = tags.ShouldTrace("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			z := c.config.Rationals.Integers()
			var out Algebraic
			z.Budget().FailAfter(prefix)
			body := c.SetRoot(&out, v, record.interval.BorrowLower(), record.interval.BorrowUpper(), true)
			z.Budget().FailAfter(-1)
			if body == nil {
				succeeded = true
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal("setter left actual numerical reservation path", body)
			}
			if c.rootTracePending() {
				reached = true
				if out.slot == 0 || c.pending.slot != 0 || !c.rootDisplayInvocationPending(&c.rootTrace.root) {
					t.Fatal("setter display failure preceded caller publication")
				}
				if e := c.Del(&out); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("setter display allowed caller deletion", e)
				}
			} else if !c.cellEvaluationPending() {
				serviceMust(t, z.CleanupPendingAllocation())
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			serviceMust(t, c.CleanupRetained())
			if c.rootTracePending() || c.cellEvaluationPending() {
				t.Fatal("setter acknowledgement retained numerical children")
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&seed))
		}) {
			return
		}
	}
	if !succeeded {
		t.Fatal("setter reservation campaign did not reach success")
	}
	// Native builds erase TRACE; only traced builds must reach display storage.
	if traced && !reached {
		t.Fatal("traced setter campaign missed actual retained display")
	}
}
