//go:build g3trace

package interval

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type fixedFastTraceWriter struct {
	bytes.Buffer
	flushes int
}

func (w *fixedFastTraceWriter) Flush() error { w.flushes++; return nil }

type fixedFastRefusingWriter struct{ flushes int }

func (*fixedFastRefusingWriter) Write([]byte) (int, error) {
	return 0, errors.New("trace write refused")
}
func (w *fixedFastRefusingWriter) Flush() error { w.flushes++; return nil }

func assertFixedFastRootTrace(t *testing.T, got string, flushes int, a, lo, hi string) {
	t.Helper()
	short := "[nth-root] (" + a + ")^(1/3) >= " + lo + "\n" +
		"[nth-root] (" + a + ")^(1/3) <= " + hi + "\n"
	if !strings.HasSuffix(got, short) {
		t.Fatalf("short trace tail: got %q, want suffix %q", got, short)
	}
	got = strings.TrimSuffix(got, short)
	bugHeader := "-------- [nth_root_bug] nth_root_pos src/math/interval/interval_def.h:344 ---------\n"
	start := strings.LastIndex(got, bugHeader)
	if start < 0 {
		t.Fatalf("missing bound-check record: %q", got)
	}
	bug := strings.TrimPrefix(got[start:], bugHeader)
	if !strings.HasSuffix(bug, "------------------------------------------------\n") {
		t.Fatalf("bound-check footer: %q", bug)
	}
	bug = strings.TrimSuffix(bug, "------------------------------------------------\n")
	lines := strings.Split(bug, "\n")
	if len(lines) != 5 || lines[0] != "Assuming upper" || lines[1] != "A: "+a ||
		!strings.HasPrefix(lines[2], "hi: ") || lines[2] == "hi: " || lines[3] != "lo: "+strings.TrimPrefix(lines[2], "hi: ") {
		t.Fatalf("bound-check fields: %q", bug)
	}
	iterations := got[:start]
	header := "-------- [nth_root] approx_nth_root src/math/interval/interval_def.h:315 ---------\n"
	parts := strings.Split(iterations, header)
	if len(parts) < 2 || parts[0] != "" || flushes != len(parts)-1+2 {
		t.Fatalf("iteration count/flushes: %d records, %d flushes", len(parts)-1, flushes)
	}
	for i, part := range parts[1:] {
		if !strings.HasSuffix(part, "------------------------------------------------\n") {
			t.Fatalf("iteration %d footer: %q", i, part)
		}
		body := strings.TrimSuffix(part, "------------------------------------------------\n")
		fields := strings.Split(body, "\n")
		for j, prefix := range []string{"A:       " + a, "x:       ", "x_prime: ", "d:       "} {
			if len(fields) != 5 || !strings.HasPrefix(fields[j], prefix) || j != 0 && len(fields[j]) == len(prefix) {
				t.Fatalf("iteration %d field %d: %q", i, j, body)
			}
		}
	}
}

func TestFixedRootRetainedTraceSitesSourceOrder(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	a, err := m.New()
	must(err)
	p, err := m.New()
	must(err)
	lo, err := m.New()
	must(err)
	hi, err := m.New()
	must(err)
	t.Cleanup(func() {
		for _, err := range []error{hi.Close(), lo.Close(), p.Close(), a.Close(), m.Close()} {
			if err != nil {
				t.Error(err)
			}
		}
	})
	show := func(v *numeral.FixedPoint) string {
		t.Helper()
		var b bytes.Buffer
		must(m.Display(&b, v))
		return b.String()
	}
	var writer fixedFastTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	root := FixedRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetRatio(p, 1, 8192))
	must(m.SetInt64(a, 2))
	must(root.Bounds(a, 3, p, lo, hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled fixed root trace: %q / %d", writer.String(), writer.flushes)
	}
	for _, tag := range []string{"nth_root", "nth_root_bug", "nth_root_trace"} {
		tags.Enable(tag)
	}
	must(m.SetInt64(a, 1))
	must(root.Bounds(a, 3, p, lo, hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("easy fixed root traced: %q / %d", writer.String(), writer.flushes)
	}
	must(m.SetInt64(a, 2))
	must(root.Bounds(a, 3, p, lo, hi))
	assertFixedFastRootTrace(t, writer.String(), writer.flushes, show(a), show(lo), show(hi))
}

func TestFastRootRetainedTraceSitesSourceOrder(t *testing.T) {
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
	owners := make([]*numeral.ScopedFastFloat, 4)
	values := make([]*numeral.FastFloat, len(owners))
	for i := range owners {
		owners[i], err = numeral.NewScopedFastFloat(m)
		must(err)
		values[i], err = owners[i].Get()
		must(err)
	}
	a, p, lo, hi := values[0], values[1], values[2], values[3]
	t.Cleanup(func() {
		for i := len(owners) - 1; i >= 0; i-- {
			if err := owners[i].Close(); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("fast root trace owners", budget.Used())
		}
	})
	show := func(v *numeral.FastFloat) string {
		t.Helper()
		var b bytes.Buffer
		must(m.DisplayTo(&b, v))
		return b.String()
	}
	var writer fixedFastTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	root := FastRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetFraction32(p, 1, 8192))
	must(m.SetInt32(a, 2))
	must(root.Bounds(a, 3, p, lo, hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled fast root trace: %q / %d", writer.String(), writer.flushes)
	}
	for _, tag := range []string{"nth_root", "nth_root_bug", "nth_root_trace"} {
		tags.Enable(tag)
	}
	must(m.SetInt32(a, 1))
	must(root.Bounds(a, 3, p, lo, hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("easy fast root traced: %q / %d", writer.String(), writer.flushes)
	}
	must(m.SetInt32(a, 2))
	must(root.Bounds(a, 3, p, lo, hi))
	assertFixedFastRootTrace(t, writer.String(), writer.flushes, show(a), show(lo), show(hi))
	writer.Reset()
	writer.flushes = 0
	refusing := &fixedFastRefusingWriter{}
	failedTags := diagnostic.NewTraceTagsWithWriter(refusing)
	for _, tag := range []string{"nth_root", "nth_root_bug", "nth_root_trace"} {
		failedTags.Enable(tag)
	}
	root.TraceTags = failedTags
	must(root.Bounds(a, 3, p, lo, hi))
	if refusing.flushes == 0 || show(lo) == "" || show(hi) == "" {
		t.Fatalf("trace writer refusal interrupted fast root: %d flushes", refusing.flushes)
	}
}
