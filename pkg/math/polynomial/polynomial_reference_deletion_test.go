package polynomial

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type polynomialDeletionReferenceState struct {
	target                       Polynomial
	reference                    *PolynomialRef
	public, internal, assignment error
	stop                         error
	calls                        int
	peers                        []Polynomial
	grew                         bool
	repeatDecrement              bool
	publicDrop, internalDrop     error
	dropOutcome                  polynomialDecrementOutcome
}

var polynomialDeletionReferences *polynomialDeletionReferenceState

func polynomialDeletionReferenceCall(_ *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
	s := polynomialDeletionReferences
	if s == nil || p != s.target {
		return nil
	}
	s.calls++
	capacity := cap(m.locations)
	for i := uint32(1); i <= 32; i++ {
		peer, e := m.MkVariable(2, i)
		if e != nil {
			return e
		}
		if e := m.IncRef(peer); e != nil {
			return e
		}
		s.peers = append(s.peers, peer)
	}
	s.grew = cap(m.locations) > capacity
	if s.repeatDecrement {
		s.publicDrop = m.DecRef(p)
		s.internalDrop = m.decRefCoreOutcome(p, &s.dropOutcome)
	}
	s.public = m.IncRef(p)
	s.internal = m.incRefCore(p)
	s.assignment = s.reference.Assign(p)
	return s.stop
}

type polynomialDeletionReferenceWriter struct {
	output          bytes.Buffer
	core            *MultivariatePolynomialCore
	target          Polynomial
	called          bool
	beforeHandlers  bool
	increment       error
	repeatDecrement bool
	decrement       error
	closeDebug      *diagnostic.DebugContext
}

func (w *polynomialDeletionReferenceWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		l, e := w.core.polynomialLocation(w.target)
		if e != nil {
			return 0, e
		}
		w.beforeHandlers = !l.callbackStarted
		if w.closeDebug != nil {
			if e := w.closeDebug.Close(); e != nil {
				return 0, e
			}
		}
		w.increment = w.core.IncRef(w.target)
		if w.repeatDecrement {
			w.decrement = w.core.DecRef(w.target)
		}
	}
	return w.output.Write(p)
}

func TestPolynomialDeletingValueCannotAcquireReferences(t *testing.T) {
	polynomialDeletionReferenceCase(t, false)
}

func TestPolynomialDeletingValueCannotRepeatDecrement(t *testing.T) {
	polynomialDeletionReferenceCase(t, true)
}

func polynomialDeletionReferenceCase(t *testing.T, repeatDecrement bool) {
	for _, traced := range []bool{false, true} {
		name := "handler"
		if traced {
			name = "trace-and-handler"
		}
		t.Run(name, func(t *testing.T) {
			m := polynomialCellFixture(t)
			p, e := m.MkVariable(0, 2)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(p))
			peer, e := m.MkVariable(1, 1)
			serviceMust(t, e)
			r, e := NewPolynomialRef(m)
			serviceMust(t, e)
			serviceMust(t, r.Assign(peer))
			state := &polynomialDeletionReferenceState{target: p, reference: r, stop: errors.New("exact deletion stop"), repeatDecrement: repeatDecrement}
			polynomialDeletionReferences = state
			h, e := NewPolynomialDeletionHandler(m.integer.Budget(), polynomialDeletionReferenceCall)
			serviceMust(t, e)
			serviceMust(t, m.AddDeletionHandler(h))
			w := &polynomialDeletionReferenceWriter{core: m, target: p, repeatDecrement: repeatDecrement}
			tags := diagnostic.NewTraceTagsWithWriter(w)
			if traced {
				tags.Enable("polynomial")
			}
			m.monomials.config.Trace = tags
			t.Cleanup(func() {
				tags.Disable("polynomial")
				polynomialDeletionReferences = nil
				serviceMust(t, r.Close())
				for _, peer := range state.peers {
					serviceMust(t, m.DecRef(peer))
				}
				serviceMust(t, m.CleanupValue(&p))
				serviceMust(t, m.RemoveDeletionHandler(h))
				serviceMust(t, h.CloseChecked())
			})
			var original polynomialDecrementOutcome
			var body error
			if repeatDecrement {
				body = m.decRefCoreOutcome(p, &original)
			} else {
				body = m.DecRef(p)
			}
			if !errors.Is(body, state.stop) || state.calls != 1 {
				t.Fatal("actual last-reference deletion did not retain its original stop", body, state.calls)
			}
			if repeatDecrement {
				if !original.consumed || !original.last || !errors.Is(state.publicDrop, ErrPolynomialPending) || !errors.Is(state.internalDrop, ErrPolynomialPending) || state.dropOutcome != (polynomialDecrementOutcome{}) {
					t.Fatal("reentered decrement changed RC or acquired a second deletion outcome", original, state.publicDrop, state.internalDrop, state.dropOutcome)
				}
				if tags.ShouldTrace("polynomial") && !errors.Is(w.decrement, ErrPolynomialPending) {
					t.Fatal("trace writer repeated the accepted decrement", w.decrement)
				}
			}
			if !state.grew || len(state.peers) != 32 {
				t.Fatal("deletion callback did not acquire independent peers and grow the real directory", state.grew, len(state.peers))
			}
			for _, e := range []error{state.public, state.internal, state.assignment} {
				if !errors.Is(e, ErrPolynomialPending) {
					t.Fatal("deleting polynomial acquired a new reference", state.public, state.internal, state.assignment)
				}
			}
			if tags.ShouldTrace("polynomial") && (!w.called || !w.beforeHandlers || !errors.Is(w.increment, ErrPolynomialPending)) {
				t.Fatal("earlier trace boundary resurrected polynomial", w.called, w.beforeHandlers, w.increment)
			}
			info, e := m.Info(p)
			serviceMust(t, e)
			got, e := r.Get()
			serviceMust(t, e)
			if info.RefCount != 0 || got != peer || r.root.incoming != (Polynomial{}) || r.root.assigning {
				t.Fatal("refused acquisition changed RC or the older owning reference", info, got, peer)
			}
			if e := m.IncRef(p); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("stopped deletion was treated as a fresh zero-reference result", e)
			}
		})
	}
}

func TestPolynomialFreshZeroReferenceAfterDeletionReuse(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 2)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	fresh, e := m.MkVariable(1, 3)
	serviceMust(t, e)
	if fresh.slot != p.slot || fresh.generation == p.generation {
		t.Fatal("fixture did not reach actual descriptor reuse", p, fresh)
	}
	info, e := m.Info(fresh)
	serviceMust(t, e)
	if info.RefCount != 0 {
		t.Fatal("factory did not return its real zero-reference result", info)
	}
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, r.Assign(fresh))
	serviceMust(t, m.IncRef(fresh))
	info, e = m.Info(fresh)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("old deletion state blocked a fresh result or missed an INC", info)
	}
	serviceMust(t, m.DecRef(fresh))
	serviceMust(t, r.Close())
}

func TestPolynomialReferenceOwnsDecrementBeforeDeletionTraceStop(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("requires source debug operand guard")
	}
	m := polynomialCellFixture(t)
	w := &polynomialDeletionReferenceWriter{core: m}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	if !tags.ShouldTrace("polynomial") {
		t.Skip("requires compiled deletion trace")
	}
	previous := diagnostic.BorrowProcessDebugContext()
	polynomialAssertionActionsFixture(t, "T")
	w.closeDebug = diagnostic.BorrowProcessDebugContext()
	p, e := m.MkVariable(0, 2)
	serviceMust(t, e)
	w.target = p
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	serviceMust(t, r.Assign(p))
	m.monomials.config.Trace = tags
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		tags.Disable("polynomial")
		serviceMust(t, m.CleanupDisplay())
		serviceMust(t, r.Close())
	})
	body := r.Close()
	_, e = diagnostic.BindProcessDebugContext(previous)
	serviceMust(t, e)
	tags.Disable("polynomial")
	l, e := m.polynomialLocation(p)
	serviceMust(t, e)
	if !errors.Is(body, arena.ErrClosed) || !w.called || !w.beforeHandlers || l.callbackStarted || l.deleteStage != 0 {
		t.Fatal("source trace did not stop before handler/stage publication", body, w.called, w.beforeHandlers, l.callbackStarted, l.deleteStage)
	}
	if !r.root.decremented || r.root.value != p {
		t.Fatal("accepted decrement lost its owning continuation before handlers", r.root.decremented, r.root.value)
	}
	serviceMust(t, m.CleanupDisplay())
	serviceMust(t, r.Close())
}
