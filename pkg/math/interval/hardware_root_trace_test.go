//go:build g3trace

package interval

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareRootRetainedTraceSitesSourceOrder(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var a, precision, lo, hi numeral.HardwareFloat
	if err = m.SetRatioInt32(&precision, 1, 8192); err != nil {
		t.Fatal(err)
	}
	if err = m.SetInt32(&a, 4); err != nil {
		t.Fatal(err)
	}
	var writer rootTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	root := HardwareRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	if err = root.Bounds(&a, 2, &precision, &lo, &hi); err != nil || writer.Len() != 0 {
		t.Fatalf("disabled trace: %v, %q", err, writer.String())
	}
	tags.Enable("nth_root_trace")
	if err = root.Bounds(&a, 2, &precision, &lo, &hi); err != nil {
		t.Fatal(err)
	}
	aText, err := m.ToString(&a)
	if err != nil {
		t.Fatal(err)
	}
	loText, err := m.ToString(&lo)
	if err != nil {
		t.Fatal(err)
	}
	hiText, err := m.ToString(&hi)
	if err != nil {
		t.Fatal(err)
	}
	want := "[nth-root] (" + aText + ")^(1/2) >= " + loText + "\n" +
		"[nth-root] (" + aText + ")^(1/2) <= " + hiText + "\n"
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("short trace: %q, want %q; flushes %d", writer.String(), want, writer.flushes)
	}
	writer.Reset()
	if err = root.Bounds(&a, 1, &precision, &lo, &hi); err != nil || writer.Len() != 0 {
		t.Fatalf("easy root emitted trace: %v, %q", err, writer.String())
	}
	if err = m.SetInt32(&a, 8); err != nil {
		t.Fatal(err)
	}
	tags.Enable("nth_root")
	tags.Enable("nth_root_bug")
	writer.flushes = 0
	if err = root.Bounds(&a, 3, &precision, &lo, &hi); err != nil {
		t.Fatal(err)
	}
	aText, err = m.ToString(&a)
	if err != nil {
		t.Fatal(err)
	}
	loText, err = m.ToString(&lo)
	if err != nil {
		t.Fatal(err)
	}
	hiText, err = m.ToString(&hi)
	if err != nil {
		t.Fatal(err)
	}
	record := func(tag, function string, line int, body string) string {
		return "-------- [" + tag + "] " + function + " src/math/interval/interval_def.h:" + strconv.Itoa(line) + " ---------\n" +
			body + "------------------------------------------------\n"
	}
	tail := record("nth_root_bug", "nth_root_pos", 344,
		"Assuming upper\nA: "+aText+"\nhi: "+hiText+"\nlo: "+hiText+"\n") +
		"[nth-root] (" + aText + ")^(1/3) >= " + loText + "\n" +
		"[nth-root] (" + aText + ")^(1/3) <= " + hiText + "\n"
	got := writer.String()
	if !strings.HasSuffix(got, tail) {
		t.Fatalf("bound check/short trace tail: got %q, want suffix %q", got, tail)
	}
	iterations := strings.TrimSuffix(got, tail)
	header := "-------- [nth_root] approx_nth_root src/math/interval/interval_def.h:315 ---------\n"
	parts := strings.Split(iterations, header)
	if len(parts) < 2 || parts[0] != "" || writer.flushes != len(parts)-1+2 {
		t.Fatalf("iteration/flush count: records %d, flushes %d", len(parts)-1, writer.flushes)
	}
	for i, part := range parts[1:] {
		if !strings.HasSuffix(part, "------------------------------------------------\n") {
			t.Fatalf("iteration %d lacks footer: %q", i, part)
		}
		body := strings.TrimSuffix(part, "------------------------------------------------\n")
		lines := strings.Split(body, "\n")
		for j, prefix := range []string{"A:       " + aText, "x:       ", "x_prime: ", "d:       "} {
			if len(lines) != 5 || !strings.HasPrefix(lines[j], prefix) || len(lines[j]) <= len(prefix) && j != 0 {
				t.Fatalf("iteration %d field %d: %q", i, j, body)
			}
		}
	}
}
