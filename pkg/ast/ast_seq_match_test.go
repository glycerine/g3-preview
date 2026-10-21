package ast

import (
	"errors"
	"testing"
)

func TestASTSequenceSourcePolymorphicSignatureMatching(t *testing.T) {
	m := astTestManager(t)
	str, err := m.MkString("x")
	astTestMust(t, err)
	stringSort, err := m.GetSort(str)
	astTestMust(t, err)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	parameter, err := m.NewParameterAST(boolSort)
	astTestMust(t, err)
	sequenceBool, err := m.MkFamilySort(m.sequencePlugin.family, 0, []ASTParameter{parameter})
	astTestMust(t, err)
	astTestMust(t, m.DelParameter(&parameter))

	m.mu.Lock()
	defer m.mu.Unlock()
	stringRef, err := m.requireKind(stringSort, ASTSort)
	astTestMust(t, err)
	boolRef, err := m.requireKind(boolSort, ASTSort)
	astTestMust(t, err)
	sequenceBoolRef, err := m.requireKind(sequenceBool, ASTSort)
	astTestMust(t, err)
	result, err := m.sequenceMatchSignature(m.sequencePlugin.signatures[ASTSequenceOpUnit], []astNodeRef{boolRef}, astNodeRef{}, false)
	astTestMust(t, err)
	if result != sequenceBoolRef {
		t.Fatalf("unit result: %v, want %v", result, sequenceBoolRef)
	}
	result, err = m.sequenceMatchSignature(m.sequencePlugin.signatures[3], []astNodeRef{stringRef, stringRef}, astNodeRef{}, false)
	astTestMust(t, err)
	if result != m.boolSort {
		t.Fatalf("prefix result: %v", result)
	}
	result, err = m.sequenceMatchSignature(m.sequencePlugin.signatures[ASTSequenceOpConcat], []astNodeRef{stringRef, stringRef, stringRef}, astNodeRef{}, true)
	astTestMust(t, err)
	if result != stringRef {
		t.Fatalf("right-associative concat result: %v", result)
	}
	if _, err = m.sequenceMatchSignature(m.sequencePlugin.signatures[3], []astNodeRef{stringRef, sequenceBoolRef}, astNodeRef{}, false); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("mismatched polymorphic arguments: %v", err)
	}
	if _, err = m.sequenceMatchSignature(m.sequencePlugin.signatures[ASTSequenceOpEmpty], nil, astNodeRef{}, false); !errors.Is(err, ErrASTDomain) {
		t.Fatalf("ambiguous empty sequence: %v", err)
	}
	result, err = m.sequenceMatchSignature(m.sequencePlugin.signatures[ASTSequenceOpEmpty], nil, sequenceBoolRef, false)
	astTestMust(t, err)
	if result != sequenceBoolRef {
		t.Fatalf("range-bound empty sequence: %v", result)
	}
}
