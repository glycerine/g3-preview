// Source: src/shell/main.cpp:214-245, retained global-parameter output options.
package shell

import (
	"bytes"
	"errors"
	"io"
)

var ErrShellTerminalOutputOpen = errors.New("shell: terminal output action is not yet ported")

// WriteTerminalAction executes the source's parameter display branches after
// Parse selects a terminal action. The same output buffer receives the -pp
// diagnostic line and descriptor text, retaining their stdout order.
func (c *CommandLine) WriteTerminalAction(out io.Writer) error {
	if c == nil {
		return ErrShellOptionService
	}
	if c.Action == CommandContinue {
		return nil
	}
	if out == nil {
		return ErrShellOptionService
	}
	if c.Action == CommandHelp || c.Action == CommandVersion {
		return ErrShellTerminalOutputOpen
	}
	if c.Globals == nil {
		return ErrShellOptionService
	}
	var text bytes.Buffer
	var render error
	switch c.Action {
	case CommandDisplayParams:
		render = c.Globals.SourceDisplayInto(&text, 0, false, false)
	case CommandDisplayParamDescriptions:
		render = c.Globals.SourceDisplayInto(&text, 0, false, true)
	case CommandDisplayModules:
		render = c.Globals.SourceDisplayModulesInto(&text)
		if render == nil {
			text.WriteString("\nUse -pm:name to display all parameters available at module 'name'\n")
		}
	case CommandDisplayModule:
		render = c.Globals.SourceDisplayModuleInto(&text, c.ActionArgument)
	case CommandDisplayParameter:
		render = c.Globals.SourceDisplayParameterInto(&text, &text, c.ActionArgument)
	default:
		return ErrShellTerminalOutputOpen
	}
	_, writeErr := io.Copy(out, &text)
	return errors.Join(render, writeErr)
}

// ParseAndWrite runs the command-line scan and the reached terminal output
// branch. The executable's exit and frontend dispatch still belong to main.
func (c *CommandLine) ParseAndWrite(args []string, out io.Writer) error {
	if err := c.Parse(args); err != nil {
		return err
	}
	return c.WriteTerminalAction(out)
}
