package engine

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: generated mem_initialize/finalize and debug.cpp:37-42,73-117.
// The generated API process must supply real standard streams and input to the
// process-global assertion action, including its C continuation and T throw.
func TestDefaultProcessActualAssertionAction(t *testing.T) {
	if route := os.Getenv("G3_DEFAULT_DEBUG_CHILD"); route != "" {
		previous := diagnostic.BorrowProcessDebugContext()
		if err := InitializeDefaultProcess(); err != nil {
			t.Fatal(err)
		}
		d := apiDefaultProcess
		if d == nil || diagnostic.BorrowProcessDebugContext() != d.debug {
			t.Fatal("generated debug context was not published")
		}
		err := diagnostic.DebugAssertionFailure(d.debug, "default.cpp", 41, "x")
		if route == "continue" {
			if err != nil {
				t.Fatal(err)
			}
			_, _ = os.Stdout.WriteString("continued\n")
		} else {
			var ex z3exception.DefaultException
			if !errors.As(err, &ex) || ex.Message() != "assertion violation" {
				t.Fatal("source T action", err)
			}
			_, _ = os.Stdout.WriteString("caught\n")
		}
		if err = FinalizeDefaultProcess(); err != nil {
			t.Fatal(err)
		}
		if apiDefaultProcess != nil || diagnostic.BorrowProcessDebugContext() != previous || d.budget.Used() != 0 {
			t.Fatal("generated assertion process survived finalization")
		}
		return
	}
	for _, tc := range []struct{ route, input, output string }{
		{"continue", " C", "continued\n"},
		{"throw", "T", "caught\n"},
	} {
		t.Run(tc.route, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDefaultProcessActualAssertionAction$")
			cmd.Env = append(os.Environ(), "G3_DEFAULT_DEBUG_CHILD="+tc.route)
			cmd.Stdin = strings.NewReader(tc.input)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatal(err, stderr.String())
			}
			if ctx.Err() != nil {
				t.Fatal("assertion action timed out", ctx.Err())
			}
			want := "ASSERTION VIOLATION\nFile: default.cpp\nLine: 41\nx\n" +
				"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
			if stdout.String() != tc.output+"PASS\n" || stderr.String() != want {
				t.Fatalf("default process assertion: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}
