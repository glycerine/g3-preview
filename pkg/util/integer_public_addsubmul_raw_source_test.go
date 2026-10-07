// Source: src/util/mpz.h:393-426. Literal RAW tmp, Mul then Add/Sub, end-only DEL.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerPublicAddSubMulRetainsReachedRAWTemporary(t *testing.T) {
	for _, subtract := range []bool{false, true} {
		name := "AddMul"
		if subtract {
			name = "SubMul"
		}
		t.Run(name, func(t *testing.T) {
			retained, success := false, false
			for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
				budget := arena.NewBudget(1 << 24)
				rt, e := NewRuntime(RuntimeConfig{Budget: budget})
				serviceMust(t, e)
				cfg := serviceConfig(budget).Integers
				cfg.Memory = rt
				cfg.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
				m, e := NewIntegerManager(cfg)
				serviceMust(t, e)
				a, b, c, d := NewInt(7), NewInt(0), NewInt(3), NewInt(17)
				digits := make([]uint32, 9)
				digits[8] = 1
				serviceMust(t, m.SetLimbs(&b, digits))
				budget.FailAfter(ordinal)
				if subtract {
					e = m.SubMul(&a, &b, &c, &d)
				} else {
					e = m.AddMul(&a, &b, &c, &d)
				}
				s := &m.AddMulInvocation
				if e != nil && s.Active && s.Stage == 2 {
					retained = true
					if s.Tmp.IsSmall() {
						t.Fatal("completed product lost actual big temporary", ordinal, e)
					}
					if m.Cell(s.Tmp.Cell).Size != 9 || m.Digit(s.Tmp.Cell, 8) != 3 {
						t.Fatal("independent power-of-two product", ordinal)
					}
					for i := uint32(0); i < 8; i++ {
						if m.Digit(s.Tmp.Cell, i) != 0 {
							t.Fatal("product lower limb", ordinal, i)
						}
					}
					// Source publish_tmp sets sign and the descriptor before all backing
					// publication succeeds; keep that destructive partial output.
					sign := int32(1)
					if subtract {
						sign = -1
					}
					if d != NewInt(17) && (d.Val != sign || d.Cell == 0) {
						t.Fatal("source partial output publication", ordinal, d)
					}
				}
				integerPhysicalAudit(t, m, &a, &b, &c, &d)
				if budget.Used() != integerActualOwnedBytes(m) {
					t.Fatal("immediate actual cold graph", ordinal)
				}
				if s.Active {
					r := integerAddMulInvocationReceipt{m, s, s.invocation}
					serviceMust(t, m.CleanupAddMulInvocation(&r, false))
					serviceMust(t, m.CleanupAddMulRaw())
				}
				budget.FailAfter(-1)
				if e == nil {
					success = true
				}
				for _, v := range []*Int{&a, &b, &c, &d} {
					serviceMust(t, m.Del(v))
				}
				serviceMust(t, m.CloseChecked())
				serviceMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Fatal("owner refund", ordinal, budget.Used())
				}
			}
			if !success || !retained {
				t.Fatal("literal Mul product never retained across later output refusal", success, retained)
			}
		})
	}
}

func TestIntegerPublicAddSubMulEndDELAfterOutput(t *testing.T) {
	for _, subtract := range []bool{false, true} {
		name := "AddMul"
		if subtract {
			name = "SubMul"
		}
		t.Run(name, func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			m, e := NewIntegerManager(cfg)
			serviceMust(t, e)
			var a, c, warm Int
			digits := make([]uint32, 64)
			digits[0] = 17
			digits[63] = 1
			serviceMust(t, m.SetLimbs(&c, digits))
			product := make([]uint32, 64)
			product[0] = 34
			product[63] = 2
			serviceMust(t, m.SetLimbs(&a, product))
			if !subtract {
				serviceMust(t, m.Neg(&a))
			}
			prime := NewInt(251)
			serviceMust(t, m.Rem(&c, &prime, &warm))
			d, bval := NewInt(9), NewInt(2)
			var receipt integerAddMulInvocationReceipt
			defer func() {
				recovery, e := rt.Allocate(400000)
				serviceMust(t, e)
				contextTLSFinish(t, rt, m.CleanupAddMulRaw)
				for _, n := range []*Int{&a, &c, &warm, &d} {
					contextTLSFinish(t, rt, func() error { return m.Del(n) })
				}
				contextTLSFinish(t, rt, m.CloseChecked)
				e = rt.Deallocate(&recovery)
				if e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (Block{})) {
					t.Fatal("recovery physical consumption", e)
				}
				contextTLSFinish(t, rt, rt.Close)
				if b.Used() != 0 {
					t.Fatal("named graph remains", b.Used())
				}
			}()
			external, e := rt.Allocate(200000)
			serviceMust(t, e)
			contextTLSNearMinimum(t, rt)
			e = rt.Deallocate(&external)
			if e == nil || external != (Block{}) {
				t.Fatal("actual external signed-domain stop", e)
			}
			if subtract {
				e = m.SubMul(&a, &bval, &c, &d)
			} else {
				e = m.AddMul(&a, &bval, &c, &d)
			}
			receipt = m.AddMulInvocation.Receipt
			if e == nil || receipt.Frame == nil || receipt.Frame.Stage != 3 || d != NewInt(0) {
				t.Fatal("caller output must precede reached end DEL", e, d, receipt)
			}
			cause := m.AddMulInvocationSourceCause(&receipt)
			if cause == nil || receipt.Frame.BodyCause != nil || receipt.Frame.ended || receipt.Frame.owned {
				t.Fatal("missing first strict source stop", cause)
			}
			integerPhysicalAudit(t, m, &a, &c, &warm, &d, &receipt.Frame.Tmp)
			used, state, stats := b.Used(), rt.State(), rt.Stats()
			if again := m.finishAddMulSource(receipt.Frame); again != cause || used != b.Used() || state != rt.State() || stats != rt.Stats() {
				t.Fatal("source retry performed another release", again)
			}
			serviceMust(t, m.CleanupAddMulInvocation(&receipt, false))
			if receipt.Frame.ended || !receipt.Frame.owned || m.AddMulInvocationSourceCause(&receipt) != cause || d != NewInt(0) {
				t.Fatal("own acknowledgment erased source stop")
			}
			serviceMust(t, m.Del(&d))
			serviceMust(t, m.CleanupAddMulRaw())
			if m.AddMulInvocationSourceCause(&receipt) != cause || receipt.Frame.ended || !receipt.Frame.owned {
				t.Fatal("abort erased historical source stop")
			}
			integerPhysicalAudit(t, m, &a, &c, &warm, &d)
			old := receipt
			nextA, nextB, nextC := NewInt(5), NewInt(2), NewInt(3)
			if subtract {
				serviceMust(t, m.SubMul(&nextA, &nextB, &nextC, &d))
			} else {
				serviceMust(t, m.AddMul(&nextA, &nextB, &nextC, &d))
			}
			expected := int32(11)
			if subtract {
				expected = -1
			}
			if d != NewInt(expected) || !errors.Is(e, arena.ErrBounds) || m.AddMulInvocationSourceCause(&old) != nil {
				t.Fatal("new invocation and immutable old end failure", d, e)
			}
			serviceMust(t, m.CleanupAddMulInvocation(&old, true))
		})
	}
}

// A real cold Mul replacement destroys old source scratch before allocation.
// Explicit owning acknowledgment consumes retained RAW acquisitions; it does
// not invent a replacement for a source-invalid shared scratch pointer.
func TestIntegerPublicAddSubMulOwnAckDoesNotRepairSourceScratchHole(t *testing.T) {
	for _, subtract := range []bool{false, true} {
		name := "AddMul"
		if subtract {
			name = "SubMul"
		}
		t.Run(name, func(t *testing.T) {
			budget := arena.NewBudget(1 << 24)
			rt, e := NewRuntime(RuntimeConfig{Budget: budget})
			serviceMust(t, e)
			cfg := serviceConfig(budget).Integers
			cfg.Memory = rt
			cfg.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
			m, e := NewIntegerManager(cfg)
			serviceMust(t, e)
			var c, d Int
			digits := make([]uint32, 64)
			digits[0] = 3
			digits[63] = 1
			serviceMust(t, m.SetLimbs(&c, digits))
			a, b := NewInt(7), NewInt(2)
			budget.FailAfter(0)
			if subtract {
				e = m.SubMul(&a, &b, &c, &d)
			} else {
				e = m.AddMul(&a, &b, &c, &d)
			}
			if !numericPhysicalReservationRefusal(e) || m.Tmp[0] != 0 {
				t.Fatal("actual old scratch deleted before refused replacement", e, m.Tmp[0])
			}
			integerPhysicalAudit(t, m, &a, &b, &c, &d)
			if budget.Used() != integerActualOwnedBytes(m) {
				t.Fatal("before-lift exact scratch failure graph")
			}
			serviceMust(t, m.CleanupAddMulRaw())
			budget.FailAfter(-1)
			if subtract {
				e = m.SubMul(&a, &b, &c, &d)
			} else {
				e = m.AddMul(&a, &b, &c, &d)
			}
			if !errors.Is(e, arena.ErrInvalidSlot) || m.Tmp[0] != 0 {
				t.Fatal("owning acknowledgment repaired undefined source scratch", e, m.Tmp[0])
			}
			serviceMust(t, m.CleanupAddMulRaw())
			for _, v := range []*Int{&a, &b, &c, &d} {
				serviceMust(t, m.Del(v))
			}
			serviceMust(t, m.CloseChecked())
			serviceMust(t, rt.Close())
			if budget.Used() != 0 {
				t.Fatal("owning suffix refund", budget.Used())
			}
		})
	}
}
