package interval

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type failAfterWrites struct {
	bytes.Buffer
	remaining int
	err       error
}

func (w *failAfterWrites) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, w.err
	}
	w.remaining--
	return w.Buffer.Write(p)
}

func (w *failAfterWrites) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

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
	// Source interval display inserts each component in order. A failed
	// upper endpoint write must leave the already emitted prefix intact.
	writeErr := errors.New("upper endpoint refused")
	refused := &failAfterWrites{remaining: 3, err: writeErr}
	if err := a.Display(refused); !errors.Is(err, writeErr) || refused.String() != "(-3/2, " {
		t.Fatalf("display refusal = (%q, %v)", refused.String(), err)
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
	must(c.Neg(a))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-2, 3/2)" {
		t.Fatalf("interval negation = %q", out.String())
	}
	must(c.Neg(c))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-3/2, 2]" {
		t.Fatalf("in-place interval negation = %q", out.String())
	}
	must(m.SetInt32(&point, -2))
	must(c.MulScalar(&point, a))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-4, 3)" {
		t.Fatalf("negative scalar multiplication = %q", out.String())
	}
	must(c.DivScalar(a, &point))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-1, 3/4)" {
		t.Fatalf("negative scalar division = %q", out.String())
	}
	must(m.SetInt32(&point, 0))
	must(c.MulScalar(&point, a))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("source zero-scalar reset = %q", out.String())
	}
	must(b.ResetLower()) // (-oo, 5/2)
	must(m.SetInt32(&point, 2))
	must(c.MulScalar(&point, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, 5)" {
		t.Fatalf("positive scalar with infinite lower = %q", out.String())
	}
	must(m.SetInt32(&point, -2))
	must(c.MulScalar(&point, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-5, +oo)" {
		t.Fatalf("negative scalar with infinite lower = %q", out.String())
	}
	must(c.Neg(b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-5/2, +oo)" {
		t.Fatalf("one-sided negation = %q", out.String())
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
	inputs := map[byte][2]string{'N': {"-3", "-1"}, 'M': {"-2", "4"}, 'P': {"1", "3"}}
	wants := map[string]string{
		"NN": "[1, 9]", "NM": "[-12, 6]", "NP": "[-9, -1]",
		"MN": "[-12, 6]", "MM": "[-8, 16]", "MP": "[-6, 12]",
		"PN": "[-9, -1]", "PM": "[-6, 12]", "PP": "[1, 9]",
	}
	setClass := func(dst *RationalInterval, class byte) {
		t.Helper()
		bounds := inputs[class]
		must(m.SetString(&lower, bounds[0]))
		must(m.SetString(&upper, bounds[1]))
		must(dst.SetFinite(&lower, &upper, false, false))
	}
	for pair, want := range wants {
		setClass(a, pair[0])
		setClass(b, pair[1])
		must(c.Mul(a, b))
		out.Reset()
		must(c.Display(&out))
		if out.String() != want {
			t.Fatalf("interval product %s = %q, want %q", pair, out.String(), want)
		}
	}
	must(m.SetInt32(&lower, -3))
	must(m.SetInt32(&upper, 0))
	must(a.SetFinite(&lower, &upper, false, false))
	must(m.SetInt32(&lower, 0))
	must(m.SetInt32(&upper, 2))
	must(b.SetFinite(&lower, &upper, false, false))
	must(c.Mul(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-6, 0]" {
		t.Fatalf("closed-zero interval product = %q", out.String())
	}
	must(a.ResetLower()) // (-oo, 0]
	must(c.Mul(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, 0]" {
		t.Fatalf("infinite interval product = %q", out.String())
	}
	must(m.SetInt32(&lower, -2))
	must(m.SetInt32(&upper, 4))
	must(a.SetFinite(&lower, &upper, true, false))
	must(m.SetInt32(&lower, -3))
	must(m.SetInt32(&upper, 6))
	must(b.SetFinite(&lower, &upper, false, false))
	must(c.Mul(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-12, 24]" {
		t.Fatalf("mixed tie prefers closed lower = %q", out.String())
	}
	must(c.SetFrom(b))
	must(c.Mul(a, c))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-12, 24]" {
		t.Fatalf("aliased mixed product = %q", out.String())
	}
	must(m.SetInt32(&lower, -2))
	must(m.SetInt32(&upper, 4))
	must(a.SetFinite(&lower, &upper, false, true))
	must(c.Mul(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[-12, 24)" {
		t.Fatalf("mixed tie prefers closed alternate = %q", out.String())
	}
	must(a.ResetLower())
	must(b.ResetUpper())
	must(c.Mul(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("mixed infinite product = %q", out.String())
	}
	setPowerInput := func(l, u string, lopen, uopen bool) {
		t.Helper()
		must(m.SetString(&lower, l))
		must(m.SetString(&upper, u))
		must(a.SetFinite(&lower, &upper, lopen, uopen))
	}
	assertPower := func(n uint32, want string) {
		t.Helper()
		must(c.Power(a, n))
		out.Reset()
		must(c.Display(&out))
		if out.String() != want {
			t.Fatalf("interval %d-power = %q, want %q", n, out.String(), want)
		}
	}
	setPowerInput("2", "3", false, false)
	assertPower(2, "[4, 9]")
	must(c.Power(c, 2))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[16, 81]" {
		t.Fatalf("aliased power = %q", out.String())
	}
	assertPower(0, "[1, 1]")
	setPowerInput("-3", "-2", false, false)
	assertPower(2, "[4, 9]")
	must(c.SetFrom(a))
	must(c.Power(c, 2))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[4, 9]" {
		t.Fatalf("aliased negative even power = %q", out.String())
	}
	must(a.ResetLower())
	assertPower(2, "[4, +oo)")
	setPowerInput("-2", "3", false, false)
	assertPower(2, "[0, 9]")
	assertPower(3, "[-8, 27]")
	assertPower(0, "[0, 1]")
	must(a.ResetLower())
	assertPower(3, "(-oo, 27]")
	setPowerInput("-3", "3", false, true)
	assertPower(2, "[0, 9]") // equal maxima prefer closed lower endpoint
	setPowerInput("-3", "3", true, true)
	assertPower(2, "[0, 9)")
	assertInverse := func(want string) {
		t.Helper()
		must(c.Inv(a))
		out.Reset()
		must(c.Display(&out))
		if out.String() != want {
			t.Fatalf("interval inverse = %q, want %q", out.String(), want)
		}
	}
	setPowerInput("2", "4", false, false)
	assertInverse("[1/4, 1/2]")
	must(c.Inv(c))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[2, 4]" {
		t.Fatalf("aliased inverse = %q", out.String())
	}
	setPowerInput("0", "4", true, false)
	assertInverse("[1/4, +oo)")
	setPowerInput("2", "4", false, false)
	must(a.ResetUpper())
	assertInverse("(0, 1/2]")
	setPowerInput("-4", "-2", false, false)
	assertInverse("[-1/2, -1/4]")
	must(a.ResetLower())
	assertInverse("[-1/2, 0)")
	setPowerInput("-4", "0", false, true)
	assertInverse("(-oo, -1/4]")
	setPowerInput("-1", "1", false, false)
	if err := c.Inv(a); err != ErrExtendedDomain {
		t.Fatalf("zero-containing inverse error = %v", err)
	}
	quotInputs := map[byte][2]string{'N': {"-4", "-2"}, 'M': {"-2", "4"}, 'P': {"2", "4"}}
	quotWant := map[string]string{
		"NN": "[1/2, 2]", "NP": "[-2, -1/2]",
		"MN": "[-2, 1]", "MP": "[-1, 2]",
		"PN": "[-2, -1/2]", "PP": "[1/2, 2]",
	}
	setQuot := func(dst *RationalInterval, class byte) {
		t.Helper()
		bounds := quotInputs[class]
		must(m.SetString(&lower, bounds[0]))
		must(m.SetString(&upper, bounds[1]))
		must(dst.SetFinite(&lower, &upper, false, false))
	}
	for pair, want := range quotWant {
		setQuot(a, pair[0])
		setQuot(b, pair[1])
		must(c.Div(a, b))
		out.Reset()
		must(c.Display(&out))
		if out.String() != want {
			t.Fatalf("interval quotient %s = %q, want %q", pair, out.String(), want)
		}
	}
	setQuot(a, 'P')
	setQuot(b, 'P')
	must(c.SetFrom(b))
	must(c.Div(a, c))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[1/2, 2]" {
		t.Fatalf("aliased denominator quotient = %q", out.String())
	}
	if err := a.Div(a, b); err != arena.ErrInvalidSlot {
		t.Fatalf("numerator alias accepted: %v", err)
	}
	must(m.SetInt32(&lower, 0))
	must(m.SetInt32(&upper, 0))
	must(a.SetFinite(&lower, &upper, false, false))
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[0, 0]" {
		t.Fatalf("zero numerator quotient = %q", out.String())
	}
	setQuot(a, 'P')
	must(m.SetInt32(&lower, 0))
	must(m.SetInt32(&upper, 2))
	must(b.SetFinite(&lower, &upper, true, false))
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[1, +oo)" {
		t.Fatalf("positive open-zero divisor = %q", out.String())
	}
	setQuot(a, 'N')
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, -1]" {
		t.Fatalf("negative over positive open zero = %q", out.String())
	}
	setQuot(a, 'M')
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("mixed open-zero divisor = %q", out.String())
	}
	setQuot(a, 'N')
	must(m.SetInt32(&lower, -2))
	must(m.SetInt32(&upper, 0))
	must(b.SetFinite(&lower, &upper, false, true))
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "[1, +oo)" {
		t.Fatalf("negative open-zero divisor = %q", out.String())
	}
	setQuot(a, 'P')
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, -1]" {
		t.Fatalf("positive over negative open zero = %q", out.String())
	}
	setQuot(a, 'M')
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(-oo, +oo)" {
		t.Fatalf("mixed over negative open zero = %q", out.String())
	}
	setQuot(a, 'P')
	setQuot(b, 'P')
	must(b.ResetUpper())
	must(c.Div(a, b))
	out.Reset()
	must(c.Display(&out))
	if out.String() != "(0, 2]" {
		t.Fatalf("infinite divisor quotient = %q", out.String())
	}
	setPowerInput("-1", "1", false, false)
	if err := c.Div(b, a); err != ErrExtendedDomain {
		t.Fatalf("zero-containing divisor error = %v", err)
	}
}
