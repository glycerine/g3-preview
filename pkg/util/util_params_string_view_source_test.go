// Source: src/util/params.cpp:639-648,902-930; src/smt/params/qi_params.cpp:23-39.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
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
	c.BudgetField.FailAfter(0)
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
	c.BudgetField.FailAfter(-1)
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
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: sb})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, out.SetWidth(7))
	paramsMust(t, (view).AppendTo(out))
	got, e := out.String()
	paramsMust(t, e)
	if got != "   ABCD" {
		t.Fatal("one actual field width and live bytes", got)
	}
	paramsMust(t, (ParamsStringView{}).AppendTo(out))
	paramsMust(t, out.AppendCString("ignored"))
	after, e := out.String()
	paramsMust(t, e)
	if after != got || !out.FileBad {
		t.Fatal("null pointer nonthrowing badbit", after)
	}
	paramsMust(t, c.Reset(&v))
	if e := (view).AppendTo(out); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("strlen must reach stale pointer before badbit sentry", e)
	}
}

func TestParamsNativeStringCopyLifetimeAndReplacement(t *testing.T) {
	base, sy, q := paramsTestOwner(t)
	c, e := NewParamsContext(sy, q, ParamsConfig{Budget: base.BudgetField})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, c.Close()) })
	source, e := c.NewValue()
	paramsMust(t, e)
	copyValue, e := c.NewValue()
	paramsMust(t, e)
	name := "native-copy"
	key, e := sy.Intern(name)
	paramsMust(t, e)
	text := "original"
	paramsMust(t, c.SetString(&source, key, &text))
	// Copy into existing backing exercises the deep merge, rather than the
	// source null-destination shortcut that intentionally shares a block.
	paramsMust(t, c.SetUInt(&copyValue, key, 1))
	paramsMust(t, c.Copy(&copyValue, &source))
	original, e := c.GetStringViewFallbackName(&source, &name, nil, nil)
	paramsMust(t, e)
	copied, e := c.GetStringViewFallbackName(&copyValue, &name, nil, nil)
	paramsMust(t, e)
	if original.owned == copied.owned {
		t.Fatal("copy shared mutable lifetime")
	}
	before := c.BudgetField.Used()
	c.BudgetField.FailAfter(0)
	defer c.BudgetField.FailAfter(-1)
	text = "replacement\x00ignored"
	paramsMust(t, c.SetString(&source, key, &text))
	if c.BudgetField.Used() != before {
		t.Fatal("native replacement consumed arena budget")
	}
	if _, e := original.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("replaced view still valid", e)
	}
	if b, e := copied.Byte(0); e != nil || b != 'o' {
		t.Fatal("replacement invalidated independent copy", b, e)
	}
	updated, e := c.GetStringViewFallbackName(&source, &name, nil, nil)
	paramsMust(t, e)
	if n, e := updated.Length(); e != nil || n != uint64(len("replacement")) {
		t.Fatal("native C-string prefix", n, e)
	}
	paramsMust(t, c.Reset(&source))
	if _, e := updated.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reset did not invalidate view", e)
	}
	if b, e := copied.Byte(0); e != nil || b != 'o' {
		t.Fatal("reset invalidated independent copy", b, e)
	}
	paramsMust(t, c.Close())
	if _, e := copied.Byte(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("close did not invalidate view", e)
	}
}
