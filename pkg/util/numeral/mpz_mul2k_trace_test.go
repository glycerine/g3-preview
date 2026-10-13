package numeral

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerMul2kSourceTraceCapacityAndShortcuts(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz_mul2k")
	compiled := tags.ShouldTrace("mpz_mul2k")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	record := func(line int, body string) string {
		return fmt.Sprintf("-------- [mpz_mul2k] mul2k src/util/mpz.cpp:%d ---------\n", line) +
			body + "------------------------------------------------\n"
	}
	zero := NewInt(0)
	if err := m.Mul2kInPlace(&zero, 96); err != nil {
		t.Fatal(err)
	}
	small := NewInt(3)
	if err := m.Mul2kInPlace(&small, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Mul2kInPlace(&small, 2); err != nil || integerString(t, m, &small) != "12" {
		t.Fatal("small shortcut", err)
	}
	first := NewInt(1)
	if err := m.Mul2kInPlace(&first, 32); err != nil || integerString(t, m, &first) != "4294967296" {
		t.Fatal("small promotion", err)
	}
	want := record(1703, "mul2k\na: 1\nk: 32\n") +
		record(1709, "word_shift: 1\nbit_shift: 0\nold_sz: 1\nnew_sz: 3\na after ensure capacity:\n1\n") +
		record(1747, "mul2k result:\n4294967296\n")
	second := NewInt(0)
	if err := m.SetString(&second, "4294967296"); err != nil {
		t.Fatal(err)
	}
	if err := m.Mul2kInPlace(&second, 1); err != nil || integerString(t, m, &second) != "8589934592" {
		t.Fatal("existing big storage", err)
	}
	want += record(1703, "mul2k\na: 4294967296\nk: 1\n") +
		record(1709, "word_shift: 0\nbit_shift: 1\nold_sz: 2\nnew_sz: 3\na after ensure capacity:\n4294967296\n") +
		record(1747, "mul2k result:\n8589934592\n")
	if err := m.Del(&first); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&second); err != nil {
		t.Fatal(err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased shift TRACE emitted", output.String())
		}
	} else if got := output.String(); got != want {
		t.Fatalf("source mul2k TRACE\ngot: %q\nwant: %q", got, want)
	}
}
