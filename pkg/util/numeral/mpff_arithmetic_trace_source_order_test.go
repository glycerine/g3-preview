package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpffArithmeticTraceWriter struct{ parts []string }

func (w *mpffArithmeticTraceWriter) Write(p []byte) (int, error) {
	w.parts = append(w.parts, string(p))
	return len(p), nil
}

func hasMPFFTraceParts(parts, want []string) bool {
	for i := 0; i+len(want) <= len(parts); i++ {
		match := true
		for j, part := range want {
			if parts[i+j] != part {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func TestFastFloatArithmeticTraceSourceInsertions(t *testing.T) {
	m := fastManager(t, 2)
	one, two, three, six, out := NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat(), NewFastFloat()
	for _, item := range []struct {
		value *FastFloat
		n     int32
	}{{&one, 1}, {&two, 2}, {&three, 3}, {&six, 6}} {
		if err := m.SetInt32(item.value, item.n); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.SetRounding(true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		run  func() error
		want [][]string
	}{
		{"add", func() error { return m.Add(&one, &two, &out) }, [][]string{
			{" ", "<=", " "}, {"add", "("},
		}},
		{"multiply", func() error { return m.Mul(&two, &three, &out) }, [][]string{
			{") ", "<=", " "}, {"num_leading_zeros: ", "1", "\n"},
			{"c.m_sign: ", "0", ", m_to_plus_inf: ", "1", "\nhas_one_at_first_k_bits: ", "0", "\n", "_inc_significand: ", "0", "\n"},
		}},
		{"divide", func() error { return m.Div(&six, &three, &out) }, [][]string{
			{") ", "<=", " "}, {"00000001", " ", "00000000", " ", "00000000", " ", "\n"},
		}},
	} {
		w := &mpffArithmeticTraceWriter{}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		for _, tag := range []string{"mpff", "mpff_trace", "mpff_div"} {
			tags.Enable(tag)
		}
		if !tags.ShouldTrace("mpff") {
			break
		}
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		if err := tc.run(); err != nil {
			t.Fatal(tc.name, err)
		}
		for _, want := range tc.want {
			if !hasMPFFTraceParts(w.parts, want) {
				t.Fatalf("%s missing source insertions %q in %q", tc.name, want, w.parts)
			}
		}
	}
	for _, value := range []*FastFloat{&one, &two, &three, &six, &out} {
		if err := m.Del(value); err != nil {
			t.Fatal(err)
		}
	}
}
