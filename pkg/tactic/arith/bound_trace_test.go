package arith

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestBoundAssertionSourceTraceAndFalseAlarmOrder(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, false))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"bound_propagator_detail", "bound_propagator_derived", "new_bound"} {
		tags.Enable(tag)
	}
	equationMust(t, s.BindTrace(tags))
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	assert := func(n int32, lower, derived, want bool) {
		t.Helper()
		equationMust(t, m.SetInt32(&value, n))
		got, err := s.assertBound(0, &value, false, lower, derived)
		equationMust(t, err)
		if got != want {
			t.Fatal("bound acceptance", n, lower, derived, got, want)
		}
	}
	assert(1, true, false, true)
	assert(1, true, true, false)
	assert(2, true, true, true)
	assert(4, false, false, true)
	assert(4, false, true, false)
	assert(3, false, true, true)
	if !tags.ShouldTrace("new_bound") {
		if output.Len() != 0 {
			t.Fatal("erased bound TRACE wrote output", output.String())
		}
		return
	}
	record := func(tag, function string, line int, body string) string {
		return "-------- [" + tag + "] " + function + " src/tactic/arith/bound_propagator.cpp:" + strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
	}
	var want strings.Builder
	want.WriteString(record("bound_propagator_detail", "assert_lower_core", 279, "new lower x0 1 strict: 0\n"))
	want.WriteString(record("new_bound", "assert_lower_core", 302, "x0 lower: 1 approx: 1\n"))
	want.WriteString(record("new_bound", "assert_lower_core", 305, "x0 lower: 1 relaxed approx: 1\n"))
	want.WriteString(record("bound_propagator_detail", "assert_lower_core", 279, "new lower x0 1 strict: 0\n"))
	want.WriteString(record("bound_propagator_detail", "assert_lower_core", 286, "false alarm\n"))
	want.WriteString(record("bound_propagator_detail", "assert_lower_core", 279, "new lower x0 2 strict: 0\n"))
	want.WriteString(record("bound_propagator_derived", "assert_lower_core", 294, "new lower x0 2 strict: 0\n"))
	want.WriteString(record("new_bound", "assert_lower_core", 302, "x0 lower: 2 approx: 2\n"))
	want.WriteString(record("new_bound", "assert_lower_core", 305, "x0 lower: 2 relaxed approx: 2\n"))
	want.WriteString(record("bound_propagator_detail", "assert_upper_core", 331, "new upper x0 4 strict: 0\n"))
	want.WriteString(record("new_bound", "assert_upper_core", 354, "x0 upper: 4 approx: 4\n"))
	want.WriteString(record("new_bound", "assert_upper_core", 357, "x0 upper: 4 relaxed approx: 4\n"))
	want.WriteString(record("bound_propagator_detail", "assert_upper_core", 331, "new upper x0 4 strict: 0\n"))
	want.WriteString(record("bound_propagator_detail", "assert_upper_core", 338, "false alarm\n"))
	want.WriteString(record("bound_propagator_detail", "assert_upper_core", 331, "new upper x0 3 strict: 0\n"))
	want.WriteString(record("bound_propagator_derived", "assert_upper_core", 347, "new upper x0 3 strict: 0\n"))
	want.WriteString(record("new_bound", "assert_upper_core", 354, "x0 upper: 3 approx: 3\n"))
	want.WriteString(record("new_bound", "assert_upper_core", 357, "x0 upper: 3 relaxed approx: 3\n"))
	if output.String() != want.String() {
		t.Fatalf("bound TRACE source records\n got: %q\nwant: %q", output.String(), want.String())
	}
	before := output.String()
	for _, tag := range []string{"bound_propagator_detail", "bound_propagator_derived", "new_bound"} {
		tags.Disable(tag)
	}
	assert(2, true, true, false)
	if output.String() != before {
		t.Fatal("disabled bound TRACE emitted output")
	}
}

func TestBoundRelevanceSourceTraceOldValueAndSmallImprovement(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, false))
	equationMust(t, s.MkVar(1, false))
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	equationMust(t, m.SetInt32(&value, 1))
	_, err = s.AssertLower(0, &value, false)
	equationMust(t, err)
	equationMust(t, m.SetInt32(&value, 4))
	_, err = s.AssertUpper(0, &value, false)
	equationMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bound_propagator_detail")
	tags.Enable("bound_propagator")
	equationMust(t, s.BindTrace(tags))
	check := func(x uint32, candidate float64, lower, want bool) {
		t.Helper()
		got, err := s.relevantBound(x, candidate, lower)
		equationMust(t, err)
		if got != want {
			t.Fatal("relevance decision", x, candidate, lower, got, want)
		}
	}
	check(0, 1.02, true, false)
	check(0, 3.98, false, false)
	check(0, 2, true, true)
	check(1, 7, true, true)
	if !tags.ShouldTrace("bound_propagator_detail") {
		if output.Len() != 0 {
			t.Fatal("erased relevance TRACE wrote output", output.String())
		}
		return
	}
	record := func(tag string, line int, body string) string {
		return "-------- [" + tag + "] relevant_bound src/tactic/arith/bound_propagator.cpp:" + strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("bound_propagator_detail", 382, "relevant_bound x0 1.02 LOWER: 1\nold: 1 | 1\n") +
		record("bound_propagator", 405, "LOWER new: 1.02 old: 1 improvement is too small\n") +
		record("bound_propagator_detail", 382, "relevant_bound x0 3.98 LOWER: 0\nold: 4 | 4\n") +
		record("bound_propagator", 411, "UPPER new: 3.98 old: 4 improvement is too small\n") +
		record("bound_propagator_detail", 382, "relevant_bound x0 2 LOWER: 1\nold: 1 | 1\n") +
		record("bound_propagator_detail", 382, "relevant_bound x1 7 LOWER: 1\n")
	if got := output.String(); got != want {
		t.Fatalf("relevance TRACE source records\n got: %q\nwant: %q", got, want)
	}
}
