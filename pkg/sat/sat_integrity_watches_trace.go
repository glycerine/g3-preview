// Source: src/sat/sat_integrity_checker.cpp:172-181; src/util/trace.h:40-46.
package sat

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (c satIntegrityChecker) watchListText(list satSolverWatchListView) (body string, result error) {
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.lifecycle.budget})
	if err != nil {
		return "", err
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	ref, err := list.ref()
	if err != nil {
		return "", err
	}
	if err = list.owner.store.DisplayInto(ref, c.lifecycle.propagation.clauses, out); err != nil {
		return "", err
	}
	return out.String()
}

func (c satIntegrityChecker) traceMissingBinaryWatch(literal, other Literal, learned, ownGone, otherGone bool, current, reciprocal satSolverWatchListView) {
	tags := c.lifecycle.traceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("sat_watched_bug") {
		return
	}
	_ = tags.EmitConditional("sat_watched_bug", "check_watches", "src/sat/sat_integrity_checker.cpp", 172,
		func() bool { return true }, func(w io.Writer) error {
			currentText, err := c.watchListText(current)
			if err != nil {
				return err
			}
			reciprocalText, err := c.watchListText(reciprocal)
			if err != nil {
				return err
			}
			boolDigit := func(v bool) string {
				if v {
					return "1"
				}
				return "0"
			}
			_, _ = io.WriteString(w, "l: "+literal.String()+" l2: "+other.String()+"\n")
			_, _ = io.WriteString(w, "was_eliminated1: "+boolDigit(ownGone)+" was_eliminated2: "+boolDigit(otherGone)+" learned: "+boolDigit(learned)+"\n")
			_, _ = io.WriteString(w, currentText+"\n"+reciprocalText+"\n")
			return nil
		})
}
