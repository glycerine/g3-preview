package numeral

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNToStringSourceEntryExitAndShortDestinationTrace(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn_to_string")
	compiled := tags.ShouldTrace("mpn_to_string")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	record := func(line int, body string) string {
		return fmt.Sprintf("-------- [mpn_to_string] to_string src/util/mpn.cpp:%d ---------\n", line) +
			body + "------------------------------------------------\n"
	}
	var want string
	for _, tc := range []struct {
		limbs  []uint32
		limit  uint64
		result string
		entry  string
	}{
		{[]uint32{42}, 16, "42", "[42]"},
		{[]uint32{123}, 2, "1", "[123]"},
		{[]uint32{0, 1}, 32, "4294967296", "[1|0]"},
	} {
		var dst [32]byte
		if err := m.Natural.sourceToCString(limbView{external: tc.limbs, size: uint32(len(tc.limbs))},
			sourceDecimalDestination{external: dst[:]}, tc.limit); err != nil {
			t.Fatal(tc, err)
		}
		if got := string(dst[:len(tc.result)]); got != tc.result {
			t.Fatal("source decimal text", got, tc.result)
		}
		want += record(362, "[mpn] to_string "+tc.entry+" == ") + record(395, tc.result+"\n")
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased to_string TRACE emitted", output.String())
		}
	} else if got := output.String(); got != want {
		t.Fatalf("source to_string TRACE got %q, want %q", got, want)
	}
}
