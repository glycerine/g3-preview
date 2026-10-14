//go:build g3trace

package interval

import (
	"bytes"
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
	if writer.String() != want || writer.flushes != 3 {
		t.Fatalf("source short trace: %q, want %q; flushes %d", writer.String(), want, writer.flushes)
	}
	writer.Reset()
	if err = root.Bounds(a, 1, precision, lower, upper); err != nil || writer.Len() != 0 {
		t.Fatalf("easy root trace: %v, %q", err, writer.String())
	}
}
