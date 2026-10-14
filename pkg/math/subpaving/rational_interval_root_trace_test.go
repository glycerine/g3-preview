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

func TestRationalIntervalRootRetainedTraceSites(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	src, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	lo, hi, precision := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	rootLo, rootHi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, err := range []error{out.Close(), src.Close(), m.Del(&rootHi), m.Del(&rootLo), m.Del(&precision), m.Del(&hi), m.Del(&lo), m.CloseChecked()} {
			if err != nil {
				t.Error(err)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational root trace owners", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	set := func(a, b int32) {
		t.Helper()
		must(m.SetInt32(&lo, a))
		must(m.SetInt32(&hi, b))
		must(src.SetFinite(&lo, &hi, false, false))
	}
	display := func(v *RationalInterval) string {
		t.Helper()
		var b bytes.Buffer
		must(v.Display(&b))
		return b.String()
	}
	displayNumeral := func(v *numeral.Rational) string {
		t.Helper()
		var b bytes.Buffer
		must(m.DisplayTo(&b, v))
		return b.String()
	}
	record := func(tag, function string, line int, body string) string {
		return "-------- [" + tag + "] " + function + " src/math/interval/interval_def.h:" +
			strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
	}
	var writer hardwareIntervalTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	roots := interval.RationalRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetFractionSigned(&precision, 1, 8192))
	set(-8, 27)
	must(out.XnEqY(src, 3, &precision, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled trace: %q, flushes %d", writer.String(), writer.flushes)
	}
	tags.Enable("interval_nth_root")
	must(out.XnEqY(src, 1, &precision, roots))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("identity branch traced: %q, flushes %d", writer.String(), writer.flushes)
	}
	srcText := display(src)
	must(out.XnEqY(src, 3, &precision, roots))
	want := record("interval_nth_root", "nth_root", 1427, srcText+" --> "+display(out)+"\n")
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("odd root trace: got %q / %d flushes; want %q / 1", writer.String(), writer.flushes, want)
	}
	writer.Reset()
	writer.flushes = 0
	tags.Enable("interval_xn_eq_y")
	set(1, 4)
	srcText = display(src)
	must(roots.Bounds(&hi, 2, &precision, &rootLo, &rootHi))
	must(out.XnEqY(src, 2, &precision, roots))
	want = record("interval_xn_eq_y", "xn_eq_y", 1459,
		"x^n = "+srcText+"\nsqrt(y) in "+displayNumeral(&rootLo)+" "+displayNumeral(&rootHi)+"\n") +
		record("interval_xn_eq_y", "xn_eq_y", 1470, "interval for x: "+display(out)+"\n")
	if writer.String() != want || writer.flushes != 2 {
		t.Fatalf("even inverse trace: got %q / %d flushes; want %q / 2", writer.String(), writer.flushes, want)
	}
	refusing := &arbitraryRootRefusingWriter{}
	failedTags := diagnostic.NewTraceTagsWithWriter(refusing)
	failedTags.Enable("interval_nth_root")
	roots.TraceTags = failedTags
	set(-8, 27)
	must(out.XnEqY(src, 3, &precision, roots))
	if refusing.flushes != 1 || display(out) == "" {
		t.Fatalf("stream refusal interrupted rational root: flushes %d", refusing.flushes)
	}
}
