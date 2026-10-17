// Source: src/ast/rewriter/fpa_rewriter.cpp:159; src/util/trace.h:40-46.
package ast

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func (m *ASTManager) traceFPAToFPArithmeticInput(r *unumeral.OwnedRational) error {
	tags := m.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("fp_rewriter") {
		return nil
	}
	err := tags.EmitConditional("fp_rewriter", "mk_to_fp", "src/ast/rewriter/fpa_rewriter.cpp", 159,
		func() bool { return true }, func(w io.Writer) error {
			if _, e := io.WriteString(w, "r: "); e != nil {
				return e
			}
			value, e := m.numerals.String(r)
			if e != nil {
				return e
			}
			_, e = io.WriteString(w, value+"\n")
			return e
		})
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}
