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
func TestUPolynomialRootIDNormalizeReachedReservations(t *testing.T) {
	success, failed := false, false
	for after := int64(0); after < 512 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			_, m, _ := algebraicNegTLSFixture(t, nil)
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
			var receipt upolyRootIDInvocationReceipt
			q, e := m.beginRootID(v, &receipt)
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
			serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
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
func TestUPolynomialRootIDNormalizeDebugStopProtectsSequence(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("old_p_i is erased outside Z3DEBUG")
	}
	reached := false
	for after := int64(0); after < 64 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			var m *UPolynomialManager
			var ref unumeral.NumericCellRef
			var requested uint64
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || m.rootID.normalization.constructed < 2 {
					return f
				}
				z := m.config.Integer
				ref = m.rootID.normalization.values[0].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				armed = false
				reached = true
				cell := z.Cell(ref)
				requested = cell.Backing.Requested
				cell.Backing.Requested = uint64(^uint32(0)) + 1
				z.Store(ref, cell)
				return f
			}
			_, m, _ = algebraicNegTLSFixture(t, transport)
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
			var receipt upolyRootIDInvocationReceipt
			q, e := m.beginRootID(v, &receipt)
			serviceMust(t, e)
			serviceMust(t, q.sturmPrefix(v))
			armed = true
			z.Budget().FailAfter(after)
			body := q.normalizeView(v)
			z.Budget().FailAfter(-1)
			armed = false
			body = q.finishVector(1, body)
			body = q.finish(body)
			if reached {
				s := &q.normalization
				var failure umemory.Failure
				if !errors.As(body, &failure) || !errors.Is(body, arena.ErrBounds) || s.sourceStop == nil || s.constructed != 1 || q.childStop == nil || q.sourceCursor != ([3]uint32{}) || !q.live[2] || z.Cell(ref).Released {
					t.Fatal("normalization stop lost g or older sequence", body, s.constructed, q.sourceCursor)
				}
				for _, value := range m.values(&q.sequence.coefficients)[:q.sequence.coefficients.size] {
					if value.Cell != 0 && z.Cell(value.Cell).Released {
						t.Fatal("older coefficient released")
					}
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
			serviceMust(t, m.cleanupRootIDWithReceipt(&receipt))
			if q.active || source != q.normalization.constructed || q.normalizePending() {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
	if !reached {
		t.Fatal("did not reach actual debug-copy failure after g construction")
	}
}
