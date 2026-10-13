package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalBinaryShortTraceOuterAndRatBranches(t *testing.T) {
	for _, synchronized := range []bool{false, true} {
		t.Run(map[bool]string{false: "cached", true: "synchronized"}[synchronized], func(t *testing.T) {
			m, _ := rationalManager(t, synchronized)
			a, b, out := NewRational(), NewRational(), NewRational()
			if err := m.SetInt64(&a, 2); err != nil {
				t.Fatal(err)
			}
			if err := m.SetInt64(&b, 3); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("mpq")
			tags.Enable("rat_mpq")
			compiled := tags.ShouldTrace("mpq")
			if err := m.BindTrace(tags); err != nil {
				t.Fatal(err)
			}
			if err := m.Add(&a, &b, &out); err != nil || rationalString(t, m, &out) != "5" {
				t.Fatal("integer rational branch", err)
			}
			if err := m.SetFraction(&a, 1, 2); err != nil {
				t.Fatal(err)
			}
			if err := m.SetFraction(&b, 1, 3); err != nil {
				t.Fatal(err)
			}
			if err := m.Add(&a, &b, &out); err != nil || rationalString(t, m, &out) != "5/6" {
				t.Fatal("rational add", err)
			}
			if err := m.Sub(&a, &b, &out); err != nil || rationalString(t, m, &out) != "1/6" {
				t.Fatal("rational sub", err)
			}
			if err := m.Mul(&a, &b, &out); err != nil || rationalString(t, m, &out) != "1/6" {
				t.Fatal("rational mul", err)
			}
			for _, q := range []*Rational{&a, &b, &out} {
				if err := m.Del(q); err != nil {
					t.Fatal(err)
				}
			}
			if !compiled {
				if output.Len() != 0 {
					t.Fatal("erased rational STRACE emitted", output.String())
				}
				return
			}
			want := "[mpq] 2 + 3 == 5\n" +
				"[mpq] 1/2 + 1/3 == [mpq] 1/2 + 1/3 == 5/6\n5/6\n" +
				"[mpq] 1/2 - 1/3 == [mpq] 1/2 - 1/3 == 1/6\n1/6\n" +
				"[mpq] 1/2 * 1/3 == [mpq] 1/2 * 1/3 == 1/6\n1/6\n"
			if output.String() != want {
				t.Fatal("source mpq/rat_mpq trace nesting", output.String())
			}
		})
	}
}

func TestRationalTraceSharesNestedIntegerStream(t *testing.T) {
	m, _ := rationalManager(t, false)
	a, b, out := NewRational(), NewRational(), NewRational()
	if err := m.SetFraction(&a, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.SetFraction(&b, 1, 3); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"mpq", "rat_mpq", "mpz"} {
		tags.Enable(tag)
	}
	compiled := tags.ShouldTrace("mpq")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.Add(&a, &b, &out); err != nil || rationalString(t, m, &out) != "5/6" {
		t.Fatal("cross-product add", err)
	}
	for _, q := range []*Rational{&a, &b, &out} {
		if err := m.Del(q); err != nil {
			t.Fatal(err)
		}
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased nested traces emitted", output.String())
		}
		return
	}
	want := "[mpq] 1/2 + 1/3 == " +
		"[mpq] 1/2 + 1/3 == " +
		"[mpz] 1 * 3 == 3\n" +
		"[mpz] 1 * 2 == 2\n" +
		"[mpz] 2 * 3 == 6\n" +
		"[mpz] 3 + 2 == 5\n" +
		"5/6\n5/6\n"
	if output.String() != want {
		t.Fatal("inherited integer/rational trace stream", output.String())
	}
}
