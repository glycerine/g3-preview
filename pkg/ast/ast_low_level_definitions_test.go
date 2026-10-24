package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTLowLevelDefinitionsShareVisitedAcrossRoots(t *testing.T) {
	m := astTestManager(t)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "definition-p"), b)
	astTestMust(t, err)
	astTestPin(t, m, p)
	np, err := m.MkNot(p)
	astTestMust(t, err)
	astTestPin(t, m, np)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	astTestMust(t, err)
	defer func() { astTestMust(t, out.Close()) }()
	astTestMust(t, m.DisplayLowLevelExpressionDefinitions(out, []ASTHandle{np, p, np}))
	pid, err := m.GetASTID(p)
	astTestMust(t, err)
	nid, err := m.GetASTID(np)
	astTestMust(t, err)
	want := fmt.Sprintf("#%d := definition-p\n#%d := (not definition-p)\n", pid, nid)
	got, err := out.String()
	astTestMust(t, err)
	if got != want {
		t.Fatalf("shared source definition traversal: got %q, want %q", got, want)
	}
	if m.LowLevel.Active || m.LowLevel.stack != nil || m.LowLevel.marks != nil {
		t.Fatal("definition printer retained completed local marks or frames")
	}
}

func TestASTLowLevelDefinitionCallerMarksAndOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "marked-definition-p"), b)
	astTestMust(t, err)
	astTestPin(t, m, p)
	np, err := m.MkNot(p)
	astTestMust(t, err)
	astTestPin(t, m, np)
	nnp, err := m.MkNot(np)
	astTestMust(t, err)
	astTestPin(t, m, nnp)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	astTestMust(t, err)
	defer func() { astTestMust(t, out.Close()) }()
	visited, err := NewASTMark(m)
	astTestMust(t, err)
	astTestMust(t, m.DisplayLowLevelDefinitionWithVisited(out, np, visited))
	astTestMust(t, m.DisplayLowLevelDefinitionWithVisited(out, nnp, visited))
	pid, err := m.GetASTID(p)
	astTestMust(t, err)
	nid, err := m.GetASTID(np)
	astTestMust(t, err)
	nnid, err := m.GetASTID(nnp)
	astTestMust(t, err)
	want := fmt.Sprintf("#%d := marked-definition-p\n#%d := (not marked-definition-p)\n#%d := (not #%d)\n", pid, nid, nnid, nid)
	got, err := out.String()
	astTestMust(t, err)
	if got != want {
		t.Fatal("caller mark was not shared across independent calls", got, want)
	}
	marked, err := visited.IsMarked(np)
	astTestMust(t, err)
	if !marked {
		t.Fatal("printer did not update the actual caller mark")
	}
	astTestMust(t, m.DisplayLowLevelWithVisited(out, np, visited))
	got, err = out.String()
	astTestMust(t, err)
	if got != want {
		t.Fatal("marked root was printed again", got)
	}
	astTestMust(t, visited.Reset())
	astTestMust(t, m.DisplayLowLevelWithVisited(out, np, visited))
	got, err = out.String()
	astTestMust(t, err)
	if got != want+"(not marked-definition-p)\n" {
		t.Fatal("compact visited overload inlined the wrong root", got)
	}
	copied := *visited
	if err := m.DisplayLowLevelWithVisited(out, np, &copied); !errors.Is(err, ErrASTHandle) {
		t.Fatal("copied mark owner was accepted", err)
	}
	foreign := astTestManager(t)
	other, err := NewASTMark(foreign)
	astTestMust(t, err)
	if err := m.DisplayLowLevelWithVisited(out, np, other); !errors.Is(err, ErrASTHandle) {
		t.Fatal("foreign mark owner was accepted", err)
	}
	astTestMust(t, other.Close())
	astTestMust(t, visited.Close())
	if err := m.DisplayLowLevelWithVisited(out, np, visited); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed mark owner was accepted", err)
	}
	if m.LowLevel.Active || m.LowLevel.visited != nil {
		t.Fatal("completed printer retained caller mark")
	}
}

func TestASTLowLevelProofUsesCallerVisitedMark(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "proof-mark-p"), b)
	astTestMust(t, err)
	proof, err := m.MkProofAsserted(p)
	astTestMust(t, err)
	astTestPin(t, m, proof)
	visited, err := NewASTMark(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, visited.Close()) }()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	astTestMust(t, err)
	defer func() { astTestMust(t, out.Close()) }()
	astTestMust(t, m.DisplayLowLevelWithVisited(out, proof, visited))
	got, err := out.String()
	astTestMust(t, err)
	if got != "[asserted]: proof-mark-p\n" {
		t.Fatal("retained compact proof overload output", got)
	}
	marked, err := visited.IsMarked(proof)
	astTestMust(t, err)
	if !marked {
		t.Fatal("proof traversal did not publish caller visitation")
	}
}
