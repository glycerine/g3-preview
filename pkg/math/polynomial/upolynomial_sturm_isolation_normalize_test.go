package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

// Each prefix uses a fresh real owner graph, preserving failed source scratch
// holes rather than repairing a manager and silently changing its next path.
func TestUPolynomialSturmIsolationNormalizeReachedReservations(t *testing.T) {
	success, failed := false, false
	for after := int64(0); after < 512 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			budget := m.config.Integer.Budget()
			z := m.config.Integer
			p := upolySourceNew(t, m, nil)
			var inputs [3]unumeral.Int
			for i, text := range []string{"18446744073709551616", "55340232221128654848", "36893488147419103232"} {
				serviceMust(t, z.SetString(&inputs[i], text))
			}
			serviceMust(t, m.SetInts(p, inputs[:]))
			for i := range inputs {
				serviceMust(t, z.Del(&inputs[i]))
			}
			v, e := m.wholeView(p)
			serviceMust(t, e)
			q, e := m.beginSturmIsolation(v, c.config.Binary)
			serviceMust(t, e)
			budget.FailAfter(after)
			body := q.normalizeView(v)
			budget.FailAfter(-1)
			body = q.finish(body)
			if body == nil {
				success = true
				for i, want := range []int32{1, 3, 2} {
					got, e := m.read(v, uint32(i))
					serviceMust(t, e)
					expected := unumeral.NewInt(want)
					equal, e := z.Equal(got, &expected)
					serviceMust(t, e)
					if !equal {
						t.Fatal("normalized coefficient", i, want)
					}
				}
			} else {
				failed = true
				var failure umemory.Failure
				if !errors.Is(body, arena.ErrBudget) && !errors.As(body, &failure) {
					t.Fatal(body)
				}
			}
			source := q.normalization.constructed
			serviceMust(t, m.CleanupSturmIsolation())
			if q.active || q.normalization.constructed != source || q.normalizePending() {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
	if !success || !failed {
		t.Fatal("normalization reservation campaign incomplete", success, failed)
	}
}

// A returning allocation failure in the real debug old_p_i copy arms a
// separately qualified source-release guard on the already constructed g.
func TestUPolynomialSturmIsolationNormalizeDebugStopProtectsPrepared(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("old_p_i is erased outside Z3DEBUG")
	}
	reached := false
	for after := int64(0); after < 64 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			var m *UPolynomialManager
			var q *upolySturmIsolationScope
			var ref unumeral.NumericCellRef
			var requested uint64
			var issued unumeral.IntegerCopyAllocationReceipt
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || q.normalization.constructed < 2 {
					return f
				}
				z := m.config.Integer
				if !z.CopyAllocation.Executing || !z.MatchesCopyAllocationInvocation(&q.normalization.copy) {
					return f
				}
				issued = q.normalization.copy
				ref = q.normalization.values[0].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
				if e := m.normalizeWithSourceContext(UPolynomialView{}, q.context()); !errors.Is(e, ErrUPolynomialOwner) {
					t.Fatal("failure transport entered normalization", e)
				}
				if used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
					t.Fatal("refused normalization changed storage")
				}
				cell := z.Cell(ref)
				requested = cell.Backing.Requested
				cell.Backing.Requested = uint64(^uint32(0)) + 1
				z.Store(ref, cell)
				return f
			}
			c, manager, _ := algebraicNegTLSFixture(t, transport)
			m = manager
			z := m.config.Integer
			p := upolySourceNew(t, m, nil)
			var inputs [3]unumeral.Int
			for i, text := range []string{"18446744073709551616", "55340232221128654848", "36893488147419103232"} {
				serviceMust(t, z.SetString(&inputs[i], text))
			}
			serviceMust(t, m.SetInts(p, inputs[:]))
			for i := range inputs {
				serviceMust(t, z.Del(&inputs[i]))
			}
			v, e := m.wholeView(p)
			serviceMust(t, e)
			q = &m.sturmIsolation
			b := c.config.Binary
			roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			armed = true
			z.Budget().FailAfter(after)
			body := m.SturmIsolateRoots(v, b, roots, lo, hi)
			z.Budget().FailAfter(-1)
			armed = false
			if reached {
				s := &q.normalization
				var failure umemory.Failure
				if !errors.As(body, &failure) || !errors.Is(body, arena.ErrBounds) || s.sourceStop == nil || s.copy != issued || issued.Frame == nil || s.constructed != 1 || q.childStop == nil || q.sourceStage != 0 || q.preparedSource != 0 || !q.preparedLive || z.Cell(ref).Released {
					t.Fatal("normalization stop lost g or older preparation", body, s.constructed, q.sourceStage)
				}
				for _, value := range m.values(&q.prepared)[:q.prepared.size] {
					if value.Cell != 0 && z.Cell(value.Cell).Released {
						t.Fatal("older coefficient released")
					}
				}
				if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("manager crossed stopped normalization", e)
				}
				before, used, stats := *s, z.Budget().Used(), z.RuntimeOwner().Stats()
				if e := q.finishNormalization(0, nil); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				if *s != before || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
					t.Fatal("source destructor replayed")
				}
				cell := z.Cell(ref)
				cell.Backing.Requested = requested
				z.Store(ref, cell)
			}
			source := q.normalization.constructed
			serviceMust(t, m.CleanupSturmIsolation())
			if q.active || source != q.normalization.constructed || q.normalizePending() {
				t.Fatal("owning cleanup changed source history")
			}
			for _, v := range []*BinaryRationalVector{roots, lo, hi} {
				serviceMust(t, v.Reset())
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{p}, nil, nil)
		})
	}
	if !reached {
		t.Fatal("did not reach actual debug-copy failure after g construction")
	}
}

func TestUPolynomialSturmIsolationNormalizeOuterReachedReservations(t *testing.T) {
	success, failed, normalizationDebt := false, false, false
	for after := int64(0); after < 1024 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			b := c.config.Binary
			roots, lo, hi := upolyIsolationVector(t, b), upolyIsolationVector(t, b), upolyIsolationVector(t, b)
			p := upolySourceNew(t, m, []int32{-6, 11, -6, 1})
			z := m.config.Integer
			var input [4]unumeral.Int
			for i, text := range []string{"-110680464442257309696", "202914184810805067776", "-110680464442257309696", "18446744073709551616"} {
				serviceMust(t, z.SetString(&input[i], text))
			}
			serviceMust(t, m.SetInts(p, input[:]))
			for i := range input {
				serviceMust(t, z.Del(&input[i]))
			}
			z.Budget().FailAfter(after)
			body := m.SturmIsolateRoots(mustUPolyView(t, m, p), b, roots, lo, hi)
			z.Budget().FailAfter(-1)
			q := &m.sturmIsolation
			if q.normalization.invocation != 0 && (q.normalizePending() || q.numericPending()) {
				normalizationDebt = true
			}
			if body == nil {
				success = true
				counts := upolyIsolationCounts{}
				upolyIsolationMatch(t, c.config.Rationals, b, roots, lo, hi, []upolyIsolationRational{{1, 1}, {2, 1}, {3, 1}}, &counts)
			} else {
				failed = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected isolation refusal", body)
				}
			}
			stage, stack, coefficient := q.sourceStage, q.sourceStack, q.sourceCoefficient
			serviceMust(t, m.CleanupSturmIsolation())
			if q.active || q.pending() || stage != q.sourceStage || stack != q.sourceStack || coefficient != q.sourceCoefficient || m.sturmPending != nil {
				t.Fatal("isolation cleanup changed source history")
			}
			for _, v := range []*BinaryRationalVector{roots, lo, hi} {
				serviceMust(t, v.Reset())
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{p}, nil, nil)
		})
	}
	if !success || !failed || !normalizationDebt {
		t.Fatal("isolation reservation campaign incomplete", success, failed, normalizationDebt)
	}
}
