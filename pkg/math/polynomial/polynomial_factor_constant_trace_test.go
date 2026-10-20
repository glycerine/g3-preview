package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"strings"
	"testing"
)

type polynomialConstantTraceWriter struct{ polynomialQuadraticTraceWriter }

func (w *polynomialConstantTraceWriter) Write(p []byte) (int, error) {
	if w.calls == 0 && (w.f.localActive || w.f.constantInvocation.localConstructed) {
		w.t.Fatal("entry TRACE constructed later source numeral")
	}
	if e := w.m.core.integer.CloseChecked(); !errors.Is(e, unumeral.ErrNumericPending) {
		w.t.Fatal("constant TRACE released integer owner", e)
	}
	if e := w.m.core.integer.Natural.CloseChecked(); !errors.Is(e, arena.ErrInvalidSlot) {
		w.t.Fatal("constant TRACE released natural owner", e)
	}
	return w.polynomialQuadraticTraceWriter.Write(p)
}
func TestPolynomialFactorConstantTraceBIGAndRefusingWriter(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		z := m.core.integer
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		var input unumeral.Int
		serviceMust(t, z.SetString(&input, "-18446744073709551616"))
		var out bytes.Buffer
		writer := &polynomialConstantTraceWriter{polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}}
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		if refuse {
			tags = diagnostic.NewTraceTagsWithWriter(writer)
		}
		tags.Enable("factor_bug")
		m.core.monomials.config.Trace = tags
		var receipt polynomialFactorConstantInvocationReceipt
		defer func() {
			tags.Finalize()
			serviceMust(t, f.cleanupFactorConstantInvocation(&receipt, false))
			serviceMust(t, f.cleanupFactorConstantInvocation(&receipt, true))
			serviceMust(t, f.CloseChecked())
			serviceMust(t, z.Del(&input))
		}()
		serviceMust(t, m.accumulateFactorConstantWithReceipt(f, &input, &receipt))
		equal, e := z.Equal(&f.constant, &input)
		serviceMust(t, e)
		if !equal {
			t.Fatal("constant accumulation value")
		}
		if tags.ShouldTrace("factor_bug") {
			if refuse {
				if writer.calls == 0 {
					t.Fatal("writer not reached")
				}
			} else if !strings.Contains(out.String(), ":6578 ") || !strings.Contains(out.String(), "acc_constant, c: -18446744073709551616\n") {
				t.Fatal("literal accumulation record", out.String())
			}
			if receipt.frame.traceNumeral.Character == nil {
				t.Fatal("actual BIG constant formatting invocation lost")
			}
		} else if out.Len() != 0 || writer.calls != 0 {
			t.Fatal("disabled TRACE emitted output")
		}
	}
}

func TestPolynomialFactorConstantTraceEveryReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			z := m.core.integer
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			var input unumeral.Int
			serviceMust(t, z.SetString(&input, "18446744073709551616"))
			var out bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&out)
			tags.Enable("factor_bug")
			m.core.monomials.config.Trace = tags
			var receipt polynomialFactorConstantInvocationReceipt
			defer func() {
				tags.Finalize()
				m.originalBudget.FailAfter(-1)
				serviceMust(t, f.cleanupFactorConstantInvocation(&receipt, false))
				serviceMust(t, f.cleanupFactorConstantInvocation(&receipt, true))
				serviceMust(t, f.CloseChecked())
				serviceMust(t, z.Del(&input))
			}()
			m.originalBudget.FailAfter(ordinal)
			cause := m.accumulateFactorConstantWithReceipt(f, &input, &receipt)
			m.originalBudget.FailAfter(-1)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual traced accumulator reservation", cause)
			}
			if cause == nil {
				success = true
				equal, e := z.Equal(&f.constant, &input)
				serviceMust(t, e)
				if !equal {
					t.Fatal("actual accumulator value")
				}
				if tags.ShouldTrace("factor_bug") && receipt.frame.traceNumeral.Character == nil {
					t.Fatal("actual accumulator trace receipt missing")
				}
			}
		})
	}
	if !success {
		t.Fatal("actual traced accumulator never succeeded")
	}
}
