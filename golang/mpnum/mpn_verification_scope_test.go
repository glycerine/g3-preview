// Licensed under the MIT license.
// Source: src/util/mpn.cpp:158-168 and 206-217.
package mpnum

import (
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func TestNaturalRetainedVerificationGuardsOnlyReachedLocal(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	if !modularSourceDebug {
		// Release has no lexical verification constructor to retain.
		q, r := []uint32{99}, []uint32{99}
		if _, e = n.Div([]uint32{10}, []uint32{2}, q, r); e != nil || q[0] != 5 || r[0] != 0 || n.verificationActive {
			t.Fatal("release division", e, q, r)
		}
		return
	}
	var a, d [16]uint32
	a[15], d[15] = 1, 2
	q, r := [1]uint32{99}, [16]uint32{}
	budget.FailAfter(0)
	_, e = n.Div(a[:], d[:], q[:], r[:])
	budget.FailAfter(-1)
	if failure, ok := e.(memory.Failure); !ok || failure.ErrorCode() != 101 || !n.verificationActive || q[0] != 0 || r != a {
		t.Fatal("actual failed verification constructor after outputs", e, q, r)
	}
	retained := n.verification
	state, stats := n.runtime.State(), n.runtime.Stats()
	// This source return precedes all denominator reads and the debug local.
	short, zero, rem := [1]uint32{7}, [2]uint32{}, [2]uint32{99, 99}
	result, e := n.divide(limbView{external: short[:], size: 1}, limbView{external: zero[:], size: 2}, limbView{}, limbView{external: rem[:], size: 2})
	if e != nil || result || rem != [2]uint32{7, 0} {
		t.Fatal("unreached retained local must not block early return", result, e, rem)
	}
	if n.verification != retained || !n.verificationActive || n.runtime.State() != state || n.runtime.Stats() != stats {
		t.Fatal("early return changed retained owner")
	}
	// A later ordinary division publishes its outputs before reaching the
	// occupied Go holder for the source lexical constructor.
	q1, r1 := [1]uint32{99}, [1]uint32{99}
	_, e = n.Div([]uint32{10}, []uint32{2}, q1[:], r1[:])
	if e != arena.ErrInvalidSlot || q1[0] != 5 || r1[0] != 0 || n.verification != retained || !n.verificationActive {
		t.Fatal("reached-local guard must preserve output prefix and original root", e, q1, r1)
	}
	if e = n.CloseChecked(); e != nil || budget.Used() != 0 {
		t.Fatal("owning cleanup", e, budget.Used())
	}
}

func TestNaturalEarlyReturnRetainsPhysicalVerificationRoot(t *testing.T) {
	if !modularSourceDebug {
		return // The source local exists only under Z3DEBUG.
	}
	for fault := int64(0); fault < 50; fault++ {
		budget := arena.NewBudget(1 << 24)
		n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
		if e != nil {
			t.Fatal(e)
		}
		var a, d [65]uint32
		a[64], d[64] = 1, 2
		q, r := [1]uint32{99}, [65]uint32{}
		budget.FailAfter(fault)
		_, e = n.Div(a[:], d[:], q[:], r[:])
		retained := e != nil && n.verificationActive && n.verification.raw != (memory.Block{})
		if retained {
			if q[0] != 0 || r != a {
				t.Fatal("failed constructor lost published outputs")
			}
			root := n.verification
			state, stats, used := n.runtime.State(), n.runtime.Stats(), budget.Used()
			if stats.LiveBlocks != 1 {
				t.Fatal("actual failed-constructor backing", stats)
			}
			short, zero, rem := [1]uint32{7}, [2]uint32{}, [2]uint32{99, 99}
			result, err := n.divide(limbView{external: short[:], size: 1}, limbView{external: zero[:], size: 2}, limbView{}, limbView{external: rem[:], size: 2})
			if err != nil || result || rem != [2]uint32{7, 0} || n.verification != root || !n.verificationActive || n.runtime.State() != state || n.runtime.Stats() != stats || budget.Used() != used {
				t.Fatal("early return under armed refusal must retain physical root", result, err, rem)
			}
			if _, err = n.runtime.ReadUint32(root.raw, 0); err != nil {
				t.Fatal("retained initialized backing", err)
			}
		}
		budget.FailAfter(-1)
		if e = n.CloseChecked(); e != nil || budget.Used() != 0 {
			t.Fatal("failed-constructor owning cleanup", e, budget.Used())
		}
		if retained {
			return
		}
	}
	t.Fatal("no actual failed-constructor raw root observed")
}
