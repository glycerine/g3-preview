package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type rationalTranslationInputWriter struct {
	strings.Builder
	core     *AlgebraicRootCore
	stop     bool
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
}

func (w *rationalTranslationInputWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *rationalTranslationInputWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if f := w.core.config.Polynomials.currentDisplay; w.stop && w.ref == 0 && f != nil && f.constructed && f.numeral.Cell != 0 {
		z := w.core.config.Rationals.Integers()
		w.ref = f.numeral.Cell
		w.original = z.Cell(w.ref)
		guard := w.original
		guard.Backing.Requested = uint64(^uint32(0)) + 1
		z.Store(w.ref, guard)
	}
	return n, e
}
func TestAlgebraicRationalTranslationInputTraceDeclarationFrontier(t *testing.T) {
	for _, stop := range []bool{false, true} {
		for _, subtract := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "success", true: "stop"}[stop], map[bool]string{false: "add", true: "subtract"}[subtract]}, "/"), func(t *testing.T) {
				c, p, _ := algebraicCompareFixtureProfile(t, true)
				poly := algebraicPolynomial(t, p, -2, 0, 1)
				v, e := p.wholeView(poly)
				serviceMust(t, e)
				var lo, hi unumeral.BinaryRational
				serviceMust(t, c.config.Binary.SetInt32(&lo, 1))
				serviceMust(t, c.config.Binary.SetInt32(&hi, 2))
				var a, out Algebraic
				if stop {
					a = algebraicLargeDyadicPromotionRoot(t, c)
				} else {
					serviceMust(t, c.SetRoot(&a, v, &lo, &hi, true))
				}
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/3"))
				w := &rationalTranslationInputWriter{core: c, stop: stop}
				tags := diagnostic.NewTraceTagsWithWriter(w)
				tags.Enable("algebraic")
				serviceMust(t, c.config.Rationals.BindTrace(tags))
				if subtract {
					e = c.SubRational(&a, &q, &out)
				} else {
					e = c.AddRational(&a, &q, &out)
				}
				if stop && tags.ShouldTrace("algebraic") {
					if !errors.Is(e, arena.ErrBounds) || w.ref == 0 || !c.rootTracePending() || !c.arithmetic.active || c.arithmetic.qCount != 0 || c.arithmetic.bCount != 0 || c.addTmp.slot != 0 {
						t.Fatal("input display crossed temporary declaration", e, c.arithmetic.qCount, c.arithmetic.bCount)
					}
					c.config.Rationals.Integers().Store(w.ref, w.original)
				} else {
					serviceMust(t, e)
					if !stop && tags.ShouldTrace("algebraic") && !strings.Contains(w.String(), "adding algebraic and basic cells:\na: x^2 - 2 (1, 2)\nb: 18446744073709551619/3\n") {
						t.Fatal(w.String())
					}
				}
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				serviceMust(t, c.CleanupRetained())
				if c.rootTracePending() || c.arithmetic.active {
					t.Fatal("retained translation display")
				}
				serviceMust(t, c.Del(&out))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
				serviceMust(t, c.config.Binary.Del(&hi))
				serviceMust(t, c.config.Binary.Del(&lo))
			})
		}
	}
}
