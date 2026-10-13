package numeral

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerBitwiseNotAndMachineDiv2kSourceTraces(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("bitwise_not")
	tags.Enable("mpz_2k")
	compiled := tags.ShouldTrace("bitwise_not")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	record := func(tag, name string, line int, body string) string {
		return fmt.Sprintf("-------- [%s] %s src/util/mpz.cpp:%d ---------\n", tag, name, line) +
			body + "------------------------------------------------\n"
	}
	var want string
	small, smallOut := NewInt(5), NewInt(0)
	if err := m.BitwiseNot(8, &small, &smallOut); err != nil || integerString(t, m, &smallOut) != "250" {
		t.Fatal("small branch", err)
	}
	largeWidth, out := NewInt(1), NewInt(0)
	if err := m.BitwiseNot(64, &largeWidth, &out); err != nil || integerString(t, m, &out) != "18446744073709551614" {
		t.Fatal("one chunk", err)
	}
	want += record("bitwise_not", "bitwise_not", 1171, "sz: 64, v: 18446744073709551614, n: 1\n")
	want += record("bitwise_not", "bitwise_not", 1181, "sz: 0 a: 1 c: 18446744073709551614\n")
	if err := m.Del(&out); err != nil {
		t.Fatal(err)
	}
	alias := NewInt(1)
	if err := m.BitwiseNot(65, &alias, &alias); err != nil || integerString(t, m, &alias) != "36893488147419103230" {
		t.Fatal("two chunks and aliased destination", err)
	}
	want += record("bitwise_not", "bitwise_not", 1171, "sz: 65, v: 18446744073709551614, n: 1\n")
	want += record("bitwise_not", "bitwise_not", 1171, "sz: 1, v: 1, n: 0\n")
	want += record("bitwise_not", "bitwise_not", 1181, "sz: 0 a: 36893488147419103230 c: 36893488147419103230\n")
	if err := m.Del(&alias); err != nil {
		t.Fatal(err)
	}
	shift := NewInt(0)
	if err := m.SetString(&shift, "4294967296"); err != nil {
		t.Fatal(err)
	}
	if err := m.MachineDiv2kInPlace(&shift, 1); err != nil || integerString(t, m, &shift) != "2147483648" {
		t.Fatal("big shift", err)
	}
	want += record("mpz_2k", "machine_div2k", 1651, "bit_shift: 1, comp_shift: 31, new_sz: 2, sz: 2\n")
	if err := m.MachineDiv2kInPlace(&shift, 32); err != nil || integerString(t, m, &shift) != "0" {
		t.Fatal("digit exhaustion", err)
	}
	if err := m.Del(&shift); err != nil {
		t.Fatal(err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased source traces emitted", output.String())
		}
	} else if got := output.String(); got != want {
		t.Fatalf("source trace order\ngot: %q\nwant: %q", got, want)
	}
}
