//go:build g3trace

package subpaving

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func intervalRootTraceRecord(tag, function string, line int, body string) string {
	return "-------- [" + tag + "] " + function + " src/math/interval/interval_def.h:" +
		strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
}

func TestFixedIntervalRootRetainedTraceSites(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	lo, err := m.New()
	must(err)
	hi, err := m.New()
	must(err)
	p, err := m.New()
	must(err)
	rootLo, err := m.New()
	must(err)
	rootHi, err := m.New()
	must(err)
	src, err := NewFixedInterval(m)
	must(err)
	out, err := NewFixedInterval(m)
	must(err)
	t.Cleanup(func() {
		for _, err := range []error{out.Close(), src.Close(), rootHi.Close(), rootLo.Close(), p.Close(), hi.Close(), lo.Close(), m.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
	})
	set := func(a, b int64) {
		t.Helper()
		must(m.SetInt64(lo, a))
		must(m.SetInt64(hi, b))
		must(src.SetFinite(lo, hi, false, false))
	}
	show := func(v *FixedInterval) string {
		t.Helper()
		var b bytes.Buffer
		must(v.Display(&b))
		return b.String()
	}
	showNumeral := func(v *numeral.FixedPoint) string {
		t.Helper()
		var b bytes.Buffer
		must(m.Display(&b, v))
		return b.String()
	}
	var writer hardwareIntervalTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	roots := interval.FixedRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetRatio(p, 1, 8192))
	set(-8, 27)
	must(out.XnEqY(src, 3, p, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled fixed trace: %q, flushes %d", writer.String(), writer.flushes)
	}
	tags.Enable("interval_nth_root")
	must(out.XnEqY(src, 1, p, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("fixed identity traced: %q, flushes %d", writer.String(), writer.flushes)
	}
	srcText := show(src)
	must(out.XnEqY(src, 3, p, roots))
	want := intervalRootTraceRecord("interval_nth_root", "nth_root", 1427, srcText+" --> "+show(out)+"\n")
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("fixed odd trace: got %q / %d flushes; want %q / 1", writer.String(), writer.flushes, want)
	}
	writer.Reset()
	writer.flushes = 0
	tags.Enable("interval_xn_eq_y")
	set(1, 4)
	srcText = show(src)
	must(roots.Bounds(hi, 2, p, rootLo, rootHi))
	must(out.XnEqY(src, 2, p, roots))
	want = intervalRootTraceRecord("interval_xn_eq_y", "xn_eq_y", 1459,
		"x^n = "+srcText+"\nsqrt(y) in "+showNumeral(rootLo)+" "+showNumeral(rootHi)+"\n") +
		intervalRootTraceRecord("interval_xn_eq_y", "xn_eq_y", 1470, "interval for x: "+show(out)+"\n")
	if writer.String() != want || writer.flushes != 2 {
		t.Fatalf("fixed even trace: got %q / %d flushes; want %q / 2", writer.String(), writer.flushes, want)
	}
}

func TestFastIntervalRootRetainedTraceSites(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	budget := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	must(err)
	owners := make([]*numeral.ScopedFastFloat, 5)
	values := make([]*numeral.FastFloat, len(owners))
	for i := range owners {
		owners[i], err = numeral.NewScopedFastFloat(m)
		must(err)
		values[i], err = owners[i].Get()
		must(err)
	}
	lo, hi, p, rootLo, rootHi := values[0], values[1], values[2], values[3], values[4]
	src, err := NewFastInterval(m)
	must(err)
	out, err := NewFastInterval(m)
	must(err)
	t.Cleanup(func() {
		for _, err := range []error{out.Close(), src.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
		for i := len(owners) - 1; i >= 0; i-- {
			if err := owners[i].Close(); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("fast interval root trace owners", budget.Used())
		}
	})
	set := func(a, b int32) {
		t.Helper()
		must(m.SetInt32(lo, a))
		must(m.SetInt32(hi, b))
		must(src.SetFinite(lo, hi, false, false))
	}
	show := func(v *FastInterval) string {
		t.Helper()
		var b bytes.Buffer
		must(v.Display(&b))
		return b.String()
	}
	showNumeral := func(v *numeral.FastFloat) string {
		t.Helper()
		var b bytes.Buffer
		must(m.DisplayTo(&b, v))
		return b.String()
	}
	var writer hardwareIntervalTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	roots := interval.FastRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetFraction32(p, 1, 8192))
	set(-8, 27)
	must(out.XnEqY(src, 3, p, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled fast trace: %q, flushes %d", writer.String(), writer.flushes)
	}
	tags.Enable("interval_nth_root")
	must(out.XnEqY(src, 1, p, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("fast identity traced: %q, flushes %d", writer.String(), writer.flushes)
	}
	srcText := show(src)
	must(out.XnEqY(src, 3, p, roots))
	want := intervalRootTraceRecord("interval_nth_root", "nth_root", 1427, srcText+" --> "+show(out)+"\n")
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("fast odd trace: got %q / %d flushes; want %q / 1", writer.String(), writer.flushes, want)
	}
	writer.Reset()
	writer.flushes = 0
	tags.Enable("interval_xn_eq_y")
	set(1, 4)
	srcText = show(src)
	must(roots.Bounds(hi, 2, p, rootLo, rootHi))
	must(out.XnEqY(src, 2, p, roots))
	want = intervalRootTraceRecord("interval_xn_eq_y", "xn_eq_y", 1459,
		"x^n = "+srcText+"\nsqrt(y) in "+showNumeral(rootLo)+" "+showNumeral(rootHi)+"\n") +
		intervalRootTraceRecord("interval_xn_eq_y", "xn_eq_y", 1470, "interval for x: "+show(out)+"\n")
	if writer.String() != want || writer.flushes != 2 {
		t.Fatalf("fast even trace: got %q / %d flushes; want %q / 2", writer.String(), writer.flushes, want)
	}
}
