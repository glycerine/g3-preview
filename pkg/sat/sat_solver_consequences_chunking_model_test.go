// Source: src/sat/sat_solver.cpp:3157-3249. Independent finite fixtures.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATConsequenceChunkIndependentBinaryCNFs(t *testing.T) {
	for mask := uint32(0); mask < 16; mask++ {
		for K := uint32(1); K <= 2; K++ {
			t.Run(fmt.Sprintf("%d/%d", mask, K), func(t *testing.T) {
				s, _, _ := satSolverCheckFixture(t)
				for i := 0; i < 2; i++ {
					_, e := s.MkVar(true, true)
					satOK(t, e)
				}
				var clauses [][]Literal
				for i := uint32(0); i < 4; i++ {
					if mask&(1<<i) != 0 {
						r := []Literal{MakeLiteral(0, i&1 != 0), MakeLiteral(1, i&2 != 0)}
						clauses = append(clauses, r)
						satOK(t, s.MkClause(append([]Literal(nil), r...)))
					}
				}
				models := satConsequenceModels(clauses, nil, 2)
				r, e := s.Check(nil)
				satOK(t, e)
				if len(models) == 0 {
					if r != util.LFalse {
						t.Fatal(r)
					}
					return
				}
				if r != util.LTrue {
					t.Fatal(r)
				}
				var mdl RuntimeUintColumn
				mdl.Bind(s.runtime.memory)
				satOK(t, satMUSCopyColumn(&mdl.Raw, &s.model.Raw, 4))
				defer func() { satOK(t, mdl.Finalize()) }()
				out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
				satOK(t, e)
				defer func() { satOK(t, out.CloseChecked()) }()
				r, e = s.coreChunking(&mdl, satConsequenceInput{variables: []BoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, K)
				satOK(t, e)
				if r != util.LTrue {
					t.Fatal(r)
				}
				seen := map[uint32]bool{}
				n, e := out.rowCount()
				satOK(t, e)
				for i := uint32(0); i < n; i++ {
					row, e := out.captureRow(i)
					satOK(t, e)
					if row.count != 1 {
						t.Fatal("nil-assumption fixed literal core", row.count)
					}
					l, e := row.read(0)
					satOK(t, e)
					lit := Literal(l.index() ^ 1)
					v := uint32(lit.variable())
					if seen[v] {
						t.Fatal("duplicate", lit)
					}
					seen[v] = true
					for _, bits := range models {
						if ((bits>>v)&1 != 0) == lit.sign() {
							t.Fatal("nonconsequence", lit, models)
						}
					}
				}
				for v := uint32(0); v < 2; v++ {
					fixed := true
					for _, bits := range models {
						fixed = fixed && ((bits>>v)&1) == ((models[0]>>v)&1)
					}
					if fixed != seen[v] {
						t.Fatal("fixed set", v, fixed, seen)
					}
				}
				roots := satFullSolverRuntimeRoots(s, out)
				roots = append(roots, satMutexColumnRoots(&mdl)...)
				satScalarAuditRoots(t, s.Variables, satMutexNonzero(roots)...)
			})
		}
	}
}
func TestSATConsequenceChunkMultipleCoreUsesBruteForce(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	var model RuntimeUintColumn
	model.Bind(s.runtime.memory)
	for i := 0; i < 2; i++ {
		satOK(t, model.Push(uint32(util.LTrue)))
	}
	defer func() { satOK(t, model.Finalize()) }()
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	defer func() { satOK(t, out.CloseChecked()) }()
	r, e := s.coreChunking(&model, satConsequenceInput{variables: []BoolVar{0, 1}, count: 2}, satConsequenceLiterals(nil), out, 2)
	satOK(t, e)
	if r != util.LTrue {
		t.Fatal(r)
	}
	n, e := out.rowCount()
	satOK(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	roots := satFullSolverRuntimeRoots(s, out)
	roots = append(roots, satMutexColumnRoots(&model)...)
	satScalarAuditRoots(t, s.Variables, satMutexNonzero(roots)...)
}
