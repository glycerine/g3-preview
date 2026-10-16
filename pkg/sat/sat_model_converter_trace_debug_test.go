//go:build g3debug && g3trace

// Source: src/sat/sat_model_converter.cpp:181-183.
package sat

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSATModelConverterInvariantSourceTraceBeforeAssertion(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		m, b := satConverterTestOwner(t)
		_, err := m.Mk(satModelElimVar, 0)
		satOK(t, err)
		later, err := m.Mk(satModelBlockLit, 1)
		satOK(t, err)
		satOK(t, m.InsertBinary(later, 0, 2))
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("sat_model_converter")
		}
		m.traceTags = tags
		_, err = m.CheckInvariant(2)
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("source invariant assertion: %v", err)
		}
		want := ""
		if enabled {
			stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
			satOK(t, err)
			satOK(t, m.DisplayInto(stream))
			body, err := stream.String()
			satOK(t, err)
			satOK(t, stream.Close())
			want = satModelConverterTraceRecord("sat_model_converter", "check_invariant", "181", "var: 0\n"+body)
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
	}
}
