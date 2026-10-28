//go:build g3debug && g3trace

package sat

// Source: src/sat/sat_elim_eqs.cpp:205-207.

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATElimEqsEliminatedLiteralSourceTraceBeforeAssertion(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		l, p, _, _ := satLifecycleFixture(t, 2)
		e, err := newSATElimEqs(l)
		satOK(t, err)
		t.Cleanup(func() { satOK(t, e.Close()) })
		c, err := p.clauses.MkClause([]Literal{0, 2}, false)
		satOK(t, err)
		satOK(t, l.pushClause(false, c))
		satOK(t, p.variables.setEliminated(1, true))
		e.borrowedRoots = []Literal{0, 0}
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("elim_eqs_bug")
		}
		l.traceTags = tags
		if err = e.checkClauses(); !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("source assertion: %v", err)
		}
		want := ""
		if enabled {
			want = satElimEqsTraceRecord("elim_eqs_bug", "check_clauses", "205", "lit: 1 0\n(0 1)+\n")
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
	}
}
