package numeral

// Source: src/util/mpn.cpp:158-168,206-217.

import (
	"testing"
)

func TestNaturalNativeVerificationScratchReuse(t *testing.T) {
	m := naturalManager(t)
	state, stats, used := m.runtime.State(), m.runtime.Stats(), m.budget.Used()
	m.budget.FailAfter(0)
	for _, size := range []int{2, 17, 65, 129, 3} {
		a, b, q, r := make([]uint32, size), make([]uint32, size), []uint32{99}, make([]uint32, size)
		a[0], a[size-1], b[size-1] = 7, 1, 2
		_, e := m.Div(a, b, q, r)
		if e != nil || q[0] != 0 || oracle(r).Cmp(oracle(a)) != 0 {
			t.Fatal(size, e, q, r)
		}
		if len(m.verificationScratch) != 0 {
			t.Fatal("scratch retained active span")
		}
	}
	if m.runtime.State() != state || m.runtime.Stats() != stats || m.budget.Used() != used {
		t.Fatal("native verification changed Runtime accounting")
	}
	a, b, q, r := make([]uint32, 65), make([]uint32, 65), []uint32{99}, make([]uint32, 65)
	a[64], b[64] = 1, 2
	var err error
	allocations := testing.AllocsPerRun(20, func() { _, err = m.Div(a, b, q, r) })
	if err != nil || allocations != 0 {
		t.Fatal("warm verification allocations", err, allocations)
	}
	m.budget.FailAfter(-1)
}

func TestNaturalNativeVerificationAssertionCleanup(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("checked verification only")
	}
	bindMPNDebugActions(t, "T")
	m := naturalManager(t)
	view := func(n uint32) limbView { return limbView{external: []uint32{n}, size: 1} }
	// The quotient is deliberately wrong: 4*2+0 != 10.
	if e := m.sourceVerifyDivision(view(10), view(2), view(4), view(0)); !isMPNVerificationThrow(e) || len(m.verificationScratch) != 0 {
		t.Fatal("assertion cleanup", e)
	}
	if e := m.sourceVerifyDivision(view(10), view(2), view(5), view(0)); e != nil || len(m.verificationScratch) != 0 {
		t.Fatal("reuse after assertion", e)
	}
	if e := m.CloseChecked(); e != nil || m.verificationScratch != nil {
		t.Fatal("owner cleanup", e)
	}
}
