// Source: src/ast/ast.cpp:2641-2643; src/util/trace.h:40-46.
package ast

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ASTManager) proofModusPonensTraceTags() *diagnostic.TraceTags {
	if m.traceTags != nil {
		return m.traceTags
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (m *ASTManager) proofModusPonensTraceEnabled() bool {
	return m.proofModusPonensTraceTags().ShouldTrace("mk_modus_ponens")
}

func (m *ASTManager) traceProofModusPonensBadFact(p, q ASTHandle) error {
	return m.proofModusPonensTraceTags().EmitConditional(
		"mk_modus_ponens", "mk_modus_ponens", "src/ast/ast.cpp", 2641,
		func() bool { return true }, func(w io.Writer) error {
			for _, proof := range [...]ASTHandle{p, q} {
				out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.BudgetField})
				if err != nil {
					return err
				}
				printErr := m.lowLevelPrint(out, astNodeRef{slot: proof.slot, generation: proof.generation}, true, true)
				body, readErr := out.String()
				_, writeErr := io.WriteString(w, body+"\n")
				closeErr := out.Close()
				if err = errors.Join(printErr, readErr, writeErr, closeErr); err != nil {
					return err
				}
			}
			return nil
		})
}
