// Source: src/ast/well_sorted.cpp:49; src/util/trace.h:40-46.
package ast

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ASTManager) traceWellSortedArity(app ASTHandle, c *uparams.ParamsContext, p *uparams.ParamsRef, g *uparams.GlobalParamsService) error {
	tags := m.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("ws") {
		return nil
	}
	return tags.EmitConditional("ws", "operator()", "src/ast/well_sorted.cpp", 49,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, "unexpected number of arguments.\n"); err != nil {
				return err
			}
			if c == nil || p == nil || g == nil {
				g = uparams.BorrowProcessGlobalParamsService()
				if g == nil || g.Context == nil {
					return uparams.ErrGlobalParamsOwner
				}
				c = g.Context
				empty := c.EmptyValue()
				p = &empty
			}
			format, err := m.SMT2PPBooleanTree(app, nil)
			if err != nil {
				return err
			}
			out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.BudgetField})
			if err != nil {
				return err
			}
			result := m.PrintFormat(out, format, c, p, g)
			if result == nil {
				var body string
				body, result = out.String()
				if result == nil {
					_, result = io.WriteString(w, body)
				}
			}
			return errors.Join(result, out.Close())
		})
}
