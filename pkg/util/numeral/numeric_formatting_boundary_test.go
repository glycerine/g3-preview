package numeral

import "testing"

func TestNumericMagnitudeBorrowKeepsCharacterAndLimbLifetimesSeparate(t *testing.T) {
	z, _ := integerManager(t)
	var a Int
	if e := z.SetString(&a, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	magnitude, e := z.BorrowMagnitude(&a)
	if e != nil {
		t.Fatal(e)
	}
	characters := make([]byte, 11*magnitude.Size())
	var limbs NaturalFormattingScope
	if e := magnitude.WriteCString(characters, &limbs); e != nil {
		t.Fatal(e)
	}
	var text []byte
	for i := uint64(0); i < 100; i++ {
		b, e := characterByte(characters, i)
		if e != nil {
			t.Fatal(e)
		}
		if b == 0 {
			break
		}
		text = append(text, b)
	}
	if string(text) != "18446744073709551617" {
		t.Fatal("source magnitude", string(text))
	}
	if limbs.Pending() {
		t.Fatal("completed limb destructors retained pending state")
	}
	if e := limbs.CleanupAfterSource(z.Naturals()); e != nil {
		t.Fatal(e)
	}
	if e := z.Del(&a); e != nil {
		t.Fatal(e)
	}
}
