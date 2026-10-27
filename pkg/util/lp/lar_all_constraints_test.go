// Source: src/util/lp/lar_solver.h:1054-1067.
package lp

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestLARAllConstraintsIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 40; world++ {
		for _, kind := range []LConstraintKind{LConstraintLE, LConstraintLT, LConstraintGE, LConstraintGT, LConstraintEQ} {
			for _, offset := range []int64{-1, 0, 1} {
				t.Run(fmt.Sprintf("%d/%d/%d", world, kind, offset), func(t *testing.T) {
					h := newLPPairHarness(t)
					shift, scale := big.NewRat(int64(world-19), 7), big.NewRat(int64(1+world%11), 13)
					pairs := [][2]*big.Rat{{shift, scale}, {new(big.Rat).Add(shift, new(big.Rat).Mul(scale, big.NewRat(1, 2))), new(big.Rat)}, {new(big.Rat).Add(shift, new(big.Rat).Mul(scale, big.NewRat(1, 4))), new(big.Rat)}}
					s := lpLARGetModelFixture(t, h, pairs)
					value := new(big.Rat).Add(shift, new(big.Rat).Mul(scale, big.NewRat(1, 8)))
					rhs := new(big.Rat).Add(value, big.NewRat(offset, 29))
					for _, bound := range []struct {
						kind LConstraintKind
						rhs  *big.Rat
					}{{LConstraintGE, new(big.Rat).Sub(value, big.NewRat(1, 1))}, {LConstraintLE, new(big.Rat).Add(value, big.NewRat(1, 1))}, {kind, rhs}} {
						c, err := NewLARVarConstraint(h.m, 0, bound.kind, &h.pair(bound.rhs.RatString(), "0").X)
						h.must(err)
						s.Constraints = append(s.Constraints, c)
					}
					want := false
					switch kind {
					case LConstraintLE:
						want = offset >= 0
					case LConstraintLT:
						want = offset > 0
					case LConstraintGE:
						want = offset <= 0
					case LConstraintGT:
						want = offset < 0
					case LConstraintEQ:
						want = offset == 0
					}
					if !want {
						c, err := NewLARVarConstraint(h.m, 0, LConstraintEQ, &h.pair("0", "0").X)
						h.must(err)
						c.stopped = true // Must not inspect past the first failed relation.
						s.Constraints = append(s.Constraints, c)
					}
					limit := &lpExactSolveCancelSequence{values: []bool{false, true}}
					s.Settings.SetResourceLimit(limit)
					var holds bool
					var err error
					output := lpCaptureFeasibleStdout(t, func() { holds, err = s.AllConstraintsHold() })
					h.must(err)
					if holds != want || limit.calls != 1 || s.validationModel != nil || s.validationCoefficients != nil || s.rationalWork != nil {
						t.Fatal("source relation/cancellation/local ownership", holds, want, limit.calls)
					}
					text := ""
					if !want {
						symbol, err := LConstraintKindString(kind)
						h.must(err)
						text = "v17 " + symbol + " " + rhs.RatString() + "\n"
					}
					if output != text {
						t.Fatal("first violated source constraint", output, text)
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
		}
	}
}

func TestLARAllConstraintsCancellationAndReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"empty", "cancel", "cancel-error", "stopped", "delta-error", "constraint-error"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			var pairs [][2]*big.Rat
			if phase != "empty" {
				pairs = [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}}
			}
			s := lpLARGetModelFixture(t, h, pairs)
			matrix := s.Core.RA
			limit := &lpTestResourceLimit{}
			s.Settings.SetResourceLimit(limit)
			want, wantHolds, reads := error(nil), true, 1
			switch phase {
			case "cancel":
				limit.cancel = true
				s.Bounds.Status.Value = StatusInfeasible
				s.Core.RA = nil // Initial cancellation bypasses extraction entirely.
			case "cancel-error":
				limit.err = lpErrExactSolveCancel
				want, wantHolds = lpErrExactSolveCancel, false
			case "stopped":
				s.stopped = true
				want, wantHolds, reads = ErrExactVectorPending, false, 0
			case "delta-error":
				s.Bounds.Types.values = nil
				want, wantHolds = ErrBounds, false
			case "constraint-error":
				c, err := NewLARVarConstraint(h.m, 0, LConstraintGE, &h.pair("0", "0").X)
				h.must(err)
				c.stopped = true
				s.Constraints = append(s.Constraints, c)
				want, wantHolds = ErrExactVectorPending, false
			}
			holds, err := s.AllConstraintsHold()
			s.Core.RA = matrix
			if err != want || holds != wantHolds || limit.calls != reads {
				t.Fatal("source entry prefix", err, want, holds, wantHolds, limit.calls)
			}
			if phase == "constraint-error" && (s.validationModel == nil || len(s.validationModel.Values) != 1) {
				t.Fatal("model remains owned after child evaluation failure")
			}
			if err != nil {
				if _, err = s.AllConstraintsHold(); err != ErrExactVectorPending {
					t.Fatal("stopped retry", err)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARAllConstraintsPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, violated := range []bool{false, true} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 2000 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%v/%v/%d", direct, violated, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					x := mustLPRat(t, strings.Repeat("1234567890", 20)+"/7")
					y := mustLPRat(t, strings.Repeat("9876543210", 20)+"/13")
					s := lpLARGetModelFixture(t, h, [][2]*big.Rat{{x, y}})
					rhs := new(big.Rat).Add(x, new(big.Rat).Mul(big.NewRat(1, 2), y))
					margin := int64(-1)
					if violated {
						margin = 1
					}
					rhs.Add(rhs, big.NewRat(margin, 1))
					c, err := NewLARVarConstraint(h.m, 0, LConstraintGE, &h.pair(rhs.RatString(), "0").X)
					h.must(err)
					s.Constraints = append(s.Constraints, c)
					var holds bool
					h.budget.FailAfter(ordinal)
					output := lpCaptureFeasibleStdout(t, func() { holds, err = s.AllConstraintsHold() })
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
						wantText := ""
						if violated {
							wantText = "v17 >= " + rhs.RatString() + "\n"
						}
						if holds == violated || output != wantText {
							t.Fatal("independent whole-model relation/diagnostic", holds, output, wantText)
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if _, err = s.AllConstraintsHold(); err != ErrExactVectorPending {
							t.Fatal("stopped retry", err)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || direct && failures == 0 {
				t.Fatal("whole-model allocation campaign", direct, violated, failures)
			}
		}
	}
}

func TestLARAllConstraintsClosedStdoutPrefix(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpLARGetModelFixture(t, h, [][2]*big.Rat{{new(big.Rat), big.NewRat(1, 1)}})
	c, err := NewLARVarConstraint(h.m, 0, LConstraintGE, &h.pair("1", "0").X)
	h.must(err)
	s.Constraints = append(s.Constraints, c)
	r, w, err := os.Pipe()
	h.must(err)
	h.must(w.Close())
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old; r.Close() }()
	holds, err := s.AllConstraintsHold()
	os.Stdout = old
	if holds || !errors.Is(err, os.ErrClosed) || s.validationModel == nil || s.rationalWork == nil {
		t.Fatal("failed diagnostic retains model and printed coefficient local", holds, err)
	}
	if _, err = s.AllConstraintsHold(); err != ErrExactVectorPending {
		t.Fatal("stopped diagnostic retry", err)
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARAllConstraintsPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARModelDeltaFixture(t, h, ColumnLowBound)
			h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("1/13", "0")))
			h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "123456789012345678901/7")))
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if _, err = s.AllConstraintsHold(); err == nil || s.calls.comparison != 0 || s.Core.calls.comparison != 0 {
					t.Fatal("adopted older caller comparison", err)
				}
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed older caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual older caller comparison")
	}
}
