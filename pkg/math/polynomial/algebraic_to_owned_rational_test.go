package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicToOwnedRationalZeroBasicAndPredicateConversion(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	s, _ := serviceNew(t) // a distinct global/service owner
	q := c.config.Rationals
	var zero, basic Algebraic
	value := unumeral.NewRational()
	serviceMust(t, q.SetFraction(&value, 7, 3))
	serviceMust(t, c.SetRational(&basic, &value))
	root := algebraicCompareRoot(t, c, []int32{-2, 1, 1}, 0, 0, 2, 0, false)
	for i, a := range []*Algebraic{&zero, &basic, &root} {
		out, e := s.NewValue()
		serviceMust(t, e)
		serviceMust(t, s.SetInt64(&out, 99))
		serviceMust(t, c.ToOwnedRational(a, s, &out))
		got, e := s.String(&out)
		serviceMust(t, e)
		if want := []string{"0", "7/3", "1"}[i]; got != want {
			t.Fatal("source conversion", got, want)
		}
		if c.toOwnedRational.active {
			t.Fatal("successful wrapper retained locals")
		}
		serviceMust(t, s.Del(&out))
	}
	serviceMust(t, c.Del(&root))
	serviceMust(t, c.Del(&basic))
	serviceMust(t, q.Del(&value))
}

func TestAlgebraicToOwnedRationalReturningTransportGuards(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	q := c.config.Rationals
	b := arena.NewBudget(1 << 26)
	active, callbacks := false, 0
	var closeErr, coreCleanupErr, wrapperCleanupErr, rationalCloseErr error
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			callbacks++
			closeErr = c.Close()
			coreCleanupErr = c.CleanupRetained()
			wrapperCleanupErr = c.CleanupToOwnedRational()
			rationalCloseErr = q.CloseChecked()
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = runtime
	s, e := unumeral.NewRationalService(cfg)
	serviceMust(t, e)
	value := unumeral.NewRational()
	words := make([]uint32, 2000)
	words[0], words[len(words)-1] = 17, 1
	serviceMust(t, q.Integers().SetLimbs(&value.Num, words))
	var input Algebraic
	serviceMust(t, c.SetRational(&input, &value))
	out, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&out, 99))
	t.Cleanup(func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, c.CleanupToOwnedRational())
		serviceMust(t, s.CleanupValue(&out))
		serviceMust(t, c.Del(&input))
		serviceMust(t, q.Del(&value))
		serviceMust(t, s.Close())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Error("destination owner graph leaked", b.Used())
		}
	})
	serviceMust(t, runtime.SetMaxSize(1))
	active = true
	body := c.ToOwnedRational(&input, s, &out)
	active = false
	if callbacks == 0 || body == nil {
		t.Fatal("returning transport not reached", callbacks, body)
	}
	if !errors.Is(closeErr, ErrAlgebraicPending) || !errors.Is(coreCleanupErr, ErrAlgebraicPending) || !errors.Is(wrapperCleanupErr, ErrAlgebraicPending) || rationalCloseErr == nil {
		t.Fatal("executing wrapper or borrowed core destroyed", closeErr, coreCleanupErr, wrapperCleanupErr, rationalCloseErr)
	}
	if out.Raw.Num != unumeral.NewInt(99) || c.closed || c.toOwnedRational.constructed {
		t.Fatal("failed constructor published output or lost core", out)
	}
}

func TestAlgebraicToOwnedRationalEveryReservation(t *testing.T) {
	for _, owner := range []string{"raw", "service"} {
		t.Run(owner, func(t *testing.T) {
			success, failed := false, false
			constructorFailure, assignmentFailure := false, false
			for prefix := int64(0); prefix < 256 && !success; prefix++ {
				t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
					c, _, _ := algebraicCompareFixture(t)
					s, sb := serviceNew(t)
					q := c.config.Rationals
					z := q.Integers()
					borrowed := unumeral.Rational{Num: z.Two64, Den: unumeral.NewInt(1)}
					var input Algebraic
					serviceMust(t, c.SetRational(&input, &borrowed))
					var outputs [32]unumeral.OwnedRational
					for i := range outputs {
						var e error
						outputs[i], e = s.NewValue()
						serviceMust(t, e)
						serviceMust(t, s.SetInt64(&outputs[i], 99))
					}
					budget := z.Budget()
					if owner == "service" {
						budget = sb
					}
					budget.FailAfter(prefix)
					var body error
					completed := 0
					for i := range outputs {
						body = c.ToOwnedRational(&input, s, &outputs[i])
						if body != nil {
							if !c.toOwnedRational.constructed {
								if owner == "service" {
									constructorFailure = true
								}
								if outputs[i].Raw.Num != unumeral.NewInt(99) || outputs[i].Raw.Den != unumeral.NewInt(1) {
									t.Fatal("caller assignment started before constructor completion", outputs[i])
								}
							} else {
								assignmentFailure = true
							}
							break
						}
						completed++
					}
					budget.FailAfter(-1)
					if body != nil {
						failed = true
						if !numericPhysicalReservationRefusal(body) {
							t.Fatal("unexpected refusal", body)
						}
						before := c.toOwnedRational.tmp
						if e := c.ToOwnedRational(nil, nil, nil); !errors.Is(e, ErrAlgebraicPending) || before != c.toOwnedRational.tmp {
							t.Fatal("pending wrapper accepted replacement", e)
						}
					}
					if prefix%2 == 0 {
						serviceMust(t, c.CleanupRetained())
					} else {
						serviceMust(t, c.CleanupToOwnedRational())
					}
					serviceMust(t, c.CleanupToOwnedRational())
					for i := 0; i < completed; i++ {
						got, e := s.String(&outputs[i])
						serviceMust(t, e)
						if got != "18446744073709551616" {
							t.Fatal("completed output lost its independent value", got)
						}
					}
					for i := range outputs {
						serviceMust(t, s.CleanupValue(&outputs[i]))
					}
					algebraicCellSourcePhysical(t, c, nil, nil, nil, &input)
					var roots []*unumeral.Int
					for i := range s.Cache {
						roots = append(roots, &s.Cache[i].Num, &s.Cache[i].Den)
					}
					integerPhysicalAudit(t, s.Manager.Z, roots...)
					serviceMust(t, c.Del(&input))
					success = body == nil
				})
				if t.Failed() {
					return
				}
			}
			if !success || !failed {
				t.Fatal("did not cover failed and complete conversions", failed, success)
			}
			if owner == "service" && (!constructorFailure || !assignmentFailure) {
				t.Fatal("missing actual constructor/assignment refusal", constructorFailure, assignmentFailure)
			}
		})
	}
}

func TestAlgebraicToOwnedRationalIrrationalAndForeignOutput(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	s, _ := serviceNew(t)
	foreign, _ := serviceNew(t)
	out, e := foreign.NewValue()
	serviceMust(t, e)
	serviceMust(t, foreign.SetInt64(&out, 99))
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	polynomialAssertionActionsFixture(t, "T")
	if e := c.ToOwnedRational(&a, s, &out); e == nil || c.toOwnedRational.constructed || !c.toOwnedRational.tmpDone {
		t.Fatal("irrational conversion constructed global temporary or skipped raw destructor", e)
	}
	serviceMust(t, c.CleanupToOwnedRational())
	serviceMust(t, c.Del(&a))
	var zero Algebraic
	if e := c.ToOwnedRational(&zero, s, &out); !errors.Is(e, unumeral.ErrRationalServiceOwner) || !c.toOwnedRational.constructed || !c.toOwnedRational.tmp2Done || !c.toOwnedRational.tmpDone {
		t.Fatal("foreign assignment skipped reached local destructors", e)
	}
	serviceMust(t, c.CleanupToOwnedRational())
	if out.Raw.Num != unumeral.NewInt(99) || out.Raw.Den != unumeral.NewInt(1) {
		t.Fatal("failed conversion changed foreign output", out)
	}
	serviceMust(t, foreign.Del(&out))
}

func TestAlgebraicToOwnedRationalNonminimalEveryReservation(t *testing.T) {
	success, failed := false, false
	for prefix := int64(0); prefix < 1024 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			s, _ := serviceNew(t)
			root := algebraicCompareRoot(t, c, []int32{-1, 2, 3}, 0, 0, 1, 0, false)
			original := root
			out, e := s.NewValue()
			serviceMust(t, e)
			serviceMust(t, s.SetInt64(&out, 99))
			budget := c.config.Rationals.Integers().Budget()
			budget.FailAfter(prefix)
			body := c.ToOwnedRational(&root, s, &out)
			budget.FailAfter(-1)
			if body != nil {
				failed = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected rationality/conversion failure", body)
				}
				if out.Raw.Num != unumeral.NewInt(99) || out.Raw.Den != unumeral.NewInt(1) {
					t.Fatal("predicate failure published caller output", out)
				}
				if root == (Algebraic{}) {
					// Source deletes the old algebraic cell before allocating its
					// basic replacement (cpp:434-436). No rollback is promised.
					l := &c.locations[original.slot-1]
					if l.kind != 0 || l.generation == original.generation || l.published {
						t.Fatal("lost input without the actual basic replacement attempt")
					}
				}
			}
			serviceMust(t, c.CleanupRetained())
			if c.predicate.active || c.predicateQ.active || c.toOwnedRational.active {
				t.Fatal("core cleanup retained reached conversion child")
			}
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &root)
			if body != nil {
				if root == (Algebraic{}) {
					root = algebraicCompareRoot(t, c, []int32{-1, 2, 3}, 0, 0, 1, 0, false)
				}
				serviceMust(t, c.ToOwnedRational(&root, s, &out))
			}
			got, e := s.String(&out)
			serviceMust(t, e)
			if got != "1/3" {
				t.Fatal("retry or original conversion lost exact root", got)
			}
			serviceMust(t, s.Del(&out))
			serviceMust(t, c.Del(&root))
			success = body == nil
		})
		if t.Failed() {
			return
		}
	}
	if !success || !failed {
		t.Fatal("missing actual predicate failure or success", failed, success)
	}
}
