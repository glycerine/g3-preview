package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalMixedAndDivisionShortTraceSourceBranches(t *testing.T) {
	for _, synchronized := range []bool{false, true} {
		t.Run(map[bool]string{false: "cached", true: "synchronized"}[synchronized], func(t *testing.T) {
			m, _ := rationalManager(t, synchronized)
			a, b, out := NewRational(), NewRational(), NewRational()
			if err := m.SetFraction(&a, 1, 2); err != nil {
				t.Fatal(err)
			}
			if err := m.SetFraction(&b, 1, 2); err != nil {
				t.Fatal(err)
			}
			z := NewInt(2)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("mpq")
			tags.Enable("rat_mpq")
			compiled := tags.ShouldTrace("mpq")
			if err := m.BindTrace(tags); err != nil {
				t.Fatal(err)
			}
			if err := m.AddInt(&a, &z, &out); err != nil || rationalString(t, m, &out) != "5/2" {
				t.Fatal("mixed add", err)
			}
			if err := m.MulInt(&z, &a, &out); err != nil || rationalString(t, m, &out) != "1" {
				t.Fatal("mixed mul", err)
			}
			if err := m.DivInt(&a, &z, &out); err != nil || rationalString(t, m, &out) != "1/4" {
				t.Fatal("mixed div", err)
			}
			if err := m.AccDiv(&a, &z); err != nil || rationalString(t, m, &a) != "1/4" {
				t.Fatal("accumulating div", err)
			}
			if err := m.Div(&a, &b, &out); err != nil || rationalString(t, m, &out) != "1/2" {
				t.Fatal("separate rational div", err)
			}
			if err := m.Div(&a, &b, &b); err != nil || rationalString(t, m, &b) != "1/2" {
				t.Fatal("aliased rational div", err)
			}
			if err := m.SetInt64(&a, 3); err != nil {
				t.Fatal(err)
			}
			if err := m.AddInt(&a, &z, &out); err != nil || rationalString(t, m, &out) != "5" {
				t.Fatal("integer mixed add", err)
			}
			if err := m.MulInt(&z, &a, &out); err != nil || rationalString(t, m, &out) != "6" {
				t.Fatal("integer mixed mul", err)
			}
			for _, q := range []*Rational{&a, &b, &out} {
				if err := m.Del(q); err != nil {
					t.Fatal(err)
				}
			}
			if !compiled {
				if output.Len() != 0 {
					t.Fatal("erased mixed STRACE emitted", output.String())
				}
				return
			}
			want := "[mpq] 1/2 + 2 == [mpq] 1/2 + 2 == 5/2\n5/2\n" +
				"[mpq] 2 * 1/2 == [mpq] 2 * 1/2 == 1\n1\n" +
				"[mpq] 1/2 / 2 == 1/4\n" +
				"[mpq] 1/2 / 2 == 1/4\n" +
				"[mpq] 1/4 / 1/2 == 1/2\n" +
				"[mpq] 1/4 / 1/2 == 1/2\n" +
				"[mpq] 3 + 2 == 5\n" +
				"[mpq] 2 * 3 == 6\n"
			if output.String() != want {
				t.Fatal("source mixed/division trace sequence", output.String())
			}
		})
	}
}
