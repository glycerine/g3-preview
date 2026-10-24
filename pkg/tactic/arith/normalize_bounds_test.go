package arith

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func normalizationFixture(t *testing.T, proofs bool) *NormalizeBoundsTactic {
	t.Helper()
	base, m := fixDLFixture(t, proofs)
	r, err := NewNormalizeBoundsTactic(m, base.context, base.globals, nil, base.services)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r
}
func TestNormalizeBoundsModelsProofsAndCoreDependencies(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprintf("int%v/proofs%v", integer, proofs), func(t *testing.T) {
				r := normalizationFixture(t, proofs)
				m := r.manager
				if !integer {
					p, err := r.context.NewValue()
					equationMust(t, err)
					t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
					key := "norm_int_only"
					equationMust(t, r.context.SetBoolName(&p, &key, false))
					equationMust(t, r.UpdateParams(&p))
				}
				x := fixDLConst(t, m, integer, "x")
				lower := "5"
				upper := "8"
				freshValue := "2"
				want := big.NewRat(7, 1)
				if !integer {
					lower = "1/3"
					upper = "2/3"
					freshValue = "1/6"
					want = big.NewRat(1, 2)
				}
				lowerOp := int32(3)
				if !integer {
					lowerOp = 5
				}
				goal := fixDLGoal(t, m, proofs, true, true, fixDLApp(t, m, lowerOp, x, fixDLNumber(t, m, integer, lower)), fixDLApp(t, m, 2, x, fixDLNumber(t, m, integer, upper)))
				boolean, err := m.MkBoolSort()
				equationMust(t, err)
				name, err := m.Symbols.Intern("premise")
				equationMust(t, err)
				label, err := m.MkConst(name, boolean)
				equationMust(t, err)
				fixDLPin(t, m, label)
				dep, err := m.DependencyLeaf(label)
				equationMust(t, err)
				form, err := goal.Form(0)
				equationMust(t, err)
				pr, err := goal.Proof(0)
				equationMust(t, err)
				equationMust(t, goal.Update(0, form, pr, dep))
				var result tactic.TacticResult
				t.Cleanup(func() { equationMust(t, result.Close()) })
				equationMust(t, r.Apply(goal, &result))
				depth, err := goal.Depth()
				equationMust(t, err)
				if depth != 1 || result.ModelConverter == nil {
					t.Fatal("normalization missing", depth)
				}
				form, err = goal.Form(0)
				equationMust(t, err)
				family0, op0, args0, _, err := r.bounds.app(form)
				equationMust(t, err)
				if family0 == m.BasicFamilyID() && op0 == ast.ASTOpNot {
					form = args0[0]
				}
				fresh, err := m.GetArg(form, 0)
				equationMust(t, err)
				family, _, args, app, err := r.bounds.app(fresh)
				equationMust(t, err)
				if !app || family != ast.NullFamilyID || len(args) != 0 || fresh == x {
					t.Fatal("expected fresh normalized variable", fresh)
				}
				for i := uint32(0); i < 2; i++ {
					h, e := goal.Form(i)
					equationMust(t, e)
					occurs, e := m.OccursExpr(x, h)
					equationMust(t, e)
					if occurs {
						t.Fatal("original variable remains")
					}
					if proofs {
						pr, e := goal.Proof(i)
						equationMust(t, e)
						fact, e := m.GetProofFact(pr)
						equationMust(t, e)
						if fact != h {
							t.Fatal("rewritten proof fact")
						}
					}
				}
				dep, err = goal.Dependency(0)
				equationMust(t, err)
				leaves, err := m.DependencyLinearize(dep, nil)
				equationMust(t, err)
				if len(leaves) != 1 || leaves[0] != label {
					t.Fatal("core dependency changed", leaves)
				}
				freshDecl, err := m.GetDecl(fresh)
				equationMust(t, err)
				originalDecl, err := m.GetDecl(x)
				equationMust(t, err)
				md, err := model.NewModel(m)
				equationMust(t, err)
				value := fixDLNumber(t, m, integer, freshValue)
				equationMust(t, md.CoreField.RegisterConstant(freshDecl, value))
				var out tactic.ModelRef
				equationMust(t, out.Set(&md.CoreField))
				t.Cleanup(func() { equationMust(t, out.Close()) })
				equationMust(t, r.Close())
				equationMust(t, result.ModelConverter.ApplyModel(&out, 0))
				converted, err := out.Get()
				equationMust(t, err)
				got, err := converted.ConstantInterpretation(originalDecl)
				equationMust(t, err)
				text, isInt, matched, err := m.ArithmeticNumeralText(got)
				equationMust(t, err)
				var rational big.Rat
				_, valid := rational.SetString(text)
				if !matched || !valid || isInt != integer || rational.Cmp(want) != 0 {
					t.Fatal("original value reconstruction", text, want.RatString())
				}
				present, err := converted.HasInterpretation(freshDecl)
				equationMust(t, err)
				if present {
					t.Fatal("fresh helper escaped the model filter")
				}
			})
		}
	}
}

func TestNormalizeBoundsNoOpAndConstructorParameterSnapshot(t *testing.T) {
	r := normalizationFixture(t, false)
	m := r.manager
	x := fixDLConst(t, m, true, "x")
	y := fixDLConst(t, m, false, "y")
	for _, form := range []ast.ASTHandle{fixDLApp(t, m, 3, x, fixDLNumber(t, m, true, "0")), fixDLApp(t, m, 2, x, fixDLNumber(t, m, true, "5")), fixDLApp(t, m, 3, y, fixDLNumber(t, m, false, "1/3"))} {
		goal := fixDLGoal(t, m, false, true, false, form)
		var result tactic.TacticResult
		equationMust(t, r.Apply(goal, &result))
		depth, err := goal.Depth()
		equationMust(t, err)
		if depth != 0 || result.ModelConverter != nil {
			t.Fatal("no-op normalized a target", depth)
		}
		equationMust(t, result.Close())
		equationMust(t, r.Cleanup())
	}
	p, err := r.context.NewValue()
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
	key := "norm_int_only"
	equationMust(t, r.context.SetBoolName(&p, &key, false))
	equationMust(t, r.UpdateParams(&p))
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, target.Close()) })
	translated, err := r.Translate(target)
	equationMust(t, err)
	equationMust(t, translated.IncRef())
	t.Cleanup(func() { equationMust(t, translated.DecRef()) })
	goal := fixDLGoal(t, m, false, false, false, fixDLApp(t, m, 3, y, fixDLNumber(t, m, false, "1/3")))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	depth, err := goal.Depth()
	equationMust(t, err)
	if depth != 1 {
		t.Fatal("live parameter update ignored")
	}
	equationMust(t, r.Cleanup())
	goal2 := fixDLGoal(t, m, false, false, false, fixDLApp(t, m, 3, y, fixDLNumber(t, m, false, "2/3")))
	equationMust(t, r.Apply(goal2, &result))
	depth, err = goal2.Depth()
	equationMust(t, err)
	if depth != 0 {
		t.Fatal("cleanup retained live update instead of constructor params")
	}
	y2 := fixDLConst(t, target, false, "y")
	targetGoal := fixDLGoal(t, target, false, false, false, fixDLApp(t, target, 3, y2, fixDLNumber(t, target, false, "1/3")))
	var targetResult tactic.TacticResult
	t.Cleanup(func() { equationMust(t, targetResult.Close()) })
	equationMust(t, translated.Apply(targetGoal, &targetResult))
	depth, err = targetGoal.Depth()
	equationMust(t, err)
	if depth != 0 {
		t.Fatal("translation used live update instead of constructor params")
	}
}

func TestNormalizeBoundsCPPApplyAndTrace(t *testing.T) {
	r := normalizationFixture(t, false)
	m := r.manager
	x := fixDLConst(t, m, true, "x")
	goal := fixDLGoal(t, m, false, false, false, fixDLApp(t, m, 3, x, fixDLNumber(t, m, true, "5")), fixDLApp(t, m, 2, x, fixDLNumber(t, m, true, "8")))
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("normalize_bounds_tactic")
	equationMust(t, r.BindTrace(tags))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	if tags.ShouldTrace("normalize_bounds_tactic") {
		if !strings.Contains(trace.String(), "x has_lower: 1 val: 5\n") || !strings.Contains(trace.String(), "operator() src/tactic/arith/normalize_bounds_tactic.cpp:144") {
			t.Fatal("normalization traces", trace.String())
		}
	} else if trace.Len() != 0 {
		t.Fatal("erased trace wrote output")
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		var out bytes.Buffer
		out.WriteString("(goals (goal ")
		size, err := goal.Size()
		equationMust(t, err)
		for i := uint32(0); i < size; i++ {
			form, e := goal.Form(i)
			equationMust(t, e)
			equationMust(t, r.bounds.writeExpr(&out, form))
			out.WriteByte(' ')
		}
		depth, err := goal.Depth()
		equationMust(t, err)
		fmt.Fprintf(&out, ":precision precise :depth %d))", depth)
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader("(set-logic QF_LIA)(declare-const x Int)(assert (>= x 5))(assert (<= x 8))(apply normalize-bounds)")
		native, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		fresh := regexp.MustCompile(`k![0-9]+`)
		tokens := func(s string) string {
			s = fresh.ReplaceAllString(s, "fresh")
			return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
		}
		if tokens(out.String()) != tokens(string(native)) {
			t.Fatalf("native apply differs\nGo: %s\nC++: %s", out.String(), native)
		}
	}
	refused := diagnostic.NewTraceTagsWithWriter(fixDLRefusedWriter{})
	refused.EnableAll(true)
	equationMust(t, r.BindTrace(refused))
	equationMust(t, r.Cleanup())
	goal2 := fixDLGoal(t, m, false, false, false, fixDLApp(t, m, 3, x, fixDLNumber(t, m, true, "5")))
	equationMust(t, r.Apply(goal2, &result))
}

func TestNormalizeBoundsAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		for _, proofs := range []bool{false, true} {
			succeeded := false
			for stop := int64(0); stop < 512; stop++ {
				t.Run(fmt.Sprintf("%s/proofs%v/%d", phase, proofs, stop), func(t *testing.T) {
					r := normalizationFixture(t, proofs)
					m := r.manager
					if phase == "constructor" {
						m.Budget().FailAfter(stop)
						fresh, err := NewNormalizeBoundsTactic(m, r.context, r.globals, nil, r.services)
						m.Budget().FailAfter(-1)
						if err == nil {
							succeeded = true
						} else if !errors.Is(err, arena.ErrInjectedFailure) {
							t.Fatal(err)
						}
						if fresh != nil {
							equationMust(t, fresh.Close())
						}
						return
					}
					x := fixDLConst(t, m, true, "x")
					goal := fixDLGoal(t, m, proofs, true, true, fixDLApp(t, m, 3, x, fixDLNumber(t, m, true, "5")), fixDLApp(t, m, 2, x, fixDLNumber(t, m, true, "8")))
					var result tactic.TacticResult
					t.Cleanup(func() { equationMust(t, result.Close()) })
					if phase == "cleanup" {
						equationMust(t, r.Apply(goal, &result))
					}
					m.Budget().FailAfter(stop)
					var err error
					if phase == "cleanup" {
						err = r.Cleanup()
					} else {
						err = r.Apply(goal, &result)
					}
					m.Budget().FailAfter(-1)
					if err == nil {
						succeeded = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
					if phase == "cleanup" {
						equationMust(t, r.drainRetired())
						equationMust(t, r.Apply(goal, &result))
					}
				})
				if succeeded {
					break
				}
			}
			if !succeeded {
				t.Fatal("never reached success", phase, proofs)
			}
		}
	}
}
