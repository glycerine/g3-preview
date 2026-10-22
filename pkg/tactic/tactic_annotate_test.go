// Source: src/tactic/tactical.cpp:1190-1223.
package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestAnnotateTacticVerboseScopeIncludesFailureExit(t *testing.T) {
	b := arena.NewBudget(4096)
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	astTestMust(t, err)
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	astTestMust(t, err)
	oldStream, err := diagnostic.BorrowProcessVerboseStream()
	astTestMust(t, err)
	astTestMust(t, diagnostic.SetProcessVerboseStream(stream))
	t.Cleanup(func() {
		astTestMust(t, diagnostic.SetProcessVerboseStream(oldStream))
		astTestMust(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
		astTestMust(t, stream.Close())
		if b.Used() != 0 {
			t.Fatal("annotation stream retained memory", b.Used())
		}
	})
	astTestMust(t, diagnostic.SetProcessVerbosityLevel(tacticVerbosityLevel))
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	child := NewFailTactic()
	annotated, err := NewAnnotateTactic("phase", child)
	astTestMust(t, err)
	tacticOwnForTest(t, annotated)
	out := &TacticResult{}
	if err = annotated.Apply(g, out); err == nil || err.Error() != "fail tactic" {
		t.Fatal("annotated child result", err)
	}
	got, err := stream.String()
	astTestMust(t, err)
	if got != "(phase start)\n(phase done)\n" {
		t.Fatal("annotation output", got)
	}
	astTestMust(t, out.Close())
}

func TestAnnotateTacticVerbosityGate(t *testing.T) {
	oldLevel, err := diagnostic.ProcessVerbosityLevel()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, diagnostic.SetProcessVerbosityLevel(oldLevel)) })
	astTestMust(t, diagnostic.SetProcessVerbosityLevel(tacticVerbosityLevel-1))
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	child := &countingSkipTactic{primitiveTactic: newPrimitiveTactic(primitiveSkip, "", 0)}
	annotated, err := NewAnnotateTactic("quiet", child)
	astTestMust(t, err)
	tacticOwnForTest(t, annotated)
	out := &TacticResult{}
	astTestMust(t, annotated.Apply(g, out))
	if child.applications != 1 {
		t.Fatal("quiet annotation child", child.applications)
	}
	astTestMust(t, out.Close())
}
