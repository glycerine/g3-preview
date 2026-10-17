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

func TestParamsMatchedScalarSymbolAssertionAfterKind(t *testing.T) {
	if !udiag.DebugBuild {
		return
	}
	for _, byName := range []bool{false, true} {
		t.Run(map[bool]string{false: "symbol-key", true: "name-key"}[byName], func(t *testing.T) {
			d, _, _ := debugFixture(t)
			c, sy, _ := paramsTestOwner(t)
			paramsMust(t, sy.BindDebug(d))
			paramsMust(t, d.EnableAssertions(true))
			v, e := c.NewValue()
			paramsMust(t, e)
			name := "matched-scalar"
			k := internSymbol(t, sy, name)
			setUInt := func() error { return c.SetUInt(&v, k, 42) }
			setSymbol := func() error { return c.SetSymbol(&v, k, usymbol.Numerical(7)) }
			if byName {
				setUInt = func() error { return c.SetUIntName(&v, &name, 42) }
				setSymbol = func() error { return c.SetSymbolName(&v, &name, usymbol.Numerical(7)) }
			}
			paramsMust(t, setUInt())
			if e := setSymbol(); !errors.Is(e, usymbol.ErrInvalid) {
				t.Fatal("numerical bare_str assertion", e)
			}
			x := c.Entry(&c.Blocks[v.Slot-1], 0)
			if x.key != k || x.Kind != ParamSymbol || x.symbol != (usymbol.Symbol{}) {
				t.Fatal("source assertion must follow scalar deletion and kind store", *x)
			}
			paramsMust(t, d.EnableAssertions(false))
			paramsMust(t, setSymbol())
			got, e := c.GetSymbol(&v, k, usymbol.Symbol{})
			paramsMust(t, e)
			spelling, present, e := sy.BareString(got)
			paramsMust(t, e)
			if got.IsNumerical() || !present || spelling != "" {
				t.Fatal("source empty-symbol retry", got, spelling)
			}
			paramsMust(t, c.Del(&v))
		})
	}
}
