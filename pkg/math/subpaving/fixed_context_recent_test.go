package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedContextRecentChildBoundsAndWatchers(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	one, err := m.New()
	must(err)
	zero, err := m.New()
	must(err)
	v, err := m.New()
	must(err)
	must(m.SetInt64(one, 1))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false, false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefinePolynomial(zero, []FixedPolynomialTerm{{0, one}, {1, one}})
	must(err)
	root, err := ctx.Root()
	must(err)
	bound := func(node *FixedNode, variable uint32, value int64, lower bool, decided bool) {
		t.Helper()
		must(m.SetInt64(v, value))
		if decided {
			must(ctx.DecideBound(node, variable, v, lower, false))
		} else {
			must(ctx.PropagateBound(node, variable, v, lower, false))
		}
	}
	bound(root, 0, 0, true, false)
	bound(root, 0, 10, false, false)
	bound(root, 1, 1, true, false)
	bound(root, 1, 3, false, false)
	must(ctx.PropagateQueued(root))
	child, err := ctx.NewChild(root)
	must(err)
	bound(child, 0, 5, false, true)
	bound(child, 0, 4, false, true) // supersedes the previous child upper
	bound(child, 1, 2, true, true)
	if len(ctx.queue) != 0 {
		t.Fatalf("decided child bounds entered queue: %d", len(ctx.queue))
	}
	must(ctx.AddRecentBounds(child))
	if len(ctx.queue) != 2 || ctx.queue[0] != child.lower[1] || ctx.queue[1] != child.upper[0] ||
		ctx.queue[0].timestamp != ctx.timestamp || ctx.queue[1].timestamp != ctx.timestamp {
		t.Fatalf("recent bound order/timestamps: queue=%v timestamp=%d", ctx.queue, ctx.timestamp)
	}
	if err := ctx.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("replay into nonempty queue = %v", err)
	}
	must(ctx.PropagateQueued(child))
	view, err := NewFixedInterval(m)
	must(err)
	must(view.SetConstant(child, x))
	var out bytes.Buffer
	must(view.Display(&out))
	if got := out.String(); got != "[2, 7]" {
		t.Fatalf("child polynomial = %s, want [2, 7]", got)
	}
	must(view.SetConstant(root, x))
	out.Reset()
	must(view.Display(&out))
	if got := out.String(); got != "[1, 13]" {
		t.Fatalf("root polynomial changed with child: %s", got)
	}
	must(view.Close())
	must(child.Close())
	must(ctx.Close())
	must(v.Close())
	must(zero.Close())
	must(one.Close())
	must(m.Close())
}

func TestSubpavingFixedContextRecentChildRejectsChangedParentTrail(t *testing.T) {
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
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	must(err)
	root, err := ctx.Root()
	must(err)
	child, err := ctx.NewChild(root)
	must(err)
	must(m.SetInt64(v, 1))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(ctx.PropagateQueued(root))
	must(m.SetInt64(v, 2))
	must(ctx.DecideBound(child, 0, v, false, false))
	if child.trail.reason != (fixedBoundReason{kind: fixedReasonAxiom}) {
		t.Fatal("source branch axiom justification", child.trail.reason)
	}
	stamp := child.trail.timestamp
	if err := ctx.AddRecentBounds(child); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("changed parent trail replay = %v", err)
	}
	if len(ctx.queue) != 0 || child.trail.timestamp != stamp {
		t.Fatalf("failed replay changed queue or timestamp: queue=%d timestamp=%d", len(ctx.queue), child.trail.timestamp)
	}
	must(child.Close())
	must(ctx.Close())
	must(v.Close())
	must(m.Close())
}
