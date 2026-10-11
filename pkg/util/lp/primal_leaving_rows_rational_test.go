package lp

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalRowLeavingIndependentMembership(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 0, 4)
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		expected := make([]*numeral.Rational, 4)
		for j := 0; j < 4; j++ {
			kind := []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed}[(j+world)%4]
			f.types[j] = kind
			h.must(h.m.SetString(&f.low.Data[j], "-3/7"))
			h.must(h.m.SetString(&f.up.Data[j], "5/11"))
			if kind == ColumnFixed {
				h.must(h.m.Copy(&f.low.Data[j], &f.up.Data[j]))
			}
			if kind == ColumnLowBound || (kind == ColumnBoxed && world%2 == 0) {
				h.must(h.m.SetInt32(&f.x.Data[j], -1))
				expected[j] = &f.low.Data[j]
			} else {
				h.must(h.m.SetInt32(&f.x.Data[j], 1))
				expected[j] = &f.up.Data[j]
			}
		}
		order := []uint32{3, 1, 2, 0}
		if world%2 == 0 {
			order = []uint32{2, 0, 3, 1}
		}
		for _, j := range order {
			h.must(m.InfSet.Insert(j))
		}
		out := lpScalarBoundValue(h, "97")
		for want := int32(0); want < 4; want++ {
			before := append([]uint32(nil), m.InfSet.Indices()...)
			got, e := m.FindLeavingTableauRows(out)
			h.must(e)
			if got != want || !reflect.DeepEqual(before, m.InfSet.Indices()) {
				t.Fatal(world, got, want)
			}
			equal, e := h.m.Equal(out, expected[got])
			h.must(e)
			if !equal {
				t.Fatal("bound", world, got)
			}
			m.InfSet.Erase(uint32(got))
		}
		h.must(h.m.SetInt32(out, 97))
		got, e := m.FindLeavingTableauRows(out)
		h.must(e)
		if got != -1 {
			t.Fatal("empty membership")
		}
		lpSparseRationalCheck(t, h, out, big.NewRat(97, 1))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalRowLeavingReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "nil", "type", "upper", "low", "boxed-low", "boxed-x", "feasible", "unknown", "alias", "empty-nil"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 0, 1)
			f.types[0] = ColumnBoxed
			h.must(h.m.SetInt32(&f.x.Data[0], -1))
			h.must(h.m.SetInt32(&f.up.Data[0], 2))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.InfSet.Insert(0))
			out := lpScalarBoundValue(h, "37")
			want := error(ErrBounds)
			savedX, savedLow, savedUp, savedTypes := f.x.Data, f.low.Data, f.up.Data, f.types
			switch phase {
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
			case "nil":
				out = nil
			case "type":
				f.types = nil
			case "upper":
				f.types[0] = ColumnUpperBound
				h.must(h.m.SetInt32(&f.x.Data[0], 3))
				f.up.Data = nil
			case "low":
				f.types[0] = ColumnLowBound
				f.low.Data = nil
			case "boxed-low":
				f.low.Data = nil
			case "boxed-x":
				f.x.Data = nil
			case "feasible":
				h.must(h.m.SetInt32(&f.x.Data[0], 1))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "unknown":
				f.types[0] = ColumnType(99)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "alias":
				out = &f.low.Data[0]
				want = nil
			case "empty-nil":
				m.InfSet.Erase(0)
				out = nil
				want = nil
			}
			got, err := m.FindLeavingTableauRows(out)
			f.x.Data, f.low.Data, f.up.Data, f.types = savedX, savedLow, savedUp, savedTypes
			if err != want {
				t.Fatal(phase, err, want)
			}
			if err == nil && phase != "empty-nil" {
				if got != 0 {
					t.Fatal(got)
				}
				lpSparseRationalCheck(t, h, out, new(big.Rat))
			}
			if err != nil {
				if _, e = m.FindLeavingTableauRows(out); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
