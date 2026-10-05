// Source: src/util/params.cpp:639-648,902-930; src/smt/params/qi_params.cpp:23-39.
// Source author: Leonardo de Moura (2011-05-09; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestParamsStringViewReturnedPointerNoFormatCopyAndInvalidation(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, c.Del(&v)) }()
	key := "fresh-view"
	text := "ABCD"
	paramsMust(t, c.SetStringName(&v, &key, &text))
	before := c.Stats()
	c.budget.FailAfter(0)
	view, e := c.GetStringViewFallbackName(&v, &key, nil, nil)
	paramsMust(t, e)
	if c.Stats() != before {
		t.Fatal("get_str copied/charged payload")
	}
	for i, want := range []byte{'A', 'B', 'C', 'D', 0} {
		got, e := view.Byte(uint64(i))
		paramsMust(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	c.budget.FailAfter(-1)
	paramsMust(t, c.Reset(&v))
	if _, e := view.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("retired source pointer read replacement", e)
	}
	empty := ""
	paramsMust(t, c.SetStringName(&v, &key, &empty))
	view, e = c.GetStringViewFallbackName(&v, &key, nil, nil)
	paramsMust(t, e)
	zero, e := view.Byte(0)
	paramsMust(t, e)
	if zero != 0 || !view.Present() {
		t.Fatal("nonnull empty pointer")
	}
	paramsMust(t, c.SetStringName(&v, &key, nil))
	view, e = c.GetStringViewFallbackName(&v, &key, nil, &text)
	paramsMust(t, e)
	if view.Present() {
		t.Fatal("matching null incorrectly used fallback default")
	}
	spelling, e := sy.Intern("borrowed-spelling")
	paramsMust(t, e)
	k, e := sy.Intern(key)
	paramsMust(t, e)
	paramsMust(t, c.SetStringSymbol(&v, k, spelling))
	view, e = c.GetStringViewFallbackName(&v, &key, nil, nil)
	paramsMust(t, e)
	paramsMust(t, c.Reset(&v))
	ch, e := view.Byte(0)
	paramsMust(t, e)
	if ch != 'b' {
		t.Fatal("borrowed global symbol incorrectly owned by params block")
	}
}

func TestParamsStringViewFallbackNullAndProviderLifetime(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	f, e := c.NewValue()
	paramsMust(t, e)
	key := "fallback-view"
	text := "module"
	def := "literal"
	paramsMust(t, c.SetStringName(&f, &key, &text))
	view, e := c.GetStringViewFallbackName(&v, &key, &f, &def)
	paramsMust(t, e)
	ch, e := view.Byte(0)
	paramsMust(t, e)
	if ch != 'm' {
		t.Fatal("module fallback pointer")
	}
	paramsMust(t, c.SetBoolName(&v, &key, true))
	view, e = c.GetStringViewFallbackName(&v, &key, &f, &def)
	paramsMust(t, e)
	ch, e = view.Byte(0)
	paramsMust(t, e)
	if ch != 'm' {
		t.Fatal("wrong local type skips to module")
	}
	paramsMust(t, c.SetStringName(&v, &key, nil))
	view, e = c.GetStringViewFallbackName(&v, &key, &f, &def)
	paramsMust(t, e)
	if view.Present() {
		t.Fatal("matching null is not absence")
	}
	paramsMust(t, c.Reset(&v))
	paramsMust(t, c.Reset(&f))
	view, e = c.GetStringViewFallbackName(&v, &key, &f, &def)
	paramsMust(t, e)
	ch, e = view.Byte(0)
	paramsMust(t, e)
	if ch != 'l' {
		t.Fatal("literal fallback")
	}
	spelling, e := sy.Intern("survives-params-context")
	paramsMust(t, e)
	k, e := sy.Intern(key)
	paramsMust(t, e)
	paramsMust(t, c.SetStringSymbol(&v, k, spelling))
	view, e = c.GetStringViewFallbackName(&v, &key, nil, nil)
	paramsMust(t, e)
	paramsMust(t, c.Close())
	ch, e = view.Byte(0)
	paramsMust(t, e)
	if ch != 's' {
		t.Fatal("borrowed spelling depended on retired wrapper instead of exact byte provider")
	}
	paramsMust(t, sy.Close())
	if _, e = view.Byte(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("actual borrowed provider closed lifetime", e)
	}
}
func TestParamsStringViewCStringSentryNullAndStaleBeforeBadbit(t *testing.T) {
	c, _, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, c.Del(&v)) }()
	key := "stream-view"
	text := "ABCD"
	paramsMust(t, c.SetStringName(&v, &key, &text))
	view, e := c.GetStringViewFallbackName(&v, &key, nil, nil)
	paramsMust(t, e)
	sb := arena.NewBudget(^uint64(0))
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, out.SetWidth(7))
	paramsMust(t, out.AppendCStringView(view))
	got, e := out.String()
	paramsMust(t, e)
	if got != "   ABCD" {
		t.Fatal("one actual field width and live bytes", got)
	}
	paramsMust(t, out.AppendCStringView(ParamsStringView{}))
	paramsMust(t, out.AppendCString("ignored"))
	after, e := out.String()
	paramsMust(t, e)
	if after != got || !out.fileBad {
		t.Fatal("null pointer nonthrowing badbit", after)
	}
	paramsMust(t, c.Reset(&v))
	if e := out.AppendCStringView(view); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("strlen must reach stale pointer before badbit sentry", e)
	}
}
