// Source: src/math/subpaving/subpaving_t_def.h:243-246,254-269,636-650;
// concrete config_mpfx child-bound display.
package subpaving

import "io"

func (c *FixedContext) displayBound(w io.Writer, b *FixedBound) error {
	if b.lower {
		value, err := c.numbers.ToRationalString(b.value)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(w, value)
		_, _ = io.WriteString(w, " <")
		if !b.open {
			_, _ = io.WriteString(w, "=")
		}
		_, _ = io.WriteString(w, " ")
		return c.writeVariable(w, b.variable)
	}
	if err := c.writeVariable(w, b.variable); err != nil {
		return err
	}
	_, _ = io.WriteString(w, " <")
	if !b.open {
		_, _ = io.WriteString(w, "=")
	}
	_, _ = io.WriteString(w, " ")
	value, err := c.numbers.ToRationalString(b.value)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, value)
	return nil
}

func (c *FixedContext) displayNodeBounds(w io.Writer, n *FixedNode) error {
	for variable := range c.isInt {
		lower, upper := n.lower[variable], n.upper[variable]
		if lower != nil {
			if err := c.displayBound(w, lower); err != nil {
				return err
			}
			_, _ = io.WriteString(w, " ")
		}
		if upper != nil {
			if err := c.displayBound(w, upper); err != nil {
				return err
			}
		}
		if lower != nil || upper != nil {
			_, _ = io.WriteString(w, "\n")
		}
	}
	return nil
}

func (c *FixedContext) traceSplitBounds(left, right *FixedNode) error {
	tags := c.polynomialTraceTags()
	if !tags.ShouldTrace("subpaving_int_split") {
		return nil
	}
	var bodyErr error
	_ = tags.EmitConditional("subpaving_int_split", "operator()", "src/math/subpaving/subpaving_t_def.h", 243,
		func() bool { return true }, func(w io.Writer) error {
			out := splitTraceStreamWriter{out: w}
			_, _ = io.WriteString(out, "LEFT:\n")
			if err := c.displayNodeBounds(out, left); err != nil {
				bodyErr = err
				return err
			}
			_, _ = io.WriteString(out, "\nRIGHT:\n")
			if err := c.displayNodeBounds(out, right); err != nil {
				bodyErr = err
				return err
			}
			return nil
		})
	return bodyErr
}
