package lp

// Source: src/util/lp/lar_solver.h:141-159,1340-1345.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func lpImpliedDisplayFixture(t *testing.T, h *lpPairHarness, term, large bool, low, strict bool) (*LARSolver, *ImpliedBound, string) {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	j, err := s.AddVar(17)
	h.must(err)
	rhs := "-7/3"
	if large {
		rhs = mustLPRat(t, "-"+strings.Repeat("1234567890", 30)+"/19").RatString()
	}
	left := "v17"
	if term {
		coefficient := "-1"
		if large {
			coefficient = mustLPRat(t, strings.Repeat("9876543210", 30)+"/23").RatString()
		}
		j, err = s.AddTerm([]RationalSparseEntry{{Index: 0, Value: h.pair(coefficient, "0").X}}, &h.pair("5/2", "0").X)
		h.must(err)
		// The current term may have been substituted. Printing must use the original.
		h.must(s.Terms[0].SetValue(&h.pair("99", "0").X))
		left = fmt.Sprintf("it is a term number %d\n5/2 + ", j)
		if large {
			left += coefficient + "v17"
		} else {
			left += " - v17"
		}
	}
	b, err := NewImpliedBound(h.m, &h.pair(rhs, "0").X, j, low, true, 0, strict)
	h.must(err)
	kind := "<="
	if low {
		kind = ">="
	}
	if strict {
		kind = strings.TrimSuffix(kind, "=")
	}
	return s, b, "implied bound\n" + left + " " + kind + " " + rhs + "\nend of implied bound\n"
}

func TestLARImpliedDisplaySourceText(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, term := range []bool{false, true} {
		for _, large := range []bool{false, true} {
			for _, low := range []bool{false, true} {
				for _, strict := range []bool{false, true} {
					h := newLPPairHarness(t)
					s, b, want := lpImpliedDisplayFixture(t, h, term, large, low, strict)
					out := &lpDriverFlushWriter{}
					h.must(s.PrintImpliedBound(b, out))
					flushes := 2
					if term {
						flushes++
					}
					if out.String() != want || out.flushes != flushes || s.rationalWork != nil || s.validationCoefficients != nil {
						t.Fatal("source report", out.String(), want, out.flushes)
					}
					h.must(b.Close())
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func TestLARImpliedDisplayEveryOutputPrefix(t *testing.T) {
	for _, term := range []bool{false, true} {
		h0 := newLPPairHarness(t)
		s0, b0, want := lpImpliedDisplayFixture(t, h0, term, false, false, false)
		h0.must(b0.Close())
		h0.must(s0.Close())
		lpExactVectorNamedCensus(t, h0, nil)
		for cut := 0; cut < len(want); cut++ {
			h := newLPPairHarness(t)
			s, b, _ := lpImpliedDisplayFixture(t, h, term, false, false, false)
			out := &larConstraintPrefixWriter{remaining: cut, err: io.ErrClosedPipe}
			err := s.PrintImpliedBound(b, out)
			if !errors.Is(err, io.ErrClosedPipe) || out.output.String() != want[:cut] || s.PrintImpliedBound(b, io.Discard) != ErrExactVectorPending {
				t.Fatal("write prefix", term, cut, err, out.output.String())
			}
			h.must(b.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
		flushes := 2
		if term {
			flushes++
		}
		for fail := 1; fail <= flushes; fail++ {
			h := newLPPairHarness(t)
			s, b, _ := lpImpliedDisplayFixture(t, h, term, false, false, false)
			out := &lpDriverFlushWriter{failAt: fail}
			err := s.PrintImpliedBound(b, out)
			end := len(want)
			if fail < flushes {
				end = strings.Index(want, "end of implied bound")
			}
			if term && fail == 1 {
				end = strings.Index(want, "5/2")
			}
			if !errors.Is(err, lpErrResidualWrite) || out.String() != want[:end] || out.flushes != fail {
				t.Fatal("endl prefix", term, fail, err, out.String(), want[:end])
			}
			h.must(b.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARImpliedDisplayPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		done, failures := false, 0
		for ordinal := int64(0); ordinal < 1200 && !done; ordinal++ {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			s, b, want := lpImpliedDisplayFixture(t, h, true, true, true, true)
			var out bytes.Buffer
			h.budget.FailAfter(ordinal)
			err := s.PrintImpliedBound(b, &out)
			h.budget.FailAfter(-1)
			if err == nil {
				done = true
				if out.String() != want {
					t.Fatal("physical report", out.String(), want)
				}
			} else {
				failures++
				if !lpExactPriorityPhysicalRefusal(err) || !strings.HasPrefix(want, out.String()) {
					t.Fatal("physical prefix", err, out.String())
				}
			}
			h.must(b.Close())
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !done || direct && failures == 0 {
			t.Fatal("physical campaign", direct, done, failures)
		}
	}
}

func TestLARImpliedDisplayReachedGuards(t *testing.T) {
	for _, phase := range []string{"nil-writer", "nil-bound", "stopped-bound", "missing-original", "short-write", "nil-term", "term-nil-writer", "stopped-term"} {
		h := newLPPairHarness(t)
		s, b, _ := lpImpliedDisplayFixture(t, h, true, false, false, false)
		var out bytes.Buffer
		var err error
		want := ErrExactVectorOwner
		switch phase {
		case "nil-writer":
			err = s.PrintImpliedBound(b, nil)
			want = ErrBounds
		case "nil-bound":
			err = s.PrintImpliedBound(nil, &out)
		case "stopped-bound":
			b.stopped = true
			err = s.PrintImpliedBound(b, &out)
			want = ErrExactVectorPending
		case "missing-original":
			original := s.OriginalTerms
			s.OriginalTerms = nil
			err = s.PrintImpliedBound(b, &out)
			s.OriginalTerms = original
			want = ErrBounds
			if out.String() != fmt.Sprintf("implied bound\nit is a term number %d\n", b.Column) {
				t.Fatal(out.String())
			}
		case "short-write":
			w := &larConstraintPrefixWriter{remaining: 2}
			err = s.PrintImpliedBound(b, w)
			want = io.ErrShortWrite
			if w.output.String() != "im" {
				t.Fatal(w.output.String())
			}
		case "nil-term":
			err = s.PrintTerm(nil, &out)
		case "term-nil-writer":
			err = s.PrintTerm(s.OriginalTerms[0], nil)
			want = ErrBounds
		case "stopped-term":
			s.OriginalTerms[0].work = NewIndexedRationalVector(h.m, 0)
			err = s.PrintTerm(s.OriginalTerms[0], &out)
			want = ErrExactVectorPending
		}
		if !errors.Is(err, want) {
			t.Fatal(phase, err, want)
		}
		h.must(b.Close())
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, constant := range []string{"0", "-5"} {
		h := newLPPairHarness(t)
		s, err := NewLARSolver(h.m)
		h.must(err)
		term, err := NewLARTermFromCoefficients(h.m, nil, &h.pair(constant, "0").X)
		h.must(err)
		var out bytes.Buffer
		h.must(s.PrintTerm(term, &out))
		want := ""
		if constant != "0" {
			want = constant + " + "
		}
		if out.String() != want {
			t.Fatal(out.String(), want)
		}
		h.must(term.Close())
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARImpliedDisplayPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, term := range []bool{false, true} {
		found := false
		for ordinal := int64(0); ordinal < 40 && !found; ordinal++ {
			h := newLPPairHarness(t)
			s, b, _ := lpImpliedDisplayFixture(t, h, term, true, false, false)
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
				_ = s.PrintImpliedBound(b, io.Discard)
				if s.calls.comparison != 0 || b.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			}
			h.must(b.Close())
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !found {
			t.Fatal("missing actual older comparison", term)
		}
	}
}
