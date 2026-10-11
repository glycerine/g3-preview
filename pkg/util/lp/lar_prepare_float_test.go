package lp

import (
	"reflect"
	"testing"
)

func TestLARPrepareFloatSourceVectorKinds(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 100; world++ {
		s, err := NewLARSolver(h.m)
		h.must(err)
		c := s.Core
		r := c.DPrimal
		n, rows := world%11, world%7
		for i := 0; i < n; i++ {
			c.DA.AddColumn()
		}
		for i := 0; i < rows; i++ {
			c.DA.AddRow()
		}
		for _, vec := range []*IndexedFloatVector{r.Ed, r.PivotRow, r.PivotRowB1, r.W} {
			h.must(vec.Resize(15))
			h.must(vec.SetValue(17, 0))
		}
		c.DRightSides = []float64{19, 23}
		r.CopyXB = []float64{29, 31}
		c.DCosts = []float64{37, 41}
		r.D = []float64{43, 47}
		r.Y = []float64{53, 59}
		r.SteepestEdge = []float64{61, 67}
		r.ColumnNorms = []float64{71, 73}
		h.must(r.InfSet.Resize(15))
		h.must(r.InfSet.Insert(14))
		r.BreakpointQueue.Resize(15)
		h.must(c.PrefixD())
		for _, tc := range []struct {
			values []float64
			size   int
			prefix []float64
		}{{c.DRightSides, rows, []float64{19, 23}}, {r.CopyXB, n, []float64{29, 31}}, {c.DCosts, n, []float64{37, 41}}, {r.D, n, []float64{43, 47}}, {r.Y, rows, []float64{53, 59}}, {r.SteepestEdge, n, []float64{61, 67}}} {
			want := make([]float64, tc.size)
			copy(want, tc.prefix)
			if !reflect.DeepEqual(tc.values, want) {
				t.Fatal(world, tc.values, want)
			}
		}
		for _, vec := range []*IndexedFloatVector{r.Ed, r.PivotRow, r.PivotRowB1, r.W} {
			if len(vec.Index) != 0 {
				t.Fatal("indexed scratch retained")
			}
			for _, x := range vec.Data {
				if x != 0 {
					t.Fatal("indexed payload retained")
				}
			}
		}
		if !r.InfSet.Empty() || r.InfSet.DataSize() != uint32(n) || len(r.BreakpointQueue.inverse) != n {
			t.Fatal("membership/queue domain")
		}
		for _, x := range r.ColumnNorms {
			if x != 2 {
				t.Fatal("column norms must reset to two")
			}
		}
		h.must(c.PrefixD())
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARPrepareFloatFailurePrefixes(t *testing.T) {
	for _, phase := range []string{"ed", "pivot", "inverse", "w", "inf"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, err := NewLARSolver(h.m)
			h.must(err)
			c := s.Core
			r := c.DPrimal
			c.DA.AddColumn()
			c.DA.AddRow()
			c.DA.AddRow()
			bad := r.Ed
			switch phase {
			case "pivot":
				bad = r.PivotRow
			case "inverse":
				bad = r.PivotRowB1
			case "w":
				bad = r.W
			case "inf":
				bad = nil
				h.must(r.InfSet.Resize(100))
				h.must(r.InfSet.Insert(99))
				h.must(r.InfSet.Resize(0))
			}
			if bad != nil {
				bad.Index = []uint32{99}
			}
			if err = c.PrefixD(); err != ErrBounds {
				t.Fatal(err)
			}
			if len(c.DRightSides) != 2 || len(r.CopyXB) != 1 || len(c.DCosts) != 1 {
				t.Fatal("lost source resize prefix")
			}
			if c.PrefixD() != ErrExactVectorPending {
				t.Fatal("lost core stop")
			}
			if bad != nil {
				bad.Index = nil
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
