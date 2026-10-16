// Source: src/sat/sat_integrity_checker.cpp:56-60; src/util/trace.h:40-46.
package sat

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (c satIntegrityChecker) eliminatedClauseText(cl StoredClauseRef, variable uint32) (body string, result error) {
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.lifecycle.budget})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	if err = out.AppendCString("l: "); err != nil {
		return "", err
	}
	if err = out.AppendUint32(variable); err != nil {
		return "", err
	}
	if err = out.AppendCString("\nc: "); err != nil {
		return "", err
	}
	if err = c.lifecycle.propagation.clauses.DisplayInto(cl, out); err != nil {
		return "", err
	}
	if err = out.AppendCString("\n"); err != nil {
		return "", err
	}
	if err = c.gc.conflict.solver.displayInto(out); err != nil {
		return "", err
	}
	return out.String()
}

func (c satIntegrityChecker) traceEliminatedClause(cl StoredClauseRef, variable uint32) {
	tags := c.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat_bug", "check_clause", "src/sat/sat_integrity_checker.cpp", 56,
		func() bool { return true }, func(w io.Writer) error {
			body, err := c.eliminatedClauseText(cl, variable)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, body)
			return err
		})
}
