// Source: src/sat/tactic/goal2sat.cpp:396-417; src/util/trace.h:40-46.
package tactic

import (
	"errors"
	"io"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (tr *GoalTranslator) traceGoalFormula(h ast.ASTHandle) (body string, result error) {
	global := uparams.BorrowProcessGlobalParamsService()
	if global == nil || global.Context == nil {
		return "", uparams.ErrGlobalParamsOwner
	}
	format, err := tr.manager.SMT2PPExprTree(h, nil)
	if err != nil {
		return "", err
	}
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: tr.manager.BudgetField})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	empty := global.Context.EmptyValue()
	if err = tr.manager.PrintFormat(out, format, global.Context, &empty, global); err != nil {
		return "", err
	}
	return out.String()
}

func (tr *GoalTranslator) traceGoalResultStack(line int, label string, fr satGoalFrame) error {
	tags := tr.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("goal2sat_bug") {
		return nil
	}
	return tags.EmitConditional("goal2sat_bug", "process", "src/sat/tactic/goal2sat.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, label); err != nil {
				return err
			}
			formula, err := tr.traceGoalFormula(fr.term)
			if err != nil {
				return err
			}
			root, sign := "0", "0"
			if fr.root {
				root = "1"
			}
			if fr.sign {
				sign = "1"
			}
			if _, err = io.WriteString(w, formula+" root: "+root+" sign: "+sign+"\n"); err != nil {
				return err
			}
			var stack strings.Builder
			for i, l := range tr.results {
				if i != 0 {
					stack.WriteByte(' ')
				}
				stack.WriteString(l.String())
			}
			_, err = io.WriteString(w, stack.String()+"\n")
			return err
		})
}
