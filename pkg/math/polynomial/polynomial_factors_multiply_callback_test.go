package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFactorsMultiplyCooperationCannotDestroyCaller(t *testing.T) {
	for _, action := range []string{"source", "cleanup", "close"} {
		t.Run(action, func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			serviceMust(t, f.Push(p, 3))
			out, e := NewPolynomialRef(m.core)
			serviceMust(t, e)
			original := m.core.config.Cooperation
			calls := 0
			var refused error
			m.core.config.Cooperation = func() error {
				calls++
				if calls == 1 {
					current, constant, header := f.current.value, f.constant, f.header
					switch action {
					case "source":
						refused = f.SourceFinalize()
					case "cleanup":
						refused = f.Cleanup()
					case "close":
						refused = f.CloseChecked()
					}
					if f.current.value != current || f.constant != constant || f.header != header || f.sourceStarted || f.closing || f.closed {
						t.Error("callback refusal changed actual factors/local ownership")
					}
				}
				return original()
			}
			t.Cleanup(func() {
				m.core.config.Cooperation = original
				serviceMust(t, f.Cleanup())
				serviceMust(t, out.Close())
				serviceMust(t, f.CloseChecked())
			})
			body := f.Multiply(out)
			m.core.config.Cooperation = original
			if calls == 0 || !errors.Is(refused, ErrPolynomialPending) || body != nil {
				t.Fatal("actual cooperation destroyed active factors caller", calls, refused, body)
			}
			if f.closed || f.closing || f.sourceStarted || len(f.factors) != 1 || f.degrees[0] != 3 {
				t.Fatal("refused callback changed source factor fields")
			}
			result, e := out.Get()
			serviceMust(t, e)
			x, value := unumeral.NewInt(2), unumeral.Int{}
			serviceMust(t, m.UnivariateEvaluate(result, 0, &x, &value))
			want := unumeral.NewInt(27)
			equal, e := m.core.integer.Equal(&value, &want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("continued product lost (2+1)^3", value)
			}
			serviceMust(t, m.core.integer.Del(&value))
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil, out)
		})
	}
}

func TestPolynomialFactorsResetDeletionCannotReenterCaller(t *testing.T) {
	for _, operation := range []string{"reset", "source", "cleanup", "close"} {
		t.Run(operation, func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, nil)
			constant := unumeral.NewInt(17)
			q, e := m.core.MkConstantTake(&constant)
			serviceMust(t, e)
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			serviceMust(t, f.Push(q, 2))
			state := &polynomialFactorsCallbackState{target: q, guardTarget: f}
			polynomialFactorsCallback = state
			h, e := NewPolynomialDeletionHandler(m.core.integer.Budget(), polynomialFactorsDeletionCopy)
			serviceMust(t, e)
			serviceMust(t, m.core.AddDeletionHandler(h))
			t.Cleanup(func() {
				polynomialFactorsCallback = nil
				serviceMust(t, f.CloseChecked())
				serviceMust(t, m.core.RemoveDeletionHandler(h))
				serviceMust(t, h.CloseChecked())
			})
			var body error
			switch operation {
			case "reset":
				body = f.Reset()
			case "source":
				body = f.SourceFinalize()
			case "cleanup":
				body = f.Cleanup()
			case "close":
				body = f.CloseChecked()
			}
			if body != nil || state.calls != 1 {
				t.Fatal("actual deletion callback failed or replayed", body, state.calls)
			}
			for _, e := range []error{state.guardSource, state.guardCleanup, state.guardClose} {
				if !errors.Is(e, ErrPolynomialPending) {
					t.Fatal("deletion callback entered caller lifetime continuation", e)
				}
			}
			if f.closing != (operation == "close") || f.closed != (operation == "close") || f.sourceStarted != (operation == "source") || f.resetting || len(f.factors) != 0 || f.total != 0 {
				t.Fatal("reentry changed actual class/source state", f.closing, f.closed, f.sourceStarted, f.resetting, len(f.factors), f.total)
			}
			polynomialFactorsAuditExtra(t, f, []Polynomial{p}, []*unumeral.Int{&state.source, &state.foreign}, h.header)
			if operation != "close" {
				serviceMust(t, f.SourceFinalize())
			}
		})
	}
}

func TestPolynomialFactorsCleanupDeletionCannotMutateCaller(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	value := unumeral.NewInt(17)
	q, e := m.core.MkConstantTake(&value)
	serviceMust(t, e)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	serviceMust(t, f.Push(q, 2))
	serviceMust(t, f.Push(p, 3))
	state := &polynomialFactorsCallbackState{target: q, guardTarget: f, mutate: true, mutationPolynomial: p}
	polynomialFactorsCallback = state
	h, e := NewPolynomialDeletionHandler(m.core.integer.Budget(), polynomialFactorsDeletionCopy)
	serviceMust(t, e)
	serviceMust(t, m.core.AddDeletionHandler(h))
	t.Cleanup(func() {
		polynomialFactorsCallback = nil
		serviceMust(t, f.CloseChecked())
		serviceMust(t, m.core.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
	})
	serviceMust(t, f.Cleanup())
	if state.calls != 1 {
		t.Fatal("actual cleanup deletion callback not reached exactly once", state.calls)
	}
	for _, e := range []error{state.push, state.setDegree, state.setConstant} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("owning cleanup accepted new factors work from its deletion callback", state.push, state.setDegree, state.setConstant)
		}
	}
	if !state.mutationUnchanged {
		t.Fatal("refused callback published a factor/degree/constant or foreign cause")
	}
	polynomialFactorsAuditExtra(t, f, []Polynomial{p}, []*unumeral.Int{&state.source, &state.foreign}, h.header)
	// Retiring the synchronous cleanup span must permit the next source call.
	serviceMust(t, f.Push(p, 1))
	serviceMust(t, f.Reset())
}
