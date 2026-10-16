// Source: src/model/model_evaluator.cpp:223,228,247,566; src/util/trace.h.
package model

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

// BindTrace borrows a trace stream for reached evaluator sites. Nil restores
// the source process stream.
func (ev *ModelEvaluator) BindTrace(tags *diagnostic.TraceTags) error {
	if e := ev.open(); e != nil {
		return e
	}
	ev.services.Trace = tags
	return nil
}

func (ev *ModelEvaluator) traceEvaluate(t ast.ASTHandle) error {
	tags := ev.services.Trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("model_evaluator") {
		return nil
	}
	return tags.EmitConditional("model_evaluator", "operator()", "src/model/model_evaluator.cpp", 566,
		func() bool { return true }, func(w io.Writer) (result error) {
			format, err := ev.model.manager.SMT2PPExprTree(t, nil)
			if err != nil {
				return err
			}
			out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: ev.model.manager.BudgetField})
			if err != nil {
				return err
			}
			defer func() { result = errors.Join(result, out.Close()) }()
			empty := ev.services.Context.EmptyValue()
			if err = ev.model.manager.PrintFormat(out, format, ev.services.Context, &empty, ev.services.Globals); err != nil {
				return err
			}
			body, err := out.String()
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, body+"\n")
			return err
		})
}

func (c *evaluatorConfiguration) traceGetMacro(d ast.ASTHandle, line int) error {
	tags := c.owner.services.Trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("model_evaluator") {
		return nil
	}
	m := c.owner.model.manager
	name, e := m.GetASTName(d)
	if e != nil {
		return e
	}
	spelling, e := m.Symbols.String(name)
	if e != nil {
		return e
	}
	completion := "0"
	if c.completion {
		completion = "1"
	}
	// Source ostream failures set its stream state without changing the macro.
	_ = tags.Emit("model_evaluator", "get_macro", "src/model/model_evaluator.cpp", line,
		"get_macro for "+spelling+" (model completion: "+completion+")\n")
	return nil
}
