//go:build g3trace

package subpaving

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type hardwareIntervalTraceWriter struct {
	bytes.Buffer
	flushes int
}

func (w *hardwareIntervalTraceWriter) Flush() error { w.flushes++; return nil }

func TestHardwareIntervalRootRetainedTraceSites(t *testing.T) {
	m, src, out, values := hardwareIntervalFixture(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	set := func(lo, hi int32) {
		t.Helper()
		must(m.SetInt32(&values[0], lo))
		must(m.SetInt32(&values[1], hi))
		must(src.SetFinite(&values[0], &values[1], false, false))
	}
	display := func(v *HardwareInterval) string {
		t.Helper()
		var b bytes.Buffer
		must(v.Display(&b))
		return b.String()
	}
	record := func(tag string, line int, body string) string {
		return "-------- [" + tag + "] " + map[string]string{"interval_nth_root": "nth_root", "interval_xn_eq_y": "xn_eq_y"}[tag] +
			" src/math/interval/interval_def.h:" + strconv.Itoa(line) + " ---------\n" + body +
			"------------------------------------------------\n"
	}
	var writer hardwareIntervalTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	roots := interval.HardwareRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	var precision numeral.HardwareFloat
	must(m.SetRatioInt32(&precision, 1, 8192))
	set(-8, 27)
	must(out.NthRoot(src, 3, &precision, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled trace: %q, flushes %d", writer.String(), writer.flushes)
	}
	tags.Enable("interval_nth_root")
	must(out.NthRoot(src, 1, &precision, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("identity branch traced: %q, flushes %d", writer.String(), writer.flushes)
	}
	srcText := display(src)
	must(out.NthRoot(src, 3, &precision, roots))
	want := record("interval_nth_root", 1427, srcText+" --> "+display(out)+"\n")
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("nth_root trace: got %q / %d flushes; want %q / 1", writer.String(), writer.flushes, want)
	}
	writer.Reset()
	writer.flushes = 0
	tags.Enable("interval_xn_eq_y")
	set(1, 4)
	srcText = display(src)
	_, srcUpper, err := src.endpoints()
	must(err)
	var lo, hi numeral.HardwareFloat
	must(roots.Bounds(srcUpper, 2, &precision, &lo, &hi))
	loText, err := m.ToString(&lo)
	must(err)
	hiText, err := m.ToString(&hi)
	must(err)
	must(out.XnEqY(src, 2, &precision, roots))
	want = record("interval_xn_eq_y", 1459, "x^n = "+srcText+"\nsqrt(y) in "+loText+" "+hiText+"\n") +
		record("interval_xn_eq_y", 1470, "interval for x: "+display(out)+"\n")
	if writer.String() != want || writer.flushes != 2 {
		t.Fatalf("xn_eq_y trace: got %q / %d flushes; want %q / 2", writer.String(), writer.flushes, want)
	}
}
