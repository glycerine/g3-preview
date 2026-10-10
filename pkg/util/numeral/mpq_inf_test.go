// Source: src/util/mpq_inf.h:26-279; src/util/mpq_inf.cpp:22-43.
package numeral

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uhash "github.com/glycerine/g3/pkg/util/hash"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func mpqInfFixture(t *testing.T, sync bool) (*MPQInfManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	p, e := NewMPQInfManager(RationalConfig{Integers: serviceConfig(b).Integers, Synchronized: sync})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, p.Close())
		if b.Used() != 0 {
			t.Fatal("retained pair manager", b.Used())
		}
	})
	return p, b
}
func mpqInfValue(t *testing.T, p *MPQInfManager, first, second string) MPQInf {
	t.Helper()
	x := NewMPQInf()
	serviceMust(t, p.m.SetString(&x.First, first))
	serviceMust(t, p.m.SetString(&x.Second, second))
	return x
}
func mpqInfText(t *testing.T, p *MPQInfManager, x *MPQInf, first, second string) {
	t.Helper()
	if rationalString(t, p.m, &x.First) != first || rationalString(t, p.m, &x.Second) != second {
		t.Fatal("pair result", first, second)
	}
}

func TestMPQInfSourceFloorQuirkAndEpsilonApproximation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		for _, tc := range []struct{ first, second, floor, ceil string }{{"-3", "0", "-4", "-3"}, {"-3", "-9/2", "-4", "-3"}, {"-3", "9/2", "-4", "-2"}, {"3", "-9/2", "3", "3"}, {"3", "9/2", "3", "4"}, {"0", "-1", "0", "0"}, {"0", "1", "0", "1"}, {"-7/2", "1", "-4", "-3"}, {"7/2", "-1", "3", "4"}} {
			x := mpqInfValue(t, p, tc.first, tc.second)
			out := NewRational()
			serviceMust(t, p.Floor(&x, &out))
			if rationalString(t, p.m, &out) != tc.floor {
				t.Fatal(tc)
			}
			serviceMust(t, p.Ceil(&x, &out))
			if rationalString(t, p.m, &out) != tc.ceil {
				t.Fatal(tc)
			}
			serviceMust(t, p.m.Del(&out))
			serviceMust(t, p.Del(&x))
		}
		for _, epsilon := range []string{"1", "20000000000000000000000000000", "-1", "-20000000000000000000000000000", "0"} {
			x := mpqInfValue(t, p, "5/2", epsilon)
			got, e := p.GetDouble(&x)
			serviceMust(t, e)
			want := 2.5
			if epsilon[0] == '-' {
				want -= 0.0001
			} else if epsilon != "0" {
				want += 0.0001
			}
			if got != want {
				t.Fatal("epsilon magnitude is ignored", epsilon, got, want)
			}
			serviceMust(t, p.SetInf(-0.5))
			got, e = p.GetDouble(&x)
			serviceMust(t, e)
			want = 2.5
			if epsilon[0] == '-' {
				want += 0.5
			} else if epsilon != "0" {
				want -= 0.5
			}
			if got != want {
				t.Fatal(got, want)
			}
			serviceMust(t, p.SetInf(0.0001))
			serviceMust(t, p.Del(&x))
		}
		x := mpqInfValue(t, p, "1", "1")
		serviceMust(t, p.SetInf(math.NaN()))
		got, e := p.GetDouble(&x)
		serviceMust(t, e)
		if !math.IsNaN(got) {
			t.Fatal(got)
		}
		serviceMust(t, p.Del(&x))
	}
}

func TestMPQInfIndependentArithmeticOrderingAndAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		pairs := [][2]string{{"-7/3", "2/5"}, {"0", "-1/3"}, {"0", "0"}, {"0", "1/3"}, {"5/7", "-9/2"}, {"5/7", "0"}, {"5/7", "11/3"}, {"123456789012345678901/7", "-9/2"}}
		for _, a := range pairs {
			for _, b := range pairs {
				x := mpqInfValue(t, p, a[0], a[1])
				y := mpqInfValue(t, p, b[0], b[1])
				out := NewMPQInf()
				a0, _ := new(big.Rat).SetString(a[0])
				a1, _ := new(big.Rat).SetString(a[1])
				b0, _ := new(big.Rat).SetString(b[0])
				b1, _ := new(big.Rat).SetString(b[1])
				order := a0.Cmp(b0)
				if order == 0 {
					order = a1.Cmp(b1)
				}
				lt, e := p.Lt(&x, &y)
				serviceMust(t, e)
				eq, e := p.Eq(&x, &y)
				serviceMust(t, e)
				gt, e := p.Gt(&x, &y)
				serviceMust(t, e)
				le, e := p.Le(&x, &y)
				serviceMust(t, e)
				ge, e := p.Ge(&x, &y)
				serviceMust(t, e)
				if lt != (order < 0) || eq != (order == 0) || gt != (order > 0) || le != (order <= 0) || ge != (order >= 0) {
					t.Fatal("independent pair order", a, b)
				}
				serviceMust(t, p.Add(&x, &y, &out))
				mpqInfText(t, p, &out, new(big.Rat).Add(a0, b0).RatString(), new(big.Rat).Add(a1, b1).RatString())
				serviceMust(t, p.Sub(&x, &y, &out))
				mpqInfText(t, p, &out, new(big.Rat).Sub(a0, b0).RatString(), new(big.Rat).Sub(a1, b1).RatString())
				serviceMust(t, p.Add(&x, &y, &x))
				mpqInfText(t, p, &x, new(big.Rat).Add(a0, b0).RatString(), new(big.Rat).Add(a1, b1).RatString())
				serviceMust(t, p.Sub(&x, &y, &y))
				mpqInfText(t, p, &y, a0.RatString(), a1.RatString())
				serviceMust(t, p.Del(&x))
				serviceMust(t, p.Del(&y))
				serviceMust(t, p.Del(&out))
			}
		}
		for _, scalar := range []string{"-3/2", "2/3"} {
			x := mpqInfValue(t, p, "5/7", "-11/13")
			r := NewRational()
			serviceMust(t, p.m.SetString(&r, scalar))
			out := NewMPQInf()
			s, _ := new(big.Rat).SetString(scalar)
			a, _ := new(big.Rat).SetString("5/7")
			b, _ := new(big.Rat).SetString("-11/13")
			serviceMust(t, p.MulRational(&x, &r, &out))
			mpqInfText(t, p, &out, new(big.Rat).Mul(a, s).RatString(), new(big.Rat).Mul(b, s).RatString())
			serviceMust(t, p.DivRational(&out, &r, &out))
			mpqInfText(t, p, &out, a.RatString(), b.RatString())
			serviceMust(t, p.AddRational(&x, &r, &out))
			mpqInfText(t, p, &out, new(big.Rat).Add(a, s).RatString(), b.RatString())
			serviceMust(t, p.SubRational(&out, &r, &out))
			mpqInfText(t, p, &out, a.RatString(), b.RatString())
			serviceMust(t, p.Del(&x))
			serviceMust(t, p.Del(&out))
			serviceMust(t, p.m.Del(&r))
		}
	}
}

func TestMPQInfSourceShortCircuitReads(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		// Invalid second components expose reads that the original guards erase.
		x := NewMPQInf()
		serviceMust(t, p.m.SetInt32(&x.First, 3))
		x.Second = Rational{}
		pos, e := p.IsPos(&x)
		serviceMust(t, e)
		if !pos {
			t.Fatal("positive first short circuit")
		}
		y := NewMPQInf()
		serviceMust(t, p.m.SetInt32(&y.First, 4))
		y.Second = Rational{}
		lt, e := p.Lt(&x, &y)
		serviceMust(t, e)
		if !lt {
			t.Fatal("first comparison short circuit")
		}
		eq, e := p.Eq(&x, &y)
		serviceMust(t, e)
		if eq {
			t.Fatal("unequal first short circuit")
		}
		out := NewRational()
		serviceMust(t, p.Floor(&x, &out))
		if rationalString(t, p.m, &out) != "3" {
			t.Fatal("floor ignores second entirely")
		}
		serviceMust(t, p.m.SetString(&x.First, "7/2"))
		serviceMust(t, p.Ceil(&x, &out))
		if rationalString(t, p.m, &out) != "4" {
			t.Fatal("noninteger ceil ignores second")
		}
		serviceMust(t, p.m.Del(&out))
		serviceMust(t, p.m.Del(&x.First))
		serviceMust(t, p.m.Del(&y.First))
	}
}

func TestMPQInfScalarKindsSettersAndUnaryBranches(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		for _, first := range []string{"-3/2", "0", "5/7"} {
			for _, second := range []string{"-7/2", "-1", "0", "1", "7/2"} {
				for _, scalar := range []string{"-3/2", "0", "5/7"} {
					x := mpqInfValue(t, p, first, second)
					b := NewRational()
					serviceMust(t, p.m.SetString(&b, scalar))
					f, _ := new(big.Rat).SetString(first)
					s, _ := new(big.Rat).SetString(second)
					r, _ := new(big.Rat).SetString(scalar)
					for k := MPQInfNegative; k <= MPQInfPositive; k++ {
						order := f.Cmp(r)
						if order == 0 {
							order = s.Cmp(new(big.Rat).SetInt64(int64(k)))
						}
						eq, e := p.EqKind(&x, &b, k)
						serviceMust(t, e)
						lt, e := p.LtKind(&x, &b, k)
						serviceMust(t, e)
						gt, e := p.GtKind(&x, &b, k)
						serviceMust(t, e)
						le, e := p.LeKind(&x, &b, k)
						serviceMust(t, e)
						ge, e := p.GeKind(&x, &b, k)
						serviceMust(t, e)
						if eq != (order == 0) || lt != (order < 0) || gt != (order > 0) || le != (order <= 0) || ge != (order >= 0) {
							t.Fatal("kind order", first, second, scalar, k)
						}
					}
					order := f.Cmp(r)
					if order == 0 {
						order = s.Sign()
					}
					eq, e := p.EqRational(&x, &b)
					serviceMust(t, e)
					lt, e := p.LtRational(&x, &b)
					serviceMust(t, e)
					gt, e := p.GtRational(&x, &b)
					serviceMust(t, e)
					le, e := p.LeRational(&x, &b)
					serviceMust(t, e)
					ge, e := p.GeRational(&x, &b)
					serviceMust(t, e)
					if eq != (order == 0) || lt != (order < 0) || gt != (order > 0) || le != (order <= 0) || ge != (order >= 0) {
						t.Fatal("rational order")
					}
					serviceMust(t, p.Del(&x))
					serviceMust(t, p.m.Del(&b))
				}
			}
		}
		x := mpqInfValue(t, p, "-5/7", "9/2")
		y := NewMPQInf()
		serviceMust(t, p.Set(&y, &x))
		serviceMust(t, p.Inc(&y))
		mpqInfText(t, p, &y, "2/7", "9/2")
		serviceMust(t, p.Dec(&y))
		serviceMust(t, p.Abs(&y))
		mpqInfText(t, p, &y, "5/7", "-9/2")
		serviceMust(t, p.Neg(&y))
		mpqInfText(t, p, &y, "-5/7", "9/2")
		r := NewRational()
		serviceMust(t, p.GetRational(&y, &r))
		if rationalString(t, p.m, &r) != "-5/7" {
			t.Fatal("projection")
		}
		serviceMust(t, p.GetInfinitesimal(&y, &r))
		if rationalString(t, p.m, &r) != "9/2" {
			t.Fatal("epsilon projection")
		}
		serviceMust(t, p.SetRational(&y, &r))
		mpqInfText(t, p, &y, "9/2", "0")
		serviceMust(t, p.SetKind(&y, &r, MPQInfNegative))
		mpqInfText(t, p, &y, "9/2", "-1")
		serviceMust(t, p.SetKind(&y, &r, MPQInfPositive))
		mpqInfText(t, p, &y, "9/2", "1")
		serviceMust(t, p.SetKind(&y, &r, MPQInfZero))
		mpqInfText(t, p, &y, "9/2", "0")
		serviceMust(t, p.SetPair(&y, &x.First, &x.Second))
		serviceMust(t, p.SetKind(&y, &r, MPQInfKind(9)))
		mpqInfText(t, p, &y, "9/2", "9/2")
		serviceMust(t, p.Swap(&x, &y))
		mpqInfText(t, p, &x, "9/2", "9/2")
		mpqInfText(t, p, &y, "-5/7", "9/2")
		h, e := p.Hash(&y)
		serviceMust(t, e)
		h1, e := p.m.Hash(&y.First)
		serviceMust(t, e)
		h2, e := p.m.Hash(&y.Second)
		serviceMust(t, e)
		if h != uhash.UintPair(h1, h2) {
			t.Fatal("combined source hashes")
		}
		serviceMust(t, p.Reset(&x))
		zero, e := p.IsZero(&x)
		serviceMust(t, e)
		integer, e := p.IsInt(&x)
		serviceMust(t, e)
		rational, e := p.IsRational(&x)
		serviceMust(t, e)
		if !zero || !integer || !rational {
			t.Fatal("reset/traits")
		}
		n := NewInt(-3)
		serviceMust(t, p.MulInteger(&y, &n, &x))
		mpqInfText(t, p, &x, "15/7", "-27/2")
		serviceMust(t, p.DivInteger(&x, &n, &x))
		mpqInfText(t, p, &x, "-5/7", "9/2")
		serviceMust(t, p.Del(&x))
		serviceMust(t, p.Del(&y))
		serviceMust(t, p.m.Del(&r))
	}
}

func TestMPQInfSourceSequentialCrossAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		x := mpqInfValue(t, p, "2", "3")
		serviceMust(t, p.MulRational(&x, &x.First, &x))
		mpqInfText(t, p, &x, "4", "12")
		serviceMust(t, p.m.SetInt32(&x.First, 8))
		serviceMust(t, p.m.SetInt32(&x.Second, 6))
		serviceMust(t, p.DivRational(&x, &x.First, &x))
		mpqInfText(t, p, &x, "1", "6")
		serviceMust(t, p.m.SetInt32(&x.First, 2))
		serviceMust(t, p.m.SetInt32(&x.Second, 3))
		serviceMust(t, p.SetPair(&x, &x.Second, &x.First))
		mpqInfText(t, p, &x, "3", "3")
		serviceMust(t, p.Del(&x))
		copy := *p
		if copy.SetInf(1) != ErrMPQInfOwner || copy.Close() != ErrMPQInfOwner {
			t.Fatal("copied owner")
		}
		m, e := p.GetMPQManager()
		serviceMust(t, e)
		if m != p.m {
			t.Fatal("borrowed member identity")
		}
	}
}

func TestMPQInfEveryColdNumericReservationAndCleanup(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, mode := range []string{"set", "add", "sub", "mul", "div", "lt", "floor"} {
			completed := false
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				p, b := mpqInfFixture(t, sync)
				x := mpqInfValue(t, p, "123456789012345678901234567890123456789/43", "-123456789012345678901234567890123456789/11")
				y := mpqInfValue(t, p, "123456789012345678901234567890123456790/47", "123456789012345678901234567890123456791/13")
				out := mpqInfValue(t, p, "123456789012345678901234567890123456792/17", "123456789012345678901234567890123456793/19")
				beforeX, beforeY := x, y
				b.FailAfter(ordinal)
				var cause error
				switch mode {
				case "set":
					cause = p.Set(&out, &x)
				case "add":
					cause = p.Add(&x, &y, &out)
				case "sub":
					cause = p.Sub(&x, &y, &out)
				case "mul":
					cause = p.MulRational(&x, &y.First, &out)
				case "div":
					cause = p.DivRational(&x, &y.First, &out)
				case "lt":
					_, cause = p.Lt(&x, &y)
				case "floor":
					cause = p.Floor(&x, &out.First)
				}
				if cause != nil && !numericPhysicalReservationRefusal(cause) {
					t.Fatal(mode, ordinal, cause)
				}
				if x != beforeX || y != beforeY {
					t.Fatal("const input publication", mode, ordinal)
				}
				qopsAuditLiveOwnership(t, p.m, []*Int{&x.First.Num, &x.First.Den, &x.Second.Num, &x.Second.Den, &y.First.Num, &y.First.Den, &y.Second.Num, &y.Second.Den, &out.First.Num, &out.First.Den, &out.Second.Num, &out.Second.Den})
				if b.Used() != p.header+p.m.HeaderBytes+integerActualOwnedBytes(p.m.Z) {
					t.Fatal("immediate pair owner budget", mode, ordinal, b.Used(), p.header+p.m.HeaderBytes+integerActualOwnedBytes(p.m.Z))
				}
				if cause != nil {
					if _, e := p.IsZero(&out); e != cause {
						t.Fatal("sticky source failure", mode, ordinal, e, cause)
					}
				}
				b.FailAfter(-1)
				serviceMust(t, p.CleanupValue(&out))
				serviceMust(t, p.CleanupValue(&y))
				serviceMust(t, p.CleanupValue(&x))
				serviceMust(t, p.Close())
				if b.Used() != 0 {
					t.Fatal("owning closure", mode, ordinal, b.Used())
				}
				if cause == nil {
					completed = true
					break
				}
			}
			if !completed {
				t.Fatal("no completion horizon", sync, mode)
			}
		}
	}
}

func TestMPQInfConstructorEveryReservation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(1 << 26)
			b.FailAfter(ordinal)
			p, e := NewMPQInfManager(RationalConfig{Integers: serviceConfig(b).Integers, Synchronized: sync})
			b.FailAfter(-1)
			if e != nil {
				constructionFailureOwningCleanup(t, e, b)
				if p != nil || b.Used() != 0 {
					t.Fatal("constructor graph", ordinal, e, b.Used())
				}
				continue
			}
			serviceMust(t, p.Close())
			if b.Used() != 0 {
				t.Fatal("constructor closure")
			}
			completed = true
			break
		}
		if !completed {
			t.Fatal("constructor completion horizon")
		}
	}
}

func TestMPQInfOwnedFormattingSignsFractionsAndLifetime(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, tc := range []struct{ first, second, want string }{
			{"0", "0", "0"}, {"-3", "0", "-3"}, {"3", "-2", "(3 -e*2)"}, {"-3/2", "-2/3", "(-3/2 -e*2/3)"}, {"3/2", "2/3", "(3/2 +e*2/3)"},
			{"123456789012345678901234567890123456789/23", "-987654321098765432109876543210987654321/17", "(123456789012345678901234567890123456789/23 -e*58097313005809731300580973130058097313)"},
		} {
			p, b := mpqInfFixture(t, sync)
			x := mpqInfValue(t, p, tc.first, tc.second)
			var text utext.OwnedString
			serviceMust(t, p.ToString(&x, &text))
			got, e := text.View().String()
			serviceMust(t, e)
			if got != tc.want {
				t.Fatal(sync, tc, got)
			}
			if p.format.active || p.format.tmpLive {
				t.Fatal("successful formatting left scope")
			}
			serviceMust(t, p.Del(&x))
			serviceMust(t, p.Close())
			got, e = text.View().String()
			serviceMust(t, e)
			if got != tc.want || b.Used() != text.StorageBytes() {
				t.Fatal("return ownership", got, b.Used())
			}
			text.SourceDestroy()
		}
	}
}

func TestMPQInfDisplayWholeStringAndFlags(t *testing.T) {
	for _, align := range []utext.NumericAlignment{utext.NumericLeft, utext.NumericRight, utext.NumericInternal} {
		p, b := mpqInfFixture(t, false)
		x := mpqInfValue(t, p, "3/2", "-2/3")
		out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
		serviceMust(t, e)
		serviceMust(t, out.SetIntegerBase(16))
		serviceMust(t, out.SetShowBase(true))
		serviceMust(t, out.SetShowPos(true))
		serviceMust(t, out.SetFill('_'))
		serviceMust(t, out.SetAlignment(align))
		serviceMust(t, out.SetWidth(20))
		serviceMust(t, p.Display(out, &x))
		got, e := out.String()
		serviceMust(t, e)
		want := ""
		// Build padding from the independently expected whole-string length.
		literal := "(3/2 -e*2/3)"
		padding := ""
		for i := len(literal); i < 20; i++ {
			padding += "_"
		}
		want = padding + literal
		if align == utext.NumericLeft {
			want = literal + padding
		}
		if got != want || out.State().Width != 0 || out.State().IntegerBase != 16 || !out.State().ShowPos {
			t.Fatal(got, want, out.State())
		}
		if p.format.active {
			t.Fatal("display retained success")
		}
		serviceMust(t, p.Del(&x))
		serviceMust(t, out.Close())
		serviceMust(t, p.Close())
	}
}

func mpqInfFormattingBytes(p *MPQInfManager, ret *utext.OwnedString) uint64 {
	s := &p.format
	strings := [5]*utext.OwnedString{&s.number, &s.denominator, &s.display, &s.stream.Buffer.String, ret}
	var bytes uint64
	for _, v := range strings {
		if v != nil {
			bytes += v.StorageBytes()
		}
	}
	return bytes
}

func TestMPQInfEveryColdFormattingAndDisplayReservation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, mode := range []string{"zero", "pair", "display", "large"} {
			completed := false
			retainedTmp := false
			for ordinal := int64(0); ordinal < 200; ordinal++ {
				p, b := mpqInfFixture(t, sync)
				first := "123456789012345678901234567890123456789/23"
				second := "-987654321098765432109876543210987654321/19"
				if mode == "zero" {
					second = "0"
				}
				if mode == "large" {
					first = strings.Repeat("9", 600) + "/23"
					second = "-" + strings.Repeat("7", 600) + "/19"
				}
				x := mpqInfValue(t, p, first, second)
				before := x // non-owning snapshot, never Del the snapshot.
				var text utext.OwnedString
				var out *utext.NumericTextStream
				if mode == "display" {
					var e error
					out, e = utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
					serviceMust(t, e)
				}
				b.FailAfter(ordinal)
				var e error
				if out == nil {
					e = p.ToString(&x, &text)
				} else {
					e = p.Display(out, &x)
				}
				roots := []*Int{&x.First.Num, &x.First.Den, &x.Second.Num, &x.Second.Den}
				if p.format.tmpLive {
					roots = append(roots, &p.format.tmp.Num, &p.format.tmp.Den)
					retainedTmp = true
				}
				qopsAuditLiveOwnership(t, p.m, roots)
				actual := p.header + p.m.HeaderBytes + integerActualOwnedBytes(p.m.Z) + mpqInfFormattingBytes(p, &text)
				if out != nil {
					st := out.Stats()
					actual += st.OwnerBytes + st.ScratchBytes + st.OutputOwnerBytes + st.OutputOverflowBytes
				}
				if b.Used() != actual {
					t.Fatal(sync, mode, ordinal, "immediate physical graph", b.Used(), actual)
				}
				if x != before {
					t.Fatal("const input mutation", sync, mode, ordinal)
				}
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(sync, mode, ordinal, e)
					}
					if !p.format.active {
						t.Fatal("failure dropped scope")
					}
					var untouched utext.OwnedString
					if p.ToString(&x, &untouched) != e || untouched.Self != nil {
						t.Fatal("sticky formatting replay")
					}
					if _, sticky := p.IsZero(&x); sticky != e {
						t.Fatal("sticky numeric replay")
					}
				} else {
					got := ""
					var err error
					if out == nil {
						got, err = text.View().String()
					} else {
						got, err = out.String()
					}
					serviceMust(t, err)
					f, ok := new(big.Rat).SetString(first)
					if !ok {
						t.Fatal("oracle")
					}
					want := f.RatString()
					if mode != "zero" {
						eps, ok := new(big.Rat).SetString(second)
						if !ok {
							t.Fatal("epsilon oracle")
						}
						eps.Abs(eps)
						want = "(" + want + " -e*" + eps.RatString() + ")"
					}
					if got != want {
						t.Fatal(sync, mode, ordinal, got, want)
					}
					completed = true
				}
				b.FailAfter(-1)
				serviceMust(t, p.CleanupValue(&x))
				if e != nil && (text.External != nil) {
					t.Fatal("return slot not reclaimed")
				}
				text.OwnAbort()
				if out != nil {
					serviceMust(t, out.Close())
				}
				serviceMust(t, p.Close())
				if b.Used() != 0 {
					t.Fatal("formatting refund", sync, mode, ordinal, b.Used())
				}
				if completed {
					break
				}
			}
			if !completed {
				t.Fatal("format completion horizon", sync, mode)
			}
			if mode == "negative" && !retainedTmp {
				t.Fatal("raw temporary failure path unexercised", sync, mode)
			}
		}
	}
}

func TestMPQInfFormattingIndependentRationalOracle(t *testing.T) {
	random := rand.New(rand.NewChaCha8([32]byte{81, 42, 9}))
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		for i := 0; i < 2000; i++ {
			fn := int64(random.IntN(200001) - 100000)
			fd := int64(random.IntN(1000) + 1)
			en := int64(random.IntN(200001) - 100000)
			ed := int64(random.IntN(1000) + 1)
			if i%11 == 0 {
				en = 0
			}
			first, second := fmt.Sprintf("%d/%d", fn, fd), fmt.Sprintf("%d/%d", en, ed)
			x := mpqInfValue(t, p, first, second)
			var out utext.OwnedString
			serviceMust(t, p.ToString(&x, &out))
			got, e := out.View().String()
			serviceMust(t, e)
			f := new(big.Rat).SetFrac(big.NewInt(fn), big.NewInt(fd))
			epsilon := new(big.Rat).SetFrac(big.NewInt(en), big.NewInt(ed))
			want := f.RatString()
			if en != 0 {
				sign := " +e*"
				if en < 0 {
					sign = " -e*"
				}
				epsilon.Abs(epsilon)
				want = "(" + want + sign + epsilon.RatString() + ")"
			}
			if got != want {
				t.Fatal(sync, i, first, second, got, want)
			}
			mpqInfText(t, p, &x, f.RatString(), new(big.Rat).SetFrac(big.NewInt(en), big.NewInt(ed)).RatString())
			out.SourceDestroy()
			serviceMust(t, p.Del(&x))
		}
	}
}

func TestMPQInfFormattingReadErasureAndExplicitCleanup(t *testing.T) {
	p, b := mpqInfFixture(t, false)
	x := mpqInfValue(t, p, "3", "0")
	x.Second.Den = NewInt(0) // untouched poisoned denominator in zero epsilon branch.
	b.FailAfter(0)
	var text utext.OwnedString
	serviceMust(t, p.ToString(&x, &text))
	got, e := text.View().String()
	serviceMust(t, e)
	if got != "3" || p.format.tmpLive {
		t.Fatal("epsilon denominator read or tmp acquired")
	}
	text.SourceDestroy()
	b.FailAfter(-1)
	serviceMust(t, p.Del(&x))
	for _, mode := range []string{"nil", "closed"} {
		p, b := mpqInfFixture(t, false)
		x := mpqInfValue(t, p, "123456789012345678901234567890123456789", "-2")
		var out *utext.NumericTextStream
		if mode == "closed" {
			out, e = utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
			serviceMust(t, e)
			serviceMust(t, out.Close())
		}
		e = p.Display(out, &x)
		wantError := utext.ErrNumericStreamOwner
		if mode == "closed" {
			wantError = arena.ErrClosed
		}
		if !errors.Is(e, wantError) {
			t.Fatal(mode, e)
		}
		if !p.format.active || p.format.tmpLive || !p.format.display.Matches("(123456789012345678901234567890123456789 -e*2)") {
			t.Fatal("completed temporary lost on insertion failure")
		}
		b.FailAfter(0)
		serviceMust(t, p.CleanupFormatting())
		serviceMust(t, p.CleanupFormatting())
		if p.format.active {
			t.Fatal("scope not joined")
		}
		var fresh utext.OwnedString
		if p.ToString(&x, &fresh) != e || fresh.Self != nil {
			t.Fatal("cleanup resumed source execution")
		}
		b.FailAfter(-1)
		serviceMust(t, p.CleanupValue(&x))
		serviceMust(t, p.Close())
	}
}

func TestMPQInfDisplayActualFileBorrowAndNonthrowingFailure(t *testing.T) {
	f, out, _ := numericFileFixture(t)
	p, b := mpqInfFixture(t, false)
	x := mpqInfValue(t, p, "3", "-2")
	used := b.Used()
	b.FailAfter(0) // all reached numeral/string objects fit their real inline storage.
	serviceMust(t, out.SetWidth(12))
	serviceMust(t, out.SetFill('_'))
	serviceMust(t, p.Display(out, &x))
	if b.Used() != used || p.format.active {
		t.Fatal("inline file display storage")
	}
	_, e := f.Seek(0, io.SeekStart)
	serviceMust(t, e)
	actual, e := io.ReadAll(f)
	serviceMust(t, e)
	if string(actual) != "____(3 -e*2)" {
		t.Fatal(string(actual))
	}
	serviceMust(t, f.Close())
	serviceMust(t, p.Display(out, &x))
	if !out.FileBad || p.format.active {
		t.Fatal("nonthrowing failed write retained temporary")
	}
	serviceMust(t, out.SetWidth(19))
	serviceMust(t, p.Display(out, &x))
	if out.State().Width != 19 || p.format.active || b.Used() != used {
		t.Fatal("failed sentry branch")
	}
	b.FailAfter(-1)
	serviceMust(t, p.Del(&x))
	serviceMust(t, p.Close())
}

func TestMPQInfFormattingCloseJoinsAbandonedScope(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		p, b := mpqInfFixture(t, false)
		x := mpqInfValue(t, p, "123456789012345678901234567890123456789/23", "-987654321098765432109876543210987654321/19")
		b.FailAfter(ordinal)
		var text utext.OwnedString
		e := p.ToString(&x, &text)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal(ordinal, e)
		}
		if e == nil {
			text.SourceDestroy()
			completed = true
		}
		b.FailAfter(-1)
		// Abandon caller numerals too: manager Close owns its physical allocations.
		serviceMust(t, p.Close())
		serviceMust(t, p.Close())
		if b.Used() != 0 || p.format.active || text.External != nil {
			t.Fatal("abandoned scope", ordinal, b.Used())
		}
		if e != nil && text.Self != nil && !text.Closed {
			t.Fatal("failure return slot remained open")
		}
		if completed {
			break
		}
	}
	if !completed {
		t.Fatal("close completion horizon")
	}
}
