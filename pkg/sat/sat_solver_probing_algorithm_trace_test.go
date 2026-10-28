package sat

// Source: src/sat/sat_probing.cpp:111.

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSolverProbingProcessCoreSourceTrace(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, lifecycle := satProbingFixture(t, 1)
		lifecycle.propagation.probingCounter = -5
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("probing")
		}
		lifecycle.traceTags = tags
		satOK(t, s.processCore(0))
		want := ""
		if tags.ShouldTrace("probing") {
			want = "-------- [probing] process_core src/sat/sat_probing.cpp:111 ---------\n" +
				"processing: 0 counter: 5\n" +
				"------------------------------------------------\n"
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
		if lifecycle.propagation.probingCounter >= -5 || lifecycle.propagation.variables.scopeLevel != 0 {
			t.Fatalf("source process did not complete: counter=%d scope=%d", lifecycle.propagation.probingCounter, lifecycle.propagation.variables.scopeLevel)
		}
	}
}
