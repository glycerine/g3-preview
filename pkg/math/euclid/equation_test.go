package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanEquationConstructionSortAndOwnedRelease(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(5))
	euclidMust(t, rb.Resize(4))
	as := []numeral.Int{{}, {}}
	euclidMust(t, z.SetString(&as[0], "123456789012345678901"))
	euclidMust(t, z.SetString(&as[1], "-7"))
	c := numeral.Int{}
	euclidMust(t, z.SetString(&c, "987654321098765432109"))
	bs := []numeral.Rational{numeral.NewRational(), numeral.NewRational()}
	euclidMust(t, m.SetString(&bs[0], "23/11"))
	euclidMust(t, m.SetString(&bs[1], "-5/13"))
	eq, e := makeEquation(m, as, []uint32{4, 1}, &c, bs, []uint32{3, 0}, ib, rb, true, nil)
	euclidMust(t, e)
	if len(eq.as) != 2 || len(eq.bs) != 2 || eq.xs[0] != 1 || eq.xs[1] != 4 || eq.js[0] != 0 || eq.js[1] != 3 {
		t.Fatal("equation source scalar order", eq.xs, eq.js)
	}
	for i, want := range []string{"-7", "123456789012345678901"} {
		got, e := z.String(&eq.as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("equation integer coefficient", i, got, want)
		}
	}
	for i, want := range []string{"-5/13", "23/11"} {
		got, e := m.String(&eq.bs[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("equation rational justification", i, got, want)
		}
	}
	if got, e := z.String(&eq.c); e != nil || got != "987654321098765432109" {
		t.Fatal("equation constant copy", got, e)
	}
	// Closing the equation releases its numerals; the reusable buffers and
	// original caller numerals keep their separate ownership.
	euclidMust(t, eq.Close())
	euclidMust(t, eq.Close())
	for i, want := range []string{"123456789012345678901", "-7"} {
		got, e := z.String(&as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("caller integer changed", i, got)
		}
	}
	for i, want := range []string{"23/11", "-5/13"} {
		got, e := m.String(&bs[i])
		euclidMust(t, e)
		if got != want {
			t.Fatal("caller rational changed", i, got)
		}
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	for i := range bs {
		euclidMust(t, m.Del(&bs[i]))
	}
	euclidMust(t, z.Del(&c))
}

func TestEuclideanEquationUnsortedDecompositionBranch(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(3))
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(-1)}
	c := numeral.NewInt(0)
	eq, e := makeEquation(m, as, []uint32{2, 0}, &c, nil, nil, ib, rb, false, nil)
	euclidMust(t, e)
	if len(eq.bs) != 0 || eq.xs[0] != 0 || eq.xs[1] != 2 || eq.as[0].Val != -1 || eq.as[1].Val != 2 {
		t.Fatal("decomposition equation sort/justification", eq.xs, eq.as, eq.bs)
	}
	euclidMust(t, eq.Close())
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
}

func TestEuclideanEquationHeaderRefusalPrecedesBufferMutation(t *testing.T) {
	m := euclidNumeralManager(t)
	ib, e := numeral.NewScopedIntVector(m.Integers())
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	a := []numeral.Int{numeral.NewInt(9)}
	c := numeral.NewInt(2)
	eq, e := makeEquation(m, a, []uint32{0}, &c, nil, nil, ib, rb, true, nil)
	if e == nil || eq != nil {
		t.Fatal("equation header refusal", eq, e)
	}
	m.BudgetField.FailAfter(-1)
	slot, e := ib.BorrowSlot(0)
	euclidMust(t, e)
	if slot.Val != 0 || a[0].Val != 9 || c.Val != 2 || m.BudgetField.Used() != before {
		t.Fatal("header refusal changed input or buffer", slot, a, c)
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
}

func TestEuclideanEquationFailedConstantCopyReleasesConstructedPrefix(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	ib, e := numeral.NewScopedIntVector(z)
	euclidMust(t, e)
	rb, e := numeral.NewScopedRationalVector(m)
	euclidMust(t, e)
	euclidMust(t, ib.Resize(1))
	var a numeral.Int
	euclidMust(t, z.SetString(&a, "123456789012345678901"))
	bad := numeral.Int{Val: 1, Cell: z.FailedRef}
	before := m.BudgetField.Used()
	eq, e := makeEquation(m, []numeral.Int{a}, []uint32{0}, &bad, nil, nil, ib, rb, true, nil)
	if e == nil || eq != nil {
		t.Fatal("failed constant copy must clean the partial equation", eq, e)
	}
	if m.BudgetField.Used() != before {
		t.Fatal("partial equation header remained charged")
	}
	if got, e := z.String(&a); e != nil || got != "123456789012345678901" {
		t.Fatal("input lost its independent value", got, e)
	}
	euclidMust(t, ib.Close())
	euclidMust(t, rb.Close())
	euclidMust(t, z.Del(&a))
}
