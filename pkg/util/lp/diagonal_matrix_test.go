package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPDiagonalTailIndependentMatrixOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewPCG(83741, 17931))
	for world := 0; world < 240; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + rng.IntN(7)
			index := uint32(rng.IntN(n))
			d := big.NewRat(int64(1+rng.IntN(19)), int64(1+rng.IntN(19)))
			if world%2 == 0 {
				d.Neg(d)
			}
			coefficient := h.pair(d.RatString(), "0")
			m, err := NewRationalDiagonalMatrix(h.m, index, &coefficient.X)
			h.must(err)
			p, err := NewPairDiagonalMatrix(h.m, index, &coefficient.X)
			h.must(err)
			df, _ := d.Float64()
			f := NewFloatDiagonalMatrix(index, df)
			// An independent permutation conjugates the diagonal dense oracle.
			values := make([]uint32, n)
			for i, j := range rng.Perm(n) {
				values[i] = uint32(j)
			}
			q, err := PermutationFromValues(uint32(n), values)
			h.must(err)
			for i, j := range values {
				if j == index {
					index = uint32(i)
					break
				}
			}
			h.must(m.Conjugate(q))
			h.must(p.Conjugate(q))
			lpStaticMust(t, f.Conjugate(q))
			if m.Index() != index || p.Index() != index || f.Index() != index || m.IsDense() || p.IsDense() || f.IsDense() {
				t.Fatal("tail identity")
			}
			a := lpExactIdentity(n)
			a[index][index].Inv(d)
			b, c := make([]*big.Rat, n), make([]*big.Rat, n)
			for i := range b {
				b[i] = big.NewRat(int64(rng.IntN(19)-9), 7)
				c[i] = big.NewRat(int64(rng.IntN(19)-9), 11)
			}
			want, wantY := lpExactMatrixVector(a, b, false), lpExactMatrixVector(a, c, false)
			for mode := 0; mode < 4; mode++ {
				v := lpExactVectorFromOracle(h, b)
				fv := NewIndexedFloatVector(uint32(n))
				fv.Index = append([]uint32(nil), v.Index...)
				for i := range b {
					fv.Data[i], _ = b[i].Float64()
				}
				s := &Settings{DropTolerance: 1e-8}
				switch mode {
				case 0:
					h.must(m.ApplyLeft(v.Data))
					lpStaticMust(t, f.ApplyLeft(fv.Data))
				case 1:
					h.must(m.ApplyRight(v.Data))
					lpStaticMust(t, f.ApplyRight(fv.Data))
				case 2:
					h.must(m.ApplyLeftToT(v, s))
					lpStaticMust(t, f.ApplyLeftToT(fv, s))
				case 3:
					h.must(m.ApplyRightIndexed(v))
					lpStaticMust(t, f.ApplyRightIndexed(fv))
				}
				lpCheckExactSparse(t, h, v, want)
				for i := range want {
					w, _ := want[i].Float64()
					if math.Abs(fv.Data[i]-w) > 2e-15*(1+math.Abs(w)) {
						t.Fatalf("float %d got %g want %g", i, fv.Data[i], w)
					}
				}
				h.must(v.Close())
			}
			pairs := make([]RationalPair, n)
			for i := range pairs {
				pairs[i] = NewRationalPair()
				h.must(h.m.SetString(&pairs[i].X, b[i].RatString()))
				h.must(h.m.SetString(&pairs[i].Y, c[i].RatString()))
			}
			h.must(p.ApplyLeft(pairs))
			for i := range pairs {
				h.check(&pairs[i], want[i], wantY[i])
				h.must(pairs[i].Del(h.m))
			}
			lpExactVectorNamedCensus(t, h, []*numeral.Rational{&m.value, &m.inverse, &p.value, &p.inverse})
			h.must(p.Close())
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPDiagonalTailLiteralToleranceAndIEEE(t *testing.T) {
	for _, x := range []float64{1e-15, 1e-14, -1e-14, 1e-8, math.Copysign(0, -1), math.NaN(), math.Inf(1)} {
		for _, left := range []bool{false, true} {
			f := NewFloatDiagonalMatrix(1, 1)
			v := &IndexedFloatVector{Data: []float64{7, x, 9}, Index: []uint32{2, 1, 0}}
			if left {
				lpStaticMust(t, f.ApplyLeftToT(v, &Settings{DropTolerance: 1e-8}))
			} else {
				lpStaticMust(t, f.ApplyRightIndexed(v))
			}
			eps := 1e-14
			if left {
				eps = 1e-8
			}
			drop := x != 0 && x > -eps && x < eps
			if drop {
				if v.Data[1] != 0 || math.Signbit(v.Data[1]) || !reflect.DeepEqual(v.Index, []uint32{2, 0}) {
					t.Fatal("drop", x, left, v)
				}
			} else if (!math.IsNaN(x) && math.Float64bits(v.Data[1]) != math.Float64bits(x)) || !reflect.DeepEqual(v.Index, []uint32{2, 1, 0}) {
				t.Fatal("retention", x, left, v)
			}
		}
	}
	// Erase removes only the first duplicate; hidden nonzero data never inserts.
	f := NewFloatDiagonalMatrix(0, 1e20)
	v := &IndexedFloatVector{Data: []float64{1}, Index: []uint32{0, 0}}
	lpStaticMust(t, f.ApplyRightIndexed(v))
	if !reflect.DeepEqual(v.Index, []uint32{0}) || v.Data[0] != 0 {
		t.Fatal("duplicate erase", v)
	}
	v = &IndexedFloatVector{Data: []float64{2}}
	lpStaticMust(t, NewFloatDiagonalMatrix(0, 2).ApplyRightIndexed(v))
	if v.Data[0] != 1 || len(v.Index) != 0 {
		t.Fatal("hidden input repaired", v)
	}
	// Sparse zero short-circuit precedes division/settings reads; dense 0/0 runs.
	z := NewFloatDiagonalMatrix(0, 0)
	v = &IndexedFloatVector{Data: []float64{math.Copysign(0, -1)}}
	lpStaticMust(t, z.ApplyLeftToT(v, nil))
	lpStaticMust(t, z.ApplyRightIndexed(v))
	if !math.Signbit(v.Data[0]) {
		t.Fatal("zero skip changed sign")
	}
	lpStaticMust(t, z.ApplyLeft(v.Data))
	if !math.IsNaN(v.Data[0]) {
		t.Fatal("dense zero division skipped")
	}
	h := newLPPairHarness(t)
	d := h.pair("1000000000000000000000000000000", "0")
	m, err := NewRationalDiagonalMatrix(h.m, 0, &d.X)
	h.must(err)
	w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1)})
	h.must(m.ApplyLeftToT(w, &Settings{DropTolerance: 1}))
	tiny := new(big.Rat).Inv(new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)))
	lpCheckExactSparse(t, h, w, []*big.Rat{tiny})
	h.must(m.ApplyRightIndexed(w))
	tiny.Mul(tiny, tiny)
	lpCheckExactSparse(t, h, w, []*big.Rat{tiny})
	w.Index = nil
	h.must(m.ApplyLeftToT(w, nil))
	tiny.Quo(tiny, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)))
	lpSparseRationalCheck(t, h, &w.Data[0], tiny)
	if len(w.Index) != 0 {
		t.Fatal("exact hidden data repaired")
	}
	h.must(m.Close())
	h.must(w.Close())
}

func TestLPDiagonalTailBoundsAndOwnerGuards(t *testing.T) {
	f := NewFloatDiagonalMatrix(1, 2)
	for _, op := range []func() error{
		func() error { return f.ApplyLeft(nil) }, func() error { return f.ApplyRight(nil) },
		func() error { return f.ApplyRightIndexed(NewIndexedFloatVector(1)) },
		func() error { return f.ApplyLeftToT(NewIndexedFloatVector(1), nil) },
		func() error { return f.Conjugate(NewPermutation(1)) },
	} {
		if err := op(); err != ErrBounds || f.Index() != 1 {
			t.Fatalf("float bounds %v", err)
		}
	}
	h := newLPPairHarness(t)
	value := h.pair("2", "0")
	for mode := 0; mode < 5; mode++ {
		m, err := NewRationalDiagonalMatrix(h.m, 1, &value.X)
		h.must(err)
		v := NewIndexedRationalVector(h.m, 1)
		switch mode {
		case 0:
			err = m.ApplyLeft(nil)
		case 1:
			err = m.ApplyRight(nil)
		case 2:
			err = m.ApplyRightIndexed(v)
		case 3:
			err = m.ApplyLeftToT(v, nil)
		case 4:
			err = m.Conjugate(NewPermutation(1))
		}
		if err != ErrBounds || m.Index() != 1 {
			t.Fatalf("exact bounds %v", err)
		}
		if mode < 4 {
			if err := m.Conjugate(NewPermutation(2)); err != ErrExactVectorPending {
				t.Fatalf("stopped tail %v", err)
			}
		}
		h.must(m.Close())
		h.must(v.Close())
	}
	m, err := NewRationalDiagonalMatrix(h.m, 0, &value.X)
	h.must(err)
	other := newLPPairHarness(t)
	foreign := NewIndexedRationalVector(other.m, 1)
	for _, op := range []func() error{func() error { return m.ApplyRightIndexed(foreign) }, func() error { return m.ApplyLeftToT(foreign, nil) }} {
		if err := op(); err != ErrExactVectorOwner {
			t.Fatalf("foreign owner %v", err)
		}
	}
	h.must(foreign.Close())
	h.must(m.Close())
	p, err := NewPairDiagonalMatrix(h.m, 1, &value.X)
	h.must(err)
	if err := p.ApplyLeft(nil); err != ErrBounds {
		t.Fatalf("pair bounds %v", err)
	}
	if err := p.ApplyLeft(nil); err != ErrExactVectorPending {
		t.Fatalf("pair stopped %v", err)
	}
	h.must(p.Close())
}

// A live constructor copies its value parameter and owns the debug reciprocal
// even when no exact debug matrix getter consumes it. Caller mutation must not
// change either tail's later application.
func TestLPDiagonalTailConstructorOwnsValueAndDebugReciprocal(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, text := range []string{"-7/3", "123456789123456789123456789/170141183460469231731687303715884105727"} {
			t.Run(fmt.Sprintf("direct=%t/%s", direct, text), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				value := h.pair(text, "0")
				original, _ := new(big.Rat).SetString(text)
				reciprocal := new(big.Rat).Inv(original)
				m, err := NewRationalDiagonalMatrix(h.m, 0, &value.X)
				h.must(err)
				p, err := NewPairDiagonalMatrix(h.m, 0, &value.X)
				h.must(err)
				h.must(h.m.SetInt64(&value.X, 2))
				lpSparseRationalCheck(t, h, &m.value, original)
				lpSparseRationalCheck(t, h, &p.value, original)
				if sourceDebugTests {
					lpSparseRationalCheck(t, h, &m.inverse, reciprocal)
					lpSparseRationalCheck(t, h, &p.inverse, reciprocal)
				}
				v := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1)})
				pairs := NewIndexedPairVector(h.m, 1)
				h.must(h.m.SetInt64(&pairs.Data[0].X, 1))
				h.must(h.m.SetInt64(&pairs.Data[0].Y, 2))
				h.must(m.ApplyLeft(v.Data))
				h.must(p.ApplyLeft(pairs.Data))
				h.must(p.Close())
				h.must(m.Close())
				h.must(m.Close())
				lpSparseRationalCheck(t, h, &v.Data[0], reciprocal)
				h.check(&pairs.Data[0], reciprocal, new(big.Rat).Mul(reciprocal, big.NewRat(2, 1)))
				lpExactVectorNamedCensus(t, h, []*numeral.Rational{&v.Data[0], &pairs.Data[0].X, &pairs.Data[0].Y})
				h.must(v.Close())
				h.must(pairs.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}
