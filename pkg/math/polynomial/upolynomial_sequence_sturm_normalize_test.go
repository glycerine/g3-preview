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
func TestUPolynomialSequenceSturmNormalizeReachedReservations(t *testing.T) {
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
			seq, e := m.NewSequence()
			serviceMust(t, e)
			q, e := seq.beginSturm()
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
			serviceMust(t, seq.CleanupSturm())
			if q.active || q.normalization.constructed != source || q.normalizePending() {
				t.Fatal("owning cleanup changed source history")
			}
			serviceMust(t, seq.Close())
		})
	}
	if !success || !failed {
		t.Fatal("normalization reservation campaign incomplete", success, failed)
	}
}

// A returning allocation failure in the real debug old_p_i copy arms a
// separately qualified source-release guard on the already constructed g.
func TestUPolynomialSequenceSturmNormalizeDebugStopProtectsSequence(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("old_p_i is erased outside Z3DEBUG")
	}
	reached := false
	for after := int64(0); after < 64 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			var m *UPolynomialManager
			var seq *UPolynomialSequence
			var ref unumeral.NumericCellRef
			var requested uint64
			var issued unumeral.IntegerCopyAllocationReceipt
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || seq.sturm.normalization.constructed < 2 {
					return f
				}
				z := m.config.Integer
				if !z.CopyAllocation.Executing || !z.MatchesCopyAllocationInvocation(&seq.sturm.normalization.copy) {
					return f
				}
				issued = seq.sturm.normalization.copy
				ref = seq.sturm.normalization.values[0].Cell
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
			seq, e = m.NewSequence()
			serviceMust(t, e)
			q, e := seq.beginSturm()
			serviceMust(t, e)
			serviceMust(t, q.pushCopy(v))
			armed = true
			z.Budget().FailAfter(after)
			body := q.normalizeView(v)
			z.Budget().FailAfter(-1)
			armed = false
			body = q.finish(body)
			if reached {
				s := &q.normalization
				var failure umemory.Failure
				if !errors.As(body, &failure) || !errors.Is(body, arena.ErrBounds) || s.sourceStop == nil || s.copy != issued || issued.Frame == nil || s.constructed != 1 || q.childStop == nil || q.sourceCursor != ([2]uint32{}) || len(seq.state.sizes) != 1 || z.Cell(ref).Released {
					t.Fatal("normalization stop lost g or older sequence", body, s.constructed, q.sourceCursor)
				}
				for _, value := range m.values(&seq.state.coefficients)[:seq.state.coefficients.size] {
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
			serviceMust(t, seq.CleanupSturm())
			if q.active || source != q.normalization.constructed || q.normalizePending() {
				t.Fatal("owning cleanup changed source history")
			}
			serviceMust(t, seq.Close())
		})
	}
	if !reached {
		t.Fatal("did not reach actual debug-copy failure after g construction")
	}
}

// Arm only at the actual terminal remainder checkpoint, after the derivative
// and pseudo-remainder prefix have executed, before source normalization.
func TestUPolynomialSequenceSturmNormalizePublicStopProtectsVectors(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("old_p_i source construction is erased")
	}
	reached := false
	for after := int64(0); after < 64 && !reached; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			var m *UPolynomialManager
			var seq *UPolynomialSequence
			var ref unumeral.NumericCellRef
			var requested uint64
			var issued unumeral.IntegerCopyAllocationReceipt
			armed := false
			transport := func(f umemory.Failure) error {
				if !armed || seq.sturm.normalization.constructed < 2 {
					return f
				}
				z := m.config.Integer
				if !z.CopyAllocation.Executing || !z.MatchesCopyAllocationInvocation(&seq.sturm.normalization.copy) {
					return f
				}
				issued = seq.sturm.normalization.copy
				ref = seq.sturm.normalization.values[0].Cell
				if !unumeral.ValidNumericRef(z.Cells, ref) || z.Cell(ref).Released {
					return f
				}
				reached = true
				armed = false
				cell := z.Cell(ref)
				requested = cell.Backing.Requested
				cell.Backing.Requested = uint64(^uint32(0)) + 1
				z.Store(ref, cell)
				return f
			}
			c, owner, _ := algebraicNegTLSFixture(t, transport)
			m = owner
			z := m.config.Integer
			p := upolySourceNew(t, m, nil)
			var coefficients [4]unumeral.Int
			for i, text := range []string{"-92233720368547758080", "-36893488147419103232", "0", "18446744073709551616"} {
				serviceMust(t, z.SetString(&coefficients[i], text))
			}
			serviceMust(t, m.SetInts(p, coefficients[:]))
			for i := range coefficients {
				serviceMust(t, z.Del(&coefficients[i]))
			}
			v, e := m.wholeView(p)
			serviceMust(t, e)
			seq, e = m.NewSequence()
			serviceMust(t, e)
			m.config.Checkpoint = func() bool {
				q := &seq.sturm
				if !armed && q.scalar.constructed && q.remainder.size == 2 {
					z.Budget().FailAfter(after)
					armed = true
				}
				return true
			}
			m.config.Cooperation = func() error { return nil }
			body := m.SturmSequence(v, seq)
			z.Budget().FailAfter(-1)
			armed = false
			q := &seq.sturm
			if reached {
				var failure umemory.Failure
				if !errors.As(body, &failure) || !errors.Is(body, arena.ErrBounds) || q.normalization.sourceStop == nil || q.normalization.copy != issued || issued.Frame == nil || q.normalization.constructed != 1 || q.childStop == nil || q.sourceCursor != ([2]uint32{}) || !q.live[0] || !q.live[1] || q.scalar.constructed || q.scalar.sourceCursor != 1 {
					t.Fatal("normalization source stop lost literal vector/scalar order", body, q.sourceCursor)
				}
				if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
					t.Fatal("manager crossed stopped normalization", e)
				}
				for _, state := range []*upolyVectorState{&q.remainder, &seq.state.coefficients} {
					for _, value := range m.values(state)[:state.size] {
						if value.Cell != 0 && z.Cell(value.Cell).Released {
							t.Fatal("older vector coefficient destroyed")
						}
					}
				}
				cell := z.Cell(ref)
				cell.Backing.Requested = requested
				z.Store(ref, cell)
			} else if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			source, constructed := q.sourceCursor, q.normalization.constructed
			serviceMust(t, seq.CleanupSturm())
			if q.active || q.sourceCursor != source || q.normalization.constructed != constructed {
				t.Fatal("owning normalization continuation changed source history")
			}
			serviceMust(t, seq.Close())
			algebraicCellSourcePhysical(t, c, []UPolynomial{p}, nil, nil)
		})
	}
	if !reached {
		t.Fatal("actual public Sturm normalization failure was not reached")
	}
}
