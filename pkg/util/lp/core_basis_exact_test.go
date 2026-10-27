package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLPPairBasisIndependentMembershipAndTrace(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 4, 9)
	m, err := NewPairCoreSolverBase(f.input())
	h.must(err)
	defer f.close()
	defer func() { h.must(m.Close()) }()
	rng := rand.New(rand.NewPCG(7173, 2131))
	basic := map[uint32]bool{0: true, 1: true, 2: true, 3: true}
	m.StartTracingBasisChanges()
	for step := 0; step < 1600; step++ {
		row := rng.IntN(len(f.basis))
		slot := rng.IntN(len(f.nb))
		enter, leave := f.nb[slot], f.basis[row]
		before := append([]uint32{}, m.Trace...)
		if step%2 == 0 {
			h.must(m.ChangeBasis(enter, leave))
		} else {
			h.must(m.ChangeBasisUnconditionally(enter, leave))
		}
		delete(basic, leave)
		basic[enter] = true
		seen := map[uint32]bool{}
		for i, j := range f.basis {
			if !basic[j] || seen[j] || f.heading[j] != int32(i) {
				t.Fatal(step, "basis", f.basis, f.heading)
			}
			seen[j] = true
		}
		for i, j := range f.nb {
			if basic[j] || seen[j] || f.heading[j] != -int32(i)-1 {
				t.Fatal(step, "nonbasis", f.nb, f.heading)
			}
			seen[j] = true
		}
		if len(seen) != 9 {
			t.Fatal("lost column")
		}
		// The inverse exchange cancels only the last trace pair.
		h.must(m.ChangeBasis(leave, enter))
		delete(basic, enter)
		basic[leave] = true
		if !reflect.DeepEqual(before, m.Trace) {
			t.Fatal("trace cancellation", before, m.Trace)
		}
		h.must(m.ChangeBasis(enter, leave))
		delete(basic, leave)
		basic[enter] = true
	}
	m.StopTracingBasisChanges()
	old := append([]uint32(nil), m.Trace...)
	h.must(m.ChangeBasis(f.nb[0], f.basis[0]))
	if !reflect.DeepEqual(old, m.Trace) {
		t.Fatal("disabled trace changed")
	}
	m.StartTracingBasisChanges()
	if len(m.Trace) != 0 || !m.Tracing {
		t.Fatal("trace reset")
	}
}

func TestLPPairBasisDetachedAndSourceGuardPrefixes(t *testing.T) {
	for _, op := range []string{"detached", "enter-bounds", "leave-bounds", "append-before-leave", "basic-slot", "nonbasic-slot", "basic-enter", "signed-overflow", "transpose-row", "transpose-column"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			m, err := NewPairCoreSolverBase(f.input())
			h.must(err)
			defer f.close()
			defer func() { h.must(m.Close()) }()
			want := ErrBounds
			switch op {
			case "detached":
				f.nb = nil
				err = m.ChangeBasisUnconditionally(2, 0)
				want = nil
				if !reflect.DeepEqual(f.basis, []uint32{2}) || !reflect.DeepEqual(f.nb, []uint32{0}) || !reflect.DeepEqual(f.heading, []int32{-1, -2, 0}) {
					t.Fatal(f.basis, f.nb, f.heading)
				}
			case "enter-bounds":
				err = m.ChangeBasis(9, 0)
			case "leave-bounds":
				err = m.ChangeBasis(2, 9)
			case "append-before-leave":
				f.nb = nil
				f.heading[2] = -3
				err = m.ChangeBasisUnconditionally(2, 9)
				if f.heading[2] != 0 || !reflect.DeepEqual(f.nb, []uint32{2}) {
					t.Fatal("missing append prefix")
				}
			case "basic-slot":
				f.heading[0] = -1
				err = m.ChangeBasis(2, 0)
				if f.heading[2] != -1 {
					t.Fatal("heading prefix")
				}
			case "nonbasic-slot":
				f.nb = nil
				err = m.ChangeBasis(2, 0)
				if f.basis[0] != 2 || f.heading[0] != -1 {
					t.Fatal("basis prefix")
				}
			case "basic-enter":
				err = m.ChangeBasis(0, 0)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "signed-overflow":
				f.heading[2] = -1 << 31
				err = m.ChangeBasis(2, 0)
			case "transpose-row":
				err = m.TransposeRowsTableau(0, 9)
			case "transpose-column":
				f.basis[0] = 9
				err = m.TransposeBasis(0, 0)
			}
			if err != want {
				t.Fatal(err, want)
			}
			if err != nil {
				if m.ChangeBasis(2, 0) != ErrExactVectorPending || m.ChangeBasisUnconditionally(2, 0) != ErrExactVectorPending || m.TransposeBasis(0, 0) != ErrExactVectorPending || m.TransposeRowsTableau(0, 0) != ErrExactVectorPending {
					t.Fatal("stopped owner accepted mutation")
				}
			}
		})
	}
}

func TestLPPairBasisTableauTransposeIndependentRows(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(7621, 901))
	v := h.pair("0", "0")
	for world := 0; world < 80; world++ {
		f := lpNewPairCoreFixture(t, h, 4, 7)
		perm := rng.Perm(7)
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		values := make(map[IndexPair]*big.Rat)
		for i := uint32(0); i < 4; i++ {
			for j := uint32(0); j < 7; j++ {
				x := big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(13)+1))
				h.must(h.m.SetString(&v.X, x.RatString()))
				h.must(f.a.Set(i, j, &v.X))
				if x.Sign() != 0 {
					values[IndexPair{i, j}] = x
				}
			}
		}
		m, err := NewPairCoreSolverBase(f.input())
		h.must(err)
		for step := 0; step < 30; step++ {
			i, j := uint32(rng.IntN(4)), uint32(rng.IntN(4))
			oldBasis := append([]uint32(nil), f.basis...)
			h.must(m.TransposeRowsTableau(i, j))
			if f.basis[i] != oldBasis[j] || f.basis[j] != oldBasis[i] {
				t.Fatal("basis transpose")
			}
			after := make(map[IndexPair]*big.Rat)
			for k, v := range values {
				if k.I == i {
					k.I = j
				} else if k.I == j {
					k.I = i
				}
				after[k] = v
			}
			values = after
			lpExactStaticCheck(t, h, f.a, values)
			ok, e := m.BasisHeadingIsCorrect()
			h.must(e)
			if !ok {
				t.Fatal("heading transpose")
			}
		}
		h.must(m.Close())
		f.close()
	}
}

func TestLPPairBasisTransposePhysicalFailureOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		failures := 0
		success := false
		for ordinal := int64(0); ordinal < 120; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("direct=%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f := lpNewPairCoreFixture(t, h, 2, 3)
				v := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
				h.must(f.a.Set(0, 0, &v.X))
				h.must(f.a.Set(0, 2, &v.Y))
				h.must(f.a.Set(1, 1, &v.Y))
				h.must(f.a.Set(1, 2, &v.X))
				m, err := NewPairCoreSolverBase(f.input())
				h.must(err)
				h.budget.FailAfter(ordinal)
				err = m.TransposeRowsTableau(0, 1)
				h.budget.FailAfter(-1)
				if !reflect.DeepEqual(f.basis, []uint32{1, 0}) || !reflect.DeepEqual(f.heading, []int32{1, 0, -1}) {
					t.Fatal("basis must precede matrix copies", f.basis, f.heading)
				}
				if err == nil {
					success = true
					done = true
					lpSparseRationalCheck(t, h, &f.a.Rows[0][0].Value, mustLPRat(t, strings.Repeat("9876543210", 100)+"/13"))
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.TransposeRowsTableau(0, 1) != ErrExactVectorPending || m.ChangeBasis(2, 1) != ErrExactVectorPending {
						t.Fatal("lost stopped owner")
					}
					if m.calls.pending() {
						t.Fatal("core adopted matrix-owned numeric failure")
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if failures == 0 || !success {
			t.Fatal("incomplete physical campaign", direct, failures, success)
		}
	}
}
