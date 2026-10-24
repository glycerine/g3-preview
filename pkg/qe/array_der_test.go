package qe_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/qe"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func selector(t *testing.T, f *fixture, indices []uint32, positive bool) *qe.VariableTest {
	t.Helper()
	v, err := qe.NewVariableSetTest(f.m, indices, positive)
	must(t, err)
	t.Cleanup(func() { must(t, v.Close()) })
	return v
}
func reducer(t *testing.T, f *fixture, v *qe.VariableTest) *qe.ArrayDER {
	t.Helper()
	d, err := qe.NewArrayDER(f.m, v, f.config)
	must(t, err)
	t.Cleanup(func() { must(t, d.Close()) })
	return d
}

func TestArrayDERSelectSubstitutionBothOrientationsAndMultipleIndices(t *testing.T) {
	for _, multi := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("multi=%v/reverse=%v", multi, reverse), func(t *testing.T) {
				f := setup(t)
				domains := []ast.ASTHandle{f.integer}
				if multi {
					domains = append(domains, f.integer)
				}
				sort := f.arraySort(t, domains, f.integer)
				a := f.variable(t, 0, sort)
				k := f.constant(t, "array-key", f.integer)
				j := f.constant(t, "array-other-key", f.integer)
				args := []ast.ASTHandle{a, k}
				other := []ast.ASTHandle{a, j}
				if multi {
					args = append(args, j)
					other = append(other, k)
				}
				seven, nine := f.numeral(t, "7"), f.numeral(t, "9")
				read := f.selectArray(t, args...)
				definition := f.equality(t, read, seven)
				if reverse {
					definition = f.equality(t, seven, read)
				}
				conjs := f.refs(t, definition, f.equality(t, f.selectArray(t, other...), nine))
				d := reducer(t, f, selector(t, f, []uint32{0}, true))
				must(t, d.Apply(conjs))
				if !f.m.IsTrue(f.get(t, conjs[0])) {
					t.Fatal("read definition was not removed")
				}
				remaining := f.get(t, conjs[1])
				newRead := f.arg(t, remaining, 0)
				store := f.arg(t, newRead, 0)
				op, err := f.m.GetDeclKind(store)
				must(t, err)
				if op != ast.ASTArrayOpStore || f.arg(t, store, 0) != a {
					t.Fatal("missing typed store substitution")
				}
				for i, h := range args {
					if f.arg(t, store, uint32(i)) != h {
						t.Fatal("store argument order", i)
					}
				}
				if f.arg(t, store, uint32(len(args))) != seven || f.arg(t, remaining, 1) != nine {
					t.Fatal("substitution changed numeral values")
				}
			})
		}
	}
}

func TestArrayDERSelectFreshnessAndBooleanEqualityRefusals(t *testing.T) {
	for _, mode := range []string{"not-selected", "value-occurs", "index-occurs", "boolean-iff", "constant-array"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			rangeSort := f.integer
			if mode == "boolean-iff" {
				rangeSort = f.boolean
			}
			sort := f.arraySort(t, []ast.ASTHandle{f.integer}, rangeSort)
			a := f.variable(t, 0, sort)
			if mode == "constant-array" {
				a = f.constant(t, "fixed-array", sort)
			}
			k := f.constant(t, "key", f.integer)
			if mode == "index-occurs" {
				name, err := f.m.Symbols.Intern("array-index")
				must(t, err)
				decl, err := f.m.MkFuncDecl(name, []ast.ASTHandle{sort}, f.integer, ast.ASTFuncDeclInfo{})
				must(t, err)
				k, err = f.m.MkApp(decl, []ast.ASTHandle{a})
				must(t, err)
				f.pin(t, k)
			}
			value := f.numeral(t, "7")
			if mode == "value-occurs" {
				value = f.selectArray(t, a, f.numeral(t, "0"))
			}
			if mode == "boolean-iff" {
				var err error
				value, err = f.m.MkTrue()
				must(t, err)
				f.pin(t, value)
			}
			original := f.equality(t, f.selectArray(t, a, k), value)
			conjs := f.refs(t, original)
			indices := []uint32{0}
			if mode == "not-selected" {
				indices = nil
			}
			d := reducer(t, f, selector(t, f, indices, true))
			must(t, d.Apply(conjs))
			if f.get(t, conjs[0]) != original {
				t.Fatal("source refusal changed input")
			}
		})
	}
}

func TestArrayDERSubstitutionShiftsInsideQuantifierAndPattern(t *testing.T) {
	f := setup(t)
	sort := f.arraySort(t, []ast.ASTHandle{f.integer}, f.integer)
	a := f.variable(t, 0, sort)
	shifted := f.variable(t, 1, sort)
	bound := f.variable(t, 0, f.integer)
	seven := f.numeral(t, "7")
	key := f.constant(t, "outside-key", f.integer)
	read := f.selectArray(t, shifted, bound)
	pattern, err := f.m.MkPattern([]ast.ASTHandle{read})
	must(t, err)
	f.pin(t, pattern)
	name, err := f.m.Symbols.Intern("inner-x")
	must(t, err)
	q, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: f.equality(t, read, bound), Patterns: []ast.ASTHandle{pattern}})
	must(t, err)
	f.pin(t, q)
	conjs := f.refs(t, f.equality(t, f.selectArray(t, a, key), seven), q)
	d := reducer(t, f, selector(t, f, []uint32{0}, true))
	must(t, d.Apply(conjs))
	got := f.get(t, conjs[1])
	body, err := f.m.GetQuantifierBody(got)
	must(t, err)
	store := f.arg(t, f.arg(t, body, 0), 0)
	if f.arg(t, store, 0) != shifted || f.arg(t, store, 1) != key || f.arg(t, store, 2) != seven || f.arg(t, body, 1) != bound {
		t.Fatal("nested substitution captured a de Bruijn variable")
	}
	newPattern, err := f.m.GetPattern(got, 0)
	must(t, err)
	if f.arg(t, f.arg(t, newPattern, 0), 0) != store {
		t.Fatal("quantifier pattern was not substituted")
	}
}

func TestArrayDERIndexDisequalityAndSourceFreshnessGuards(t *testing.T) {
	for _, mode := range []string{"one-index", "multiple-indices", "duplicate-indices", "different-index", "constant-index", "other-use", "index-in-array", "pattern-use", "boolean-iff", "complement"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			domains := []ast.ASTHandle{f.integer}
			rangeSort := f.integer
			if mode == "multiple-indices" || mode == "duplicate-indices" {
				domains = append(domains, f.integer)
			}
			if mode == "boolean-iff" {
				rangeSort = f.boolean
			}
			sort := f.arraySort(t, domains, rangeSort)
			a, b := f.constant(t, "array-a", sort), f.constant(t, "array-b", sort)
			x := f.variable(t, 0, f.integer)
			y := f.variable(t, 1, f.integer)
			leftArgs, rightArgs := []ast.ASTHandle{a, x}, []ast.ASTHandle{b, x}
			if mode == "multiple-indices" {
				leftArgs = append(leftArgs, y)
				rightArgs = append(rightArgs, y)
			}
			if mode == "duplicate-indices" {
				leftArgs = append(leftArgs, x)
				rightArgs = append(rightArgs, x)
			}
			if mode == "different-index" {
				rightArgs[1] = y
			}
			if mode == "constant-index" {
				leftArgs[1] = f.numeral(t, "0")
				rightArgs[1] = leftArgs[1]
			}
			if mode == "index-in-array" {
				stored, err := f.m.MkArrayAppArguments(f.arrayFamily, ast.ASTArrayOpStore, ast.SliceArguments([]ast.ASTHandle{a, x, f.numeral(t, "0")}))
				must(t, err)
				f.pin(t, stored)
				leftArgs[0] = stored
			}
			original := f.not(t, f.equality(t, f.selectArray(t, leftArgs...), f.selectArray(t, rightArgs...)))
			forms := []ast.ASTHandle{original}
			if mode == "other-use" {
				forms = append(forms, f.equality(t, x, f.numeral(t, "3")))
			}
			if mode == "pattern-use" {
				name, err := f.m.Symbols.Intern("pattern-only-use")
				must(t, err)
				decl, err := f.m.MkFuncDecl(name, []ast.ASTHandle{f.integer}, f.integer, ast.ASTFuncDeclInfo{})
				must(t, err)
				term, err := f.m.MkApp(decl, []ast.ASTHandle{x})
				must(t, err)
				f.pin(t, term)
				pat, err := f.m.MkPattern([]ast.ASTHandle{term})
				must(t, err)
				f.pin(t, pat)
				body, err := f.m.MkTrue()
				must(t, err)
				q, err := f.m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{f.integer}, Names: []usymbol.Symbol{name}, Body: body, Patterns: []ast.ASTHandle{pat}})
				must(t, err)
				forms = append(forms, f.pin(t, q))
			}
			conjs := f.refs(t, forms...)
			selected := selector(t, f, []uint32{0, 1}, true)
			if mode == "complement" {
				selected = selector(t, f, []uint32{7}, false)
			}
			d := reducer(t, f, selected)
			must(t, d.Apply(conjs))
			wantChange := mode == "one-index" || mode == "multiple-indices" || mode == "duplicate-indices" || mode == "complement"
			if wantChange {
				want := f.not(t, f.equality(t, a, b))
				if f.get(t, conjs[0]) != want {
					t.Fatal("read disequality was not replaced by array disequality")
				}
			} else if f.get(t, conjs[0]) != original {
				t.Fatal("source freshness refusal changed the literal")
			}
		})
	}
}

func TestArrayDERSelectorUpdateCancellationAndOwnerGuards(t *testing.T) {
	f := setup(t)
	sort := f.arraySort(t, []ast.ASTHandle{f.integer}, f.integer)
	a := f.variable(t, 0, sort)
	zero := f.numeral(t, "0")
	eq := f.equality(t, f.selectArray(t, a, zero), zero)
	conjs := f.refs(t, eq)
	d := reducer(t, f, selector(t, f, nil, true))
	must(t, d.Apply(conjs))
	if f.get(t, conjs[0]) != eq {
		t.Fatal("empty bound set selected array")
	}
	must(t, d.SetVariableTest(selector(t, f, []uint32{0}, true)))
	f.m.ResourceLimit().IncCancel()
	err := d.Apply(conjs)
	wantReason := f.m.ResourceLimit().CancelMessage()
	f.m.ResourceLimit().DecCancel()
	var canceled tactic.TacticException
	if !errors.As(err, &canceled) || canceled.Message() != wantReason {
		t.Fatal("source cancellation", err)
	}
	if f.get(t, conjs[0]) != eq {
		t.Fatal("canceled call changed the formula")
	}
	must(t, d.Apply(conjs))
	if !f.m.IsTrue(f.get(t, conjs[0])) {
		t.Fatal("selector update or cancellation recovery failed")
	}
	copy := *d
	if err := copy.Apply(conjs); err == nil {
		t.Fatal("copied reducer accepted")
	}
	must(t, d.Close())
	if err := d.Apply(conjs); err == nil {
		t.Fatal("closed reducer accepted")
	}
}

type closedTrace struct{}

func (closedTrace) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestArrayDERTraceClosedStreamAndAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"selector", "constructor", "select", "disequality", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 2048 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := setup(t)
					sort := f.arraySort(t, []ast.ASTHandle{f.integer}, f.integer)
					a := f.variable(t, 0, sort)
					k := f.constant(t, "frontier-key", f.integer)
					j := f.constant(t, "frontier-other-key", f.integer)
					seven := f.numeral(t, "7")
					conjs := f.refs(t, f.equality(t, f.selectArray(t, a, k), seven), f.equality(t, f.selectArray(t, a, j), f.numeral(t, "9")))
					var out bytes.Buffer
					if phase == "trace" || phase == "closed-trace" {
						var writer io.Writer = &out
						if phase == "closed-trace" {
							writer = closedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe_lite")
						t.Cleanup(tags.Finalize)
						f.config.Trace = tags
					}
					var v *qe.VariableTest
					var d *qe.ArrayDER
					var err error
					if phase != "selector" {
						v = selector(t, f, []uint32{0}, true)
					}
					if phase != "selector" && phase != "constructor" {
						d = reducer(t, f, v)
					}
					if phase == "disequality" {
						b := f.constant(t, "frontier-array-b", sort)
						x := f.variable(t, 0, f.integer)
						conjs = f.refs(t, f.not(t, f.equality(t, f.selectArray(t, f.constant(t, "frontier-array-a", sort), x), f.selectArray(t, b, x))))
					}
					f.budget.FailAfter(stop)
					if phase == "selector" {
						v, err = qe.NewVariableSetTest(f.m, []uint32{0, 7}, false)
					} else if phase == "constructor" {
						d, err = qe.NewArrayDER(f.m, v, f.config)
					} else {
						err = d.Apply(conjs)
					}
					f.budget.FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						must(t, failure.Cleanup())
					}
					if phase == "selector" && v != nil {
						must(t, v.Close())
					}
					if phase == "constructor" && d != nil {
						must(t, d.Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.config.Trace.ShouldTrace("qe_lite") {
							if !strings.Contains(out.String(), "src/qe/qe_lite.cpp:889") || !strings.Contains(out.String(), " = 7\n") {
								t.Fatal(out.String())
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
					t.Fatal("array DER allocation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful allocation frontier")
			}
		})
	}
}
