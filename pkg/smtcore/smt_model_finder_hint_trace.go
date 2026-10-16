// Source: src/smt/smt_model_finder.cpp:2925-2929.
package smtcore

import (
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (s *hintSearchState) traceResidueCandidates(m *ast.ASTManager, candidates map[ast.ASTHandle]struct{}) {
	tags := s.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("model_finder_hint", "greedy", "src/smt/smt_model_finder.cpp", 2925,
		func() bool { return true }, func(w io.Writer) error {
			if _, err := io.WriteString(w, "candidates from residue:\n"); err != nil {
				return err
			}
			for f := range candidates {
				name, err := m.GetASTName(f)
				if err != nil {
					return err
				}
				if err := m.Symbols.WriteDisplay(w, name); err != nil {
					return err
				}
				if _, err := io.WriteString(w, " "); err != nil {
					return err
				}
			}
			_, err := io.WriteString(w, "\n")
			return err
		})
}
