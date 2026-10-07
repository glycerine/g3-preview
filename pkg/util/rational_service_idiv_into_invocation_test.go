// Source: src/util/rational.h:182-184; src/util/mpq.h:523-527.
package util

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceIDivIntoInvocationAliasesAndInline(t *testing.T) {
	for _, alias := range []string{"distinct", "left", "right"} {
		for _, denominator := range []string{"8", "-8"} {
			for _, numerator := range []string{"0", "1", "-1", "17", "-17", "18446744073709551617", "-18446744073709551617"} {
				s, b := serviceNew(t)
				a := serviceValue(t, s, numerator)
				divisor := serviceValue(t, s, denominator)
				separate := serviceValue(t, s, "99/123")
				out := &separate
				if alias == "left" {
					out = &a
				}
				if alias == "right" {
					out = &divisor
				}
				var call RationalServiceInvocation
				if len(numerator) < 4 {
					b.FailAfter(0)
				}
				e := s.IDivIntoWithInvocation(&a, &divisor, out, &call)
				b.FailAfter(-1)
				serviceMust(t, e)
				parsed, _ := new(big.Int).SetString(numerator, 10)

				divisorInt, _ := new(big.Int).SetString(denominator, 10)
				want := new(big.Int).Div(parsed, divisorInt)
				// mpz.h:468-482 rereads b after publishing c. If c==b,
				// adjustment follows the quotient sign instead of the divisor.
				if alias == "right" && parsed.Sign() < 0 {
					quotient, remainder := new(big.Int), new(big.Int)
					quotient.QuoRem(parsed, divisorInt, remainder)
					if remainder.Sign() != 0 {
						if quotient.Sign() < 0 {
							quotient.Add(quotient, big.NewInt(1))
						} else {
							quotient.Sub(quotient, big.NewInt(1))
						}
					}
					want = quotient
				}
				serviceText(t, s, out, want.String())
				if !out.Raw.Den.IsSmall() || out.Raw.Den.Val != 1 {
					t.Fatal("direct integer division DEN reset")
				}
				copied := call
				if e = s.IDivIntoWithInvocation(&a, &divisor, out, &copied); !errors.Is(e, arena.ErrInvalidSlot) {
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
}

func TestRationalServiceIDivIntoInvocationEveryActualReservation(t *testing.T) {
	for _, returned := range []bool{false, true} {
		success := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			b := arena.NewBudget(1 << 27)
			cfg := serviceConfig(b)
			cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
			s, e := NewRationalService(cfg)
			serviceMust(t, e)
			a := serviceValue(t, s, "-184467440737095516170000000000000000003")
			divisor := serviceValue(t, s, "34359738369")
			var out OwnedRational
			if !returned {
				out = serviceValue(t, s, "1/2")
			}
			oldDenCell, oldDenVal := out.Raw.Den.Cell, out.Raw.Den.Val
			var call RationalServiceInvocation
			b.FailAfter(ordinal)
			var body error
			if returned {
				out, body = s.IDivWithInvocation(&a, &divisor, &call)
			} else {
				body = s.IDivIntoWithInvocation(&a, &divisor, &out, &call)
			}
			b.FailAfter(-1)
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if !returned && (out.Raw.Den.Cell != oldDenCell || out.Raw.Den.Val != oldDenVal) {
					t.Fatal("DEN reset before failed NUM", ordinal)
				}
			} else {
				parsed, _ := new(big.Int).SetString("-184467440737095516170000000000000000003", 10)
				modulus, _ := new(big.Int).SetString("34359738369", 10)
				serviceText(t, s, &out, new(big.Int).Div(parsed, modulus).String())
			}
			if call.division.Frame != &s.Manager.Z.DivisionRaw || call.division.Invocation == 0 {
				t.Fatal("missing exact negative-division child", ordinal)
			}
			tuple, cause := call.invocation, call.bodyCause
			serviceMust(t, call.Cleanup(false))
			if out.ownerID != 0 {
				serviceMust(t, s.CleanupValue(&out))
			}
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
				t.Fatal("direct integer division caller graph retained", ordinal, b.Used())
			}
			if body == nil {
				if ordinal == 0 {
					t.Fatal("missing actual output allocation frontier")
				}
				t.Log(fmt.Sprint("first completion ", returned, " ", ordinal))
				success = true
				break
			}
		}
		if !success {
			t.Fatal("integer division reservation horizon", returned)
		}
	}
}
