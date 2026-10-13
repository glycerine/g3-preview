package numeral

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerBitwiseOrAndRootRetainedTraceSites(t *testing.T) {
	m, _ := integerManager(t)
	big := NewInt(0)
	if err := m.SetString(&big, "4294967296"); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpz")
	tags.Enable("mpz_gcd")
	compiled := tags.ShouldTrace("mpz")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	smallA, smallB, result := NewInt(2), NewInt(3), NewInt(0)
	if err := m.BitwiseOr(&smallA, &smallB, &result); err != nil || integerString(t, m, &result) != "3" {
		t.Fatal("small OR", err)
	}
	one := NewInt(1)
	if err := m.BitwiseOr(&big, &one, &result); err != nil || integerString(t, m, &result) != "4294967297" {
		t.Fatal("big OR", err)
	}
	root := NewInt(9)
	exact, err := m.RootInPlace(&root, 2)
	if err != nil || !exact || integerString(t, m, &root) != "3" {
		t.Fatal("root bisection", exact, err)
	}
	if err := m.Del(&big); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&result); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&root); err != nil {
		t.Fatal(err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased MPZ TRACE emitted", output.String())
		}
		return
	}
	got := output.String()
	if strings.Contains(got, "[mpz_gcd]") {
		t.Fatal("disabled source GCD optimization traced", got)
	}
	record := func(name string, line int, body string) string {
		return fmt.Sprintf("-------- [mpz] %s src/util/mpz.cpp:%d ---------\n", name, line) +
			body + "------------------------------------------------\n"
	}
	for _, part := range []string{
		record("bitwise_or", 1021, "is_small(a): 1, is_small(b): 1\n"),
		record("bitwise_or", 1021, "is_small(a): 0, is_small(b): 1\n"),
		record("bitwise_or", 1034, "a1: 4294967296, b1: 1\n"),
		record("bitwise_or", 1037, "a2: 4294967296, b2: 1\n"),
		record("bitwise_or", 1039, "uint(a2): 4294967296, uint(b2): 1\n"),
		record("root", 1987, "upper: 4\nlower: 2\nmid: 3\n"),
	} {
		i := strings.Index(got, part)
		if i < 0 {
			t.Fatalf("missing or out-of-order source TRACE %q in %q", part, output.String())
		}
		got = got[i+len(part):]
	}
}
