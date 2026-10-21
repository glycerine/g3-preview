package ast

import "testing"

func TestASTSequenceConcatUtilitiesAndRegexLoopRecognition(t *testing.T) {
	m := astTestManager(t)
	a, err := m.MkString("a")
	astTestMust(t, err)
	b, err := m.MkString("b")
	astTestMust(t, err)
	empty, err := m.MkString("")
	astTestMust(t, err)
	right, err := m.MkSequenceConcat(empty, b)
	astTestMust(t, err)
	root, err := m.MkSequenceConcat(a, right)
	astTestMust(t, err)
	leftPart, rightPart, matched, err := m.SequenceConcatParts(root)
	astTestMust(t, err)
	if !matched || leftPart != a || rightPart != right {
		t.Fatalf("concat parts: %v %v %v", leftPart, rightPart, matched)
	}
	leaves, err := m.SequenceFlattenConcat(root)
	astTestMust(t, err)
	if len(leaves) != 2 || leaves[0] != a || leaves[1] != b {
		t.Fatalf("flattened concat: %v", leaves)
	}
	leftmost, err := m.SequenceLeftmostConcat(root)
	astTestMust(t, err)
	rightmost, err := m.SequenceRightmostConcat(root)
	astTestMust(t, err)
	if leftmost != a || rightmost != b {
		t.Fatalf("concat edges: left=%v right=%v", leftmost, rightmost)
	}

	regex, err := m.MkSequenceToRegex(a)
	astTestMust(t, err)
	for _, tc := range []struct {
		lower, upper int32
		bounded      bool
	}{
		{3, 0, false},
		{2, 7, true},
	} {
		var loop ASTHandle
		if tc.bounded {
			loop, err = m.MkRegexLoop(regex, tc.lower, tc.upper)
		} else {
			loop, err = m.MkRegexLoop(regex, tc.lower)
		}
		astTestMust(t, err)
		body, lower, upper, bounded, matched, e := m.RegexLoopParameters(loop)
		astTestMust(t, e)
		if !matched || body != regex || lower != tc.lower || upper != tc.upper || bounded != tc.bounded {
			t.Fatalf("loop parameters: body=%v lower=%d upper=%d bounded=%v matched=%v", body, lower, upper, bounded, matched)
		}
	}
	if _, _, _, _, matched, err = m.RegexLoopParameters(a); err != nil || matched {
		t.Fatalf("non-loop recognition: matched=%v error=%v", matched, err)
	}
}
