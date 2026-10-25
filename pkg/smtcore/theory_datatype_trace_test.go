package smtcore

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func smtDatatypeTraceRecord(tag, function string, line int, body string) string {
	return fmt.Sprintf("-------- [%s] %s src/smt/theory_datatype.cpp:%d ---------\n%s------------------------------------------------\n", tag, function, line, body)
}

func TestSMTDatatypeEqualityTraceSourceBranchesBeforeQueuedMerge(t *testing.T) {
	for _, proofs := range []bool{false, true} {
		for _, guard := range []string{"null", "true", "undefined"} {
			t.Run(fmt.Sprint(proofs, "/", guard), func(t *testing.T) {
				m, dt, x, sort, _ := smtDatatypeSplitFixture(t)
				if proofs {
					astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
				}
				s, err := dt.State()
				astTestMust(t, err)
				y, err := m.MkConst(astTestSymbol(t, m, "datatype-trace-y"), sort)
				astTestMust(t, err)
				astTestMust(t, s.Internalize(y, false))
				p := smtRelevancyBool(t, m, "datatype-trace-guard")
				astTestMust(t, s.Internalize(p, true))
				antecedent, err := s.Literal(p)
				astTestMust(t, err)
				if guard == "null" {
					antecedent = SMTNullLiteral
				} else if guard == "true" {
					astTestMust(t, s.AssignLiteral(antecedent, NewSMTBooleanAxiom()))
				}
				var output bytes.Buffer
				tags := diagnostic.NewTraceTagsWithWriter(&output)
				tags.Enable("datatype")
				tags.Enable("datatype_detail")
				dt.trace = tags
				astTestMust(t, dt.equalityAxiom(x, y, antecedent))
				if proofs || !tags.ShouldTrace("datatype") {
					if output.Len() != 0 {
						t.Fatal("proof branch or native erasure evaluated datatype trace operands", output.String())
					}
					return
				}
				want := smtDatatypeTraceRecord("datatype", "assert_eq_axiom", 100, "adding axiom:\nincremental-x\n=\ndatatype-trace-y\n")
				if guard != "true" {
					if output.String() != want {
						t.Fatal("source equality trace branch", output.String(), want)
					}
					return
				}
				left, right := s.context.enodes[x], s.context.enodes[y]
				lid, err := left.OwnerID()
				astTestMust(t, err)
				rid, err := right.OwnerID()
				astTestMust(t, err)
				want += smtDatatypeTraceRecord("datatype", "assert_eq_axiom", 116, fmt.Sprintf("assigning... #%d #%d\nv%d v%d\n", lid, rid, left.TheoryVar(dt.FamilyID()), right.TheoryVar(dt.FamilyID())))
				want += "-------- [datatype_detail] assert_eq_axiom src/smt/theory_datatype.cpp:118 ---------\nTheory datatype:\n"
				if !strings.HasPrefix(output.String(), want) || left.root == right.root {
					t.Fatal("source pre-merge record order", output.String(), want)
				}
			})
		}
	}
}

func TestSMTDatatypeTraceEntryOrderAndIndependentDetailTag(t *testing.T) {
	m, dt, x, sort, constructors := smtDatatypeSplitFixture(t)
	s, err := dt.State()
	astTestMust(t, err)
	recognizer, err := m.DatatypeConstructorRecognizer(constructors[0])
	astTestMust(t, err)
	atom, err := m.MkApp(recognizer, []ast.ASTHandle{x})
	astTestMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("datatype")
	dt.trace = tags
	astTestMust(t, s.Internalize(atom, true))
	if tags.ShouldTrace("datatype") {
		atomEntry := strings.Index(output.String(), "[datatype] internalize_atom src/smt/theory_datatype.cpp:258")
		termEntry := strings.Index(output.String(), "[datatype] internalize_term src/smt/theory_datatype.cpp:263")
		if atomEntry < 0 || termEntry <= atomEntry {
			t.Fatal("source datatype entry records missing or reordered", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("native entry trace was not erased")
	}
	y, err := m.MkConst(astTestSymbol(t, m, "detail-trace-y"), sort)
	astTestMust(t, err)
	astTestMust(t, s.Internalize(y, false))
	literal, err := s.Literal(atom)
	astTestMust(t, err)
	astTestMust(t, s.AssignLiteral(literal, NewSMTBooleanAxiom()))
	output.Reset()
	tags.Disable("datatype")
	tags.Enable("datatype_detail")
	astTestMust(t, dt.equalityAxiom(x, y, literal))
	if tags.ShouldTrace("datatype_detail") {
		if !strings.HasPrefix(output.String(), "-------- [datatype_detail] assert_eq_axiom src/smt/theory_datatype.cpp:118") || strings.Contains(output.String(), "[datatype]") {
			t.Fatal("detail tag was coupled to datatype tag", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("native detail trace was not erased")
	}
}

func TestSMTDatatypeTraceBadStreamPreservesQueuedEquality(t *testing.T) {
	m, dt, x, sort, _ := smtDatatypeSplitFixture(t)
	s, err := dt.State()
	astTestMust(t, err)
	y, err := m.MkConst(astTestSymbol(t, m, "bad-stream-datatype-y"), sort)
	astTestMust(t, err)
	astTestMust(t, s.Internalize(y, false))
	tags := diagnostic.NewTraceTagsWithWriter(smtPBDisplayFailure{})
	tags.Enable("datatype")
	dt.trace = tags
	astTestMust(t, dt.equalityAxiom(x, y, SMTNullLiteral))
	if s.context.enodes[x].root == s.context.enodes[y].root {
		t.Fatal("bad stream changed source deferred equality")
	}
	_, err = s.PropagateCore()
	astTestMust(t, err)
	if s.context.enodes[x].root != s.context.enodes[y].root {
		t.Fatal("sticky ostream failure stopped equality publication")
	}
	var output bytes.Buffer
	dt.trace = diagnostic.NewTraceTagsWithWriter(&output)
	astTestMust(t, dt.traceDatatypeExpression("internalize_term", 263, "erased:\n", ast.ASTHandle{}))
	if output.Len() != 0 {
		t.Fatal("disabled trace evaluated operands")
	}
}
