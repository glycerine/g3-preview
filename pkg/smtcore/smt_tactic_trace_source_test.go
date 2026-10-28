package smtcore

// Source: src/smt/tactic/smt_tactic.cpp:53,78,128,157-165,175-176,231,298.

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type smtTacticRefusingTrace struct{}

func (smtTacticRefusingTrace) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func smtTacticTraceFixture(t *testing.T, writer io.Writer) (*ast.ASTManager, model.ModelEvaluatorConfig, *diagnostic.TraceTags) {
	t.Helper()
	owner, config := smtModelGeneratorFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	for _, tag := range []string{"smt_tactic", "smt_tactic_params", "smt_tactic_detail", "smt_tactic_memory", "mus"} {
		tags.Enable(tag)
	}
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{
		Symbols: owner.Symbols, Numerals: owner.NumeralService(), Budget: owner.Budget(), TraceTags: tags,
	})
	astTestMust(t, err)
	astTestMust(t, manager.RegisterDeclarationPlugins())
	t.Cleanup(func() { astTestMust(t, manager.Close()) })
	return manager, config, tags
}

func TestSMTTacticSourceTraceSitesAndOrdering(t *testing.T) {
	var output bytes.Buffer
	m, evaluator, tags := smtTacticTraceFixture(t, &output)
	settings, err := evaluator.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, evaluator.Context.Del(&settings)) })
	auto, relevancy := "auto_config", "relevancy"
	astTestMust(t, evaluator.Context.SetBoolName(&settings, &auto, false))
	astTestMust(t, evaluator.Context.SetUIntName(&settings, &relevancy, 1))
	created, err := NewSMTTactic(m, &settings, SMTSolverConfig{Params: evaluator.Context, Globals: evaluator.Globals, ModelEvaluator: evaluator})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, created.Close()) })
	astTestMust(t, created.UpdateParams(&settings))
	astTestMust(t, created.SetLogic(astTestSymbol(t, m, "QF_UF")))

	goal, err := tactic.NewGoal(m, false, false, true)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, goal.Close()) })
	falsehood, err := m.MkFalse()
	astTestMust(t, err)
	astTestMust(t, goal.Assert(falsehood))
	var result tactic.TacticResult
	t.Cleanup(func() { astTestMust(t, result.Close()) })
	astTestMust(t, created.Apply(goal, &result))

	under, err := tactic.NewGoal(m, false, false, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, under.Close()) })
	astTestMust(t, under.Assert(falsehood))
	astTestMust(t, under.SetPrecision(tactic.GoalUnder))
	var underResult tactic.TacticResult
	t.Cleanup(func() { astTestMust(t, underResult.Close()) })
	if err = created.Apply(under, &underResult); err == nil || !strings.Contains(err.Error(), "under-approximated goal") {
		t.Fatal("missing approximation rejection", err)
	}

	using, err := NewSMTTacticUsing(m, &settings, "QF_UF", false, SMTSolverConfig{Params: evaluator.Context, Globals: evaluator.Globals, ModelEvaluator: evaluator})
	astTestMust(t, err)
	astTestMust(t, using.IncRef())
	t.Cleanup(func() { astTestMust(t, using.DecRef()) })

	text := output.String()
	if !tags.ShouldTrace("smt_tactic") {
		if text != "" {
			t.Fatal("erased TRACE emitted output", text)
		}
		return
	}
	wants := []string{
		"[smt_tactic] smt_tactic src/smt/tactic/smt_tactic.cpp:53",
		"[smt_tactic] updt_params src/smt/tactic/smt_tactic.cpp:78",
		"[smt_tactic] operator() src/smt/tactic/smt_tactic.cpp:157",
		"[smt_tactic_params] operator() src/smt/tactic/smt_tactic.cpp:163",
		"[smt_tactic_detail] operator() src/smt/tactic/smt_tactic.cpp:164",
		"[smt_tactic_memory] operator() src/smt/tactic/smt_tactic.cpp:165",
		"[smt_tactic] scoped_init_ctx src/smt/tactic/smt_tactic.cpp:128",
		"[mus] operator() src/smt/tactic/smt_tactic.cpp:175",
		"[smt_tactic] operator() src/smt/tactic/smt_tactic.cpp:231",
		"[smt_tactic] mk_smt_tactic_using src/smt/tactic/smt_tactic.cpp:298",
	}
	position := -1
	for _, want := range wants {
		next := strings.Index(text[position+1:], want)
		if next < 0 {
			t.Fatal("missing or out-of-order SMT tactic trace", want, text)
		}
		position += next + 1
	}
	for _, want := range []string{
		"AUTO_CONFIG: 0 HIDIV0: 1  PREPROCESS: 1",
		"RELEVANCY: 1",
		"logic: QF_UF",
		"wasted_size: 0",
		"failed to show to be unsat...",
	} {
		if !strings.Contains(text, want) {
			t.Fatal("missing SMT tactic trace body", want, text)
		}
	}
}

func TestSMTTacticTraceBadStreamDoesNotChangeExecution(t *testing.T) {
	m, evaluator, tags := smtTacticTraceFixture(t, smtTacticRefusingTrace{})
	created, err := NewSMTTactic(m, nil, SMTSolverConfig{Params: evaluator.Context, Globals: evaluator.Globals, ModelEvaluator: evaluator})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, created.Close()) })
	goal, err := tactic.NewGoal(m, false, false, false)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, goal.Close()) })
	var result tactic.TacticResult
	t.Cleanup(func() { astTestMust(t, result.Close()) })
	astTestMust(t, created.Apply(goal, &result))
	if tags.ShouldTrace("smt_tactic") && !goal.IsDecidedSat() {
		t.Fatal("refused source trace changed tactic execution")
	}
}
