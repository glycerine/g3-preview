package numeral

import (
	"fmt"
	"math/big"
	"testing"
)

func TestBinarySourceStringRawShapeAndScalarPower(t *testing.T) {
	m, _ := binaryManager(t)
	for _, f := range []struct {
		n    int32
		k    uint32
		want string
	}{{0, 0, "0"}, {-7, 0, "-7"}, {-3, 1, "-3/2"}, {-3, 4, "-3/2^4"}, {2, 1, "2/2"}, {1, ^uint32(0), "1/2^4294967295"}} {
		t.Run(fmt.Sprint(f.n, "/", f.k), func(t *testing.T) {
			a := NewBinaryIntPower(f.n, f.k)
			original := a
			var receipt NumeralInsertionReceipt
			text, e := m.SourceStringTracked(&a, &receipt)
			serviceMust(t, e)
			if text != f.want || a != original {
				t.Fatal("source raw spelling/unchanged input", text, f.want)
			}
			serviceMust(t, m.CleanupSourceString(&receipt))
			if receipt.Active() {
				t.Fatal("completed receipt survived cleanup")
			}
		})
	}
}

// Source formatter buffers are native Go storage, not numeral-budget owners.
// A closed numerical reservation gate must not invent an allocation failure.
func TestBinarySourceStringLargeZeroNumeralBudget(t *testing.T) {
	z, budget := mpzPublicationDirectOwner(t)
	d, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, d.Close()) })
	m, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: d})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	a := NewBinaryIntPower(0, 17)
	words := make([]uint32, 300)
	words[0], words[len(words)-1] = 17, 1
	serviceMust(t, z.SetLimbs(a.BorrowNumerator(), words))
	serviceMust(t, z.Neg(a.BorrowNumerator()))
	t.Cleanup(func() { serviceMust(t, m.Del(&a)) })
	want := new(big.Int).Lsh(big.NewInt(1), 32*299)
	want.Add(want, big.NewInt(17))
	want.Neg(want)
	expected := want.String()
	original, used := a, budget.Used()
	var receipt NumeralInsertionReceipt
	budget.FailAfter(0)
	text, body := m.SourceStringTracked(&a, &receipt)
	budget.FailAfter(-1)
	serviceMust(t, body)
	if text != expected+"/2^17" || a != original || budget.Used() != used {
		t.Fatal("source spelling, unchanged input and native buffer ownership")
	}
	serviceMust(t, m.CleanupSourceString(&receipt))
	if receipt.Active() {
		t.Fatal("formatting child survived cleanup")
	}
	value, e := z.String(a.BorrowNumerator())
	serviceMust(t, e)
	if value != expected {
		t.Fatal("formatting changed the input numeral")
	}
}
