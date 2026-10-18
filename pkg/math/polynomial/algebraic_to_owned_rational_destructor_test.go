package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicToOwnedRationalRawDestructorStop(t *testing.T) {
	algebraicOwnedRationalDestructorStopCase(t, false, false)
}

func TestAlgebraicToOwnedRationalGlobalDestructorStop(t *testing.T) {
	algebraicOwnedRationalDestructorStopCase(t, true, false)
}

func TestAlgebraicToOwnedRationalRawDenominatorDestructorStop(t *testing.T) {
	algebraicOwnedRationalDestructorStopCase(t, false, true)
}

func TestAlgebraicToOwnedRationalGlobalDenominatorDestructorStop(t *testing.T) {
	algebraicOwnedRationalDestructorStopCase(t, true, true)
}

func algebraicOwnedRationalDestructorStopCase(t *testing.T, globalDestructor, denominator bool) {
	algebraicOwnedRationalDestructorStopOperation(t, globalDestructor, denominator, 0)
}

func TestAlgebraicOwnedBoundOrderedDestructorStops(t *testing.T) {
	for _, operation := range []uint8{3, 4} {
		for _, global := range []bool{false, true} {
			for _, denominator := range []bool{false, true} {
				t.Run(strconv.Itoa(int(operation))+"/"+strconv.FormatBool(global)+"/"+strconv.FormatBool(denominator), func(t *testing.T) {
					algebraicOwnedRationalDestructorStopOperation(t, global, denominator, operation)
				})
			}
		}
	}
}

func algebraicOwnedRationalDestructorStopOperation(t *testing.T, globalDestructor, denominator bool, operation uint8) {
	reached := false
	for prefix := int64(0); prefix < 128 && !reached; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, rawRuntime := algebraicNegTLSFixture(t, nil)
			q := c.config.Rationals
			budget := arena.NewBudget(1 << 26)
			armed := false
			var lane umemory.Block
			var destinationRuntime *umemory.Runtime
			poisonedRuntime := rawRuntime
			destinationRuntime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, Transport: func(f umemory.Failure) error {
				if armed && c.toOwnedRational.constructed {
					armed = false
					reached = true
					budget.FailAfter(-1)
					// Public source-runtime operations reach the signed-host guard
					// on the later raw temporary deletion. No flags are fabricated.
					contextTLSNearMinimum(t, poisonedRuntime)
					var e error
					lane, e = poisonedRuntime.Allocate(8)
					serviceMust(t, e)
					delta := int64(-1000000)
					if e := poisonedRuntime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
						t.Fatal(e)
					}
				}
				return f
			}})
			serviceMust(t, e)
			if globalDestructor {
				poisonedRuntime = destinationRuntime
			}
			cfg := serviceConfig(budget)
			cfg.Integers.Memory = destinationRuntime
			global, e := unumeral.NewRationalService(cfg)
			serviceMust(t, e)
			value := unumeral.NewRational()
			words := make([]uint32, 2000)
			words[0], words[len(words)-1] = 17, 1
			big := &value.Num
			if denominator {
				value.Num = unumeral.NewInt(7)
				big = &value.Den
			}
			serviceMust(t, q.Integers().SetLimbs(big, words))
			var input Algebraic
			serviceMust(t, c.SetRational(&input, &value))
			out, e := global.NewValue()
			serviceMust(t, e)
			serviceMust(t, global.SetInt64(&out, 99))
			budget.FailAfter(prefix)
			armed = true
			var body error
			switch operation {
			case 3:
				body = c.GetLowerOwnedRationalAtPrecision(&input, global, &out, 2)
			case 4:
				body = c.GetUpperOwnedRationalAtPrecision(&input, global, &out, 2)
			default:
				body = c.ToOwnedRational(&input, global, &out)
			}
			armed = false
			budget.FailAfter(-1)
			if reached {
				s := &c.toOwnedRational
				if !errors.Is(body, arena.ErrBounds) || !s.active || !s.constructed || s.tmp2Done == globalDestructor || s.tmpDone || s.sourceCause == nil {
					t.Fatal("raw destructor stop lost ordered local state", body, s.constructed, s.tmp2Done, s.tmpDone, s.sourceCause)
				}
				cause, original := s.sourceCause, s.bodyCause
				if !errors.Is(body, original) || original == nil {
					t.Fatal("destructor stop replaced assignment failure", body, original)
				}
				rawBig, globalBig := &s.tmp.Num, &s.tmp2.Raw.Num
				if denominator {
					rawBig, globalBig = &s.tmp.Den, &s.tmp2.Raw.Den
				}
				if globalDestructor {
					if !global.Manager.Z.Cell(globalBig.Cell).Released || q.Integers().Cell(rawBig.Cell).Released {
						t.Fatal("global stop crossed the older raw temporary destructor")
					}
				} else if !q.Integers().Cell(rawBig.Cell).Released {
					t.Fatal("raw source stop did not consume actual backing")
				}
				before, stats := s.tmp, rawRuntime.Stats()
				if e := c.ToOwnedRational(nil, nil, nil); !errors.Is(e, cause) || s.tmp != before || rawRuntime.Stats() != stats {
					t.Fatal("stopped source suffix replayed or replaced", e)
				}
				if e := c.CleanupToRational(); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("raw entrypoint acquired the active wrapper's child", e)
				}
			}
			callerPrefix := out
			contextTLSFinish(t, poisonedRuntime, c.CleanupRetained)
			if out != callerPrefix {
				t.Fatal("wrapper owning cleanup consumed caller assignment prefix")
			}
			serviceMust(t, global.CleanupValue(&out))
			contextTLSFinish(t, rawRuntime, func() error { return c.Del(&input) })
			contextTLSFinish(t, rawRuntime, func() error { return q.Del(&value) })
			if lane != (umemory.Block{}) {
				contextTLSFinish(t, poisonedRuntime, func() error {
					if lane == (umemory.Block{}) {
						return nil
					}
					return poisonedRuntime.Deallocate(&lane)
				})
			}
			contextTLSFinish(t, destinationRuntime, global.Close)
			contextTLSFinish(t, destinationRuntime, destinationRuntime.Close)
			if budget.Used() != 0 {
				t.Fatal("destination graph leaked", budget.Used())
			}
		})
		if t.Failed() {
			return
		}
	}
	if !reached {
		t.Fatal("no actual assignment failure reached")
	}
}
