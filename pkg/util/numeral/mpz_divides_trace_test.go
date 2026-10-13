package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerDividesRetainedShortAndBugTrace(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("divides")
	tags.Enable("divides_bug")
	compiled := tags.ShouldTrace("divides")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		a, b int32
		want bool
	}{
		{3, 12, true},
		{0, 5, false},
		{0, 0, true},
	} {
		a, b := NewInt(test.a), NewInt(test.b)
		got, err := m.Divides(&a, &b)
		if err != nil || got != test.want {
			t.Fatal(test, got, err)
		}
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased STRACE/TRACE emitted", output.String())
		}
		return
	}
	header := "-------- [divides_bug] divides src/util/mpz.cpp:2068 ---------\n"
	footer := "------------------------------------------------\n"
	want := "[mpz] Divisible[12, 3] == True\n" + header + "tmp: 0\n" + footer +
		"[mpz] Divisible[5, 0] == False\n" + header + "tmp: 0\n" + footer +
		"[mpz] Divisible[0, 0] == True\n" + header + "tmp: 0\n" + footer
	if output.String() != want {
		t.Fatal("source trace order or body", output.String())
	}
}
