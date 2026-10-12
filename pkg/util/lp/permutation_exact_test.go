package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

// Exact fixtures use the retained identity constructor and individual Set
// operations. The source values constructor is reached only by float inverse.
func lpTestRationalPermutation(m *numeral.RationalManager, size uint32, values []uint32) (*RationalPermutation, error) {
	p := NewRationalPermutation(m, size)
	for i, value := range values {
		if err := p.Set(uint32(i), value); err != nil {
			return p, err
		}
	}
	return p, nil
}

func lpTestPairPermutation(m *numeral.RationalManager, size uint32, values []uint32) (*PairPermutation, error) {
	p := NewPairPermutation(m, size)
	for i, value := range values {
		if err := p.Set(uint32(i), value); err != nil {
			return p, err
		}
	}
	return p, nil
}

// Ordinary matrix arithmetic is independent of the implementation's mapping
// traversal and temporary buffers. Every scalar is an exact fraction.
func lpExactPermutationProduct(m [][]int, in []*big.Rat, right, inverse bool) []*big.Rat {
	if inverse {
		m = lpMatrixTranspose(m)
	}
	out := make([]*big.Rat, len(in))
	for i := range out {
		out[i] = new(big.Rat)
		for j := range in {
			coefficient := m[i][j]
			if right {
				coefficient = m[j][i]
			}
			out[i].Add(out[i], new(big.Rat).Mul(big.NewRat(int64(coefficient), 1), in[j]))
		}
	}
	return out
}
func lpCheckExactPermutationCoordinate(t *testing.T, h *lpPairHarness, got *numeral.Rational, want *big.Rat) {
	t.Helper()
	text, err := h.m.String(got)
	h.must(err)
	parsed, ok := new(big.Rat).SetString(text)
	if !ok || parsed.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", text, want)
	}
}

func TestLPRationalPermutationIndependentDenseProducts(t *testing.T) {
	h := newLPPairHarness(t)
	fractions := []string{"-1/7", "123456789012345678901234567890123456789/11", "0", "-999999999999999999999999999999999999999/13"}
	second := []string{"1/99999999999999999999999999999999999999", "-1/2", "3/17", "0"}
	originals := make([]*RationalPair, 4)
	oracle, oracleY := make([]*big.Rat, 4), make([]*big.Rat, 4)
	for i := range originals {
		originals[i] = h.pair(fractions[i], second[i])
		oracle[i], _ = new(big.Rat).SetString(fractions[i])
		oracleY[i], _ = new(big.Rat).SetString(second[i])
	}
	for _, values := range lpPermutations(4) {
		p, err := lpTestRationalPermutation(h.m, 4, values)
		h.must(err)
		for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
			w := NewIndexedRationalVector(h.m, 4)
			x := NewIndexedRationalVector(h.m, 4)
			for i := range originals {
				h.must(w.copySlot(&w.Data[i], &originals[i].X))
				h.must(x.copySlot(&x.Data[i], &originals[i].X))
			}
			switch op {
			case "left":
				h.must(p.ApplyLeft(x.Data))
			case "right":
				h.must(p.ApplyRight(w.Data))
			case "reverse-left-T":
				h.must(p.ApplyReverseLeftToT(w.Data))
			case "reverse-left-X":
				h.must(p.ApplyReverseLeftToX(x.Data))
			case "reverse-right-T":
				h.must(p.ApplyReverseRightToT(w.Data))
			case "reverse-right-X":
				h.must(p.ApplyReverseRightToX(x.Data))
			}
			want := lpExactPermutationProduct(lpPermutationMatrix(values), oracle, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			wantY := lpExactPermutationProduct(lpPermutationMatrix(values), oracleY, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			for i := range want {
				if op == "left" || strings.HasSuffix(op, "-X") {
					lpCheckExactPermutationCoordinate(t, h, &x.Data[i], want[i])
					_ = wantY
				} else {
					lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
				}
			}
			h.must(x.Close())
			h.must(w.Close())
		}
		h.must(p.Close())
		h.must(p.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

// Each campaign walks actual physical refusals through its first success.
// Direct allocation exposes copy prefixes that pooled storage may reuse; both
// profiles must preserve the caller coordinates and release the owner roots.
func TestLPRationalPermutationDenseAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
			failures, successes := 0, 0
			for fail := int64(0); fail < 80; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					small := h.pair("1", "-2")
					large := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
					p, err := lpTestRationalPermutation(h.m, 2, []uint32{1, 0})
					h.must(err)
					w := NewIndexedRationalVector(h.m, 2)
					x := NewIndexedRationalVector(h.m, 2)
					for i, v := range []*RationalPair{small, large} {
						h.must(w.copySlot(&w.Data[i], &v.X))
						h.must(x.copySlot(&x.Data[i], &v.X))
					}
					h.budget.FailAfter(fail)
					switch op {
					case "left":
						err = p.ApplyLeft(x.Data)
					case "right":
						err = p.ApplyRight(w.Data)
					case "reverse-left-T":
						err = p.ApplyReverseLeftToT(w.Data)
					case "reverse-left-X":
						err = p.ApplyReverseLeftToX(x.Data)
					case "reverse-right-T":
						err = p.ApplyReverseRightToT(w.Data)
					case "reverse-right-X":
						err = p.ApplyReverseRightToX(x.Data)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						// The successful endpoint must produce the literal swapped
						// coordinates, including the untouched input buffer.
						wantX, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
						xs := []*big.Rat{big.NewRat(1, 1), wantX}
						for i := 0; i < 2; i++ {
							wi, xi := i, 1-i
							if op != "left" && !strings.HasSuffix(op, "-X") {
								wi, xi = 1-i, i
							}
							lpCheckExactPermutationCoordinate(t, h, &w.Data[i], xs[wi])
							lpCheckExactPermutationCoordinate(t, h, &x.Data[i], xs[xi])
						}
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatalf("unexpected allocation failure: %v", err)
						}
						failures++
					}
					if p.tBuffer.calls.pending() || p.xBuffer.calls.pending() {
						if e := p.Init(2); e != ErrExactVectorPending {
							t.Fatal("pending mutation", e)
						}
						if e := p.ApplyRight(w.Data); e != ErrExactVectorPending {
							t.Fatal("pending application", e)
						}
					}
					h.must(p.Close())
					h.must(p.Close())
					// The caller retains its partially assigned coordinates after owner Close.
					// Check actual named ownership before deleting those inputs.
					extra := []*numeral.Rational{}
					for i := range w.Data {
						extra = append(extra, &w.Data[i])
					}
					for i := range x.Data {
						extra = append(extra, &x.Data[i])
					}
					lpExactVectorNamedCensus(t, h, extra)
					h.must(w.Close())
					h.must(x.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete fault campaign", op, failures, successes)
			}
		}
	}
}

func TestLPRationalPermutationResizeAndGuardContracts(t *testing.T) {
	h := newLPPairHarness(t)
	p, err := lpTestRationalPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
	w := NewIndexedRationalVector(h.m, 3)
	x := NewIndexedRationalVector(h.m, 3)
	for i := range w.Data {
		h.must(w.copySlot(&w.Data[i], &large.X))
		h.must(x.copySlot(&x.Data[i], &large.X))
	}
	h.must(p.ApplyRight(w.Data))
	h.must(p.ApplyLeft(x.Data))
	h.must(p.Resize(2))
	if !reflect.DeepEqual(p.Values(), []uint32{2, 0}) {
		t.Fatal(p.Values())
	}
	if err := p.ApplyLeft(x.Data[:2]); err != ErrBounds {
		t.Fatal(err)
	}
	h.must(p.Init(5))
	if !p.IsIdentity() || p.Size() != 5 {
		t.Fatal(p.Values())
	}
	if err := p.ApplyRight(w.Data); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(p.Init(0))
	h.must(p.ApplyRight(nil))
	h.must(p.ApplyLeft(nil))
	h.must(p.Close())
	h.must(w.Close())
	h.must(x.Close())
	lpExactVectorNamedCensus(t, h, nil)

}

func TestLPPairPermutationIndependentDenseProducts(t *testing.T) {
	h := newLPPairHarness(t)
	fractions := []string{"-1/7", "123456789012345678901234567890123456789/11", "0", "-999999999999999999999999999999999999999/13"}
	second := []string{"1/99999999999999999999999999999999999999", "-1/2", "3/17", "0"}
	originals := make([]*RationalPair, 4)
	oracle, oracleY := make([]*big.Rat, 4), make([]*big.Rat, 4)
	for i := range originals {
		originals[i] = h.pair(fractions[i], second[i])
		oracle[i], _ = new(big.Rat).SetString(fractions[i])
		oracleY[i], _ = new(big.Rat).SetString(second[i])
	}
	for _, values := range lpPermutations(4) {
		p, err := lpTestPairPermutation(h.m, 4, values)
		h.must(err)
		for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
			w := NewIndexedRationalVector(h.m, 4)
			x := NewIndexedPairVector(h.m, 4)
			for i := range originals {
				h.must(w.copySlot(&w.Data[i], &originals[i].X))
				h.must(x.copySlot(&x.Data[i], originals[i]))
			}
			switch op {
			case "left":
				h.must(p.ApplyLeft(x.Data))
			case "right":
				h.must(p.ApplyRight(w.Data))
			case "reverse-left-T":
				h.must(p.ApplyReverseLeftToT(w.Data))
			case "reverse-left-X":
				h.must(p.ApplyReverseLeftToX(x.Data))
			case "reverse-right-T":
				h.must(p.ApplyReverseRightToT(w.Data))
			case "reverse-right-X":
				h.must(p.ApplyReverseRightToX(x.Data))
			}
			want := lpExactPermutationProduct(lpPermutationMatrix(values), oracle, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			wantY := lpExactPermutationProduct(lpPermutationMatrix(values), oracleY, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			for i := range want {
				if op == "left" || strings.HasSuffix(op, "-X") {
					lpCheckExactPermutationCoordinate(t, h, &x.Data[i].X, want[i])
					lpCheckExactPermutationCoordinate(t, h, &x.Data[i].Y, wantY[i])
				} else {
					lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
				}
			}
			h.must(x.Close())
			h.must(w.Close())
		}
		h.must(p.Close())
		h.must(p.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPPairPermutationDenseAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
			failures, successes := 0, 0
			for fail := int64(0); fail < 80; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					small := h.pair("1", "-2")
					large := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
					p, err := lpTestPairPermutation(h.m, 2, []uint32{1, 0})
					h.must(err)
					w := NewIndexedRationalVector(h.m, 2)
					x := NewIndexedPairVector(h.m, 2)
					for i, v := range []*RationalPair{small, large} {
						h.must(w.copySlot(&w.Data[i], &v.X))
						h.must(x.copySlot(&x.Data[i], v))
					}
					h.budget.FailAfter(fail)
					switch op {
					case "left":
						err = p.ApplyLeft(x.Data)
					case "right":
						err = p.ApplyRight(w.Data)
					case "reverse-left-T":
						err = p.ApplyReverseLeftToT(w.Data)
					case "reverse-left-X":
						err = p.ApplyReverseLeftToX(x.Data)
					case "reverse-right-T":
						err = p.ApplyReverseRightToT(w.Data)
					case "reverse-right-X":
						err = p.ApplyReverseRightToX(x.Data)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						// The successful endpoint must produce the literal swapped
						// coordinates, including the untouched input buffer.
						wantX, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
						wantY, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/11")
						xs := []*big.Rat{big.NewRat(1, 1), wantX}
						ys := []*big.Rat{big.NewRat(-2, 1), wantY}
						for i := 0; i < 2; i++ {
							wi, xi := i, 1-i
							if op != "left" && !strings.HasSuffix(op, "-X") {
								wi, xi = 1-i, i
							}
							lpCheckExactPermutationCoordinate(t, h, &w.Data[i], xs[wi])
							lpCheckExactPermutationCoordinate(t, h, &x.Data[i].X, xs[xi])
							lpCheckExactPermutationCoordinate(t, h, &x.Data[i].Y, ys[xi])
						}
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatalf("unexpected allocation failure: %v", err)
						}
						failures++
					}
					if p.tBuffer.calls.pending() || p.xBuffer.calls.pending() {
						if e := p.Init(2); e != ErrExactVectorPending {
							t.Fatal("pending mutation", e)
						}
						if e := p.ApplyRight(w.Data); e != ErrExactVectorPending {
							t.Fatal("pending application", e)
						}
					}
					h.must(p.Close())
					h.must(p.Close())
					// The caller retains its partially assigned coordinates after owner Close.
					// Check actual named ownership before deleting those inputs.
					extra := []*numeral.Rational{}
					for i := range w.Data {
						extra = append(extra, &w.Data[i])
					}
					for i := range x.Data {
						extra = append(extra, &x.Data[i].X, &x.Data[i].Y)
					}
					lpExactVectorNamedCensus(t, h, extra)
					h.must(w.Close())
					h.must(x.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete fault campaign", op, failures, successes)
			}
		}
	}
}

func TestLPPairPermutationResizeAndGuardContracts(t *testing.T) {
	h := newLPPairHarness(t)
	p, err := lpTestPairPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	large := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/11")
	w := NewIndexedRationalVector(h.m, 3)
	x := NewIndexedPairVector(h.m, 3)
	for i := range w.Data {
		h.must(w.copySlot(&w.Data[i], &large.X))
		h.must(x.copySlot(&x.Data[i], large))
	}
	h.must(p.ApplyRight(w.Data))
	h.must(p.ApplyLeft(x.Data))
	h.must(p.Resize(2))
	if !reflect.DeepEqual(p.Values(), []uint32{2, 0}) {
		t.Fatal(p.Values())
	}
	if err := p.ApplyLeft(x.Data[:2]); err != ErrBounds {
		t.Fatal(err)
	}
	h.must(p.Init(5))
	if !p.IsIdentity() || p.Size() != 5 {
		t.Fatal(p.Values())
	}
	if err := p.ApplyRight(w.Data); err != map[bool]error{true: ErrAssertion, false: ErrBounds}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(p.Init(0))
	h.must(p.ApplyRight(nil))
	h.must(p.ApplyLeft(nil))
	h.must(p.Close())
	h.must(w.Close())
	h.must(x.Close())
	lpExactVectorNamedCensus(t, h, nil)

}

func TestLPRationalPermutationIndependentSparseProducts(t *testing.T) {
	h := newLPPairHarness(t)
	operands := []*RationalPair{h.pair("-1/7", "0"), h.pair("999999999999999999999999999999999999999/11", "0"), h.pair("0", "0"), h.pair("1/999999999999999999999999999999999999999", "0")}
	original := make([]*big.Rat, 4)
	for i, v := range operands {
		text, err := h.m.String(&v.X)
		h.must(err)
		original[i], _ = new(big.Rat).SetString(text)
	}
	for _, values := range lpPermutations(4) {
		p, err := lpTestRationalPermutation(h.m, 4, values)
		h.must(err)
		for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
			w := NewIndexedRationalVector(h.m, 4)
			// Deliberately unsorted and includes an indexed zero coordinate.
			indices := []uint32{3, 0, 2, 1}
			for _, i := range indices {
				h.must(w.SetValue(&operands[i].X, i))
			}
			switch op {
			case "left":
				h.must(p.ApplyLeftIndexed(w))
			case "right":
				h.must(p.ApplyRightIndexed(w))
			case "reverse-left":
				h.must(p.ApplyReverseLeftIndexed(w))
			case "reverse-right":
				h.must(p.ApplyReverseRightIndexed(w))
			}
			m := lpPermutationMatrix(values)
			want := lpExactPermutationProduct(m, original, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			for i := range want {
				lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
			}
			// Find the movement of each original basis vector through matrix products,
			// independently of the implementation's forward/reverse arrays.
			wantIndex := make([]uint32, len(indices))
			for k, j := range indices {
				basis := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)}
				basis[j].SetInt64(1)
				image := lpExactPermutationProduct(m, basis, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
				for i, v := range image {
					if v.Sign() != 0 {
						wantIndex[k] = uint32(i)
					}
				}
			}
			if !reflect.DeepEqual(w.Index, wantIndex) {
				t.Fatal(op, values, w.Index, wantIndex)
			}
			if p.scratch != nil {
				t.Fatal("successful lexical owner retained")
			}
			h.must(w.Close())
		}
		h.must(p.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPRationalPermutationSparseAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
			failures, successes := 0, 0
			for fail := int64(0); fail < 100; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
					b := h.pair(strings.Repeat("9876543210", 100)+"/11", "0")
					p, err := lpTestRationalPermutation(h.m, 3, []uint32{2, 0, 1})
					h.must(err)
					w := NewIndexedRationalVector(h.m, 3)
					h.must(w.SetValue(&a.X, 2))
					h.must(w.SetValue(&b.X, 0))
					h.budget.FailAfter(fail)
					switch op {
					case "left":
						err = p.ApplyLeftIndexed(w)
					case "right":
						err = p.ApplyRightIndexed(w)
					case "reverse-left":
						err = p.ApplyReverseLeftIndexed(w)
					case "reverse-right":
						err = p.ApplyReverseRightIndexed(w)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						// For P=[2,0,1] and w=[b,0,a], these two literal
						// products also fix the source insertion order.
						aWant, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
						bWant, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/11")
						want := []*big.Rat{aWant, bWant, big.NewRat(0, 1)}
						indices := []uint32{0, 1}
						if op == "right" || op == "reverse-left" {
							want = []*big.Rat{big.NewRat(0, 1), aWant, bWant}
							indices = []uint32{1, 2}
						}
						for i := range want {
							lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
						}
						if !reflect.DeepEqual(w.Index, indices) {
							t.Fatalf("index order: got %v want %v", w.Index, indices)
						}
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatalf("unexpected allocation failure: %v", err)
						}
						failures++
					}
					if p.scratch != nil {
						if e := p.Resize(3); e != ErrExactVectorPending {
							t.Fatal("stopped lexical owner", e)
						}
					}
					// Destination numerical roots belong to w; lexical copies and preliminary
					// dense/debug arithmetic belong to p. Neither owner can consume the other.
					h.must(p.Close())
					h.must(w.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete fault campaign", op, failures, successes)
			}
		}
	}
}

func TestLPRationalPermutationSparseGuardAndMutationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	value := h.pair("3/7", "0")
	for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
		p, err := lpTestRationalPermutation(h.m, 3, []uint32{2, 0, 1})
		h.must(err)
		w := NewIndexedRationalVector(h.m, 3)
		h.must(w.SetValue(&value.X, 0))
		w.Index = append(w.Index, 99)
		var e error
		switch op {
		case "left":
			e = p.ApplyLeftIndexed(w)
		case "right":
			e = p.ApplyRightIndexed(w)
		case "reverse-left":
			e = p.ApplyReverseLeftIndexed(w)
		case "reverse-right":
			e = p.ApplyReverseRightIndexed(w)
		}
		if e != ErrBounds || !reflect.DeepEqual(w.Index, []uint32{0, 99}) {
			t.Fatal(op, e, w.Index)
		}
		lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(3, 7))
		// A failed sparse call retains its lexical copies. Every mutating
		// numeric entry point must reject reuse until that owner is closed.
		for _, stopped := range []func() error{
			func() error { return p.Init(3) }, func() error { return p.Resize(3) },
			func() error { return p.ApplyLeft(nil) }, func() error { return p.ApplyRight(nil) },
			func() error { return p.ApplyReverseLeftToT(nil) }, func() error { return p.ApplyReverseLeftToX(nil) },
			func() error { return p.ApplyReverseRightToT(nil) }, func() error { return p.ApplyReverseRightToX(nil) },
			func() error { return p.ApplyLeftIndexed(w) }, func() error { return p.ApplyRightIndexed(w) },
			func() error { return p.ApplyReverseLeftIndexed(w) }, func() error { return p.ApplyReverseRightIndexed(w) },
		} {
			if err := stopped(); err != ErrExactVectorPending {
				t.Fatal(op, err)
			}
		}
		h.must(p.Close())
		h.must(w.Close())
	}
	// A stale shrunk mapping fails after sparse data clearing and the first
	// descending replacement, retaining the exact completed source prefix.
	p, err := lpTestRationalPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	h.must(p.Resize(2))
	w := NewIndexedRationalVector(h.m, 2)
	h.must(w.SetValue(&value.X, 0))
	h.must(w.SetValue(&value.X, 1))
	if err := p.ApplyReverseLeftIndexed(w); err != ErrBounds {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Index, []uint32{0, 0}) {
		t.Fatal(w.Index)
	}
	lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(3, 7))
	lpCheckExactPermutationCoordinate(t, h, &w.Data[1], big.NewRat(0, 1))
	h.must(p.Close())
	h.must(w.Close())
	// Source debug compares dense data including values absent from the index.
	p, err = lpTestRationalPermutation(h.m, 2, []uint32{1, 0})
	h.must(err)
	w = NewIndexedRationalVector(h.m, 2)
	h.must(w.copySlot(&w.Data[0], &value.X))
	err = p.ApplyRightIndexed(w)
	if err != map[bool]error{true: ErrAssertion, false: nil}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(p.Close())
	h.must(w.Close())
	// Manager equality is checked before copying any exact numerical slots.
	other := newLPPairHarness(t)
	p = NewRationalPermutation(h.m, 1)
	w = NewIndexedRationalVector(other.m, 1)
	if err := p.ApplyLeftIndexed(w); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(p.Close())
	other.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairPermutationIndependentSparseProducts(t *testing.T) {
	h := newLPPairHarness(t)
	operands := []*RationalPair{h.pair("-1/7", "0"), h.pair("999999999999999999999999999999999999999/11", "0"), h.pair("0", "0"), h.pair("1/999999999999999999999999999999999999999", "0")}
	original := make([]*big.Rat, 4)
	for i, v := range operands {
		text, err := h.m.String(&v.X)
		h.must(err)
		original[i], _ = new(big.Rat).SetString(text)
	}
	for _, values := range lpPermutations(4) {
		p, err := lpTestPairPermutation(h.m, 4, values)
		h.must(err)
		for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
			w := NewIndexedRationalVector(h.m, 4)
			// Deliberately unsorted and includes an indexed zero coordinate.
			indices := []uint32{3, 0, 2, 1}
			for _, i := range indices {
				h.must(w.SetValue(&operands[i].X, i))
			}
			switch op {
			case "left":
				h.must(p.ApplyLeftIndexed(w))
			case "right":
				h.must(p.ApplyRightIndexed(w))
			case "reverse-left":
				h.must(p.ApplyReverseLeftIndexed(w))
			case "reverse-right":
				h.must(p.ApplyReverseRightIndexed(w))
			}
			m := lpPermutationMatrix(values)
			want := lpExactPermutationProduct(m, original, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
			for i := range want {
				lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
			}
			// Find the movement of each original basis vector through matrix products,
			// independently of the implementation's forward/reverse arrays.
			wantIndex := make([]uint32, len(indices))
			for k, j := range indices {
				basis := []*big.Rat{new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)}
				basis[j].SetInt64(1)
				image := lpExactPermutationProduct(m, basis, strings.Contains(op, "right"), strings.HasPrefix(op, "reverse"))
				for i, v := range image {
					if v.Sign() != 0 {
						wantIndex[k] = uint32(i)
					}
				}
			}
			if !reflect.DeepEqual(w.Index, wantIndex) {
				t.Fatal(op, values, w.Index, wantIndex)
			}
			if p.scratch != nil {
				t.Fatal("successful lexical owner retained")
			}
			h.must(w.Close())
		}
		h.must(p.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPPairPermutationSparseAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
			failures, successes := 0, 0
			for fail := int64(0); fail < 100; fail++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, fail), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
					b := h.pair(strings.Repeat("9876543210", 100)+"/11", "0")
					p, err := lpTestPairPermutation(h.m, 3, []uint32{2, 0, 1})
					h.must(err)
					w := NewIndexedRationalVector(h.m, 3)
					h.must(w.SetValue(&a.X, 2))
					h.must(w.SetValue(&b.X, 0))
					h.budget.FailAfter(fail)
					switch op {
					case "left":
						err = p.ApplyLeftIndexed(w)
					case "right":
						err = p.ApplyRightIndexed(w)
					case "reverse-left":
						err = p.ApplyReverseLeftIndexed(w)
					case "reverse-right":
						err = p.ApplyReverseRightIndexed(w)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						// For P=[2,0,1] and w=[b,0,a], these two literal
						// products also fix the source insertion order.
						aWant, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
						bWant, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/11")
						want := []*big.Rat{aWant, bWant, big.NewRat(0, 1)}
						indices := []uint32{0, 1}
						if op == "right" || op == "reverse-left" {
							want = []*big.Rat{big.NewRat(0, 1), aWant, bWant}
							indices = []uint32{1, 2}
						}
						for i := range want {
							lpCheckExactPermutationCoordinate(t, h, &w.Data[i], want[i])
						}
						if !reflect.DeepEqual(w.Index, indices) {
							t.Fatalf("index order: got %v want %v", w.Index, indices)
						}
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatalf("unexpected allocation failure: %v", err)
						}
						failures++
					}
					if p.scratch != nil {
						if e := p.Resize(3); e != ErrExactVectorPending {
							t.Fatal("stopped lexical owner", e)
						}
					}
					// Destination numerical roots belong to w; lexical copies and preliminary
					// dense/debug arithmetic belong to p. Neither owner can consume the other.
					h.must(p.Close())
					h.must(w.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 {
				t.Fatal("incomplete fault campaign", op, failures, successes)
			}
		}
	}
}

func TestLPPairPermutationSparseGuardAndMutationPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	value := h.pair("3/7", "0")
	for _, op := range []string{"left", "right", "reverse-left", "reverse-right"} {
		p, err := lpTestPairPermutation(h.m, 3, []uint32{2, 0, 1})
		h.must(err)
		w := NewIndexedRationalVector(h.m, 3)
		h.must(w.SetValue(&value.X, 0))
		w.Index = append(w.Index, 99)
		var e error
		switch op {
		case "left":
			e = p.ApplyLeftIndexed(w)
		case "right":
			e = p.ApplyRightIndexed(w)
		case "reverse-left":
			e = p.ApplyReverseLeftIndexed(w)
		case "reverse-right":
			e = p.ApplyReverseRightIndexed(w)
		}
		if e != ErrBounds || !reflect.DeepEqual(w.Index, []uint32{0, 99}) {
			t.Fatal(op, e, w.Index)
		}
		lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(3, 7))
		// A failed sparse call retains its lexical copies. Every mutating
		// numeric entry point must reject reuse until that owner is closed.
		for _, stopped := range []func() error{
			func() error { return p.Init(3) }, func() error { return p.Resize(3) },
			func() error { return p.ApplyLeft(nil) }, func() error { return p.ApplyRight(nil) },
			func() error { return p.ApplyReverseLeftToT(nil) }, func() error { return p.ApplyReverseLeftToX(nil) },
			func() error { return p.ApplyReverseRightToT(nil) }, func() error { return p.ApplyReverseRightToX(nil) },
			func() error { return p.ApplyLeftIndexed(w) }, func() error { return p.ApplyRightIndexed(w) },
			func() error { return p.ApplyReverseLeftIndexed(w) }, func() error { return p.ApplyReverseRightIndexed(w) },
		} {
			if err := stopped(); err != ErrExactVectorPending {
				t.Fatal(op, err)
			}
		}
		h.must(p.Close())
		h.must(w.Close())
	}
	// A stale shrunk mapping fails after sparse data clearing and the first
	// descending replacement, retaining the exact completed source prefix.
	p, err := lpTestPairPermutation(h.m, 3, []uint32{2, 0, 1})
	h.must(err)
	h.must(p.Resize(2))
	w := NewIndexedRationalVector(h.m, 2)
	h.must(w.SetValue(&value.X, 0))
	h.must(w.SetValue(&value.X, 1))
	if err := p.ApplyReverseLeftIndexed(w); err != ErrBounds {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Index, []uint32{0, 0}) {
		t.Fatal(w.Index)
	}
	lpCheckExactPermutationCoordinate(t, h, &w.Data[0], big.NewRat(3, 7))
	lpCheckExactPermutationCoordinate(t, h, &w.Data[1], big.NewRat(0, 1))
	h.must(p.Close())
	h.must(w.Close())
	// Source debug compares dense data including values absent from the index.
	p, err = lpTestPairPermutation(h.m, 2, []uint32{1, 0})
	h.must(err)
	w = NewIndexedRationalVector(h.m, 2)
	h.must(w.copySlot(&w.Data[0], &value.X))
	err = p.ApplyRightIndexed(w)
	if err != map[bool]error{true: ErrAssertion, false: nil}[sourceDebugTests] {
		t.Fatal(err)
	}
	h.must(p.Close())
	h.must(w.Close())
	// Manager equality is checked before copying any exact numerical slots.
	other := newLPPairHarness(t)
	p = NewPairPermutation(h.m, 1)
	w = NewIndexedRationalVector(other.m, 1)
	if err := p.ApplyLeftIndexed(w); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	h.must(p.Close())
	other.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPermutationAllDenseGuardProfiles(t *testing.T) {
	h := newLPPairHarness(t)
	fixture := h.pair("3/7", "-2/11")
	for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
		for _, world := range []string{"short", "long", "stale", "duplicate-short"} {
			p := NewRationalPermutation(h.m, 2)
			n := uint32(2)
			switch world {
			case "short":
				n = 1
			case "long":
				n = 3
			case "stale":
				h.must(p.Init(3))
				h.must(p.Set(0, 2))
				h.must(p.Set(1, 0))
				h.must(p.Set(2, 1))
				h.must(p.Resize(2))
			case "duplicate-short":
				n = 1
				h.must(p.Set(0, 0))
				h.must(p.Set(1, 0))
			}
			w := NewIndexedRationalVector(h.m, n)
			x := NewIndexedRationalVector(h.m, n)
			for i := uint32(0); i < n; i++ {
				h.must(w.copySlot(&w.Data[i], &fixture.X))
				h.must(x.copySlot(&x.Data[i], &fixture.X))
			}
			var err error
			switch op {
			case "left":
				err = p.ApplyLeft(x.Data)
			case "right":
				err = p.ApplyRight(w.Data)
			case "reverse-left-T":
				err = p.ApplyReverseLeftToT(w.Data)
			case "reverse-left-X":
				err = p.ApplyReverseLeftToX(x.Data)
			case "reverse-right-T":
				err = p.ApplyReverseRightToT(w.Data)
			case "reverse-right-X":
				err = p.ApplyReverseRightToX(x.Data)
			}
			want := ErrBounds
			if world != "stale" && sourceDebugTests {
				want = ErrAssertion
			} else if world == "long" {
				want = nil
			}
			if err != want {
				t.Fatal(op, world, err, want)
			}
			if world == "long" {
				lpCheckExactPermutationCoordinate(t, h, &w.Data[2], big.NewRat(3, 7))
				lpCheckExactPermutationCoordinate(t, h, &x.Data[2], big.NewRat(3, 7))
			}
			h.must(p.Close())
			h.must(w.Close())
			h.must(x.Close())
		}
	}
	p := NewRationalPermutation(h.m, 2)
	h.must(p.Set(0, 1))
	h.must(p.Set(1, 0))
	h.must(p.Resize(4))
	if !reflect.DeepEqual(p.Values(), []uint32{1, 0, 2, 3}) {
		t.Fatal(p.Values())
	}
	h.must(p.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairPermutationAllDenseGuardProfiles(t *testing.T) {
	h := newLPPairHarness(t)
	fixture := h.pair("3/7", "-2/11")
	for _, op := range []string{"left", "right", "reverse-left-T", "reverse-left-X", "reverse-right-T", "reverse-right-X"} {
		for _, world := range []string{"short", "long", "stale", "duplicate-short"} {
			p := NewPairPermutation(h.m, 2)
			n := uint32(2)
			switch world {
			case "short":
				n = 1
			case "long":
				n = 3
			case "stale":
				h.must(p.Init(3))
				h.must(p.Set(0, 2))
				h.must(p.Set(1, 0))
				h.must(p.Set(2, 1))
				h.must(p.Resize(2))
			case "duplicate-short":
				n = 1
				h.must(p.Set(0, 0))
				h.must(p.Set(1, 0))
			}
			w := NewIndexedRationalVector(h.m, n)
			x := NewIndexedPairVector(h.m, n)
			for i := uint32(0); i < n; i++ {
				h.must(w.copySlot(&w.Data[i], &fixture.X))
				h.must(x.copySlot(&x.Data[i], fixture))
			}
			var err error
			switch op {
			case "left":
				err = p.ApplyLeft(x.Data)
			case "right":
				err = p.ApplyRight(w.Data)
			case "reverse-left-T":
				err = p.ApplyReverseLeftToT(w.Data)
			case "reverse-left-X":
				err = p.ApplyReverseLeftToX(x.Data)
			case "reverse-right-T":
				err = p.ApplyReverseRightToT(w.Data)
			case "reverse-right-X":
				err = p.ApplyReverseRightToX(x.Data)
			}
			want := ErrBounds
			if world != "stale" && sourceDebugTests {
				want = ErrAssertion
			} else if world == "long" {
				want = nil
			}
			if err != want {
				t.Fatal(op, world, err, want)
			}
			if world == "long" {
				lpCheckExactPermutationCoordinate(t, h, &w.Data[2], big.NewRat(3, 7))
				lpCheckExactPermutationCoordinate(t, h, &x.Data[2].X, big.NewRat(3, 7))
			}
			h.must(p.Close())
			h.must(w.Close())
			h.must(x.Close())
		}
	}
	p := NewPairPermutation(h.m, 2)
	h.must(p.Set(0, 1))
	h.must(p.Set(1, 0))
	h.must(p.Resize(4))
	if !reflect.DeepEqual(p.Values(), []uint32{1, 0, 2, 3}) {
		t.Fatal(p.Values())
	}
	h.must(p.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPermutationStoppedDestinationOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	w := NewIndexedRationalVector(h.m, 1)
	h.must(w.SetValue(&a.X, 0))
	h.budget.FailAfter(0)
	err := w.Add(0, &b.X)
	h.budget.FailAfter(-1)
	if err == nil || !w.calls.pending() {
		t.Fatal("missing actual stopped destination", err)
	}
	state := h.m.RawInvocationState()
	p := NewRationalPermutation(h.m, 1)
	for _, apply := range []func(*IndexedRationalVector) error{p.ApplyLeftIndexed, p.ApplyRightIndexed, p.ApplyReverseLeftIndexed, p.ApplyReverseRightIndexed} {
		if err := apply(w); err != ErrExactVectorPending {
			t.Fatal(err)
		}
		if p.scratch != nil {
			t.Fatal("claimed destination ownership")
		}
	}
	h.must(p.Close())
	if got := h.m.RawInvocationState(); got != state {
		t.Fatal("consumed destination raw frame", got, state)
	}
	h.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairPermutationStoppedDestinationOwnership(t *testing.T) {
	h := newLPPairHarness(t)
	a := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
	b := h.pair(strings.Repeat("9876543210", 100)+"/13", "0")
	w := NewIndexedRationalVector(h.m, 1)
	h.must(w.SetValue(&a.X, 0))
	h.budget.FailAfter(0)
	err := w.Add(0, &b.X)
	h.budget.FailAfter(-1)
	if err == nil || !w.calls.pending() {
		t.Fatal("missing actual stopped destination", err)
	}
	state := h.m.RawInvocationState()
	p := NewPairPermutation(h.m, 1)
	for _, apply := range []func(*IndexedRationalVector) error{p.ApplyLeftIndexed, p.ApplyRightIndexed, p.ApplyReverseLeftIndexed, p.ApplyReverseRightIndexed} {
		if err := apply(w); err != ErrExactVectorPending {
			t.Fatal(err)
		}
		if p.scratch != nil {
			t.Fatal("claimed destination ownership")
		}
	}
	h.must(p.Close())
	if got := h.m.RawInvocationState(); got != state {
		t.Fatal("consumed destination raw frame", got, state)
	}
	h.must(w.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
