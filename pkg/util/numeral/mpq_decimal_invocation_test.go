package numeral

import (
	"bytes"
	"testing"
)

func TestRationalDecimalInvocationOldReceiptCannotAcquireReplacement(t *testing.T) {
	m, _ := rationalManager(t, false)
	oldValue := NewRational()
	bigValue := NewRational()
	var old, current RationalDecimalInvocationReceipt
	defer func() {
		m.BudgetField.FailAfter(-1)
		if e := m.CleanupDecimalInvocation(&current); e != nil {
			t.Error(e)
		}
		if e := m.Del(&bigValue); e != nil {
			t.Error(e)
		}
		if e := m.Del(&oldValue); e != nil {
			t.Error(e)
		}
	}()
	if e := m.SetInt32(&oldValue, 1); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e := m.DisplayDecimalIntoWithInvocation(numeralDisplayStream{&out}, &oldValue, 10, false, &old); e != nil {
		t.Fatal(e)
	}
	if !m.MatchesDecimalInvocation(&old) || m.DecimalInvocationPending(&old) {
		t.Fatal("completed actual invocation missing")
	}
	if e := m.Z.SetString(bigValue.BorrowNumerator(), "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	m.BudgetField.FailAfter(0)
	body := m.DisplayDecimalIntoWithInvocation(numeralDisplayStream{&out}, &bigValue, 10, false, &current)
	m.BudgetField.FailAfter(-1)
	if body == nil || !m.DecimalInvocationPending(&current) || m.MatchesDecimalInvocation(&old) {
		t.Fatal("actual replacement refusal absent", body)
	}
	before := m.decimalStream
	if e := m.CleanupDecimalInvocation(&old); e != nil {
		t.Fatal(e)
	}
	if !m.DecimalInvocationPending(&current) || m.decimalStream.invocation != before.invocation || m.decimalStream.count != before.count {
		t.Fatal("stale cleanup acquired replacement")
	}
	if e := m.CleanupDecimalInvocation(&current); e != nil {
		t.Fatal(e)
	}
	if m.DecimalInvocationPending(&current) {
		t.Fatal("matched cleanup retained decimal")
	}
}
