package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func smtPBTraceRecord(function string, line int, body string) string {
	return fmt.Sprintf("-------- [pb] %s src/smt/theory_pb.cpp:%d ---------\n%s------------------------------------------------\n", function, line, body)
}

func TestSMTPBTraceInternalizationAndPropagationExactRecords(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	_, _, s := smtInternalizerFixture(t, m, nil)
	pb, err := NewSMTPseudoBooleanTheory(family)
	astTestMust(t, err)
	astTestMust(t, s.RegisterTheory(pb))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("pb")
	pb.trace = tags
	h := smtPBResolveExpression(t, m, family, []ast.ASTHandle{smtRelevancyBool(t, m, "trace-x"), smtRelevancyBool(t, m, "trace-y")}, []int32{1, 1}, 2)
	astTestMust(t, s.Internalize(h, true))
	expr := "((_ pbge 2 1 1) trace-x trace-y)"
	want := smtPBTraceRecord("internalize_atom", 470, "p1@(l_undef) "+expr+"\np2@(l_undef) + p3@(l_undef) >= 2\n") +
		smtPBTraceRecord("internalize_atom", 484, "p1 "+expr+"\np2 + p3 >= 2\n := p1\n") +
		smtPBTraceRecord("internalize_atom", 533, "compilation threshold: 18\n") +
		smtPBTraceRecord("internalize_atom", 602, "p1 "+expr+"\np2 + p3 >= 2\nsum: [0:2] \n")
	if tags.ShouldTrace("pb") {
		if output.String() != want {
			t.Fatal("source internalization trace", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("native trace was not erased")
	}
	output.Reset()
	astTestMust(t, s.PushScope())
	atom, err := s.Literal(h)
	astTestMust(t, err)
	smtPBWatchAssign(t, s, atom)
	want = smtPBTraceRecord("assign_eh", 785, "assign: p1\n") +
		smtPBTraceRecord("assign_ineq", 949, "assign: p1\np1 "+expr+"\np2 + p3 >= 2\n") +
		smtPBTraceRecord("add_assign", 1431, "#prop:1 - p1 => p2\np1@(l_true:1) "+expr+"\np2@(l_undef) + p3@(l_undef) .w  >= 2\npropagations: 1 max_watch: 1 watch size: 2 watch-sum: 2 \n") +
		smtPBTraceRecord("add_assign", 1431, "#prop:2 - p1 => p3\np1@(l_true:1) "+expr+"\np2@(l_true:1) + p3@(l_undef) .w  >= 2\npropagations: 2 max_watch: 1 watch size: 2 watch-sum: 2 \n") +
		smtPBTraceRecord("assign_eh", 785, "assign: p2\n") +
		smtPBTraceRecord("assign_eh", 785, "assign: p3\n")
	if tags.ShouldTrace("pb") {
		if output.String() != want {
			t.Fatal("source propagation trace", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("native trace was not erased")
	}
}

func TestSMTPBTraceFormatterFailurePrecedesPropagationPublication(t *testing.T) {
	for _, badStream := range []bool{false, true} {
		t.Run(fmt.Sprint(badStream), func(t *testing.T) {
			m, s, pb, atom, inputs := smtPBWatchFixture(t, []int64{1, 1}, 2, false)
			astTestMust(t, s.AssignLiteral(atom, NewSMTBooleanAxiom()))
			var output bytes.Buffer
			var writer io.Writer = &output
			if badStream {
				writer = smtPBDisplayFailure{}
			}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("pb")
			pb.trace = tags
			beforeOwners := len(s.ownedJustifications)
			m.Budget().FailAfter(0)
			err := pb.addAssign(pb.constraints[0], []SMTLiteral{atom}, inputs[0])
			m.Budget().FailAfter(-1)
			value, valueErr := s.Assignment(inputs[0])
			astTestMust(t, valueErr)
			if tags.ShouldTrace("pb") {
				if !errors.Is(err, arena.ErrInjectedFailure) || value != lbool.Undef || len(s.ownedJustifications) != beforeOwners || pb.statistics.propagations != 1 {
					t.Fatal("trace operand failure lost source publication boundary", err, value, pb.statistics)
				}
				if !badStream {
					want := "-------- [pb] add_assign src/smt/theory_pb.cpp:1431 ---------\n#prop:1 - p1 => p2\np1@(l_true:0) "
					if output.String() != want {
						t.Fatal("formatter refusal source prefix", output.String(), want)
					}
				}
			} else {
				astTestMust(t, err)
				if output.Len() != 0 || value != lbool.True {
					t.Fatal("native trace operands were evaluated", value, output.String())
				}
			}
		})
	}
}

func TestSMTPBTraceBadStreamAndDisabledOperandErasure(t *testing.T) {
	_, s, pb, atom, inputs := smtPBWatchFixture(t, []int64{1, 1}, 2, false)
	tags := diagnostic.NewTraceTagsWithWriter(smtPBDisplayFailure{})
	pb.trace = tags
	astTestMust(t, pb.traceConstraint("internalize_atom", 470, nil, true))
	tags.Enable("pb")
	astTestMust(t, s.AssignLiteral(atom, NewSMTBooleanAxiom()))
	astTestMust(t, pb.addAssign(pb.constraints[0], []SMTLiteral{atom}, inputs[0]))
	value, err := s.Assignment(inputs[0])
	astTestMust(t, err)
	if value != lbool.True {
		t.Fatal("ostream badbit stopped source assignment")
	}
	var output bytes.Buffer
	disabled := diagnostic.NewTraceTagsWithWriter(&output)
	pb.trace = disabled
	astTestMust(t, pb.traceConstraint("internalize_atom", 470, nil, true))
	if output.Len() != 0 {
		t.Fatal("disabled trace evaluated operands")
	}
}

func TestSMTPBTraceSimplexAndCompilationUseSourceSites(t *testing.T) {
	_, s, pb, xs := smtPBSimplexFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("pb")
	pb.trace = tags
	_ = smtPBSimplexAtom(t, s, pb, xs, []int32{1, 1, 1}, 2, false)
	_ = smtPBSimplexAtom(t, s, pb, xs, []int32{1, 1, 1}, 3, false)
	if tags.ShouldTrace("pb") {
		for _, part := range []string{"internalize_atom src/smt/theory_pb.cpp:555", "internalize_atom src/smt/theory_pb.cpp:597", "New row: 4 2\np1 + p2 + p3 ~ 2\n", "internalize_atom src/smt/theory_pb.cpp:558", "Old row: 5 |-> 4 2 vs. 3\n"} {
			if !strings.Contains(output.String(), part) {
				t.Fatal("missing source simplex record", part, output.String())
			}
		}
	}
	output.Reset()
	astTestMust(t, pb.compileInequality(pb.constraints[0]))
	if tags.ShouldTrace("pb") {
		if !strings.Contains(output.String(), smtPBTraceRecord("compile_ineq", 1279, "p1 p2 p3 >= 2\n")) || !strings.Contains(output.String(), "compile_ineq src/smt/theory_pb.cpp:1294") {
			t.Fatal("missing source compilation records", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("native compilation trace was not erased")
	}
}

func TestSMTPBTraceWatchConflictStillEmitsTrailingRecord(t *testing.T) {
	_, s, pb, atom, inputs := smtPBWatchFixture(t, []int64{4, 3, 2, 2, 1}, 6, false)
	smtPBWatchAssign(t, s, atom)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("pb")
	pb.trace = tags
	astTestMust(t, s.AssignLiteral(inputs[0].Negated(), NewSMTBooleanAxiom()))
	astTestMust(t, s.AssignLiteral(inputs[1].Negated(), NewSMTBooleanAxiom()))
	_, err := s.PropagateCore()
	astTestMust(t, err)
	if !s.Inconsistent() {
		t.Fatal("weighted conflict fixture did not conflict")
	}
	if tags.ShouldTrace("pb") {
		last := strings.LastIndex(output.String(), "-------- [pb] assign_watch_ge src/smt/theory_pb.cpp:1157")
		clause := strings.LastIndex(output.String(), "-------- [pb] add_clause src/smt/theory_pb.cpp:1469")
		if clause < 0 || last < clause || strings.Contains(output.String()[last:], "@(") {
			t.Fatal("source conflict tail was skipped or gained value annotations", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("native watch trace was not erased")
	}
}
