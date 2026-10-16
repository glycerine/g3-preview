// Source: src/tactic/extension_model_converter.cpp:59; src/util/trace.h:51.
package tactic

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (c *tacticExtensionModelConverter) extensionTraceTags() *diagnostic.TraceTags {
	if c.services.Trace != nil {
		return c.services.Trace
	}
	return c.manager.BorrowMemoryTraceTags()
}

func (c *tacticExtensionModelConverter) writeExtensionSMT2(w io.Writer, h ast.ASTHandle) (result error) {
	format, err := c.manager.SMT2PPExprTree(h, nil)
	if err != nil {
		return err
	}
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.manager.Budget()})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	empty := c.services.Context.EmptyValue()
	if err = c.manager.PrintFormat(out, format, c.services.Context, &empty, c.services.Globals); err != nil {
		return err
	}
	body, err := out.String()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, body)
	return err
}

func (c *tacticExtensionModelConverter) traceExtensionDefinition(decl, definition, value ast.ASTHandle) error {
	tags := c.extensionTraceTags()
	if !tags.ShouldTrace("extension_mc") {
		return nil
	}
	name, err := c.manager.GetASTName(decl)
	if err != nil {
		return err
	}
	spelling, err := c.manager.Symbols.String(name)
	if err != nil {
		return err
	}
	return tags.EmitConditional("extension_mc", "operator()", "src/tactic/extension_model_converter.cpp", 59,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, spelling+" ->\n"); err != nil {
				return err
			}
			if err := c.writeExtensionSMT2(w, definition); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n==>\n"); err != nil {
				return err
			}
			if err := c.writeExtensionSMT2(w, value); err != nil {
				return err
			}
			_, err := io.WriteString(w, "\n")
			return err
		})
}
