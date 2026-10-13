package interval

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func arbitraryExtendedFixture(t *testing.T) (*numeral.ArbitraryFloatManager, *numeral.ArbitraryFloatNumeralManager) {
	t.Helper()
	b := arena.NewBudget(1 << 27)
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{
		Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
		if err := core.Close(); err != nil {
			t.Error(err)
		}
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("MPF owners", b.Used())
		}
	})
	return core, m
}

func TestSubpavingMPFExtendedFiniteAndInfiniteBranches(t *testing.T) {
	core, m := arbitraryExtendedFixture(t)
	var a, b, out numeral.ArbitraryFloat
	t.Cleanup(func() {
		for _, x := range []*numeral.ArbitraryFloat{&a, &b, &out} {
			if err := m.Del(x); err != nil {
				t.Error(err)
			}
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	value := func(x *numeral.ArbitraryFloat, want float64) {
		t.Helper()
		got, err := core.ToDouble(x)
		if err != nil || got != want {
			t.Fatalf("value: got %v, %v; want %v", got, err, want)
		}
	}
	e := ArbitraryExtended{Numbers: m}
	must(m.SetInt32(&a, -3))
	must(m.SetInt32(&b, 2))
	negative, err := e.IsNeg(&a, Finite)
	must(err)
	positive, err := e.IsPos(nil, PlusInfinity)
	must(err)
	less, err := e.Less(&a, Finite, &b, Finite)
	must(err)
	cmp, err := e.Compare(nil, MinusInfinity, &b, Finite)
	must(err)
	if !negative || !positive || !less || cmp >= 0 {
		t.Fatal("extended order and signs", negative, positive, less, cmp)
	}
	var display bytes.Buffer
	must(e.Display(&display, &a, Finite))
	if display.String() != "-1.5 1" {
		t.Fatal("finite display", display.String())
	}
	display.Reset()
	must(e.Display(&display, nil, MinusInfinity))
	if display.String() != "-oo" {
		t.Fatal("infinite display", display.String())
	}
	k, err := e.Add(&a, Finite, &b, Finite, &out)
	must(err)
	if k != Finite {
		t.Fatal(k)
	}
	value(&out, -1)
	k, err = e.Sub(&a, Finite, &b, Finite, &out)
	must(err)
	if k != Finite {
		t.Fatal(k)
	}
	value(&out, -5)
	k, err = e.Mul(&a, Finite, &b, Finite, &out)
	must(err)
	if k != Finite {
		t.Fatal(k)
	}
	value(&out, -6)
	k, err = e.Div(&a, Finite, &b, Finite, &out)
	must(err)
	if k != Finite {
		t.Fatal(k)
	}
	value(&out, -1.5)
	k, err = e.Add(&a, MinusInfinity, &b, Finite, &out)
	must(err)
	if k != MinusInfinity {
		t.Fatal(k)
	}
	zero, err := m.IsZero(&out)
	must(err)
	if !zero {
		t.Fatal("infinite sum did not clear finite output")
	}
	k, err = e.Mul(&out, Finite, &b, PlusInfinity, &out)
	must(err)
	if k != Finite {
		t.Fatal("zero times infinity", k)
	}
	zero, err = m.IsZero(&out)
	must(err)
	if !zero {
		t.Fatal("zero times infinity output")
	}
	_, err = e.Add(&a, MinusInfinity, &b, PlusInfinity, &out)
	if !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("opposite infinities", err)
	}
	_, err = e.Sub(&a, PlusInfinity, &b, PlusInfinity, &out)
	if !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("same infinity subtraction", err)
	}
	k, err = e.Mul(&a, MinusInfinity, &b, Finite, &out)
	must(err)
	if k != MinusInfinity {
		t.Fatal("infinite product", k)
	}
	k, err = e.Div(&a, PlusInfinity, &b, Finite, &out)
	must(err)
	if k != PlusInfinity {
		t.Fatal("infinite quotient", k)
	}
	k, err = e.Inv(&out, PlusInfinity)
	must(err)
	if k != Finite {
		t.Fatal("infinite inverse", k)
	}
	zero, err = m.IsZero(&out)
	must(err)
	if !zero {
		t.Fatal("infinite inverse output")
	}
	_, err = e.Div(&a, Finite, &out, Finite, &out)
	if !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("division by zero", err)
	}
	k, err = e.Power(&a, MinusInfinity, 2)
	must(err)
	if k != PlusInfinity {
		t.Fatal("even negative infinity", k)
	}
	must(m.SetInt32(&a, -3))
	k, err = e.Power(&a, Finite, 2)
	must(err)
	if k != Finite {
		t.Fatal(k)
	}
	value(&a, 9)
}
