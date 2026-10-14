package numeral

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
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

func TestMPNToStringSourceEndlFlushAndRefusedStream(t *testing.T) {
	for _, refused := range []bool{false, true} {
		m, _ := integerManager(t)
		w := &mpfEndlWriter{}
		if refused {
			w.failAt, w.refusal = 2, errors.New("result endl flush refused")
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_to_string")
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		var dst [32]byte
		if err := m.Natural.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{external: dst[:]}, uint64(len(dst))); err != nil {
			t.Fatal(err)
		}
		if got := string(dst[:10]); got != "4294967296" {
			t.Fatal("trace stream changed decimal result", got)
		}
		if !tags.ShouldTrace("mpn_to_string") {
			if len(w.events) != 0 {
				t.Fatal("erased trace wrote", w.events)
			}
			continue
		}
		for _, sequence := range [][]string{
			{"write:[mpn] to_string ", "write:[", "write:1", "write:|", "write:0", "write:]", "write: == "},
			{"write:4294967296", "write:\n", "flush"},
		} {
			joined := strings.Join(w.events, "|")
			if !strings.Contains(joined, strings.Join(sequence, "|")) {
				t.Fatalf("source insertion/endl sequence %q missing from %q", sequence, w.events)
			}
		}
		if w.flushes != 3 {
			t.Fatal("input TRACE footer, result endl and result footer", w.flushes, w.events)
		}
		if refused {
			flushes := 0
			for _, event := range w.events {
				if event == "flush" {
					flushes++
				} else if flushes >= 2 {
					t.Fatal("physical write after refused endl", w.events)
				}
			}
		}
	}
}
