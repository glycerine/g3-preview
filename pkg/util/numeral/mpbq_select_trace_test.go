package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestBinarySelectSmallRationalLowerRetainedTrace(t *testing.T) {
	m, _ := binaryManager(t)
	qm := m.Diagnostics.Manager
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("select_small")
	compiled := tags.ShouldTrace("select_small")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	upper := NewBinaryIntPower(3, 2)
	defer m.Del(&upper)
	for _, tc := range []struct {
		name  string
		num   int32
		den   uint32
		valid bool
		body  string
	}{
		{"inside", 1, 3, true, "lower (q): 1/3, upper (bq): 3/2^2\n"},
		{"reversed", 1, 1, false, "lower (q): 1, upper (bq): 3/2^2\n"},
	} {
		lower := NewRational()
		if err := qm.SetFractionSigned(&lower, tc.num, int32(tc.den)); err != nil {
			t.Fatal(tc.name, err)
		}
		var result BinaryRational
		output.Reset()
		err := m.SelectSmallRationalLower(qm, &lower, &upper, &result)
		if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrBinaryInterval) {
			t.Fatal(tc.name, err)
		}
		if tc.valid {
			binaryText(t, m, &result, "1/2")
		}
		want := ""
		if compiled {
			want = "-------- [select_small] select_small_core src/util/mpbq.cpp:752 ---------\n" + tc.body +
				"------------------------------------------------\n"
		}
		if got := output.String(); got != want {
			t.Fatalf("%s: trace got %q, want %q", tc.name, got, want)
		}
		if err := m.Del(&result); err != nil {
			t.Fatal(err)
		}
		if err := qm.Del(&lower); err != nil {
			t.Fatal(err)
		}
	}
}
