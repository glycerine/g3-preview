package smt2_test

// Source: src/muz/fp/dl_cmds.cpp declare-rel, rule, and query commands.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
)

func fixedpointReference(t *testing.T, script string) string {
	t.Helper()
	path := os.Getenv("G3_CPP_REFERENCE")
	if path == "" {
		var err error
		path, err = filepath.Abs("../../../build/z3")
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("C++ reference executable unavailable:", err)
	}
	command := exec.Command(path, "-in")
	command.Stdin = strings.NewReader(script)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(output))
	}
	return string(output)
}

func TestSMT2FixedpointPropositionalQueriesMatchCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel P ())
(declare-rel Q ())
(declare-rel R ())
(declare-rel U ())
(rule P)
(rule (=> P Q))
(rule (=> Q R))
(query R :print-answer true :print-statistics false :print-certificate false)
(query U :print-answer true)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\ntrue\nunsat\n" {
		t.Fatal("fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointRuleNameAndBoundMatchCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel P ())
(rule P named-rule 0)
(query P)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\n" {
		t.Fatal("fixedpoint rule bound differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointRuleScopesMatchCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel P ())
(declare-rel Q ())
(rule P)
(fixedpoint-push)
(rule (=> P Q))
(fixedpoint-pop)
(query Q :print-answer true)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "unsat\n" {
		t.Fatal("fixedpoint rule-scope differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointFiniteRelationsMatchCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel P (Bool))
(declare-rel Q (Bool))
(declare-rel U (Bool))
(declare-var x Bool)
(rule (P true))
(rule (=> (P x) (Q x)))
(query Q)
(query U)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\nunsat\n" {
		t.Fatal("finite fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointTwoTailJoinMatchesCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel P (Bool Bool))
(declare-rel Q (Bool Bool))
(declare-rel R (Bool Bool))
(declare-rel U (Bool Bool))
(declare-var x Bool)
(declare-var y Bool)
(declare-var z Bool)
(rule (P true false))
(rule (P false true))
(rule (Q false true))
(rule (Q true false))
(rule (=> (and (P x y) (Q y z)) (R z x)))
(query R)
(query U)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\nunsat\n" {
		t.Fatal("two-tail fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointSimpleJoinNormalizationMatchesCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
	(declare-rel A (Bool Bool))
(declare-rel B (Bool Bool))
(declare-rel C (Bool Bool))
(declare-rel Goal (Bool Bool))
(declare-var w Bool)
(declare-var x Bool)
(declare-var y Bool)
(declare-var z Bool)
(rule (A true false))
(rule (B false true))
(rule (C true false))
(rule (=> (and (A x y) (B y z) (C z w)) (Goal x w)))
(query Goal)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\n" {
		t.Fatal("simple-join fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointRecursiveClosureMatchesCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-rel Edge (Bool Bool))
(declare-rel Path (Bool Bool))
(declare-rel Goal ())
(declare-var x Bool)
(declare-var y Bool)
(declare-var z Bool)
(rule (Edge true false))
(rule (Edge false true))
(rule (=> (Edge x y) (Path x y)))
(rule (=> (and (Path x y) (Edge y z)) (Path x z)))
(rule (=> (Path true true) Goal))
(query Goal)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\n" {
		t.Fatal("recursive fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointEmptyProgramQueryMatchesCPP(t *testing.T) {
	script := "(declare-rel P ())\n(query P)\n"
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "unsat\n" {
		t.Fatal("empty fixedpoint program differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointQueryRefusesUnknownOutputParameters(t *testing.T) {
	for _, parameter := range []string{":print-proof true"} {
		t.Run(parameter, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			})
			script := "(declare-rel P ())\n(query P " + parameter + ")\n"
			err = smt2.ParseCommands(context, strings.NewReader(script), &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "unsupported fixedpoint query parameter") {
				t.Fatal("unavailable query output accepted", err)
			}
		})
	}
}

func TestSMT2FixedpointPDRStatistics(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := `(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(declare-rel Q (Int))
(declare-var x Int)
(rule (P 0))
(rule (=> (P x) (Q x)))
(query Q :print-answer false :print-statistics true)
`
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, expected := range []string{"sat\n(", ":PDR-num-unfoldings", ":PDR-max-depth", ":time"} {
		if !strings.Contains(text, expected) {
			t.Fatal("missing PDR statistic", expected, text)
		}
	}
	if cpp := fixedpointReference(t, script); !strings.Contains(cpp, "\nsat\n") || !strings.Contains(cpp, ":time") {
		t.Fatal("C++ PDR statistics result", cpp)
	}
}

func TestSMT2FixedpointTupleAnswerMatchesCPP(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := "(declare-rel P (Bool))\n(rule (P true))\n(rule (P false))\n(query P :print-answer true)\n"
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\n(or (= (:var 0) true) (= (:var 0) false))\n" {
		t.Fatal("tuple answer differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointFiniteVariableExpansionMatchesCPP(t *testing.T) {
	script := `(set-option :fixedpoint.engine datalog)
(declare-var x Bool)
(declare-rel All (Bool))
(rule (All x))
(query All :print-answer true)
(declare-rel Blocked (Bool))
(declare-rel Q (Bool))
(declare-rel Goal ())
(rule (Blocked true))
(rule (=> (not (Blocked x)) (Q x)))
(rule (=> (Q false) Goal))
(query Goal)
`
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	context.PrintSuccess = false
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	want := "sat\n(or (= (:var 0) false) (= (:var 0) true))\nsat\n"
	if output.String() != cpp || cpp != want {
		t.Fatal("finite variable expansion differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointRejectsUnstratifiedNegation(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	script := `(declare-rel P ())
(declare-rel Q ())
(rule (=> (not P) Q))
(rule (=> (not Q) P))
(query P)
`
	err = smt2.ParseCommands(context, strings.NewReader(script), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "negation is not stratified") {
		t.Fatal("unstratified negation accepted", err)
	}
}

func TestSMT2FixedpointSpacerIntegerQueriesMatchCPP(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := `(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(declare-rel Q (Int))
(declare-rel U (Int))
(declare-var x Int)
(rule (P 0))
(rule (=> (P x) (Q x)))
(query Q :print-answer true :print-certificate true)
(query U :print-answer false)
`
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\ntrue\n\nunsat\n" {
		t.Fatal("spacer integer SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointPDRBackgroundAssertionsMatchCPP(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := `(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(rule (P 0))
(query P :print-answer false)
(assert false)
(query P :print-answer false)
`
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\nunsat\n" {
		t.Fatal("PDR background SMT-LIB differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointPDRRecursiveTraceAnswerMatchesCPP(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := `(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(declare-rel Goal ())
(declare-rel U ())
(declare-var x Int)
(rule (P 0))
(rule (=> (and (P x) (< x 2)) (P (+ x 1))))
(rule (=> (P 2) Goal))
(query Goal :print-answer true)
(query U :print-answer false)
`
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	want := "sat\n(and (P 1) (P 0))\nunsat\n"
	if output.String() != cpp || cpp != want {
		t.Fatal("PDR recursive trace differential", output.String(), cpp)
	}
}

func TestSMT2FixedpointPDRTraceReductionMatchesCPP(t *testing.T) {
	tests := []struct {
		name, script, want string
	}{
		{
			"acyclic-branch",
			`(set-option :fixedpoint.engine spacer)
(declare-rel A (Int))
(declare-rel B (Int))
(declare-rel C (Int))
(declare-rel Goal ())
(declare-var x Int)
(rule (A 0))
(rule (B 0))
(rule (=> (and (A x) (B x)) (C x)))
(rule (=> (C 0) Goal))
(query Goal :print-answer true)
`,
			"sat\ntrue\n",
		},
		{
			"mutual-recursion",
			`(set-option :fixedpoint.engine spacer)
(declare-rel P (Int))
(declare-rel Q (Int))
(declare-rel Goal ())
(declare-var x Int)
(rule (P 0))
(rule (=> (P x) (Q (+ x 1))))
(rule (=> (Q x) (P (+ x 1))))
(rule (=> (P 2) Goal))
(query Goal :print-answer true)
`,
			"sat\n(and (P 0))\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, err := cmd_context.NewStandaloneContext()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := context.Close(); err != nil {
					t.Fatal(err)
				}
			})
			context.PrintSuccess = false
			var output bytes.Buffer
			if err = smt2.ParseCommands(context, strings.NewReader(test.script), &output); err != nil {
				t.Fatal(err)
			}
			cpp := fixedpointReference(t, test.script)
			if output.String() != cpp || cpp != test.want {
				t.Fatal("PDR trace reduction differential", output.String(), cpp)
			}
		})
	}
}

func TestSMT2FixedpointAutoConfigIntegerQueriesMatchCPP(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Fatal(err)
		}
	})
	context.PrintSuccess = false
	script := `(declare-rel P (Int))
(declare-rel Q (Int))
(declare-rel U (Int))
(declare-var x Int)
(rule (P 0))
(rule (=> (P x) (Q x)))
(query Q :print-answer false)
(query U :print-answer false)
`
	var output bytes.Buffer
	if err = smt2.ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err)
	}
	cpp := fixedpointReference(t, script)
	if output.String() != cpp || cpp != "sat\nunsat\n" {
		t.Fatal("auto-config integer SMT-LIB differential", output.String(), cpp)
	}
}
