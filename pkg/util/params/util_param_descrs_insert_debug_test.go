//go:build g3debug

package params

import (
	"errors"
	"os"
	"strconv"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: src/util/params.cpp:75-80. The first assertion precedes dictionary lookup.
func TestParamDescrsInsertSourceAssertionActions(t *testing.T) {
	d, _, budget := debugFixture(t)
	in, e := os.CreateTemp(t.TempDir(), "descr-input-*")
	paramsMust(t, e)
	_, e = in.WriteString("TCTC")
	paramsMust(t, e)
	_, e = in.Seek(0, 0)
	paramsMust(t, e)
	out, e := os.CreateTemp(t.TempDir(), "descr-output-*")
	paramsMust(t, e)
	streams, e := udiag.NewMemoryStandardStreams(udiag.MemoryStandardStreamsConfig{Budget: budget, Stdout: out, Stderr: out})
	paramsMust(t, e)
	input, e := udiag.NewDebuggerCharInput(streams, in)
	paramsMust(t, e)
	actions, e := udiag.NewDebugActionContext(udiag.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(actions))
	old, e := udiag.BindProcessDebugContext(d)
	paramsMust(t, e)
	t.Cleanup(func() {
		_, e := udiag.BindProcessDebugContext(old)
		paramsMust(t, e)
		paramsMust(t, actions.Close())
		paramsMust(t, input.Close())
		paramsMust(t, streams.Close())
		paramsMust(t, in.Close())
		paramsMust(t, out.Close())
	})

	symbols := symbolManagerTest(t, nil, false)
	descr := pdnew(t, symbols, nil)
	numeric := usymbol.Numerical(7)
	var thrown z3exception.DefaultException
	if e := descr.Insert(numeric, ParamUInt, nil, nil, nil); !errors.As(e, &thrown) {
		t.Fatal("numerical Throw", e)
	}
	if kind, e := descr.Kind(numeric); e != nil || kind != ParamInvalid {
		t.Fatal("Throw inserted descriptor", kind, e)
	}
	paramsMust(t, descr.Insert(numeric, ParamUInt, nil, nil, nil))
	name, e := symbols.Intern("mode")
	paramsMust(t, e)
	paramsMust(t, descr.Insert(name, ParamUInt, nil, nil, nil))
	if e := descr.Insert(name, ParamBool, nil, nil, nil); !errors.As(e, &thrown) {
		t.Fatal("duplicate-kind Throw", e)
	}
	paramsMust(t, descr.Insert(name, ParamBool, nil, nil, nil))
	if kind, e := descr.Kind(name); e != nil || kind != ParamUInt {
		t.Fatal("duplicate changed original kind", kind, e)
	}
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, descr.Insert(usymbol.Numerical(8), ParamBool, nil, nil, nil))
	paramsMust(t, descr.Insert(name, ParamSymbol, nil, nil, nil))
	state, e := input.State()
	paramsMust(t, e)
	if state.Position != 4 {
		t.Fatal("disabled assertions consumed input", state)
	}
	got, e := os.ReadFile(out.Name())
	paramsMust(t, e)
	const prompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	want := ""
	for _, line := range []int{75, 75, 78, 78} {
		condition := "!name.is_numerical()"
		if line == 78 {
			condition = "i.m_kind == k"
		}
		want += "ASSERTION VIOLATION\nFile: src/util/params.cpp\nLine: " + strconv.Itoa(line) + "\n" + condition + "\n" + prompt
	}
	if string(got) != want {
		t.Fatalf("source diagnostics: got %q, want %q", got, want)
	}
}
