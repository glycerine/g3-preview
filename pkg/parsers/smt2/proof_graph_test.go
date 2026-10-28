package smt2

// Source differential: src/cmd_context/basic_cmds.cpp get_proof_graph_cmd.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
)

func TestSMT2GetProofGraphWritesConfiguredDOTFile(t *testing.T) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	})
	path := filepath.Join(t.TempDir(), "proof graph.dot")
	script := `(set-option :produce-proofs true)(set-option :dot-proof-file ` + strconv.Quote(path) + `)
(declare-const a Bool)(assert a)(assert (not a))(check-sat)(get-proof-graph)`
	var output bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &output); err != nil {
		t.Fatal(err, output.String())
	}
	if output.String() != "unsat\n" {
		t.Fatal("proof graph command output", output.String())
	}
	graph, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(graph, []byte("digraph proof {")) || !bytes.Contains(graph, []byte("color=\"red\"")) {
		t.Fatal("proof graph file", string(graph))
	}
}

func TestSMT2GetProofGraphCPPReference(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the C++ z3 executable")
	}
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := context.Close(); err != nil {
			t.Error(err)
		}
	}()
	path := filepath.Join(t.TempDir(), "proof.dot")
	script := `(set-option :produce-proofs true)(set-option :dot-proof-file ` + strconv.Quote(path) + `)
(declare-const a Bool)(assert a)(assert (not a))(check-sat)(get-proof-graph)`
	var goOutput bytes.Buffer
	if err = ParseCommands(context, strings.NewReader(script), &goOutput); err != nil {
		t.Fatal(err)
	}
	goGraph, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(reference, "-in")
	command.Stdin = strings.NewReader(script)
	cppOutput, err := command.Output()
	if err != nil {
		t.Fatal(err, string(cppOutput))
	}
	cppGraph, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if goOutput.String() != string(cppOutput) || !bytes.Equal(goGraph, cppGraph) {
		t.Fatal("proof graph C++ differential", goOutput.String(), string(cppOutput), string(goGraph), string(cppGraph))
	}
}

func TestSMT2GetProofGraphGuards(t *testing.T) {
	for name, script := range map[string]string{
		"disabled":    `(get-proof-graph)`,
		"unavailable": `(set-option :produce-proofs true)(get-proof-graph)`,
		"trailing":    `(get-proof-graph extra)`,
	} {
		t.Run(name, func(t *testing.T) {
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
			if err = ParseCommands(context, strings.NewReader(script), &output); err == nil {
				t.Fatal("proof graph guard accepted", script, output.String())
			}
		})
	}
}
