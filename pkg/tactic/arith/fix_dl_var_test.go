package arith

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/core"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func fixDLFixture(t *testing.T, proofs bool) (*FixDLVarTactic, *ast.ASTManager) {
	t.Helper()
	m, sy, q, _ := polynomialASTManager(t)
	if proofs {
		equationMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	}
	c, err := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: m.Budget()})
	equationMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	equationMust(t, err)
	g, err := uparams.NewGlobalParamsService(sy, c, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	equationMust(t, err)
	equationMust(t, g.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(g)
	equationMust(t, err)
	cooperation, err := cooperate.NewCooperation(m.Budget())
	equationMust(t, err)
	t.Cleanup(func() {
		equationMust(t, cooperation.Close())
		_, err := uparams.BindProcessGlobalParamsService(previous)
		equationMust(t, err)
		equationMust(t, g.Close())
		equationMust(t, catalog.Close())
		equationMust(t, c.Close())
	})
	r, err := NewFixDLVarTactic(m, c, g, nil, model.ModelEvaluatorConfig{Context: c, Globals: g, Memory: q.RuntimeOwner(), Cooperation: cooperation})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r, m
}
func fixDLPin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle) {
	t.Helper()
	ref, err := m.NewRef(h)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, m.DelRef(&ref)) })
}
func fixDLConst(t *testing.T, m *ast.ASTManager, integer bool, name string) ast.ASTHandle {
	t.Helper()
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	kind := int32(0)
	if integer {
		kind = 1
	}
	sort, err := m.MkFamilySort(family, kind, nil)
	equationMust(t, err)
	symbol, err := m.Symbols.Intern(name)
	equationMust(t, err)
	h, err := m.MkConst(symbol, sort)
	equationMust(t, err)
	fixDLPin(t, m, h)
	return h
}
func fixDLApp(t *testing.T, m *ast.ASTManager, op int32, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	h, err := m.MkFamilyApp(family, op, nil, args)
	equationMust(t, err)
	fixDLPin(t, m, h)
	return h
}
func fixDLNumber(t *testing.T, m *ast.ASTManager, integer bool, text string) ast.ASTHandle {
	t.Helper()
	var h ast.ASTHandle
	var err error
	if integer {
		h, err = m.ArithmeticIntegerNumeralText(text)
	} else {
		h, err = m.ArithmeticRealNumeralText(text)
	}
	equationMust(t, err)
	fixDLPin(t, m, h)
	return h
}
func fixDLBound(t *testing.T, m *ast.ASTManager, x, y ast.ASTHandle, integer bool, n string) ast.ASTHandle {
	t.Helper()
	minus := fixDLNumber(t, m, integer, "-1")
	neg := fixDLApp(t, m, 9, minus, y)
	sum := fixDLApp(t, m, 6, x, neg)
	return fixDLApp(t, m, 2, sum, fixDLNumber(t, m, integer, n))
}
func fixDLGoal(t *testing.T, m *ast.ASTManager, proofs, models, cores bool, forms ...ast.ASTHandle) *tactic.Goal {
	t.Helper()
	goal, err := tactic.NewGoal(m, proofs, models, cores)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, goal.Close()) })
	for _, form := range forms {
		if proofs {
			pr, err := m.MkProofAsserted(form)
			equationMust(t, err)
			equationMust(t, goal.AssertWithProof(form, pr, ast.ASTDependency{}))
		} else {
			equationMust(t, goal.Assert(form))
		}
	}
	return goal
}

func TestFixDLVarRewritesAndReconstructsIntegerAndRealModels(t *testing.T) {
	for _, integer := range []bool{false, true} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprintf("int%v/proofs%v", integer, proofs), func(t *testing.T) {
				r, m := fixDLFixture(t, proofs)
				x := fixDLConst(t, m, integer, "x")
				y := fixDLConst(t, m, integer, "y")
				z := fixDLConst(t, m, integer, "z")
				goal := fixDLGoal(t, m, proofs, true, true, fixDLBound(t, m, x, y, integer, "2"), fixDLBound(t, m, x, z, integer, "3"))
				var labels []ast.ASTHandle
				for i := uint32(0); i < 2; i++ {
					boolean, e := m.MkBoolSort()
					equationMust(t, e)
					name, e := m.Symbols.Intern(fmt.Sprintf("premise%d", i))
					equationMust(t, e)
					label, e := m.MkConst(name, boolean)
					equationMust(t, e)
					fixDLPin(t, m, label)
					labels = append(labels, label)
					dep, e := m.DependencyLeaf(label)
					equationMust(t, e)
					form, e := goal.Form(i)
					equationMust(t, e)
					pr, e := goal.Proof(i)
					equationMust(t, e)
					equationMust(t, goal.Update(i, form, pr, dep))
				}
				selected, err := r.selectDLVariable(goal)
				equationMust(t, err)
				if selected != x {
					t.Fatal("unique most frequent unit variable", selected, x)
				}
				var result tactic.TacticResult
				t.Cleanup(func() { equationMust(t, result.Close()) })
				equationMust(t, r.Apply(goal, &result))
				depth, err := goal.Depth()
				equationMust(t, err)
				if depth != 1 || result.ModelConverter == nil {
					t.Fatal("missing transformation/model converter", depth)
				}
				size, err := goal.Size()
				equationMust(t, err)
				for i := uint32(0); i < size; i++ {
					form, err := goal.Form(i)
					equationMust(t, err)
					occurs, err := m.OccursExpr(x, form)
					equationMust(t, err)
					if occurs {
						t.Fatal("fixed variable remains")
					}
					dep, err := goal.Dependency(i)
					equationMust(t, err)
					leaves, err := m.DependencyLinearize(dep, nil)
					equationMust(t, err)
					if len(leaves) != 1 || leaves[0] != labels[i] {
						t.Fatal("core dependency changed", leaves)
					}
					if proofs {
						pr, err := goal.Proof(i)
						equationMust(t, err)
						fact, err := m.GetProofFact(pr)
						equationMust(t, err)
						if fact != form {
							t.Fatal("rewritten proof fact")
						}
					}
				}
				equationMust(t, r.Cleanup())
				equationMust(t, r.Close())
				var md tactic.ModelRef
				t.Cleanup(func() { equationMust(t, md.Close()) })
				equationMust(t, tactic.ModelConverterToModel(m, result.ModelConverter, &md))
				model, err := md.Get()
				equationMust(t, err)
				decl, err := m.GetDecl(x)
				equationMust(t, err)
				value, err := model.ConstantInterpretation(decl)
				equationMust(t, err)
				text, isInt, matched, err := m.ArithmeticNumeralText(value)
				equationMust(t, err)
				if !matched || text != "0" || isInt != integer {
					t.Fatal("fixed-zero reconstruction", text, isInt, integer)
				}
			})
		}
	}
}

func TestFixDLVarCPPApply(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the native executable")
	}
	r, m := fixDLFixture(t, false)
	x := fixDLConst(t, m, true, "x")
	y := fixDLConst(t, m, true, "y")
	z := fixDLConst(t, m, true, "z")
	goal := fixDLGoal(t, m, false, false, false, fixDLBound(t, m, x, y, true, "2"), fixDLBound(t, m, x, z, true, "3"))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	simplifier, err := core.NewSimplifyTactic(m, r.context, r.globals, nil)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, simplifier.Close()) })
	equationMust(t, simplifier.Apply(goal, &result))
	equationMust(t, r.Apply(goal, &result))
	var out bytes.Buffer
	out.WriteString("(goals (goal ")
	size, err := goal.Size()
	equationMust(t, err)
	for i := uint32(0); i < size; i++ {
		form, e := goal.Form(i)
		equationMust(t, e)
		equationMust(t, r.writeDLExpr(&out, form))
		out.WriteByte(' ')
	}
	depth, err := goal.Depth()
	equationMust(t, err)
	fmt.Fprintf(&out, ":precision precise :depth %d))", depth)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = strings.NewReader("(set-logic QF_LIA)(declare-const x Int)(declare-const y Int)(declare-const z Int)(assert (<= (+ x (* (- 1) y)) 2))(assert (<= (+ x (* (- 1) z)) 3))(apply (then simplify fix-dl-var))")
	native, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(native))
	}
	tokens := func(s string) string {
		return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
	}
	if tokens(out.String()) != tokens(string(native)) {
		t.Fatalf("native apply differs\nGo: %s\nC++: %s", out.String(), native)
	}
}

func TestFixDLVarTranslationPolicyAndCleanup(t *testing.T) {
	r, m := fixDLFixture(t, false)
	p, err := r.context.NewValue()
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
	key := "max_steps"
	equationMust(t, r.context.SetUIntName(&p, &key, 0))
	equationMust(t, r.UpdateParams(&p))
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, target.Close()) })
	translated, err := r.Translate(target)
	equationMust(t, err)
	equationMust(t, translated.IncRef())
	t.Cleanup(func() { equationMust(t, translated.DecRef()) })
	equationMust(t, r.context.SetUIntName(&p, &key, ^uint32(0)))
	equationMust(t, r.UpdateParams(&p))
	equationMust(t, r.Close())
	x := fixDLConst(t, target, true, "x")
	y := fixDLConst(t, target, true, "y")
	z := fixDLConst(t, target, true, "z")
	goal := fixDLGoal(t, target, false, false, false, fixDLBound(t, target, x, y, true, "2"), fixDLBound(t, target, x, z, true, "3"))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	err = translated.Apply(goal, &result)
	var exception tactic.TacticException
	if !errors.As(err, &exception) || exception.Error() != tactic.TacticMaxStepsMessage {
		t.Fatal("translated rewrite policy", err)
	}
	equationMust(t, translated.Cleanup())
	equationMust(t, translated.UpdateParams(&p))
	equationMust(t, translated.Cleanup())
	equationMust(t, translated.Apply(goal, &result))
	depth, err := goal.Depth()
	equationMust(t, err)
	if depth != 1 {
		t.Fatal("cleanup did not recover interrupted rewriting", depth)
	}
}

func TestFixDLVarNestedPenaltyBoundaryAndSharedAtoms(t *testing.T) {
	for _, count := range []int{20, 21} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			r, m := fixDLFixture(t, false)
			x := fixDLConst(t, m, true, "x")
			y := fixDLConst(t, m, true, "y")
			v := fixDLConst(t, m, true, "v")
			z := fixDLConst(t, m, true, "z")
			goal := fixDLGoal(t, m, false, false, false, fixDLBound(t, m, x, y, true, "1"), fixDLBound(t, m, x, v, true, "2"))
			var disj []ast.ASTHandle
			for i := 0; i < count; i++ {
				w := fixDLConst(t, m, true, fmt.Sprintf("w%d", i))
				disj = append(disj, fixDLBound(t, m, z, w, true, "1"))
			}
			or, err := m.MkOrArgs(disj)
			equationMust(t, err)
			equationMust(t, goal.Assert(or))
			selected, err := r.selectDLVariable(goal)
			equationMust(t, err)
			want := x
			if count == 21 {
				want = z
			}
			if selected != want {
				t.Fatal("strict nested penalty", selected, want)
			}
			// Repeating an already marked arithmetic atom in the same Boolean
			// DAG contributes one nested visit, not repeated visits to its terms.
			shared, err := m.MkOrArgs([]ast.ASTHandle{disj[0], disj[0]})
			equationMust(t, err)
			equationMust(t, goal.Assert(shared))
			selected, err = r.selectDLVariable(goal)
			equationMust(t, err)
			if selected != want {
				t.Fatal("shared atom changed counts", selected, want)
			}
		})
	}
}

func TestFixDLVarRefusalsAndEmptyGoalDoNotChangeDepth(t *testing.T) {
	r, m := fixDLFixture(t, false)
	x := fixDLConst(t, m, true, "x")
	y := fixDLConst(t, m, true, "y")
	two := fixDLNumber(t, m, true, "2")
	for _, form := range []ast.ASTHandle{{}, fixDLApp(t, m, 2, x, two), fixDLApp(t, m, 2, fixDLApp(t, m, 7, x, y), two), fixDLApp(t, m, 2, fixDLApp(t, m, 9, x, y), two)} {
		var forms []ast.ASTHandle
		if !form.IsNull() {
			forms = append(forms, form)
		}
		goal := fixDLGoal(t, m, false, true, false, forms...)
		var result tactic.TacticResult
		equationMust(t, r.Apply(goal, &result))
		depth, err := goal.Depth()
		equationMust(t, err)
		if depth != 0 || result.ModelConverter != nil {
			t.Fatal("refused goal transformed", depth)
		}
		equationMust(t, result.Close())
	}
}

type fixDLRefusedWriter struct{}

func (fixDLRefusedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestFixDLVarChoiceTraceAndStreamRefusal(t *testing.T) {
	r, m := fixDLFixture(t, false)
	x := fixDLConst(t, m, true, "x")
	y := fixDLConst(t, m, true, "y")
	z := fixDLConst(t, m, true, "z")
	goal := fixDLGoal(t, m, false, false, false, fixDLBound(t, m, x, y, true, "2"), fixDLBound(t, m, x, z, true, "3"))
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("fix_dl_var_choice")
	equationMust(t, r.BindTrace(tags))
	_, err := r.selectDLVariable(goal)
	equationMust(t, err)
	if tags.ShouldTrace("fix_dl_var_choice") {
		want := "-------- [fix_dl_var_choice] most_occs src/tactic/arith/fix_dl_var_tactic.cpp:204 ---------\nr1 occs: 2\nx\nr2 occs: 2\nx\n------------------------------------------------\n"
		if out.String() != want {
			t.Fatal("choice trace", out.String(), want)
		}
	} else if out.Len() != 0 {
		t.Fatal("erased trace wrote output")
	}
	refused := diagnostic.NewTraceTagsWithWriter(fixDLRefusedWriter{})
	refused.EnableAll(true)
	equationMust(t, r.BindTrace(refused))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(goal, &result))
	copy := *r
	if err = copy.Cleanup(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner", err)
	}
	equationMust(t, r.Close())
	if err = r.IncRef(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
}

func TestFixDLVarFailureShapeTrace(t *testing.T) {
	r, m := fixDLFixture(t, false)
	x := fixDLConst(t, m, true, "x")
	y := fixDLConst(t, m, true, "y")
	form := fixDLApp(t, m, 2, fixDLApp(t, m, 6, x, y), fixDLNumber(t, m, true, "2"))
	goal := fixDLGoal(t, m, false, false, false, form)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("fix_dl_var")
	equationMust(t, r.BindTrace(tags))
	selected, err := r.selectDLVariable(goal)
	equationMust(t, err)
	if !selected.IsNull() {
		t.Fatal("accepted sum instead of difference")
	}
	if tags.ShouldTrace("fix_dl_var") {
		record := func(function string, line int, body string) string {
			return fmt.Sprintf("-------- [fix_dl_var] %s src/tactic/arith/fix_dl_var_tactic.cpp:%d ---------\n%s------------------------------------------------\n", function, line, body)
		}
		want := record("process_arith_atom", 110, fmt.Sprintf("is_times_minus_one: 0\nis_uninterp(t):     1\nt.family_id():      -1\nutil.family_id:     %d\n", r.family)) + record("throw_failed", 48, "(+ x y)\n2\n")
		if out.String() != want {
			t.Fatal("failure trace", out.String(), want)
		}
	} else if out.Len() != 0 {
		t.Fatal("erased failure trace wrote output")
	}
}

func TestFixDLVarAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		for _, proofs := range []bool{false, true} {
			succeeded := false
			for stop := int64(0); stop < 512; stop++ {
				t.Run(fmt.Sprintf("%s/proofs%v/%d", phase, proofs, stop), func(t *testing.T) {
					r, m := fixDLFixture(t, proofs)
					if phase == "constructor" {
						m.Budget().FailAfter(stop)
						fresh, err := NewFixDLVarTactic(m, r.context, r.globals, nil, r.services)
						m.Budget().FailAfter(-1)
						if err == nil {
							succeeded = true
						} else if !errors.Is(err, arena.ErrInjectedFailure) {
							t.Fatal(err)
						}
						if fresh != nil {
							equationMust(t, fresh.Close())
						}
					} else {
						x := fixDLConst(t, m, true, "x")
						y := fixDLConst(t, m, true, "y")
						z := fixDLConst(t, m, true, "z")
						goal := fixDLGoal(t, m, proofs, true, true, fixDLBound(t, m, x, y, true, "2"), fixDLBound(t, m, x, z, true, "3"))
						var result tactic.TacticResult
						t.Cleanup(func() { equationMust(t, result.Close()) })
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
							equationMust(t, r.Apply(goal, &result))
						}
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
