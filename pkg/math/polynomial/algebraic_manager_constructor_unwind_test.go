package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicManagerConstructorSourceEveryActualPrefix(t *testing.T) {
	for _, mode := range []string{"global", "private", "borrowed-allocator"} {
		t.Run(mode, func(t *testing.T) {
			succeeded := false
			sawFailedPolynomial, sawCompletedPolynomial, sawUP, sawPrivate, sawBody := false, false, false, false, false
			for ordinal := int64(0); ordinal < 512 && !succeeded; ordinal++ {
				if !t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					cfg, _, budget := algebraicManagerFixture(t)
					if mode == "private" {
						q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: serviceConfig(budget).Integers})
						serviceMust(t, e)
						t.Cleanup(func() { serviceMust(t, q.CloseChecked()) })
						cfg.Numerals = q
					}
					if mode == "borrowed-allocator" {
						a, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: cfg.Rationals.Manager.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: polynomialSourceDebug}})
						serviceMust(t, e)
						t.Cleanup(func() { serviceMust(t, a.Close()) })
						cfg.Allocator = a
					}
					warm, e := NewAlgebraicManager(cfg)
					serviceMust(t, e)
					serviceMust(t, warm.Close())
					baseline := budget.Used()
					budget.FailAfter(ordinal)
					m, body := NewAlgebraicManager(cfg)
					budget.FailAfter(-1)
					succeeded = body == nil
					if m == nil {
						if body == nil {
							t.Fatal("nil successful constructor")
						}
						return
					}
					if body == nil {
						serviceMust(t, m.Close())
						return
					}
					if !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected constructor body error", body)
					}
					if !m.unwindStarted || m.completed || m.closed || m.header == 0 || m.constructing || m.sourceRunning || m.sourceStarted || m.sourceFinalized || m.sourceStage != 0 || m.sourceIndex != 0 || m.sourceCause != m.unwindCause {
						t.Fatal("constructor unwind relabeled a completed owner", ordinal)
					}
					if m.ownAllocator != (cfg.Allocator == nil) {
						t.Fatal("outer allocator ownership assignment occurred after construction")
					}
					if m.unwindFinalized {
						if m.unwindStage != 14 || m.unwindIndex != 0 || m.unwindCause != nil {
							t.Fatal("source prefix did not end before outer allocator", m.unwindStage, m.unwindCause)
						}
					} else if m.unwindCause == nil || !numericPhysicalReservationRefusal(m.unwindCause) {
						t.Fatal("unexpected source-unwind stop", m.unwindStage, m.unwindCause)
					}
					if a := m.allocator; a != nil {
						_, e := a.NumFreeObjects()
						if errors.Is(e, arena.ErrClosed) {
							t.Fatal("uncompleted outer manager source-closed its raw allocator")
						}
					}
					if pm := m.polynomial; pm != nil {
						if !m.polynomialConstructed {
							sawFailedPolynomial = true
							if pm.sourceStarted || pm.sourceFinalized || pm.bodyStage != 0 {
								t.Fatal("failed child received a complete-object destructor")
							}
							if pm.unwindCause == nil && !m.unwindFinalized {
								t.Fatal("successful failed-child unwind blocked older source members", m.unwindCause)
							}
						} else {
							sawCompletedPolynomial = true
							if m.unwindStage > 11 && (!pm.sourceFinalized || pm.closed || pm.header == 0) {
								t.Fatal("completed polynomial member used owning close")
							}
						}
					}
					if up := m.univariate; up != nil {
						sawUP = true
						if m.unwindStage > 10 && (!up.sourceFinalized || up.closed || up.header == 0) {
							t.Fatal("completed UP member used owning close")
						}
					}
					if m.coreMembersReady {
						sawPrivate = true
						if m.params.Context != nil {
							sawBody = true
						}
						if fs := m.factors; fs != nil && m.unwindStage > 4 && (!fs.sourceFinalized || fs.closed || fs.header == 0) {
							t.Fatal("completed factors member used owning close")
						}
					}
					if m.unwindFinalized && m.binary != nil {
						if m.binary.HeaderBytes == 0 {
							t.Fatal("source unwind retired binary cold owner")
						}
						if e := m.binary.Open(); !errors.Is(e, arena.ErrClosed) {
							t.Fatal("binary source member did not end", e)
						}
					}
					if cfg.Limit.Count() != 0 {
						t.Fatal("constructor/unwind consumed a checkpoint")
					}
					stage, index, cause, done, original := m.unwindStage, m.unwindIndex, m.unwindCause, m.unwindFinalized, m.originalCause
					before, bytes := m.config.Numerals.Integers().RuntimeOwner().Stats(), budget.Used()
					if again := m.unwindConstructorSource(); again != cause || m.unwindStage != stage || m.unwindIndex != index || budget.Used() != bytes || m.config.Numerals.Integers().RuntimeOwner().Stats() != before {
						t.Fatal("constructor source prefix replayed", again, cause)
					}
					if e := m.SourceFinalize(); e == nil || m.sourceStarted {
						t.Fatal("failed constructor admitted normal destructor", e)
					}
					serviceMust(t, m.Close())
					if m.unwindStage != stage || m.unwindIndex != index || m.unwindCause != cause || m.unwindFinalized != done || m.originalCause != original || m.sourceCause != cause || m.sourceStarted || !m.closed {
						t.Fatal("owning cleanup rewrote constructor history")
					}
					if budget.Used() != baseline {
						t.Fatal("cold owning roots leaked", budget.Used(), baseline)
					}
					serviceMust(t, m.config.Numerals.CheckOwnerReady())
				}) {
					return
				}
			}
			if !succeeded || !sawFailedPolynomial || !sawCompletedPolynomial || !sawUP || !sawPrivate || !sawBody {
				t.Fatal("actual prefix horizon incomplete", succeeded, sawFailedPolynomial, sawCompletedPolynomial, sawUP, sawPrivate, sawBody)
			}
		})
	}
}

func TestAlgebraicManagerConstructorSourceClosedAssertionOwner(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion expressions erased")
	}
	cfg, _, _ := algebraicManagerFixture(t)
	var m *AlgebraicManager
	var stage uint8
	var index uint32
	var source, bodyCause error
	var finalized bool
	// The assertion fixture restores its actual process binding before the older
	// failed manager's owning cleanup runs. No descriptor or flag is repaired.
	t.Cleanup(func() {
		if m == nil {
			return
		}
		serviceMust(t, m.Close())
		if m.unwindStage != stage || m.unwindIndex != index || m.unwindCause != source || m.originalCause != bodyCause || m.unwindFinalized != finalized || m.sourceStarted {
			t.Fatal("owning cleanup rewrote assertion failure history")
		}
	})
	polynomialClosedCellAssertionContext(t)
	var body error
	m, body = NewAlgebraicManager(cfg)
	if body == nil || m == nil || !errors.Is(body, arena.ErrClosed) || !m.unwindStarted || m.completed || m.closed || m.sourceStarted {
		t.Fatal("actual assertion constructor failure missing", body)
	}
	stage, index, source, bodyCause, finalized = m.unwindStage, m.unwindIndex, m.unwindCause, m.originalCause, m.unwindFinalized
	before := cfg.Rationals.Manager.Integers().RuntimeOwner().Stats()
	if again := m.unwindConstructorSource(); again != source || cfg.Rationals.Manager.Integers().RuntimeOwner().Stats() != before {
		t.Fatal("assertion unwind replayed", again, source)
	}
	if m.polynomial != nil && !m.polynomialConstructed && (m.polynomial.sourceStarted || m.polynomial.bodyStage != 0) {
		t.Fatal("uncompleted child received normal source destructor")
	}
	if a := m.allocator; a != nil {
		_, e := a.NumFreeObjects()
		if errors.Is(e, arena.ErrClosed) {
			t.Fatal("failed outer constructor closed its raw allocator")
		}
	}
}
