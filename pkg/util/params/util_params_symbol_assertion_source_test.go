// Source: src/util/params.cpp:742-750,1017-1035; src/util/symbol.h:96.
package params

import (
	"os"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
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
