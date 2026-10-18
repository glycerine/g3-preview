package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func sequenceSturmCopyRoute(m *UPolynomialManager, s *UPolynomialSequence, v, small UPolynomialView, route string) error {
	switch route {
	case "unary":
		return m.SturmSequence(v, s)
	case "pair":
		return m.SturmSequencePair(v, v, s)
	case "pair-second":
		return m.SturmSequencePair(small, v, s)
	case "tarski":
		return m.SturmTarskiSequence(v, v, s)
	default:
		return m.FourierSequence(v, s)
	}
}

func TestUPolynomialSequenceSturmInitialCopyExactOwnership(t *testing.T) {
	for _, route := range []string{"unary", "pair", "pair-second", "tarski", "fourier"} {
		t.Run(route, func(t *testing.T) {
			c, m, runtime := algebraicNegTLSFixture(t, nil)
			root := algebraicLargeDyadicPromotionRoot(t, c)
			l, e := c.location(&root)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			v, e := c.sourcePolynomial(&r)
			serviceMust(t, e)
			small := upolySourceNew(t, m, []int32{1})
			sv, e := m.wholeView(small)
			serviceMust(t, e)
			s, e := m.NewSequence()
			serviceMust(t, e)
			z := m.config.Integer
			pad := algebraicCellSourceTLSBoundary(t, c)
			body := sequenceSturmCopyRoute(m, s, v, sv, route)
			q := &s.sturm
			rows, coefficients := 1, 1
			if route == "pair-second" {
				rows, coefficients = 2, 2
			}
			if body == nil || !q.active || q.executing || !z.MatchesCopyAllocationInvocation(&q.copy) || len(s.state.sizes) != rows || int(s.state.coefficients.size) != coefficients || q.live != ([2]bool{}) {
				t.Fatal("initial sequence SET lost accepted receipt/source prefix", body, s.state.sizes, s.state.coefficients.size)
			}
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			source := q.sourceCursor
			serviceMust(t, s.CleanupSturm())
			if q.active || z.MatchesCopyAllocationInvocation(&q.copy) || q.sourceCursor != source {
				t.Fatal("copy acknowledgement changed source history")
			}
			// The failed invocation did not run its later source body. Start a new
			// actual construction, then check its first copied row independently.
			serviceMust(t, sequenceSturmCopyRoute(m, s, v, sv, route))
			row, e := s.RowView(0)
			serviceMust(t, e)
			expected := v
			if route == "pair-second" {
				expected = sv
			}
			equal, e := m.EqualViews(row, expected)
			serviceMust(t, e)
			if !equal {
				t.Fatal("retry changed preserved input row")
			}
			serviceMust(t, s.Close())
			algebraicCellSourcePhysical(t, c, []UPolynomial{small}, nil, nil, &root)
			serviceMust(t, c.Del(&root))
		})
	}
}

func TestUPolynomialSequenceSturmRefusedCopyPreservesOlderInvocation(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
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
	var old unumeral.IntegerCopyAllocationReceipt
	var dst unumeral.Int
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := z.CopyFromOwnerWithReceipt(&dst, z, &z.Two64, &old)
	if body == nil || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("older actual copy refusal not reached", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	before, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	body = m.SturmSequence(v, s)
	if body == nil || s.sturm.copy != (unumeral.IntegerCopyAllocationReceipt{}) || s.sturm.active {
		t.Fatal("refused constructor adopted older copy", body)
	}
	serviceMust(t, s.CleanupSturm())
	if !reflect.DeepEqual(before, z.CopyAllocation) || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("Sturm cleanup consumed independent copy")
	}
	serviceMust(t, s.Close())
	if !reflect.DeepEqual(before, z.CopyAllocation) || !z.MatchesCopyAllocationInvocation(&old) {
		t.Fatal("sequence Close consumed independent copy")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&old))
	serviceMust(t, z.Del(&dst))
	serviceMust(t, c.Del(&root))
}

func TestUPolynomialSequenceSturmCopyRejectsFailureTransportReentry(t *testing.T) {
	var m *UPolynomialManager
	var s *UPolynomialSequence
	var v UPolynomialView
	armed, probed := false, false
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		probed = true
		before, used, stats := s.Stats(), m.config.Integer.Budget().Used(), m.config.Integer.RuntimeOwner().Stats()
		if e := s.sturm.pushCopy(v); !errors.Is(e, ErrUPolynomialOwner) {
			t.Error("paused copy entry was not refused", e)
		}
		if before != s.Stats() || used != m.config.Integer.Budget().Used() || stats != m.config.Integer.RuntimeOwner().Stats() {
			t.Error("paused copy entry changed source vector/metadata")
		}
		return f
	}
	c, owner, runtime := algebraicNegTLSFixture(t, transport)
	m = owner
	root := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&root)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	v, e = c.sourcePolynomial(&r)
	serviceMust(t, e)
	s, e = m.NewSequence()
	serviceMust(t, e)
	pad := algebraicCellSourceTLSBoundary(t, c)
	armed = true
	body := m.SturmSequence(v, s)
	if body == nil || !probed {
		t.Fatal("actual copy failure transport not reached", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, s.CleanupSturm())
	serviceMust(t, s.Close())
	serviceMust(t, c.Del(&root))
}
