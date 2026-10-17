// Source: src/ast/normal_forms/pull_quant.cpp:109,131.
package pattern

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (p *patternQuantifierPull) tracePulledChild(quantifier bool, child, source, adjusted ast.ASTHandle, count, bound, shift1, shift2 uint32) error {
	tags := p.owner.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("pull_quant") {
		return nil
	}
	line := 109
	if quantifier {
		line = 131
	}
	return tags.EmitConditional("pull_quant", "pull_quant1_core", "src/ast/normal_forms/pull_quant.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			var prefix string
			if quantifier {
				// The source diagnostic labels print the two shift arguments in
				// reverse order from the actual call.
				prefix = "shifted  bound: " + strconv.FormatUint(uint64(bound), 10) +
					" shift1: " + strconv.FormatUint(uint64(shift2), 10) +
					" shift2: " + strconv.FormatUint(uint64(shift1), 10) + "\n"
			} else {
				prefix = "shifted by: " + strconv.FormatUint(uint64(count), 10) + "\n"
				source = child
			}
			if _, err := io.WriteString(w, prefix); err != nil {
				return err
			}
			if err := p.owner.writePatternSMT2(w, source); err != nil {
				return err
			}
			if _, err := io.WriteString(w, "\n---->\n"); err != nil {
				return err
			}
			if err := p.owner.writePatternSMT2(w, adjusted); err != nil {
				return err
			}
			_, err := io.WriteString(w, "\n")
			return err
		})
}
