// Source: src/sat/sat_model_converter.cpp:108,142,152,163.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func satModelConverterTraceRecord(tag, function, line, body string) string {
	return "-------- [" + tag + "] " + function + " src/sat/sat_model_converter.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATModelConverterInsertSourceTraces(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		m, _ := satConverterTestOwner(t)
		a := satStore(t, nil)
		c := satMk(t, a, []Literal{0, 3, 4}, false)
		r, err := m.Mk(satModelBlockLit, 0)
		satOK(t, err)
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("sat_mc_bug")
		}
		m.traceTags = tags
		satOK(t, m.InsertClause(r, a, c))
		satOK(t, m.InsertBinary(r, 0, 4))
		satOK(t, m.InsertWrapper(r, a, newSATClauseWrapper(c)))
		want := ""
		if tags.ShouldTrace("sat_mc_bug") {
			want = satModelConverterTraceRecord("sat_mc_bug", "insert", "142", "adding: (0 -1 2)+\n") +
				satModelConverterTraceRecord("sat_mc_bug", "insert", "152", "adding (binary): 0 2\n") +
				satModelConverterTraceRecord("sat_mc_bug", "insert", "163", "adding (wrapper): 0 -1 2 \n")
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
		satOK(t, a.Delete(c))
		if len(m.entries[0].literals) != 11 {
			t.Fatal("three source insertion forms lost their stored clauses")
		}
	}
}

func TestSATModelConverterFailedClauseSourceTrace(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		m, _ := satConverterTestOwner(t)
		r, err := m.Mk(satModelBlockLit, 0)
		satOK(t, err)
		satOK(t, m.InsertBinary(r, 0, 2))
		satOK(t, m.InsertBinary(r, 0, 4))
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("sat_model_bug")
		}
		m.traceTags = tags
		ok, err := m.CheckModel(Model{lbool.False, lbool.False, lbool.False})
		satOK(t, err)
		if ok {
			t.Fatal("false model satisfied eliminated clauses")
		}
		want := ""
		if tags.ShouldTrace("sat_model_bug") {
			want = satModelConverterTraceRecord("sat_model_bug", "check_model", "108", "failed eliminated: 0 1\n") +
				satModelConverterTraceRecord("sat_model_bug", "check_model", "108", "failed eliminated: 0 2\n")
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
	}
}
