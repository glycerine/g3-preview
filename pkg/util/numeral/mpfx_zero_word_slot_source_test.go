// Source: src/util/mpfx.cpp:280-306,419-444. Nonexceptional zero-word results
// can retain the source allocated slot and sign after directed rounding.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFixedPointProductUnderflowRetainsAllocatedZeroWords(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	b, _ := m.New()
	out, _ := m.New()
	defer func() {
		for _, v := range []*FixedPoint{a, b, out} {
			if err := v.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	}()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(m.SetRatio(a, 1, 1<<32))
	must(m.SetRatio(b, 1, 2))
	must(m.Mul(a, b, out))
	if zero, err := m.IsZero(out); err != nil || zero {
		t.Fatal("positive underflow lost allocated slot", zero, err)
	}
	if pos, err := m.IsPos(out); err != nil || !pos {
		t.Fatal("positive underflow sign", pos, err)
	}
	must(m.SetRounding(true))
	must(m.SetRatio(b, -1, 2))
	must(m.Mul(a, b, out))
	if text, err := m.String(out); err != nil || text != "-0" {
		t.Fatal("negative underflow slot and sign", text, err)
	}
	must(m.Mul(out, a, out))
	if neg, err := m.IsNeg(out); err != nil || !neg {
		t.Fatal("aliased zero-word product lost source sign", neg, err)
	}
	must(m.Reset(a))
	must(m.Mul(a, b, out))
	if zero, err := m.IsZero(out); err != nil || !zero {
		t.Fatal("canonical zero operand did not reset product", zero, err)
	}
}

func TestFixedPointNonintegerRationalUnderflowRetainsDestinationSign(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, err := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := m.New()
	r := NewRational()
	defer func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
		if err := q.Manager.Del(&r); err != nil {
			t.Error(err)
		}
		if err := q.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("rational conversion retained budget", budget.Used())
		}
	}()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(q.Manager.SetString(&r, "1/8589934592"))
	must(m.SetFromRational(out, q.Manager, &r))
	if zero, err := m.IsZero(out); err != nil || zero {
		t.Fatal("noninteger zero quotient released source slot", zero, err)
	}
	must(m.SetInt64(out, -1))
	must(m.SetRounding(true))
	must(m.SetFromRational(out, q.Manager, &r))
	if text, err := m.String(out); err != nil || text != "-0" {
		t.Fatal("noninteger setter lost prior negative sign", text, err)
	}
	must(m.SetFromRational(out, q.Manager, &r))
	if neg, err := m.IsNeg(out); err != nil || !neg {
		t.Fatal("repeated setter did not read retained sign", neg, err)
	}
	must(q.Manager.SetString(&r, "0"))
	must(m.SetFromRational(out, q.Manager, &r))
	if zero, err := m.IsZero(out); err != nil || !zero {
		t.Fatal("integer zero setter did not reset slot", zero, err)
	}
}
