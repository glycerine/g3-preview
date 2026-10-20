// Source: src/ast/fpa_decl_plugin.cpp:176-180; src/util/mpf.cpp:316-333.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatDirectCopyFromExplicitForeignOwner(t *testing.T) {
	sourceBudget := arena.NewBudget(1 << 26)
	targetBudget := arena.NewBudget(1 << 26)
	source, err := NewArbitraryFloatManager(serviceConfig(sourceBudget).Integers)
	serviceMust(t, err)
	target, err := NewArbitraryFloatManager(serviceConfig(targetBudget).Integers)
	serviceMust(t, err)
	digits := []uint32{0xffffffff, 0xfffffffe, 0x7fffffff}
	var significand Int
	serviceMust(t, source.integer.SetLimbs(&significand, digits))
	var input, output ArbitraryFloat
	serviceMust(t, source.SetComponents(&input, 11, 113, true, -42, &significand))
	serviceMust(t, target.CopyFromOwner(&output, source, &input))
	if output.EBits() != 11 || output.SBits() != 113 || !output.Sign() || output.Exponent() != -42 {
		t.Fatal("copied scalar", output)
	}
	actual, err := target.BorrowSignificand(&output)
	serviceMust(t, err)
	cell := target.integer.Cell(actual.Cell)
	if cell.Size != uint32(len(digits)) {
		t.Fatal("copied significand size", cell)
	}
	for i, want := range digits {
		if got := target.integer.Digit(actual.Cell, uint32(i)); got != want {
			t.Fatal("copied significand", i, got, want)
		}
	}
	serviceMust(t, target.Del(&output))
	serviceMust(t, source.Del(&input))
	serviceMust(t, source.integer.Del(&significand))
	serviceMust(t, target.Close())
	serviceMust(t, source.Close())
	if sourceBudget.Used() != 0 || targetBudget.Used() != 0 {
		t.Fatal("copy retained storage", sourceBudget.Used(), targetBudget.Used())
	}
}
