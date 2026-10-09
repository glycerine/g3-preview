// Source: src/sat/sat_simplifier.cpp; src/sat/sat_simplifier.h; src/sat/sat_simplifier_params.pyg.
// These are independent source-branch fixtures, not registered user-scope Check coverage.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func satSimplifierFixture(t *testing.T, n uint32) (*satSimplifier, *satSolverClauseLifecycle) {
	t.Helper()
	d, g := satDecisionFixture(t, n)
	c, l := g.cleaner, g.lifecycle()
	satOK(t, l.bindIntegrity(g))
	params, e := l.propagation.config.context.NewValue()
	satOK(t, e)
	s, e := newSATSimplifier(l, c, d, &params)
	satOK(t, l.propagation.config.context.Del(&params))
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, l
}
func TestSATSimplifierFullSourceDefaultsAndThreeCalls(t *testing.T) {
	s, _ := satSimplifierFixture(t, 3)
	if !s.subsumption || !s.resolution || !s.elimVars || s.elimBlocked || s.blockedAt != 2 || s.blockedLimit != 100000000 || s.subsumptionLimit != 100000000 || s.resLimit != 500000000 || s.occCutoff != 10 || s.occRanges != [3]uint32{8, 5, 3} || s.litRanges != [3]uint32{700, 400, 300} || s.clsCutoffs != [2]uint32{100000000, 700000000} {
		t.Fatal("all17 actual defaults")
	}
	for i := uint32(0); i < 3; i++ {
		satOK(t, s.Run(false))
		if s.numCalls != i+1 {
			t.Fatal("actual default full orchestration calls", s.numCalls)
		}
	}
	satOK(t, s.Run(true))
	if s.numCalls != 3 {
		t.Fatal("learned source route does not increment calls")
	}
}
func TestSATSimplifierDefaultSubsumptionThenActualResolution(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	p := l.propagation
	for i := uint32(1); i < 4; i++ {
		satOK(t, p.variables.external.Write(i, 1))
	}
	c1, e := l.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, e)
	_, e = l.MkClauseCore(3, []Literal{1, 2, 4}, false)
	satOK(t, e)
	_, e = l.MkClauseCore(4, []Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	satOK(t, p.clauses.MarkStrengthened(c1))
	satOK(t, s.Run(false))
	x, e := p.variables.eliminated.Read(0)
	satOK(t, e)
	if x != 1 || s.numElimVars != 1 || s.numSubsumed == 0 || len(l.Original) != 0 {
		t.Fatal("actual removal schedules resolution", x, s.numElimVars, s.numSubsumed, len(l.Original))
	}
	// The real model converter is required, not a synthesized projection.
	if p.converter.Size() == 0 {
		t.Fatal("source resolution published actual model conversion")
	}
}
func TestSATSimplifierLearnedTransfersAndFrozenOwnership(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	for i := uint32(0); i < 4; i++ {
		satOK(t, l.propagation.variables.external.Write(i, 1))
	}
	small, e := l.MkClauseCore(3, []Literal{0, 2, 4}, true)
	satOK(t, e)
	big, e := l.MkClauseCore(4, []Literal{0, 2, 4, 6}, false)
	satOK(t, e)
	frozen := satCleanerClause(t, l, []Literal{1, 3, 5}, true, true)
	satOK(t, l.propagation.clauses.MarkStrengthened(small))
	satOK(t, s.Run(true))
	if l.propagation.clauses.IsLive(big) || !l.propagation.clauses.IsLive(small) || !l.propagation.clauses.IsLive(frozen) {
		t.Fatal("subsumption and true frozen ownership")
	}
	learned, e := l.propagation.clauses.Learned(small)
	satOK(t, e)
	if learned {
		t.Fatal("learned source subsumer becomes original")
	}
}

func TestSATSimplifierDefaultBlockedAtThirdCallUsesRealModelConverter(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	c, e := l.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, e)
	for i := uint32(0); i < 2; i++ {
		satOK(t, s.Run(false))
		if !l.propagation.clauses.IsLive(c) || s.numBlocked != 0 {
			t.Fatal("source default blocked_at has not fired", i)
		}
	}
	satOK(t, s.Run(false))
	if l.propagation.clauses.IsLive(c) || len(l.Original) != 0 || s.numBlocked != 1 || l.propagation.converter.Size() != 4 {
		t.Fatal("BLOCKED entry followed by three source empty-variable elimination entries", s.numBlocked, l.propagation.converter.Size())
	}
}

func satSimplifierFormulaValue(clauses [][]Literal, model Model) bool {
	for _, c := range clauses {
		yes := false
		for _, x := range c {
			v := model[uint32(x)>>1] == lbool.True
			if x&1 != 0 {
				v = !v
			}
			yes = yes || v
		}
		if !yes {
			return false
		}
	}
	return true
}
func TestSATSimplifierIndependentFiniteFullDefaultModelOracle(t *testing.T) {
	fixtures := [][][]Literal{
		{{0, 2, 4}, {0, 2, 4, 6}},
		{{0, 2, 4}, {1, 2, 4}, {0, 2, 4, 6}},
		{{0, 2}, {1, 2}, {0, 2, 4}, {3, 4, 6}},
		{{0, 2, 4}, {1, 3, 4}, {0, 3, 6}, {1, 2, 6}},
	}
	for fi, input := range fixtures {
		t.Run(string(rune('a'+fi)), func(t *testing.T) {
			s, l := satSimplifierFixture(t, 4)
			p := l.propagation
			// Actual external variables exercise source guards, without turning off any
			// default algorithm. Fixture b leaves its real pivot internal for resolution.
			for v := uint32(0); v < 4; v++ {
				if fi != 1 || v != 0 {
					satOK(t, p.variables.external.Write(v, 1))
				}
			}
			for _, cl := range input {
				r, e := l.MkClauseCore(uint32(len(cl)), cl, false)
				satOK(t, e)
				if r != (StoredClauseRef{}) {
					satOK(t, p.clauses.MarkStrengthened(r))
				}
			}
			satOK(t, s.Run(false))
			reducedCount := 0
			for mask := uint32(0); mask < 16; mask++ {
				model := make(Model, 4)
				for v := uint32(0); v < 4; v++ {
					model[v] = lbool.False
					if mask&(1<<v) != 0 {
						model[v] = lbool.True
					}
				}
				reduced := !p.inconsistent
				for v := uint32(0); v < 4; v++ {
					assigned, e := p.variables.valueVar(BoolVar(v))
					satOK(t, e)
					if assigned != lbool.Undef && assigned != model[v] {
						reduced = false
					}
				}
				for _, r := range l.Original {
					n, e := p.clauses.Size(r)
					satOK(t, e)
					yes := false
					for j := uint32(0); j < n; j++ {
						x, e := p.clauses.Literal(r, j)
						satOK(t, e)
						v := model[uint32(x)>>1] == lbool.True
						if x&1 != 0 {
							v = !v
						}
						yes = yes || v
					}
					reduced = reduced && yes
				}
				// Binary source clauses reside in real watch lists, without clause records.
				for idx := uint32(0); idx < 8; idx++ {
					wl, e := p.variables.watches.Get(idx)
					satOK(t, e)
					n, e := wl.Size()
					satOK(t, e)
					for j := uint32(0); j < n; j++ {
						w, e := wl.Read(j)
						satOK(t, e)
						if w.Binary() {
							b, e := w.Literal()
							satOK(t, e)
							a := Literal(idx).Negated()
							va := model[uint32(a)>>1] == lbool.True
							if a&1 != 0 {
								va = !va
							}
							vb := model[uint32(b)>>1] == lbool.True
							if b&1 != 0 {
								vb = !vb
							}
							reduced = reduced && (va || vb)
						}
					}
				}
				if reduced {
					reducedCount++
					// Actual solver omits eliminated variables from its projected model;
					// model_converter's DEBUG contract requires their l_undef.
					for v := uint32(0); v < 4; v++ {
						eliminated, e := p.variables.eliminated.Read(v)
						satOK(t, e)
						if eliminated != 0 {
							model[v] = lbool.Undef
						}
					}
					satOK(t, p.converter.Apply(model))
					if !satSimplifierFormulaValue(input, model) {
						t.Fatal("real converter reconstructed a nonmodel", mask, model)
					}
				}
			}
			if reducedCount == 0 {
				t.Fatal("satisfiable finite fixture lost every model")
			}
		})
	}
}

func TestSATSimplifierEmptyBinaryScanConsumesSharedSolverRNG(t *testing.T) {
	s, _ := satSimplifierFixture(t, 0)
	expected := s.decision.rng
	expected.Next()
	satOK(t, s.Run(false))
	if got, want := s.decision.rng.Next(), expected.Next(); got != want {
		t.Fatal("source unconditional binary-scan RNG draw", got, want)
	}
}
