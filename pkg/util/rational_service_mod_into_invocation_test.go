// Source: src/util/rational.h:198-200; src/util/mpq.h:549-553.
package util

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceModIntoInvocationAliasesAndInline(t *testing.T) {
	for _, alias := range []string{"distinct", "left", "right"} {
		for _, numerator := range []string{"17", "-17", "18446744073709551617", "-18446744073709551617"} {
			s, b := serviceNew(t)
			a := serviceValue(t, s, numerator)
			divisor := serviceValue(t, s, "8")
			separate := serviceValue(t, s, "99/123")
			out := &separate
			if alias == "left" {
				out = &a
			}
			if alias == "right" {
				out = &divisor
			}
			var call RationalServiceInvocation
			if numerator == "17" || numerator == "-17" {
				b.FailAfter(0)
			}
			e := s.ModIntoWithInvocation(&a, &divisor, out, &call)
			b.FailAfter(-1)
			serviceMust(t, e)
			parsed, _ := new(big.Int).SetString(numerator, 10)
			want := new(big.Int).Mod(parsed, big.NewInt(8)).String()
			// Source mpz.h:487-497 rereads b after rem publishes c. With
			// c==b and a negative remainder, sub(c,b,c) produces zero.
			if alias == "right" && parsed.Sign() < 0 {
				want = "0"
			}
			serviceText(t, s, out, want)
			if !out.Raw.Den.IsSmall() || out.Raw.Den.Val != 1 {
				t.Fatal("direct modulo DEN reset")
			}
			copied := call
			if e = s.ModIntoWithInvocation(&a, &divisor, out, &copied); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("copied invocation accepted", e)
			}
			serviceMust(t, call.SourceReady())
			serviceMust(t, call.Cleanup(false))
			serviceMust(t, call.Cleanup(true))
			serviceMust(t, call.EndContext())
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.Del(&divisor))
			serviceMust(t, s.Del(&separate))
		}
	}
}

func TestRationalServiceModIntoInvocationEveryActualReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 27)
		cfg := serviceConfig(b)
		cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		serviceMust(t, e)
		a := serviceValue(t, s, "184467440737095516170000000000000000003")
		divisor := serviceValue(t, s, "34359738369")
		out := serviceValue(t, s, "1/2")
		oldDenCell, oldDenVal := out.Raw.Den.Cell, out.Raw.Den.Val
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		body := s.ModIntoWithInvocation(&a, &divisor, &out, &call)
		b.FailAfter(-1)
		if body != nil {
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected refusal", ordinal, body)
			}
			if out.Raw.Den.Cell != oldDenCell || out.Raw.Den.Val != oldDenVal {
				t.Fatal("DEN reset before failed NUM", ordinal)
			}
		} else {
			parsed, _ := new(big.Int).SetString("184467440737095516170000000000000000003", 10)
			modulus, _ := new(big.Int).SetString("34359738369", 10)
			serviceText(t, s, &out, new(big.Int).Mod(parsed, modulus).String())
		}
		tuple, cause := call.invocation, call.bodyCause
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, s.CleanupValue(&out))
		serviceMust(t, call.Cleanup(true))
		if call.invocation != tuple || call.bodyCause != cause {
			t.Fatal("cleanup changed invocation history")
		}
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&divisor))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, s.Manager.Z.Memory.Close())
		if b.Used() != 0 {
			t.Fatal("direct modulo caller graph retained", ordinal, b.Used())
		}
		if body == nil {
			if ordinal == 0 {
				t.Fatal("missing actual output allocation frontier")
			}
			t.Log(fmt.Sprint("first completion ", ordinal))
			return
		}
	}
	t.Fatal("direct modulo reservation horizon")
}
