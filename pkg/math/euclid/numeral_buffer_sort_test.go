package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func euclidNumeralManager(t *testing.T) *numeral.RationalManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	m, e := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: b},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.CloseChecked(); e != nil {
			t.Fatal(e)
		}
		if used := b.Used(); used != 0 {
			t.Fatalf("numeral owner retained %d bytes", used)
		}
	})
	return m
}

func euclidMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestEuclideanIntegerBufferSortAndRawSwap(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	b, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	euclidMust(t, b.Resize(6))
	coeff := []numeral.Int{{}, {}, {}}
	input := []string{"123456789012345678901", "-98765432109876543211", "17"}
	for i := range coeff {
		euclidMust(t, z.SetString(&coeff[i], input[i]))
	}
	ids := []uint32{5, 1, 3}
	euclidMust(t, sortInt(coeff, ids, b))
	want := []string{input[1], input[2], input[0]}
	for i := range coeff {
		got, e := z.String(&coeff[i])
		euclidMust(t, e)
		if ids[i] != []uint32{1, 3, 5}[i] || got != want[i] {
			t.Fatalf("sorted integer coefficient %d = (%d,%s), want (%d,%s)", i, ids[i], got, []uint32{1, 3, 5}[i], want[i])
		}
	}
	borrowed, e := b.BorrowedValues()
	euclidMust(t, e)
	if len(borrowed) != 6 {
		t.Fatal("borrowed source buffer length", len(borrowed))
	}
	euclidMust(t, b.SwapRaw(&coeff))
	for i, want := range want {
		slot, e := b.BorrowSlot(uint32(i))
		euclidMust(t, e)
		got, e := z.String(slot)
		euclidMust(t, e)
		if got != want {
			t.Fatal("raw swap lost sorted coefficient", i, got, want)
		}
	}
	euclidMust(t, b.Close())
	for i := range coeff { // source raw vector now owns old buffer payloads
		euclidMust(t, z.Del(&coeff[i]))
	}
}

func TestEuclideanRationalBufferSortAndRawSwap(t *testing.T) {
	m := euclidNumeralManager(t)
	b, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, b.Resize(5))
	coeff := []numeral.Rational{numeral.NewRational(), numeral.NewRational(), numeral.NewRational()}
	input := []string{"123456789012345678901/7", "-33/11", "5/13"}
	for i := range coeff {
		euclidMust(t, m.SetString(&coeff[i], input[i]))
	}
	ids := []uint32{4, 0, 2}
	euclidMust(t, sortRational(coeff, ids, b))
	want := []string{"-3", "5/13", input[0]}
	for i := range coeff {
		got, e := m.String(&coeff[i])
		euclidMust(t, e)
		if ids[i] != []uint32{0, 2, 4}[i] || got != want[i] {
			t.Fatalf("sorted rational coefficient %d = (%d,%s), want (%d,%s)", i, ids[i], got, []uint32{0, 2, 4}[i], want[i])
		}
	}
	euclidMust(t, b.SwapRaw(&coeff))
	for i, want := range want {
		slot, e := b.BorrowSlot(uint32(i))
		euclidMust(t, e)
		got, e := m.String(slot)
		euclidMust(t, e)
		if got != want {
			t.Fatal("raw rational swap lost sorted coefficient", i, got, want)
		}
	}
	euclidMust(t, b.Close())
	for i := range coeff {
		euclidMust(t, m.Del(&coeff[i]))
	}
}

// mk_eq fills numeral_buffer slots before it calls sort_core. Its coefficient
// array starts with default numerals, so each swap transfers an old array
// value back into the indexed buffer slot.
func TestEuclideanBufferSortCoreUsesPublishedSlots(t *testing.T) {
	t.Run("integer", func(t *testing.T) {
		m := euclidNumeralManager(t)
		z := m.Integers()
		buffer, e := numeral.NewScopedIntVector(z)
		euclidMust(t, e)
		euclidMust(t, buffer.Resize(6))
		ids := []uint32{5, 1, 3}
		for i, value := range []string{"500000000000000000001", "100000000000000000003", "300000000000000000007"} {
			var n numeral.Int
			euclidMust(t, z.SetString(&n, value))
			euclidMust(t, buffer.Set(ids[i], &n))
			euclidMust(t, z.Del(&n))
		}
		coeff := []numeral.Int{{}, {}, {}}
		euclidMust(t, sortIntCore(coeff, ids, buffer))
		for i, want := range []string{"100000000000000000003", "300000000000000000007", "500000000000000000001"} {
			got, e := z.String(&coeff[i])
			euclidMust(t, e)
			if ids[i] != []uint32{1, 3, 5}[i] || got != want {
				t.Fatal(i, ids[i], got, want)
			}
			slot, e := buffer.BorrowSlot(ids[i])
			euclidMust(t, e)
			zero, e := z.IsZero(slot)
			euclidMust(t, e)
			if !zero {
				t.Fatal("old coefficient was not swapped into buffer", i)
			}
			euclidMust(t, z.Del(&coeff[i]))
		}
		euclidMust(t, buffer.Close())
	})
	t.Run("rational", func(t *testing.T) {
		m := euclidNumeralManager(t)
		buffer, e := numeral.NewScopedRationalVector(m)
		euclidMust(t, e)
		euclidMust(t, buffer.Resize(6))
		ids := []uint32{5, 1, 3}
		for i, value := range []string{"500000000000000000001/7", "100000000000000000003/11", "300000000000000000007/13"} {
			n := numeral.NewRational()
			euclidMust(t, m.SetString(&n, value))
			euclidMust(t, buffer.Set(ids[i], &n))
			euclidMust(t, m.Del(&n))
		}
		coeff := []numeral.Rational{numeral.NewRational(), numeral.NewRational(), numeral.NewRational()}
		euclidMust(t, sortRationalCore(coeff, ids, buffer))
		for i, want := range []string{"100000000000000000003/11", "300000000000000000007/13", "500000000000000000001/7"} {
			got, e := m.String(&coeff[i])
			euclidMust(t, e)
			if ids[i] != []uint32{1, 3, 5}[i] || got != want {
				t.Fatal(i, ids[i], got, want)
			}
			slot, e := buffer.BorrowSlot(ids[i])
			euclidMust(t, e)
			zero, e := m.IsZero(slot)
			euclidMust(t, e)
			if !zero {
				t.Fatal("old coefficient was not swapped into buffer", i)
			}
			euclidMust(t, m.Del(&coeff[i]))
		}
		euclidMust(t, buffer.Close())
	})
}
