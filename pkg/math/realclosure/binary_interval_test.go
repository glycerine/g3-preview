package realclosure

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type binaryDisplayRefusal struct {
	bytes.Buffer
	remaining int
	err       error
}

func (w *binaryDisplayRefusal) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, w.err
	}
	w.remaining--
	return w.Buffer.Write(p)
}

func (w *binaryDisplayRefusal) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

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
	out.Reset()
	must(a.DisplayPP(&out))
	if out.String() != "(-&infin;, +&infin;)" {
		t.Fatalf("default pretty interval = %q", out.String())
	}
	z, err := a.ContainsZero()
	must(err)
	if !z {
		t.Fatal("default unbounded interval excludes zero")
	}
	must(m.SetInt32Power(&lo, -3, 1))
	must(m.SetInt32(&hi, 2))
	must(a.SetFinite(&lo, &hi, true, false)) // (-3/2,2]
	out.Reset()
	must(a.DisplayPP(&out))
	if out.String() != "(-3/2, 2]" {
		t.Fatalf("finite pretty interval = %q", out.String())
	}
	writeErr := errors.New("binary upper endpoint refused")
	refused := &binaryDisplayRefusal{remaining: 3, err: writeErr}
	if err := a.DisplayPP(refused); !errors.Is(err, writeErr) || refused.String() != "(-3/2, " {
		t.Fatalf("pretty display refusal = (%q, %v)", refused.String(), err)
	}
	must(m.SetInt32Power(&lo, -3, 3))
	must(m.SetInt32Power(&hi, 5, 2))
	must(c.SetFinite(&lo, &hi, false, true))
	out.Reset()
	must(c.DisplayPP(&out))
	if out.String() != "[-3/2<sup>3</sup>, 5/2<sup>2</sup>)" {
		t.Fatalf("powered pretty interval = %q", out.String())
	}
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
	out.Reset()
	must(c.DisplayPP(&out))
	if out.String() != "[-2, +&infin;)" {
		t.Fatalf("one-sided pretty interval = %q", out.String())
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
	bracketsThird := func(v *BinaryInterval) {
		t.Helper()
		one, three, scaled := numeral.NewBinaryInt(1), numeral.NewBinaryInt(3), numeral.NewBinaryRational()
		defer func() { must(m.Del(&scaled)) }()
		must(m.Mul(&v.lower, &three, &scaled))
		below, err := m.Less(&scaled, &one)
		must(err)
		must(m.Mul(&v.upper, &three, &scaled))
		above, err := m.Less(&one, &scaled)
		must(err)
		if !below || !above {
			t.Fatalf("directed interval does not strictly enclose 1/3: %s", show(v))
		}
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
	set(a, 3, 3, false, false)
	must(c.Inv(a, 12))
	less, err := m.Less(&c.lower, &c.upper)
	must(err)
	if !less {
		t.Fatalf("directed reciprocal failed to bracket 1/3: %s", show(c))
	}
	bracketsThird(c)
	set(a, 0, 2, true, false)
	must(c.Inv(a, 12))
	if got := show(c); got != "[1/2, +oo)" {
		t.Fatalf("open positive zero reciprocal = %q", got)
	}
	set(a, -2, 0, false, true)
	must(c.Inv(a, 12))
	if got := show(c); got != "(-oo, -1/2]" {
		t.Fatalf("open negative zero reciprocal = %q", got)
	}
	set(a, -2, -1, false, false)
	must(c.Inv(a, 12))
	if got := show(c); got != "[-1, -1/2]" {
		t.Fatalf("negative reciprocal = %q", got)
	}
	set(a, 2, 3, false, false)
	must(a.ResetUpper())
	must(c.SetFrom(a))
	must(c.Inv(c, 12))
	if got := show(c); got != "(0, 1/2]" {
		t.Fatalf("aliased unbounded reciprocal = %q", got)
	}
	set(a, -3, -2, false, false)
	must(a.ResetLower())
	must(c.Inv(a, 12))
	if got := show(c); got != "[-2049/2^12, 0)" {
		t.Fatalf("negative unbounded reciprocal = %q", got)
	}
	set(a, 0, 2, false, false)
	if err := c.Inv(a, 12); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("closed-zero reciprocal error = %v", err)
	}
	quotients := []struct {
		name           string
		al, au, bl, bu int32
		want           string
	}{
		{"NN", -3, -1, -1, -1, "[1, 3]"},
		{"NP", -3, -1, 1, 1, "[-3, -1]"},
		{"MN", -2, 4, -1, -1, "[-4, 2]"},
		{"MP", -2, 4, 1, 1, "[-2, 4]"},
		{"PN", 1, 3, -1, -1, "[-3, -1]"},
		{"PP", 1, 3, 1, 1, "[1, 3]"},
	}
	for _, tc := range quotients {
		t.Run("div-"+tc.name, func(t *testing.T) {
			set(a, tc.al, tc.au, false, false)
			set(b, tc.bl, tc.bu, false, false)
			must(c.Div(a, b, 12))
			if got := show(c); got != tc.want {
				t.Fatalf("binary interval division = %q, want %q", got, tc.want)
			}
		})
	}
	set(a, 1, 1, false, false)
	set(b, 3, 3, false, false)
	must(c.Div(a, b, 12))
	less, err = m.Less(&c.lower, &c.upper)
	must(err)
	if !less {
		t.Fatalf("directed quotient failed to bracket 1/3: %s", show(c))
	}
	bracketsThird(c)
	set(a, 1, 3, false, false)
	set(b, 0, 2, true, false)
	must(c.Div(a, b, 12))
	if got := show(c); got != "[1/2, +oo)" {
		t.Fatalf("positive open-zero quotient = %q", got)
	}
	must(c.SetFrom(b))
	must(c.Div(a, c, 12))
	if got := show(c); got != "[1/2, +oo)" {
		t.Fatalf("aliased denominator quotient = %q", got)
	}
	set(b, -2, 0, false, true)
	must(c.Div(a, b, 12))
	if got := show(c); got != "(-oo, -1/2]" {
		t.Fatalf("negative open-zero quotient = %q", got)
	}
	set(b, 2, 3, false, false)
	must(b.ResetUpper())
	must(c.Div(a, b, 12))
	if got := show(c); got != "(0, 3/2]" {
		t.Fatalf("unbounded denominator quotient = %q", got)
	}
	set(b, 0, 2, true, false)
	set(a, -2, 4, false, false)
	must(c.Div(a, b, 12))
	if got := show(c); got != "(-oo, +oo)" {
		t.Fatalf("mixed open-zero quotient = %q", got)
	}
	set(a, 0, 0, false, false)
	must(c.Div(a, b, 12))
	if got := show(c); got != "[0, 0]" {
		t.Fatalf("zero numerator quotient = %q", got)
	}
	if err := a.Div(a, b, 12); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("numerator alias error = %v", err)
	}
	set(b, 0, 2, false, false)
	if err := c.Div(a, b, 12); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("zero-containing denominator error = %v", err)
	}
}
