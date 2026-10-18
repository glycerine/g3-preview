package polynomial

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type polynomialDeletionCleanupWriter struct {
	output          bytes.Buffer
	core            *MultivariatePolynomialCore
	target, after   Polynomial
	called          bool
	cleanup, source error
	stage           uint8
	peer            Polynomial
	peerCleanup     error
}

func (w *polynomialDeletionCleanupWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		candidate := w.target
		w.cleanup = w.core.CleanupValue(&candidate)
		w.after = candidate
		w.source = w.core.deleteCore(w.target)
		w.peerCleanup = w.core.CleanupValue(&w.peer)
		if l, e := w.core.polynomialLocation(w.target); e == nil {
			w.stage = l.deleteStage
		}
	}
	return w.output.Write(p)
}

func TestPolynomialDeletionTraceCannotCleanupExecutingValue(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 2)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	peer, e := m.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(peer))
	w := &polynomialDeletionCleanupWriter{core: m, target: p, peer: peer}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	m.monomials.config.Trace = tags
	t.Cleanup(func() {
		tags.Disable("polynomial")
		if w.peer != (Polynomial{}) {
			serviceMust(t, m.CleanupValue(&w.peer))
		}
		if _, e := m.polynomialLocation(p); e == nil {
			serviceMust(t, m.CleanupValue(&p))
		}
	})
	serviceMust(t, m.DecRef(p))
	if tags.ShouldTrace("polynomial") {
		if !w.called || !errors.Is(w.cleanup, ErrPolynomialPending) || !errors.Is(w.source, ErrPolynomialPending) || w.after != p || w.stage != 0 {
			t.Fatal("trace cleanup consumed or changed the executing deletion", w.called, w.cleanup, w.source, w.after, w.stage)
		}
		serviceMust(t, w.peerCleanup)
		if w.peer != (Polynomial{}) {
			t.Fatal("executing deletion blocked an independent owned peer cleanup")
		}
	} else if w.called {
		t.Fatal("erased trace reached writer")
	}
	if _, e := m.polynomialLocation(p); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("original source deletion did not finish", e)
	}
}
