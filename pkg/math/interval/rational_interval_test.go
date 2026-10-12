package interval

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalIntervalRetainedPredicatesAddSubAndOwnership(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	lower, upper, point := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	var a, b, c, d *RationalInterval
	defer func() {
		for _, v := range []*RationalInterval{a, b, c, d} {
			if v != nil {
				must(v.Close())
			}
		}
		must(m.Del(&lower))
		must(m.Del(&upper))
		must(m.Del(&point))
		must(m.CloseChecked())
		if budget.Used() != 0 {
			t.Fatalf("interval owner retained %d bytes", budget.Used())
		}
	}()
	a, err = NewRationalInterval(m)
	must(err)
	b, err = NewRationalInterval(m)
	must(err)
	c, err = NewRationalInterval(m)
	must(err)
	d, err = NewRationalInterval(m)
	must(err)
	must(m.SetString(&lower, "-3/2"))
	must(m.SetString(&upper, "2"))
	must(a.SetFinite(&lower, &upper, true, false)) // (-3/2, 2]
	must(m.SetString(&lower, "2"))
	must(m.SetString(&upper, "5/2"))
	must(b.SetFinite(&lower, &upper, false, true)) // [2, 5/2)
	var out bytes.Buffer
	must(a.Display(&out))
	if out.String() != "(-3/2, 2]" {
		t.Fatalf("interval display = %q", out.String())
	}
	before, err := a.Before(b)
	must(err)
	if before {
		t.Fatal("source before predicate separates intervals touching at a closed upper bound")
	}
	must(m.SetString(&point, "2"))
	contains, err := a.Contains(&point)
	must(err)
	if !contains {
		t.Fatal("closed upper bound excluded")
	}
	must(m.SetString(&point, "-3/2"))
	contains, err = a.Contains(&point)
	must(err)
	if contains {
		t.Fatal("open lower bound included")
	}
	contains, err = a.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("zero missing from crossing interval")
	}
	must(c.Add(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(1/2, 9/2)" {
		t.Fatalf("interval add = %q", out.String())
	}
	must(d.Sub(a, b))
	out.Reset()
	must(d.Display(&out))
	if out.String() != "(-4, 0]" {
		t.Fatalf("interval sub = %q", out.String())
	}
	contains, err = d.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("subtraction lost closed zero endpoint")
	}
	equal, err := a.Equal(b)
	must(err)
	if equal {
		t.Fatal("distinct intervals equal")
	}
	must(b.Reset())
	must(c.Add(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("unbounded interval sum = %q", out.String())
	}
	contains, err = c.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("unbounded sum excludes zero")
	}
	must(c.CheckInvariant())
}
