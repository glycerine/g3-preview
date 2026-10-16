// Source: src/tactic/goal.cpp:402-410.
package tactic

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func goalDisplayWrite(out io.Writer, value string) error {
	n, err := io.WriteString(out, value)
	if err != nil {
		return err
	}
	if n != len(value) {
		return io.ErrShortWrite
	}
	return nil
}

func (g *Goal) displaySMT2Formula(h ast.ASTHandle) (body string, result error) {
	global := uparams.BorrowProcessGlobalParamsService()
	if global == nil || global.Context == nil {
		return "", uparams.ErrGlobalParamsOwner
	}
	format, err := g.manager.SMT2PPExprTree(h, nil)
	if err != nil {
		return "", err
	}
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: g.manager.BudgetField})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, stream.Close()) }()
	empty := global.Context.EmptyValue()
	if err = g.manager.PrintFormat(stream, format, global.Context, &empty, global); err != nil {
		return "", err
	}
	return stream.String()
}

// Display writes the source default goal printer's formula-only SMT2 form.
// The process parameter owner is borrowed only when a formula is printed.
func (g *Goal) Display(out io.Writer) error {
	if err := g.open(); err != nil {
		return err
	}
	if out == nil {
		return arena.ErrInvalidSlot
	}
	if err := goalDisplayWrite(out, "(goal"); err != nil {
		return err
	}
	for _, form := range g.forms {
		if err := goalDisplayWrite(out, "\n  "); err != nil {
			return err
		}
		body, err := g.displaySMT2Formula(form)
		if err != nil {
			return err
		}
		if err = goalDisplayWrite(out, body); err != nil {
			return err
		}
	}
	return goalDisplayWrite(out, ")\n")
}
