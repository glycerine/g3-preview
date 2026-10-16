// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:510.
package arith

import (
	"io"
	"strings"

	goaltactic "github.com/glycerine/g3/pkg/tactic"
)

// The final TRACE reads the restored goal after all bound assertions. The
// process printer is reached only when the tag and compile gates are open.
func (b *PolynomialBuilder) traceResultGoal(result *goaltactic.Goal) error {
	tags := b.store.boundTraceTags()
	if !tags.ShouldTrace("propagate_ineqs_tactic") {
		return nil
	}
	var body strings.Builder
	if err := result.Display(&body); err != nil {
		return err
	}
	_ = tags.EmitConditional("propagate_ineqs_tactic", "operator()", "src/tactic/arith/propagate_ineqs_tactic.cpp", 510,
		func() bool { return true }, func(w io.Writer) error {
			_, err := io.WriteString(w, body.String())
			return err
		})
	return nil
}
