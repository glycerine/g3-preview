// Source: src/util/inf_s_integer.h:25-347; src/util/inf_s_integer.cpp:22-24.
package util

import (
	"errors"
	"io"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestInfSIntegerConstructorsTraitsAndDebugGuards(t *testing.T) {
	var zero InfSInteger
	if zero != InfSIntegerZero() || !InfSIntegerOne().IsOne() || !InfSIntegerMinusOne().IsMinusOne() {
		t.Fatal("static/default values")
	}
	for _, a := range []int32{-2147483648, -1, 0, 1, 2147483647} {
		for _, b := range []int32{-2, -1, 0, 1, 2} {
			x := NewInfSIntegerPair(NewSInteger(a), NewSInteger(b))
			if x.GetRational().GetInt() != a || x.GetInfinitesimal().GetInt() != b || x.GetInt64() != int64(a) || x.GetUint64() != uint64(int64(a)) {
				t.Fatal("projections", a, b)
			}
			if x.IsInt() != (b == 0) || x.IsInt64() != (b == 0) || x.IsUint64() != (b == 0) || x.IsRational() != (b == 0) {
				t.Fatal("traits", a, b)
			}
			copy := x
			copy.Reset()
			if !copy.IsZero() || x.GetInfinitesimal().GetInt() != b {
				t.Fatal("reset/copy")
			}
			n, e := x.Numerator()
			d, de := x.Denominator()
			if sourceDebug && b != 0 {
				if e != ErrIntegerAssertion || de != ErrIntegerAssertion {
					t.Fatal("source projection guards", e, de)
				}
			} else if e != nil || de != nil || n != NewInfSInteger(a) || d != InfSIntegerOne() {
				t.Fatal("projection assertion erasure", n, d, e, de)
			}
			for _, den := range []int32{0, 1, 2} {
				got, e := NewInfSIntegerFraction(a, den)
				if sourceDebug && den != 1 {
					if e != ErrIntegerAssertion {
						t.Fatal(e)
					}
				} else if e != nil || got != NewInfSInteger(a) {
					t.Fatal("fraction constructor ignores denominator in release", got, e)
				}
			}
			if NewInfSIntegerFromSInteger(NewSInteger(a)) != NewInfSInteger(a) || NewInfSIntegerOffset(NewSInteger(a), true).GetInfinitesimal() != SIntegerOne() || NewInfSIntegerOffset(NewSInteger(a), false).GetInfinitesimal() != SIntegerMinusOne() {
				t.Fatal("scalar constructors")
			}
		}
	}
}

func TestInfSIntegerIndependentLexicographicAlgebra(t *testing.T) {
	var values []InfSInteger
	for a := int32(-4); a <= 4; a++ {
		for b := int32(-4); b <= 4; b++ {
			values = append(values, NewInfSIntegerPair(NewSInteger(a), NewSInteger(b)))
		}
	}
	// Embedding a bounded pair as 100*a+b independently determines lexicographic order.
	for _, x := range values {
		a, b := x.GetRational().GetInt(), x.GetInfinitesimal().GetInt()
		code := int64(a)*100 + int64(b)
		if x.IsNeg() != (code < 0) || x.IsPos() != (code > 0) || x.IsNonneg() != (code >= 0) || x.IsNonpos() != (code <= 0) || x.IsZero() != (code == 0) || x.IsOne() != (a == 1 && b == 0) || x.IsMinusOne() != (a == -1 && b == 0) {
			t.Fatal("lexicographic signs", x)
		}
		floor, ceil := a, a
		if b < 0 {
			floor--
		}
		if b > 0 {
			ceil++
		}
		if x.Floor().GetInt() != floor || x.Ceil().GetInt() != ceil {
			t.Fatal("epsilon rounding", x)
		}
		if x.Hash() != uint32(a^(b+1)) || (InfSIntegerHashProc{}).Hash(x) != x.Hash() || !(InfSIntegerEqProc{}).Equal(x, x) {
			t.Fatal("hash/equality functors")
		}
		neg := x.Negated()
		if neg.GetRational().GetInt() != -a || neg.GetInfinitesimal().GetInt() != -b {
			t.Fatal("negation")
		}
		if abs := x.Abs(); abs.IsNeg() || abs != x && abs != neg {
			t.Fatal("abs")
		}
		z := x
		old := z.PostInc()
		if old != x || z.GetRational().GetInt() != a+1 || z.GetInfinitesimal().GetInt() != b || z.Inc() != &z {
			t.Fatal("increment")
		}
		old = z.PostDec()
		if old.GetRational().GetInt() != a+2 || z.Dec() != &z || z != x {
			t.Fatal("decrement")
		}
		for _, y := range values {
			c, d := y.GetRational().GetInt(), y.GetInfinitesimal().GetInt()
			otherCode := int64(c)*100 + int64(d)
			if x.Equal(y) != (code == otherCode) || x.NotEqual(y) != (code != otherCode) || x.Less(y) != (code < otherCode) || x.Greater(y) != (code > otherCode) || x.LessEqual(y) != (code <= otherCode) || x.GreaterEqual(y) != (code >= otherCode) {
				t.Fatal("pair order", x, y)
			}
			if x.Add(y) != NewInfSIntegerPair(NewSInteger(a+c), NewSInteger(b+d)) || x.Sub(y) != NewInfSIntegerPair(NewSInteger(a-c), NewSInteger(b-d)) {
				t.Fatal("pair arithmetic", x, y)
			}
			z = x
			if z.AddAssign(y) != &z || z != x.Add(y) {
				t.Fatal("add assign")
			}
			z = x
			if z.SubAssign(y) != &z || z != x.Sub(y) {
				t.Fatal("sub assign")
			}
			z = x
			copy := y
			z.Swap(&copy)
			if z != y || copy != x {
				t.Fatal("swap")
			}
			z = x
			if z.Assign(y) != &z || z != y {
				t.Fatal("assign")
			}
		}
		for c := int32(-4); c <= 4; c++ {
			s := NewSInteger(c)
			scalarCode := int64(c) * 100
			if x.EqualSInteger(s) != (code == scalarCode) || x.NotEqualSInteger(s) != (code != scalarCode) || x.LessSInteger(s) != (code < scalarCode) || x.GreaterSInteger(s) != (code > scalarCode) || x.LessEqualSInteger(s) != (code <= scalarCode) || x.GreaterEqualSInteger(s) != (code >= scalarCode) {
				t.Fatal("right scalar comparison")
			}
			if s.EqualInfSInteger(x) != (scalarCode == code) || s.NotEqualInfSInteger(x) != (scalarCode != code) || s.LessInfSInteger(x) != (scalarCode < code) || s.GreaterInfSInteger(x) != (scalarCode > code) || s.LessEqualInfSInteger(x) != (scalarCode <= code) || s.GreaterEqualInfSInteger(x) != (scalarCode >= code) {
				t.Fatal("left scalar comparison")
			}
			z = x
			if z.AddAssignSInteger(s) != &z || z.GetRational().GetInt() != a+c || z.GetInfinitesimal().GetInt() != b {
				t.Fatal("scalar addition")
			}
			z = x
			z.SubAssignSInteger(s)
			if z.GetRational().GetInt() != a-c || z.GetInfinitesimal().GetInt() != b {
				t.Fatal("scalar subtraction")
			}
			z = x
			if z.MulAssign(s) != &z || z.GetRational().GetInt() != a*c || z.GetInfinitesimal().GetInt() != b*c || s.MulInfSInteger(x) != z {
				t.Fatal("scalar multiply")
			}
			if c != 0 {
				z = x
				if z.DivAssign(s) != &z || z.GetRational().GetInt() != a/c || z.GetInfinitesimal().GetInt() != b/c || x.Div(s) != z {
					t.Fatal("componentwise truncated divide")
				}
			}
			z = x
			z.Addmul(s, x)
			if z.GetRational().GetInt() != a+c*a || z.GetInfinitesimal().GetInt() != b+c*b {
				t.Fatal("addmul alias")
			}
			z = x
			z.Submul(s, x)
			if z.GetRational().GetInt() != a-c*a || z.GetInfinitesimal().GetInt() != b-c*b {
				t.Fatal("submul alias")
			}
			z = x
			if z.AssignSInteger(s) != &z || z != NewInfSInteger(c) {
				t.Fatal("scalar assignment clears epsilon")
			}
		}
	}
}

func TestInfSIntegerRationalConstructorAndAssignment(t *testing.T) {
	s, _ := serviceNew(t)
	for _, text := range []string{"-4294967297", "0", "4294967297", "1/2"} {
		v := serviceValue(t, s, text)
		got, e := InfSIntegerFromRational(s, &v)
		if sourceDebug && text == "1/2" {
			if e != ErrIntegerAssertion {
				t.Fatal(e)
			}
		} else {
			serviceMust(t, e)
			wide, _ := new(big.Rat).SetString(text)
			want := int32(wide.Num().Int64())
			if got != NewInfSInteger(want) {
				t.Fatal(text, got, want)
			}
		}
		x := NewInfSIntegerOffset(SIntegerOne(), true)
		saved := x
		cause := x.AssignRational(s, &v)
		if sourceDebug && text == "1/2" {
			if cause != ErrIntegerAssertion || x != saved {
				t.Fatal("getter error precedes assignment", x, cause)
			}
		} else if cause != nil || x != got {
			t.Fatal(x, got, cause)
		}
		serviceMust(t, s.Del(&v))
	}
}

func TestInfSIntegerDisplayOriginalNegativeCoefficientAndColdPrefixes(t *testing.T) {
	for _, tc := range []struct {
		a, b int32
		text string
	}{{3, 0, "3"}, {3, 2, "(3 +e*2)"}, {3, -2, "(3 -e*-2)"}, {-3, -2, "(-3 -e*-2)"}, {0, -2147483648, "(0 -e*-2147483648)"}} {
		value := NewInfSIntegerPair(NewSInteger(tc.a), NewSInteger(tc.b))
		if value.ToString() != tc.text {
			t.Fatal("Go convenience preserves source stream signs")
		}
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(^uint64(0))
			out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
			memoryRuntimeRequire(t, e)
			b.FailAfter(ordinal)
			cause := value.Display(out)
			stats := out.Stats()
			if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes || value.GetInfinitesimal().GetInt() != tc.b {
				t.Fatal("immediate display graph/value")
			}
			got, e := out.String()
			memoryRuntimeRequire(t, e)
			if !strings.HasPrefix(tc.text, got) || cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
				t.Fatal(got, cause)
			}
			b.FailAfter(-1)
			memoryRuntimeRequire(t, out.Close())
			if b.Used() != 0 {
				t.Fatal("refund")
			}
			if cause == nil {
				if got != tc.text {
					t.Fatal(got)
				}
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("completion horizon")
		}
	}
}

func TestInfSIntegerWideOrderingAndRepresentableArithmetic(t *testing.T) {
	values := []int32{-2147483648, -2147483647, -1, 0, 1, 2147483646, 2147483647}
	encode := func(a, b int32) *big.Int {
		x := new(big.Int).Lsh(big.NewInt(int64(a)), 33)
		return x.Add(x, big.NewInt(int64(b)))
	}
	fits := func(n int64) bool { return n >= -2147483648 && n <= 2147483647 }
	for _, a := range values {
		for _, b := range values {
			x := NewInfSIntegerPair(NewSInteger(a), NewSInteger(b))
			if !(a == -2147483648 && b < 0) && x.Floor().GetInt64() != int64(a)-infSIntegerTestBoolInt64(b < 0) {
				t.Fatal("boundary floor", a, b)
			}
			if !(a == 2147483647 && b > 0) && x.Ceil().GetInt64() != int64(a)+infSIntegerTestBoolInt64(b > 0) {
				t.Fatal("boundary ceil", a, b)
			}
			for _, c := range values {
				for _, d := range values {
					y := NewInfSIntegerPair(NewSInteger(c), NewSInteger(d))
					order := encode(a, b).Cmp(encode(c, d))
					if x.Less(y) != (order < 0) || x.Equal(y) != (order == 0) {
						t.Fatal("wide lexicographic order", a, b, c, d)
					}
					if first, second := int64(a)+int64(c), int64(b)+int64(d); fits(first) && fits(second) {
						got := x.Add(y)
						if got.GetRational().GetInt64() != first || got.GetInfinitesimal().GetInt64() != second {
							t.Fatal("wide addition")
						}
					}
					if first, second := int64(a)-int64(c), int64(b)-int64(d); fits(first) && fits(second) {
						got := x.Sub(y)
						if got.GetRational().GetInt64() != first || got.GetInfinitesimal().GetInt64() != second {
							t.Fatal("wide subtraction")
						}
					}
				}
			}
		}
	}
}

func infSIntegerTestBoolInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func TestInfSIntegerFileFormattingFlagsAndBorrow(t *testing.T) {
	f, out, b := numericFileFixture(t)
	memoryRuntimeRequire(t, out.SetIntegerBase(16))
	memoryRuntimeRequire(t, out.SetShowBase(true))
	memoryRuntimeRequire(t, out.SetShowPos(true))
	memoryRuntimeRequire(t, out.SetWidth(4))
	memoryRuntimeRequire(t, out.SetFill('_'))
	x := NewInfSIntegerPair(NewSInteger(-3), NewSInteger(-2))
	before := b.Used()
	b.FailAfter(0)
	memoryRuntimeRequire(t, x.Display(out))
	if b.Used() != before {
		t.Fatal("file display allocated")
	}
	b.FailAfter(-1)
	_, e := f.Seek(0, io.SeekStart)
	memoryRuntimeRequire(t, e)
	got, e := io.ReadAll(f)
	memoryRuntimeRequire(t, e)
	if string(got) != "___(0xfffffffd -e*0xfffffffe)" {
		t.Fatal(string(got))
	}
	memoryRuntimeRequire(t, out.Close())
	if _, e = f.Write([]byte("caller")); e != nil {
		t.Fatal("borrowed file closed", e)
	}
	if e = x.Display(out); e != arena.ErrClosed {
		t.Fatal("closed stream", e)
	}
	if e = x.Display(nil); e != ErrNumericStreamOwner {
		t.Fatal("invalid stream", e)
	}
}
