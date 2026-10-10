package ast

import (
	"strconv"
	"strings"
	"testing"
)

func TestASTSMT2BitVectorNativeBufferLongRadixOracle(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	value, e := m.numerals.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, m.numerals.CleanupValue(&value)) }()
	astTestMust(t, m.numerals.SetUint64(&value, 19))
	for _, width := range []uint32{16, 17, 64, 65, 257, 4096, 4097} {
		h, e := m.BitVectorNumeral(&value, width)
		astTestMust(t, e)
		astTestPin(t, m, h)
		got, e := m.ppBitVectorLiteral(h, false, true, nil)
		astTestMust(t, e)
		astTestPin(t, m.Format, got)
		base, digits, prefix := 2, int(width), "#b"
		if width%4 == 0 {
			base, digits, prefix = 16, int(width/4), "#x"
		}
		tail := strconv.FormatUint(19, base)
		want, e := m.Format.FormatString(prefix + strings.Repeat("0", digits-len(tail)) + tail)
		astTestMust(t, e)
		if got != want {
			t.Fatal("independent padded radix spelling", width, got, want)
		}
	}
}
