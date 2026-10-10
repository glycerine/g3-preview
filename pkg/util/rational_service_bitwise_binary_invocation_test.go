// Source: src/util/mpz.cpp:1018-1148; src/util/rational.h bitwise forwards.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func binaryBitwiseInvocation(s *RationalService, op int, a, b *OwnedRational, c *RationalServiceInvocation) (OwnedRational, error) {
	switch op {
	case 0:
		return s.BitwiseOrWithInvocation(a, b, c)
	case 1:
		return s.BitwiseAndWithInvocation(a, b, c)
	default:
		return s.BitwiseXorWithInvocation(a, b, c)
	}
}
func binaryBitwiseRaw(z *IntegerManager, op int) *IntegerRawScope {
	switch op {
	case 0:
		return &z.BitwiseOrRaw
	case 1:
		return &z.BitwiseAndRaw
	default:
		return &z.BitwiseXorRaw
	}
}
func binaryBitwiseReceipt(c *RationalServiceInvocation, op int) *integerRawInvocationReceipt {
	switch op {
	case 0:
		return &c.bitwiseOr
	case 1:
		return &c.bitwiseAnd
	default:
		return &c.bitwiseXor
	}
}
func TestRationalServiceBinaryBitwiseEveryActualReservation(t *testing.T) {
	for op := 0; op < 3; op++ {
		for _, tc := range []struct{ a, b string }{{"1", "3"}, {"18446744073709551617", "1461501637330902918203684832716283019655932542981"}, {"1461501637330902918203684832716283019655932542981", "18446744073709551617"}, {"0", "18446744073709551617"}} {
			complete, sawBody := false, false
			for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
				budget := arena.NewBudget(1 << 27)
				rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
				serviceMust(t, e)
				cfg := serviceConfig(budget)
				cfg.Integers.Memory = rt
				cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
				s, e := NewRationalService(cfg)
				serviceMust(t, e)
				a, b := serviceValue(t, s, tc.a), serviceValue(t, s, tc.b)
				var call RationalServiceInvocation
				budget.FailAfter(ordinal)
				out, body := binaryBitwiseInvocation(s, op, &a, &b, &call)
				budget.FailAfter(-1)
				raw, receipt := binaryBitwiseRaw(s.Manager.Z, op), binaryBitwiseReceipt(&call, op)
				if body == nil {
					complete = true
					x, _ := new(big.Int).SetString(tc.a, 10)
					y, _ := new(big.Int).SetString(tc.b, 10)
					switch op {
					case 0:
						x.Or(x, y)
					case 1:
						x.And(x, y)
					case 2:
						x.Xor(x, y)
					}
					serviceText(t, s, &out, x.String())
					serviceText(t, s, &a, tc.a)
					serviceText(t, s, &b, tc.b)
					serviceMust(t, call.SourceReady())
					if a.Raw.Num.IsSmall() && b.Raw.Num.IsSmall() {
						if receipt.Invocation != 0 {
							t.Fatal("small path invented RAW scope")
						}
					} else if receipt.Frame != raw || receipt.Invocation != raw.Invocation || raw.Count != 6 || raw.cursor != 6 || !raw.ended {
						t.Fatal("original six forward source DELs absent", op, ordinal)
					}
				} else if !numericPhysicalReservationRefusal(body) {
					t.Fatal(op, ordinal, body)
				}
				if raw.BodyCause != nil {
					sawBody = true
					if raw.cursor != 0 || raw.ended || raw.BodySucceeded || !raw.Active {
						t.Fatal("body failure ran explicit DELs", op, ordinal)
					}
				}
				copy := call
				if e = copy.Cleanup(false); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("copied context", e)
				}
				oldBody, oldSource, generation := raw.BodyCause, raw.sourceCause, raw.Invocation
				serviceMust(t, call.Cleanup(false))
				if out.ownerID != 0 {
					serviceMust(t, s.CleanupValue(&out))
				}
				serviceMust(t, call.Cleanup(true))
				if receipt.Invocation != 0 && raw.Active {
					t.Fatal("exact RAW child not joined", op, ordinal)
				}
				if raw.BodyCause != oldBody || raw.sourceCause != oldSource || raw.Invocation != generation {
					t.Fatal("cleanup rewrote source history")
				}
				serviceMust(t, call.EndContext())
				serviceMust(t, s.Del(&a))
				serviceMust(t, s.Del(&b))
				serviceMust(t, s.CloseChecked())
				serviceMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Fatal("retained graph", budget.Used())
				}
			}
			if !complete {
				t.Fatal("reservation campaign incomplete", op, tc)
			}
			if tc.a != "1" && !sawBody {
				t.Fatal("general RAW body refusal absent", op, tc)
			}
		}
	}
}

func TestRationalServiceBinaryBitwiseGenerationIsolation(t *testing.T) {
	for op := 0; op < 3; op++ {
		s, budget := servicePublicationDirectOwner(t)
		a, zero := serviceValue(t, s, "18446744073709551617"), serviceValue(t, s, "0")
		var older, young RationalServiceInvocation
		old, e := binaryBitwiseInvocation(s, op, &a, &zero, &older)
		serviceMust(t, e)
		z := s.Manager.Z
		var bank []Int
		for z.Cells.Stats().Live+z.Cells.Stats().Retired < z.Cells.Stats().Slots {
			var value Int
			serviceMust(t, z.Copy(&value, &a.Raw.Num))
			bank = append(bank, value)
		}
		budget.FailAfter(0)
		out, body := binaryBitwiseInvocation(s, op, &a, &a, &young)
		budget.FailAfter(-1)
		if !numericPhysicalReservationRefusal(body) {
			t.Fatal("young actual copy refusal", op, body)
		}
		raw := binaryBitwiseRaw(z, op)
		generation, cursor, active, cause := raw.Invocation, raw.cursor, raw.Active, raw.BodyCause
		if generation == binaryBitwiseReceipt(&older, op).Invocation || !active {
			t.Fatal("young source frame not retained")
		}
		serviceMust(t, older.Cleanup(false))
		serviceMust(t, s.CleanupValue(&old))
		serviceMust(t, older.Cleanup(true))
		serviceMust(t, older.EndContext())
		if raw.Invocation != generation || raw.cursor != cursor || raw.Active != active || raw.BodyCause != cause {
			t.Fatal("old caller adopted young generation")
		}
		serviceMust(t, young.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, young.Cleanup(true))
		serviceMust(t, young.EndContext())
		for i := range bank {
			serviceMust(t, z.Del(&bank[i]))
		}
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&zero))
	}
}

func TestRationalServiceBinaryBitwiseExecutingRawGuard(t *testing.T) {
	for op := 0; op < 3; op++ {
		budget := arena.NewBudget(1 << 27)
		var s *RationalService
		var call RationalServiceInvocation
		active, calls := false, 0
		rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: false, SizeTBits: 64, Transport: func(f umemory.Failure) error {
			if active {
				calls++
				z := s.Manager.Z
				raw, receipt := binaryBitwiseRaw(z, op), binaryBitwiseReceipt(&call, op)
				if receipt.Frame != raw || receipt.Invocation != raw.Invocation || !raw.executing || raw.Count != 6 || receipt.Kind() != []IntegerRawKind{IntegerRawBitwiseOr, IntegerRawBitwiseAnd, IntegerRawBitwiseXor}[op] {
					t.Fatal("actual executing RAW identity", op)
				}
				if e := call.Cleanup(false); e != arena.ErrInvalidSlot {
					t.Fatal("executing caller cleanup", e)
				}
				if e := call.SourceReady(); e != ErrRationalLexicalPending {
					t.Fatal("executing source readiness", e)
				}
				var e error
				switch op {
				case 0:
					e = z.CleanupBitwiseOr()
				case 1:
					e = z.CleanupBitwiseAnd()
				case 2:
					e = z.CleanupBitwiseXor()
				}
				if e != ErrNumericPending {
					t.Fatal("executing RAW cleanup", e)
				}
				if e = z.CloseChecked(); e != ErrNumericPending {
					t.Fatal("executing integer close", e)
				}
			}
			return f
		}})
		serviceMust(t, e)
		cfg := serviceConfig(budget)
		cfg.Integers.Memory = rt
		cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e = NewRationalService(cfg)
		serviceMust(t, e)
		text := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 4096), big.NewInt(7)).String()
		a, b := serviceValue(t, s, text), serviceValue(t, s, "18446744073709551617")
		serviceMust(t, rt.SetMaxCount(uint64(rt.State().GlobalCount)+1))
		active = true
		out, body := binaryBitwiseInvocation(s, op, &a, &b, &call)
		active = false
		if body == nil || calls == 0 {
			t.Fatal("actual held RAW transport absent", op, body, calls)
		}
		serviceMust(t, rt.SetMaxCount(0))
		serviceMust(t, call.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
		serviceMust(t, s.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("callback graph retained", op, budget.Used())
		}
	}
}
