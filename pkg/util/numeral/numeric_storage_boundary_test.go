package numeral

import "testing"

func TestNumericNativeWordsBorrowOwnerAndRejectReleasedCell(t *testing.T) {
	z, _ := integerManager(t)
	for _, value := range []int64{-1 << 62, -17, 0, 31, 1 << 62} {
		var original Int
		if e := z.SetInt64(&original, value); e != nil {
			t.Fatal(e)
		}
		borrowed := IntFromNativeWords(original.NativeWords())
		if borrowed != original {
			t.Fatal("native representation changed", value)
		}
		got, e := z.Int64(&borrowed)
		if e != nil || got != value {
			t.Fatal("decoded value", value, got, e)
		}
		big := !original.IsSmall()
		if e := z.Del(&original); e != nil {
			t.Fatal(e)
		}
		if big {
			if _, e := z.Int64(&borrowed); e == nil {
				t.Fatal("decoded numeral outlived owning cell", value)
			}
		}
	}
}
