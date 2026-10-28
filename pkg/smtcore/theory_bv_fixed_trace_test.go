package smtcore

// Source: src/smt/theory_bv.cpp:321,337-350,471-473.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSMTBVInitWatchAndFixedEqualitySourceTrace(t *testing.T) {
	var output bytes.Buffer
	m, state, theory := bvOccurrenceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"init_bits", "find_wpos", "fixed_var_eh"} {
		tags.Enable(tag)
	}
	theory.trace = tags
	left, leftVariable := bvFixedSymbolic(t, m, state, theory, "trace-fixed-left", 2)
	right, rightVariable := bvFixedSymbolic(t, m, state, theory, "trace-fixed-right", 2)
	sum, err := m.MkBitVectorAdd(left.owner, right.owner)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(sum, false))

	bvFixedAssign(t, state, theory, leftVariable, 1)
	bvFixedAssign(t, state, theory, rightVariable, 1)
	astTestMust(t, theory.findWatchedPosition(leftVariable))
	astTestMust(t, theory.findWatchedPosition(rightVariable))

	if tags.ShouldTrace("init_bits") {
		trace := output.String()
		for _, want := range []string{
			"-------- [init_bits] init_bits src/smt/theory_bv.cpp:321 ---------\nbit 0 of #",
			"-------- [find_wpos] find_wpos src/smt/theory_bv.cpp:337 ---------\ncurr bit: ",
			"-------- [find_wpos] find_wpos src/smt/theory_bv.cpp:339 ---------\nmoved wpos of v",
			"-------- [find_wpos] find_wpos src/smt/theory_bv.cpp:350 ---------\nv0 is a fixed variable.\n",
			"-------- [fixed_var_eh] fixed_var_eh src/smt/theory_bv.cpp:471 ---------\ndetected equality: v1 = v0\n",
		} {
			if !strings.Contains(trace, want) {
				t.Fatalf("missing BV fixed-bit source trace %q in %q", want, trace)
			}
		}
	} else if output.Len() != 0 {
		t.Fatal("compiled-out BV fixed-bit source trace executed", output.String())
	}
}
