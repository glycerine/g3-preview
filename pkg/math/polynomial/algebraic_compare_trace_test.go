package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strings"
	"testing"
)

func TestAlgebraicCompareTraceSourceIntervalsAndFailbit(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	var basic, zero Algebraic
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 1, 2))
	serviceMust(t, c.SetRational(&basic, &q))
	defer func() {
		serviceMust(t, c.config.Rationals.BindTrace(nil))
		serviceMust(t, c.Del(&root))
		serviceMust(t, c.Del(&basic))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	order, e := c.Compare(&zero, &basic)
	serviceMust(t, e)
	if order != -1 {
		t.Fatal("zero/half order", order)
	}
	order, e = c.Compare(&root, &basic)
	serviceMust(t, e)
	if order != 1 {
		t.Fatal("root/half order", order)
	}
	if tags.ShouldTrace("algebraic") {
		for _, body := range []string{"comparing: [0, 0] [1/2, 1/2]\n", "comparing: (1, 2) [1/2, 1/2]\n"} {
			if !strings.Contains(text.String(), body) {
				t.Fatal("missing source interval trace", body, text.String())
			}
		}
	} else if text.Len() != 0 {
		t.Fatal("erased comparison TRACE emitted", text.String())
	}
	fail := &linearTraceFailWriter{}
	tags = diagnostic.NewTraceTagsWithWriter(fail)
	tags.Enable("algebraic")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	order, e = c.Compare(&root, &basic)
	serviceMust(t, e)
	if order != 1 || c.comparisonTraceRunning {
		t.Fatal("failed output changed comparison", order)
	}
	want := 0
	if tags.ShouldTrace("algebraic") {
		want = 1
	}
	if fail.calls != want {
		t.Fatal("sticky trace stream retry", fail.calls, want)
	}
}

func TestAlgebraicCompareTraceExecutingCallbackKeepsOwner(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	var zero Algebraic
	output := &linearOperationProbeWriter{probe: func() {
		before := c.config.Rationals.Budget().Used()
		actions := []func() error{c.Close, c.CleanupRetained, c.CleanupIsolation, c.CleanupNegation, func() error { return c.Del(&zero) }, func() error { return c.CleanupValue(&zero) }, func() error { return c.Swap(&zero, &zero) }, func() error { return c.DisplayInterval(io.Discard, &zero) }, func() error { _, e := c.Compare(&zero, &zero); return e }}
		for i, action := range actions {
			if e := action(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatalf("callback action %d: %v", i, e)
			}
		}
		if c.closed || c.closing || !c.comparisonTraceRunning || before != c.config.Rationals.Budget().Used() {
			t.Fatal("callback changed active owner")
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic")
	if !tags.ShouldTrace("algebraic") {
		t.Skip("source TRACE erased")
	}
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	defer func() { serviceMust(t, c.config.Rationals.BindTrace(nil)) }()
	order, e := c.Compare(&zero, &zero)
	serviceMust(t, e)
	if order != 0 || !output.called || c.comparisonTraceRunning {
		t.Fatal("comparison did not resume", order)
	}
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&zero, &one))
	serviceMust(t, c.Del(&zero))
}

func TestAlgebraicManagerCompareTraceRefusalPreservesCleanupState(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.Close()); serviceMust(t, cfg.Rationals.Manager.BindTrace(nil)) }()
	var zero Algebraic
	output := &linearOperationProbeWriter{probe: func() {
		cause := m.cleanupCause
		if e := m.Close(); !errors.Is(e, ErrAlgebraicPending) || m.finishing || m.closed || m.cleanupCause != cause {
			t.Fatal("refused outer destructor changed cleanup state", e)
		}
	}}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	tags.Enable("algebraic")
	if !tags.ShouldTrace("algebraic") {
		t.Skip("source TRACE erased")
	}
	serviceMust(t, cfg.Rationals.Manager.BindTrace(tags))
	order, e := m.Compare(&zero, &zero)
	serviceMust(t, e)
	if order != 0 || !output.called {
		t.Fatal("owning manager comparison did not resume", order)
	}
}
