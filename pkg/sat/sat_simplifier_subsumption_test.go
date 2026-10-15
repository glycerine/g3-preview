// Source: src/sat/sat_simplifier.cpp:421–681.
// Independent finite branch oracles; no registered full-solver claim.
package sat

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
)

func TestSATSimplifierCleanupAll27AssignmentsAndUseCounts(t *testing.T) {
	for encoded := uint32(0); encoded < 27; encoded++ {
		t.Run(string(rune('a'+encoded)), func(t *testing.T) {
			s, l := satSimplifierFixture(t, 3)
			p := l.propagation
			satOK(t, s.useList.Init(3))
			satOK(t, s.kernel.InitVisited(3))
			c, e := p.clauses.MkClause([]Literal{0, 2, 4}, false)
			satOK(t, e)
			satOK(t, l.pushClause(false, c))
			satOK(t, s.useList.Insert(c))
			n := encoded
			want := make([]Literal, 0, 3)
			satisfied := false
			for v := uint32(0); v < 3; v++ {
				state := n % 3
				n /= 3
				value := [3]lbool.LBool{lbool.Undef, lbool.False, lbool.True}[state]
				satOK(t, p.variables.setAssignment(Literal(v*2), value))
				satOK(t, p.variables.setAssignment(Literal(v*2+1), -value))
				if value != lbool.False {
					want = append(want, Literal(v*2))
				}
				satisfied = satisfied || value == lbool.True
			}
			got, e := s.cleanupClause(c, true)
			satOK(t, e)
			if got != satisfied {
				t.Fatal("full-scan truth result", encoded, got, satisfied)
			}
			size, e := p.clauses.Size(c)
			satOK(t, e)
			if size != uint32(len(want)) {
				t.Fatal("source compacted size", size, want)
			}
			for i, x := range want {
				v, e := p.clauses.Literal(c, uint32(i))
				satOK(t, e)
				if v != x {
					t.Fatal("stable compacted literal", i, v, x)
				}
			}
			n = encoded
			for v := uint32(0); v < 3; v++ {
				state := n % 3
				n /= 3
				u, e := s.useList.Get(Literal(v * 2))
				satOK(t, e)
				count, e := u.Size()
				satOK(t, e)
				expected := uint32(1)
				if state == 1 {
					expected = 0
				}
				if count != expected {
					t.Fatal("actual lazy-use count", v, count, expected)
				}
			}
		})
	}
}
func TestSATSimplifierActualNumVarsUsesLevelAtRead(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	v := l.propagation.variables
	// Source-pointer safety-domain fixture: initialized columns have differing
	// logical lengths. The source getter is literally level.size(), not assignment.
	v.level = v.level[:1]
	satOK(t, s.initialize())
	if len(s.useList.records) != 2 || s.kernel.length != 2 {
		t.Fatal("source level getter must determine both actual planes", len(s.useList.records), s.kernel.length)
	}
}
