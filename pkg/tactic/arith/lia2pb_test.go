package arith

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func lia2pbFixture(t *testing.T) *LIA2PBTactic {
	t.Helper()
	base, m := fixDLFixture(t, false)
	r, err := NewLIA2PBTactic(m, base.context, base.globals, nil, base.services)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r
}

func TestLIA2PBConvertedDisequalities(t *testing.T) {
	r := lia2pbFixture(t)
	m := r.manager
	x := fixDLConst(t, m, true, "x")
	forms := diffNeqTestBounds(t, m, x, "7")
	for i := 0; i < 6; i++ {
		eq, err := m.MkEq(x, fixDLNumber(t, m, true, fmt.Sprint(i)))
		equationMust(t, err)
		fixDLPin(t, m, eq)
		not, err := m.MkNot(eq)
		equationMust(t, err)
		fixDLPin(t, m, not)
		forms = append(forms, not)
	}
	goal := fixDLGoal(t, m, false, true, false, forms...)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	var output bytes.Buffer
	equationMust(t, goal.Display(&output))
	if strings.Count(output.String(), "(not (= (+ k!0 (* 2 k!1) (* 4 k!2))") != 6 {
		t.Fatal("weighted disequality encoding changed", output.String())
	}
}

func TestLIA2PBModelGridAndBoundDependencies(t *testing.T) {
	for upper := 2; upper <= 9; upper++ {
		t.Run(fmt.Sprint(upper), func(t *testing.T) {
			r := lia2pbFixture(t)
			m := r.manager
			x := fixDLConst(t, m, true, "x")
			goal := fixDLGoal(t, m, false, true, true, diffNeqTestBounds(t, m, x, fmt.Sprint(upper))...)
			var labels []ast.ASTHandle
			b, err := m.MkBoolSort()
			equationMust(t, err)
			for i, name := range []string{"lower", "upper"} {
				sym, e := m.Symbols.Intern(name)
				equationMust(t, e)
				label, e := m.MkConst(sym, b)
				equationMust(t, e)
				fixDLPin(t, m, label)
				labels = append(labels, label)
				dep, e := m.DependencyLeaf(label)
				equationMust(t, e)
				form, e := goal.Form(uint32(i))
				equationMust(t, e)
				equationMust(t, goal.Update(uint32(i), form, ast.ASTHandle{}, dep))
			}
			var result tactic.TacticResult
			t.Cleanup(func() { equationMust(t, result.Close()) })
			equationMust(t, r.Apply(goal, &result))
			bits := 0
			for n := upper; n != 0; n >>= 1 {
				bits++
			}
			size, err := goal.Size()
			equationMust(t, err)
			if size != uint32(2+2*bits) || result.ModelConverter == nil || len(r.newDeps) != 1 {
				t.Fatal("conversion count/model/dependency roots", size, bits, len(r.newDeps))
			}
			for i := uint32(0); i < 2; i++ {
				dep, e := goal.Dependency(i)
				equationMust(t, e)
				for _, label := range labels {
					found, e := m.DependencyContains(dep, label)
					equationMust(t, e)
					if !found {
						t.Fatal("rewritten bound lost substitution dependency", i)
					}
				}
			}
			var helpers []ast.ASTHandle
			for i := 0; i < bits; i++ {
				form, e := goal.Form(uint32(2 + 2*i))
				equationMust(t, e)
				helper, e := m.GetArg(form, 1)
				equationMust(t, e)
				decl, e := m.GetDecl(helper)
				equationMust(t, e)
				helpers = append(helpers, decl)
				dep, e := goal.Dependency(uint32(2 + 2*i))
				equationMust(t, e)
				if !dep.IsNull() {
					t.Fatal("helper bound must have no original assumptions")
				}
			}
			equationMust(t, r.Close())
			decl, err := m.GetDecl(x)
			equationMust(t, err)
			for value := 0; value <= upper; value++ {
				md, e := model.NewModel(m)
				equationMust(t, e)
				var out tactic.ModelRef
				equationMust(t, out.Set(&md.CoreField))
				for bit, helper := range helpers {
					n := fixDLNumber(t, m, true, fmt.Sprint((value>>bit)&1))
					equationMust(t, md.CoreField.RegisterConstant(helper, n))
				}
				equationMust(t, result.ModelConverter.ApplyModel(&out, 0))
				converted, e := out.Get()
				equationMust(t, e)
				n, e := converted.ConstantInterpretation(decl)
				equationMust(t, e)
				text, _, found, e := m.ArithmeticNumeralText(n)
				equationMust(t, e)
				if !found || text != fmt.Sprint(value) {
					t.Fatal("binary reconstruction", value, text)
				}
				for _, helper := range helpers {
					found, e := converted.HasInterpretation(helper)
					equationMust(t, e)
					if found {
						t.Fatal("bit helper escaped filter")
					}
				}
				equationMust(t, out.Close())
			}
		})
	}
}

func TestLIA2PBAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		success := false
		for stop := int64(0); stop < 1024; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				r := lia2pbFixture(t)
				m := r.manager
				if phase == "constructor" {
					m.Budget().FailAfter(stop)
					fresh, err := NewLIA2PBTactic(m, r.context, r.globals, nil, r.services)
					m.Budget().FailAfter(-1)
					if fresh != nil {
						equationMust(t, fresh.Close())
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
					return
				}
				x := fixDLConst(t, m, true, "x")
				goal := fixDLGoal(t, m, false, true, true, diffNeqTestBounds(t, m, x, "5")...)
				boolean, err := m.MkBoolSort()
				equationMust(t, err)
				for i, name := range []string{"failure-lower", "failure-upper"} {
					sym, e := m.Symbols.Intern(name)
					equationMust(t, e)
					label, e := m.MkConst(sym, boolean)
					equationMust(t, e)
					fixDLPin(t, m, label)
					dep, e := m.DependencyLeaf(label)
					equationMust(t, e)
					form, e := goal.Form(uint32(i))
					equationMust(t, e)
					equationMust(t, goal.Update(uint32(i), form, ast.ASTHandle{}, dep))
				}
				var result tactic.TacticResult
				t.Cleanup(func() { equationMust(t, result.Close()) })
				if phase == "cleanup" {
					equationMust(t, r.Apply(goal, &result))
				}
				m.Budget().FailAfter(stop)
				err = nil
				if phase == "cleanup" {
					err = r.Cleanup()
				} else {
					err = r.Apply(goal, &result)
				}
				m.Budget().FailAfter(-1)
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if success {
				break
			}
		}
		if !success {
			t.Fatal("no successful reservation frontier", phase)
		}
	}
}

func TestLIA2PBPolicyUpdatesCleanupAndTranslation(t *testing.T) {
	r := lia2pbFixture(t)
	m := r.manager
	p, err := r.context.NewValue()
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
	key := "lia2pb_max_bits"
	equationMust(t, r.context.SetUIntName(&p, &key, 1))
	equationMust(t, r.UpdateParams(&p))
	equationMust(t, r.Cleanup())
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, target.Close()) })
	translated, err := r.Translate(target)
	equationMust(t, err)
	equationMust(t, translated.IncRef())
	t.Cleanup(func() { equationMust(t, translated.DecRef()) })
	other := translated.(*LIA2PBTactic)
	if r.maxBits != 1 || other.maxBits != 1 {
		t.Fatal("updated parameter snapshot lost across cleanup/translation")
	}
	x := fixDLConst(t, m, true, "x")
	g := fixDLGoal(t, m, false, false, false, diffNeqTestBounds(t, m, x, "3")...)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(g, &result))
	size, err := g.Size()
	equationMust(t, err)
	depth, err := g.Depth()
	equationMust(t, err)
	if size != 2 || depth != 1 || result.ModelConverter != nil {
		t.Fatal("no-target source path", size, depth)
	}
	equationMust(t, r.context.SetUIntName(&p, &key, 2))
	equationMust(t, r.UpdateParams(&p))
	equationMust(t, r.Cleanup())
	equationMust(t, r.Apply(g, &result))
	size, err = g.Size()
	equationMust(t, err)
	if size != 6 || other.maxBits != 1 {
		t.Fatal("updated policy conversion or translated snapshot", size, other.maxBits)
	}
	for _, inconsistent := range []bool{false, true} {
		empty := fixDLGoal(t, m, false, true, false)
		if inconsistent {
			f, e := m.MkFalse()
			equationMust(t, e)
			equationMust(t, empty.Assert(f))
		}
		equationMust(t, r.Apply(empty, &result))
		depth, e := empty.Depth()
		equationMust(t, e)
		want := uint32(1)
		if inconsistent {
			want = 0
		}
		if depth != want || result.ModelConverter != nil {
			t.Fatal("empty/inconsistent depth and model", depth, inconsistent)
		}
	}
}

func TestLIA2PBFragmentAndProofGuards(t *testing.T) {
	for _, partial := range []bool{false, true} {
		for _, nonlinear := range []bool{false, true} {
			t.Run(fmt.Sprint(partial, "/", nonlinear), func(t *testing.T) {
				r := lia2pbFixture(t)
				m := r.manager
				p, err := r.context.NewValue()
				equationMust(t, err)
				t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
				key := "lia2pb_partial"
				equationMust(t, r.context.SetBoolName(&p, &key, partial))
				equationMust(t, r.UpdateParams(&p))
				x, y := fixDLConst(t, m, true, "x"), fixDLConst(t, m, true, "y")
				op := int32(6)
				if nonlinear {
					op = 9
				}
				form := fixDLApp(t, m, 2, fixDLApp(t, m, op, x, y), fixDLNumber(t, m, true, "3"))
				forms := append(diffNeqTestBounds(t, m, x, "3"), form)
				goal := fixDLGoal(t, m, false, false, false, forms...)
				var result tactic.TacticResult
				t.Cleanup(func() { equationMust(t, result.Close()) })
				err = r.Apply(goal, &result)
				if partial && !nonlinear {
					equationMust(t, err)
				} else if err == nil || err.Error() != "goal is in a fragment unsupported by lia2pb" {
					t.Fatal("fragment refusal", err)
				}
			})
		}
	}
	r := lia2pbFixture(t)
	equationMust(t, r.manager.ToggleProofMode(ast.ASTProofEnabled))
	g := fixDLGoal(t, r.manager, true, false, false)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	if err := r.Apply(g, &result); err == nil {
		t.Fatal("proof generation guard ignored")
	}
}

func TestLIA2PBExactTraceAndOwnerGuards(t *testing.T) {
	r := lia2pbFixture(t)
	m := r.manager
	x := fixDLConst(t, m, true, "x")
	goal := fixDLGoal(t, m, false, false, false, diffNeqTestBounds(t, m, x, "3")...)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("lia2pb")
	equationMust(t, r.BindTrace(tags))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	if tags.ShouldTrace("lia2pb") {
		var bounds, after bytes.Buffer
		equationMust(t, r.bounds.Display(&bounds))
		equationMust(t, goal.Display(&after))
		record := func(line int, body string) string {
			return fmt.Sprintf("-------- [lia2pb] operator() src/tactic/arith/lia2pb_tactic.cpp:%d ---------\n%s------------------------------------------------\n", line, body)
		}
		want := record(211, bounds.String()) + record(273, "x -> 0\n") + record(303, after.String())
		if output.String() != want {
			t.Fatal("source trace", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased trace emitted output")
	}
	refused := diagnostic.NewTraceTagsWithWriter(fixDLRefusedWriter{})
	refused.EnableAll(true)
	equationMust(t, r.BindTrace(refused))
	equationMust(t, r.Cleanup())
	equationMust(t, r.Apply(goal, &result))
	copy := *r
	if err := copy.Cleanup(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner", err)
	}
	equationMust(t, r.Close())
	if err := r.IncRef(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
}
