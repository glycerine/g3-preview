package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPRationalPivotCompareIndependentOracle(t *testing.T) {
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 3)
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		cp, rp := big.NewRat(int64(world%29-14), 7), big.NewRat(int64(world%23-11), 13)
		switch world % 8 {
		case 0:
			cp.SetInt64(0)
		case 1:
			rp.SetInt64(0)
		case 2:
			rp.Set(cp)
		case 3:
			cp.SetInt64(17)
			rp.Add(cp, mustLPRat(t, "1/"+strings.Repeat("9", 80)))
		}
		h.must(h.m.SetString(&m.Ed.Data[0], cp.RatString()))
		h.must(h.m.SetString(&m.PivotRow.Data[1], rp.RatString()))
		m.Settings.HarrisFeasibilityTolerance = 1e100
		if world%2 == 0 {
			m.Settings.HarrisFeasibilityTolerance = 0
		}
		want := 0
		if cp.Sign() == 0 || rp.Sign() == 0 {
			want = 1
		} else if cp.Sign() != rp.Sign() {
			want = 2
		} else if cp.Cmp(rp) != 0 {
			want = 1
		}
		got, err := m.PivotsInColumnAndRowAreDifferent(1, 0)
		h.must(err)
		if got != want || m.pivotWork != nil {
			t.Fatal(world, got, want)
		}
		lpSparseRationalCheck(t, h, &m.Ed.Data[0], cp)
		lpSparseRationalCheck(t, h, &m.PivotRow.Data[1], rp)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPivotComparePhysicalOwnership(t *testing.T) {
	for _, kind := range []string{"equal-positive", "equal-negative", "drift-positive", "drift-negative", "minimum"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", kind, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 2)
					m, e := NewRationalCoreSolverBase(f.input())
					h.must(e)
					cp, rp := strings.Repeat("1234567890", 12)+"1/7", strings.Repeat("1234567890", 12)+"1/7"
					if strings.HasPrefix(kind, "drift") {
						rp = strings.Repeat("1234567890", 12) + "3/13"
					}
					if strings.HasSuffix(kind, "negative") {
						cp = "-" + cp
						rp = "-" + rp
					}
					if kind == "minimum" {
						h.must(h.m.SetInt32(&m.Ed.Data[0], -2147483648))
						h.must(h.m.SetInt32(&m.PivotRow.Data[1], -2147483648))
					} else {
						h.must(h.m.SetString(&m.Ed.Data[0], cp))
						h.must(h.m.SetString(&m.PivotRow.Data[1], rp))
					}
					h.budget.FailAfter(ordinal)
					got, err := m.PivotsInColumnAndRowAreDifferent(1, 0)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						want := 0
						if strings.HasPrefix(kind, "drift") {
							want = 1
						}
						if got != want {
							t.Fatal(got, want)
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if _, e = m.PivotsInColumnAndRowAreDifferent(1, 0); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if t.Failed() {
					return
				}
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal(kind, direct, success, failures)
			}
		}
	}
}

func TestLPRationalPivotCompareReachedGuards(t *testing.T) {
	for _, phase := range []string{"heading", "ed", "row", "negative", "zero", "opposite"} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 2)
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		h.must(h.m.SetInt32(&m.Ed.Data[0], 3))
		h.must(h.m.SetInt32(&m.PivotRow.Data[1], 5))
		enter := int32(1)
		wantErr := error(ErrBounds)
		want := 0
		switch phase {
		case "heading":
			f.heading = nil
		case "ed":
			f.heading[0] = -1
		case "row":
			h.must(m.PivotRow.resizeData(0))
		case "negative":
			enter = -1
		case "zero":
			h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
			wantErr = nil
			want = 1
		case "opposite":
			h.must(h.m.SetInt32(&m.Ed.Data[0], -3))
			wantErr = nil
			want = 2
		}
		h.budget.FailAfter(0)
		got, err := m.PivotsInColumnAndRowAreDifferent(enter, 0)
		h.budget.FailAfter(-1)
		if err != wantErr || got != want || m.pivotWork != nil {
			t.Fatal(phase, got, err)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
