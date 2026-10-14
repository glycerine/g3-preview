// Source: src/util/params.cpp:550-595,960-1035.
package params

import (
	"testing"
)

func TestParamsReferenceCopyAllKindsSharedTargetOrderedMerge(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	shadow, e := c.NewValue()
	paramsMust(t, e)
	b := internSymbol(t, sy, "b")
	u := internSymbol(t, sy, "u")
	d := internSymbol(t, sy, "d")
	s := internSymbol(t, sy, "s")
	z := internSymbol(t, sy, "z")
	r := internSymbol(t, sy, "r")
	old := internSymbol(t, sy, "old")
	sym := internSymbol(t, sy, "sym")
	paramsMust(t, c.SetBool(&src, b, true))
	paramsMust(t, c.SetUInt(&src, u, 7))
	paramsMust(t, c.SetDouble(&src, d, 1.5))
	text := "hello"
	paramsMust(t, c.SetString(&src, s, &text))
	paramsMust(t, c.SetSymbol(&src, z, sym))
	value, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, q.SetFractionSigned(&value, -7, 3))
	paramsMust(t, c.SetRat(&src, r, &value))
	paramsMust(t, q.Del(&value))
	paramsMust(t, c.SetUInt(&dst, u, 99))
	before := "before"
	paramsMust(t, c.SetString(&dst, old, &before))
	paramsMust(t, c.Assign(&shadow, &dst))
	paramsMust(t, c.Copy(&dst, &src))
	paramsMust(t, c.Del(&src))
	got, e := c.Display(&dst)
	paramsMust(t, e)
	if want := "(params u 7 old before b true d 1.5 s hello z sym r -7/3)"; got != want {
		t.Fatalf("source ordered merge: %q", got)
	}
	unchanged, e := c.Display(&shadow)
	paramsMust(t, e)
	if unchanged != "(params u 99 old before)" {
		t.Fatalf("shared predecessor changed: %q", unchanged)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&shadow))
}
