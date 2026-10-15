package subpaving

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedMidpointSourceCasesAndNoInterior(t *testing.T) {
	for _, tc := range []struct {
		name         string
		lower, upper *int64
		leftOpen     bool
		wantMid      int64
	}{
		{name: "unbounded", leftOpen: true, wantMid: 0},
		{name: "upper-only", upper: splitInt64(10), leftOpen: true, wantMid: 8},
		{name: "lower-only", lower: splitInt64(4), leftOpen: false, wantMid: 6},
		{name: "both", lower: splitInt64(0), upper: splitInt64(10), leftOpen: true, wantMid: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			if tc.lower != nil {
				must(m.SetInt64(v, *tc.lower))
				must(ctx.PropagateBound(root, 0, v, true, false))
			}
			if tc.upper != nil {
				must(m.SetInt64(v, *tc.upper))
				must(ctx.PropagateBound(root, 0, v, false, false))
			}
			must(ctx.PropagateQueued(root))
			left, right, err := ctx.SplitMidpoint(root, 0, tc.leftOpen, 2)
			must(err)
			if left.upper[0] == nil || right.lower[0] == nil || len(ctx.queue) != 0 {
				t.Fatalf("midpoint child publication: left=%v right=%v queue=%d", left.upper[0], right.lower[0], len(ctx.queue))
			}
			lo, err := m.Rational(left.upper[0].value)
			must(err)
			hi, err := m.Rational(right.lower[0].value)
			must(err)
			want := big.NewRat(tc.wantMid, 1)
			if lo.Cmp(want) != 0 || hi.Cmp(want) != 0 || left.upper[0].open != tc.leftOpen || right.lower[0].open == tc.leftOpen {
				t.Fatalf("midpoint %s/%s openness %v/%v, want %s openness %v/%v", lo, hi, left.upper[0].open, right.lower[0].open, want, tc.leftOpen, !tc.leftOpen)
			}
			must(left.Close())
			must(right.Close())
			must(ctx.Close())
			must(v.Close())
			must(m.Close())
		})
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ctx.Root()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(v, 0); err != nil {
		t.Fatal(err)
	}
	if err := ctx.PropagateBound(root, 0, v, true, false); err != nil {
		t.Fatal(err)
	}
	if err := m.SetRatio(v, 1, 1<<32); err != nil {
		t.Fatal(err)
	}
	if err := ctx.PropagateBound(root, 0, v, false, false); err != nil {
		t.Fatal(err)
	}
	if err := ctx.PropagateQueued(root); err != nil {
		t.Fatal(err)
	}
	left, right, err := ctx.SplitMidpoint(root, 0, true, 1)
	if !errors.Is(err, ErrFixedSplitNoInterior) || !errors.Is(err, ErrSubpavingFailure) || left == nil || right == nil || root.children != 2 || left.trail != root.trail || right.trail != root.trail {
		t.Fatalf("unrepresentable midpoint: left=%v right=%v children=%d err=%v", left, right, root.children, err)
	}
	if err := right.Close(); err != nil {
		t.Fatal(err)
	}
	if err := left.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func splitInt64(n int64) *int64 { return &n }

func TestSubpavingFixedMidpointChildrenPropagateDefinition(t *testing.T) {
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
	zero, err := m.New()
	must(err)
	two, err := m.New()
	must(err)
	must(m.SetInt64(two, 2))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefinePolynomial(zero, []FixedPolynomialTerm{{Variable: 0, Coefficient: two}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 0))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(m.SetInt64(v, 10))
	must(ctx.PropagateBound(root, 0, v, false, false))
	must(ctx.PropagateQueued(root))
	left, right, err := ctx.SplitMidpoint(root, 0, true, 1)
	must(err)
	must(ctx.AddRecentBounds(left))
	must(ctx.PropagateQueued(left))
	must(ctx.AddRecentBounds(right))
	must(ctx.PropagateQueued(right))
	if left.upper[x] == nil || right.lower[x] == nil {
		t.Fatal("split children did not propagate polynomial bounds")
	}
	upper, err := m.Rational(left.upper[x].value)
	must(err)
	lower, err := m.Rational(right.lower[x].value)
	must(err)
	if upper.Cmp(big.NewRat(10, 1)) != 0 || lower.Cmp(big.NewRat(10, 1)) != 0 || !left.upper[x].open || right.lower[x].open {
		t.Fatalf("split polynomial output: left upper=%s open=%v, right lower=%s open=%v", upper, left.upper[x].open, lower, right.lower[x].open)
	}
	must(left.Close())
	must(right.Close())
	must(ctx.Close())
	must(two.Close())
	must(zero.Close())
	must(v.Close())
	must(m.Close())
}

func TestSubpavingFixedMidpointIntegerChildNormalization(t *testing.T) {
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
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{true}, 0, 2, 8192)
	must(err)
	root, err := ctx.Root()
	must(err)
	must(m.SetInt64(v, 0))
	must(ctx.PropagateBound(root, 0, v, true, false))
	must(m.SetInt64(v, 3))
	must(ctx.PropagateBound(root, 0, v, false, false))
	must(ctx.PropagateQueued(root))
	left, right, err := ctx.SplitMidpoint(root, 0, true, 1)
	must(err)
	upper, err := m.Rational(left.upper[0].value)
	must(err)
	lower, err := m.Rational(right.lower[0].value)
	must(err)
	if upper.Cmp(big.NewRat(1, 1)) != 0 || lower.Cmp(big.NewRat(2, 1)) != 0 || left.upper[0].open || right.lower[0].open {
		t.Fatalf("integer split bounds: left upper=%s open=%v, right lower=%s open=%v", upper, left.upper[0].open, lower, right.lower[0].open)
	}
	must(left.Close())
	must(right.Close())
	must(ctx.Close())
	must(v.Close())
	must(m.Close())
}
