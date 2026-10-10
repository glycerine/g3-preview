package lp

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/numeral"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func lpCheckActiveQueue(t *testing.T, q *UintPairPriorityQueue, want map[IndexPair]uint32, n uint32) {
	t.Helper()
	if int(q.Size()) != len(want) {
		t.Fatalf("queue size %d want %d", q.Size(), len(want))
	}
	for i := uint32(0); i < n; i++ {
		for j := uint32(0); j < n; j++ {
			pair := IndexPair{i, j}
			score, present := want[pair]
			if q.Contains(i, j) != present {
				t.Fatalf("queue membership %v", pair)
			}
			if present {
				got, err := q.Priority(i, j)
				lpStaticMust(t, err)
				if got != score {
					t.Fatalf("score %v = %d want %d", pair, got, score)
				}
			}
		}
	}
}

func TestLPSparseActiveShorteningIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(771, 911))
	for world := 0; world < 160; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("1", "0")
			n := uint32(2 + rng.IntN(5))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			cells := map[IndexPair]bool{}
			rnz := make([]uint32, n)
			cnz := make([]uint32, n)
			for _, off := range rng.Perm(int(n * n)) {
				i, j := uint32(off)/n, uint32(off)%n
				if i != j && rng.IntN(3) == 0 {
					continue
				}
				h.must(m.AddNewElement(i, j, &value.X))
				lpStaticMust(t, f.AddNewElement(i, j, 1))
				cells[IndexPair{i, j}] = true
				rnz[i]++
				cnz[j]++
			}
			// Use the same independent physical permutation for rows and columns so each
			// logical diagonal remains present while the two mapping directions differ.
			logical := make([]uint32, n)
			inverse := make([]uint32, n)
			for i := range logical {
				logical[i] = uint32(i)
			}
			for k := 0; k < 8; k++ {
				a := rng.Uint32N(n)
				b := (a + 1 + rng.Uint32N(n-1)) % n
				h.must(m.SwapRows(a, b))
				h.must(m.SwapColumns(a, b))
				lpStaticMust(t, f.SwapRows(a, b))
				lpStaticMust(t, f.SwapColumns(a, b))
				logical[a], logical[b] = logical[b], logical[a]
			}
			for i, p := range logical {
				inverse[p] = uint32(i)
			}
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			queue := map[IndexPair]uint32{}
			for p := range cells {
				queue[p] = rnz[p.I] * (cnz[p.J] - 1)
			}
			shortened := make([]uint32, n)
			active := uint32(len(cells))
			for k := uint32(0); k < n; k++ {
				var eta *RationalEtaMatrix
				var feta *FloatEtaMatrix
				var touched []uint32
				if world%2 == 0 {
					eta = NewRationalEtaMatrix(h.m, k, n)
					feta = NewFloatEtaMatrix(k, n)
					for i := k + 1; i < n; i++ {
						if rng.IntN(2) == 0 {
							h.must(eta.Append(i, &value.X))
							lpStaticMust(t, feta.Append(i, 1))
							touched = append(touched, logical[i])
						}
					}
				}
				p := logical[k]
				for cell := range cells {
					if cell.I == p {
						delete(queue, cell)
						active--
						if inverse[cell.J] > k {
							shortened[cell.J]++
						}
					}
				}
				for cell := range queue {
					if cell.J == p && inverse[cell.I] >= k {
						delete(queue, cell)
					}
				}
				// Derive the affected coordinate domain from the original incidence map,
				// independently of the matrix's column links and queue heap layout.
				affected := map[uint32]bool{}
				for cell := range cells {
					if cell.I == p {
						affected[cell.J] = true
					}
				}
				for cell := range cells {
					if inverse[cell.I] > k && affected[cell.J] {
						queue[cell] = rnz[cell.I] * (cnz[cell.J] - shortened[cell.J] - 1)
					}
				}
				for _, r := range touched {
					for cell := range cells {
						if cell.I == r {
							queue[cell] = rnz[r] * (cnz[cell.J] - shortened[cell.J] - 1)
						}
					}
				}
				ok, err := m.ShortenActiveMatrix(k, eta)
				h.must(err)
				if !ok {
					t.Fatal("unexpected exact zero column")
				}
				ok, err = f.ShortenActiveMatrix(k, feta)
				lpStaticMust(t, err)
				if !ok {
					t.Fatal("unexpected float zero column")
				}
				if m.active != active || f.active != active {
					t.Fatal("active count", m.active, f.active, active)
				}
				for j := range shortened {
					if m.Columns[j].ShortenedMarkovitz != shortened[j] || f.Columns[j].ShortenedMarkovitz != shortened[j] {
						t.Fatal("shortened column", j)
					}
				}
				lpCheckActiveQueue(t, m.pivotQueue, queue, n)
				lpCheckActiveQueue(t, f.pivotQueue, queue, n)
				if eta != nil {
					h.must(eta.Close())
				}
			}
			// Shortening owns no coefficient copies and leaves stored values/links intact.
			for i := range m.Rows {
				for j := range m.Rows[i] {
					v := &m.Rows[i][j]
					other := &m.Columns[v.Index].Values[v.Other]
					if other.Index != uint32(i) || other.Other != uint32(j) {
						t.Fatal("backlink changed")
					}
					lpSparseRationalCheck(t, h, &v.Value, big.NewRat(1, 1))
				}
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseActiveLiteralFailurePrefixes(t *testing.T) {
	// The first empty remaining column stops before visiting the rest of the row,
	// removing the pivot column, or restoring any scores.
	h := newLPPairHarness(t)
	value := h.pair("1", "0")
	m := lpNewExactSparse(t, h, 3)
	f := newFloatSparseMatrix(3)
	for _, p := range []IndexPair{{0, 1}, {0, 2}, {0, 0}, {1, 0}, {2, 2}} {
		h.must(m.AddNewElement(p.I, p.J, &value.X))
		lpStaticMust(t, f.AddNewElement(p.I, p.J, 1))
	}
	h.must(m.PrepareForFactorization())
	lpStaticMust(t, f.PrepareForFactorization())
	for _, exact := range []bool{false, true} {
		var ok bool
		var err error
		var active uint32
		var q *UintPairPriorityQueue
		var shortened uint32
		if exact {
			ok, err = m.ShortenActiveMatrix(0, nil)
			active = m.active
			q = m.pivotQueue
			shortened = m.Columns[1].ShortenedMarkovitz
		} else {
			ok, err = f.ShortenActiveMatrix(0, nil)
			active = f.active
			q = f.pivotQueue
			shortened = f.Columns[1].ShortenedMarkovitz
		}
		lpStaticMust(t, err)
		if ok || active != 4 || shortened != 1 || q.Size() != 4 || q.Contains(0, 1) || !q.Contains(0, 0) || !q.Contains(0, 2) || !q.Contains(1, 0) {
			t.Fatal("zero column prefix", exact, ok, active, shortened)
		}
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPSparseActiveEtaRequeuesInactiveRows(t *testing.T) {
	// eta's final refresh has no active-row/column filter. It can reinsert a pair
	// removed earlier in the same shortening operation.
	f := newFloatSparseMatrix(2)
	for i := uint32(0); i < 2; i++ {
		for j := uint32(0); j < 2; j++ {
			lpStaticMust(t, f.AddNewElement(i, j, 1))
		}
	}
	lpStaticMust(t, f.PrepareForFactorization())
	eta := NewFloatEtaMatrix(1, 2)
	lpStaticMust(t, eta.Append(0, 0))
	ok, err := f.ShortenActiveMatrix(0, eta)
	lpStaticMust(t, err)
	if !ok || !f.pivotQueue.Contains(0, 0) || !f.pivotQueue.Contains(0, 1) {
		t.Fatal("inactive eta row filtered")
	}
	lpCheckActiveQueue(t, f.pivotQueue, map[IndexPair]uint32{{0, 0}: 2, {0, 1}: 0, {1, 0}: 2, {1, 1}: 0}, 2)
}

func TestLPSparseActiveSignedScoreProfiles(t *testing.T) {
	// Signed negative score assertions are checked before enqueuing; release
	// continues with unsigned multiplication and wrap.
	for _, etaStage := range []bool{false, true} {
		t.Run(fmt.Sprint(etaStage), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("1", "0")
			m := lpNewExactSparse(t, h, 2)
			f := newFloatSparseMatrix(2)
			for _, p := range []IndexPair{{0, 0}, {1, 0}} {
				h.must(m.AddNewElement(p.I, p.J, &value.X))
				lpStaticMust(t, f.AddNewElement(p.I, p.J, 1))
			}
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			m.Columns[0].ShortenedMarkovitz = 2
			f.Columns[0].ShortenedMarkovitz = 2
			if etaStage {
				eta := NewRationalEtaMatrix(h.m, 0, 2)
				feta := NewFloatEtaMatrix(0, 2)
				h.must(eta.Append(1, &value.X))
				lpStaticMust(t, feta.Append(1, 1))
				// An empty pivot row skips the earlier score refresh and reaches eta's copy.
				// Clear row0 only after preserving its owned numeric cells for cleanup.
				saved := m.Rows[0]
				m.Rows[0] = nil
				f.Rows[0] = nil
				_, err := m.ShortenActiveMatrix(0, eta)
				_, ferr := f.ShortenActiveMatrix(0, feta)
				m.Rows[0] = saved
				if sourceDebugTests {
					if err != ErrAssertion || ferr != ErrAssertion {
						t.Fatal("eta signed assertion", err, ferr)
					}
				} else {
					h.must(err)
					lpStaticMust(t, ferr)
					score, e := m.pivotQueue.Priority(1, 0)
					h.must(e)
					if score != ^uint32(0) {
						t.Fatal("eta wrapped score", score)
					}
				}
				h.must(eta.Close())
			} else {
				err := m.updateActivePivots(0)
				ferr := f.updateActivePivots(0)
				if sourceDebugTests {
					if err != ErrAssertion || ferr != ErrAssertion {
						t.Fatal("signed assertion", err, ferr)
					}
				} else {
					h.must(err)
					lpStaticMust(t, ferr)
					score, e := m.pivotQueue.Priority(1, 0)
					h.must(e)
					if score != ^uint32(0) {
						t.Fatal("wrapped score", score)
					}
				}
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseActiveEtaEliminationIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(1177, 293))
	for world := 0; world < 100; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			v := h.pair("0", "0")
			n := uint32(2 + rng.IntN(4))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			a := make([][]*big.Rat, n)
			for i := uint32(0); i < n; i++ {
				a[i] = make([]*big.Rat, n)
				for j := uint32(0); j < n; j++ {
					value := int64(1 + rng.IntN(4))
					if i == j {
						value += 100
					}
					a[i][j] = big.NewRat(value, 1)
					h.must(h.m.SetString(&v.X, a[i][j].RatString()))
					h.must(m.AddNewElement(i, j, &v.X))
					lpStaticMust(t, f.AddNewElement(i, j, float64(value)))
				}
			}
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			eta := NewRationalEtaMatrix(h.m, 0, n)
			feta := NewFloatEtaMatrix(0, n)
			h.must(h.m.SetString(&v.X, a[0][0].RatString()))
			ok, err := eta.SetDiagonal(&v.X)
			h.must(err)
			if !ok || !feta.SetDiagonal(float64(a[0][0].Num().Int64())) {
				t.Fatal("diagonal")
			}
			for i := uint32(1); i < n; i++ {
				alpha := new(big.Rat).Neg(new(big.Rat).Quo(a[i][0], a[0][0]))
				h.must(h.m.SetString(&v.X, alpha.RatString()))
				h.must(eta.Append(i, &v.X))
				af, _ := alpha.Float64()
				lpStaticMust(t, feta.Append(i, af))
				for j := n; j > 0; {
					j--
					a[i][j] = new(big.Rat).Add(a[i][j], new(big.Rat).Mul(alpha, a[0][j]))
				}
			}
			pivot := new(big.Rat).Set(a[0][0])
			for j := range a[0] {
				a[0][j] = new(big.Rat).Quo(a[0][j], pivot)
			}
			ok, err = m.PivotWithEta(0, eta, &Settings{DropTolerance: 1e-14, ZeroTolerance: 1e-12})
			h.must(err)
			if !ok {
				t.Fatal("exact pivot")
			}
			ok, err = f.PivotWithEta(0, feta, &Settings{DropTolerance: 1e-14, ZeroTolerance: 1e-12})
			lpStaticMust(t, err)
			if !ok {
				t.Fatal("float pivot")
			}
			for i := uint32(0); i < n; i++ {
				for j := uint32(0); j < n; j++ {
					x, e := m.Get(i, j)
					h.must(e)
					lpSparseRationalCheck(t, h, x, a[i][j])
					y, e := f.Get(i, j)
					lpStaticMust(t, e)
					want, _ := a[i][j].Float64()
					if math.Abs(y-want) > 1e-12 {
						t.Fatal("floating elimination", i, j, y, want)
					}
				}
			}
			if m.active != (n-1)*(n-1) || f.active != m.active {
				t.Fatal("elimination active count")
			}
			wantQueue := map[IndexPair]uint32{}
			for i := uint32(1); i < n; i++ {
				// Eliminated entries retain their old queued candidates until
				// GetPivotForColumn discovers they are absent from the row.
				wantQueue[IndexPair{i, 0}] = n * (n - 1)
				for j := uint32(1); j < n; j++ {
					wantQueue[IndexPair{i, j}] = (n - 1) * (n - 2)
				}
			}
			lpCheckActiveQueue(t, m.pivotQueue, wantQueue, n)
			lpCheckActiveQueue(t, f.pivotQueue, wantQueue, n)
			h.must(eta.Close())
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseActiveEtaEarlyEmptyTarget(t *testing.T) {
	h := newLPPairHarness(t)
	v := h.pair("1", "0")
	m := lpNewExactSparse(t, h, 2)
	f := newFloatSparseMatrix(2)
	for i := uint32(0); i < 2; i++ {
		for j := uint32(0); j < 2; j++ {
			h.must(m.AddNewElement(i, j, &v.X))
			lpStaticMust(t, f.AddNewElement(i, j, 1))
		}
	}
	h.must(m.PrepareForFactorization())
	lpStaticMust(t, f.PrepareForFactorization())
	eta := NewRationalEtaMatrix(h.m, 0, 2)
	feta := NewFloatEtaMatrix(0, 2)
	h.must(h.m.SetString(&v.X, "2"))
	_, err := eta.SetDiagonal(&v.X)
	h.must(err)
	feta.SetDiagonal(2)
	h.must(h.m.SetString(&v.X, "-1"))
	h.must(eta.Append(1, &v.X))
	lpStaticMust(t, feta.Append(1, -1))
	ok, err := m.PivotWithEta(0, eta, &Settings{DropTolerance: 1e-14, ZeroTolerance: 1e-12})
	h.must(err)
	if ok {
		t.Fatal("empty row succeeded")
	}
	ok, err = f.PivotWithEta(0, feta, &Settings{DropTolerance: 1e-14, ZeroTolerance: 1e-12})
	lpStaticMust(t, err)
	if ok {
		t.Fatal("float empty row succeeded")
	}
	// The pivot row was never divided and shortening never ran.
	x, err := m.Get(0, 0)
	h.must(err)
	lpSparseRationalCheck(t, h, x, big.NewRat(1, 1))
	if m.active != 2 || f.active != 2 || m.pivotQueue.Size() != 4 || f.pivotQueue.Size() != 4 || m.Columns[1].ShortenedMarkovitz != 0 {
		t.Fatal("empty target continuation")
	}
	h.must(eta.Close())
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactSparseActiveEtaAllocationFailureCensus(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			failures, successes, receipts := 0, 0, 0
			for ordinal := int64(0); ordinal < 400; ordinal++ {
				done := false
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					m := lpNewExactSparse(t, h, 2)
					p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
					v := h.pair("2", "6")
					h.must(m.AddNewElement(0, 0, &v.X))
					h.must(m.AddNewElement(0, 1, &p.X))
					h.must(m.AddNewElement(1, 0, &v.Y))
					h.must(m.AddNewElement(1, 1, &p.Y))
					h.must(m.PrepareForFactorization())
					eta := NewRationalEtaMatrix(h.m, 0, 2)
					_, err := eta.SetDiagonal(&v.X)
					h.must(err)
					h.must(h.m.SetString(&v.Y, "-3"))
					h.must(eta.Append(1, &v.Y))
					h.budget.FailAfter(ordinal)
					ok, err := m.PivotWithEta(0, eta, nil)
					h.budget.FailAfter(-1)
					if err == nil {
						if !ok || m.active != 1 {
							t.Fatal("successful pivot")
						}
						x, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 100) + "/7")
						y, _ := new(big.Rat).SetString(strings.Repeat("9876543210", 100) + "/13")
						out, e := m.Get(1, 1)
						h.must(e)
						lpSparseRationalCheck(t, h, out, new(big.Rat).Sub(y, new(big.Rat).Mul(big.NewRat(3, 1), x)))
						successes++
						done = true
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if m.calls.pending() {
							receipts++
						}
						if _, e := m.ShortenActiveMatrix(0, eta); e != ErrExactVectorPending {
							t.Fatal("stopped shortening", e)
						}
						if _, e := m.PivotWithEta(0, eta, nil); e != ErrExactVectorPending {
							t.Fatal("stopped eta pivot", e)
						}
						if e := eta.ready(); e != nil {
							t.Fatal("borrowed eta adopted matrix failure", e)
						}
						if m.Columns[1].ShortenedMarkovitz != 0 {
							t.Fatal("shortened after numeric failure")
						}
					}
					h.must(m.Close())
					// Matrix cleanup must preserve the borrowed eta and its independent cells.
					lpSparseRationalCheck(t, h, &eta.Diagonal, big.NewRat(2, 1))
					lpSparseRationalCheck(t, h, &eta.Column.Data[0].Value, big.NewRat(-3, 1))
					lpExactVectorNamedCensus(t, h, []*numeral.Rational{&eta.Diagonal, &eta.Column.Data[0].Value})
					h.must(eta.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || successes != 1 || receipts == 0 {
				t.Fatal("incomplete eta failure campaign", failures, successes, receipts)
			}
		})
	}
}

func TestLPSparseActiveBoundsAndPublicationPrefixes(t *testing.T) {
	for _, phase := range []string{"remove logical", "remove physical", "remove inverse", "remove column", "pivot logical", "pivot column", "pivot inverse", "update logical", "update physical", "update column", "update inverse", "update target", "shorten pivot", "shorten update", "eta logical", "eta physical", "eta column", "divide logical"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("1", "0")
			m := lpNewExactSparse(t, h, 2)
			f := newFloatSparseMatrix(2)
			for i := uint32(0); i < 2; i++ {
				for j := uint32(0); j < 2; j++ {
					h.must(m.AddNewElement(i, j, &value.X))
					lpStaticMust(t, f.AddNewElement(i, j, 1))
				}
			}
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			savedRows := m.Rows
			savedCols := m.Columns
			var err, ferr error
			switch phase {
			case "remove logical":
				_, err = m.removeRowFromActivePivotsAndShortenColumns(2)
				_, ferr = f.removeRowFromActivePivotsAndShortenColumns(2)
			case "remove physical":
				m.Rows = nil
				f.Rows = nil
				_, err = m.removeRowFromActivePivotsAndShortenColumns(0)
				_, ferr = f.removeRowFromActivePivotsAndShortenColumns(0)
			case "remove inverse":
				m.Rows[0][0].Index = 2
				f.Rows[0][0].Index = 2
				_, err = m.ShortenActiveMatrix(0, nil)
				_, ferr = f.ShortenActiveMatrix(0, nil)
				if m.active != 3 || f.active != 3 {
					t.Fatal("inverse guard before active decrement")
				}
			case "remove column":
				m.Columns = m.Columns[:1]
				f.Columns = f.Columns[:1]
				_, err = m.ShortenActiveMatrix(0, nil)
				_, ferr = f.ShortenActiveMatrix(0, nil)
			case "pivot logical":
				err = m.removePivotColumn(2)
				ferr = f.removePivotColumn(2)
			case "pivot column":
				m.Columns = nil
				f.Columns = nil
				err = m.removePivotColumn(0)
				ferr = f.removePivotColumn(0)
			case "pivot inverse":
				m.Columns[0].Values[0].Index = 2
				f.Columns[0].Values[0].Index = 2
				err = m.removePivotColumn(0)
				ferr = f.removePivotColumn(0)
			case "update logical":
				err = m.updateActivePivots(2)
				ferr = f.updateActivePivots(2)
			case "update physical":
				m.Rows = nil
				f.Rows = nil
				err = m.updateActivePivots(0)
				ferr = f.updateActivePivots(0)
			case "update column":
				m.Rows[0][0].Index = 2
				f.Rows[0][0].Index = 2
				err = m.updateActivePivots(0)
				ferr = f.updateActivePivots(0)
			case "update inverse":
				m.Columns[0].Values[0].Index = 2
				f.Columns[0].Values[0].Index = 2
				err = m.updateActivePivots(0)
				ferr = f.updateActivePivots(0)
			case "update target":
				m.Rows = m.Rows[:1]
				f.Rows = f.Rows[:1]
				err = m.updateActivePivots(0)
				ferr = f.updateActivePivots(0)
			case "shorten pivot":
				// Make row0 empty without dropping any owned coefficients.
				m.Rows = append([][]RationalIndexedValue(nil), m.Rows...)
				m.Rows[0] = nil
				f.Rows[0] = nil
				m.Columns[0].Values[0].Index = 2
				f.Columns[0].Values[0].Index = 2
				_, err = m.ShortenActiveMatrix(0, nil)
				_, ferr = f.ShortenActiveMatrix(0, nil)
			case "shorten update":
				m.Columns[1].Values[1].Index = 2
				f.Columns[1].Values[1].Index = 2
				_, err = m.ShortenActiveMatrix(0, nil)
				_, ferr = f.ShortenActiveMatrix(0, nil)
			case "eta logical", "eta physical", "eta column", "divide logical":
				eta := NewRationalEtaMatrix(h.m, 0, 2)
				feta := NewFloatEtaMatrix(0, 2)
				if phase == "divide logical" {
					_, e := eta.SetDiagonal(&value.X)
					h.must(e)
					feta.SetDiagonal(1)
					_, err = m.PivotWithEta(2, eta, nil)
					_, ferr = f.PivotWithEta(2, feta, &Settings{})
				} else {
					row := uint32(1)
					if phase == "eta logical" {
						row = 2
					}
					h.must(eta.Append(row, &value.X))
					lpStaticMust(t, feta.Append(row, 1))
					m.Rows = append([][]RationalIndexedValue(nil), m.Rows...)
					m.Rows[0] = nil
					f.Rows[0] = nil
					// Empty pivot column lets the malformed eta row reach the final refresh.
					m.Columns = append([]RationalSparseColumn(nil), m.Columns...)
					m.Columns[0].Values = nil
					f.Columns[0].Values = nil
					if phase == "eta physical" {
						m.Rows = m.Rows[:1]
						f.Rows = f.Rows[:1]
					}
					if phase == "eta column" {
						m.Rows[1][0].Index = 2
						f.Rows[1][0].Index = 2
					}
					_, err = m.ShortenActiveMatrix(0, eta)
					_, ferr = f.ShortenActiveMatrix(0, feta)
				}
				h.must(eta.Close())
			}
			if err != ErrBounds || ferr != ErrBounds {
				t.Fatal("bounds", phase, err, ferr)
			}
			m.Rows = savedRows
			m.Columns = savedCols
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSparseActiveBorrowedEtaPendingAndNoCopies(t *testing.T) {
	h := lpNewDirectSparseHarness(t)
	m := lpNewExactSparse(t, h, 2)
	p := h.pair(strings.Repeat("1234567890", 100)+"/7", "1")
	for i := uint32(0); i < 2; i++ {
		for j := uint32(0); j < 2; j++ {
			h.must(m.AddNewElement(i, j, &p.X))
		}
	}
	h.must(m.PrepareForFactorization())
	eta := NewRationalEtaMatrix(h.m, 0, 2)
	h.budget.FailAfter(0)
	err := eta.Append(1, &p.X)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(err) || eta.ready() != ErrExactVectorPending {
		t.Fatal("expected real eta copy refusal", err)
	}
	if _, err = m.ShortenActiveMatrix(0, eta); err != ErrExactVectorPending {
		t.Fatal("borrowed stopped eta shortening", err)
	}
	if _, err = m.PivotWithEta(0, eta, nil); err != ErrExactVectorPending {
		t.Fatal("borrowed stopped eta pivot", err)
	}
	if m.active != 4 || m.pivotQueue.Size() != 4 || m.Columns[1].ShortenedMarkovitz != 0 || m.work != nil || m.calls.pending() {
		t.Fatal("borrowed eta changed matrix")
	}
	h.must(eta.Close())
	// Actual large coefficients would require allocator reservations if copied.
	// Shortening is entirely reference/index work and succeeds with none available.
	h.budget.FailAfter(0)
	ok, err := m.ShortenActiveMatrix(0, nil)
	h.budget.FailAfter(-1)
	h.must(err)
	if !ok || m.active != 2 || m.calls.pending() || m.work != nil {
		t.Fatal("shortening copied coefficients")
	}
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
