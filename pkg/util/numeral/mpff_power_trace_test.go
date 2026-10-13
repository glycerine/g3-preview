package numeral

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatRawPowerTraceExponentInsertionBoundary(t *testing.T) {
	m := fastManager(t, 2)
	var value FastFloat
	if err := m.SetInt32(&value, -3); err != nil {
		t.Fatal(err)
	}
	if err := m.SetExponent(&value, -7); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Del(&value); err != nil {
			t.Fatal(err)
		}
	})
	full := &fastDisplayWrites{}
	if err := m.displayRawTraceTo(full, &value); err != nil {
		t.Fatal(err)
	}
	if len(full.parts) != 5 || full.parts[0] != "-" || full.parts[3] != "*2^" || full.parts[4] != "-7" {
		t.Fatal("raw display insertion order", full.parts)
	}
	for at := range full.parts {
		refusal := errors.New("refused raw insertion")
		w := &fastDisplayWrites{at: at + 1, refusal: refusal}
		err := m.displayRawTraceTo(w, &value)
		if !errors.Is(err, refusal) || w.String() != strings.Join(full.parts[:at], "") || len(w.parts) != at+1 {
			t.Fatal("raw display refusal prefix", at, err, w.String(), w.parts)
		}
	}
}

func TestFastFloatPowerRetainedTraceBranchesAndAlias(t *testing.T) {
	m := fastManager(t, 2)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	tags.Enable("mpff_power")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		base  int32
		power uint32
		alias bool
		up    bool
		want  string
	}{
		{"zero", 0, 3, false, true, "0"},
		{"identity", 3, 0, false, true, "1"},
		{"identity-alias", 3, 0, true, true, "1"},
		{"copy", -2, 1, false, false, "-2"},
		{"copy-alias", -2, 1, true, false, "-2"},
		{"square", -2, 2, false, true, "4"},
		{"small", 2, 3, false, true, "8"},
		{"small-alias", -2, 3, true, false, "-8"},
		{"minimum-significand", 2, 9, false, true, "512"},
		{"general", 3, 9, false, false, "19683"},
	} {
		var a, b FastFloat
		if err := m.SetInt32(&a, tc.base); err != nil {
			t.Fatal(tc.name, err)
		}
		if err := m.SetRounding(tc.up); err != nil {
			t.Fatal(tc.name, err)
		}
		out := &b
		if tc.alias {
			out = &a
		}
		output.Reset()
		if err := m.Power(&a, tc.power, out); err != nil {
			t.Fatal(tc.name, err)
		}
		if got, err := m.String(out); err != nil || got != tc.want {
			t.Fatal(tc.name, got, err)
		}
		raw, err := m.RawString(out)
		if err != nil {
			t.Fatal(tc.name, err)
		}
		want := ""
		if compiled {
			bound := ">="
			if tc.up {
				bound = "<="
			}
			want = "[mpff] (" + strconv.FormatInt(int64(tc.base), 10) + ") ^ " + strconv.FormatUint(uint64(tc.power), 10) + bound + tc.want + "\n" +
				"-------- [mpff_power] power src/util/mpff.cpp:1053 ---------\n" + raw + "\n" +
				"------------------------------------------------\n"
		}
		got := output.String()
		if compiled && (tc.power == 2 || tc.power == 3 && !tc.alias && tc.base != 0 || tc.base == 3 && tc.power == 9) {
			if !strings.HasSuffix(got, want) || !strings.Contains(strings.TrimSuffix(got, want), "[mpff] (") {
				t.Fatalf("%s: nested trace and power suffix got %q, want suffix %q", tc.name, got, want)
			}
		} else if got != want {
			t.Fatalf("%s: trace got %q, want %q", tc.name, output.String(), want)
		}
		if err := m.Del(&a); err != nil {
			t.Fatal(err)
		}
		if err := m.Del(&b); err != nil {
			t.Fatal(err)
		}
	}
}
