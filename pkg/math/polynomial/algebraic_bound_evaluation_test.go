package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestAlgebraicPrecisionBoundOwnsActualHornerChild(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(map[bool]string{false: "lower", true: "upper"}[upper], func(t *testing.T) {
			c, p := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicTinyBinaryBoundsRoot(t, c)
			q := c.config.Rationals
			out := unumeral.NewRational()
			get := func() error {
				if upper {
					return c.GetUpperRationalAtPrecision(&root, &out, 0)
				}
				return c.GetLowerRationalAtPrecision(&root, &out, 0)
			}
			pad := algebraicCellSourceTLSBoundary(t, c)
			body := get()
			if body == nil || !c.bounds.evaluationPending(c) || !p.matchesBinaryEvaluationReceipt(&c.bounds.evaluation) {
				t.Fatal("missing actual failed Horner child", body)
			}
			if e := get(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("pending Horner child replaced", e)
			}
			runtime := q.Integers().RuntimeOwner()
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			serviceMust(t, c.CleanupRetained())
			if p.binaryEvaluation.active {
				t.Fatal("bounds cleanup retained its Horner child")
			}
			serviceMust(t, q.Del(&out))
			// This refusal occurred in the first Horner copy, before source
			// scratch transfer. Owning acknowledgement permits this retry.
			p.config.Checkpoint = func() bool { return true }
			p.config.Cooperation = func() error { return nil }
			out = unumeral.NewRational()
			serviceMust(t, get())
			want := unumeral.NewRational()
			text := "1/18446744073709551616"
			if upper {
				text = "1/9223372036854775808"
			}
			serviceMust(t, q.SetString(&want, text))
			equal, e := q.Equal(&out, &want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("bound changed after Horner acknowledgement", text)
			}
			serviceMust(t, q.Del(&want))
			serviceMust(t, q.Del(&out))
			serviceMust(t, c.Del(&root))
		})
	}
}

func TestAlgebraicPrecisionBoundPreservesOlderHornerChild(t *testing.T) {
	c, p := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e := c.sourcePolynomial(&r)
	serviceMust(t, e)
	a := unumeral.NewBinaryIntPower(1, 64)
	pad := algebraicCellSourceTLSBoundary(t, c)
	_, body := p.EvalSignBinary(v, &a)
	if body == nil || !p.binaryEvaluation.active {
		t.Fatal("missing actual older Horner entry", body)
	}
	older := p.binaryEvaluationReceipt()
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	out := unumeral.NewRational()
	body = c.GetLowerRationalAtPrecision(&root, &out, 0)
	if !errors.Is(body, ErrUPolynomialCleanupPending) || c.bounds.evaluation.owner != nil {
		t.Fatal("refused getter acquired older evaluation", body)
	}
	state, used, stats := p.binaryEvaluation, p.config.Integer.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !reflect.DeepEqual(state, p.binaryEvaluation) || used != p.config.Integer.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("bounds cleanup consumed older Horner owner")
	}
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, false))
	serviceMust(t, p.cleanupBinaryEvaluationWithReceipt(&older, true))
	serviceMust(t, c.config.Rationals.Del(&out))
	serviceMust(t, c.Del(&root))
}
