// Source: src/util/symbol.h:96; src/util/debug.h:60-65.
package symbol

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestNumericalBareStringBorrowsDebugAssertionSwitch(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := diagnostic.NewDebugContext(region, b)
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewManager(Config{Budget: b, Debug: d})
	if e != nil {
		t.Fatal(e)
	}
	numeric := Numerical(7)
	if e := d.EnableAssertions(false); e != nil {
		t.Fatal(e)
	}
	text, present, e := m.BareString(numeric)
	if e != nil || !present || text != "" {
		t.Fatal("disabled source assertion must continue", text, present, e)
	}
	empty, e := m.Intern("")
	if e != nil {
		t.Fatal(e)
	}
	got, e := m.InternBorrowedSymbolCString(numeric)
	if e != nil || got != empty {
		t.Fatal("borrowed numerical C string continuation", got, empty, e)
	}
	if e := d.EnableAssertions(true); e != nil {
		t.Fatal(e)
	}
	text, present, e = m.BareString(numeric)
	if sourceDebug {
		if !errors.Is(e, ErrInvalid) || present || text != "" {
			t.Fatal("enabled source assertion", text, present, e)
		}
		if _, e := m.InternBorrowedSymbolCString(numeric); !errors.Is(e, ErrInvalid) {
			t.Fatal("borrowed source assertion", e)
		}
	} else if e != nil || !present || text != "" {
		t.Fatal("erased assertion", text, present, e)
	}
	if e := d.Close(); e != nil {
		t.Fatal(e)
	}
	text, present, e = m.BareString(numeric)
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) || present || text != "" {
			t.Fatal("stopped borrowed debug owner", text, present, e)
		}
	} else if e != nil || !present || text != "" {
		t.Fatal("erased borrowed debug owner", text, present, e)
	}
	if e := m.Close(); e != nil {
		t.Fatal(e)
	}
	if e := region.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("retained owner charge", b.Used())
	}
}

func TestNumericalBareStringContinuesAfterDebugAction(t *testing.T) {
	if !sourceDebug {
		return
	}
	b := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := diagnostic.NewDebugContext(region, b)
	if e != nil {
		t.Fatal(e)
	}
	inputFile, e := os.CreateTemp(t.TempDir(), "symbol-debug-input-*")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = inputFile.WriteString("CC"); e != nil {
		t.Fatal(e)
	}
	if _, e = inputFile.Seek(0, 0); e != nil {
		t.Fatal(e)
	}
	outputFile, e := os.CreateTemp(t.TempDir(), "symbol-debug-output-*")
	if e != nil {
		t.Fatal(e)
	}
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stderr: outputFile})
	if e != nil {
		t.Fatal(e)
	}
	input, e := diagnostic.NewDebuggerCharInput(streams, inputFile)
	if e != nil {
		t.Fatal(e)
	}
	action, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	if e != nil {
		t.Fatal(e)
	}
	if e = d.BindAssertionStreams(streams); e != nil {
		t.Fatal(e)
	}
	if e = d.BindAssertionActions(action); e != nil {
		t.Fatal(e)
	}
	m, e := NewManager(Config{Budget: b, Debug: d})
	if e != nil {
		t.Fatal(e)
	}
	numeric := Numerical(7)
	text, present, e := m.BareString(numeric)
	if e != nil || !present || text != "" {
		t.Fatal("action C must continue bare_str", text, present, e)
	}
	empty, e := m.Intern("")
	if e != nil {
		t.Fatal(e)
	}
	got, e := m.InternBorrowedSymbolCString(numeric)
	if e != nil || got != empty {
		t.Fatal("action C must continue borrowed bare_str", got, empty, e)
	}
	data, e := os.ReadFile(outputFile.Name())
	if e != nil {
		t.Fatal(e)
	}
	one := "ASSERTION VIOLATION\nFile: src/util/symbol.h\nLine: 96\n!is_numerical()\n(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(data) != one+one {
		t.Fatal("source assertion/action stream", string(data))
	}
	for _, close := range []func() error{m.Close, action.Close, input.Close, streams.Close, inputFile.Close, outputFile.Close, d.Close, region.Close} {
		if e = close(); e != nil {
			t.Fatal(e)
		}
	}
	if b.Used() != 0 {
		t.Fatal("retained assertion owner charge", b.Used())
	}
}
