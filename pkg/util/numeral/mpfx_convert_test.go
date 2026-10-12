package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFixedPointRetainedRationalIntegerConversionAndDisplay(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
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
	a, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	integer, returned := NewInt(0), NewInt(0)
	rational, exact := NewRational(), NewRational()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		must(a.Close())
		must(m.Close())
		must(q.Manager.Del(&rational))
		must(q.Manager.Del(&exact))
		must(q.Manager.Z.Del(&integer))
		must(q.Manager.Z.Del(&returned))
		must(q.Close())
		if budget.Used() != 0 {
			t.Fatalf("mpfx conversion retained %d bytes", budget.Used())
		}
	}()
	want := func(expected string) {
		t.Helper()
		s, err := m.String(a)
		must(err)
		if s != expected {
			t.Fatalf("mpfx display = %q, want %q", s, expected)
		}
	}
	must(m.SetRatio(a, 3, 8))
	want("3/2^3")
	s, err := m.ToRationalString(a)
	must(err)
	if s != "3/2^3" {
		t.Fatalf("rational display = %q", s)
	}
	var out bytes.Buffer
	must(m.Display(&out, a))
	if out.String() != "3/2^3" {
		t.Fatalf("stream display = %q", out.String())
	}
	must(m.ToRational(a, q.Manager, &exact))
	s, err = q.Manager.String(&exact)
	must(err)
	if s != "3/8" {
		t.Fatalf("exact rational conversion = %q", s)
	}
	if err := m.ToInteger(a, q.Manager.Z, &returned); !errors.Is(err, ErrFixedPointDomain) {
		t.Fatalf("fraction-to-integer error = %v", err)
	}
	must(q.Manager.SetString(&rational, "-1/3"))
	must(m.Reset(a))
	must(m.SetFromRational(a, q.Manager, &rational))
	want("1431655765/2^32") // source inherits the old zero sign
	must(m.SetInt64(a, -1))
	must(q.Manager.SetString(&rational, "1/3"))
	must(m.SetFromRational(a, q.Manager, &rational))
	want("-715827883/2^31") // source inherits the old negative sign
	must(q.Manager.SetString(&rational, "-7"))
	must(m.SetFromRational(a, q.Manager, &rational))
	want("-7") // the integer branch reads its input sign
	must(q.Manager.Z.SetString(&integer, "18446744073709551615"))
	must(m.SetFromInteger(a, q.Manager.Z, &integer))
	want("18446744073709551615")
	must(m.ToInteger(a, q.Manager.Z, &returned))
	s, err = q.Manager.Z.String(&returned)
	must(err)
	if s != "18446744073709551615" {
		t.Fatalf("integer roundtrip = %q", s)
	}
	must(q.Manager.Z.SetString(&integer, "18446744073709551616"))
	if err := m.SetFromInteger(a, q.Manager.Z, &integer); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("integer-word overflow = %v", err)
	}
	want("18446744073709551615")
}
