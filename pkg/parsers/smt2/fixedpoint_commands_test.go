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
		t.Fatal("fixedpoint SMT-LIB differential", output.String(), cpp)
	}
}
