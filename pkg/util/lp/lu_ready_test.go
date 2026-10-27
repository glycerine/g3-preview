package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"
)

func lpPreparedColumnOracle(u [][]*big.Rat, r []uint32, x []*big.Rat) []*big.Rat {
	rx := make([]*big.Rat, len(x))
	for i, j := range r {
		rx[i] = x[j]
	}
	return lpExactMatrixVector(u, rx, false)
}

func TestLPLUPreparedSolvesIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(11317, 971))
	for world := 0; world < 40; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + world%8
			b := lpExactIdentity(n)
			order := rng.Perm(n)
			columns := rng.Perm(2*n + 1)
			basis := make([]uint32, n)
			for i := range b {
				basis[i] = uint32(columns[i])
				for j := range b[i] {
					v := int64(rng.IntN(7) - 3)
					if j == order[i] {
						v = int64(30*n + i)
					}
					b[i][j].SetFrac64(v, 8)
				}
			}
			a := lpExactLUStatic(t, h, b, basis)
			fa := lpFloatLUSource(t, b, basis)
			s := lpFloatLUSettings(math.Inf(1))
			if (world/8)%2 == 0 {
				s.DensityThreshold = 0
			}
			f, err := NewFloatLU(fa, basis, s)
			lpStaticMust(t, err)
			defer f.Close()
			r, err := NewRationalLU(a, basis, s)
			h.must(err)
			defer func() { h.must(r.Close()) }()
			p, err := NewPairLU(a, basis, s)
			h.must(err)
			defer func() { h.must(p.Close()) }()
			x, y := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range x {
				x[i] = big.NewRat(int64(rng.IntN(9)-4), 8)
				y[i] = big.NewRat(int64(rng.IntN(11)-5), 16)
			}
			fu, ru, pu := lpExactIdentity(n), lpExactIdentity(n), lpExactIdentity(n)
			for i := range fu {
				for j := range fu[i] {
					fv, e := f.U.Get(uint32(i), uint32(j))
					lpStaticMust(t, e)
					fu[i][j].SetFloat64(fv)
					rv, e := r.U.Get(uint32(i), uint32(j))
					h.must(e)
					text, e := h.m.String(rv)
					h.must(e)
					ru[i][j] = mustLPRat(t, text)
					pv, e := p.U.Get(uint32(i), uint32(j))
					h.must(e)
					text, e = h.m.String(pv)
					h.must(e)
					pu[i][j] = mustLPRat(t, text)
				}
			}
			ready := lpPreparedColumnOracle(fu, f.R.values, x)
			dx, dt := make([]float64, n), make([]float64, n)
			fi := NewIndexedFloatVector(uint32(n))
			for i := range ready {
				dx[i], _ = ready[i].Float64()
				dt[i] = dx[i]
				if dx[i] != 0 {
					lpStaticMust(t, fi.SetValue(dx[i], uint32(i)))
				}
			}
			prefix := []uint32{0, uint32(n - 1)}
			index := append([]uint32(nil), prefix...)
			wantIndex := append([]uint32(nil), prefix...)
			for i := n; i > 0; {
				i--
				if x[i].Sign() != 0 {
					wantIndex = append(wantIndex, uint32(i))
				}
			}
			lpStaticMust(t, f.SolveByWhenReadyForX(dx))
			lpStaticMust(t, f.SolveByWhenReadyForT(dt, &index))
			lpStaticMust(t, f.SolveByForTIndexedOnly(fi, nil))
			for i := range x {
				lpFloatBlockNear(t, dx[i], x[i])
				lpFloatBlockNear(t, dt[i], x[i])
				lpFloatBlockNear(t, fi.Data[i], x[i])
			}
			if !reflect.DeepEqual(index, wantIndex) {
				t.Fatal("descending append", index, wantIndex)
			}
			for _, pair := range []bool{false, true} {
				u, perm := ru, r.R.values
				if pair {
					u, perm = pu, p.R.values
				}
				rhs := lpPreparedColumnOracle(u, perm, x)
				vx := lpExactLUVector(h, rhs)
				vt := lpExactLUVector(h, rhs)
				vi := lpExactLUVector(h, rhs)
				exactIndex := append([]uint32(nil), prefix...)
				if pair {
					pairs := NewIndexedPairVector(h.m, uint32(n))
					rhsY := lpPreparedColumnOracle(u, perm, y)
					for i := range pairs.Data {
						h.must(h.m.Copy(&pairs.Data[i].X, &vx.Data[i]))
						h.must(h.m.SetString(&pairs.Data[i].Y, rhsY[i].RatString()))
					}
					h.must(p.SolveByWhenReadyForX(pairs.Data))
					for i := range x {
						h.check(&pairs.Data[i], x[i], y[i])
					}
					h.must(pairs.Close())
					h.must(p.SolveByWhenReadyForT(vt.Data, &exactIndex))
					h.must(p.SolveByForTIndexedOnly(vi, nil))
				} else {
					h.must(r.SolveByWhenReadyForX(vx.Data))
					lpExactLUCheckVector(t, h, vx.Data, x)
					h.must(r.SolveByWhenReadyForT(vt.Data, &exactIndex))
					h.must(r.SolveByForTIndexedOnly(vi, nil))
				}
				lpExactLUCheckVector(t, h, vt.Data, x)
				lpExactLUCheckVector(t, h, vi.Data, x)
				if !reflect.DeepEqual(exactIndex, wantIndex) {
					t.Fatal("precise descending append", exactIndex, wantIndex)
				}
				h.must(vx.Close())
				h.must(vt.Close())
				h.must(vi.Close())
			}
			h.must(r.Close())
			h.must(p.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
		})
	}
}

func TestLPLUPreparedSolvesToleranceAndSettings(t *testing.T) {
	for _, memberDrop := range []float64{0, 0.5} {
		// Four rows keep the one-entry residual on the sparse correction route.
		a := lpFloatLUSource(t, lpExactIdentity(4), []uint32{0, 1, 2, 3})
		s := lpFloatLUSettings(math.Inf(1))
		s.DropTolerance = memberDrop
		f, err := NewFloatLU(a, []uint32{0, 1, 2, 3}, s)
		lpStaticMust(t, err)
		defer f.Close()
		vi := NewIndexedFloatVector(4)
		lpStaticMust(t, vi.SetValue(0.25, 0))
		lpStaticMust(t, f.SolveByForTIndexedOnly(vi, &Settings{DropTolerance: 0.5 - memberDrop}))
		want := 0.25
		if memberDrop == 0.5 {
			want = 0
		}
		if vi.Data[0] != want {
			t.Fatal("floating must use member settings", memberDrop, vi.Data)
		}
	}
	// At one row the residual takes the dense correction route, which restores
	// a value dropped by the first sparse pass before fixed-tolerance cleanup.
	{
		a := lpFloatLUSource(t, lpExactIdentity(1), []uint32{0})
		s := lpFloatLUSettings(math.Inf(1))
		s.DropTolerance = 0.5
		f, e := NewFloatLU(a, []uint32{0}, s)
		lpStaticMust(t, e)
		defer f.Close()
		y := NewIndexedFloatVector(1)
		lpStaticMust(t, y.SetValue(0.25, 0))
		lpStaticMust(t, f.SolveByForTIndexedOnly(y, nil))
		if y.Data[0] != 0.25 || len(y.Index) != 1 {
			t.Fatal("dense residual restores value", y.Data, y.Index)
		}
	}

	a := lpFloatLUSource(t, lpExactIdentity(4), []uint32{0, 1, 2, 3})
	s := lpFloatLUSettings(math.Inf(1))
	f, err := NewFloatLU(a, []uint32{0, 1, 2, 3}, s)
	lpStaticMust(t, err)
	defer f.Close()
	// Isolate the cleanup profile from the constructor's equal-priority swaps.
	f.R.Init(4)
	data := []float64{1e-15, 1e-14, 0, math.Inf(1)}
	index := []uint32{2}
	lpStaticMust(t, f.SolveByWhenReadyForT(data, &index))
	if data[0] != 0 || data[1] != 1e-14 || !math.IsNaN(data[3]) || !reflect.DeepEqual(index, []uint32{2, 3, 1}) {
		t.Fatal("strict dense cleanup/IEEE", data, index)
	}
	for _, pair := range []bool{false, true} {
		h := newLPPairHarness(t)
		a := lpExactLUStatic(t, h, lpExactIdentity(1), []uint32{0})
		s := lpFloatLUSettings(math.Inf(1))
		s.DropTolerance = 1e200
		v := lpExactLUVector(h, []*big.Rat{mustLPRat(t, "1/1000000000000000000000000000000000000000000")})
		index := []uint32{0}
		if pair {
			m, e := NewPairLU(a, []uint32{0}, s)
			h.must(e)
			h.must(m.SolveByWhenReadyForT(v.Data, &index))
			h.must(m.SolveByForTIndexedOnly(v, nil))
			h.must(m.Close())
		} else {
			m, e := NewRationalLU(a, []uint32{0}, s)
			h.must(e)
			h.must(m.SolveByWhenReadyForT(v.Data, &index))
			h.must(m.SolveByForTIndexedOnly(v, nil))
			h.must(m.Close())
		}
		lpSparseRationalCheck(t, h, &v.Data[0], mustLPRat(t, "1/1000000000000000000000000000000000000000000"))
		if !reflect.DeepEqual(index, []uint32{0, 0}) {
			t.Fatal("precise tiny append", index)
		}
		h.must(v.Close())
		lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
	}
}
