package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTProofUnitResolutionCountShortCircuit(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	a := proofTestBool(t, m, "a")
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(0, ASTOpNot, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	defer m.DelFuncDeclInfo(&info)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "zero-arity-not"), nil, s, info)
	astTestMust(t, e)
	f, e := m.MkApp(d, nil)
	astTestMust(t, e)
	p, e := m.MkProofHypothesis(f)
	astTestMust(t, e)
	q, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	r, e := m.MkProofUnitResolution([]ASTHandle{p, q, q})
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) {
			t.Fatal("source OR assertion precedes complement reads", e)
		}
	} else {
		astTestMust(t, e)
		proofTestFact(t, m, r, m.external(m.falseApp))
	}
}

func TestASTProofUnitResolutionLexicalScratchPeak(t *testing.T) {
	fixture := func(limit uint64) (*ASTManager, []ASTHandle) {
		b := arena.NewBudget(limit)
		s, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, ProofMode: ASTProofEnabled})
		astTestMust(t, e)
		t.Cleanup(func() {
			astTestMust(t, m.Close())
			astTestMust(t, q.Close())
			astTestMust(t, s.Close())
			if b.Used() != 0 {
				t.Errorf("fixture graph retained %d", b.Used())
			}
		})
		a, c := proofTestBool(t, m, "a"), proofTestBool(t, m, "c")
		lits := make([]ASTHandle, 19)
		lits[0] = c
		for i := 1; i < len(lits); i++ {
			lits[i] = a
		}
		clause := proofTestApp(t, m, ASTOpOr, lits...)
		p, e := m.MkProofHypothesis(clause)
		astTestMust(t, e)
		nc := proofTestApp(t, m, ASTOpNot, c)
		pc, e := m.MkProofHypothesis(nc)
		astTestMust(t, e)
		ps := make([]ASTHandle, 16)
		ps[0] = p
		for i := 1; i < len(ps); i++ {
			ps[i] = pc
		}
		fact := proofTestApp(t, m, ASTOpOr, lits[1:]...)
		// Warm the exact source result declaration/candidate payload classes.
		args := append(append([]ASTHandle{}, ps...), fact)
		for i := 0; i < 3; i++ {
			_, e = m.MkFamilyApp(0, ASTPrUnitResolution, nil, args)
			astTestMust(t, e)
		}
		return m, ps
	}
	m, _ := fixture(1 << 30)
	limit := m.BudgetField.Used() + 800
	if ASTSourceDebug {
		// Checked core descriptors and the source-only found svector
		// have larger temporary peaks than the release fixture.
		limit += 600
	}
	n, ps := fixture(limit)
	before := n.BudgetField.Used()
	_, e := n.MkProofUnitResolution(ps)
	astTestMust(t, e)
	if n.BudgetField.Used() != before {
		t.Fatal("lexical scratch/candidate not refunded", before, n.BudgetField.Used())
	}
}

func TestASTProofUnitResolutionBooleanOracle(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	var literals [6]ASTHandle
	for i := 0; i < 3; i++ {
		literals[2*i] = proofTestBool(t, m, string(rune('a'+i)))
		literals[2*i+1] = proofTestApp(t, m, ASTOpNot, literals[2*i])
	}
	truth := func(lit, assignment int) bool {
		v := assignment&(1<<(lit/2)) != 0
		if lit&1 != 0 {
			return !v
		}
		return v
	}
	cases := 0
	for x := 0; x < 6; x++ {
		for y := 0; y < 6; y++ {
			for z := 0; z < 6; z++ {
				clauseLits := [3]int{x, y, z}
				clause := proofTestApp(t, m, ASTOpOr, literals[x], literals[y], literals[z])
				p, e := m.MkProofHypothesis(clause)
				astTestMust(t, e)
				present := (1 << x) | (1 << y) | (1 << z)
				for removed := 1; removed < 64; removed++ {
					if removed & ^present != 0 {
						continue
					}
					ps := []ASTHandle{p}
					for l := 0; l < 6; l++ {
						if removed&(1<<l) == 0 {
							continue
						}
						q, e := m.MkProofHypothesis(literals[l^1])
						astTestMust(t, e)
						ps = append(ps, q)
					}
					r, e := m.MkProofUnitResolution(ps)
					astTestMust(t, e)
					var remaining []ASTHandle
					for _, l := range clauseLits {
						if removed&(1<<l) == 0 {
							remaining = append(remaining, literals[l])
						}
					}
					want := m.external(m.falseApp)
					if len(remaining) == 1 {
						want = remaining[0]
					}
					if len(remaining) > 1 {
						want = proofTestApp(t, m, ASTOpOr, remaining...)
					}
					proofTestFact(t, m, r, want)
					for assignment := 0; assignment < 8; assignment++ {
						premises := truth(x, assignment) || truth(y, assignment) || truth(z, assignment)
						for l := 0; l < 6; l++ {
							if removed&(1<<l) != 0 {
								premises = premises && truth(l^1, assignment)
							}
						}
						conclusion := false
						for _, l := range clauseLits {
							if removed&(1<<l) == 0 {
								conclusion = conclusion || truth(l, assignment)
							}
						}
						if premises && !conclusion {
							t.Fatal("independent Boolean entailment", clauseLits, removed, assignment)
						}
					}
					cases++
				}
			}
		}
	}
	if cases != 1116 {
		t.Fatal("complete finite matrix", cases)
	}
}
