package smtcore

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTStringFinalVariableScanUsesAllPrivateClasses(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	first := stringGroundingWord(t, theory, "final-free-first")
	second := stringGroundingWord(t, theory, "final-free-dependent")
	assigned := stringGroundingWord(t, theory, "final-assigned")
	internal := stringGroundingWord(t, theory, "final-internal")
	regex := stringGroundingWord(t, theory, "final-regex-witness")
	theory.internalVariables[internal] = struct{}{}
	theory.regexVariables[regex] = struct{}{}
	constant := astTestString(t, m, "assigned-value")
	astTestMust(t, state.Internalize(constant, false))
	stringDependencyPrivateMerge(t, theory, assigned, constant)
	analysis := &smtStringDependencyAnalysis{variables: map[ast.ASTHandle]int{first: 1, second: 2, assigned: 1, internal: 1, regex: 1}, freeVariables: map[ast.ASTHandle]int{first: 1}}
	free, err := theory.unassignedStringVariables(analysis)
	astTestMust(t, err)
	if len(free) != 2 || free[0] != first || free[1] != second {
		t.Fatal("source scan used only dependency roots, ignored private values or included excluded witnesses", free)
	}
	stringDependencyPrivateMerge(t, theory, first, constant)
	free, err = theory.unassignedStringVariables(analysis)
	astTestMust(t, err)
	if len(free) != 1 || free[0] != second {
		t.Fatal("private equivalence-class assignment was not recognized", free)
	}
}

func TestSMTStringAssignedFinalCheckRequiresAssignedRegexPaths(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	word := stringGroundingWord(t, theory, "assigned-path-word")
	member := stringRegexRegister(t, theory, word, stringRegexLiteral(t, m, "a"))
	status, err := theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("undefined membership blocked native assigned completion", status)
	}
	for _, assignment := range []lbool.LBool{lbool.True, lbool.False} {
		variable, found := state.BoolVar(member)
		if !found {
			t.Fatal("missing membership variable")
		}
		astTestMust(t, state.SetAssignment(variable, assignment))
		status, err = theory.finalCheckAssignedStrings()
		astTestMust(t, err)
		if status != SMTFinalCheckContinue {
			t.Fatal("assigned membership without paths was accepted", assignment, status)
		}
		theory.regexStage.pathDone[member] = theory.level
		status, err = theory.finalCheckAssignedStrings()
		astTestMust(t, err)
		if status != SMTFinalCheckDone {
			t.Fatal("completed membership path was not recognized", assignment, status)
		}
		delete(theory.regexStage.pathDone, member)
	}
	// Source automata-disabled mode bypasses the path-completeness gate.
	m, theory, _, _ = stringLegacyRegexFixture(t)
	word = stringGroundingWord(t, theory, "assigned-legacy-word")
	member, err = m.MkSequenceInRegex(word, stringRegexLiteral(t, m, "a"))
	astTestMust(t, err)
	astTestMust(t, theory.state.Internalize(member, true))
	variable, _ := theory.state.BoolVar(member)
	astTestMust(t, theory.state.SetAssignment(variable, lbool.True))
	theory.regexTerms[member] = struct{}{}
	status, err = theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("disabled automata mode required an unused path flag", status)
	}
}

func TestSMTStringAssignedFinalCheckConversionOrderAndContinuation(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	word := stringGroundingWord(t, theory, "assigned-stoi-word")
	text := astTestString(t, m, "42")
	astTestMust(t, state.Internalize(text, false))
	stringDependencyPrivateMerge(t, theory, word, text)
	stoi, err := m.MkStringToInt(word)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(stoi, false))
	astTestMust(t, theory.recordStringIntConversion(stoi))
	integer := stringLibraryInteger(t, m, state, "assigned-itos-argument")
	itos, err := m.MkIntToString(integer)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(itos, false))
	astTestMust(t, theory.recordStringIntConversion(itos))
	seven := astTestString(t, m, "7")
	astTestMust(t, state.Internalize(seven, false))
	stringDependencyPrivateMerge(t, theory, itos, seven)
	status, err := theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || len(theory.conversionAxioms) != 2 {
		t.Fatal("conversion dispatch stopped early or skipped an operator", status, len(theory.conversionAxioms))
	}
	// The unknown stoi arithmetic root deliberately keeps requesting the
	// native unasserted zero split, even after agreement is memoized.
	before := len(theory.refs)
	status, err = theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue || len(theory.refs) != before {
		t.Fatal("unknown conversion completion or memoization changed", status, len(theory.refs), before)
	}
	fortyTwo, err := m.ArithmeticIntegerNumeralText("42")
	astTestMust(t, err)
	stringConversionCoreEquality(t, state, stoi, fortyTwo)
	// The forward canonical-string agreement is a distinct source axiom;
	// let the complete ordered conversion pass publish it before completion.
	status, err = theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckContinue {
		t.Fatal("new forward conversion agreement did not continue", status)
	}
	status, err = theory.finalCheckAssignedStrings()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("stable assigned conversions did not complete", status)
	}
}

func TestSMTStringAssignedFinalCheckDoesNotHideLaterConversionError(t *testing.T) {
	m, theory, state, _ := stringLibraryFixture(t)
	word := stringGroundingWord(t, theory, "assigned-conversion-error-word")
	text := astTestString(t, m, "5")
	astTestMust(t, state.Internalize(text, false))
	stringDependencyPrivateMerge(t, theory, word, text)
	stoi, err := m.MkStringToInt(word)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(stoi, false))
	astTestMust(t, theory.recordStringIntConversion(stoi))
	integer := stringLibraryInteger(t, m, state, "unassigned-itos-argument")
	itos, err := m.MkIntToString(integer)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(itos, false))
	astTestMust(t, theory.recordStringIntConversion(itos))
	status, err := theory.finalCheckAssignedStrings()
	if err == nil || status != SMTFinalCheckGiveUp || len(theory.conversionAxioms) != 1 {
		t.Fatal("first conversion progress hid the next source precondition failure", status, err, len(theory.conversionAxioms))
	}
}

func TestSMTStringGroundedFinalCheckAndModelUseAssignedBytes(t *testing.T) {
	for _, want := range []string{"grounded", "a\x00\xffb"} {
		m, parent, theory := stringExpressionSolverFixture(t)
		word := stringGroundingWord(t, theory, "grounded-final-model-word")
		value := astTestString(t, m, want)
		equal, err := m.MkEq(word, value)
		astTestMust(t, err)
		astTestMust(t, parent.Assert(equal, ast.ASTHandle{}))
		parent.produceModel = true
		status, err := parent.Check()
		astTestMust(t, err)
		if status != lbool.True || parent.generated == nil {
			t.Fatal("actual grounded String kernel did not complete/build its model", status)
		}
		decl, err := m.GetDecl(word)
		astTestMust(t, err)
		actual, err := parent.GeneratedModel().ConstantInterpretation(decl)
		astTestMust(t, err)
		text, constant, err := m.SequenceStringValue(actual)
		astTestMust(t, err)
		if !constant || text != want {
			t.Fatal("grounded model lost assigned String bytes", constant, text)
		}
		if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" && want == "grounded" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			command := exec.CommandContext(ctx, reference, "-in")
			command.Stdin = strings.NewReader("(set-option :smt.string_solver z3str3)\n(set-logic QF_S)\n(declare-const s String)\n(assert (= s \"grounded\"))\n(check-sat)\n(get-value (s))\n")
			output, err := command.CombinedOutput()
			cancel()
			if err != nil || strings.TrimSpace(string(output)) != "sat\n((s \""+text+"\"))" {
				t.Fatal("grounded legacy String model differs from the prebuilt C++ reference", err, string(output))
			}
		}
	}
}
