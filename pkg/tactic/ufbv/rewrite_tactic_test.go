package ufbv

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func ufbvTacticServicesFixture(t *testing.T, m *ast.ASTManager) (*uparams.ParamsContext, *uparams.GlobalParamsService) {
	t.Helper()
	budget := arena.NewBudget(1 << 30)
	numbers, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
	cacheMust(t, e)
	ctx, e := uparams.NewParamsContext(m.Symbols, numbers, uparams.ParamsConfig{Budget: budget, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	cacheMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: budget, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	cacheMust(t, e)
	globals, e := uparams.NewGlobalParamsService(m.Symbols, ctx, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	cacheMust(t, e)
	cacheMust(t, globals.InitializeSource())
	t.Cleanup(func() {
		cacheMust(t, globals.Close())
		cacheMust(t, catalog.Close())
		cacheMust(t, ctx.Close())
		cacheMust(t, numbers.Close())
		if budget.Used() != 0 {
			t.Fatal("tactic service roots", budget.Used())
		}
	})
	return ctx, globals
}

func TestUFBVRewriterTacticPublishesIndependentGoalReference(t *testing.T) {
	m := cacheFixture(t)
	ctx, globals := ufbvTacticServicesFixture(t, m)
	rewriter, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, rewriter.Close()) })
	goal, e := tactic.NewGoal(m, false, false, false)
	cacheMust(t, e)
	cacheMust(t, goal.IncRef()) // The incoming goal_ref.
	result, e := rewriter.Apply(goal)
	cacheMust(t, e)
	if result != goal || goal.RefCount != 2 {
		t.Fatal("result buffer must own a second reference", result, goal.RefCount)
	}
	cacheMust(t, goal.DecRef())
	if goal.Closed || goal.RefCount != 1 {
		t.Fatal("result must survive incoming reference release", goal.Closed, goal.RefCount)
	}
	cacheMust(t, result.DecRef())
	if !goal.Closed {
		t.Fatal("result release must retire goal")
	}
}

func TestUFBVRewriterTacticGoalTransitionAndCoreRefusal(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	pin := func(h ast.ASTHandle) ast.ASTHandle {
		t.Helper()
		cacheMust(t, m.IncRef(h))
		t.Cleanup(func() { cacheMust(t, m.DecRef(h)) })
		return h
	}
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	pin(x)
	f, e := m.MkFuncDecl(symbol.Numerical(401), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	pin(f)
	aDecl, e := m.MkFuncDecl(symbol.Numerical(402), nil, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	pin(aDecl)
	a, e := m.MkApp(aDecl, nil)
	cacheMust(t, e)
	pin(a)
	fx, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	pin(fx)
	fa, e := m.MkApp(f, []ast.ASTHandle{a})
	cacheMust(t, e)
	pin(fa)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []symbol.Symbol{symbol.Numerical(403)}, Body: fx})
	cacheMust(t, e)
	pin(q)
	ctx, globals := ufbvTacticServicesFixture(t, m)
	rewriter, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, rewriter.Close()) })
	goal, e := tactic.NewGoal(m, false, true, false)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(q))
	cacheMust(t, goal.Assert(fa))
	cacheMust(t, goal.SetDepth(7))
	reportBudget := arena.NewBudget(1 << 20)
	reportStream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: reportBudget})
	cacheMust(t, e)
	oldLevel, e := diagnostic.ProcessVerbosityLevel()
	cacheMust(t, e)
	oldStream, e := diagnostic.BorrowProcessVerboseStream()
	cacheMust(t, e)
	cacheMust(t, diagnostic.SetProcessVerboseStream(reportStream))
	cacheMust(t, diagnostic.SetProcessVerbosityLevel(10))
	t.Cleanup(func() {
		cacheMust(t, diagnostic.SetProcessVerboseStream(oldStream))
		cacheMust(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
		cacheMust(t, reportStream.Close())
		if reportBudget.Used() != 0 {
			t.Fatal("tactic report stream roots", reportBudget.Used())
		}
	})
	got, e := rewriter.Apply(goal)
	cacheMust(t, e)
	reportText, e := reportStream.String()
	cacheMust(t, e)
	if !strings.HasPrefix(reportText, "(ufbv-rewriter :num-exprs ") || !strings.Contains(reportText, " :before-memory ") || !strings.Contains(reportText, " :after-memory ") {
		t.Fatal("live UFBV tactic omitted source report", reportText)
	}
	if got != goal {
		t.Fatal("source tactic must publish the input goal")
	}
	n, e := goal.Size()
	cacheMust(t, e)
	form, e := goal.Form(0)
	cacheMust(t, e)
	depth, e := goal.Depth()
	cacheMust(t, e)
	if n != 1 || form != q || depth != 8 {
		t.Fatal("rule reschedule, true elimination and depth", n, form, depth)
	}
	coreGoal, e := tactic.NewGoal(m, false, true, true)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, coreGoal.Close()) })
	cacheMust(t, coreGoal.Assert(fa))
	if _, e := rewriter.Apply(coreGoal); !errors.Is(e, ErrRewriterUnsatCores) {
		t.Fatal("source unsat-core guard", e)
	}
	n, e = coreGoal.Size()
	cacheMust(t, e)
	if n != 1 {
		t.Fatal("core refusal changed input goal", n)
	}
	cacheMust(t, rewriter.Close())
	if _, e := rewriter.Apply(goal); e == nil {
		t.Fatal("closed tactic accepted goal")
	}
}

func TestUFBVRewriterTacticProofModePreservesFormulaAndProof(t *testing.T) {
	warnings := diagnostic.BorrowProcessWarningService()
	oldEnabled := warnings.Enabled()
	var warningOutput bytes.Buffer
	cacheMust(t, warnings.SetWarningStream(&warningOutput))
	t.Cleanup(func() {
		cacheMust(t, warnings.SetWarningStream(nil))
		cacheMust(t, warnings.Enable(oldEnabled))
	})
	cacheMust(t, warnings.Enable(true))
	m := cacheFixtureProofMode(t, ast.ASTProofEnabled)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	p, e := m.MkConst(symbol.Numerical(501), b)
	cacheMust(t, e)
	cacheMust(t, m.IncRef(p))
	t.Cleanup(func() { cacheMust(t, m.DecRef(p)) })
	ctx, globals := ufbvTacticServicesFixture(t, m)
	rewriter, e := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, rewriter.Close()) })
	goal, e := tactic.NewGoal(m, true, true, false)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, goal.Close()) })
	cacheMust(t, goal.Assert(p))
	beforeForm, e := goal.Form(0)
	cacheMust(t, e)
	beforeRefs, e := m.GetRefCount(beforeForm)
	cacheMust(t, e)
	beforeProof, e := goal.Proof(0)
	cacheMust(t, e)
	if beforeProof.IsNull() {
		t.Fatal("proof-enabled source goal lacks an asserted proof")
	}
	beforeProofRefs, e := m.GetRefCount(beforeProof)
	cacheMust(t, e)
	got, e := rewriter.Apply(goal)
	cacheMust(t, e)
	if got != goal {
		t.Fatal("proof branch must publish the same goal")
	}
	n, e := goal.Size()
	cacheMust(t, e)
	afterForm, e := goal.Form(0)
	cacheMust(t, e)
	afterProof, e := goal.Proof(0)
	cacheMust(t, e)
	depth, e := goal.Depth()
	cacheMust(t, e)
	if n != 1 || afterForm != beforeForm || afterProof != beforeProof || depth != 1 {
		t.Fatal("source proof bypass and goal depth", n, afterForm, afterProof, depth)
	}
	afterRefs, e := m.GetRefCount(afterForm)
	cacheMust(t, e)
	if afterRefs != beforeRefs {
		t.Fatal("formula ref_vector pin survived tactic return", beforeRefs, afterRefs)
	}
	afterProofRefs, e := m.GetRefCount(afterProof)
	cacheMust(t, e)
	if afterProofRefs != beforeProofRefs {
		t.Fatal("proof ref_vector pin survived tactic return", beforeProofRefs, afterProofRefs)
	}
	if got, want := warningOutput.String(), "WARNING: PRE_DEMODULATOR=true is not supported when proofs are enabled.\n"; got != want {
		t.Fatal("source proof-mode warning", got, want)
	}
	envBudget := arena.NewBudget(1 << 20)
	global, e := uparams.NewGlobalParams(m.Symbols, ctx, uparams.GlobalParamsConfig{Budget: envBudget, Register: func(r uparams.GlobalParamsRegistrar) error {
		d, err := uparams.NewParamDescrs(m.Symbols, uparams.ParamDescrsConfig{Budget: envBudget, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: envBudget}}})
		if err != nil {
			return err
		}
		defer d.Close()
		description, defaultValue := "enable/disable warning messages", "true"
		if err := d.InsertName("warning", uparams.ParamBool, &description, &defaultValue, nil); err != nil {
			return err
		}
		return r.RegisterGlobal(d)
	}})
	cacheMust(t, e)
	t.Cleanup(func() {
		cacheMust(t, global.Close())
		if envBudget.Used() != 0 {
			t.Error("environment global roots", envBudget.Used())
		}
	})
	cacheMust(t, global.SetName("warning", "false"))
	cacheMust(t, uparams.UpdateProcessEnvParams(global))
	if warnings.Enabled() {
		t.Fatal("source environment update did not reach process warning state")
	}
	_, e = rewriter.Apply(goal)
	cacheMust(t, e)
	if got, want := warningOutput.String(), "WARNING: PRE_DEMODULATOR=true is not supported when proofs are enabled.\n"; got != want {
		t.Fatal("disabled process warning emitted", got)
	}
}
