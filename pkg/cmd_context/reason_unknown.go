// Source: src/cmd_context/cmd_context.cpp:775-778;
// src/cmd_context/basic_cmds.cpp get_info_cmd :reason-unknown branch.
package cmd_context

import (
	"io"

	"github.com/glycerine/g3/pkg/util/escaped"
)

func (c *Context) ReasonUnknown() (string, error) {
	if err := c.valid(); err != nil {
		return "", err
	}
	if !c.checked {
		return "state of the most recent check-sat command is not known", nil
	}
	return c.checkSatResult().ReasonUnknown()
}
func (c *Context) WriteReasonUnknown(out io.Writer) error {
	reason, err := c.ReasonUnknown()
	if err != nil {
		return err
	}
	if out == nil {
		return io.ErrClosedPipe
	}
	if _, err = io.WriteString(out, "(:reason-unknown \""); err != nil {
		return err
	}
	if err = escaped.Write(out, &reason, false, 0); err != nil {
		return err
	}
	_, err = io.WriteString(out, "\")\n")
	return err
}
