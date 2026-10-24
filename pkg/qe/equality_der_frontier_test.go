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
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestEqualityDERAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "chain", "rational", "quantifier", "proof", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 2048 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := setupProofs(t, phase == "proof")
					sort := f.integer
					if phase == "rational" {
						family, err := f.m.ArithmeticFamilyID()
						must(t, err)
						sort, err = f.m.MkFamilySort(family, 0, nil)
						must(t, err)
						f.pin(t, sort)
					}
					x, y := f.variable(t, 0, sort), f.variable(t, 1, sort)
					value := f.numeral(t, "3")
					definition := f.equality(t, x, y)
					if phase == "rational" {
						n, err := f.m.NumeralService().NewValue()
						must(t, err)
						must(t, f.m.NumeralService().SetString(&n, "123456789012345678901234567890/7"))
						coefficient, err := f.m.ArithmeticNumeral(&n, false)
						must(t, err)
						f.pin(t, coefficient)
						must(t, f.m.NumeralService().Del(&n))
						product, err := f.m.MkArithmeticMul(coefficient, x)
						must(t, err)
						f.pin(t, product)
						definition = f.equality(t, product, y)
						value = y
					}
					forms := conjunctions(t, f, definition, f.equality(t, y, value), predicate(t, f, "frontier-p", x, y))
					body, err := f.m.MkAndArray([]ast.ASTHandle{definition, predicate(t, f, "frontier-p", x, y)})
					must(t, err)
					f.pin(t, body)
					q := exists(t, f, []ast.ASTHandle{sort, sort}, body)
					out := f.refs(t, ast.ASTHandle{}, ast.ASTHandle{})
					var trace bytes.Buffer
					if phase == "trace" || phase == "closed-trace" {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = closedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						for _, tag := range []string{"qe_lite", "qe_def", "top_sort", "der_bug"} {
							tags.Enable(tag)
						}
						t.Cleanup(tags.Finalize)
						f.config.Trace = tags
					}
					var d *qe.EqualityDER
					if phase != "constructor" {
						d = equalityReducer(t, f, nil, selector(t, f, []uint32{0, 1}, true))
					}
					f.budget.FailAfter(stop)
					switch phase {
					case "constructor":
						d, err = qe.NewEqualityDER(f.m, nil, f.config)
					case "quantifier", "proof":
						err = d.ApplyQuantifier(q, out[0], out[1])
					default:
						err = d.EliminateConjunctions(forms)
					}
					f.budget.FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						must(t, failure.Cleanup())
					}
					if phase == "constructor" && d != nil {
						must(t, d.Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.config.Trace.ShouldTrace("top_sort") && !strings.Contains(trace.String(), "DEFINITIONS:") {
							t.Fatal("missing source definition-order trace", trace.String())
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
				t.Fatal("no successful allocation frontier")
			}
		})
	}
}

func TestEqualityDERSparseDefinitionsTrace(t *testing.T) {
	f := setup(t)
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("der_bug")
	t.Cleanup(tags.Finalize)
	f.config.Trace = tags
	x, outer := f.variable(t, 0, f.integer), f.variable(t, 9, f.integer)
	forms := conjunctions(t, f, f.equality(t, x, outer), predicate(t, f, "sparse-p", x))
	d := equalityReducer(t, f, nil, selector(t, f, []uint32{0}, true))
	must(t, d.EliminateConjunctions(forms))
	if len(*forms) != 1 || f.get(t, (*forms)[0]) != predicate(t, f, "sparse-p", outer) {
		t.Fatal("sparse outer variable lost")
	}
	if tags.ShouldTrace("der_bug") && !strings.Contains(trace.String(), "vidx: 9\n") {
		t.Fatal("missing conditional source trace", trace.String())
	}
}
