package arith

// Source: src/tactic/arith/arith_bounds_tactic.cpp bound subsumption.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestArithmeticBoundsKeepStrongestBounds(t *testing.T) {
	_, manager := fixDLFixture(t, false)
	x := fixDLConst(t, manager, true, "arith-bounds-x")
	one := fixDLNumber(t, manager, true, "1")
	two := fixDLNumber(t, manager, true, "2")
	three := fixDLNumber(t, manager, true, "3")
	upperThree := fixDLApp(t, manager, 2, x, three)
	upperTwo := fixDLApp(t, manager, 2, x, two)
	strictUpperTwo := fixDLApp(t, manager, 4, x, two)
	lowerOne := fixDLApp(t, manager, 2, one, x)
	lowerTwo := fixDLApp(t, manager, 3, x, two)
	notLowerThree := fixDLApp(t, manager, 4, three, x)
	notLowerThree, err := manager.MkNot(notLowerThree)
	equationMust(t, err)
	boolean := fixDLConst(t, manager, true, "arith-bounds-unrelated")
	formulas := []ast.ASTHandle{upperThree, upperTwo, strictUpperTwo, lowerOne, lowerTwo, notLowerThree, boolean}
	keep, err := ArithmeticBoundsKeep(manager, formulas)
	equationMust(t, err)
	want := []bool{false, false, true, false, true, false, true}
	for i := range want {
		if keep[i] != want[i] {
			t.Fatalf("bound %d keep %v, want %v", i, keep[i], want[i])
		}
	}
}
