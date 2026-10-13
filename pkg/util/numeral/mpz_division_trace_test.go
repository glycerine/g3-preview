package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerDivisionShortTraceNestedSourceOrder(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz")
	compiled := tags.ShouldTrace("mpz")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a, b, q, r := NewInt(-7), NewInt(3), NewInt(0), NewInt(0)
	if err := m.MachineDivRem(&a, &b, &q, &r); err != nil || q.Val != -2 || r.Val != -1 {
		t.Fatal("truncating pair", q, r, err)
	}
	if err := m.MachineDiv(&a, &b, &q); err != nil || q.Val != -2 {
		t.Fatal("truncating quotient", q, err)
	}
	if err := m.Rem(&a, &b, &r); err != nil || r.Val != -1 {
		t.Fatal("signed remainder", r, err)
	}
	if err := m.Div(&a, &b, &q); err != nil || q.Val != -3 {
		t.Fatal("Euclidean quotient", q, err)
	}
	if err := m.Mod(&a, &b, &r); err != nil || r.Val != 2 {
		t.Fatal("Euclidean modulus", r, err)
	}
	positive := NewInt(7)
	if err := m.Div(&positive, &b, &q); err != nil || q.Val != 2 {
		t.Fatal("positive quotient", q, err)
	}
	shared := NewInt(0)
	if err := m.MachineDivRem(&positive, &b, &shared, &shared); err != nil || shared.Val != 1 {
		t.Fatal("aliased pair publication", shared, err)
	}
	zero := NewInt(0)
	if err := m.Rem(&positive, &zero, &r); !errors.Is(err, ErrDivideByZero) {
		t.Fatal("source prefix before refused division", err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased division STRACE emitted", output.String())
		}
		return
	}
	want := "[mpz-ext] divrem(-7,  3) == (-2, -1)\n" +
		"[mpz-ext] machine-div(-7,  3) == -2\n" +
		"[mpz-ext] rem(-7,  3) == -1\n" +
		"[mpz-ext] div(-7,  3) == " +
		"[mpz-ext] divrem(-7,  3) == (-2, -1)\n" +
		"[mpz] -2 - 1 == -3\n" +
		"-3\n" +
		"[mpz-ext] mod(-7,  3) == " +
		"[mpz-ext] rem(-7,  3) == -1\n" +
		"[mpz] -1 + 3 == 2\n" +
		"2\n" +
		"[mpz-ext] div(7,  3) == " +
		"[mpz-ext] machine-div(7,  3) == 2\n" +
		"2\n" +
		"[mpz-ext] divrem(7,  3) == (1, 1)\n" +
		"[mpz-ext] rem(7,  0) == "
	if output.String() != want {
		t.Fatal("source division trace order and early refusal", output.String())
	}
}
