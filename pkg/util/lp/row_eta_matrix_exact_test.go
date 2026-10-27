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

func TestLPExactRowEtaIndependentMatrixProducts(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	p := h.pair("0", "0")
	r := rand.New(rand.NewPCG(103, 2026))
	for world := 0; world < 240; world++ {
		n := 1 + r.IntN(6)
		pivot := r.IntN(n)
		e := NewRationalRowEtaMatrix(h.m, 999, uint32(pivot), uint32(n))
		pe := NewPairRowEtaMatrix(h.m, 999, uint32(pivot), uint32(n))
		matrix := lpExactIdentity(n)
		for i := 0; i < n; i++ {
			if i == pivot {
				continue
			}
			c := big.NewRat(int64(r.IntN(15)-7), int64(1+r.IntN(13)))
			if world%7 == 0 {
				c.Quo(c, new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 160)))
			}
			h.must(h.m.SetString(&p.X, c.RatString()))
			h.must(e.Append(uint32(i), &p.X))
			h.must(pe.Append(uint32(i), &p.X))
			matrix[pivot][i].Set(c)
		}
		x, y := make([]*big.Rat, n), make([]*big.Rat, n)
		for i := range x {
			x[i] = big.NewRat(int64(r.IntN(13)-6), int64(1+r.IntN(17)))
			y[i] = big.NewRat(int64(r.IntN(13)-6), int64(1+r.IntN(17)))
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

func TestLPExactRowEtaConjugationAndElements(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("3/7", "-2/11")
	for _, values := range lpPermutations(4) {
		e := NewRationalRowEtaMatrix(h.m, 99, 1, 4)
		h.must(e.Append(3, &p.Y))
		h.must(e.Append(0, &p.X))
		perm, err := PermutationFromValues(4, values)
		h.must(err)
		h.must(e.Conjugate(perm))
		old := lpExactIdentity(4)
		old[1][3].SetFrac64(-2, 11)
		old[1][0].SetFrac64(3, 7)
		// Every column is checked through the retained application, without
		// introducing an unused exact debug matrix interface.
		for j := 0; j < 4; j++ {
			in := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)}
			in[j].SetInt64(1)
			w := lpExactVectorFromOracle(h, in)
			h.must(e.ApplyLeft(w.Data))
			for i := 0; i < 4; i++ {
				lpCheckExactPermutationCoordinate(t, h, &w.Data[i], old[values[i]][values[j]])
			}
			h.must(w.Close())
		}
		if e.RowIndex() != perm.reverse[1] || e.IsDense() {
			t.Fatal("metadata")
		}
		h.must(e.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func lpExactRowEtaCheckFaultSuccess(t *testing.T, h *lpPairHarness, op string, e *PairRowEtaMatrix, w *IndexedRationalVector, pw *IndexedPairVector, p *RationalPair) {
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
	if op == "append" {
		if e.Row.Size() != 1 || e.Row.Data[0].Index != 1 {
			t.Fatal("append endpoint")
		}
		lpCheckExactPermutationCoordinate(t, h, &e.Row.Data[0].Value, px)
		return
	}

	matrix := lpExactIdentity(2)
	matrix[0][1].Set(px)
	x := []*big.Rat{big.NewRat(2, 1), big.NewRat(2, 1)}
	y := []*big.Rat{big.NewRat(3, 1), big.NewRat(3, 1)}
	if strings.Contains(op, "large-source") {
		matrix[0][1].SetInt64(3)
		if strings.Contains(op, "right") {
			x[0] = px
			y[0] = py
		} else {
			x[1] = px
			y[1] = py
		}
	}
	if op == "sparse-left-zero-pivot" {
		x[0].SetInt64(0)
	}
	if op == "sparse-right-zero-dest" {
		x[1].SetInt64(0)
	}
	if op == "sparse-left-cancel" {
		x[0] = px
		x[1].SetInt64(1)
		matrix[0][1].Neg(px)
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

func TestLPExactRowEtaAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"append", "left", "right", "sparse-left", "sparse-right", "pair-left", "left-large-source", "right-large-source", "sparse-left-large-source", "sparse-right-large-source", "pair-large-source", "sparse-left-zero-pivot", "sparse-right-zero-dest", "sparse-left-cancel"} {
			failures, successes, publishedStops := 0, 0, 0
			for ordinal := int64(0); ordinal < 150; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
					small := h.pair("2", "3")
					e := NewPairRowEtaMatrix(h.m, 99, 0, 2)
					if op != "append" {
						c := &p.X
						if strings.Contains(op, "large-source") {
							c = &small.Y
						}
						if op == "sparse-left-cancel" {
							neg := h.pair("-"+strings.Repeat("1234567890", 100)+"/7", "0")
							c = &neg.X
						}
						h.must(e.Append(1, c))
					}
					w := NewIndexedRationalVector(h.m, 2)
					pw := NewIndexedPairVector(h.m, 2)
					for i := range w.Data {
						h.must(w.copySlot(&w.Data[i], &small.X))
						h.must(pw.copySlot(&pw.Data[i], small))
					}
					w.Index = []uint32{1, 0}
					if strings.Contains(op, "large-source") {
						i := 1
						if strings.Contains(op, "right") {
							i = 0
						}
						h.must(w.copySlot(&w.Data[i], &p.X))
						h.must(pw.copySlot(&pw.Data[i], p))
					}
					if op == "sparse-left-zero-pivot" {
						z := numeral.NewRational()
						h.must(w.copySlot(&w.Data[0], &z))
						w.Index = []uint32{1}
					}
					if op == "sparse-right-zero-dest" {
						z := numeral.NewRational()
						h.must(w.copySlot(&w.Data[1], &z))
						w.Index = []uint32{0}
					}
					if op == "sparse-left-cancel" {
						h.must(w.copySlot(&w.Data[0], &p.X))
						h.must(h.m.SetInt64(&w.Data[1], 1))
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "append":
						err = e.Append(1, &p.X)
					case "left", "left-large-source":
						err = e.RationalRowEtaMatrix.ApplyLeft(w.Data)
					case "right", "right-large-source":
						err = e.ApplyRight(w.Data)
					case "sparse-left", "sparse-left-large-source", "sparse-left-zero-pivot", "sparse-left-cancel":
						err = e.ApplyLeftToT(w, nil)
					case "sparse-right", "sparse-right-large-source", "sparse-right-zero-dest":
						err = e.ApplyRightIndexed(w)
					case "pair-left", "pair-large-source":
						err = e.ApplyLeft(pw.Data)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						lpExactRowEtaCheckFaultSuccess(t, h, op, e, w, pw, p)
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if op == "sparse-left-zero-pivot" && len(w.Index) == 2 {
							publishedStops++
						}
					}
					if e.ready() != nil {
						if er := e.Append(1, &small.X); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.ApplyLeft(pw.Data); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.RationalRowEtaMatrix.ApplyLeft(w.Data); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.ApplyRight(w.Data); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.ApplyLeftToT(w, nil); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.ApplyRightIndexed(w); er != ErrExactVectorPending {
							t.Fatal(er)
						}
						if er := e.Conjugate(NewPermutation(2)); er != ErrExactVectorPending {
							t.Fatal(er)
						}
					}
					h.must(e.Close())
					h.must(e.Close())
					h.must(w.Close())
					h.must(pw.Close())
					lpExactVectorNamedCensus(t, h, nil)
					if h.m.DivisionInvocationState().Active || h.m.RawInvocationState().Active != [3]bool{} || h.m.Z.DivisionRaw.Active || h.m.Z.GcdRaw.Active {
						t.Fatal("orphan numeric frame")
					}
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal(op, failures, successes)
			}
			if op == "sparse-left-zero-pivot" && publishedStops == 0 {
				t.Fatal("index-before-result-copy stop not reached")
			}
		}
	}
}

func TestLPExactRowEtaLiteralStorageAndMutationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("1", "-1")
	e := NewRationalRowEtaMatrix(h.m, 999, 0, 2)
	h.must(e.Append(1, &p.Y))
	h.must(e.Append(1, &p.X))
	w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
	h.must(e.ApplyRightIndexed(w))
	if !reflect.DeepEqual(w.Index, []uint32{0, 1}) {
		t.Fatal("erase and reappend order", w.Index)
	}
	lpCheckExactSparse(t, h, w, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
	h.must(e.Close())
	h.must(w.Close())
	// Source sparse-left zeroes a canceled row before erase(find(row)); a
	// missing row is rejected only after that valid numerical prefix.
	e = NewRationalRowEtaMatrix(h.m, 0, 0, 2)
	h.must(e.Append(1, &p.Y))
	w = lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
	w.Index = []uint32{1}
	if err := e.ApplyLeftToT(w, nil); err != ErrBounds {
		t.Fatal(err)
	}
	lpCheckExactPermutationCoordinate(t, h, &w.Data[0], new(big.Rat))
	if !reflect.DeepEqual(w.Index, []uint32{1}) {
		t.Fatal(w.Index)
	}
	h.must(e.Close())
	h.must(w.Close())
	// Sparse right reads the actual dense row, even with an empty index.
	e = NewRationalRowEtaMatrix(h.m, 0, 0, 2)
	h.must(e.Append(1, &p.X))
	w = lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(1, 1), new(big.Rat)})
	w.Index = nil
	h.must(e.ApplyRightIndexed(w))
	if !reflect.DeepEqual(w.Index, []uint32{1}) {
		t.Fatal(w.Index)
	}
	lpCheckExactPermutationCoordinate(t, h, &w.Data[1], big.NewRat(1, 1))
	h.must(e.Close())
	h.must(w.Close())
	// A zero live row hides an invalid term in both right applications.
	e = NewRationalRowEtaMatrix(h.m, 0, 0, 1)
	h.must(e.Append(99, &p.X))
	w = NewIndexedRationalVector(h.m, 1)
	h.must(e.ApplyRight(w.Data))
	h.must(e.ApplyRightIndexed(w))
	h.must(e.Close())
	h.must(w.Close())
	for _, op := range []string{"left", "sparse-left", "right", "sparse-right", "pair-left"} {
		pe := NewPairRowEtaMatrix(h.m, 0, 0, 1)
		err := pe.Append(0, &p.X)
		if sourceDebugTests {
			if err != ErrAssertion {
				t.Fatal(err)
			}
			h.must(pe.Close())
			continue
		}
		h.must(err)
		h.must(pe.Append(0, &p.X))
		w = lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(2, 1)})
		pw := NewIndexedPairVector(h.m, 1)
		h.must(h.m.SetInt64(&pw.Data[0].X, 2))
		h.must(h.m.SetInt64(&pw.Data[0].Y, 3))
		switch op {
		case "left":
			h.must(pe.RationalRowEtaMatrix.ApplyLeft(w.Data))
		case "sparse-left":
			h.must(pe.ApplyLeftToT(w, nil))
		case "right":
			h.must(pe.ApplyRight(w.Data))
		case "sparse-right":
			h.must(pe.ApplyRightIndexed(w))
		case "pair-left":
			h.must(pe.ApplyLeft(pw.Data))
		}
		want := int64(8)
		if op == "sparse-left" {
			want = 6
		}
		if op == "pair-left" {
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[0].X, big.NewRat(8, 1))
			lpCheckExactPermutationCoordinate(t, h, &pw.Data[0].Y, big.NewRat(12, 1))
		} else {
			lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(want, 1))
		}
		h.must(pe.Close())
		h.must(w.Close())
		h.must(pw.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactRowEtaBoundsAndConjugationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	p := h.pair("1", "0")
	for _, op := range []string{"left", "right", "sparse-left", "sparse-right", "pair-left"} {
		for _, badRow := range []bool{false, true} {
			row := uint32(0)
			if badRow {
				row = 9
			}
			e := NewPairRowEtaMatrix(h.m, 0, row, 2)
			h.must(e.Append(1, &p.X))
			h.must(e.Append(99, &p.X))
			w := lpExactVectorFromOracle(h, []*big.Rat{big.NewRat(2, 1), big.NewRat(3, 1)})
			pw := NewIndexedPairVector(h.m, 2)
			for i := range pw.Data {
				h.must(pw.copySlot(&pw.Data[i], p))
			}
			var err error
			switch op {
			case "left":
				err = e.RationalRowEtaMatrix.ApplyLeft(w.Data)
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
				t.Fatal(op, badRow, err)
			}
			if !badRow {
				switch op {
				case "left":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(5, 1))
				case "sparse-left":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(2, 1))
				case "right", "sparse-right":
					lpCheckExactPermutationCoordinate(t, h, &w.Data[1], big.NewRat(5, 1))
				case "pair-left":
					lpCheckExactPermutationCoordinate(t, h, &pw.Data[0].X, big.NewRat(2, 1))
				}
			}
			h.must(e.Close())
			h.must(w.Close())
			h.must(pw.Close())
		}
	}
	e := NewRationalRowEtaMatrix(h.m, 0, 1, 3)
	for _, i := range []uint32{0, 99, 2} {
		h.must(e.Append(i, &p.X))
	}
	perm, err := PermutationFromValues(3, []uint32{2, 0, 1})
	h.must(err)
	if err = e.Conjugate(perm); err != ErrBounds {
		t.Fatal(err)
	}
	if e.row != 2 || e.Row.Data[0].Index != 0 || e.Row.Data[1].Index != 99 || e.Row.Data[2].Index != 0 {
		t.Fatal("descending conjugation prefix", e.row, e.Row.Data)
	}
	h.must(e.Close())
	e = NewRationalRowEtaMatrix(h.m, 0, 99, 0)
	if err = e.Conjugate(NewPermutation(0)); err != ErrBounds || e.row != 99 {
		t.Fatal(err)
	}
	h.must(e.Close())
	foreign := newLPPairHarness(t)
	fw := NewIndexedRationalVector(foreign.m, 1)
	e = NewRationalRowEtaMatrix(h.m, 0, 0, 1)
	if err = e.ApplyLeftToT(fw, nil); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	if err = e.ApplyRightIndexed(fw); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(e.Close())
	h.must(fw.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactRowEtaTailComposition(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair("-3/11", "1/7")
	d := h.pair("-2/3", "0")
	row := NewRationalRowEtaMatrix(h.m, 0, 1, 3)
	prow := NewPairRowEtaMatrix(h.m, 0, 1, 3)
	h.must(row.Append(0, &a.X))
	h.must(prow.Append(0, &a.X))
	col := NewRationalEtaMatrix(h.m, 2, 3)
	pcol := NewPairEtaMatrix(h.m, 2, 3)
	_, err := col.SetDiagonal(&d.X)
	h.must(err)
	_, err = pcol.SetDiagonal(&d.X)
	h.must(err)
	h.must(col.Append(0, &a.Y))
	h.must(pcol.Append(0, &a.Y))
	perm, err := lpTestRationalPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	pperm, err := lpTestPairPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	rm, cm, pm := lpExactIdentity(3), lpExactIdentity(3), lpExactIdentity(3)
	rm[1][0].SetFrac64(-3, 11)
	cm[2][2].SetFrac64(-3, 2)
	cm[0][2].SetFrac64(1, 7)
	for i := range pm {
		for j := range pm[i] {
			pm[i][j].SetInt64(0)
		}
		pm[i][perm.values[i]].SetInt64(1)
	}
	matrices := [][][]*big.Rat{rm, pm, cm, rm}
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
			for i := 0; i < 4; i++ {
				if pair {
					tail := []PairTailMatrix{prow, pperm, pcol, prow}[i]
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
					tail := []RationalTailMatrix{row, perm, col, row}[i]
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
			h.must(w.Close())
			h.must(pw.Close())
		}
	}
	for _, tail := range []RationalTailMatrix{row, perm, col} {
		h.must(tail.Close())
	}
	for _, tail := range []PairTailMatrix{prow, pperm, pcol} {
		h.must(tail.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactRowEtaStoppedDestinationAndOlderRawOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		var h *lpPairHarness
		if direct {
			h = lpNewDirectSparseHarness(t)
		} else {
			h = newLPPairHarness(t)
		}
		a := h.pair(strings.Repeat("1234567890", 100)+"/7", "1")
		b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
		w := NewIndexedRationalVector(h.m, 2)
		h.must(w.SetValue(&a.X, 0))
		e := NewRationalRowEtaMatrix(h.m, 0, 0, 2)
		h.budget.FailAfter(0)
		err := w.Add(0, &b.X)
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
			t.Fatal("claimed destination lexical storage")
		}
		h.must(e.Close())
		if w.calls != before || h.m.RawInvocationState() != raw {
			t.Fatal("consumed destination receipt")
		}
		// A different matrix can complete an inline operation without acquiring
		// the older pending addition. Its Close leaves the real owner's raw frame.
		other := NewRationalRowEtaMatrix(h.m, 0, 0, 1)
		small := NewIndexedRationalVector(h.m, 1)
		h.must(other.ApplyLeft(small.Data))
		h.must(other.ApplyRight(small.Data))
		h.must(other.Close())
		h.must(small.Close())
		if h.m.RawInvocationState() != raw {
			t.Fatal("consumed unrelated addition")
		}
		h.must(w.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}
