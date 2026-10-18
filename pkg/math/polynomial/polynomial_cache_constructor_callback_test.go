package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type polynomialCacheConstructorProbe struct {
	t       *testing.T
	manager *MultivariatePolynomialManager
	frame   *PolynomialCache
	called  bool
}

func (w *polynomialCacheConstructorProbe) Write(p []byte) (int, error) {
	if w.called {
		return len(p), nil
	}
	w.called = true
	m := w.manager
	w.frame = m.core.cacheCaller
	if w.frame == nil || !w.frame.executing || w.frame.manager != m {
		w.t.Error("cache constructor trace has no actual borrowed manager scope")
		return len(p), nil
	}
	stage, used := m.core.sourceStage, m.originalBudget.Used()
	for _, e := range []error{m.SourceFinalize(), m.CloseChecked(), m.core.SourceFinalizePrefix(), m.core.CloseChecked(), w.frame.Close(), w.frame.Reset(), w.frame.CleanupReset()} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Error("constructor trace acquired a borrowed lifetime", e)
		}
	}
	if stage != m.core.sourceStage || m.core.sourceStarted || used != m.originalBudget.Used() {
		w.t.Error("constructor trace advanced destruction or changed charged storage")
	}
	return len(p), nil
}

func TestPolynomialCacheConstructorTraceProtectsBorrowedManager(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	w := polynomialCacheConstructorProbe{t: t, manager: m}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("memory")
	m.core.monomials.config.Trace = tags
	c, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.Close()) })
	if w.called != diagnostic.DebugBuild {
		t.Fatal("constructor trace compile gate", w.called)
	}
	if diagnostic.DebugBuild && w.frame != c {
		t.Fatal("constructor guarded a different cache owner")
	}
	if m.core.cacheCaller != nil || c.previous != nil || c.executing || c.imp == nil {
		t.Fatal("constructor did not restore borrowed execution scope")
	}
}
