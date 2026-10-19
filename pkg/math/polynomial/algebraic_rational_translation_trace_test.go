package polynomial

import (
	"errors"
	"fmt"
	"io"
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
	marker   string
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
}

func (w *rationalTranslationInputWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *rationalTranslationInputWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if f := w.core.config.Polynomials.currentDisplay; w.stop && strings.Contains(w.String(), w.marker) && w.ref == 0 && f != nil && f.constructed && f.numeral.Cell != 0 {
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

func TestAlgebraicRationalTranslationBoundsAndResultRecords(t *testing.T) {
	for _, subtract := range []bool{false, true} {
		for _, den := range []uint64{2, 3} {
			t.Run(fmt.Sprint(subtract, "/", den), func(t *testing.T) {
				c, p, _ := algebraicCompareFixtureProfile(t, true)
				poly := algebraicPolynomial(t, p, -2, 0, 1)
				v, e := p.wholeView(poly)
				serviceMust(t, e)
				var lo, hi unumeral.BinaryRational
				serviceMust(t, c.config.Binary.SetInt32(&lo, 1))
				serviceMust(t, c.config.Binary.SetInt32(&hi, 2))
				var a, out Algebraic
				serviceMust(t, c.SetRoot(&a, v, &lo, &hi, true))
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetFraction(&q, 1, den))
				var text strings.Builder
				tags := diagnostic.NewTraceTagsWithWriter(&text)
				tags.Enable("algebraic")
				serviceMust(t, c.config.Rationals.BindTrace(tags))
				if subtract {
					e = c.SubRational(&a, &q, &out)
				} else {
					e = c.AddRational(&a, &q, &out)
				}
				serviceMust(t, e)
				serviceMust(t, c.config.Rationals.BindTrace(nil))
				if tags.ShouldTrace("algebraic") {
					got := text.String()
					if den == 3 {
						sign := ""
						if subtract {
							sign = "-"
						}
						if !strings.Contains(got, "nbv: "+sign+"1/3\nil: 1, iu: 2\n") {
							t.Fatal("nonbinary pretranslation bounds", got)
						}
						coefficient := "- 6"
						if subtract {
							coefficient = "+ 6"
						}
						if !strings.Contains(got, "9 x^2 "+coefficient+" x - 17, l: ") {
							t.Fatal("translated polynomial", got)
						}
					} else {
						if strings.Contains(got, "nbv: ") {
							t.Fatal("dyadic branch executed rational-bound diagnostic", got)
						}
						want := "4 x^2 - 4 x - 7, l: 3/2, u: 5/2\nl_sign: -1\nu_sign: 1\n"
						if subtract {
							want = "4 x^2 + 4 x - 7, l: 1/2, u: 3/2\nl_sign: -1\nu_sign: 1\n"
						}
						if !strings.Contains(got, want) {
							t.Fatal("dyadic result record", got, want)
						}
					}
					if c.arithmetic.traceEvaluations[0].owner != p || c.arithmetic.traceEvaluations[1].owner != p {
						t.Fatal("missing actual Horner children")
					}
				} else if text.Len() != 0 || c.arithmetic.traceEvaluations[0].owner != nil || c.arithmetic.traceEvaluations[1].owner != nil {
					t.Fatal("erased diagnostic evaluated operands")
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.Del(&out))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
				serviceMust(t, c.config.Binary.Del(&hi))
				serviceMust(t, c.config.Binary.Del(&lo))
			})
		}
	}
}

func TestAlgebraicRationalTranslationResultDisplayRetainsOlderBounds(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
	w := &rationalTranslationInputWriter{core: c, stop: true, marker: ":1425 "}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	var out Algebraic
	e := c.AddRational(&a, &q, &out)
	if tags.ShouldTrace("algebraic") {
		if !errors.Is(e, arena.ErrBounds) || w.ref == 0 || !c.rootTracePending() || !c.arithmetic.active || c.arithmetic.qCount != 1 || c.arithmetic.bCount != 2 || out.slot != 0 {
			t.Fatal("final display lost older locals or published result", e)
		}
		if c.arithmetic.traceEvaluations[0].owner != nil || c.arithmetic.traceEvaluations[1].owner != nil {
			t.Fatal("evaluations crossed polynomial display stop")
		}
		c.config.Rationals.Integers().Store(w.ref, w.original)
	} else {
		serviceMust(t, e)
	}
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	serviceMust(t, c.CleanupRetained())
	if c.arithmetic.active || c.rootTracePending() {
		t.Fatal("owning final-display cleanup")
	}
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}

type rationalTranslationEvaluationWriter struct {
	strings.Builder
	core   *AlgebraicRootCore
	marker string
	prefix int64
	armed  bool
}

func (w *rationalTranslationEvaluationWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *rationalTranslationEvaluationWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if w.armed && strings.Contains(w.String()[strings.LastIndex(w.String(), w.marker):], "\n") {
		w.core.config.Rationals.Budget().FailAfter(-1)
	}
	if !w.armed && strings.HasSuffix(w.String(), w.marker) {
		w.armed = true
		w.core.config.Rationals.Budget().FailAfter(w.prefix)
	}
	return n, e
}
func TestAlgebraicRationalTranslationTraceActualEvaluationReservations(t *testing.T) {
	for _, marker := range []string{"l_sign: ", "u_sign: "} {
		t.Run(marker, func(t *testing.T) {
			succeeded, refused, raw := false, false, false
			for prefix := int64(0); prefix < 512 && !succeeded; prefix++ {
				if !t.Run(fmt.Sprint(prefix), func(t *testing.T) {
					c, p, _ := algebraicCompareFixtureProfile(t, true)
					a := algebraicLargeDyadicPromotionRoot(t, c)
					q := unumeral.NewRational()
					serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 3))
					w := &rationalTranslationEvaluationWriter{core: c, marker: marker, prefix: prefix}
					tags := diagnostic.NewTraceTagsWithWriter(w)
					tags.Enable("algebraic")
					serviceMust(t, c.config.Rationals.BindTrace(tags))
					var out Algebraic
					body := c.AddRational(&a, &q, &out)
					c.config.Rationals.Budget().FailAfter(-1)
					serviceMust(t, c.config.Rationals.BindTrace(nil))
					if !tags.ShouldTrace("algebraic") {
						serviceMust(t, body)
						succeeded = true
						if w.armed {
							t.Fatal("erased trace armed numeric failure")
						}
					} else {
						if !w.armed {
							t.Fatal("diagnostic did not reach selected evaluation", body, w.String())
						}
						if body == nil {
							succeeded = true
						} else {
							refused = true
							if !numericPhysicalReservationRefusal(body) {
								t.Fatal("evaluation reservation escaped actual allocator", body)
							}
							index := 0
							if marker == "u_sign: " {
								index = 1
							}
							receipt := &c.arithmetic.traceEvaluations[index]
							if receipt.owner != p || receipt.invocation == 0 || out.slot != 0 {
								t.Fatal("failed Horner child lost identity or published output", body)
							}
							if p.matchesBinaryEvaluationReceipt(receipt) && p.binaryEvaluation.active && p.binaryEvaluation.sourceCause == nil {
								raw = true
								if !c.arithmetic.active || c.arithmetic.stage != 5 || c.arithmetic.sourceStop != nil {
									t.Fatal("RAW body failure crossed completed older source destructors", body, c.arithmetic.stage, c.arithmetic.sourceStop)
								}
							}
						}
					}
					serviceMust(t, c.config.Rationals.Integers().CleanupPendingAllocation())
					serviceMust(t, c.CleanupRetained())
					if c.arithmetic.active || c.rootTracePending() || p.binaryEvaluation.active {
						t.Fatal("actual diagnostic evaluation cleanup remained pending", body)
					}
					serviceMust(t, c.Del(&out))
					serviceMust(t, c.Del(&a))
					serviceMust(t, c.config.Rationals.Del(&q))
				}) {
					return
				}
			}
			if !succeeded {
				t.Fatal("never reached complete trace evaluation")
			}
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("algebraic")
			if tags.ShouldTrace("algebraic") && (!refused || !raw) {
				t.Fatal("evaluation sweep missed actual refusal or retained RAW")
			}
		})
	}
}
