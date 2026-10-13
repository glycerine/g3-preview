package numeral

import (
	"bytes"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerNegSourceShortTraceAndAbsExclusion(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz")
	compiled := tags.ShouldTrace("mpz")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	a := NewInt(-4)
	if err := m.Abs(&a); err != nil || a.Val != 4 {
		t.Fatal("source abs", a, err)
	}
	if err := m.Neg(&a); err != nil || a.Val != -4 {
		t.Fatal("ordinary neg", a, err)
	}
	minAbs := NewInt(math.MinInt32)
	if err := m.Abs(&minAbs); err != nil || integerString(t, m, &minAbs) != "2147483648" {
		t.Fatal("INT_MIN abs", err)
	}
	if err := m.Del(&minAbs); err != nil {
		t.Fatal(err)
	}
	min := NewInt(math.MinInt32)
	if err := m.Neg(&min); err != nil || integerString(t, m, &min) != "2147483648" {
		t.Fatal("INT_MIN early return", err)
	}
	if err := m.Neg(&min); err != nil || integerString(t, m, &min) != "-2147483648" {
		t.Fatal("big negative", err)
	}
	if err := m.Del(&min); err != nil {
		t.Fatal(err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased neg STRACE emitted", output.String())
		}
		return
	}
	want := "[mpz] 0 - 4 == -4\n" +
		"[mpz] 0 - -2147483648 == " +
		"[mpz] 0 - 2147483648 == -2147483648\n"
	if output.String() != want {
		t.Fatal("source neg trace and INT_MIN early return", output.String())
	}
}
