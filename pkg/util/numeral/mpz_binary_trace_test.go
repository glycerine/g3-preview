package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerBinaryShortTraceAliasAndNestedAddMul(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz")
	compiled := tags.ShouldTrace("mpz")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a, b := NewInt(4), NewInt(7)
	if err := m.Add(&a, &b, &a); err != nil || a.Val != 11 {
		t.Fatal("aliased add", a, err)
	}
	if err := m.Sub(&a, &b, &a); err != nil || a.Val != 4 {
		t.Fatal("aliased sub", a, err)
	}
	if err := m.Mul(&a, &b, &a); err != nil || a.Val != 28 {
		t.Fatal("aliased mul", a, err)
	}
	c, d, out := NewInt(2), NewInt(3), NewInt(0)
	if err := m.AddMul(&a, &c, &d, &out); err != nil || out.Val != 34 {
		t.Fatal("nested mul/add", out, err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased binary STRACE emitted", output.String())
		}
		return
	}
	want := "[mpz] 4 + 7 == 11\n" +
		"[mpz] 11 - 7 == 4\n" +
		"[mpz] 4 * 7 == 28\n" +
		"[mpz] 2 * 3 == 6\n" +
		"[mpz] 28 + 6 == 34\n"
	if output.String() != want {
		t.Fatal("source binary trace sequence", output.String())
	}
}

func TestIntegerBinaryShortTraceInsideStringSet(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz")
	compiled := tags.ShouldTrace("mpz")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	var out Int
	if err := m.SetString(&out, "12"); err != nil || integerString(t, m, &out) != "12" {
		t.Fatal("nested source parser", err)
	}
	if err := m.Del(&out); err != nil {
		t.Fatal(err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased parser trace emitted", output.String())
		}
		return
	}
	want := "[mpz] 0 * 10 == 0\n" +
		"[mpz] 0 + 1 == 1\n" +
		"[mpz] 1 * 10 == 10\n" +
		"[mpz] 10 + 2 == 12\n"
	if output.String() != want {
		t.Fatal("source parser call trace order", output.String())
	}
}
