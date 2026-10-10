package sat

import (
	utext "github.com/glycerine/g3/pkg/util/text"
	"math"
	"testing"
)

func TestSATJustificationSourceAllKindsAndRawGetters(t *testing.T) {
	none := satJustification{}
	if !none.None() {
		t.Fatal(none)
	}
	b := newSATBinaryJustification(3)
	if !b.Binary() {
		t.Fatal(b)
	}
	if l, e := b.Literal(); l != 3 || e != nil {
		t.Fatal(l, e)
	}
	ter := newSATTernaryJustification(8, Literal(math.MaxUint32))
	if l, e := ter.Literal2(); l != Literal(0x1fffffff) || e != nil {
		t.Fatal(l, e)
	}
	ext := newSATExternalJustification(88)
	if ext.ExternalIndex() != 88 || !ext.External() {
		t.Fatal(ext)
	}
	if b.ExternalIndex() != 3 {
		t.Fatal("source has no kind assert")
	}
	a := satStore(t, nil)
	r := satMk(t, a, satLs(0, 2), false)
	off, e := a.Offset(r)
	satOK(t, e)
	c := newSATClauseJustification(off)
	if !c.Clause() || c.ClauseOffset() != off {
		t.Fatal(c)
	}
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{})
	satOK(t, e)
	defer out.Close()
	for i, j := range []satJustification{none, b, ter, c, ext} {
		if i > 0 {
			satOK(t, out.AppendCString("|"))
		}
		satOK(t, j.DisplayInto(out))
	}
	text, _ := out.String()
	if text != "none|binary -1|ternary 4 -268435455|clause|external" {
		t.Fatal(text)
	}
}
