package smt2

// Source differential: src/cmd_context/basic_cmds.cpp pp_cmd.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2DisplayUsesParsedExpressionPrinter(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	var output bytes.Buffer
	script := `(declare-const x Int)(display (+ x 1))(display (and true false))`
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if output.String() != "(+ x 1)\n(and true false)\n" {
		t.Fatal("display output", output.String())
	}
}

func TestSMT2DisplayRejectsMissingAndTrailingTerms(t *testing.T) {
	for _, script := range []string{`(display)`, `(display true false)`} {
		context, err := cmd_context.NewStandaloneContext()
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		err = ParseCommands(context, strings.NewReader(script), &output)
		closeErr := context.Close()
		if err == nil || closeErr != nil || output.Len() != 0 {
			t.Fatal("display syntax guard", script, err, closeErr, output.String())
		}
	}
}
