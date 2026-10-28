package sat

// Source: src/sat/sat_solver.cpp:2859-3017; src/util/sstream.h.

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func satInspectionStreamAudit(t *testing.T, s *Solver, o *utext.NumericTextStream, b *arena.Budget) {
	t.Helper()
	q := o.Stats()
	if sum := q.OwnerBytes + q.ScratchBytes + q.OutputOwnerBytes + q.OutputOverflowBytes; sum != b.Used() {
		t.Fatal("actual borrowed output owner", sum, b.Used())
	}
	satFullSolverRuntimeAudit(t, s)
}
func satInspectionLarge(t *testing.T) *Solver {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 24; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	for i := uint32(0); i < 21; i++ {
		_, e := s.Lifecycle.MkClauseCore(3, []Literal{Literal(i * 2), Literal((i + 1) * 2), Literal((i + 2) * 2)}, i&1 != 0)
		satOK(t, e)
	}
	return s
}
func TestSATSolverInspectionEveryActualStreamRefusalPrefix(t *testing.T) {
	for route := 0; route < 5; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			call := func(s *Solver, o *utext.NumericTextStream) error {
				switch route {
				case 0:
					return s.displayInto(o)
				case 1:
					return s.displayDIMACS(o)
				case 2:
					return s.displayWCNF(o, 2, []Literal{1, 4}, []uint32{2, 7})
				case 3:
					return s.displayWatches(o)
				default:
					return s.displayAssignment(o)
				}
			}
			horizon := int64(0)
			baseline := ""
			t.Run("calibrate", func(t *testing.T) {
				s := satInspectionLarge(t)
				b := arena.NewBudget(math.MaxUint64)
				o := satInspectionStream(t, b)
				const n = int64(1 << 50)
				b.FailAfter(n)
				satOK(t, call(s, o))
				horizon = n - testFieldBySourceName(reflect.ValueOf(b).Elem(), "remaining").Int()
				satInspectionStreamAudit(t, s, o, b)
				baseline, _ = o.String()
				b.FailAfter(-1)
			})
			failed, success := int64(0), 0
			for fault := int64(0); fault <= horizon; fault++ {
				t.Run(fmt.Sprint(fault), func(t *testing.T) {
					s := satInspectionLarge(t)
					b := arena.NewBudget(math.MaxUint64)
					o := satInspectionStream(t, b)
					solverBytes := s.budget.Used()
					b.FailAfter(fault)
					e := call(s, o)
					satInspectionStreamAudit(t, s, o, b)
					text, x := o.String()
					satOK(t, x)
					if !strings.HasPrefix(baseline, text) {
						t.Fatal("source emitted prefix", text, baseline)
					}
					if s.budget.Used() != solverBytes {
						t.Fatal("printing changed borrowed solver")
					}
					if e != nil {
						if !errors.Is(e, arena.ErrInjectedFailure) {
							t.Fatal(e)
						}
						failed++
					} else {
						if text != baseline {
							t.Fatal(text, baseline)
						}
						success++
					}
					b.FailAfter(-1)
				})
			}
			if failed != horizon || success != 1 {
				t.Fatal("exact finite refusal sweep", failed, success, horizon)
			}
			t.Log("actual output reservations", horizon)
		})
	}
}
func TestSATSolverInspectionWarmOutputAndOwnerGuards(t *testing.T) {
	s := satInspectionLarge(t)
	b := arena.NewBudget(math.MaxUint64)
	o := satInspectionStream(t, b)
	satOK(t, s.displayDIMACS(o))
	before := b.Used()
	solver := s.budget.Used()
	for i := 0; i < 10; i++ {
		satOK(t, o.ResetText())
		b.FailAfter(0)
		satOK(t, s.displayDIMACS(o))
		satInspectionStreamAudit(t, s, o, b)
		if b.Used() != before || s.budget.Used() != solver {
			t.Fatal("warm physical plateau")
		}
		b.FailAfter(-1)
	}
	copy := *s
	if e := copy.displayInto(o); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if _, e := copy.checkModelSpan(satSolverModelSpan{}); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if e := s.displayWatches(o); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	satOK(t, o.Close())
}

// Native output growth must not consume Runtime allocation reservations.
func TestSATSolverInspectionBinaryOutputNativeGrowth(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 96; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	for i := uint32(0); i < 95; i++ {
		_, e := s.Lifecycle.MkClauseCore(2, []Literal{Literal(i * 2), Literal((i + 1) * 2)}, i&1 != 0)
		satOK(t, e)
	}
	before := s.budget.Used()
	runtimeBefore := s.runtime.memory.Stats()
	s.budget.FailAfter(0)
	defer s.budget.FailAfter(-1)
	out, e := s.collectBinClauses(nil, true)
	satOK(t, e)
	if len(out) != 95 {
		t.Fatal(len(out))
	}
	for i, pair := range out {
		if pair != newSATSolverBinPair(Literal(2*i), Literal(2*(i+1))) {
			t.Fatal("watch traversal order", i, pair)
		}
	}
	if s.budget.Used() != before || s.runtime.memory.Stats() != runtimeBefore {
		t.Fatal("Go output acquired a simulated Runtime allocation")
	}
	satFullSolverRuntimeAudit(t, s)
}
