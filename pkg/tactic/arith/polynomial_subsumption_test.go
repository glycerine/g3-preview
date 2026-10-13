package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:132-169,256-277;
// src/util/numeral_buffer.h:35-85.
func TestInequalityPolynomialSubsumptionUsesOwnedCoefficientBuffer(t *testing.T) {
	m, sy, q, _ := polynomialASTManager(t)
	owner, _ := equationManager(t)
	store, err := NewBoundEquationStore(owner)
	equationMust(t, err)
	defer func() { equationMust(t, store.Close()) }()
	family, err := m.RegisterArithmeticPlugin()
	equationMust(t, err)
	realSort, err := m.MkFamilySort(family, 0, nil)
	equationMust(t, err)
	xName, err := sy.Intern("subsumption-x")
	equationMust(t, err)
	yName, err := sy.Intern("subsumption-y")
	equationMust(t, err)
	x, err := m.MkConst(xName, realSort)
	equationMust(t, err)
	y, err := m.MkConst(yName, realSort)
	equationMust(t, err)
	sum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{x, y})
	equationMust(t, err)
	builder, err := NewPolynomialBuilder(m, store)
	equationMust(t, err)
	defer func() { equationMust(t, builder.Close()) }()

	k := numeral.NewRational()
	defer func() { equationMust(t, owner.Del(&k)) }()
	set := func(n int32) { t.Helper(); equationMust(t, owner.SetInt32(&k, n)) }
	check := func(lower bool, n int32, want bool, strict bool) {
		t.Helper()
		set(n)
		var got bool
		var e error
		if lower {
			got, e = builder.LowerSubsumed(sum, &k, strict)
		} else {
			got, e = builder.UpperSubsumed(sum, &k, strict)
		}
		equationMust(t, e)
		if got != want {
			t.Fatalf("lower=%v k=%d strict=%v: got %v want %v", lower, n, strict, got, want)
		}
	}
	check(true, 5, false, false) // the first query creates both source variables
	if builder.coeff.Size() != 2 || len(builder.ids) != 2 {
		t.Fatalf("polynomial buffer size = (%d,%d)", builder.coeff.Size(), len(builder.ids))
	}
	if got, e := builder.LowerSubsumed(x, &k, false); e != nil || got || builder.coeff.Size() != 2 {
		t.Fatalf("non-add changed buffer: got=%v size=%d err=%v", got, builder.coeff.Size(), e)
	}
	bound := func(x uint32, n int32, strict, lower bool) {
		t.Helper()
		set(n)
		var accepted bool
		var e error
		if lower {
			accepted, e = store.AssertLower(x, &k, strict)
		} else {
			accepted, e = store.AssertUpper(x, &k, strict)
		}
		equationMust(t, e)
		if !accepted {
			t.Fatal("source bound refused", x, n, lower)
		}
	}
	bound(0, 2, false, true)
	bound(1, 3, true, true)
	bound(0, 7, false, false)
	bound(1, 11, false, false)
	check(true, 5, true, true)
	check(true, 4, true, true)
	check(true, 6, false, false)
	check(false, 18, true, false)
	check(false, 18, false, true)
	check(false, 19, true, true)
	check(false, 17, false, false)

	// A negative coefficient must select the opposite bound for x. Reusing the
	// same source buffers also proves that the earlier sum was cleared.
	negativeTwo, err := q.NewValue()
	equationMust(t, err)
	equationMust(t, q.SetString(&negativeTwo, "-2"))
	coefficient, err := m.ArithmeticNumeral(&negativeTwo, false)
	equationMust(t, err)
	equationMust(t, q.Del(&negativeTwo))
	product, err := m.MkFamilyApp(family, 9, nil, []ast.ASTHandle{coefficient, x})
	equationMust(t, err)
	negativeSum, err := m.MkFamilyApp(family, 6, nil, []ast.ASTHandle{product, y})
	equationMust(t, err)
	set(-11)
	got, err := builder.LowerSubsumed(negativeSum, &k, true)
	equationMust(t, err)
	if !got || builder.coeff.Size() != 2 || len(builder.ids) != 2 {
		t.Fatalf("negative lower: got=%v buffer=(%d,%d)", got, builder.coeff.Size(), len(builder.ids))
	}
	set(7)
	got, err = builder.UpperSubsumed(negativeSum, &k, true)
	equationMust(t, err)
	if got {
		t.Fatal("nonstrict upper bound subsumed a strict inequality")
	}
	got, err = builder.UpperSubsumed(negativeSum, &k, false)
	equationMust(t, err)
	if !got {
		t.Fatal("equal nonstrict negative-coefficient upper bound was missed")
	}
}
