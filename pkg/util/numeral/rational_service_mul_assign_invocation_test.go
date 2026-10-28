package numeral

// Source: src/util/rational.h:161-164; src/util/mpq.h:324-332;
// src/ast/rewriter/poly_rewriter_def.h:217,248.

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceMulAssignInvocationAliasingAndSourceBranches(t *testing.T) {
	s, budget := serviceNew(t)
	for _, row := range []struct {
		a, b, want string
		alias      bool
	}{
		{"7", "-3", "-21", false}, {"7", "7", "49", true},
		{"2/3", "9/4", "3/2", false}, {"-2/3", "-2/3", "4/9", true},
		{"0", "5/7", "0", false},
	} {
		a := serviceValue(t, s, row.a)
		b := serviceValue(t, s, row.b)
		rhs := &b
		if row.alias {
			rhs = &a
		}
		var c RationalServiceInvocation
		before := budget.Used()
		if row.a == "7" {
			budget.FailAfter(0)
		}
		e := s.MulAssignWithInvocation(&a, rhs, &c)
		budget.FailAfter(-1)
		serviceMust(t, e)
		if row.a == "7" && budget.Used() > before {
			t.Fatal("inline multiplication allocated")
		}
		got, e := s.String(&a)
		serviceMust(t, e)
		if got != row.want {
			t.Fatal(row, got)
		}
		other, e := s.String(&b)
		serviceMust(t, e)
		if other != row.b {
			t.Fatal("right operand changed", other)
		}
		serviceMust(t, c.SourceReady())
		serviceMust(t, c.Cleanup(false))
		serviceMust(t, c.Cleanup(true))
		serviceMust(t, c.EndContext())
		serviceMust(t, s.Del(&a))
		serviceMust(t, s.Del(&b))
	}
}

func TestRationalServiceMulAssignInvocationEveryActualReservation(t *testing.T) {
	for _, alias := range []bool{false, true} {
		t.Run(fmt.Sprint(alias), func(t *testing.T) {
			complete, failures := false, 0
			for ordinal := int64(0); ordinal < 128 && !complete; ordinal++ {
				t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
					budget := arena.NewBudget(1 << 27)
					rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
					serviceMust(t, e)
					cfg := serviceConfig(budget)
					cfg.Integers.Memory = rt
					cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
					s, e := NewRationalService(cfg)
					serviceMust(t, e)
					n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 2048), big.NewInt(17))
					d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 1024), big.NewInt(3))
					a := serviceValue(t, s, n.String()+"/"+d.String())
					b := serviceValue(t, s, "98765432109876543210987654321/34359738369")
					before, e := s.String(&a)
					serviceMust(t, e)
					right, e := s.String(&b)
					serviceMust(t, e)
					beforeB := right
					rhs := &b
					if alias {
						rhs = &a
						right = before
					}
					leftRat, _ := new(big.Rat).SetString(before)
					rightRat, _ := new(big.Rat).SetString(right)
					product := new(big.Rat).Mul(leftRat, rightRat)
					var call RationalServiceInvocation
					budget.FailAfter(ordinal)
					body := s.MulAssignWithInvocation(&a, rhs, &call)
					budget.FailAfter(-1)
					roots := []*Int{&a.Raw.Num, &a.Raw.Den, &b.Raw.Num, &b.Raw.Den}
					for i := uint8(0); i < call.scope.Count; i++ {
						roots = append(roots, &call.scope.Roots[i].Num, &call.scope.Roots[i].Den)
					}
					var extra []umemory.Block
					if block := call.transferred.Block(); block != (umemory.Block{}) {
						extra = append(extra, block)
					}
					integerPhysicalAuditWithRaw(t, s.Manager.Z, extra, roots...)
					if want := s.HeaderBytes + s.Manager.HeaderBytes + integerActualOwnedBytes(s.Manager.Z); budget.Used() != want {
						t.Fatal("actual multiplication graph", budget.Used(), want)
					}
					if body != nil {
						failures++
						if !numericPhysicalReservationRefusal(body) {
							t.Fatal("unexpected refusal", ordinal, body)
						}
						rawNum := new(big.Int).Mul(leftRat.Num(), rightRat.Num()).String()
						rawDen := new(big.Int).Mul(leftRat.Denom(), rightRat.Denom()).String()
						num := powerPublicationComponent(t, s.Manager.Z, &a.Raw.Num, leftRat.Num().String(), rawNum, product.Num().String())
						powerPublicationComponent(t, s.Manager.Z, &a.Raw.Den, leftRat.Denom().String(), rawDen, product.Denom().String())
						if num == "failed" || num == leftRat.Num().String() {
							if roundPublicationDecode(t, s.Manager.Z, &a.Raw.Den).String() != leftRat.Denom().String() {
								t.Fatal("denominator preceded numerator")
							}
						}
						if e = s.MulAssignWithInvocation(&a, rhs, &call); !errors.Is(e, arena.ErrInvalidSlot) {
							t.Fatal("stopped call reused", e)
						}
					} else {
						complete = true
						serviceText(t, s, &a, product.RatString())
						serviceMust(t, call.SourceReady())
					}
					serviceText(t, s, &b, beforeB)
					copied := call
					if e = s.MulAssignWithInvocation(&a, rhs, &copied); !errors.Is(e, arena.ErrInvalidSlot) {
						t.Fatal("copied invocation accepted", e)
					}
					tuple, bodyHistory, sourceHistory := call.invocation, call.bodyCause, call.sourceCause
					serviceMust(t, call.Cleanup(false))
					serviceMust(t, s.CleanupValue(&a))
					serviceMust(t, call.Cleanup(true))
					if call.invocation != tuple || call.bodyCause != bodyHistory || call.sourceCause != sourceHistory {
						t.Fatal("cleanup changed history")
					}
					serviceMust(t, call.EndContext())
					serviceMust(t, s.Del(&b))
					serviceMust(t, s.CloseChecked())
					serviceMust(t, rt.Close())
					if budget.Used() != 0 {
						t.Fatal("actual owner graph retained", ordinal, budget.Used())
					}
				})
			}
			if !complete || failures == 0 {
				t.Fatal("reservation horizon", complete, failures)
			}
		})
	}
}
