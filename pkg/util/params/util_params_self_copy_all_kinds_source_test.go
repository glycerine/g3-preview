package params

// Source: src/util/params.cpp:559-595,598-613. A self-copy first detaches a
// shared block, then copy_core visits the newly published destination.

import "testing"

func TestParamsSelfCopyAllKindsDetachesSharedBlock(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	shadow, e := c.NewValue()
	paramsMust(t, e)
	keyBool := internSymbol(t, sy, "b")
	keyUInt := internSymbol(t, sy, "u")
	keyText := internSymbol(t, sy, "s")
	keySymbol := internSymbol(t, sy, "z")
	keyRat := internSymbol(t, sy, "r")
	spelling := internSymbol(t, sy, "sym")
	paramsMust(t, c.SetBool(&v, keyBool, true))
	paramsMust(t, c.SetUInt(&v, keyUInt, 7))
	text := "hello"
	paramsMust(t, c.SetString(&v, keyText, &text))
	paramsMust(t, c.SetSymbol(&v, keySymbol, spelling))
	r, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, q.SetFractionSigned(&r, -7, 3))
	paramsMust(t, c.SetRat(&v, keyRat, &r))
	paramsMust(t, q.Del(&r))
	paramsMust(t, c.Assign(&shadow, &v))
	oldSlot := v.Slot
	paramsMust(t, c.Copy(&v, &v))
	if v.Slot == oldSlot || shadow.Slot != oldSlot {
		t.Fatal("self-copy did not detach shared source block")
	}
	paramsMust(t, c.Append(&v, &v))
	want := "(params b true u 7 s hello z sym r -7/3)"
	for _, item := range []*ParamsRef{&v, &shadow} {
		got, e := c.Display(item)
		paramsMust(t, e)
		if got != want {
			t.Fatalf("self-copy changed source entries: %q", got)
		}
	}
	updated := "world"
	paramsMust(t, c.SetString(&v, keyText, &updated))
	got, e := c.Display(&shadow)
	paramsMust(t, e)
	if got != want {
		t.Fatalf("shared predecessor changed: %q", got)
	}
	paramsMust(t, c.Del(&v))
	paramsMust(t, c.Del(&shadow))
}
