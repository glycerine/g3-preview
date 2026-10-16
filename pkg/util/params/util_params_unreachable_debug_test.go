//go:build g3debug

package params

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Source: src/util/params.cpp:434,469,592. A malformed private kind can
// reach each default branch; Continue preserves the remainder of the loop.
func TestParamsInvalidEntrySourceUnreachableContinue(t *testing.T) {
	input, debugOut := bindProcessAssertionActions(t, "CCC")
	c, symbols, _ := paramsTestOwner(t)
	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	broken := internSymbol(t, symbols, "broken")
	ok := internSymbol(t, symbols, "ok")
	base := internSymbol(t, symbols, "base")
	paramsMust(t, c.SetUInt(&src, broken, 1))
	paramsMust(t, c.SetUInt(&src, ok, 2))
	paramsMust(t, c.SetUInt(&dst, base, 3))
	x := c.Entry(&c.Blocks[src.Slot-1], 0)
	x.Kind = ParamInvalid
	defer func() { x.Kind = ParamUInt }()

	var full bytes.Buffer
	paramsMust(t, c.DisplayTo(&full, &src))
	if got := full.String(); got != "(params broken ok 2)" {
		t.Fatal("full display Continue", got)
	}
	if got, e := c.DisplayKey(&src, broken); e != nil || got != "internal" {
		t.Fatal("key display default", got, e)
	}
	descr := pdnew(t, symbols, nil)
	paramsMust(t, descr.Insert(broken, ParamUInt, nil, nil, nil))
	paramsMust(t, descr.Insert(ok, ParamUInt, nil, nil, nil))
	module := "mod"
	var smt bytes.Buffer
	paramsMust(t, c.DisplaySMT2To(&smt, &src, &module, descr))
	if got := smt.String(); got != "(set-option :mod.broken)\n(set-option :mod.ok 2)\n" {
		t.Fatal("SMT2 display Continue", got)
	}
	paramsMust(t, c.Copy(&dst, &src))
	if got, e := c.Display(&dst); e != nil || got != "(params base 3 ok 2)" {
		t.Fatal("copy Continue", got, e)
	}
	state, e := input.State()
	paramsMust(t, e)
	if state.Position != 3 {
		t.Fatal("three source actions", state)
	}
	trace, e := os.ReadFile(debugOut.Name())
	paramsMust(t, e)
	for _, line := range []string{"Line: 434\n", "Line: 469\n", "Line: 592\n"} {
		if strings.Count(string(trace), line) != 1 {
			t.Fatal("missing or repeated source line", line, string(trace))
		}
	}
	if strings.Count(string(trace), "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n") != 3 {
		t.Fatal("source prompts", string(trace))
	}
	x.Kind = ParamUInt
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&src))
}
