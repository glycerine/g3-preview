// Source: src/util/mpz.cpp:1412-1429; src/util/mpq.cpp:142-153.
package numeral

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestRationalSMT2StreamLiteral(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, v := range []struct {
		n, d    string
		decimal bool
		want    string
	}{
		{"-2147483648", "1", false, "(- 2147483648)"}, {"0", "1", false, "0"}, {"-7", "1", true, "(- 7.0)"}, {"-3", "2", false, "(/ (- 3) 2)"}, {"3", "-2", true, "(/ 3.0 (- 2.0))"}, {"9", "0", false, "(/ 9 0)"},
		{"-123456789012345678901234567890", "98765432109876543210987654321", true, "(/ (- 123456789012345678901234567890.0) 98765432109876543210987654321.0)"},
	} {
		a := NewRational()
		serviceMust(t, m.integer.SetString(&a.Num, v.n))
		serviceMust(t, m.integer.SetString(&a.Den, v.d))
		before := a
		var out utext.OwnedStringStream
		serviceMust(t, out.Construct(m.rational.BudgetField))
		serviceMust(t, out.AppendCString("prefix:"))
		serviceMust(t, m.rational.DisplaySMT2Into(&out, &a, v.decimal))
		got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != "prefix:"+v.want || a != before {
			t.Fatal("literal SMT2 stream", v, out.Buffer.String)
		}
		out.Buffer.String.SourceDestroy()
		serviceMust(t, m.rational.Del(&a))
	}
}

func TestRationalSMT2StreamIndependentIntegers(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{14, 12, 14, 29}))
	numeral := func(v *big.Int, decimal bool) string {
		text := new(big.Int).Abs(v).String()
		if decimal {
			text += ".0"
		}
		if v.Sign() < 0 {
			text = "(- " + text + ")"
		}
		return text
	}
	for i := 0; i < 96; i++ {
		n := new(big.Int).SetUint64(rng.Uint64())
		n.Lsh(n, uint(rng.IntN(200)))
		d := new(big.Int).SetUint64(rng.Uint64())
		d.Lsh(d, uint(rng.IntN(120)))
		if i%2 == 0 {
			n.Neg(n)
		}
		if i%3 == 0 {
			d.Neg(d)
		}
		if i%7 == 0 {
			d.SetInt64(1)
		}
		if i%11 == 0 {
			d.SetInt64(0)
		}
		a := NewRational()
		serviceMust(t, m.integer.SetString(&a.Num, n.String()))
		serviceMust(t, m.integer.SetString(&a.Den, d.String()))
		for _, decimal := range []bool{false, true} {
			var out utext.OwnedStringStream
			serviceMust(t, out.Construct(b))
			serviceMust(t, m.rational.DisplaySMT2Into(&out, &a, decimal))
			want := numeral(n, decimal)
			if d.Cmp(big.NewInt(1)) != 0 {
				want = "(/ " + want + " " + numeral(d, decimal) + ")"
			}
			got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
			serviceMust(t, e)
			if got != want {
				t.Fatal("independent SMT2 syntax", i, decimal, got, want)
			}
			out.Buffer.OwnAbort()
		}
		serviceMust(t, m.rational.Del(&a))
	}
}

func TestRationalSMT2StreamAuthorityAndPreconditions(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	a := Rational{Num: NewInt(-7), Den: NewInt(3)}
	var out utext.OwnedStringStream
	serviceMust(t, out.Construct(b))
	initial := m.rational.smt2Stream.invocation
	if e := m.rational.DisplaySMT2Into(nil, &a, false); !errors.Is(e, utext.ErrOwnedStringOwner) || m.rational.smt2Stream.invocation != initial {
		t.Fatal("nil stream began parent", e)
	}
	if e := m.integer.DisplaySMT2Into(nil, &a.Num, false); !errors.Is(e, utext.ErrOwnedStringOwner) || m.integer.smt2Stream.invocation != 0 {
		t.Fatal("nil stream began child", e)
	}
	copied := *m.rational
	if e := copied.DisplaySMT2Into(&out, &a, false); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("copied rational owner", e)
	}
	serviceMust(t, m.rational.DisplaySMT2Into(&out, &a, false))
	scope := m.integer.smt2Stream
	if scope.valid() || scope.NumericRoots(m.integer, nil) != nil || scope.NumericExecuting() {
		t.Fatal("copied scope gained authority")
	}
	parent := m.rational.smt2Stream
	if parent.valid() || parent.NumericExecuting() {
		t.Fatal("copied parent gained authority")
	}
	out.Buffer.OwnAbort()
	childInvocation := m.integer.smt2Stream.invocation
	m.integer.smt2Stream.invocation = ^uint64(0)
	if e := m.integer.DisplaySMT2Into(&out, &a.Num, false); !errors.Is(e, ErrNumericPending) || m.integer.smt2Stream.active {
		t.Fatal("child invocation wrapped", e)
	}
	m.integer.smt2Stream.invocation = childInvocation
	parentInvocation := m.rational.smt2Stream.invocation
	m.rational.smt2Stream.invocation = ^uint64(0)
	if e := m.rational.DisplaySMT2Into(&out, &a, false); !errors.Is(e, ErrNumericPending) || m.rational.smt2Stream.active {
		t.Fatal("parent invocation wrapped", e)
	}
	m.rational.smt2Stream.invocation = parentInvocation
}
