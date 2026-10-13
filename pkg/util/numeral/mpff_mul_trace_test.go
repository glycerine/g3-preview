package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestFastFloatMulOuterAndInnerTraceOrder(t *testing.T) {
	m := fastManager(t, 2)
	zero, two, three, minusTwo, out := NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat()
	for _, assignment := range []struct {
		value *FastFloat
		word  int32
	}{{&zero, 0}, {&two, 2}, {&three, 3}, {&minusTwo, -2}} {
		if err := m.SetInt32(assignment.value, assignment.word); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff_trace")
	tags.Enable("mpff")
	compiled := tags.ShouldTrace("mpff_trace")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	header := func(line string) string { return "-------- [mpff] mul src/util/mpff.cpp:" + line + " ---------\n" }
	footer := "------------------------------------------------\n"
	for _, test := range []struct {
		name     string
		a, b     *FastFloat
		up       bool
		want     string
		wantText string
	}{
		{"positive", &two, &three, true,
			"[mpff] (2) * (3) <= " + header("788") + "mul(2, 3)\n" + footer +
				header("800") + "num_leading_zeros: 1\n" + footer +
				header("811") + "c.m_sign: 0, m_to_plus_inf: 1\nhas_one_at_first_k_bits: 0\n_inc_significand: 0\n" + footer +
				header("821") + "result: 6\n" + footer + "6\n", "6"},
		{"negative", &minusTwo, &three, false,
			"[mpff] (-2) * (3) >= " + header("788") + "mul(-2, 3)\n" + footer +
				header("800") + "num_leading_zeros: 1\n" + footer +
				header("811") + "c.m_sign: 1, m_to_plus_inf: 0\nhas_one_at_first_k_bits: 0\n_inc_significand: 0\n" + footer +
				header("821") + "result: -6\n" + footer + "-6\n", "-6"},
		{"zero", &zero, &three, true, "[mpff] (0) * (3) <= 0\n", "0"},
	} {
		output.Reset()
		if err := m.SetRounding(test.up); err != nil {
			t.Fatal(err)
		}
		if err := m.Mul(test.a, test.b, &out); err != nil {
			t.Fatal(test.name, err)
		}
		if got, err := m.String(&out); err != nil || got != test.wantText {
			t.Fatal("product", test.name, got, err)
		}
		want := test.want
		if !compiled {
			want = ""
		}
		if output.String() != want {
			t.Fatal("source mul trace", test.name, output.String())
		}
	}
	for _, value := range []*FastFloat{&zero, &two, &three, &minusTwo, &out} {
		if err := m.Del(value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFastFloatMulTraceReachedImpreciseRounding(t *testing.T) {
	m := fastManager(t, 2)
	a, out := NewFastFloat(), NewFastFloat()
	if err := m.SetFraction32(&a, 1, 3); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpff")
	compiled := tags.ShouldTrace("mpff")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, up := range []bool{true, false} {
		output.Reset()
		if err := m.SetRounding(up); err != nil {
			t.Fatal(err)
		}
		if err := m.Mul(&a, &a, &out); err != nil {
			t.Fatal(err)
		}
		if !compiled {
			if output.Len() != 0 {
				t.Fatal("erased rounding trace emitted", output.String())
			}
			continue
		}
		want := "c.m_sign: 0, m_to_plus_inf: 0\nhas_one_at_first_k_bits: 1\n_inc_significand: 0\n"
		if up {
			want = "c.m_sign: 0, m_to_plus_inf: 1\nhas_one_at_first_k_bits: 1\n_inc_significand: 1\n"
		}
		if !strings.Contains(output.String(), want) {
			t.Fatal("reached rounding decision", up, output.String())
		}
	}
	for _, value := range []*FastFloat{&a, &out} {
		if err := m.Del(value); err != nil {
			t.Fatal(err)
		}
	}
}
