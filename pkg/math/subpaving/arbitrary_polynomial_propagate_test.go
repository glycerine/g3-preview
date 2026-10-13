package subpaving

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryPolynomialForwardInverseAndQueue(t *testing.T) {
	setup := func(t *testing.T) (*arbitraryClauseFixture, uint32, *ArbitraryNode) {
		t.Helper()
		f := newArbitraryClauseFixture(t)
		var two, three numeral.ArbitraryFloat
		if err := f.numbers.SetInt32(&two, 2); err != nil {
			t.Fatal(err)
		}
		if err := f.numbers.SetInt32(&three, 3); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := f.numbers.Del(&two); err != nil {
				t.Error(err)
			}
			if err := f.numbers.Del(&three); err != nil {
				t.Error(err)
			}
		})
		output, err := f.context.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{two, three}, []uint32{f.x, f.y})
		if err != nil {
			t.Fatal(err)
		}
		root, err := f.context.Root()
		if err != nil {
			t.Fatal(err)
		}
		return f, output, root
	}
	bound := func(t *testing.T, f *arbitraryClauseFixture, n *ArbitraryNode, variable uint32, lo, hi int32) {
		t.Helper()
		for _, entry := range []struct {
			value int32
			lower bool
		}{{lo, true}, {hi, false}} {
			var value numeral.ArbitraryFloat
			if err := f.numbers.SetInt32(&value, entry.value); err != nil {
				t.Fatal(err)
			}
			if err := f.context.PublishBound(n, variable, &value, entry.lower, false, true); err != nil {
				t.Fatal(err)
			}
			if err := f.numbers.Del(&value); err != nil {
				t.Fatal(err)
			}
		}
	}
	toRat := func(t *testing.T, f *arbitraryClauseFixture, b *ArbitraryBound) *big.Rat {
		t.Helper()
		if b == nil {
			t.Fatal("missing polynomial bound")
		}
		core, err := f.numbers.BorrowCore()
		if err != nil {
			t.Fatal(err)
		}
		value, err := core.ToDouble(&b.value)
		if err != nil {
			t.Fatal(err)
		}
		return new(big.Rat).SetFloat64(value)
	}
	t.Run("forward and repeated candidate", func(t *testing.T) {
		f, output, n := setup(t)
		bound(t, f, n, f.x, 1, 2)
		bound(t, f, n, f.y, 3, 4)
		visitTimestamp := f.context.timestamp
		if err := f.context.PropagatePolynomial(output, n); err != nil {
			t.Fatal(err)
		}
		if toRat(t, f, n.lower[output]).Cmp(big.NewRat(11, 1)) != 0 ||
			toRat(t, f, n.upper[output]).Cmp(big.NewRat(16, 1)) != 0 ||
			!n.lower[output].definitionReason || n.upper[output].reasonVariable != output ||
			f.context.polynomials[output].visited != visitTimestamp {
			t.Fatal("forward bounds or source definition reason/timestamp")
		}
		before := len(f.context.queue)
		if err := f.context.PropagatePolynomial(output, n); err != nil || len(f.context.queue) != before {
			t.Fatalf("repeated candidates should be irrelevant: %v, %d -> %d", err, before, len(f.context.queue))
		}
		if err := f.context.PropagateQueued(n); err != nil || len(f.context.queue) != 0 || f.context.queueHead != 0 {
			t.Fatalf("queued polynomial watches: %v, %d, %d", err, len(f.context.queue), f.context.queueHead)
		}
	})
	t.Run("inverse directed bounds", func(t *testing.T) {
		f, output, n := setup(t)
		bound(t, f, n, f.x, 1, 2)
		bound(t, f, n, output, 11, 16)
		if err := f.context.PropagatePolynomial(output, n); err != nil {
			t.Fatal(err)
		}
		if toRat(t, f, n.lower[f.y]).Cmp(big.NewRat(7, 3)) > 0 ||
			toRat(t, f, n.upper[f.y]).Cmp(big.NewRat(14, 3)) < 0 ||
			!n.lower[f.y].definitionReason || n.upper[f.y].reasonVariable != output {
			t.Fatal("inverse bounds must enclose exact rational target and retain definition reason")
		}
	})
	t.Run("queued watcher dispatch", func(t *testing.T) {
		f, output, n := setup(t)
		bound(t, f, n, f.x, 1, 2)
		bound(t, f, n, f.y, 3, 4)
		if err := f.context.PropagateQueued(n); err != nil {
			t.Fatal(err)
		}
		if toRat(t, f, n.lower[output]).Cmp(big.NewRat(11, 1)) != 0 ||
			toRat(t, f, n.upper[output]).Cmp(big.NewRat(16, 1)) != 0 ||
			len(f.context.queue) != 0 || f.context.queueHead != 0 || f.context.numVisited == 0 {
			t.Fatal("queued input bounds must dispatch and drain polynomial watchers")
		}
	})
	t.Run("two unbounded variables", func(t *testing.T) {
		f, output, n := setup(t)
		bound(t, f, n, f.x, 1, 2)
		before := len(f.context.queue)
		if err := f.context.PropagatePolynomial(output, n); err != nil {
			t.Fatal(err)
		}
		if len(f.context.queue) != before || n.lower[output] != nil || n.lower[f.y] != nil {
			t.Fatal("two unbounded polynomial variables cannot propagate")
		}
	})
}
