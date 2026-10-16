package ufbv

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// Source: src/tactic/ufbv/ufbv_rewriter_tactic.cpp:63-69. The result buffer
// retains the goal before TRACE displays its rewritten formulas.
func TestUFBVRewriterPublishedGoalSourceTraceAndFormatterRefusal(t *testing.T) {
	m := cacheFixture(t)
	b := m.Budget()
	numbers, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	cacheMust(t, err)
	ctx, err := uparams.NewParamsContext(m.Symbols, numbers, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	cacheMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	cacheMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, ctx, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	cacheMust(t, err)
	cacheMust(t, globals.InitializeSource())
	t.Cleanup(func() {
		cacheMust(t, globals.Close())
		cacheMust(t, catalog.Close())
		cacheMust(t, ctx.Close())
		cacheMust(t, numbers.Close())
	})
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	cacheMust(t, err)
	defer func() { _, e := uparams.BindProcessGlobalParamsService(previous); cacheMust(t, e) }()
	rewriter, err := NewRewriterTactic(m, ctx, globals)
	cacheMust(t, err)
	defer func() { cacheMust(t, rewriter.Close()) }()
	goal, err := tactic.NewGoal(m, false, false, false)
	cacheMust(t, err)
	cacheMust(t, goal.IncRef())
	defer func() { cacheMust(t, goal.DecRef()) }()
	bs, err := m.MkBoolSort()
	cacheMust(t, err)
	name, err := m.Symbols.Intern("trace_atom")
	cacheMust(t, err)
	atom, err := m.MkConst(name, bs)
	cacheMust(t, err)
	cacheMust(t, goal.Assert(atom))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	cacheMust(t, rewriter.BindTrace(tags))
	result, err := rewriter.Apply(goal)
	cacheMust(t, err)
	cacheMust(t, result.DecRef())
	if output.Len() != 0 || goal.RefCount != 1 {
		t.Fatalf("disabled trace or result ownership: %q, refs %d", output.String(), goal.RefCount)
	}
	tags.Enable("ufbv-rewriter")
	result, err = rewriter.Apply(goal)
	cacheMust(t, err)
	if result != goal || goal.RefCount != 2 {
		t.Fatal("trace ran before result publication", result, goal.RefCount)
	}
	cacheMust(t, result.DecRef())
	if !tags.ShouldTrace("ufbv-rewriter") {
		if output.Len() != 0 {
			t.Fatal("erased trace emitted", output.String())
		}
		return
	}
	want := "-------- [ufbv-rewriter] operator() src/tactic/ufbv/ufbv_rewriter_tactic.cpp:67 ---------\n" +
		"(goal\n  trace_atom)\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source rewritten-goal trace: got %q, want %q", got, want)
	}
	output.Reset()
	_, err = uparams.BindProcessGlobalParamsService(nil)
	cacheMust(t, err)
	result, err = rewriter.Apply(goal)
	if result != nil || !errors.Is(err, uparams.ErrGlobalParamsOwner) || goal.RefCount != 1 {
		t.Fatalf("formatter refusal retained published goal: %v, %v, refs %d", result, err, goal.RefCount)
	}
	want = "-------- [ufbv-rewriter] operator() src/tactic/ufbv/ufbv_rewriter_tactic.cpp:67 ---------\n(goal\n  "
	if got := output.String(); got != want {
		t.Fatalf("source formatter-refusal prefix: got %q, want %q", got, want)
	}
}
