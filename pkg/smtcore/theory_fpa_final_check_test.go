package smtcore

// Source: src/smt/theory_fpa.cpp:716-720.

import (
	"bytes"
	"testing"
)

func TestSMTFPAFinalCheckSourceTraceAndInvariant(t *testing.T) {
	var output bytes.Buffer
	_, theory, _, _, tags := smtFPAInternalizeTraceFixture(t, &output)
	tags.Enable("t_fpa")
	status, err := theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("FPA final check status", status)
	}
	if tags.ShouldTrace("t_fpa") {
		want := "-------- [t_fpa] final_check_eh src/smt/theory_fpa.cpp:717 ---------\nfinal_check_eh\n------------------------------------------------\n"
		if output.String() != want {
			t.Fatal("FPA final-check source trace", output.String(), want)
		}
	} else if output.Len() != 0 {
		t.Fatal("compiled-out FPA final-check trace executed", output.String())
	}
}

func TestSMTFPAFinalCheckTraceStreamRefusalDoesNotStop(t *testing.T) {
	_, theory, _, _, tags := smtFPAInternalizeTraceFixture(t, smtFPAInternalizeRefusingWriter{})
	tags.Enable("t_fpa")
	status, err := theory.FinalCheck()
	astTestMust(t, err)
	if status != SMTFinalCheckDone {
		t.Fatal("ostream badbit stopped FPA final check", status)
	}
}
