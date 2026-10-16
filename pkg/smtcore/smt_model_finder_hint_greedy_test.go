package smtcore

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestModelFinderHintGreedyNestedSuccessCycleRollbackAndDepthLimit(t *testing.T) {
	for _, cyclic := range []bool{false, true} {
		name := "acyclic"
		if cyclic {
			name = "cyclic"
		}
		t.Run(name, func(t *testing.T) {
			m := astTestManager(t)
			b, e := m.MkBoolSort()
			astTestMust(t, e)
			pin := func(h ast.ASTHandle) ast.ASTHandle {
				t.Helper()
				astTestMust(t, m.IncRef(h))
				t.Cleanup(func() { astTestMust(t, m.DecRef(h)) })
				return h
			}
			v, e := m.MkVar(0, b)
			astTestMust(t, e)
			pin(v)
			var q [2]ast.ASTHandle
			for i := range q {
				q[i], e = m.MkQuantifier(ast.ASTQuantifierSpec{
					Forall: true, Sorts: []ast.ASTHandle{b},
					Names: []symbol.Symbol{symbol.Numerical(uint32(i))}, Body: v,
				})
				astTestMust(t, e)
				pin(q[i])
			}
			var f, g ast.ASTHandle
			f, e = m.MkFuncDecl(symbol.Numerical(10), nil, b, ast.ASTFuncDeclInfo{})
			astTestMust(t, e)
			pin(f)
			g, e = m.MkFuncDecl(symbol.Numerical(11), nil, b, ast.ASTFuncDeclInfo{})
			astTestMust(t, e)
			pin(g)
			fApp, e := m.MkApp(f, nil)
			astTestMust(t, e)
			pin(fApp)
			gApp, e := m.MkApp(g, nil)
			astTestMust(t, e)
			pin(gApp)
			gDef := v
			if cyclic {
				gDef = fApp
			}
			s := newHintSearchState()
			var trace bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&trace)
			tags.Enable("model_finder_hint")
			s.trace = tags
			s.infos[q[0]] = &hintQuantifierInfo{
				nonGround: []ast.ASTHandle{f},
				macros:    []hintConditionalMacro{{function: f, definition: gApp, satisfyAtom: true}},
			}
			s.infos[q[1]] = &hintQuantifierInfo{
				nonGround: []ast.ASTHandle{f, g},
				macros:    []hintConditionalMacro{{function: g, definition: gDef, satisfyAtom: true}},
			}
			astTestMust(t, s.makeQuantifierIndexes(q[:]))
			limited, e := s.greedyFunction(m, f, hintGreedyMaxDepth)
			astTestMust(t, e)
			if limited || len(s.satisfied.scopes) != 0 || len(s.definitions) != 0 {
				t.Fatal("source depth ten must return before state mutation")
			}
			found, e := s.searchFrom(m, f)
			astTestMust(t, e)
			if tags.ShouldTrace("model_finder_hint") {
				const footer = "------------------------------------------------\n"
				const first = "-------- [model_finder_hint] greedy src/smt/smt_model_finder.cpp:2841 ---------\n" +
					"greedy depth: 0, f: k!10\nfs:\n\nsatisfied:\n\nresidue:\n\n" + footer
				const candidates = "-------- [model_finder_hint] greedy src/smt/smt_model_finder.cpp:2925 ---------\n" +
					"candidates from residue:\nk!11 \n" + footer
				const second = "-------- [model_finder_hint] greedy src/smt/smt_model_finder.cpp:2841 ---------\n" +
					"greedy depth: 1, f: k!11\nfs:\nk!10 \nsatisfied:\nnull \nresidue:\nnull \n" + footer
				prefix := first + candidates + second
				got := trace.String()
				if cyclic {
					if got != prefix {
						t.Fatalf("source cyclic greedy TRACE: got %q, want %q", got, prefix)
					}
				} else {
					const foundHead = "-------- [model_finder_hint] greedy src/smt/smt_model_finder.cpp:2917 ---------\n" +
						"found subset that is satisfied by macros\nfs:\n"
					const foundTail = "\nsatisfied:\nnull null \nresidue:\n\n" + footer
					if got != prefix+foundHead+"k!10 k!11 "+foundTail && got != prefix+foundHead+"k!11 k!10 "+foundTail {
						t.Fatalf("source greedy TRACE: got %q", got)
					}
				}
			} else if trace.Len() != 0 {
				t.Fatal("erased residue-candidate TRACE", trace.String())
			}
			if found == cyclic {
				t.Fatal("greedy result", found, "cyclic", cyclic)
			}
			if cyclic {
				if !s.satisfied.empty() || !s.residue.empty() || len(s.definitions) != 0 || len(s.satisfied.scopes) != 0 || len(s.residue.scopes) != 0 || s.infos[q[0]].theOne != (ast.ASTHandle{}) || s.infos[q[1]].theOne != (ast.ASTHandle{}) {
					t.Fatal("failed branch must pop both scopes and erase m_fs")
				}
			} else {
				if !s.satisfied.contains(q[0]) || !s.satisfied.contains(q[1]) || !s.residue.empty() || len(s.definitions) != 2 || len(s.satisfied.scopes) != 2 || s.infos[q[0]].theOne != f || s.infos[q[1]].theOne != g {
					t.Fatal("source found_satisfied_subset retains winning branch")
				}
			}
		})
	}
}
