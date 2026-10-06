// Source: src/util/mpq.cpp:123-126; src/util/mpz.cpp:1432-1436.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"reflect"
	"strings"
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
		astTestMust(t, text.construct(q, m.budget, &value))
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
		q, b := m.numerals, m.budget
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

func TestASTBitVectorIntegerTextLaterCallKeepsFailedCharacterConstructor(t *testing.T) {
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		m := astTestManager(t)
		q, b := m.numerals, m.budget
		firstValue, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&firstValue, strings.Repeat("9", 1200)))
		laterValue, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&laterValue, "18446744073709551617"))
		base := b.Used() - q.RuntimeOwner().Stats().TotalBytes
		var first, later astBVIntegerText
		b.FailAfter(ordinal)
		body := first.construct(q, b, &firstValue)
		receipt := reflect.ValueOf(&first.invocation).Elem().FieldByName("insertion")
		character := receipt.FieldByName("character")
		reached := false
		if !character.IsNil() {
			scope := character.Elem()
			reached = scope.FieldByName("pending").Bool() && !scope.FieldByName("completed").Bool()
		}
		if want := base + q.RuntimeOwner().Stats().TotalBytes + first.storageBytes(); b.Used() != want {
			t.Fatal("before-lift retained text constructor graph", ordinal, body, b.Used(), want)
		}
		astFloatingPointRawGraphWithBlocks(t, q, nil, nil, &firstValue, &laterValue)
		b.FailAfter(-1)
		if reached {
			t.Cleanup(func() {
				astTestMust(t, later.cleanup(false))
				astTestMust(t, later.cleanup(true))
				astTestMust(t, first.cleanup(false))
				astTestMust(t, first.cleanup(true))
				astTestMust(t, q.CleanupValue(&laterValue))
				astTestMust(t, q.CleanupValue(&firstValue))
			})
			if body == nil {
				t.Fatal("actual failed ctor missing body cause")
			}
			issued := receipt.FieldByName("invocation").Uint()
			characterPointer := character.Pointer()
			originalBody, originalSource, originalOwn := first.invocation.Causes()
			if e = later.construct(q, b, &laterValue); e != nil {
				t.Fatal("source-valid later call borrowed old incomplete char constructor", ordinal, e)
			}
			if receipt.FieldByName("invocation").Uint() != issued || character.Pointer() != characterPointer || !character.Elem().FieldByName("pending").Bool() {
				t.Fatal("later call changed older issued character tuple")
			}
			a, b, c := first.invocation.Causes()
			if a != originalBody || b != originalSource || c != originalOwn {
				t.Fatal("later call rewrote older body/source/own history")
			}
			astFloatingPointRawGraphWithBlocks(t, q, nil, nil, &firstValue, &laterValue)
			if !receipt.FieldByName("active").Bool() {
				t.Fatal("later call erased actual older child")
			}
			return
		}
		astTestMust(t, first.cleanup(false))
		astTestMust(t, first.cleanup(true))
		astTestMust(t, q.CleanupValue(&laterValue))
		astTestMust(t, q.CleanupValue(&firstValue))
		astTestMust(t, m.Close())
	}
	t.Fatal("did not reach actual retained character constructor")
}
