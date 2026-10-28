package api

import (
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTContextSMT2ReplaysDeclarationsAssertionsAndScopes(t *testing.T) {
	source, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	script := `(set-option :produce-unsat-cores true)
(set-logic QF_UF)
(declare-sort U 0)
(declare-fun p (U) Bool)
(declare-const u U)
(assert (p u))
(push 1)
(declare-const q Bool)
(assert (! q :named inner))
(push 1)
(declare-const r Bool)
(assert (not r))
`
	var responses strings.Builder
	astTestMust(t, source.ExecuteSMT2(strings.NewReader(script), &responses))
	state, err := source.SMT2()
	astTestMust(t, err)
	if strings.Count(state, "(push 1)") != 2 || !strings.Contains(state, "(declare-sort U)") || !strings.Contains(state, ":named inner") {
		t.Fatalf("incomplete replay state:\n%s", state)
	}
	firstPush := strings.Index(state, "(push 1)")
	qDeclaration := strings.Index(state, "(declare-fun q () Bool)")
	secondPush := strings.LastIndex(state, "(push 1)")
	rDeclaration := strings.Index(state, "(declare-fun r () Bool)")
	if firstPush < 0 || qDeclaration < firstPush || secondPush < qDeclaration || rDeclaration < secondPush {
		t.Fatalf("declarations lost their scope:\n%s", state)
	}

	replayed, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, replayed.Close()) })
	responses.Reset()
	astTestMust(t, replayed.ExecuteSMT2(strings.NewReader(state), &responses))
	for _, context := range []*SMTContext{source, replayed} {
		level, x := context.ScopeLevel()
		astTestMust(t, x)
		assertions, x := context.Assertions()
		astTestMust(t, x)
		if level != 2 || len(assertions) != 3 {
			t.Fatal("replayed state shape", level, len(assertions))
		}
		status, x := context.CheckSat()
		astTestMust(t, x)
		if status != lbool.True {
			t.Fatal("replayed state status", status)
		}
		astTestMust(t, context.Pop(1))
		responses.Reset()
		astTestMust(t, context.ExecuteSMT2(strings.NewReader("(assert (not q))"), &responses))
		status, x = context.CheckSat()
		astTestMust(t, x)
		if status != lbool.False {
			t.Fatal("replayed scoped assertion", status)
		}
	}
}

func TestSMTContextWriteSMT2GuardsAndMatchesStringForm(t *testing.T) {
	context, err := NewSMTContext()
	astTestMust(t, err)
	state, err := context.SMT2()
	astTestMust(t, err)
	var output strings.Builder
	astTestMust(t, context.WriteSMT2(&output))
	if output.String() != state {
		t.Fatal("stream and string exports differ")
	}
	if err = context.WriteSMT2(nil); err == nil {
		t.Fatal("nil export destination accepted")
	}
	astTestMust(t, context.Close())
	if _, err = context.SMT2(); err == nil {
		t.Fatal("closed context exported")
	}
}

func TestSMTContextSMT2ReplaysParametricSortsAndNestedConstantArrays(t *testing.T) {
	source, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, source.Close()) })
	script := `(set-logic ALL)
(declare-sort Box 1)
(define-sort Grid (T) (Array T (Array T Bool)))
(declare-const g (Grid Bool))
(assert (= g ((as const (Grid Bool)) ((as const (Array Bool Bool)) false))))
(push 1)
(declare-const boxed (Box Bool))
`
	var output strings.Builder
	astTestMust(t, source.ExecuteSMT2(strings.NewReader(script), &output))
	state, err := source.SMT2()
	astTestMust(t, err)
	if !strings.Contains(state, "(declare-sort Box 1)") || !strings.Contains(state, "(define-sort Grid (T) (Array T (Array T Bool)))") || strings.Contains(state, "(declare-sort (Box Bool))") {
		t.Fatalf("parametric sort replay:\n%s", state)
	}
	replayed, err := NewSMTContext()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, replayed.Close()) })
	output.Reset()
	if err = replayed.ExecuteSMT2(strings.NewReader(state), &output); err != nil {
		t.Fatalf("replay failed: %v\n%s", err, state)
	}
	status, err := replayed.CheckSat()
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal("nested constant-array replay", status)
	}
}
