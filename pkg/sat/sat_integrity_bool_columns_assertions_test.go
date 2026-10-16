// Source: src/sat/sat_integrity_checker.cpp:133-145; src/sat/sat_solver.h:246.
package sat

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityBoolColumnAssertionsUseSourceActions(t *testing.T) {
	cases := []struct {
		line      int
		condition string
	}{
		{133, "s.m_watches.size() == s.num_vars() * 2"},
		{134, "s.m_assignment.size() == s.num_vars() * 2"},
		{135, "s.m_lit_mark.size() == s.num_vars() * 2"},
		{136, "s.m_justification.size() == s.num_vars()"},
		{137, "s.m_decision.size() == s.num_vars()"},
		{138, "s.m_eliminated.size() == s.num_vars()"},
		{139, "s.m_external.size() == s.num_vars()"},
		{141, "s.m_mark.size() == s.num_vars()"},
		{142, "s.m_activity.size() == s.num_vars()"},
		{143, "s.m_phase.size() == s.num_vars()"},
		{144, "s.m_prev_phase.size() == s.num_vars()"},
		{145, "s.m_assigned_since_gc.size() == s.num_vars()"},
	}
	for _, tc := range cases {
		t.Run(tc.condition, func(t *testing.T) {
			g, _ := satGCFixture(t, 2)
			l := g.lifecycle()
			v := l.propagation.variables
			switch tc.line {
			case 133:
				prior := v.watches.lists
				v.watches.lists = nil
				defer func() { v.watches.lists = prior }()
			case 134:
				prior := v.assignment
				v.assignment = nil
				defer func() { v.assignment = prior }()
			case 135:
				prior := v.literalMark
				v.literalMark = nil
				defer func() { v.literalMark = prior }()
			case 136:
				prior := v.justification.values
				v.justification.values = nil
				defer func() { v.justification.values = prior }()
			case 137:
				prior := v.decision
				v.decision = nil
				defer func() { v.decision = prior }()
			case 138:
				prior := v.eliminated
				v.eliminated = nil
				defer func() { v.eliminated = prior }()
			case 139:
				prior := v.external
				v.external = nil
				defer func() { v.external = prior }()
			case 141:
				prior := v.mark
				v.mark = nil
				defer func() { v.mark = prior }()
			case 142:
				prior := v.activity.values
				v.activity.values = nil
				defer func() { v.activity.values = prior }()
			case 143:
				prior := v.phase
				v.phase = nil
				defer func() { v.phase = prior }()
			case 144:
				prior := v.previousPhase
				v.previousPhase = nil
				defer func() { v.previousPhase = prior }()
			case 145:
				prior := v.assignedSinceGC
				v.assignedSinceGC = nil
				defer func() { v.assignedSinceGC = prior }()
			}
			debug, input, outputFile := satIntegrityActionFixture(t, "T")
			satOK(t, debug.EnableAssertions(true))
			checker, err := newSATIntegrityChecker(l, g)
			satOK(t, err)
			checker.debugActions = true
			yes, err := checker.checkBoolVars()
			if !satStorageSourceDebug {
				if tc.line == 138 {
					// The non-debug source still reads eliminated[v] in the
					// loop after erasing the length assertion.
					if yes || !errors.Is(err, arena.ErrBounds) {
						t.Fatalf("release eliminated read guard: result=%v error=%v", yes, err)
					}
					return
				}
				if !yes || err != nil {
					t.Fatalf("release checker reached assertion: result=%v error=%v", yes, err)
				}
				return
			}
			var ex z3exception.DefaultException
			if yes || !errors.As(err, &ex) {
				t.Fatalf("source assertion action: result=%v error=%v", yes, err)
			}
			state, err := input.State()
			satOK(t, err)
			if state.Position != 1 {
				t.Fatalf("debugger action count: %d", state.Position)
			}
			got, err := os.ReadFile(outputFile.Name())
			satOK(t, err)
			want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: " + strconv.Itoa(tc.line) + "\n" + tc.condition + "\n" +
				"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
			if string(got) != want {
				t.Fatalf("source bool-column assertion: got %q, want %q", got, want)
			}
		})
	}
}
