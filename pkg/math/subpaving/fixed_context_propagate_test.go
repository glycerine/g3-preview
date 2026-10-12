package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedContextQueuedDefinitionWatchers(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	coefficient, err := m.New()
	must(err)
	zero, err := m.New()
	must(err)
	v, err := m.New()
	must(err)
	must(m.SetInt64(coefficient, 2))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefinePolynomial(zero, []FixedPolynomialTerm{{Variable: 0, Coefficient: coefficient}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 1))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(m.SetInt64(v, 2))
	must(ctx.PropagateBound(root, 0, v, false, false))
	must(ctx.PropagateQueued(root))
	view, err := NewFixedInterval(m)
	must(err)
	must(view.SetConstant(root, x))
	var out bytes.Buffer
	must(view.Display(&out))
	if got := out.String(); got != "[2, 4]" {
		t.Fatalf("watched polynomial bounds = %s, want [2, 4]", got)
	}
	if len(ctx.queue) != 0 || ctx.queueHead != 0 || ctx.numVisited != 2 || ctx.polynomials[x].visited != 4 {
		t.Fatalf("queue/visited state: queue=%d head=%d visits=%d timestamp=%d", len(ctx.queue), ctx.queueHead, ctx.numVisited, ctx.polynomials[x].visited)
	}
	must(view.Close())
	must(ctx.Close())
	must(v.Close())
	must(zero.Close())
	must(coefficient.Close())
	must(m.Close())
}

func TestSubpavingFixedContextQueuedCancellationResumes(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	v, err := m.New()
	must(err)
	limit := &control.Reslimit{Limit: 1}
	ctx, err := NewFixedContext(m, limit, []bool{false}, 0, 2, 8192)
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 1))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(m.SetInt64(v, 2))
	must(ctx.PropagateBound(root, 0, v, false, false))
	if err := ctx.PropagateQueued(root); !errors.Is(err, ErrFixedContextCanceled) {
		t.Fatalf("first queue drain = %v, want cancellation", err)
	}
	if ctx.queueHead != 1 || len(ctx.queue) != 2 || limit.Steps != 2 {
		t.Fatalf("canceled queue state: head=%d queue=%d steps=%d", ctx.queueHead, len(ctx.queue), limit.Steps)
	}
	limit.Limit = 100
	must(ctx.PropagateQueued(root))
	if ctx.queueHead != 0 || len(ctx.queue) != 0 || limit.Steps != 3 {
		t.Fatalf("resumed queue state: head=%d queue=%d steps=%d", ctx.queueHead, len(ctx.queue), limit.Steps)
	}
	must(ctx.Close())
	must(v.Close())
	must(m.Close())
}

func TestSubpavingFixedContextInitialDefinitionSweep(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	v, err := m.New()
	must(err)
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefineMonomial([]FixedPower{{Variable: 0, Degree: 2}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 2))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(m.SetInt64(v, 3))
	must(ctx.PropagateBound(root, 0, v, false, false))
	must(ctx.PropagateAllDefinitions(root))
	if ctx.numVisited != 1 || root.lower[x] == nil || root.upper[x] == nil || len(ctx.queue) != 4 {
		t.Fatalf("initial definition sweep: visits=%d output=%v/%v queue=%d", ctx.numVisited, root.lower[x], root.upper[x], len(ctx.queue))
	}
	must(ctx.PropagateQueued(root))
	if len(ctx.queue) != 0 || root.inconsistent {
		t.Fatalf("post-sweep queue=%d conflict=%v", len(ctx.queue), root.inconsistent)
	}
	must(ctx.Close())
	must(v.Close())
	must(m.Close())
}
