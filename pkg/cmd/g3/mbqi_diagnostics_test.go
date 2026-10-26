package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSMTLibRunnerMBQITraceParameterUpdatesCPPReference(t *testing.T) {
	script, err := os.ReadFile("test_vectors/mbqi_trace_parameter_updates.smt2")
	if err != nil {
		t.Fatal(err)
	}
	previous, err := diagnostic.BorrowProcessVerboseStream()
	if err != nil {
		t.Fatal(err)
	}
	level, err := diagnostic.ProcessVerbosityLevel()
	if err != nil {
		t.Fatal(err)
	}
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := diagnostic.SetProcessVerboseStream(previous); err != nil {
			t.Error(err)
		}
		if err := diagnostic.SetProcessVerbosityLevel(level); err != nil {
			t.Error(err)
		}
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = diagnostic.SetProcessVerboseStream(stream); err != nil {
		t.Fatal(err)
	}
	if err = diagnostic.SetProcessVerbosityLevel(0); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = run(bytes.NewReader(script), &out); err != nil {
		t.Fatal(err)
	}
	const result = "sat\nsat\nsat\n"
	if out.String() != result {
		t.Fatalf("results %q", out.String())
	}
	got, err := stream.String()
	if err != nil {
		t.Fatal(err)
	}
	const round = "(smt.mbqi \"started\")\n(smt.mbqi :checking accepted)\n(smt.mbqi :failed accepted)\n(smt.mbqi :num-failures 1)\n(smt.mbqi \"started\")\n(smt.mbqi :checking accepted)\n(smt.mbqi :succeeded true)\n"
	if got != round+round {
		t.Fatalf("trace enable/disable/re-enable differs:\n%s", got)
	}
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in")
	cmd.Stdin = strings.NewReader(string(script))
	var cppOut, cppErr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &cppOut, &cppErr
	if err = cmd.Run(); err != nil {
		t.Fatal(err, cppErr.String())
	}
	if cppOut.String() != result || cppErr.String() != got {
		t.Fatalf("native/C++ MBQI diagnostics differ\nGo result %q\nC++ result %q\nGo trace:\n%s\nC++ trace:\n%s", out.String(), cppOut.String(), got, cppErr.String())
	}
}
