package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatFloorCeilRetainedShortTraceBranches(t *testing.T) {
	m := fastManager(t, 2)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		numerator   int32
		denominator uint32
		ceil        bool
		wantValue   string
		wantTrace   string
	}{
		{"floor-fraction", 3, 2, false, "1", "[mpff] Floor[3/2] == 1\n"},
		{"ceil-fraction", 3, 2, true, "2", "[mpff] Ceiling[3/2] == 2\n"},
		{"floor-negative", -3, 2, false, "-2", "[mpff] Floor[-3/2] == -2\n"},
		{"ceil-negative", -3, 2, true, "-1", "[mpff] Ceiling[-3/2] == -1\n"},
		{"floor-tiny", 1, 2, false, "0", "[mpff] Floor[1/2] == 0\n"},
		{"ceil-tiny", 1, 2, true, "1", "[mpff] Ceiling[1/2] == 1\n"},
		{"floor-tiny-negative", -1, 2, false, "-1", "[mpff] Floor[-1/2] == -1\n"},
		{"ceil-tiny-negative", -1, 2, true, "0", "[mpff] Ceiling[-1/2] == 0\n"},
		{"floor-integer", 2, 1, false, "2", "[mpff] Floor[2] == 2\n"},
		{"ceil-integer", 2, 1, true, "2", "[mpff] Ceiling[2] == 2\n"},
	} {
		var value FastFloat
		if err := m.SetFraction32(&value, test.numerator, test.denominator); err != nil {
			t.Fatal(test.name, err)
		}
		output.Reset()
		var err error
		if test.ceil {
			err = m.CeilInPlace(&value)
		} else {
			err = m.FloorInPlace(&value)
		}
		if err != nil {
			t.Fatal(test.name, err)
		}
		if got, err := m.String(&value); err != nil || got != test.wantValue {
			t.Fatal("rounded value", test.name, got, err)
		}
		want := test.wantTrace
		if !compiled {
			want = ""
		}
		if output.String() != want {
			t.Fatal("source floor/ceil trace", test.name, output.String())
		}
		if err := m.Del(&value); err != nil {
			t.Fatal(err)
		}
	}
	var whole FastFloat
	if err := m.SetUint64(&whole, 1<<63); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := m.FloorInPlace(&whole); err != nil {
		t.Fatal(err)
	}
	if err := m.CeilInPlace(&whole); err != nil || output.Len() != 0 {
		t.Fatal("nonnegative-exponent early return", err, output.String())
	}
	if err := m.Del(&whole); err != nil {
		t.Fatal(err)
	}
}
