package polynomial

import (
	"bytes"
	"errors"
	"testing"

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
	s.public = m.IncRef(p)
	s.internal = m.incRefCore(p)
	s.assignment = s.reference.Assign(p)
	return s.stop
}

type polynomialDeletionReferenceWriter struct {
	output         bytes.Buffer
	core           *MultivariatePolynomialCore
	target         Polynomial
	called         bool
	beforeHandlers bool
	increment      error
}

func (w *polynomialDeletionReferenceWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		l, e := w.core.polynomialLocation(w.target)
		if e != nil {
			return 0, e
		}
		w.beforeHandlers = !l.callbackStarted
		w.increment = w.core.IncRef(w.target)
	}
	return w.output.Write(p)
}

func TestPolynomialDeletingValueCannotAcquireReferences(t *testing.T) {
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
			state := &polynomialDeletionReferenceState{target: p, reference: r, stop: errors.New("exact deletion stop")}
			polynomialDeletionReferences = state
			h, e := NewPolynomialDeletionHandler(m.integer.Budget(), polynomialDeletionReferenceCall)
			serviceMust(t, e)
			serviceMust(t, m.AddDeletionHandler(h))
			w := &polynomialDeletionReferenceWriter{core: m, target: p}
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
			if e := m.DecRef(p); !errors.Is(e, state.stop) || state.calls != 1 {
				t.Fatal("actual last-reference deletion did not retain its original stop", e, state.calls)
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
