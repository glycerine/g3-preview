// Source: src/sat/sat_solver.cpp:2859-3017; src/util/sstream.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"strings"
	"testing"
)

func satInspectionStreamAudit(t *testing.T, s *satSolver, o *NumericTextStream, b *arena.Budget) {
	t.Helper()
	q := o.Stats()
	if sum := q.OwnerBytes + q.ScratchBytes + q.OutputOwnerBytes + q.OutputOverflowBytes; sum != b.Used() {
		t.Fatal("actual borrowed output owner", sum, b.Used())
	}
	satFullSolverRuntimeAudit(t, s)
}
func satInspectionLarge(t *testing.T) *satSolver {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 24; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	for i := uint32(0); i < 21; i++ {
		_, e := s.lifecycle.mkClauseCore(3, []satLiteral{satLiteral(i * 2), satLiteral((i + 1) * 2), satLiteral((i + 2) * 2)}, i&1 != 0)
		satOK(t, e)
	}
	return s
}
func TestSATSolverInspectionEveryActualStreamRefusalPrefix(t *testing.T) {
	for route := 0; route < 5; route++ {
		t.Run(fmt.Sprint(route), func(t *testing.T) {
			call := func(s *satSolver, o *NumericTextStream) error {
				switch route {
				case 0:
					return s.displayInto(o)
				case 1:
					return s.displayDIMACS(o)
				case 2:
					return s.displayWCNF(o, 2, []satLiteral{1, 4}, []uint32{2, 7})
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
				horizon = n - reflect.ValueOf(b).Elem().FieldByName("remaining").Int()
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

func TestSATSolverInspectionBinaryOutputEveryActualReservation(t *testing.T) {
	makeSolver := func(t *testing.T) *satSolver {
		s, _, _ := satSolverCheckFixture(t)
		for i := 0; i < 24; i++ {
			_, e := s.MkVar(false, true)
			satOK(t, e)
		}
		for i := uint32(0); i < 23; i++ {
			_, e := s.lifecycle.mkClauseCore(2, []satLiteral{satLiteral(i * 2), satLiteral((i + 1) * 2)}, i&1 != 0)
			satOK(t, e)
		}
		return s
	}
	h := int64(0)
	t.Run("calibrate", func(t *testing.T) {
		s := makeSolver(t)
		const n = int64(1 << 50)
		s.budget.FailAfter(n)
		satOK(t, s.collectBinClauses(s.userBinClauses, true))
		h = n - reflect.ValueOf(s.budget).Elem().FieldByName("remaining").Int()
		satFullSolverRuntimeAudit(t, s)
		s.budget.FailAfter(-1)
	})
	fail, ok := int64(0), 0
	for fault := int64(0); fault <= h; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s := makeSolver(t)
			s.budget.FailAfter(fault)
			e := s.collectBinClauses(s.userBinClauses, true)
			satFullSolverRuntimeAudit(t, s)
			n, x := s.userBinClauses.Size()
			satOK(t, x)
			for i := uint32(0); i < n; i++ {
				pair, x := s.userBinClauses.Get(i)
				satOK(t, x)
				if pair.first != 2*i || pair.second != 2*(i+1) {
					t.Fatal("completed owning append prefix", i, pair)
				}
			}
			if e != nil {
				var mf memory.Failure
				if !errors.Is(e, arena.ErrInjectedFailure) && !(errors.As(e, &mf) && mf.ErrorCode() == 101 && s.runtime.memory.State().OutOfMemory) {
					t.Fatal(e)
				}
				fail++
			} else {
				if n != 23 {
					t.Fatal(n)
				}
				ok++
			}
			s.budget.FailAfter(-1)
		})
	}
	if fail != h || ok != 1 {
		t.Fatal(fail, ok, h)
	}
	t.Log("binary actual reservations", h)
}
