package numeral

import (
	"bytes"
	"io"
	"testing"
)

func TestMPNDisplayRawSourceLimbOrder(t *testing.T) {
	var m NaturalManager
	for _, tc := range []struct {
		limbs []uint32
		want  string
	}{
		{nil, "[]"},
		{[]uint32{0}, "[0]"},
		{[]uint32{7, 0, ^uint32(0)}, "[4294967295|0|7]"},
	} {
		var out bytes.Buffer
		if e := m.DisplayRaw(&out, tc.limbs); e != nil || out.String() != tc.want {
			t.Fatal(tc.limbs, out.String(), e, tc.want)
		}
	}
}

func TestMPNDisplayRawSourceInsertionRefusals(t *testing.T) {
	var m NaturalManager
	testNumeralDisplayParts(t,
		func(w io.Writer) error { return m.DisplayRaw(w, []uint32{7, 0, ^uint32(0)}) },
		"[4294967295|0|7]",
		[]string{"[", "4294967295", "|", "0", "|", "7", "]"})
}
