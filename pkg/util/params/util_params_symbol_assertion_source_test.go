// Source: src/util/params.cpp:742-750,1017-1035; src/util/symbol.h:96.
package params

import (
	"errors"
	"os"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsNumericalSymbolValueContinuesSourceAssertion(t *testing.T) {
	if !udiag.DebugBuild {
		return
	}
	d, _, b := debugFixture(t)
	c, sy, _ := paramsTestOwner(t)
	paramsMust(t, sy.BindDebug(d))
	inputFile, e := os.CreateTemp(t.TempDir(), "params-debug-input-*")
	paramsMust(t, e)
	_, e = inputFile.WriteString("C")
	paramsMust(t, e)
	_, e = inputFile.Seek(0, 0)
	paramsMust(t, e)
	outputFile, e := os.CreateTemp(t.TempDir(), "params-debug-output-*")
	paramsMust(t, e)
	streams, e := udiag.NewMemoryStandardStreams(udiag.MemoryStandardStreamsConfig{Budget: b, Stderr: outputFile})
	paramsMust(t, e)
	input, e := udiag.NewDebuggerCharInput(streams, inputFile)
	paramsMust(t, e)
	action, e := udiag.NewDebugActionContext(udiag.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, action.Close())
		paramsMust(t, input.Close())
		paramsMust(t, streams.Close())
		paramsMust(t, inputFile.Close())
		paramsMust(t, outputFile.Close())
	})
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(action))
	v, e := c.NewValue()
	paramsMust(t, e)
	k := internSymbol(t, sy, "numeric-value")
	paramsMust(t, c.SetSymbol(&v, k, usymbol.Numerical(7)))
	got, e := c.GetSymbol(&v, k, usymbol.Symbol{})
	paramsMust(t, e)
	spelling, present, e := sy.BareString(got)
	paramsMust(t, e)
	if !present || spelling != "" || got.IsNumerical() {
		t.Fatal("continued setter must store ordinary empty symbol", got, spelling, present)
	}
	data, e := os.ReadFile(outputFile.Name())
	paramsMust(t, e)
	want := "ASSERTION VIOLATION\nFile: src/util/symbol.h\nLine: 96\n!is_numerical()\n(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(data) != want {
		t.Fatal("setter must emit one reached bare_str assertion", string(data))
	}
	paramsMust(t, c.Del(&v))
}

func TestParamsMatchedNumeralSymbolAssertionAfterDeleteAndKind(t *testing.T) {
	if !udiag.DebugBuild {
		return
	}
	d, _, _ := debugFixture(t)
	c, sy, q := paramsTestOwner(t)
	paramsMust(t, sy.BindDebug(d))
	paramsMust(t, d.EnableAssertions(true))
	v, e := c.NewValue()
	paramsMust(t, e)
	k := internSymbol(t, sy, "numeric-replacement")
	r, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetRat(&v, k, &r))
	if e := c.SetSymbol(&v, k, usymbol.Numerical(7)); !errors.Is(e, usymbol.ErrInvalid) {
		t.Fatal("numerical bare_str assertion", e)
	}
	b := &c.Blocks[v.Slot-1]
	x := c.Entry(b, 0)
	if x.Kind != ParamSymbol || x.Rat != (unumeral.OwnedRational{}) {
		t.Fatal("source assertion precedes kind or leaf deletion", *x)
	}
	stopped, e := c.StoppedSetReplacement(&v)
	paramsMust(t, e)
	if f := c.replacementFrames[v.Slot-1]; f.phase != 2 || f.leaf != (unumeral.OwnedRational{}) {
		t.Fatal("source stop did not pass the old leaf deletion", f)
	}
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, c.CompleteSetReplacement(&v, stopped))
	got, e := c.GetSymbol(&v, k, usymbol.Symbol{})
	paramsMust(t, e)
	spelling, present, e := sy.BareString(got)
	paramsMust(t, e)
	if got.IsNumerical() || !present || spelling != "" {
		t.Fatal("source empty-symbol continuation", got, spelling)
	}
	paramsMust(t, c.Del(&v))
	paramsMust(t, q.Del(&r))
}
