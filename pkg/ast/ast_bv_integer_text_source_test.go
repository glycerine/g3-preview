// Source: src/util/mpq.cpp:123-126; src/util/mpz.cpp:1432-1436.
package ast

import (
	"testing"
)

func TestASTBitVectorIntegerToStringActualStreamAndReturnedCString(t *testing.T) {
	m := astTestManager(t)
	q := m.numerals
	for _, literal := range []string{"0", "255", "18446744073709551617"} {
		value, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&value, literal))
		var text astBVIntegerText
		astTestMust(t, text.construct(q, m.BudgetField, &value))
		view, e := text.cstring()
		astTestMust(t, e)
		n, e := view.Length()
		astTestMust(t, e)
		if n != uint64(len(literal)) {
			t.Fatal("actual returned string length", n, literal)
		}
		for i := range literal {
			c, e := view.Byte(uint64(i))
			astTestMust(t, e)
			if c != literal[i] {
				t.Fatal("actual returned CString", i, c, literal)
			}
		}
		astTestMust(t, text.sourceDestroy())
		if _, e = view.Byte(0); e == nil {
			t.Fatal("source DT left borrowed c_str live")
		}
		astTestMust(t, text.cleanup(false))
		astTestMust(t, text.cleanup(true))
		astTestMust(t, q.Del(&value))
	}
}

func TestASTBitVectorIntegerTextEveryReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m := astTestManager(t)
		q, b := m.numerals, m.BudgetField
		value, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&value, "18446744073709551617"))
		runtime := q.RuntimeOwner()
		base := b.Used() - runtime.Stats().TotalBytes
		var text astBVIntegerText
		b.FailAfter(ordinal)
		body := text.construct(q, b, &value)
		if want := base + runtime.Stats().TotalBytes + text.storageBytes(); b.Used() != want {
			t.Fatal("before-lift literal text cold owners", ordinal, body, b.Used(), want)
		}
		astFloatingPointRawGraphWithBlocks(t, q, nil, nil, &value)
		b.FailAfter(-1)
		astTestMust(t, text.cleanup(false))
		astTestMust(t, text.cleanup(true))
		astTestMust(t, q.CleanupValue(&value))
		astTestMust(t, m.Close())
		if body == nil {
			t.Log("first returned source string", ordinal)
			return
		}
	}
	t.Fatal("source text reservation horizon")
}
