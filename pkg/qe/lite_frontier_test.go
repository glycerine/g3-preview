package qe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestLiteAllocationFrontiersAndClosedTrace(t *testing.T) {
	for _, phase := range []string{"constructor", "conjunctions", "expression", "constants", "constants-proofs", "traversal", "traversal-proofs", "array", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newFMFixtureProofs(t, phase == "constants-proofs" || phase == "traversal-proofs")
					m := f.r.manager
					forms, _, sorts, _ := fmDriverCase(f, "real")
					vector := f.conjunctions(forms...)
					body := f.own(m.MkAndArray(forms))
					formula := f.conjunctions(body)
					vars := f.conjunctions(f.constant("lite-k", f.integer), f.constant("lite-n", f.integer))
					if phase == "constants" || phase == "constants-proofs" {
						k, err := m.RefGet((*vars)[0])
						fmMust(t, err)
						n, err := m.RefGet((*vars)[1])
						fmMust(t, err)
						body = f.own(m.MkAndArray([]ast.ASTHandle{f.own(m.MkEq(k, f.number("3"))), f.predicate("lite-p", k, n)}))
						formula = f.conjunctions(body)
					}
					if phase == "traversal" || phase == "traversal-proofs" {
						var names []usymbol.Symbol
						for i := range sorts {
							name, err := m.Symbols.Intern(fmt.Sprint("lite-bound-", i))
							fmMust(t, err)
							names = append(names, name)
						}
						q := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: sorts, Names: names, Body: body}))
						formula = f.conjunctions(q)
					}
					if phase == "array" {
						fmMust(t, m.RegisterDeclarationPlugins())
						family, err := m.ArrayFamilyID()
						fmMust(t, err)
						sort := f.own(m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{f.integer}), f.integer))
						a := f.variable(0, sort)
						key, other := f.constant("lite-key", f.integer), f.constant("lite-other-key", f.integer)
						read := func(index ast.ASTHandle) ast.ASTHandle {
							return f.own(m.MkArrayAppArguments(family, ast.ASTArrayOpSelect, ast.SliceArguments([]ast.ASTHandle{a, index})))
						}
						vector = f.conjunctions(f.own(m.MkEq(read(key), f.number("7"))), f.own(m.MkEq(read(other), f.number("9"))))
					}
					proof := f.conjunctions(ast.ASTHandle{})
					var trace bytes.Buffer
					if phase == "trace" || phase == "closed-trace" {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = fmClosedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe_lite")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					var l *Lite
					var err error
					if phase != "constructor" {
						l = f.lite(true)
					}
					m.Budget().FailAfter(stop)
					switch phase {
					case "constructor":
						l, err = NewLite(m, nil, true, f.r.config)
					case "constants", "constants-proofs":
						err = l.EliminateConstants(vars, (*formula)[0])
					case "expression":
						err = l.EliminateIndices([]uint32{0, 1, 2}, true, (*formula)[0])
					case "traversal", "traversal-proofs":
						err = l.Rewrite((*formula)[0], (*proof)[0])
					default:
						err = l.EliminateConjunctions([]uint32{0, 1, 2}, true, vector)
					}
					m.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if phase == "constructor" && l != nil {
						fmMust(t, l.Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.r.config.Trace.ShouldTrace("qe_lite") {
							for _, text := range []string{"src/qe/qe_lite.cpp:2464", "src/qe/qe_lite.cpp:2475"} {
								if !strings.Contains(trace.String(), text) {
									t.Fatal("missing combined-phase trace", text, trace.String())
								}
							}
						}
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						var memout memory.Failure
						if !errors.As(err, &memout) || memout.Kind != memory.FailureMemout {
							t.Fatal(err)
						}
					}
				})
				if t.Failed() {
					t.Fatalf("%s allocation frontier %d", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}
