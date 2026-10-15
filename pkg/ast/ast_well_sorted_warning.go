// Source: src/ast/well_sorted.cpp:63-71.
package ast

import (
	"errors"

	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ASTManager) wellSortedMismatchWarning(w wellSortedWarning, c *uparams.ParamsContext, p *uparams.ParamsRef, g *uparams.GlobalParamsService) (message string, result error) {
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.BudgetField})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	if err = out.AppendCString("Sort mismatch for argument "); err != nil {
		return "", err
	}
	if err = out.AppendUint32(w.argument); err != nil {
		return "", err
	}
	if err = out.AppendCString(" of "); err != nil {
		return "", err
	}
	if err = m.DisplayLowLevel(out, w.app, false, true); err != nil {
		return "", err
	}
	if err = out.AppendCString("\nExpected sort: "); err != nil {
		return "", err
	}
	want, err := m.SMT2PPSort(w.want)
	if err != nil {
		return "", err
	}
	if err = m.PrintFormat(out, want, c, p, g); err != nil {
		return "", err
	}
	if err = out.AppendCString("\nActual sort:   "); err != nil {
		return "", err
	}
	actual, err := m.SMT2PPSort(w.actual)
	if err != nil {
		return "", err
	}
	if err = m.PrintFormat(out, actual, c, p, g); err != nil {
		return "", err
	}
	if err = out.AppendCString("\nFunction sort: "); err != nil {
		return "", err
	}
	decl, err := m.SMT2PPDeclareFun(w.decl)
	if err != nil {
		return "", err
	}
	if err = m.PrintFormat(out, decl, c, p, g); err != nil {
		return "", err
	}
	if err = out.AppendCString("."); err != nil {
		return "", err
	}
	return out.String()
}
