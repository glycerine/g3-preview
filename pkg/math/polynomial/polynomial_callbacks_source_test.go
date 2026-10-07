// Source: src/math/polynomial/polynomial.cpp:2396-2440; src/math/polynomial/polynomial.h:226-240. Licensed under the MIT license.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Static unit dispatch, with a fixed test-owned observation object. It is not
// a process-global source binding or a success callback substituted for one.
var polynomialCallbackObservation struct {
	peers   [2]Polynomial
	values  [8]Polynomial
	trace   [8]uint64
	count   uint32
	remove  *PolynomialDeletionHandler
	failure error
}

func polynomialTestDeletionCall(h *PolynomialDeletionHandler, m *MultivariatePolynomialCore, p Polynomial) error {
	i := polynomialCallbackObservation.count
	if i >= 8 {
		panic("unexpected callback count")
	}
	polynomialCallbackObservation.trace[i] = h.State
	polynomialCallbackObservation.values[i] = p
	polynomialCallbackObservation.count++
	if h.State == 7 {
		for j := range polynomialCallbackObservation.peers {
			a, e := m.MkVariable(uint32(j), 1)
			if e != nil {
				return e
			}
			if e = m.IncRef(a); e != nil {
				return e
			}
			polynomialCallbackObservation.peers[j] = a
		}
		return nil
	}
	if h.State == 8 && i == 0 {
		return m.DecRef(polynomialCallbackObservation.peers[0])
	}
	if h.State == 6 {
		return polynomialCallbackObservation.failure
	}
	if _, e := m.Info(p); e != nil {
		return e
	}
	if h.State == 4 {
		if e := m.RemoveDeletionHandler(h); e != nil {
			return e
		}
		return h.CloseChecked()
	}
	if h.State == 3 && polynomialCallbackObservation.remove != nil {
		return m.RemoveDeletionHandler(polynomialCallbackObservation.remove)
	}
	return polynomialCallbackObservation.failure
}
func polynomialCallbackHandler(t *testing.T, m *MultivariatePolynomialCore, state uint64) *PolynomialDeletionHandler {
	h, e := NewPolynomialDeletionHandler(m.integer.Budget(), polynomialTestDeletionCall)
	serviceMust(t, e)
	h.State = state
	t.Cleanup(func() { serviceMust(t, h.CloseChecked()) })
	return h
}
func TestPolynomialDeletionSourceListMutation(t *testing.T) {
	m := polynomialCellFixture(t)
	polynomialCallbackObservation.count = 0
	polynomialCallbackObservation.failure = nil
	h1 := polynomialCallbackHandler(t, m, 1)
	h2 := polynomialCallbackHandler(t, m, 2)
	h3 := polynomialCallbackHandler(t, m, 3)
	serviceMust(t, m.AddDeletionHandler(h1))
	serviceMust(t, m.AddDeletionHandler(h2))
	serviceMust(t, m.AddDeletionHandler(h3))
	polynomialCallbackObservation.remove = h2
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	serviceMust(t, m.DecRef(p))
	if polynomialCallbackObservation.count != 2 || polynomialCallbackObservation.trace[0] != 3 || polynomialCallbackObservation.trace[1] != 1 {
		t.Fatal(polynomialCallbackObservation)
	}
	serviceMust(t, m.RemoveDeletionHandler(h3))
	serviceMust(t, m.RemoveDeletionHandler(h1))
	polynomialCallbackObservation.remove = nil
}
func TestPolynomialDeletionCallbackBeforeSize(t *testing.T) {
	m := polynomialCellFixture(t)
	h := polynomialCallbackHandler(t, m, 1)
	serviceMust(t, m.AddDeletionHandler(h))
	polynomialCallbackObservation.count = 0
	polynomialCallbackObservation.remove = nil
	stop := errors.New("actual test handler refusal")
	polynomialCallbackObservation.failure = stop
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	// The reached callback must win over the genuinely malformed later SIZE.
	serviceMust(t, m.writeWord(p, 8, 2))
	if e = m.DecRef(p); e != stop {
		t.Fatal("callback must precede size access", e)
	}
	if polynomialCallbackObservation.count != 1 {
		t.Fatal("handler not reached")
	}
	// Restore only the deliberately malformed scalar after checking the prefix;
	// owning abort does not replay the completed/refused source callback.
	serviceMust(t, m.writeWord(p, 8, 1))
	polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
	polynomialCallbackObservation.failure = nil
	serviceMust(t, m.CleanupValue(&p))
	if p != (Polynomial{}) || polynomialCallbackObservation.count != 1 {
		t.Fatal("abort replayed callback", p)
	}
	serviceMust(t, m.RemoveDeletionHandler(h))
}
func TestPolynomialDeletionHandlerBorrowLifetime(t *testing.T) {
	m := polynomialCellFixture(t)
	h := polynomialCallbackHandler(t, m, 1)
	serviceMust(t, m.AddDeletionHandler(h))
	if e := h.CloseChecked(); e != ErrPolynomialLive {
		t.Fatal("registered handler is borrowed", e)
	}
	copy := *h
	if e := copy.CloseChecked(); e != ErrPolynomialOwner {
		t.Fatal(e)
	}
	serviceMust(t, m.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

// Native current->next would read a destroyed handler. The Go guard belongs
// after its successful callback, with the real polynomial fields still owned.
func TestPolynomialDeletionCurrentLifetimeAtNextRead(t *testing.T) {
	m := polynomialCellFixture(t)
	h := polynomialCallbackHandler(t, m, 4)
	serviceMust(t, m.AddDeletionHandler(h))
	polynomialCallbackObservation.count = 0
	polynomialCallbackObservation.failure = nil
	polynomialCallbackObservation.remove = nil
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	if e = m.DecRef(p); e != arena.ErrClosed {
		t.Fatal("current lifetime guard", e)
	}
	l, e := m.polynomialLocation(p)
	serviceMust(t, e)
	if !l.callbackInvoked || l.deleteStage != 0 || l.initializedCoefficients != 1 || l.initializedMonomials != 1 || polynomialCallbackObservation.count != 1 {
		t.Fatal("completed callback prefix lost", l)
	}
	polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
	serviceMust(t, m.CleanupValue(&p))
	if polynomialCallbackObservation.count != 1 {
		t.Fatal("owning abort repeated handler")
	}
}

func TestPolynomialOwningCloseStoppedCallbacks(t *testing.T) {
	q, b := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: q.Integers().Memory})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { t.Fatal("unexpected private teardown checkpoint"); return false }, Cooperation: func() error { t.Fatal("unexpected private teardown cooperation"); return nil }})
	serviceMust(t, e)
	h, e := NewPolynomialDeletionHandler(b, polynomialTestDeletionCall)
	serviceMust(t, e)
	h.State = 6
	serviceMust(t, m.AddDeletionHandler(h))
	stop := errors.New("private source handler refusal")
	polynomialCallbackObservation.failure = stop
	polynomialCallbackObservation.count = 0
	polynomialCallbackObservation.remove = nil
	if e = m.CloseChecked(); e != stop {
		t.Fatal("first actual handler", e)
	}
	if !m.closing || m.closed {
		t.Fatal("stopped close owner guard")
	}
	polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
	// Explicit owning retries abort each refused private action, without changing
	// the dispatcher's failure or replaying an earlier completed handler.
	if e = m.CloseChecked(); e != stop {
		t.Fatal("second private handler", e)
	}
	polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
	serviceMust(t, m.CloseChecked())
	if polynomialCallbackObservation.count != 2 {
		t.Fatal("callback replay", polynomialCallbackObservation.count)
	}
	if m.live != 0 {
		t.Fatal("owning retry dropped live private cells", m.live)
	}
	serviceMust(t, h.CloseChecked())
	serviceMust(t, mm.DecManagerRef())
	polynomialCallbackObservation.failure = nil
}

func TestPolynomialDeletionPeerActionsAndDirectoryGrowth(t *testing.T) {
	for _, tag := range []uint64{7, 8} {
		t.Run(fmt.Sprint(tag), func(t *testing.T) {
			m := polynomialCellFixture(t)
			h := polynomialCallbackHandler(t, m, tag)
			serviceMust(t, m.AddDeletionHandler(h))
			polynomialCallbackObservation.count = 0
			polynomialCallbackObservation.failure = nil
			polynomialCallbackObservation.remove = nil
			polynomialCallbackObservation.peers = [2]Polynomial{}
			p, e := m.MkVariable(0, 1)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(p))
			if tag == 8 {
				peer, e := m.MkVariable(1, 1)
				serviceMust(t, e)
				serviceMust(t, m.IncRef(peer))
				polynomialCallbackObservation.peers[0] = peer
			}
			oldCapacity := cap(m.locations)
			serviceMust(t, m.DecRef(p))
			if _, e = m.polynomialLocation(p); e != ErrPolynomialOwner {
				t.Fatal("outer deletion did not consume real current", e)
			}
			serviceMust(t, m.RemoveDeletionHandler(h))
			if tag == 7 {
				if cap(m.locations) <= oldCapacity {
					t.Fatal("actual callback did not grow cold directory")
				}
				for _, peer := range polynomialCallbackObservation.peers {
					serviceMust(t, m.DecRef(peer))
				}
			} else {
				if polynomialCallbackObservation.count != 2 || polynomialCallbackObservation.values[0] != p || polynomialCallbackObservation.values[1] != polynomialCallbackObservation.peers[0] {
					t.Fatal("nested actual arguments/order", polynomialCallbackObservation)
				}
				if _, e = m.polynomialLocation(polynomialCallbackObservation.peers[0]); e != ErrPolynomialOwner {
					t.Fatal("peer source deletion", e)
				}
			}
			polynomialPhysicalAuditWithHandlers(t, m, []*PolynomialDeletionHandler{h})
		})
	}
}
