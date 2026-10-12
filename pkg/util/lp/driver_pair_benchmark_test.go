package lp

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func lpPairBenchmarkOracle(seed, iterations uint32) ([]*big.Rat, string, string, uint32) {
	inputs, result, state := lpScalarBenchmarkOracle(seed, iterations)
	prefix := "x = "
	for _, v := range inputs[10:] {
		prefix += "(" + v.RatString() + ", 0) "
	}
	prefix += "\n"
	text := lpBenchmarkNumberOracle(result)
	return inputs, prefix, text, state
}

func TestLPPairBenchmarkIndependentInputsProductsAndReport(t *testing.T) {
	for seed := uint32(0); seed < 30; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			h := newLPPairHarness(t)
			iterations := []uint32{0, 1, 3, 7}[seed%4]
			inputs, prefix, result, state := lpPairBenchmarkOracle(seed, iterations)
			var d *PairBenchmarkDriver
			inspected := false
			w := &lpMPSDriverActionWriter{action: func(text string) error {
				if text == "operation with rationals " {
					for i := 0; i < 10; i++ {
						lpSparseRationalCheck(t, h, &d.scalars.Data[i], inputs[i])
						h.check(&d.pairs.Data[i], inputs[10+i], new(big.Rat))
					}
					_, want, _ := lpScalarBenchmarkOracle(seed, iterations)
					h.check(&d.pairs.Data[10], want, new(big.Rat))
					inspected = true
				}
				return nil
			}}
			d = NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: w, Clock: lpBenchmarkClock()})
			r := z3rand.New(seed)
			h.must(d.runIterations(r, iterations))
			want := prefix + "operation with rationals 2\n" + result + "\n"
			if !inspected || w.buffer.String() != want || w.flushes != 3 || d.ResultText != result || d.CompletedIterations != iterations || d.CompletedProducts != 10*iterations || d.Coefficients != 10 || d.Values != 10 || !d.closed || d.pairs != nil || d.scalars != nil || d.calls.pending() {
				t.Fatal(d, w.buffer.String(), want)
			}
			state = state*214013 + 2531011
			if r.Next() != (state>>16)&32767 {
				t.Fatal("shared RNG order")
			}
			if d.runIterations(r, 1) != ErrExactVectorPending {
				t.Fatal("repeat")
			}
			h.must(d.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPPairBenchmarkEveryReportByteAndEndlRefusal(t *testing.T) {
	_, prefix, result, _ := lpPairBenchmarkOracle(17, 1)
	report := prefix + "operation with rationals 2\n" + result + "\n"
	for _, cause := range []error{nil, lpErrResidualWrite} {
		for n := 0; n < len(report); n++ {
			h := newLPPairHarness(t)
			w := &larConstraintPrefixWriter{remaining: n, err: cause}
			reads := 0
			clock := func() (uint64, error) { reads++; return uint64(reads) * 2e9, nil }
			d := NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: w, Clock: clock})
			err := d.runIterations(z3rand.New(17), 1)
			expected := cause
			if expected == nil {
				expected = io.ErrShortWrite
			}
			completed, clockReads := uint32(0), 0
			if n >= len(prefix) {
				completed, clockReads = 1, 2
			}
			resultReady := n >= len(prefix)+len("operation with rationals 2\n")
			if !errors.Is(err, expected) || w.output.String() != report[:n] || d.CompletedIterations != completed || d.CompletedProducts != 10*completed || reads != clockReads || d.closed || d.pairs == nil || d.scalars == nil || (d.ResultText != "") != resultReady {
				t.Fatal(n, err, d, w.output.String())
			}
			h.must(d.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
	for flush := 1; flush <= 3; flush++ {
		h := newLPPairHarness(t)
		w := &lpDriverFlushWriter{failAt: flush}
		d := NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: w, Clock: lpBenchmarkClock()})
		err := d.runIterations(z3rand.New(17), 1)
		n := len(prefix)
		if flush >= 2 {
			n += len("operation with rationals 2\n")
		}
		if flush == 3 {
			n = len(report)
		}
		if !errors.Is(err, lpErrResidualWrite) || w.String() != report[:n] || w.flushes != flush || d.closed || (d.CompletedIterations == 1) != (flush > 1) {
			t.Fatal(flush, err, d, w.String())
		}
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairBenchmarkClockAndOwnerBoundaries(t *testing.T) {
	failure := errors.New("pair benchmark clock")
	for failAt := 1; failAt <= 2; failAt++ {
		h := newLPPairHarness(t)
		reads := 0
		clock := func() (uint64, error) {
			reads++
			if reads == failAt {
				return 0, failure
			}
			return uint64(reads) * 1e9, nil
		}
		w := &lpDriverFlushWriter{}
		d := NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: w, Clock: clock})
		err := d.runIterations(z3rand.New(23), 3)
		_, prefix, _, _ := lpPairBenchmarkOracle(23, 3)
		if !errors.Is(err, failure) || reads != failAt || w.String() != prefix || w.flushes != 1 || d.CompletedIterations != uint32((failAt-1)*3) || d.CompletedProducts != uint32((failAt-1)*30) || d.closed || d.watch.Running() != (failAt == 2) {
			t.Fatal(failAt, err, d, w.String())
		}
		h.must(d.Close())
		h.must(d.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	d := NewPairBenchmarkDriver(nil, MPSDriverConfig{Output: io.Discard})
	if d.Run(z3rand.New(0)) != ErrBounds {
		t.Fatal("nil manager")
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	h := newLPPairHarness(t)
	d = NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: io.Discard})
	if d.Run(nil) != ErrBounds || d.scalars != nil || d.pairs != nil {
		t.Fatal("nil random")
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairBenchmarkPhysicalRefusalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		done, failures, partial := false, 0, false
		for ordinal := int64(0); ordinal < 1500 && !done; ordinal++ {
			if !t.Run(fmt.Sprintf("direct=%v/refusal=%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				d := NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
				h.budget.FailAfter(ordinal)
				err := d.runIterations(z3rand.New(19), 2)
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
					_, _, want, _ := lpPairBenchmarkOracle(19, 2)
					if d.ResultText != want || !d.closed || d.CompletedProducts != 20 {
						t.Fatal("completed result", d)
					}
				} else {
					if !lpExactPriorityPhysicalRefusal(err) || d.closed {
						t.Fatal(err)
					}
					failures++
					if d.CompletedProducts > 0 {
						partial = true
					}
					if d.runIterations(z3rand.New(19), 1) != ErrExactVectorPending {
						t.Fatal("failed owner reentered")
					}
				}
				h.must(d.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		if !done || direct && (failures == 0 || !partial) {
			t.Fatal("missing physical campaign prefixes", direct, done, failures, partial)
		}
	}
}

func TestLPPairBenchmarkPreservesOlderComparison(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 40 && !found; ordinal++ {
		h := newLPPairHarness(t)
		v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
		var caller exactEtaCalls
		before, err := caller.begin(h.m)
		h.must(err)
		h.budget.FailAfter(ordinal)
		_, err = h.m.Compare(&v.X, &v.Y)
		h.budget.FailAfter(-1)
		err = caller.after(h.m, before, err)
		older := h.m.ComparisonInvocationState()
		if older.Active {
			found = true
			if !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			d := NewPairBenchmarkDriver(h.m, MPSDriverConfig{Output: io.Discard, Clock: lpBenchmarkClock()})
			// Actual stopped shared scratch is unavailable to normalization.
			if err := d.runIterations(z3rand.New(19), 3); !errors.Is(err, arena.ErrInvalidSlot) {
				t.Fatal(err)
			}
			if d.calls.comparison != 0 || h.m.ComparisonInvocationState() != older {
				t.Fatal("adopted or consumed caller comparison")
			}
			h.must(d.Close())
			if h.m.ComparisonInvocationState() != older {
				t.Fatal("cleanup consumed caller comparison")
			}
		}
		h.must(caller.cleanupRaw(h.m))
		h.must(caller.cleanupRoots(h.m))
		lpExactVectorNamedCensus(t, h, nil)
	}
	if !found {
		t.Fatal("no actual older comparison")
	}
}

// Each sample executes the literal registered source body and validates the
// exact result and successful owning cleanup. Run explicitly with -benchtime=1x.
func BenchmarkLPPairSource1000000(b *testing.B) {
	budget := arena.NewBudget(8 << 20)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{PageSize: 4, Budget: budget},
		Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 4, Budget: budget}},
	}, Synchronized: true})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := m.CloseChecked(); err != nil {
			b.Error(err)
		}
		if budget.Used() != 0 {
			b.Error("retained bytes", budget.Used())
		}
	})
	_, _, want, _ := lpPairBenchmarkOracle(19, 1000000)
	_, exact, _ := lpScalarBenchmarkOracle(19, 1000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var d *PairBenchmarkDriver
		checked := false
		out := &lpMPSDriverActionWriter{action: func(text string) error {
			if text == "operation with rationals " {
				actual, e := d.pairs.Data[10].String(m)
				if e != nil {
					b.Fatal(e)
				}
				if actual != "("+exact.RatString()+", 0)" {
					b.Fatal("independent exact accumulator", actual, exact)
				}
				checked = true
			}
			return nil
		}}
		d = NewPairBenchmarkDriver(m, MPSDriverConfig{Output: out})
		b.Cleanup(func() {
			if err := d.Close(); err != nil {
				b.Error(err)
			}
		})
		if err := d.Run(z3rand.New(19)); err != nil {
			b.Fatal(err)
		}
		if !checked || d.CompletedIterations != 1000000 || d.CompletedProducts != 10000000 || d.ResultText != want || !d.closed {
			b.Fatal("registered count/result/cleanup", d)
		}
	}
}
