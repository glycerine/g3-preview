package numeral

import (
	"errors"
	"math/big"
	"strconv"
	"testing"
)

func TestBinaryRootInvocationEveryReservationAndEnclosures(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, lower := range []bool{false, true} {
			for _, exponent := range []uint32{17, 18, 200, 201} {
				t.Run(strconv.FormatBool(negative)+"/"+strconv.FormatBool(lower)+"/"+strconv.Itoa(int(exponent)), func(t *testing.T) {
					succeeded, refused, raw := false, false, false
					for prefix := int64(0); prefix < 4096 && !succeeded; prefix++ {
						if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
							qm, budget, _ := rawArithmeticGuardOwner(t)
							diagnostics, e := NewRationalService(serviceConfig(budget))
							serviceMust(t, e)
							t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
							m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
							serviceMust(t, e)
							a, out := NewBinaryInt(0), NewBinaryInt(0)
							n := NewInt(0)
							input := "18446744073709551619"
							if negative {
								input = "-" + input
							}
							serviceMust(t, qm.Z.SetString(&n, input))
							serviceMust(t, m.SetIntegerPower(&a, &n, exponent))
							var call BinaryRootInvocation
							budget.FailAfter(prefix)
							_, body := m.rootWithInvocation(&a, 3, &out, lower, &call)
							budget.FailAfter(-1)
							if body == nil {
								succeeded = true
							} else {
								refused = true
								if !numericPhysicalReservationRefusal(body) {
									t.Fatal("unexpected root refusal", body)
								}
							}
							if qm.Z.MatchesRootInvocation(&call.root) && call.SourceCause() == nil {
								raw = true
								before := qm.Z.RootRaw.SourceIndex
								serviceMust(t, call.Cleanup(false))
								if !call.Pending() || qm.Z.RootRaw.SourceIndex != before {
									t.Fatal("source acknowledgement consumed RAW")
								}
							} else {
								serviceMust(t, call.Cleanup(false))
							}
							serviceMust(t, call.Cleanup(true))
							if call.Pending() {
								t.Fatal("root child survived owning cleanup")
							}
							zero := NewBinaryInt(0)
							if _, e := m.IsZero(&zero); e != nil {
								t.Fatalf("binary root owner still pending: %v; original: %v", e, body)
							}
							if body == nil {
								text, e := qm.Z.String(&out.Num)
								serviceMust(t, e)
								numerator, _ := new(big.Int).SetString(text, 10)
								value := new(big.Rat).SetFrac(numerator, new(big.Int).Lsh(big.NewInt(1), uint(out.KField)))
								cube := new(big.Rat).Mul(new(big.Rat).Mul(value, value), value)
								original, _ := new(big.Int).SetString(input, 10)
								want := new(big.Rat).SetFrac(original, new(big.Int).Lsh(big.NewInt(1), uint(exponent)))
								cmp := cube.Cmp(want)
								if lower && cmp > 0 || !lower && cmp < 0 {
									t.Fatal("independent enclosure", cube, want)
								}
							}
							serviceMust(t, m.Del(&out))
							serviceMust(t, m.Del(&a))
							serviceMust(t, qm.Z.Del(&n))
							serviceMust(t, m.CloseChecked())
						}) {
							return
						}
					}
					if !succeeded || !refused || !raw {
						t.Fatal("missing success, refusal or RAW", succeeded, refused, raw)
					}
				})
			}
		}
	}
}

func TestBinaryRootInvocationRejectsLaterIntegerCall(t *testing.T) {
	found := false
	for allowance := int64(0); allowance < 256 && !found; allowance++ {
		if !t.Run(strconv.FormatInt(allowance, 10), func(t *testing.T) {
			qm, budget, _ := rawArithmeticGuardOwner(t)
			diagnostics, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
			m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
			serviceMust(t, e)
			a, out := NewBinaryInt(0), NewBinaryInt(0)
			serviceMust(t, qm.Z.SetString(&a.Num, "340282366920938463574714963092423127049"))
			var call BinaryRootInvocation
			budget.FailAfter(allowance)
			_, body := m.RootLowerWithInvocation(&a, 2, &out, &call)
			budget.FailAfter(-1)
			z := qm.Z
			if z.MatchesRootInvocation(&call.root) {
				found = true
				if body == nil || call.SourceCause() != nil || z.RootRaw.EndEntered {
					t.Fatal("RAW acquired source end", body)
				}
				copied := call
				if e := copied.Cleanup(true); !errors.Is(e, ErrBinaryOwner) {
					t.Fatal("copied root owner acquired child", e)
				}
				if _, e := m.RootLowerWithInvocation(nil, 2, nil, &call); !errors.Is(e, ErrRootPending) {
					t.Fatal("pending entry read operands", e)
				}
				serviceMust(t, call.Cleanup(false))
				serviceMust(t, m.Del(&out))
				serviceMust(t, call.Cleanup(true))
				var later IntegerRootInvocationReceipt
				budget.FailAfter(0)
				_, next := z.rootInPlaceWithReceipt(&a.Num, 2, &later)
				budget.FailAfter(-1)
				if next == nil || !z.MatchesRootInvocation(&later) {
					t.Fatal("no later actual RAW", next)
				}
				values, pending, used := z.RootRaw.Values, z.PendingCell, budget.Used()
				serviceMust(t, call.Cleanup(false))
				serviceMust(t, call.Cleanup(true))
				if !z.MatchesRootInvocation(&later) || z.RootRaw.Values != values || z.PendingCell != pending || budget.Used() != used {
					t.Fatal("old binary-root owner consumed later call")
				}
				serviceMust(t, z.CleanupRootInvocation(&later, true))
			} else {
				serviceMust(t, call.Cleanup(false))
				serviceMust(t, m.Del(&out))
				serviceMust(t, call.Cleanup(true))
			}
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.CloseChecked())
		}) {
			return
		}
	}
	if !found {
		t.Fatal("no actual integer-root child reached")
	}
}
