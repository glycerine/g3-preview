package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestUPolynomialSequenceSturmStoppedRemainderRetainsOlderPrime(t *testing.T) {
	for _, closeOwner := range []bool{false, true} {
		name := "cleanup-retry"
		if closeOwner {
			name = "close"
		}
		t.Run(name, func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			s, e := m.NewSequence()
			serviceMust(t, e)
			z := m.config.Integer
			reached := false
			var first, stopped unumeral.NumericCellRef
			var requested uint64
			m.config.Checkpoint = func() bool {
				q := &s.sturm
				if reached || !q.executing || !q.live[0] || q.remainder.size < 2 {
					return true
				}
				values := m.values(&q.remainder)
				first, stopped = values[0].Cell, values[1].Cell
				if !unumeral.ValidNumericRef(z.Cells, first) || !unumeral.ValidNumericRef(z.Cells, stopped) {
					return true
				}
				reached = true
				cell := z.Cell(stopped)
				requested = cell.Backing.Requested
				cell.Backing.Requested = uint64(^uint32(0)) + 1
				z.Store(stopped, cell)
				return false
			}
			m.config.Cooperation = func() error { return nil }
			body := m.SturmSequence(v, s)
			q := &s.sturm
			if !reached || !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, arena.ErrBounds) || q.sourceStop == nil || !q.active || q.executing || q.sourceCursor != ([2]uint32{1, 0}) || !q.live[0] || !q.live[1] || (unumeral.ValidNumericRef(z.Cells, first) && !z.Cell(first).Released) || !unumeral.ValidNumericRef(z.Cells, stopped) || z.Cell(stopped).Released {
				t.Fatal("source remainder stop lost body/older prime", body, q.sourceCursor, q.live)
			}
			for _, value := range m.values(&s.state.coefficients)[:s.state.coefficients.size] {
				if value.Cell != 0 && z.Cell(value.Cell).Released {
					t.Fatal("caller output coefficient destroyed")
				}
			}
			prior, used, stats := *q, z.Budget().Used(), z.RuntimeOwner().Stats()
			for _, e := range []error{s.Reset(), s.PushCopy(v), s.PushMove(v), m.SturmSequence(v, s), m.SturmSequencePair(v, v, s), m.SturmTarskiSequence(v, v, s), m.FourierSequence(v, s)} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					t.Fatal("mutation discarded stopped local", e)
				}
			}
			if e := q.finishVector(1, nil); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(prior, *q) || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
				t.Fatal("source finalizer/mutation replayed stopped scope")
			}
			cell := z.Cell(stopped)
			cell.Backing.Requested = requested
			z.Store(stopped, cell)
			source := q.sourceCursor
			if closeOwner {
				serviceMust(t, s.Close())
			} else {
				serviceMust(t, s.CleanupSturm())
			}
			if q.active || q.live != ([2]bool{}) || q.sourceCursor != source || q.ownCursor[0] != 3 || q.ownCursor[1] != 2 {
				t.Fatal("owning continuation changed source history", q.sourceCursor, q.ownCursor)
			}
			if !closeOwner {
				m.config.Checkpoint = func() bool { return true }
				serviceMust(t, m.SturmSequence(v, s))
				minus, e := m.SignVariationsMinusInfinity(s)
				serviceMust(t, e)
				plus, e := m.SignVariationsPlusInfinity(s)
				serviceMust(t, e)
				if minus-plus != 2 {
					t.Fatal("independent two real roots", minus, plus)
				}
				serviceMust(t, s.Close())
			}
			algebraicComparisonAudit(t, c, &root)
			serviceMust(t, c.Del(&root))
		})
	}
}

func TestUPolynomialSequenceExecutingSturmRejectsMutation(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{-2, 0, 1})
	v, e := m.wholeView(p)
	serviceMust(t, e)
	s, e := m.NewSequence()
	serviceMust(t, e)
	probed := false
	m.config.Checkpoint = func() bool {
		if probed {
			return true
		}
		probed = true
		used := m.config.Integer.Budget().Used()
		for _, e := range []error{s.Close(), s.CleanupSturm(), s.Reset(), s.PushCopy(v), s.PushMove(v), m.SturmSequence(v, s), m.FourierSequence(v, s)} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("checkpoint acquired Sturm local owner", e)
			}
		}
		if !s.sturm.executing || used != m.config.Integer.Budget().Used() {
			t.Fatal("checkpoint mutated executing scope")
		}
		return true
	}
	m.config.Cooperation = func() error { return nil }
	serviceMust(t, m.SturmSequence(v, s))
	if !probed || s.sturm.active || s.sturm.executing {
		t.Fatal("public Sturm body did not finish")
	}
	serviceMust(t, s.Close())
}
