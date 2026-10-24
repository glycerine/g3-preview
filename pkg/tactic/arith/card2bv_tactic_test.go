package arith

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func cardFixture(t *testing.T) (*Card2BVTactic, model.ModelEvaluatorConfig) {
	t.Helper()
	base, m := fixDLFixture(t, false)
	equationMust(t, m.RegisterDeclarationPlugins())
	r, err := NewCard2BVTactic(m, base.context, base.globals, nil)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r, base.services
}
func cardGoal(t *testing.T, r *Card2BVTactic) (*tactic.Goal, ast.ASTHandle) {
	t.Helper()
	m := r.manager
	bs, err := m.MkBoolSort()
	equationMust(t, err)
	var args []ast.ASTHandle
	for i := 0; i < 7; i++ {
		name, err := m.Symbols.Intern(fmt.Sprint("card-p", i))
		equationMust(t, err)
		x, err := m.MkConst(name, bs)
		equationMust(t, err)
		fixDLPin(t, m, x)
		args = append(args, x)
	}
	f, err := m.MkPseudoBooleanAtMost(args, 2)
	equationMust(t, err)
	fixDLPin(t, m, f)
	g, err := tactic.NewGoal(m, false, true, true)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, g.Close()) })
	dep, err := m.DependencyLeaf(f)
	equationMust(t, err)
	equationMust(t, g.AssertWithProof(f, ast.ASTHandle{}, dep))
	return g, f
}

func TestCard2BVSideConstraintsCoreAndIndependentModel(t *testing.T) {
	r, services := cardFixture(t)
	g, original := cardGoal(t, r)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(g, &result))
	if len(r.pending) != 0 || result.ModelConverter == nil || len(result.Goals) != 1 || result.Goals[0] != g {
		t.Fatal("lexical rewriter lifetime or filter publication")
	}
	dep, err := g.Dependency(0)
	equationMust(t, err)
	leaves, err := r.manager.DependencyLinearize(dep, nil)
	equationMust(t, err)
	if len(leaves) != 1 || leaves[0] != original {
		t.Fatal("lost original dependency", leaves)
	}
	s, err := smtcore.NewSMTSolver(r.manager, nil, "QF_BV", smtcore.SMTSolverConfig{Params: r.context, Globals: r.globals, ModelEvaluator: services})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, s.Close()) })
	equationMust(t, s.SetProduceModels(true))
	n, err := g.Size()
	equationMust(t, err)
	if n < 2 {
		t.Fatal("sorting network produced no side constraints", n)
	}
	for i := uint32(0); i < n; i++ {
		f, err := g.Form(i)
		equationMust(t, err)
		equationMust(t, s.Assert(f))
	}
	status, err := s.CheckSat(nil)
	equationMust(t, err)
	if status != lbool.True {
		t.Fatal("encoded cardinality was not satisfiable", status)
	}
	md, err := s.AcquireModel()
	equationMust(t, err)
	var out tactic.ModelRef
	equationMust(t, out.Set(md))
	equationMust(t, md.DecRef())
	t.Cleanup(func() { equationMust(t, out.Close()) })
	equationMust(t, r.Close())
	equationMust(t, s.Close())
	equationMust(t, result.ModelConverter.ApplyModel(&out, 0))
	equationMust(t, result.Close())
	concrete, err := out.Get()
	equationMust(t, err)
	n, err = concrete.NumConstants()
	equationMust(t, err)
	for i := uint32(0); i < n; i++ {
		d, err := concrete.Constant(i)
		equationMust(t, err)
		name, err := r.manager.GetASTName(d)
		equationMust(t, err)
		s, err := r.manager.Symbols.String(name)
		equationMust(t, err)
		if !strings.HasPrefix(s, "card-p") {
			t.Fatal("auxiliary constant escaped model filtering", s)
		}
	}
	ev, err := model.NewModelEvaluator(concrete, services)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, ev.Close()) })
	equationMust(t, ev.SetModelCompletion(true))
	value, err := r.manager.NewRef(ast.ASTHandle{})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.manager.DelRef(&value)) })
	equationMust(t, ev.Evaluate(original, value))
	v, err := r.manager.RefGet(value)
	equationMust(t, err)
	if !r.manager.IsTrue(v) {
		t.Fatal("filtered model violates original cardinality")
	}
}

func TestCard2BVParametersTranslationInconsistencyAndTrace(t *testing.T) {
	r, _ := cardFixture(t)
	g, _ := cardGoal(t, r)
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("card2bv-before")
	tags.Enable("card2bv")
	equationMust(t, r.BindTrace(tags))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(g, &result))
	if tags.ShouldTrace("card2bv") {
		for _, text := range []string{"Rewriting ", "card2bv_tactic.cpp:59", "card2bv_tactic.cpp:75", "card2bv_tactic.cpp:100"} {
			if !strings.Contains(trace.String(), text) {
				t.Fatal("missing trace", text, trace.String())
			}
		}
	}
	bad, err := tactic.NewGoal(r.manager, false, true, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, bad.Close()) })
	falsity, err := r.manager.MkFalse()
	equationMust(t, err)
	equationMust(t, bad.Assert(falsity))
	equationMust(t, bad.SetDepth(11))
	equationMust(t, r.Apply(bad, &result))
	if result.ModelConverter != nil || len(result.Goals) != 1 || result.Goals[0] != bad {
		t.Fatal("inconsistent path failed to clear previous metadata")
	}
	var displayed bytes.Buffer
	equationMust(t, bad.DisplayWithAttributes(&displayed))
	if !strings.Contains(displayed.String(), ":depth 11") {
		t.Fatal("inconsistent path incremented depth", displayed.String())
	}
	p, err := r.context.NewValue()
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.context.Del(&p)) })
	key := "max_steps"
	equationMust(t, r.context.SetUIntName(&p, &key, 0))
	equationMust(t, r.UpdateParams(&p))
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: r.manager.Symbols, Numerals: r.manager.NumeralService(), Budget: r.manager.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, target.Close()) })
	equationMust(t, target.RegisterDeclarationPlugins())
	copy, err := r.Translate(target)
	equationMust(t, err)
	translated := copy.(*Card2BVTactic)
	t.Cleanup(func() { equationMust(t, translated.Close()) })
	equationMust(t, r.UpdateParams(nil))
	tr, err := ast.NewASTTranslation(r.manager, target, true)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, tr.Close()) })
	fresh, _ := cardGoal(t, r)
	translatedGoal, err := fresh.Translate(tr)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, translatedGoal.Close()) })
	var translatedResult tactic.TacticResult
	t.Cleanup(func() { equationMust(t, translatedResult.Close()) })
	err = translated.Apply(translatedGoal, &translatedResult)
	if !rewriter.IsRewriterException(err) || !strings.Contains(err.Error(), "max. steps") {
		t.Fatal("translated parameter snapshot", err)
	}
	descr, err := uparams.NewParamDescrs(r.manager.Symbols, uparams.ParamDescrsConfig{Budget: r.manager.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, descr.Close()) })
	equationMust(t, r.CollectParamDescrs(descr))
	size, err := descr.Size()
	equationMust(t, err)
	if size != 0 {
		t.Fatal("source has no tactic parameter descriptors", size)
	}
}

func TestCard2BVAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 2048 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					r, _ := cardFixture(t)
					g, _ := cardGoal(t, r)
					var result tactic.TacticResult
					t.Cleanup(func() { equationMust(t, result.Close()) })
					r.manager.Budget().FailAfter(stop)
					var err error
					if phase == "constructor" {
						var fresh *Card2BVTactic
						fresh, err = NewCard2BVTactic(r.manager, r.context, r.globals, &r.params)
						r.manager.Budget().FailAfter(-1)
						if fresh != nil {
							equationMust(t, fresh.Close())
						}
						var failure memory.ConstructionFailure
						if errors.As(err, &failure) {
							equationMust(t, failure.Cleanup())
						}
					} else {
						err = r.Apply(g, &result)
					}
					r.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
					equationMust(t, r.Cleanup())
				})
				if t.Failed() {
					t.Fatal("reservation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful reservation frontier", phase)
			}
		})
	}
}

func TestCard2BVCancellationAndClosedTraceRecovery(t *testing.T) {
	r, _ := cardFixture(t)
	g, _ := cardGoal(t, r)
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	r.manager.ResourceLimit().IncCancel()
	err := r.Apply(g, &result)
	r.manager.ResourceLimit().DecCancel()
	if !rewriter.IsRewriterException(err) || !strings.Contains(err.Error(), "canceled") {
		t.Fatal("theory rewriter cancellation", err)
	}
	equationMust(t, r.Cleanup())
	closed := diagnostic.NewTraceTagsWithWriter(addBoundsClosedWriter{})
	closed.Enable("card2bv-before")
	closed.Enable("card2bv")
	equationMust(t, r.BindTrace(closed))
	equationMust(t, r.Apply(g, &result))
	if result.ModelConverter == nil {
		t.Fatal("closed diagnostic stream changed encoding")
	}
	bad, err := tactic.NewGoal(r.manager, false, true, false)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, bad.Close()) })
	f, err := r.manager.MkFalse()
	equationMust(t, err)
	equationMust(t, bad.Assert(f))
	r.manager.ResourceLimit().IncCancel()
	err = r.Apply(bad, &result)
	r.manager.ResourceLimit().DecCancel()
	equationMust(t, err) // Inconsistent path never reaches a rewrite checkpoint.
}
