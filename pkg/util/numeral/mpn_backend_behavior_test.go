package numeral

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestNaturalSourceNoAddbackDoesNotPrepareAb(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	var a [40]uint32
	var d [19]uint32
	for i := range a {
		a[i] = 0x12345678
	}
	d[18] = 0x80000000
	var q [22]uint32
	var r [19]uint32
	if _, e = n.Div(a[:], d[:], q[:], r[:]); e != nil {
		t.Fatal(e)
	}
	wq, wr := new(big.Int), new(big.Int)
	wq.QuoRem(oracle(a[:]), oracle(d[:]), wr)
	if oracle(q[:]).Cmp(wq) != 0 || oracle(r[:]).Cmp(wr) != 0 {
		t.Fatal("independent exact division")
	}
	// mpn.cpp:341–344 initializes/grows ab only AFTER a borrow and quotient
	// decrement. This fixture never borrows: ab remains unused.
	if n.Division[3] != nil {
		t.Fatal("unconditional source add-back allocation", naturalDivisionCapacities(n))
	}
}

func TestNaturalSourceDebugVerificationAfterOutputs(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG local only")
	}
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	var a, d [16]uint32
	a[15] = 1
	d[15] = 2
	var q [1]uint32
	var r [16]uint32
	q[0] = 99
	budget.FailAfter(0)
	_, e = n.Div(a[:], d[:], q[:], r[:])
	budget.FailAfter(-1)
	if e != nil || q[0] != 0 || r != a || n.RuntimeOwner().Stats().LiveBlocks != 0 {
		t.Fatal("native verification after outputs", e, q, r)
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if budget.Used() != 0 {
		t.Fatal("lexical failure cleanup")
	}
}

func TestNaturalSourceDebugLexicalTempDestroyedEachCall(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG local only")
	}
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	var a, d [16]uint32
	a[15] = 1
	d[15] = 2
	var q [1]uint32
	var r [16]uint32
	for iteration := 0; iteration < 2; iteration++ {
		if _, e = n.Div(a[:], d[:], q[:], r[:]); e != nil {
			t.Fatal(e)
		}
		if n.RuntimeOwner().Stats().LiveBlocks != 0 {
			t.Fatal("source lexical temp retained")
		}
	}
	if n.RuntimeOwner().State().GlobalCount != 0 {
		t.Fatal("actual repeated lexical allocation count", n.RuntimeOwner().State())
	}
}

func TestNaturalSourceShortNumeratorEarlyReturn(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	a := [1]uint32{7}
	d := [2]uint32{}
	r := [2]uint32{99, 99}
	budget.FailAfter(0)
	// Source first branch precedes even all-zero denominator validation. The
	// unsigned quotient length is zero when the denominator is one limb longer.
	result, e := n.divide(limbView{external: a[:], size: 1}, limbView{external: d[:], size: 2}, limbView{}, limbView{external: r[:], size: 2})
	budget.FailAfter(-1)
	if e != nil || result || r != [2]uint32{7, 0} || n.RuntimeOwner().Stats().LiveBlocks != 0 {
		t.Fatal("early source return", result, e, r)
	}
}

func TestNaturalPublicShortNumeratorSourceWrites(t *testing.T) {
	n := naturalManager(t)
	state, stats := n.RuntimeOwner().State(), n.RuntimeOwner().Stats()
	n.budget.FailAfter(0)
	defer n.budget.FailAfter(-1)
	r := []uint32{99, 99}
	branch, err := n.Div([]uint32{7}, []uint32{0, 0}, nil, r)
	if err != nil || branch || r[0] != 7 || r[1] != 0 {
		t.Fatal("public short-numerator branch precedes denominator validation", branch, err, r)
	}
	q := []uint32{99}
	one := []uint32{88}
	branch, err = n.Div(nil, []uint32{0}, q, one)
	if err != nil || branch || q[0] != 99 || one[0] != 0 {
		t.Fatal("empty numerator leaves zero-length quotient untouched", branch, err, q, one)
	}
	r = []uint32{88, 88, 88}
	branch, err = n.Div([]uint32{7}, []uint32{0, 0, 0}, q, r)
	if !errors.Is(err, arena.ErrBounds) || branch || q[0] != 0 || r[0] != 88 || r[1] != 88 || r[2] != 88 {
		t.Fatal("wrapped quotient length stops at first out-of-bounds write", branch, err, q, r)
	}
	if n.RuntimeOwner().State() != state || n.RuntimeOwner().Stats() != stats {
		t.Fatal("short source branch acquired Runtime storage")
	}
}

func TestNaturalSourceAllZeroDenominatorStopsBeforeOutputs(t *testing.T) {
	if modularSourceDebug {
		t.Skip("debug UNREACHABLE action is checked with an explicit process binding")
	}
	n := naturalManager(t)
	_, err := n.Div([]uint32{7}, nil, nil, nil)
	if !errors.Is(err, diagnostic.ErrDebugNativeUnreachable) {
		t.Fatal("empty denominator source guard", err)
	}
	for _, denom := range [][]uint32{{0}, {0, 0}} {
		a := make([]uint32, len(denom))
		a[0] = 7
		q := []uint32{99}
		r := make([]uint32, len(denom))
		for i := range r {
			r[i] = 88
		}
		_, err := n.Div(a, denom, q, r)
		if !errors.Is(err, diagnostic.ErrDebugNativeUnreachable) || q[0] != 99 {
			t.Fatal("all-zero source guard", denom, err, q)
		}
		for _, v := range r {
			if v != 88 {
				t.Fatal("remainder changed before source guard", denom, r)
			}
		}
	}
}

func TestNaturalSourceUnnormalizedDenominatorShiftGuard(t *testing.T) {
	if modularSourceDebug {
		t.Skip("debug assertion action is checked with an explicit process binding")
	}
	n := naturalManager(t)
	q, r := []uint32{99}, []uint32{88, 88}
	_, err := n.Div([]uint32{7, 1}, []uint32{1, 0}, q, r)
	if !errors.Is(err, arena.ErrBounds) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("guard first undefined source shift", err, q, r)
	}
}

func TestNaturalSourceUnsignedQuotientGuardAtWrite(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	d := [2]uint32{}
	q := [1]uint32{99}
	r := [2]uint32{99, 99}
	// Source unsigned lnum-lden+1 wraps here. First quotient write is defined;
	// only the next actual out-of-bounds access is guarded by the Go adapter.
	_, e = n.divide(limbView{}, limbView{external: d[:], size: 2}, limbView{external: q[:], size: 1}, limbView{external: r[:], size: 2})
	if e != arena.ErrBounds || q[0] != 0 || r != [2]uint32{99, 99} {
		t.Fatal("source write prefix", e, q, r)
	}
}

func TestNaturalNativeScratchExcludedFromRuntimeStats(t *testing.T) {
	s, _ := serviceNew(t)
	before, e := s.Stats()
	if e != nil {
		t.Fatal(e)
	}
	n := s.Manager.Z.Natural
	var a [40]uint32
	var d [19]uint32
	for i := range a {
		a[i] = 0x12345678
	}
	d[18] = 0x80000000
	var q [22]uint32
	var r [19]uint32
	if _, e = n.Div(a[:], d[:], q[:], r[:]); e != nil {
		t.Fatal(e)
	}
	after, e := s.Stats()
	if e != nil {
		t.Fatal(e)
	}
	if after.NaturalPayloads != 0 || after.LivePayloads != before.LivePayloads {
		t.Fatal("native scratch changed Runtime payload statistics", before, after)
	}
	if n.Division[3] != nil {
		t.Fatal("unused add-back workspace allocated")
	}
}

func TestNaturalNativeAddbackUnderRuntimeRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	// Scale the fixed three-limb subtraction/add-back source fixture by B^12.
	// The conditional add-back requires seventeen limbs of native workspace.
	var a, d [15]uint32
	a[12] = 1
	d[12] = 2
	a[14] = 0x80000000
	d[14] = 0x80000000
	q := [1]uint32{55}
	var r [15]uint32
	for i := range r {
		r[i] = 99
	}
	baseline := budget.Used()
	budget.FailAfter(0)
	_, e = n.Div(a[:], d[:], q[:], r[:])
	budget.FailAfter(-1)
	if e != nil || q[0] != 0 || n.Division[3] == nil || n.RuntimeOwner().Stats().LiveBlocks != 0 || budget.Used() != baseline {
		t.Fatal("native conditional add-back", e, q)
	}
	if oracle(r[:]).Cmp(oracle(a[:])) != 0 {
		t.Fatal("add-back remainder", r)
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if budget.Used() != 0 {
		t.Fatal("source refusal cleanup")
	}
}

func TestNaturalSourceInlineDivisionWarmBudgetAndZeroAlloc(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	a := [4]uint32{0x12345678, 0x87654321, 0xffffffff, 3}
	d := [2]uint32{7, 0x80000000}
	var q [3]uint32
	var r [2]uint32
	if _, e = n.Div(a[:], d[:], q[:], r[:]); e != nil {
		t.Fatal(e)
	}
	wq, wr := new(big.Int), new(big.Int)
	wq.QuoRem(oracle(a[:]), oracle(d[:]), wr)
	if oracle(q[:]).Cmp(wq) != 0 || oracle(r[:]).Cmp(wr) != 0 {
		t.Fatal("exact warm setup")
	}
	charge := budget.Used()
	physical := n.RuntimeOwner().Stats()
	state := n.RuntimeOwner().State()
	budget.FailAfter(0)
	var failure error
	allocations := testing.AllocsPerRun(100, func() { _, failure = n.Div(a[:], d[:], q[:], r[:]) })
	budget.FailAfter(-1)
	if allocations != 0 || failure != nil || budget.Used() != charge || n.RuntimeOwner().Stats() != physical || n.RuntimeOwner().State() != state {
		t.Fatal("inline source division changed storage", allocations, failure)
	}
	if oracle(q[:]).Cmp(wq) != 0 || oracle(r[:]).Cmp(wr) != 0 {
		t.Fatal("warm result")
	}
}
