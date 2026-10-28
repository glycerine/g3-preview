package sat

// Source: src/sat/sat_solver.cpp.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverDynamicSubsumptionBinaryAndFUIP(t *testing.T) {
	probe, life := satProbingFixture(t, 3)
	a, e := newSATSolverConflictAnalysis(life.propagation, life.budget)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	d, e := newSATSolverDynamicSubsumption(a, probe, life.budget)
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	for _, l := range []uint32{0, 2, 4} {
		satOK(t, a.lemma.Push(l))
	}
	// x OR !y permits removing y; x OR !x must never remove FUIP x.
	satOK(t, life.mkBinClause(0, 3, false))
	satOK(t, life.mkBinClause(0, 1, false))
	satOK(t, d.run())
	n, e := a.lemma.Size()
	satOK(t, e)
	if n != 2 || d.eliminated != 1 {
		t.Fatal(n, d.eliminated)
	}
	for i, w := range []uint32{0, 4} {
		x, e := a.lemma.Read(uint32(i))
		satOK(t, e)
		if x != w {
			t.Fatal(x, w)
		}
	}
	for i := uint32(0); i < 6; i++ {
		if life.propagation.variables.literalMark[i] {
			t.Fatal("source ascending cleanup", i)
		}
	}
}
func TestSATSolverDynamicSubsumptionCachedImplicationAndFUIP(t *testing.T) {
	probe, life := satProbingFixture(t, 3)
	p := life.propagation
	a, e := newSATSolverConflictAnalysis(p, life.budget)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	d, e := newSATSolverDynamicSubsumption(a, probe, life.budget)
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	satOK(t, p.assignCore(3, satJustification{}))
	satOK(t, p.assignCore(1, satJustification{}))
	satOK(t, probe.cacheBins(1, 0))
	for _, l := range []uint32{0, 2, 4} {
		satOK(t, a.lemma.Push(l))
	}
	satOK(t, d.run())
	n, e := a.lemma.Size()
	satOK(t, e)
	if n != 2 || d.eliminated != 1 {
		t.Fatal(n, d.eliminated)
	}
	first, e := a.lemma.Read(0)
	satOK(t, e)
	if first != 0 {
		t.Fatal("FUIP protected", first)
	}
}

func TestSATSolverDynamicSubsumptionIndependentTernaryResolution(t *testing.T) {
	for a := Literal(0); a < 8; a++ {
		for b := Literal(0); b < 8; b++ {
			if a == b || a.Negated() == b {
				continue
			}
			t.Run(fmt.Sprintf("%d/%d", a, b), func(t *testing.T) {
				probe, life := satProbingFixture(t, 4)
				analysis, e := newSATSolverConflictAnalysis(life.propagation, life.budget)
				satOK(t, e)
				defer func() { satOK(t, analysis.Close()) }()
				d, e := newSATSolverDynamicSubsumption(analysis, probe, life.budget)
				satOK(t, e)
				defer func() { satOK(t, d.Close()) }()
				for _, l := range []uint32{0, 2, 4} {
					satOK(t, analysis.lemma.Push(l))
				}
				watch, e := newSATTernaryWatch(a, b)
				satOK(t, e)
				satOK(t, life.pushWatch(1, watch))
				satOK(t, d.run())
				n, e := analysis.lemma.Size()
				satOK(t, e)
				if n < 1 || n > 3 || satScalarTestUint(t, &analysis.lemma, 0) != 0 || d.eliminated != 3-n {
					t.Fatal("FUIP/stats", n, d.eliminated)
				}
				// The original lemma together with the actual ternary clause entails every
				// surviving resolvent. This oracle independently checks all16 valuations.
				truth := func(mask uint32, l Literal) bool { return (mask>>uint32(l.variable())&1 != 0) != (l.index()&1 != 0) }
				for mask := uint32(0); mask < 16; mask++ {
					old := truth(mask, 0) || truth(mask, 2) || truth(mask, 4)
					clause := truth(mask, 0) || truth(mask, a) || truth(mask, b)
					now := false
					for i := uint32(0); i < n; i++ {
						now = now || truth(mask, Literal(satScalarTestUint(t, &analysis.lemma, i)))
					}
					if old && clause && !now {
						t.Fatal("invalid resolution", a, b, mask)
					}
				}
				for i := uint32(0); i < 8; i++ {
					if life.propagation.variables.literalMark[i] {
						t.Fatal("mark cleanup", i)
					}
				}
				satProbingAudit(t, probe, analysis, d)
			})
		}
	}
}
func TestSATSolverDynamicSubsumptionNonbinaryBreakAndCacheFallback(t *testing.T) {
	for cached := 0; cached < 2; cached++ {
		t.Run(fmt.Sprint(cached), func(t *testing.T) {
			probe, life := satProbingFixture(t, 3)
			p := life.propagation
			analysis, e := newSATSolverConflictAnalysis(p, life.budget)
			satOK(t, e)
			defer func() { satOK(t, analysis.Close()) }()
			d, e := newSATSolverDynamicSubsumption(analysis, probe, life.budget)
			satOK(t, e)
			defer func() { satOK(t, d.Close()) }()
			ext, e := newSATExternalWatch(3)
			satOK(t, e)
			satOK(t, life.pushWatch(1, ext))
			bin, e := newSATBinaryWatch(3, false)
			satOK(t, e)
			satOK(t, life.pushWatch(1, bin))
			if cached != 0 {
				satOK(t, p.assignCore(3, satJustification{}))
				satOK(t, probe.cacheBins(1, 0))
			}
			for _, l := range []uint32{0, 2, 4} {
				satOK(t, analysis.lemma.Push(l))
			}
			satOK(t, d.run())
			n, e := analysis.lemma.Size()
			satOK(t, e)
			if n != uint32(3-cached) {
				t.Fatal("first nonbinary stops watch scan; cache still queried", n)
			}
			satProbingAudit(t, probe, analysis, d)
		})
	}
}
func TestSATSolverDynamicSubsumptionDuplicateMarkFailureAndRetainedEmpty(t *testing.T) {
	probe, life := satProbingFixture(t, 2)
	a, e := newSATSolverConflictAnalysis(life.propagation, life.budget)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	d, e := newSATSolverDynamicSubsumption(a, probe, life.budget)
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	satOK(t, a.lemma.Push(0))
	satOK(t, a.lemma.Push(0))
	e = d.run()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || !life.propagation.variables.literalMark[0] || satScalarTestUintSize(t, &a.lemma) != 2 {
			t.Fatal("mark failure prefix", e)
		}
		// Explicit caller cleanup of the actually retained bit, not algorithm repair.
		life.propagation.variables.literalMark[0] = false
	} else {
		satOK(t, e)
		if satScalarTestUintSize(t, &a.lemma) != 1 {
			t.Fatal("duplicate mark source compaction")
		}
	}
	satOK(t, a.lemma.Shrink(0))
	e = d.run()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("empty lemma bounds guard", e)
		}
	}
	satProbingAudit(t, probe, a, d)
}

func TestSATSolverDynamicSubsumptionRawComplementarySourceQuirk(t *testing.T) {
	// Actual solver simplify_clause cpp442 rejects these tautological clauses.
	// This raw watch-overload fixture preserves the source method's noncanonical
	// behavior, including its mathematical counterexample; no repair is claimed.
	for a := Literal(0); a < 8; a++ {
		t.Run(fmt.Sprint(a.index()), func(t *testing.T) {
			probe, life := satProbingFixture(t, 4)
			analysis, e := newSATSolverConflictAnalysis(life.propagation, life.budget)
			satOK(t, e)
			defer func() { satOK(t, analysis.Close()) }()
			d, e := newSATSolverDynamicSubsumption(analysis, probe, life.budget)
			satOK(t, e)
			defer func() { satOK(t, d.Close()) }()
			for _, l := range []uint32{0, 2, 4} {
				satOK(t, analysis.lemma.Push(l))
			}
			watch, e := newSATTernaryWatch(a, a.Negated())
			satOK(t, e)
			satOK(t, life.pushWatch(1, watch))
			satOK(t, d.run())
			want := []uint32{0, 2, 4}
			if a.variable() == 1 {
				want = []uint32{0, 4}
			} else if a.variable() == 2 {
				want = []uint32{0, 2}
			}
			if satScalarTestUintSize(t, &analysis.lemma) != uint32(len(want)) {
				t.Fatal("literal source result")
			}
			for i, l := range want {
				if satScalarTestUint(t, &analysis.lemma, uint32(i)) != l {
					t.Fatal(i, l)
				}
			}
			witness := false
			for mask := uint32(0); mask < 16; mask++ {
				old := mask&7 != 0
				now := false
				for _, l := range want {
					now = now || mask>>uint32(Literal(l).variable())&1 != 0
				}
				witness = witness || (old && !now)
			}
			if witness != (a.variable() == 1 || a.variable() == 2) {
				t.Fatal("independent raw-domain counterexample", witness)
			}
			satProbingAudit(t, probe, analysis, d)
		})
	}
}
func TestSATSolverDynamicSubsumptionMarkWriteFailurePrefix(t *testing.T) {
	probe, life := satProbingFixture(t, 2)
	a, e := newSATSolverConflictAnalysis(life.propagation, life.budget)
	satOK(t, e)
	defer func() { satOK(t, a.Close()) }()
	d, e := newSATSolverDynamicSubsumption(a, probe, life.budget)
	satOK(t, e)
	defer func() { satOK(t, d.Close()) }()
	satOK(t, a.lemma.Push(0))
	satOK(t, a.lemma.Push(12))
	e = d.run()
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal(e)
		}
	} else {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
	}
	if !life.propagation.variables.literalMark[0] || satScalarTestUintSize(t, &a.lemma) != 2 || d.eliminated != 0 {
		t.Fatal("completed first mark retained")
	}
	satProbingAudit(t, probe, a, d)
	life.propagation.variables.literalMark[0] = false // explicit caller unwind.
}
