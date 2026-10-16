// Source: src/sat/sat_simplifier.cpp:223.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATActualSimplifierFinalizerSolverAndConverterSourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	satOK(t, s.Variables.setEliminated(0, true))
	entry, err := s.converter.Mk(satModelElimVar, 0)
	satOK(t, err)
	satOK(t, s.converter.InsertLiterals(entry, []Literal{0}))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("after_simplifier")
	s.Lifecycle.traceTags = tags
	satOK(t, s.simplifier.Run(false))
	if !tags.ShouldTrace("after_simplifier") {
		if output.Len() != 0 {
			t.Fatal("build-erased source finalizer trace", output.String())
		}
		return
	}
	var solver, converter bytes.Buffer
	satOK(t, s.traceUserDisplayInto(&solver))
	satOK(t, s.traceModelConverterInto(&converter))
	if converter.String() != "(sat::model-converter\n  (elim 0\n    (0)))\n" {
		t.Fatal("expected concrete nonempty converter display", converter.String())
	}
	want := "-------- [after_simplifier] scoped_finalize_fn src/sat/sat_simplifier.cpp:216 ---------\n" +
		"skipping cleanup...\n------------------------------------------------\n" +
		"-------- [after_simplifier] scoped_finalize_fn src/sat/sat_simplifier.cpp:223 ---------\n" +
		solver.String() + "model_converter:\n" + converter.String() + "------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source finalizer combined display: got %q, want %q", got, want)
	}
}
