package subpaving

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedPolynomialForwardReverseAndSourceConstant(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	a, err := m.New()
	must(err)
	b, err := m.New()
	must(err)
	constant, err := m.New()
	must(err)
	value, err := m.New()
	must(err)
	must(m.SetInt64(a, 2))
	must(m.SetInt64(b, 3))
	must(m.SetInt64(constant, 5))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false, false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefinePolynomial(constant, []FixedPolynomialTerm{{0, a}, {1, b}})
	must(err)
	if x != 2 || ctx.isInt[x] || len(ctx.watchers[0]) != 1 || ctx.watchers[0][0] != x {
		t.Fatalf("definition metadata: x=%d integer=%v watchers=%v", x, ctx.isInt[x], ctx.watchers[0])
	}
	root, err := ctx.Root()
	must(err)
	bound := func(variable uint32, n int64, lower bool) {
		t.Helper()
		must(m.SetInt64(value, n))
		must(ctx.PropagateBound(root, variable, value, lower, false))
	}
	bound(0, 1, true)
	bound(0, 2, false)
	bound(1, 2, true)
	bound(1, 3, false)
	must(ctx.PropagatePolynomial(x, root))
	view, err := NewFixedInterval(m)
	must(err)
	must(view.SetConstant(root, x))
	var out bytes.Buffer
	must(view.Display(&out))
	if got := out.String(); got != "[8, 13]" {
		t.Fatalf("forward bounds = %s, want [8, 13] (source omits constant 5)", got)
	}
	if root.lower[x].reason != (fixedBoundReason{kind: fixedReasonVariableDefinition, variable: x}) ||
		root.upper[x].reason != (fixedBoundReason{kind: fixedReasonVariableDefinition, variable: x}) {
		t.Fatal("forward polynomial justification")
	}
	if len(ctx.queue) != 6 {
		t.Fatalf("forward queue length = %d, want 6", len(ctx.queue))
	}
	must(view.Close())
	must(ctx.Close())
	must(value.Close())
	must(constant.Close())
	must(a.Close())
	must(b.Close())
	must(m.Close())
}

func TestSubpavingFixedPolynomialSingleUnboundedAndDuplicates(t *testing.T) {
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
	two, err := m.New()
	must(err)
	three, err := m.New()
	must(err)
	value, err := m.New()
	must(err)
	must(m.SetInt64(one, 1))
	must(m.SetInt64(two, 2))
	must(m.SetInt64(three, 3))
	ctx, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, []bool{false, false}, 0, 2, 8192)
	must(err)
	x, err := ctx.DefinePolynomial(one, []FixedPolynomialTerm{{0, two}, {1, three}})
	must(err)
	root, err := ctx.Root()
	must(err)
	must(ctx.PropagatePolynomial(x, root))
	if len(ctx.queue) != 0 {
		t.Fatal("two or more unbounded variables propagated")
	}
	bound := func(variable uint32, n int64, lower bool) {
		t.Helper()
		must(m.SetInt64(value, n))
		must(ctx.PropagateBound(root, variable, value, lower, false))
	}
	bound(0, 1, true)
	bound(0, 2, false)
	bound(x, 8, true)
	bound(x, 13, false)
	must(ctx.PropagatePolynomial(x, root))
	if root.lower[1] == nil || root.upper[1] == nil {
		t.Fatal("single unbounded term did not receive both bounds")
	}
	lo, err := m.Rational(root.lower[1].value)
	must(err)
	hi, err := m.Rational(root.upper[1].value)
	must(err)
	if lo.Cmp(big.NewRat(4, 3)) > 0 || hi.Cmp(big.NewRat(11, 3)) < 0 {
		t.Fatalf("reverse interval [%s,%s] misses [4/3,11/3]", lo, hi)
	}
	dup, err := ctx.DefinePolynomial(one, []FixedPolynomialTerm{{0, two}, {0, three}})
	must(err)
	if len(ctx.polynomials[dup].terms) != 2 {
		t.Fatal("duplicate source terms collapsed")
	}
	first, err := m.Rational(ctx.polynomials[dup].terms[0].Coefficient)
	must(err)
	second, err := m.Rational(ctx.polynomials[dup].terms[1].Coefficient)
	must(err)
	if first.Cmp(big.NewRat(3, 1)) != 0 || second.Sign() != 0 {
		t.Fatalf("duplicate source swap coefficients = %s, %s", first, second)
	}
	must(ctx.Close())
	must(value.Close())
	must(three.Close())
	must(two.Close())
	must(one.Close())
	must(m.Close())
}
