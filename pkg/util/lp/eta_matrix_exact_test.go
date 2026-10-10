package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpExactIdentity(n int) [][]*big.Rat {
	out := make([][]*big.Rat, n)
	for i := range out {
		out[i] = make([]*big.Rat, n)
		for j := range out[i] {
			out[i][j] = new(big.Rat)
			if i == j {
				out[i][j].SetInt64(1)
			}
		}
	}
	return out
}
func lpExactMatrixVector(m [][]*big.Rat, v []*big.Rat, right bool) []*big.Rat {
	out := make([]*big.Rat, len(v))
	for i := range out {
		out[i] = new(big.Rat)
		for j := range v {
			c := m[i][j]
			if right {
				c = m[j][i]
			}
			out[i].Add(out[i], new(big.Rat).Mul(c, v[j]))
		}
	}
	return out
}
func lpExactVectorFromOracle(h *lpPairHarness, values []*big.Rat) *IndexedRationalVector {
	v := NewIndexedRationalVector(h.m, uint32(len(values)))
	for i, x := range values {
		h.must(h.m.SetString(&v.Data[i], x.RatString()))
	}
	for i := len(values) - 1; i >= 0; i-- {
		if values[i].Sign() != 0 {
			v.Index = append(v.Index, uint32(i))
		}
	}
	return v
}
func lpCheckExactSparse(t *testing.T, h *lpPairHarness, v *IndexedRationalVector, want []*big.Rat) {
	t.Helper()
	seen := make(map[uint32]bool)
	for _, i := range v.Index {
		if uint64(i) >= uint64(len(want)) || want[i].Sign() == 0 || seen[i] {
			t.Fatal("sparse index", v.Index)
		}
		seen[i] = true
	}
	for i, x := range want {
		lpCheckExactPermutationCoordinate(t, h, &v.Data[i], x)
		if seen[uint32(i)] != (x.Sign() != 0) {
			t.Fatal("missing sparse index", i, v.Index)
		}
	}
}
func TestLPExactEtaIndependentMatrixProducts(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("0", "0")
	r := rand.New(rand.NewPCG(828, 2026))
	for world := 0; world < 240; world++ {
		n := 1 + r.IntN(5)
		pivot := r.IntN(n)
		e := NewRationalEtaMatrix(h.m, uint32(pivot), uint32(n))
		pe := NewPairEtaMatrix(h.m, uint32(pivot), uint32(n))
		d := big.NewRat(int64(1+r.IntN(9)), int64(1+r.IntN(9)))
		if world%2 == 0 {
			d.Neg(d)
		}
		h.must(h.m.SetString(&p.X, d.RatString()))
		ok, err := e.SetDiagonal(&p.X)
		h.must(err)
		if !ok {
			t.Fatal("nonzero diagonal rejected")
		}
		ok, err = pe.SetDiagonal(&p.X)
		h.must(err)
		if !ok {
			t.Fatal("nonzero pair diagonal rejected")
		}
		matrix := lpExactIdentity(n)
		matrix[pivot][pivot].Inv(d)
		for i := 0; i < n; i++ {
			if i == pivot {
				continue
			}
			c := big.NewRat(int64(r.IntN(15)-7), int64(1+r.IntN(11)))
			if world%7 == 0 {
				c.Quo(c, new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 160)))
			}
			h.must(h.m.SetString(&p.X, c.RatString()))
			h.must(e.Append(uint32(i), &p.X))
			h.must(pe.Append(uint32(i), &p.X))
			matrix[i][pivot].Set(c)
		}
		x, y := make([]*big.Rat, n), make([]*big.Rat, n)
		for i := range x {
			x[i] = big.NewRat(int64(r.IntN(13)-6), int64(1+r.IntN(13)))
			y[i] = big.NewRat(int64(r.IntN(13)-6), int64(1+r.IntN(13)))
		}
		for _, op := range []string{"left", "right", "sparse-left", "sparse-right"} {
			w := lpExactVectorFromOracle(h, x)
			var tail RationalTailMatrix = e
			switch op {
			case "left":
				h.must(tail.ApplyLeft(w.Data))
			case "right":
				h.must(tail.ApplyRight(w.Data))
			case "sparse-left":
				h.must(tail.ApplyLeftToT(w, nil))
			case "sparse-right":
				h.must(tail.ApplyRightIndexed(w))
			}
			want := lpExactMatrixVector(matrix, x, strings.Contains(op, "right"))
			if strings.HasPrefix(op, "sparse") {
				lpCheckExactSparse(t, h, w, want)
			} else {
				for i := range want {
					lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
				}
			}
			h.must(w.Close())
		}
		pw := NewIndexedPairVector(h.m, uint32(n))
		for i := range x {
			h.must(h.m.SetString(&pw.Data[i].X, x[i].RatString()))
			h.must(h.m.SetString(&pw.Data[i].Y, y[i].RatString()))
		}
		var tail PairTailMatrix = pe
		h.must(tail.ApplyLeft(pw.Data))
		wantX, wantY := lpExactMatrixVector(matrix, x, false), lpExactMatrixVector(matrix, y, false)
		for i := range x {
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].X, wantX[i])
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].Y, wantY[i])
		}
		h.must(pw.Close())
		h.must(pe.Close())
		h.must(e.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactEtaConjugationAndElements(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("3/7", "-2/11")
	out := h.pair("0", "0")
	for _, values := range lpPermutations(4) {
		e := NewRationalEtaMatrix(h.m, 1, 4)
		ok, err := e.SetDiagonal(&p.X)
		h.must(err)
		if !ok {
			t.Fatal(ok)
		}
		h.must(e.Append(3, &p.Y))
		h.must(e.Append(0, &p.X))
		perm, err := PermutationFromValues(4, values)
		h.must(err)
		h.must(e.Conjugate(perm))
		old := lpExactIdentity(4)
		old[1][1] = big.NewRat(7, 3)
		old[3][1] = big.NewRat(-2, 11)
		old[0][1] = big.NewRat(3, 7)
		for i := uint32(0); i < 4; i++ {
			for j := uint32(0); j < 4; j++ {
				h.must(e.Element(i, j, &out.X))
				lpCheckExactPermutationCoordinate(t, h, &out.X, old[values[i]][values[j]])
			}
		}
		if e.ColumnIndex() != perm.reverse[1] || e.RowCount() != 4 || e.ColumnCount() != 4 || e.IsDense() {
			t.Fatal("metadata")
		}
		h.must(e.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactEtaLiteralSparseAndBoundsPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("1", "-1")
	e := NewRationalEtaMatrix(h.m, 0, 3)
	t.Cleanup(func() { h.must(e.Close()) })
	ok, err := e.SetDiagonal(&p.X)
	h.must(err)
	if !ok {
		t.Fatal(ok)
	}
	unit, err := e.IsUnit()
	h.must(err)
	if !unit {
		t.Fatal("identity")
	}
	h.must(e.Append(1, &p.Y))
	h.must(e.Append(1, &p.X))
	unit, err = e.IsUnit()
	h.must(err)
	if unit {
		t.Fatal("literal duplicate entries omitted")
	}
	w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1), big.NewRat(0, 1)})
	t.Cleanup(func() { h.must(w.Close()) })
	h.must(e.ApplyLeftToT(w, nil))
	if !reflect.DeepEqual(w.Index, []uint32{0, 1}) {
		t.Fatal("cancel then append index order", w.Index)
	}
	lpCheckExactSparse(t, h, w, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1), new(big.Rat)})
	h.must(e.Element(1, 0, &p.Y))
	lpCheckExactPermutationCoordinate(t, h, &p.Y, big.NewRat(-1, 1))
	h.must(e.DivideByDiagonal())
	// Dense right publishes only at the end, whereas sparse right mutates
	// the pivot first. Bounds errors preserve those distinct valid prefixes.
	h.must(e.Append(99, &p.X))
	if err = e.ApplyRight(w.Data); err != ErrBounds {
		t.Fatal(err)
	}
	lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(1, 1))
	if err = e.ApplyLeft(w.Data); err != ErrExactVectorPending {
		t.Fatal(err)
	}
	h.must(e.Close())
	lpExactVectorNamedCensus(t, h, []*numeral.Rational{&w.Data[0], &w.Data[1], &w.Data[2]})
}

func TestLPExactEtaAllocationFailureCensus(t *testing.T) {
	for _, op := range []string{"set", "append", "divide-column", "left", "right", "sparse-left", "sparse-right", "pair-left", "element", "left-large-pivot", "right-large-pivot", "sparse-left-large-pivot", "sparse-right-large-pivot", "pair-large-pivot", "sparse-left-zero-dest", "element-large-diagonal"} {
		failures, successes := 0, 0
		for ordinal := int64(0); ordinal < 150; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
				small := h.pair("2", "3")
				e := NewPairEtaMatrix(h.m, 0, 2)
				_, err := e.SetDiagonal(&small.X)
				h.must(err)
				if op != "append" {
					coefficient := &p.X
					if strings.Contains(op, "large-pivot") {
						coefficient = &small.Y
					}
					h.must(e.Append(1, coefficient))
				}
				if op == "element-large-diagonal" {
					_, err = e.SetDiagonal(&p.X)
					h.must(err)
				}
				w := NewIndexedRationalVector(h.m, 2)
				pw := NewIndexedPairVector(h.m, 2)
				for i := range w.Data {
					h.must(w.copySlot(&w.Data[i], &small.X))
					h.must(pw.copySlot(&pw.Data[i], small))
				}
				w.Index = []uint32{1, 0}
				if strings.Contains(op, "large-pivot") {
					h.must(w.copySlot(&w.Data[0], &p.X))
					h.must(pw.copySlot(&pw.Data[0], p))
				}
				if op == "sparse-left-zero-dest" {
					z := numeral.NewRational()
					h.must(w.copySlot(&w.Data[1], &z))
					w.Index = []uint32{0}
				}
				h.budget.FailAfter(ordinal)
				switch op {
				case "set":
					_, err = e.SetDiagonal(&p.X)
				case "append":
					err = e.Append(1, &p.X)
				case "divide-column":
					err = e.DivideByDiagonal()
				case "left", "left-large-pivot":
					err = e.RationalEtaMatrix.ApplyLeft(w.Data)
				case "right", "right-large-pivot":
					err = e.ApplyRight(w.Data)
				case "sparse-left", "sparse-left-large-pivot", "sparse-left-zero-dest":
					err = e.ApplyLeftToT(w, nil)
				case "sparse-right", "sparse-right-large-pivot":
					err = e.ApplyRightIndexed(w)
				case "pair-left", "pair-large-pivot":
					err = e.ApplyLeft(pw.Data)
				case "element", "element-large-diagonal":
					err = e.Element(0, 0, &w.Data[0])
				}
				h.budget.FailAfter(-1)
				if err == nil {
					lpExactEtaCheckFaultSuccess(t, h, op, e, w, pw, p)
					successes++
					done = true
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
				}
				if e.ready() != nil {
					if er := e.ApplyRight(w.Data); er != ErrExactVectorPending {
						t.Fatal(er)
					}
					if er := e.ApplyLeft(pw.Data); er != ErrExactVectorPending {
						t.Fatal(er)
					}
					if er := e.Append(1, &small.X); er != ErrExactVectorPending {
						t.Fatal(er)
					}
					if _, er := e.SetDiagonal(&small.X); er != ErrExactVectorPending {
						t.Fatal(er)
					}
					if er := e.DivideByDiagonal(); er != ErrExactVectorPending {
						t.Fatal(er)
					}
				}
				h.must(e.Close())
				h.must(e.Close())
				h.must(w.Close())
				h.must(pw.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.DivisionInvocationState().Active || h.m.RawInvocationState().Active != [3]bool{} || h.m.Z.DivisionRaw.Active || h.m.Z.GcdRaw.Active {
					t.Fatal("orphan raw frame")
				}
			})
			if done {
				break
			}
		}
		if successes != 1 {
			t.Fatal(op, "no success", failures)
		}
		if op != "element" && failures == 0 {
			t.Fatal(op, "no actual refusal")
		}
	}
}

func TestLPExactEtaTailComposition(t *testing.T) {
	h := newLPPairHarness(t)
	v := h.pair("-2/3", "1/7")
	e := NewRationalEtaMatrix(h.m, 1, 3)
	pe := NewPairEtaMatrix(h.m, 1, 3)
	_, err := e.SetDiagonal(&v.X)
	h.must(err)
	_, err = pe.SetDiagonal(&v.X)
	h.must(err)
	h.must(e.Append(0, &v.Y))
	h.must(pe.Append(0, &v.Y))
	p, err := RationalPermutationFromValues(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	pp, err := PairPermutationFromValues(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	em, pm := lpExactIdentity(3), lpExactIdentity(3)
	em[1][1].SetFrac64(-3, 2)
	em[0][1].SetFrac64(1, 7)
	for i := range pm {
		for j := range pm[i] {
			pm[i][j].SetInt64(0)
		}
		pm[i][p.values[i]].SetInt64(1)
	}
	matrices := [][][]*big.Rat{em, pm, em}
	x := []*big.Rat{big.NewRat(0, 1), big.NewRat(2, 5), big.NewRat(-3, 11)}
	y := []*big.Rat{big.NewRat(1, 13), big.NewRat(0, 1), big.NewRat(2, 17)}
	for _, op := range []string{"left", "right", "sparse-left", "sparse-right"} {
		for _, pair := range []bool{false, true} {
			w := lpExactVectorFromOracle(h, x)
			pw := NewIndexedPairVector(h.m, 3)
			for i := range x {
				h.must(h.m.SetString(&pw.Data[i].X, x[i].RatString()))
				h.must(h.m.SetString(&pw.Data[i].Y, y[i].RatString()))
			}
			want, wantY := x, y
			for i := 0; i < 3; i++ {
				if pair {
					tail := []PairTailMatrix{pe, pp, pe}[i]
					switch op {
					case "left":
						h.must(tail.ApplyLeft(pw.Data))
					case "right":
						h.must(tail.ApplyRight(w.Data))
					case "sparse-left":
						h.must(tail.ApplyLeftToT(w, nil))
					case "sparse-right":
						h.must(tail.ApplyRightIndexed(w))
					}
				} else {
					tail := []RationalTailMatrix{e, p, e}[i]
					switch op {
					case "left":
						h.must(tail.ApplyLeft(w.Data))
					case "right":
						h.must(tail.ApplyRight(w.Data))
					case "sparse-left":
						h.must(tail.ApplyLeftToT(w, nil))
					case "sparse-right":
						h.must(tail.ApplyRightIndexed(w))
					}
				}
				want = lpExactMatrixVector(matrices[i], want, strings.Contains(op, "right"))
				wantY = lpExactMatrixVector(matrices[i], wantY, false)
			}
			if pair && op == "left" {
				for i := range want {
					lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].X, want[i])
					lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].Y, wantY[i])
				}
			} else if strings.HasPrefix(op, "sparse") {
				lpCheckExactSparse(t, h, w, want)
			} else {
				for i := range want {
					lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
				}
			}
			h.must(pw.Close())
			h.must(w.Close())
		}
	}
	for _, tail := range []RationalTailMatrix{e, p} {
		h.must(tail.Close())
	}
	for _, tail := range []PairTailMatrix{pe, pp} {
		h.must(tail.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactEtaReachedGuardsAndConjugationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	v := h.pair("1", "0")
	for _, op := range []string{"left", "right", "sparse-left", "sparse-right", "pair-left"} {
		for _, pivotBad := range []bool{false, true} {
			pivot := uint32(0)
			if pivotBad {
				pivot = 9
			}
			e := NewPairEtaMatrix(h.m, pivot, 2)
			_, err := e.SetDiagonal(&v.X)
			h.must(err)
			h.must(e.Append(1, &v.X))
			h.must(e.Append(99, &v.X))
			w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(2, 1), big.NewRat(3, 1)})
			pw := NewIndexedPairVector(h.m, 2)
			for i := range pw.Data {
				h.must(pw.copySlot(&pw.Data[i], v))
			}
			switch op {
			case "left":
				err = e.RationalEtaMatrix.ApplyLeft(w.Data)
			case "right":
				err = e.ApplyRight(w.Data)
			case "sparse-left":
				err = e.ApplyLeftToT(w, nil)
			case "sparse-right":
				err = e.ApplyRightIndexed(w)
			case "pair-left":
				err = e.ApplyLeft(pw.Data)
			}
			if err != ErrBounds {
				t.Fatal(op, pivotBad, err)
			}
			if !pivotBad {
				switch op {
				case "left", "sparse-left":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[1], big.NewRat(5, 1))
				case "right":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(2, 1))
				case "sparse-right":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(5, 1))
				case "pair-left":
					lpCheckExactPermutationCoordinate(t, h, &pw.Data[1].X, big.NewRat(2, 1))
				}
			}
			if err = e.Conjugate(NewPermutation(2)); err != ErrExactVectorPending {
				t.Fatal(err)
			}
			if err = e.Element(0, 0, &v.Y); err != ErrExactVectorPending {
				t.Fatal(err)
			}
			if err = e.ApplyLeftToT(w, nil); err != ErrExactVectorPending {
				t.Fatal(err)
			}
			if err = e.ApplyRightIndexed(w); err != ErrExactVectorPending {
				t.Fatal(err)
			}
			h.must(e.Close())
			h.must(w.Close())
			h.must(pw.Close())
		}
	}
	e := NewRationalEtaMatrix(h.m, 0, 2)
	unit, err := e.IsUnit()
	h.must(err)
	if unit {
		t.Fatal("default rational diagonal is zero")
	}
	ok, err := e.SetDiagonal(&v.Y)
	h.must(err)
	if ok {
		t.Fatal("zero accepted")
	}
	_, err = e.SetDiagonal(&v.X)
	h.must(err)
	err = e.Append(0, &v.X)
	if sourceDebugTests {
		if err != ErrAssertion {
			t.Fatal(err)
		}
		h.must(e.Close())
		e = NewRationalEtaMatrix(h.m, 0, 2)
		_, err = e.SetDiagonal(&v.X)
		h.must(err)
	} else if err != nil {
		t.Fatal(err)
	}
	// Self terms are permitted in the native source and use a live pivot.
	if !sourceDebugTests {
		w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(2, 1), big.NewRat(0, 1)})
		h.must(e.ApplyLeft(w.Data))
		lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(4, 1))
		h.must(w.Close())
	}
	h.must(e.Append(1, &v.X))
	h.must(e.Append(99, &v.X))
	perm, err := PermutationFromValues(2, []uint32{1, 0})
	h.must(err)
	if err = e.Conjugate(perm); err != ErrBounds {
		t.Fatal(err)
	}
	if e.column != 1 {
		t.Fatal("pivot prefix")
	}
	last := len(e.Column.Data) - 1
	if e.Column.Data[last].Index != 99 || e.Column.Data[last-1].Index != 0 {
		t.Fatal("ordered term prefix")
	}
	h.must(e.Close())
	// Sparse right's empty-index return precedes pivot bounds.
	e = NewRationalEtaMatrix(h.m, 99, 0)
	w := NewIndexedRationalVector(h.m, 0)
	h.must(e.ApplyRightIndexed(w))
	if err = e.Conjugate(NewPermutation(0)); err != ErrBounds || e.column != 99 {
		t.Fatal(err, e.column)
	}
	h.must(e.Close())
	h.must(w.Close())
	foreign := newLPPairHarness(t)
	fw := NewIndexedRationalVector(foreign.m, 1)
	e = NewRationalEtaMatrix(h.m, 0, 1)
	if err = e.ApplyLeftToT(fw, nil); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	if err = e.ApplyRightIndexed(fw); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(fw.Close())
	h.must(e.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactEtaStoppedDestinationOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair(strings.Repeat("1234567890", 100)+"/7", "1")
	b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	w := NewIndexedRationalVector(h.m, 2)
	h.must(w.SetValue(&a.X, 0))
	e := NewRationalEtaMatrix(h.m, 0, 2)
	_, err := e.SetDiagonal(&a.Y)
	h.must(err)
	h.budget.FailAfter(0)
	err = w.Add(0, &b.X)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(err) || !w.calls.pending() {
		t.Fatal("no actual stopped destination", err)
	}
	before := w.calls
	raw := h.m.RawInvocationState()
	if err = e.ApplyLeftToT(w, nil); err != ErrExactVectorPending {
		t.Fatal(err)
	}
	if err = e.ApplyRightIndexed(w); err != ErrExactVectorPending {
		t.Fatal(err)
	}
	if e.work != nil {
		t.Fatal("matrix claimed destination lexical owner")
	}
	h.must(e.Close())
	if w.calls != before || h.m.RawInvocationState() != raw {
		t.Fatal("matrix consumed stopped destination ownership")
	}
	h.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactEtaLeavesOlderDivisionOwned(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			z := h.pair("0", "1")
			var caller exactEtaCalls
			b, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			err = h.m.Div(&p.X, &p.Y, &p.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, b, err)
			state := h.m.DivisionInvocationState()
			if state.Active {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				matched = true
				e := NewRationalEtaMatrix(h.m, 0, 1)
				_, err = e.SetDiagonal(&z.Y)
				h.must(err)
				w := NewIndexedRationalVector(h.m, 1)
				h.must(e.ApplyLeft(w.Data))
				h.must(e.Close())
				h.must(w.Close())
				if h.m.DivisionInvocationState() != state {
					t.Fatal("matrix consumed older raw division")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual stopped division reached")
	}
}

// A successful fault-campaign endpoint must also compute the independently
// expected value; cleanup accounting alone cannot prove an application ran.
func lpExactEtaCheckFaultSuccess(t *testing.T, h *lpPairHarness, op string, e *PairEtaMatrix, w *IndexedRationalVector, pw *IndexedPairVector, p *RationalPair) {
	t.Helper()
	sx, err := h.m.String(&p.X)
	h.must(err)
	sy, err := h.m.String(&p.Y)
	h.must(err)
	px, ok := new(big.Rat).SetString(sx)
	if !ok {
		t.Fatal(sx)
	}
	py, ok := new(big.Rat).SetString(sy)
	if !ok {
		t.Fatal(sy)
	}
	switch op {
	case "set":
		lpCheckExactPermutationCoordinate(t, h, &e.Diagonal, px)
		return
	case "append":
		if len(e.Column.Data) != 1 || e.Column.Data[0].Index != 1 {
			t.Fatal("append endpoint")
		}
		lpCheckExactPermutationCoordinate(t, h, &e.Column.Data[0].Value, px)
		return
	case "divide-column":
		lpCheckExactPermutationCoordinate(t, h, &e.Column.Data[0].Value, new(big.Rat).Quo(px, big.NewRat(2, 1)))
		return
	case "element":
		lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(1, 2))
		return
	case "element-large-diagonal":
		lpCheckExactPermutationCoordinate(t, h, &w.Data[0], new(big.Rat).Inv(px))
		return
	}
	matrix := lpExactIdentity(2)
	matrix[0][0].SetFrac64(1, 2)
	matrix[1][0].Set(px)
	x := []*big.Rat{big.NewRat(2, 1), big.NewRat(2, 1)}
	y := []*big.Rat{big.NewRat(3, 1), big.NewRat(3, 1)}
	if strings.Contains(op, "large-pivot") {
		x[0] = px
		y[0] = py
		matrix[1][0].SetInt64(3)
	}
	if op == "sparse-left-zero-dest" {
		x[1].SetInt64(0)
	}
	want := lpExactMatrixVector(matrix, x, strings.Contains(op, "right"))
	if strings.HasPrefix(op, "pair") {
		wantY := lpExactMatrixVector(matrix, y, false)
		for i := range want {
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].X, want[i])
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[i].Y, wantY[i])
		}
	} else if strings.HasPrefix(op, "sparse") {
		lpCheckExactSparse(t, h, w, want)
	} else {
		for i := range want {
			lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
		}
	}
}
