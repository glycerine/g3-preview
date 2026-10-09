// Source: src/sat/sat_solver.h:286,445; src/sat/sat_solver.cpp:2818-2838.
package sat

import "testing"

func TestSATSolverUserBinaryPairPreservesFullLiteralWords(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	s.userBinClauses = append(s.userBinClauses, newSATSolverBinPair(Literal(0x80000001), Literal(0xfffffffe)))
	p := s.userBinClauses[0]
	if p.Literal1() != Literal(0x80000001) || p.Literal2() != Literal(0xfffffffe) {
		t.Fatal("source pair has two complete uint32 words", p.Literal1(), p.Literal2())
	}
}
