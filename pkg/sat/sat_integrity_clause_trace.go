// Source: src/sat/sat_integrity_checker.cpp:56-93; src/util/trace.h:40-46.
package sat

import (
	"errors"
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (c satIntegrityChecker) eliminatedClauseDisplayText(cl StoredClauseRef) (body string, result error) {
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.lifecycle.budget})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	if err = c.lifecycle.propagation.clauses.DisplayInto(cl, out); err != nil {
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
			_, _ = io.WriteString(w, "l: ")
			_, _ = io.WriteString(w, strconv.FormatUint(uint64(variable), 10))
			_, _ = io.WriteString(w, "\n")
			_, _ = io.WriteString(w, "c: ")
			body, err := c.eliminatedClauseDisplayText(cl)
			if err != nil {
				return err
			}
			_, _ = io.WriteString(w, body)
			_, _ = io.WriteString(w, "\n")
			body, err = c.eliminatedWatchDisplayText(false)
			if err != nil {
				return err
			}
			_, _ = io.WriteString(w, body)
			return nil
		})
}

func (c satIntegrityChecker) traceMissingTernaryWatch(cl StoredClauseRef, index uint32) {
	tags := c.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("sat_ter_watch_bug") {
		return
	}
	_ = tags.EmitConditional("sat_ter_watch_bug", "check_clause", "src/sat/sat_integrity_checker.cpp", 69,
		func() bool { return true }, func(w io.Writer) error {
			body, err := c.eliminatedClauseDisplayText(cl)
			if err != nil {
				return err
			}
			_, _ = io.WriteString(w, body)
			_, _ = io.WriteString(w, "\n")
			_, _ = io.WriteString(w, "watch_list:\n")
			list, err := c.lifecycle.propagation.variables.watches.Get(index)
			if err != nil {
				return err
			}
			body, err = c.watchListText(list)
			if err != nil {
				return err
			}
			_, _ = io.WriteString(w, body)
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}

func (c satIntegrityChecker) residualClauseText(cl StoredClauseRef) (body string, result error) {
	p := c.lifecycle.propagation
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.lifecycle.budget})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	if err = p.clauses.DisplayInto(cl, out); err != nil {
		return "", err
	}
	if err = out.AppendCString(" status: "); err != nil {
		return "", err
	}
	status, err := p.status(cl)
	if err != nil {
		return "", err
	}
	if err = out.AppendCString(status.String() + "\n"); err != nil {
		return "", err
	}
	for i := uint32(0); ; i++ {
		n, e := p.clauses.Size(cl)
		if e != nil {
			return "", e
		}
		if i >= n {
			break
		}
		if e = out.AppendCString("val("); e != nil {
			return "", e
		}
		if e = out.AppendUint32(i); e != nil {
			return "", e
		}
		if e = out.AppendCString("): "); e != nil {
			return "", e
		}
		lit, e := p.clauses.Literal(cl, i)
		if e != nil {
			return "", e
		}
		value, e := p.variables.valueLiteral(lit)
		if e != nil {
			return "", e
		}
		if e = out.AppendCString(value.String() + "\n"); e != nil {
			return "", e
		}
	}
	return out.String()
}

func (c satIntegrityChecker) traceResidualClause(cl StoredClauseRef) {
	tags := c.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("sat_bug", "check_clause", "src/sat/sat_integrity_checker.cpp", 90,
		func() bool { return true }, func(w io.Writer) error {
			body, err := c.residualClauseText(cl)
			if err != nil {
				return err
			}
			_, err = io.WriteString(w, body)
			return err
		})
}
