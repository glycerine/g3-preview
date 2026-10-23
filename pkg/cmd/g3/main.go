// g3 executes SMT-LIB scripts through the native Go SMT backend.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/shell"
)

func run(input io.Reader, output io.Writer) (result error) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, context.Close()) }()
	return smt2.ParseCommands(context, input, output)
}

func runArguments(args []string, input io.Reader, output io.Writer) (result error) {
	context, err := cmd_context.NewStandaloneContext()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, context.Close()) }()
	globals, err := context.GlobalParameters()
	if err != nil {
		return err
	}
	command := shell.NewCommandLine(nil, nil, globals)
	if err = command.Parse(args); err != nil {
		return err
	}
	if command.Action == shell.CommandHelp {
		_, err = fmt.Fprintln(output, "g3 Go SMT-LIB runner\nUsage: g3 [-smt2] <file.smt2> | g3 -in\nGlobal parameter options: name=value; -p, -pd, -pm, -pp:name")
		return err
	}
	if command.Action == shell.CommandVersion {
		_, err = fmt.Fprintln(output, "g3 (Go SMT-LIB backend)")
		return err
	}
	if command.Action != shell.CommandContinue {
		return command.WriteTerminalAction(output)
	}
	filename, selected := command.Input.Selected()
	if selected && command.StandardInput {
		return fmt.Errorf("using standard input to read formula")
	}
	if !selected && !command.StandardInput {
		return fmt.Errorf("input file was not specified")
	}
	kind, err := command.ResolveInputKind()
	if err != nil {
		return err
	}
	if kind != shell.InputSMT2 {
		return fmt.Errorf("selected input frontend is not ported yet")
	}
	if command.DisplayStatistics || command.DisplayIStatistics {
		return fmt.Errorf("shell statistics output is not ported yet")
	}
	if selected {
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, file.Close()) }()
		input = file
	}
	if err = context.RefreshGlobalParameters(); err != nil {
		return err
	}
	return smt2.ParseCommands(context, input, output)
}

func main() {
	if err := runArguments(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stdout, "(error \"%s\")\n", strings.ReplaceAll(err.Error(), "\"", "\"\""))
		os.Exit(1)
	}
}
