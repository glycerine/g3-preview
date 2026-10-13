package numeral

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNArithmeticSourceTraceOrder(t *testing.T) {
	m := naturalManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	m.trace = tags
	traceCompiled := tags.ShouldTrace("mpn")
	compiled := SourceExtendedGCDDebug && traceCompiled
	record := func(function string, line int, body string) string {
		return fmt.Sprintf("-------- [mpn] %s src/util/mpn.cpp:%d ---------\n", function, line) +
			body + "------------------------------------------------\n"
	}
	a, b := []uint32{7}, []uint32{3}
	var out [2]uint32
	if size := m.Add(a, b, out[:]); size != 1 || out != [2]uint32{10, 0} {
		t.Fatal("addition", size, out)
	}
	var diff [1]uint32
	if borrow := m.Sub(a, b, diff[:]); borrow != 0 || diff[0] != 4 {
		t.Fatal("subtraction", borrow, diff)
	}
	m.Mul(a, b, out[:])
	if out != [2]uint32{21, 0} {
		t.Fatal("multiplication", out)
	}
	if got := m.Compare(a, b); got != 1 {
		t.Fatal("comparison", got)
	}
	var q, r [1]uint32
	if _, err := m.Div(a, b, q[:], r[:]); err != nil || q[0] != 2 || r[0] != 1 {
		t.Fatal("division", q, r, err)
	}
	if !compiled {
		want := ""
		if traceCompiled {
			want = record("compare", 55, " > ")
		}
		if got := output.String(); got != want {
			t.Fatalf("MPN compare-only TRACE got %q, want %q", got, want)
		}
		return
	}
	want := record("trace", 411, "[mpn] 7 + 3 == ") + record("trace_nl", 427, "10\n") +
		record("trace", 411, "[mpn] 7 - 3 == ") + record("trace_nl", 427, "4\n") +
		record("trace", 411, "[mpn] 7 * 3 == ") + record("trace_nl", 427, "21\n") +
		record("trace", 420, "7") + record("compare", 55, " > ") + record("trace_nl", 427, "3\n") +
		record("trace", 411, "[mpn] 7 / 3 == ") + record("trace_nl", 427, "2\n") +
		record("trace", 411, "[mpn] 7 % 3 == ") + record("trace_nl", 427, "1\n") +
		record("trace", 411, "[mpn] 2 * 3 == ") + record("trace_nl", 427, "6\n") +
		record("trace", 411, "[mpn] 6 + 1 == ") + record("trace_nl", 427, "7\n")
	if got := output.String(); got != want {
		t.Fatalf("source MPN TRACE got %q, want %q", got, want)
	}
}

func TestMPNCompareTraceUsesSourceDecimalConversionForMultipleLimbs(t *testing.T) {
	m := naturalManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	tags.Enable("mpn_to_string")
	m.trace = tags
	if got := m.Compare([]uint32{0, 1}, []uint32{1, 1}); got != -1 {
		t.Fatal("multi-limb comparison", got)
	}
	got := output.String()
	if !tags.ShouldTrace("mpn") {
		if got != "" {
			t.Fatal("erased comparison trace", got)
		}
		return
	}
	if !strings.Contains(got, "-------- [mpn] compare src/util/mpn.cpp:55 ---------\n < ") {
		t.Fatal("missing source comparison record", got)
	}
	if !SourceExtendedGCDDebug {
		if strings.Contains(got, "mpn_to_string") {
			t.Fatal("debug-only value conversion reached", got)
		}
		return
	}
	if strings.Count(got, "[mpn] to_string [1|0] == ") != 1 ||
		strings.Count(got, "[mpn] to_string [1|1] == ") != 1 ||
		strings.Count(got, "4294967296\n") != 1 ||
		strings.Count(got, "4294967297\n") != 2 {
		t.Fatal("source multi-limb decimal trace", got)
	}
}

func TestMPNMultiLimbDivisionNestedArithmeticTrace(t *testing.T) {
	m := naturalManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	m.trace = tags
	a, b := []uint32{0, 0, 1}, []uint32{1, 1}
	q, r := make([]uint32, 2), make([]uint32, 2)
	if _, err := m.Div(a, b, q, r); err != nil {
		t.Fatal(err)
	}
	if q[0] != ^uint32(0) || q[1] != 0 || r[0] != 1 || r[1] != 0 {
		t.Fatal("multi-limb division", q, r)
	}
	got := output.String()
	if !SourceExtendedGCDDebug || !tags.ShouldTrace("mpn") {
		if got != "" {
			t.Fatal("erased debug trace", got)
		}
		return
	}
	if strings.Count(got, "[mpn] ") < 8 ||
		strings.Count(got, " * ") != 3 ||
		strings.Count(got, " - ") != 2 ||
		!strings.Contains(got, "[mpn] 4294967295 * 4294967297 == ") ||
		!strings.Contains(got, "[mpn] 18446744073709551616 % 4294967297 == ") {
		t.Fatal("source nested division arithmetic", got)
	}
}

func TestMPNDivisionAddbackTraceFollowsBorrow(t *testing.T) {
	m := naturalManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	m.trace = tags
	a, b := []uint32{1, 0, 0x80000000}, []uint32{2, 0, 0x80000000}
	q, r := make([]uint32, 1), make([]uint32, 3)
	if _, err := m.Div(a, b, q, r); err != nil || q[0] != 0 || r[0] != 1 || r[2] != 0x80000000 {
		t.Fatal("add-back division", q, r, err)
	}
	if m.Division[3] == nil {
		t.Fatal("source add-back branch not reached")
	}
	got := output.String()
	if !SourceExtendedGCDDebug || !tags.ShouldTrace("mpn") {
		if got != "" {
			t.Fatal("erased debug trace", got)
		}
		return
	}
	addback := strings.Index(got, " + ")
	remainder := strings.Index(got, " % ")
	if strings.Count(got, " + ") != 2 || addback < 0 || remainder < 0 || addback > remainder {
		t.Fatal("conditional add-back and verification record order", got)
	}
}
