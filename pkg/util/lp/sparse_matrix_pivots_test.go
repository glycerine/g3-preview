package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPSparsePivotPreparationIndependentOracle(t *testing.T) {
	rng := rand.New(rand.NewPCG(109, 886))
	for world := 0; world < 120; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			p := h.pair("0", "0")
			n := uint32(1 + rng.IntN(5))
			m := lpNewExactSparse(t, h, n)
			f := newFloatSparseMatrix(n)
			want := make(map[IndexPair]*big.Rat)
			fwant := make(map[IndexPair]float64)
			heads := make([]uint32, n)
			maxima := make([]float64, n)
			rowCounts := make([]uint32, n)
			colCounts := make([]uint32, n)
			for _, offset := range rng.Perm(int(n * n)) {
				if rng.IntN(3) == 0 {
					continue
				}
				i, j := uint32(offset)/n, uint32(offset)%n
				value := math.Ldexp(float64(rng.IntN(19)-9), rng.IntN(100)-50)
				x := new(big.Rat).SetFloat64(value)
				h.must(h.m.SetString(&p.X, x.RatString()))
				h.must(m.AddNewElement(i, j, &p.X))
				lpStaticMust(t, f.AddNewElement(i, j, value))
				want[IndexPair{i, j}] = x
				fwant[IndexPair{i, j}] = value
				if rowCounts[i] == 0 || math.Abs(value) > maxima[i] {
					heads[i] = j
					maxima[i] = math.Abs(value)
				}
				rowCounts[i]++
				colCounts[j]++
			}
			rows, cols := make([]uint32, n), make([]uint32, n)
			for i := range rows {
				rows[i] = uint32(i)
				cols[i] = uint32(i)
				m.Columns[i].ShortenedMarkovitz = 17
				f.Columns[i].ShortenedMarkovitz = 17
			}
			if n > 1 {
				for k := 0; k < 3; k++ {
					a := rng.Uint32N(n)
					b := (a + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapRows(a, b))
					lpStaticMust(t, f.SwapRows(a, b))
					rows[a], rows[b] = rows[b], rows[a]
					a = rng.Uint32N(n)
					b = (a + 1 + rng.Uint32N(n-1)) % n
					h.must(m.SwapColumns(a, b))
					lpStaticMust(t, f.SwapColumns(a, b))
					cols[a], cols[b] = cols[b], cols[a]
				}
			}
			h.must(m.PrepareForFactorization())
			lpStaticMust(t, f.PrepareForFactorization())
			lpExactSparseCheck(t, h, m, want)
			lpFloatSparseCheck(t, f, fwant)
			for i := range heads {
				if m.Columns[i].ShortenedMarkovitz != 0 || f.Columns[i].ShortenedMarkovitz != 0 {
					t.Fatal("counter reset")
				}
				if rowCounts[i] > 0 && (m.Rows[i][0].Index != heads[i] || f.Rows[i][0].Index != heads[i]) {
					t.Fatal("stable maximum")
				}
			}
			for pair := range want {
				expected := rowCounts[pair.I] * (colCounts[pair.J] - 1)
				got, e := m.pivotQueue.Priority(pair.I, pair.J)
				h.must(e)
				if got != expected {
					t.Fatal("exact score", pair, got, expected)
				}
				got, e = f.pivotQueue.Priority(pair.I, pair.J)
				lpStaticMust(t, e)
				if got != expected {
					t.Fatal("float score")
				}
			}
			if int(m.pivotQueue.Size()) != len(want) || int(f.pivotQueue.Size()) != len(want) {
				t.Fatal("queue domain")
			}
			inverseRows, inverseCols := make([]uint32, n), make([]uint32, n)
			for i := range rows {
				inverseRows[rows[i]] = uint32(i)
				inverseCols[cols[i]] = uint32(i)
			}
			partial := int32(rng.IntN(7) - 2)
			k := rng.Uint32N(n + 1)
			// Before each selection, independently project admissible candidates
			// and mathematical scores from the actual remaining queue domain.
			for attempt := 0; attempt <= len(want); attempt++ {
				admissible := make(map[IndexPair]uint32)
				rejected := make(map[IndexPair]bool)
				var min uint32
				first := true
				for pair := range m.pivotQueue.index {
					if inverseRows[pair.I] < k || inverseCols[pair.J] < k {
						continue
					}
					small := pair.J != heads[pair.I] && math.Abs(fwant[pair])*float64(partial) < maxima[pair.I]
					if small {
						rejected[pair] = true
						continue
					}
					score := rowCounts[pair.I] * (colCounts[pair.J] - 1)
					admissible[pair] = score
					if first || score < min {
						min = score
						first = false
					}
				}
				i, j, fi, fj := uint32(91), uint32(92), uint32(91), uint32(92)
				ok, e := m.GetPivotForColumn(&i, &j, partial, k)
				h.must(e)
				fok, e := f.GetPivotForColumn(&fi, &fj, partial, k)
				lpStaticMust(t, e)
				if ok != (!first) || fok != ok || fi != i || fj != j {
					t.Fatal("selection profiles", ok, fok, i, j, fi, fj, first)
				}
				if ok {
					physical := IndexPair{rows[i], cols[j]}
					score, present := admissible[physical]
					if !present || score != min || m.pivotQueue.Contains(physical.I, physical.J) {
						t.Fatal("minimum admissible pivot", physical, score, min)
					}
				}
				for pair := range rejected {
					if !m.pivotQueue.Contains(pair.I, pair.J) || !f.pivotQueue.Contains(pair.I, pair.J) {
						t.Fatal("rejected pivot lost")
					}
				}
				if !ok {
					break
				}
			}
			lpExactSparseCheck(t, h, m, want)
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparsePivotLiteralQueueAndThresholds(t *testing.T) {
	t.Run("exhaustion recovery shortened wrap and retry threshold", func(t *testing.T) {
		h := newLPPairHarness(t)
		m := lpNewExactSparse(t, h, 2)
		f := newFloatSparseMatrix(2)
		p := h.pair("10", "1")
		h.must(m.AddNewElement(0, 0, &p.X))
		h.must(m.AddNewElement(0, 1, &p.Y))
		lpStaticMust(t, f.AddNewElement(0, 0, 10))
		lpStaticMust(t, f.AddNewElement(0, 1, 1))
		h.must(m.pivotQueue.Enqueue(0, 1, 0))
		lpStaticMust(t, f.pivotQueue.Enqueue(0, 1, 0))
		m.Columns[1].ShortenedMarkovitz = 1
		f.Columns[1].ShortenedMarkovitz = 1
		i, j, fi, fj := uint32(98), uint32(99), uint32(98), uint32(99)
		ok, e := m.GetPivotForColumn(&i, &j, 1, 0)
		h.must(e)
		fok, e := f.GetPivotForColumn(&fi, &fj, 1, 0)
		lpStaticMust(t, e)
		if ok || fok || i != 0 || j != 1 || fi != 0 || fj != 1 || m.pivotQueue.Size() != 1 || f.pivotQueue.Size() != 1 {
			t.Fatal("exhaustion recovery", ok, fok, i, j)
		}
		score, e := m.pivotQueue.Priority(0, 1)
		h.must(e)
		fs, e := f.pivotQueue.Priority(0, 1)
		lpStaticMust(t, e)
		if score != 0xfffffffe || fs != score {
			t.Fatal("unsigned shortened score", score, fs)
		}
		ok, e = m.GetPivotForColumn(&i, &j, 10, 0)
		h.must(e)
		fok, e = f.GetPivotForColumn(&fi, &fj, 10, 0)
		lpStaticMust(t, e)
		if !ok || !fok || i != 0 || j != 1 || !m.pivotQueue.IsEmpty() || !f.pivotQueue.IsEmpty() {
			t.Fatal("threshold retry")
		}
		h.must(m.Close())
		lpExactVectorNamedCensus(t, h, nil)
	})
	t.Run("IEEE comparison continuation", func(t *testing.T) {
		for _, head := range []float64{math.NaN(), math.Inf(1)} {
			f := newFloatSparseMatrix(2)
			lpStaticMust(t, f.AddNewElement(0, 0, head))
			lpStaticMust(t, f.AddNewElement(0, 1, math.NaN()))
			lpStaticMust(t, f.pivotQueue.Enqueue(0, 1, 0))
			i, j := uint32(98), uint32(99)
			ok, e := f.GetPivotForColumn(&i, &j, 0, 0)
			lpStaticMust(t, e)
			if !ok || i != 0 || j != 1 {
				t.Fatal("NaN strict comparison", ok, i, j)
			}
		}
	})
	t.Run("equal maxima first occurrence and exact threshold", func(t *testing.T) {
		h := newLPPairHarness(t)
		m := lpNewExactSparse(t, h, 3)
		f := newFloatSparseMatrix(3)
		p := h.pair("-10", "1")
		for _, entry := range []struct {
			i, j uint32
			v    int64
		}{{0, 2, -10}, {0, 0, 10}, {0, 1, 1}, {1, 2, 2}} {
			h.must(h.m.SetInt64(&p.X, entry.v))
			h.must(m.AddNewElement(entry.i, entry.j, &p.X))
			lpStaticMust(t, f.AddNewElement(entry.i, entry.j, float64(entry.v)))
		}
		h.must(m.PrepareForFactorization())
		lpStaticMust(t, f.PrepareForFactorization())
		if m.Rows[0][0].Index != 2 || f.Rows[0][0].Index != 2 {
			t.Fatal("strict tie ordering")
		}
		for _, factor := range []int32{9, 10, 11, -1} {
			b, e := m.beginWork(4)
			h.must(e)
			small, e := m.elementTooSmall(0, 1, factor)
			m.finishWork(b, &e)
			h.must(e)
			fs, e := f.elementTooSmall(0, 1, factor)
			lpStaticMust(t, e)
			want := 0
			if factor < 10 {
				want = 1
			}
			if small != want || fs != want {
				t.Fatal("strict threshold", factor, small, fs)
			}
		}
		if small, e := f.elementTooSmall(0, 2, -1); e != nil || small != 0 {
			t.Fatal("head bypass", small, e)
		}
		// Stale pair missing from a still-nonempty row is discarded, not recovered.
		for _, pair := range []IndexPair{{0, 2}, {0, 0}, {0, 1}, {1, 2}} {
			h.must(m.pivotQueue.Remove(pair.I, pair.J))
		}
		h.must(m.pivotQueue.Enqueue(0, 99, 0))
		lpStaticMust(t, f.pivotQueue.Enqueue(0, 99, 0))
		// Invalid column metadata in the inverse mapping is a reached bounds stop.
		i, j := uint32(80), uint32(81)
		_, e := m.GetPivotForColumn(&i, &j, 1, 0)
		if e != ErrBounds || i != 0 || j != 99 {
			t.Fatal("dequeue before inverse guard", i, j, e)
		}
		h.must(m.Close())
	})
	t.Run("stale and inactive candidates and empty outputs", func(t *testing.T) {
		h := newLPPairHarness(t)
		m := lpNewExactSparse(t, h, 3)
		f := newFloatSparseMatrix(3)
		p := h.pair("7", "0")
		for _, pair := range []IndexPair{{0, 0}, {1, 1}, {2, 2}} {
			h.must(m.AddNewElement(pair.I, pair.J, &p.X))
			lpStaticMust(t, f.AddNewElement(pair.I, pair.J, 7))
		}
		h.must(m.PrepareForFactorization())
		lpStaticMust(t, f.PrepareForFactorization())
		// Add a valid-index absent candidate with a forced smaller score.
		h.must(m.pivotQueue.Enqueue(0, 2, 0))
		lpStaticMust(t, f.pivotQueue.Enqueue(0, 2, 0))
		i, j := uint32(88), uint32(89)
		fi, fj := i, j
		ok, e := m.GetPivotForColumn(&i, &j, 1, 3)
		h.must(e)
		fok, e := f.GetPivotForColumn(&fi, &fj, 1, 3)
		lpStaticMust(t, e)
		if ok || fok || !m.pivotQueue.IsEmpty() || !f.pivotQueue.IsEmpty() {
			t.Fatal("inactive candidates kept")
		}
		i, j = 88, 89
		fi, fj = i, j
		ok, e = m.GetPivotForColumn(&i, &j, 1, 0)
		h.must(e)
		fok, e = f.GetPivotForColumn(&fi, &fj, 1, 0)
		lpStaticMust(t, e)
		if ok || fok || i != 88 || j != 89 || fi != 88 || fj != 89 {
			t.Fatal("empty queue changed outputs")
		}
		// Stale in-range candidate: search reaches the missing-element code2.
		h.must(m.pivotQueue.Enqueue(0, 2, 0))
		lpStaticMust(t, f.pivotQueue.Enqueue(0, 2, 0))
		ok, e = m.GetPivotForColumn(&i, &j, 1, 0)
		h.must(e)
		fok, e = f.GetPivotForColumn(&fi, &fj, 1, 0)
		lpStaticMust(t, e)
		if ok || fok || i != 0 || j != 2 || fi != 0 || fj != 2 {
			t.Fatal("missing pair output")
		}
		h.must(m.Close())
	})
	t.Run("prepare repeated assertion prefix", func(t *testing.T) {
		h := newLPPairHarness(t)
		m := lpNewExactSparse(t, h, 1)
		f := newFloatSparseMatrix(1)
		p := h.pair("1", "0")
		h.must(m.AddNewElement(0, 0, &p.X))
		lpStaticMust(t, f.AddNewElement(0, 0, 1))
		h.must(m.PrepareForFactorization())
		lpStaticMust(t, f.PrepareForFactorization())
		m.Columns[0].ShortenedMarkovitz = 5
		f.Columns[0].ShortenedMarkovitz = 5
		e := m.PrepareForFactorization()
		fe := f.PrepareForFactorization()
		if sourceDebugTests {
			if e != ErrAssertion || fe != ErrAssertion {
				t.Fatal(e, fe)
			}
		} else {
			h.must(e)
			lpStaticMust(t, fe)
		}
		if m.Columns[0].ShortenedMarkovitz != 0 || f.Columns[0].ShortenedMarkovitz != 0 {
			t.Fatal("reset must precede queue assertion")
		}
		h.must(m.Close())
	})
}

func TestLPSparsePivotBoundsAndPublicationPrefixes(t *testing.T) {
	for _, phase := range []string{"empty row", "bad row", "bad column", "prepare bad column", "prepare row extent", "max link", "start link"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 3)
			f := newFloatSparseMatrix(3)
			p := h.pair("1", "3")
			var e, fe error
			switch phase {
			case "empty row", "bad row", "bad column":
				pair := IndexPair{0, 0}
				if phase == "bad row" {
					pair.I = 3
				}
				if phase == "bad column" {
					pair.J = 3
					h.must(m.AddNewElement(0, 0, &p.X))
					lpStaticMust(t, f.AddNewElement(0, 0, 1))
				}
				h.must(m.pivotQueue.Enqueue(pair.I, pair.J, 0))
				lpStaticMust(t, f.pivotQueue.Enqueue(pair.I, pair.J, 0))
				i, j, fi, fj := uint32(98), uint32(99), uint32(98), uint32(99)
				_, e = m.GetPivotForColumn(&i, &j, 1, 0)
				_, fe = f.GetPivotForColumn(&fi, &fj, 1, 0)
				if i != pair.I || j != pair.J || fi != pair.I || fj != pair.J || !m.pivotQueue.IsEmpty() || !f.pivotQueue.IsEmpty() {
					t.Fatal("physical dequeue prefix")
				}
			case "prepare bad column":
				h.must(m.AddNewElement(0, 0, &p.X))
				lpStaticMust(t, f.AddNewElement(0, 0, 1))
				m.Rows[0][0].Index = 3
				f.Rows[0][0].Index = 3
				e = m.PrepareForFactorization()
				fe = f.PrepareForFactorization()
			case "prepare row extent":
				m.Rows = m.Rows[:2]
				f.Rows = f.Rows[:2]
				e = m.PrepareForFactorization()
				fe = f.PrepareForFactorization()
			case "max link", "start link":
				h.must(m.AddNewElement(0, 0, &p.X))
				h.must(m.AddNewElement(0, 1, &p.Y))
				lpStaticMust(t, f.AddNewElement(0, 0, 1))
				lpStaticMust(t, f.AddNewElement(0, 1, 3))
				if phase == "max link" {
					m.Rows[0][1].Other = 99
					f.Rows[0][1].Other = 99
				} else {
					m.Rows[0][0].Other = 99
					f.Rows[0][0].Other = 99
				}
				e = m.SetMaxInRow(0)
				fe = f.SetMaxInRow(0)
				if phase == "start link" && (m.Columns[1].Values[0].Other != 0 || f.Columns[1].Values[0].Other != 0) {
					t.Fatal("maximum backlink before start lookup")
				}
			}
			if e != ErrBounds || fe != ErrBounds {
				t.Fatal(e, fe)
			}
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactSparsePivotAllocationFailureCensus(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, campaign := range []string{"max", "prepare", "choose", "max-direct", "prepare-direct", "choose-direct"} {
		op := strings.TrimSuffix(campaign, "-direct")
		failures, successes, receipts, prefixes := 0, 0, 0, 0
		for ordinal := int64(0); ordinal < 280; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%s/%d", campaign, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if strings.HasSuffix(campaign, "-direct") {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				m := lpNewExactSparse(t, h, 3)
				xs, ys := strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 200)+"/13"
				p := h.pair(xs, ys)
				one := h.pair("1", "0")
				want := make(map[IndexPair]*big.Rat)
				x, _ := new(big.Rat).SetString(xs)
				y, _ := new(big.Rat).SetString(ys)
				if op == "choose" {
					h.must(m.AddNewElement(0, 0, &p.Y))
					want[IndexPair{0, 0}] = y
					h.must(m.AddNewElement(0, 1, &p.X))
					want[IndexPair{0, 1}] = x
					h.must(m.AddNewElement(1, 0, &one.X))
					want[IndexPair{1, 0}] = big.NewRat(1, 1)
					h.must(m.PrepareForFactorization())
					h.must(m.SwapRows(0, 1))
					h.must(m.SwapColumns(0, 1))
				} else {
					h.must(m.AddNewElement(0, 0, &p.X))
					want[IndexPair{0, 0}] = x
					h.must(m.AddNewElement(0, 1, &p.Y))
					want[IndexPair{0, 1}] = y
				}
				h.budget.FailAfter(ordinal)
				i, j := uint32(88), uint32(89)
				var e error
				switch op {
				case "max":
					e = m.SetMaxInRow(0)
				case "prepare":
					e = m.PrepareForFactorization()
				default:
					var ok bool
					ok, e = m.GetPivotForColumn(&i, &j, 1, 0)
					if e == nil && (!ok || i != 0 || j != 1) {
						t.Fatal("chosen logical pivot", ok, i, j)
					}
				}
				h.budget.FailAfter(-1)
				if e == nil {
					successes++
					done = true
					lpExactSparseCheck(t, h, m, want)
					if op != "choose" && m.Rows[0][0].Index != 1 {
						t.Fatal("maximum")
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(e) {
						t.Fatal(e)
					}
					failures++
					if m.calls.pending() {
						receipts++
					}
					if op == "max" && m.Columns[0].Values[0].Other == 1 && m.Columns[1].Values[0].Other == 0 {
						prefixes++
					}
					if op == "prepare" && m.pivotQueue.Size() > 0 {
						prefixes++
					}
					if op == "choose" && len(m.work.Data) > 4 && i == 1 && j == 0 && !m.pivotQueue.Contains(0, 1) {
						prefixes++
					}
					for _, f := range []func() error{func() error { return m.SetMaxInRow(0) }, func() error { return m.PrepareForFactorization() }, func() error { _, e := m.GetPivotForColumn(&i, &j, 1, 0); return e }} {
						if f() != ErrExactVectorPending {
							t.Fatal("pending pivot owner")
						}
					}
				}
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
				if h.m.ComparisonInvocationState().Active || h.m.RawInvocationState().Active != [3]bool{} {
					t.Fatal("orphan comparison/arithmetic frame")
				}
			})
			if done {
				break
			}
		}
		if failures == 0 || successes != 1 || receipts == 0 || (strings.HasSuffix(campaign, "-direct") && prefixes == 0) {
			t.Fatal("incomplete pivot campaign", campaign, failures, successes, receipts, prefixes)
		}
	}
}

func TestLPExactSparseMaximumDoesNotAdoptOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpNewExactSparse(t, h, 2)
			p := h.pair(strings.Repeat("1234567890", 100)+"/7", strings.Repeat("9876543210", 100)+"/13")
			h.must(m.AddNewElement(0, 0, &p.X))
			h.must(m.AddNewElement(0, 1, &p.Y))
			var caller exactPriorityCalls
			b, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&p.X, &p.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, b, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				matched = true
				e = m.SetMaxInRow(0)
				if e != numeral.ErrRationalRawArithmeticPending || m.calls.comparison != 0 {
					t.Fatal("older comparison ownership", e, m.calls.comparison)
				}
				h.must(m.Close())
				if h.m.ComparisonInvocationState() != older {
					t.Fatal("closed caller's comparison")
				}
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real comparison refusal")
	}
}
