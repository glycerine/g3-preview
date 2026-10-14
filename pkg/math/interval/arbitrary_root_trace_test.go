//go:build g3trace

package interval

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type rootTraceWriter struct {
	bytes.Buffer
	flushes int
}

func (w *rootTraceWriter) Flush() error { w.flushes++; return nil }

func TestArbitraryRootRetainedShortTraceAfterCertifiedBounds(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var owners [4]*numeral.ScopedArbitraryFloatNumeral
	for i := range owners {
		owners[i], err = numeral.NewScopedArbitraryFloatNumeral(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := len(owners) - 1; i >= 0; i-- {
			if e := owners[i].Close(); e != nil {
				t.Error(e)
			}
		}
		for _, close := range []func() error{m.Close, core.Close, runtime.Close} {
			if e := close(); e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("root trace owner storage retained", budget.Used())
		}
	})
	var values [4]*numeral.ArbitraryFloat
	for i, owner := range owners {
		values[i], err = owner.Get()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, precision, lower, upper := values[0], values[1], values[2], values[3]
	if err = m.SetInt32(a, 4); err != nil {
		t.Fatal(err)
	}
	if err = m.SetFractionSigned(precision, 1, 8192); err != nil {
		t.Fatal(err)
	}
	var writer rootTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	root := ArbitraryRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	if err = root.Bounds(a, 2, precision, lower, upper); err != nil || writer.Len() != 0 {
		t.Fatalf("disabled trace: %v, %q", err, writer.String())
	}
	tags.Enable("nth_root_trace")
	if err = root.Bounds(a, 2, precision, lower, upper); err != nil {
		t.Fatal(err)
	}
	aText, err := m.ToString(a)
	if err != nil {
		t.Fatal(err)
	}
	loText, err := m.ToString(lower)
	if err != nil {
		t.Fatal(err)
	}
	hiText, err := m.ToString(upper)
	if err != nil {
		t.Fatal(err)
	}
	want := "[nth-root] (" + aText + ")^(1/2) >= " + loText + "\n" +
		"[nth-root] (" + aText + ")^(1/2) <= " + hiText + "\n"
	if writer.String() != want || writer.flushes != 1 {
		t.Fatalf("source short trace: %q, want %q; flushes %d", writer.String(), want, writer.flushes)
	}
	writer.Reset()
	if err = root.Bounds(a, 1, precision, lower, upper); err != nil || writer.Len() != 0 {
		t.Fatalf("easy root trace: %v, %q", err, writer.String())
	}
	if err = m.SetInt32(a, 8); err != nil {
		t.Fatal(err)
	}
	tags.Enable("nth_root")
	tags.Enable("nth_root_bug")
	writer.flushes = 0
	if err = root.Bounds(a, 3, precision, lower, upper); err != nil {
		t.Fatal(err)
	}
	aText, err = m.ToString(a)
	if err != nil {
		t.Fatal(err)
	}
	loText, err = m.ToString(lower)
	if err != nil {
		t.Fatal(err)
	}
	hiText, err = m.ToString(upper)
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
		t.Fatalf("bound check and short trace tail:\n got %q\nwant suffix %q", got, tail)
	}
	iterations := strings.TrimSuffix(got, tail)
	header := "-------- [nth_root] approx_nth_root src/math/interval/interval_def.h:315 ---------\n"
	parts := strings.Split(iterations, header)
	if len(parts) < 2 || parts[0] != "" || writer.flushes != len(parts)-1+2 {
		t.Fatalf("iteration record/flush count: records %d, flushes %d", len(parts)-1, writer.flushes)
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
