package params

import (
	"errors"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamsAllocatedEmptyBackingSkipsKeyInspection(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	v, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, c.Del(&v)) })
	key := internSymbol(t, sy, "one")
	invalid := usymbol.Symbol{Tag: 4}
	paramsMust(t, c.SetBool(&v, key, true))
	paramsMust(t, c.ResetKey(&v, key))
	if v.Slot == 0 || c.Blocks[v.Slot-1].Size != 0 {
		t.Fatal("fixture needs allocated empty source backing", v)
	}
	if got, err := c.GetBool(&v, invalid, true); err != nil || !got {
		t.Fatal("bool default read key", got, err)
	}
	if got, err := c.GetUInt(&v, invalid, 17); err != nil || got != 17 {
		t.Fatal("unsigned default read key", got, err)
	}
	if got, err := c.GetDouble(&v, invalid, 2.5); err != nil || got != 2.5 {
		t.Fatal("double default read key", got, err)
	}
	defaultSymbol := usymbol.Numerical(7)
	if got, err := c.GetSymbol(&v, invalid, defaultSymbol); err != nil || got != defaultSymbol {
		t.Fatal("symbol default read key", got, err)
	}
	defaultText := "fallback"
	if got, present, err := c.GetString(&v, invalid, &defaultText); err != nil || !present || got != defaultText {
		t.Fatal("string default read key", got, present, err)
	}
	def, err := q.NewValue()
	paramsMust(t, err)
	gotRat, err := c.GetRat(&v, invalid, &def)
	paramsMust(t, err)
	if same, err := q.Equal(&gotRat, &def); err != nil || !same {
		t.Fatal("rational default read key", same, err)
	}
	paramsMust(t, q.Del(&gotRat))
	paramsMust(t, q.Del(&def))
	if got, err := c.Contains(&v, invalid); err != nil || got {
		t.Fatal("empty contains read key", got, err)
	}
	paramsMust(t, c.ResetKey(&v, invalid))
	paramsMust(t, c.SetBool(&v, key, true))
	if _, err := c.GetBool(&v, invalid, false); !errors.Is(err, usymbol.ErrInvalid) {
		t.Fatal("populated lookup skipped key guard", err)
	}
}
