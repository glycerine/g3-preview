package sat

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satCleanerTraceRecord(tag string, line int, body string) string {
	return "-------- [" + tag + "] cleanup_clauses src/sat/sat_cleaner.cpp:" + strconv.Itoa(line) + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATCleanerClauseSourceTraceStages(t *testing.T) {
	cases := []struct {
		name        string
		assignments []Literal
		learned     bool
		frozen      bool
		want        string
	}{
		{
			name:        "binary",
			assignments: []Literal{1},
			learned:     true,
			frozen:      true,
			want: satCleanerTraceRecord("sat_cleaner_bug", 86, "cleaning: (0 1 2)+*\n0: l_false\n1: l_undef\n2: l_undef\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 88, "(0 1 2)+*\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 108, "sat: 0, new_size: 2\n1 2\n") +
				satCleanerTraceRecord("cleanup_bug", 137, "clause became binary: 1 2\n"),
		},
		{
			name:        "unit",
			assignments: []Literal{1, 3},
			learned:     true,
			frozen:      true,
			want: satCleanerTraceRecord("sat_cleaner_bug", 86, "cleaning: (0 1 2)+*\n0: l_false\n1: l_false\n2: l_undef\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 88, "(0 1 2)+*\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 108, "sat: 0, new_size: 1\n2\n") +
				satCleanerTraceRecord("sat_cleaner_bug", 117, "new_sz: 1\nunit: 2\n"),
		},
		{
			name:        "satisfied",
			assignments: []Literal{0},
			learned:     true,
			frozen:      true,
			want: satCleanerTraceRecord("sat_cleaner_bug", 86, "cleaning: (0 1 2)+*\n0: l_true\n1: l_undef\n2: l_undef\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 88, "(0 1 2)+*\n") +
				satCleanerTraceRecord("sat_cleaner_frozen", 108, "sat: 1, new_size: 0\n\n"),
		},
		{
			name:        "active-binary",
			assignments: []Literal{1},
			want: satCleanerTraceRecord("sat_cleaner_bug", 86, "cleaning: (0 1 2)+\n0: l_false\n1: l_undef\n2: l_undef\n") +
				satCleanerTraceRecord("cleanup_bug", 137, "clause became binary: 1 2\n"),
		},
	}
	for _, tc := range cases {
		for _, traced := range []bool{false, true} {
			name := tc.name + "-disabled"
			if traced {
				name = tc.name + "-traced"
			}
			t.Run(name, func(t *testing.T) {
				c, l := satCleanerFixture(t, 3)
				satCleanerClause(t, l, []Literal{0, 2, 4}, tc.learned, tc.frozen)
				for _, lit := range tc.assignments {
					satOK(t, l.propagation.assign(lit, satJustification{}))
				}
				defer func() { satOK(t, l.unassignVars(0)) }()
				var output bytes.Buffer
				tags := diagnostic.NewTraceTagsWithWriter(&output)
				l.traceTags = tags
				if traced {
					tags.Enable("sat_cleaner_bug")
					tags.Enable("sat_cleaner_frozen")
					tags.Enable("cleanup_bug")
				}
				satOK(t, c.cleanupClauses(tc.learned))
				want := ""
				if tags.ShouldTrace("sat_cleaner_bug") {
					want = tc.want
				}
				if got := output.String(); got != want {
					t.Fatalf("source clause trace: got %q, want %q", got, want)
				}
			})
		}
	}
}
