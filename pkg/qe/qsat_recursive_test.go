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

func TestQSATRecursiveCPPFormulaEquivalence(t *testing.T) {
	for _, mode := range qsatPreparationModes {
		t.Run(mode, func(t *testing.T) {
			c := qsatPrepareFixture(t, mode)
			f := c.f
			m := f.r.manager
			fmMust(t, m.RegisterDeclarationPlugins())
			s := qsatSearchOwner(t, f, qsatRecursiveQE)
			out := predicateRefs(t, f, ast.ASTHandle{})
			fmMust(t, s.eliminateRecursive(c.original, (*out)[0]))
			result := predicateValue(t, f, (*out)[0])
			has, err := m.HasQuantifiers(result)
			fmMust(t, err)
			if has {
				t.Fatal("quantifier remains in completed Boolean/linear projection")
			}
			format := func(h ast.ASTHandle) string {
				var b bytes.Buffer
				fmMust(t, writeExpression(m, f.r.config, &b, h))
				return b.String()
			}
			script := "(set-logic ALL)\n(declare-const free Bool)\n(declare-fun opaque (Bool) Bool)\n(assert (not (= " + format(c.original) + " " + format(result) + ")))\n"
			got := projectionCPP(t, script+"(check-sat)\n")
			if got == "unknown" {
				got = projectionCPP(t, script+"(check-sat-using (then simplify qe smt))\n")
			}
			if got != "unsat" {
				t.Fatal(got, script)
			}
			fmMust(t, s.Close())
			_ = predicateValue(t, f, (*out)[0])
		})
	}
}

func TestQSATRecursiveSharedDAGAndExtractionShift(t *testing.T) {
	c := qsatPrepareFixture(t, "exists")
	f := c.f
	m := f.r.manager
	fmMust(t, m.RegisterDeclarationPlugins())
	s := qsatSearchOwner(t, f, qsatRecursiveQE)
	root := f.own(m.MkIff(c.original, c.original))
	out := predicateRefs(t, f, ast.ASTHandle{})
	fmMust(t, s.eliminateRecursive(root, (*out)[0]))
	if s.rounds == 0 {
		t.Fatal("no recursive search")
	}
	result := predicateValue(t, f, (*out)[0])
	a, err := m.GetArg(result, 0)
	fmMust(t, err)
	b, err := m.GetArg(result, 1)
	fmMust(t, err)
	if a != b {
		t.Fatal("shared quantifier result not reused")
	}
	// A free de Bruijn variable above the extracted binder shifts down once.
	body := f.own(m.MkIff(f.variable(0, f.boolean), f.variable(1, f.boolean)))
	name, err := m.Symbols.Intern("bound")
	fmMust(t, err)
	q := f.own(m.MkQuantifier(ast.ASTQuantifierSpec{Sorts: []ast.ASTHandle{f.boolean}, Names: []usymbol.Symbol{name}, Body: body}))
	call, err := s.begin()
	fmMust(t, err)
	vars := call.vector()
	r, err := call.ref(body)
	fmMust(t, err)
	err = call.extractVariables(q, r, vars)
	if err == nil {
		h := predicateValue(t, f, r)
		left, e := m.GetArg(h, 0)
		fmMust(t, e)
		right, e := m.GetArg(h, 1)
		fmMust(t, e)
		if left != predicateValue(t, f, (*vars)[0]) {
			t.Fatal("declaration order changed")
		}
		index, e := m.GetVarIndex(right)
		fmMust(t, e)
		if index != 0 {
			t.Fatal("outer variable was not shifted", index)
		}
	}
	s.finish(call, &err)
	fmMust(t, err)
}

func TestQSATRecursiveAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"exists", "alternating", "blocked", "extract", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 16384 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					mode := phase
					if phase == "extract" || strings.Contains(phase, "trace") {
						mode = "exists"
					}
					c := qsatPrepareFixture(t, mode)
					f := c.f
					m := f.r.manager
					fmMust(t, m.RegisterDeclarationPlugins())
					var trace bytes.Buffer
					if strings.Contains(phase, "trace") {
						var w io.Writer = &trace
						if phase == "closed-trace" {
							w = closedPredicateTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(w)
						tags.Enable("qe")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					s := qsatSearchOwner(t, f, qsatRecursiveQE)
					out := predicateRefs(t, f, ast.ASTHandle{})
					var err error
					var call *qsatSearchCall
					var vars *[]ast.ASTRef
					var r ast.ASTRef
					if phase == "extract" {
						call, err = s.begin()
						fmMust(t, err)
						vars = call.vector()
						body, e := m.GetQuantifierBody(c.original)
						fmMust(t, e)
						r, err = call.ref(body)
						fmMust(t, err)
					}
					m.Budget().FailAfter(stop)
					if phase == "extract" {
						err = call.extractVariables(c.original, r, vars)
						s.finish(call, &err)
					} else {
						err = s.eliminateRecursive(c.original, (*out)[0])
					}
					m.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						var memout memory.Failure
						if !errors.As(err, &memout) || memout.Kind != memory.FailureMemout {
							t.Fatalf("%v\n%s", err, trace.String())
						}
					}
				})
				if t.Failed() {
					t.Fatal("recursive frontier", stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}
