package realclosure

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureBinaryIntervalRetainedCore(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewBinaryRationalManager(q.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: q})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	lo, hi := numeral.NewBinaryRational(), numeral.NewBinaryRational()
	var a, b, c *BinaryInterval
	defer func() {
		for _, v := range []*BinaryInterval{a, b, c} {
			if v != nil {
				must(v.Close())
			}
		}
		must(m.Del(&lo))
		must(m.Del(&hi))
		must(m.CloseChecked())
		must(q.Close())
		if budget.Used() != 0 {
			t.Fatalf("real-closure interval retained %d bytes", budget.Used())
		}
	}()
	a, err = NewBinaryInterval(m)
	must(err)
	b, err = NewBinaryInterval(m)
	must(err)
	c, err = NewBinaryInterval(m)
	must(err)
	var out bytes.Buffer
	must(a.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("default interval = %q", out.String())
	}
	z, err := a.ContainsZero()
	must(err)
	if !z {
		t.Fatal("default unbounded interval excludes zero")
	}
	must(m.SetInt32Power(&lo, -3, 1))
	must(m.SetInt32(&hi, 2))
	must(a.SetFinite(&lo, &hi, true, false)) // (-3/2,2]
	must(m.SetInt32(&lo, 1))
	must(m.SetInt32Power(&hi, 3, 1))
	must(b.SetFinite(&lo, &hi, false, true)) // [1,3/2)
	must(c.Add(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-1/2, 7/2)" {
		t.Fatalf("binary interval add = %q", out.String())
	}
	must(c.SetFrom(a))
	must(c.Add(c, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-1/2, 7/2)" {
		t.Fatalf("aliased binary interval add = %q", out.String())
	}
	must(c.Neg(a))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-2, 3/2)" {
		t.Fatalf("binary interval neg = %q", out.String())
	}
	must(c.Neg(c))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-3/2, 2]" {
		t.Fatalf("in-place binary neg = %q", out.String())
	}
	p, err := a.IsP()
	must(err)
	n, err := a.IsN()
	must(err)
	if p || n {
		t.Fatal("crossing interval classified as positive or negative")
	}
	p, err = b.IsP()
	must(err)
	if !p {
		t.Fatal("positive interval misclassified")
	}
	must(m.SetInt32(&lo, 2))
	must(m.SetInt32(&hi, 3))
	must(b.SetFinite(&lo, &hi, false, false))
	before, err := a.Before(b)
	must(err)
	if before {
		t.Fatal("touching closed upper bound ordered before next interval")
	}
	must(a.ResetLower())
	must(c.Neg(a))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-2, +oo)" {
		t.Fatalf("one-sided binary neg = %q", out.String())
	}
	must(b.Reset())
	must(c.Add(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("unbounded binary add = %q", out.String())
	}
	must(c.CheckInvariant())
	set := func(v *BinaryInterval, l, u int32, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetInt32(&lo, l))
		must(m.SetInt32(&hi, u))
		must(v.SetFinite(&lo, &hi, lowerOpen, upperOpen))
	}
	show := func(v *BinaryInterval) string {
		t.Helper()
		out.Reset()
		must(v.Display(&out))
		return out.String()
	}
	classes := []struct {
		name           string
		al, au, bl, bu int32
		want           string
	}{
		{"NN", -3, -1, -3, -1, "[1, 9]"},
		{"NM", -3, -1, -2, 4, "[-12, 6]"},
		{"NP", -3, -1, 1, 3, "[-9, -1]"},
		{"MN", -2, 4, -3, -1, "[-12, 6]"},
		{"MM", -2, 4, -2, 4, "[-8, 16]"},
		{"MP", -2, 4, 1, 3, "[-6, 12]"},
		{"PN", 1, 3, -3, -1, "[-9, -1]"},
		{"PM", 1, 3, -2, 4, "[-6, 12]"},
		{"PP", 1, 3, 1, 3, "[1, 9]"},
	}
	for _, tc := range classes {
		t.Run(tc.name, func(t *testing.T) {
			set(a, tc.al, tc.au, false, false)
			set(b, tc.bl, tc.bu, false, false)
			must(c.Mul(a, b))
			if got := show(c); got != tc.want {
				t.Fatalf("binary multiplication = %q, want %q", got, tc.want)
			}
		})
	}
	set(a, -3, 0, false, false)
	set(b, 0, 2, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[-6, 0]" {
		t.Fatalf("closed-zero multiplication = %q", got)
	}
	must(a.ResetLower())
	must(c.Mul(a, b))
	if got := show(c); got != "(-oo, 0]" {
		t.Fatalf("infinite multiplication = %q", got)
	}
	set(a, 0, 0, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[0, 0]" {
		t.Fatalf("zero singleton multiplication = %q", got)
	}
	set(a, -2, 4, true, false)
	set(b, -3, 6, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[-12, 24]" {
		t.Fatalf("mixed endpoint tie = %q", got)
	}
	must(c.SetFrom(b))
	must(c.Mul(a, c))
	if got := show(c); got != "[-12, 24]" {
		t.Fatalf("aliased binary multiplication = %q", got)
	}
	set(a, -2, 4, false, true)
	set(b, -4, 2, false, false)
	must(a.Mul(a, b))
	if got := show(a); got != "(-16, 8]" {
		t.Fatalf("upper mixed tie with aliased left input = %q", got)
	}
}
