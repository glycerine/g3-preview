// Source: src/math/subpaving/subpaving_t_def.h:668-679,762.
package subpaving

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// polynomialInteger is the concrete config_mpq is_int(polynomial) body.
// TRACE calls it once for display, then mk_var calls it again.
func (c *RationalContext) polynomialInteger(p *RationalPolynomial) (bool, error) {
	for i := range p.terms {
		term := &p.terms[i]
		if !c.variables[term.variable] {
			return false, nil
		}
		ok, err := c.numbers.IsInt(&term.coefficient)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return c.numbers.IsInt(&p.constant)
}

// BindTrace borrows this concrete context's diagnostic stream. Nil restores
// the source process-global stream.
func (c *RationalContext) BindTrace(tags *diagnostic.TraceTags) error {
	if err := c.open(); err != nil {
		return err
	}
	c.trace = tags
	return nil
}

func (c *RationalContext) tracePolynomialInteger(p *RationalPolynomial) error {
	tags := c.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("subpaving_mk_sum") {
		return nil
	}
	var bodyErr error
	_ = tags.EmitConditional("subpaving_mk_sum", "mk_sum", "src/math/subpaving/subpaving_t_def.h", 762,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "new variable is integer: ")
			integer, err := c.polynomialInteger(p)
			if err != nil {
				bodyErr = err
				return err
			}
			if integer {
				_, _ = io.WriteString(w, "1")
			} else {
				_, _ = io.WriteString(w, "0")
			}
			_, _ = io.WriteString(w, "\n")
			return nil
		})
	return bodyErr
}
