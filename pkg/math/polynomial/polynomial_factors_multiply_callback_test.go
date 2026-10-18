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
