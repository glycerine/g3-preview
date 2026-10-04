// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math/big"
	"testing"
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
	// decrement. This fixture never borrows: ab remains its source inline16.
	if n.divCapacity[3] != 16 {
		t.Fatal("unconditional source add-back allocation", n.divCapacity)
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
	failure, ok := e.(memory.Failure)
	if !ok || failure.ErrorCode() != 101 {
		t.Fatalf("debug lexical temp should refuse after output: %T %v", e, e)
	}
	if q[0] != 0 || r != a {
		t.Fatal("completed source q/rem prefix lost", q, r)
	}
	if !n.verificationActive || n.RuntimeOwner().Stats().LiveBlocks != 0 {
		t.Fatal("lexical failure state", n.RuntimeOwner().Stats())
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
		if n.RuntimeOwner().Stats().LiveBlocks != 0 || n.verificationActive {
			t.Fatal("source lexical temp retained")
		}
	}
	if n.RuntimeOwner().State().GlobalCount != 2 {
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
	if e != nil || result || r != [2]uint32{7, 0} || n.verificationActive || n.RuntimeOwner().Stats().LiveBlocks != 0 {
		t.Fatal("early source return", result, e, r)
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

func TestNaturalSourceStatsCountsActualOverflowOnly(t *testing.T) {
	s, _ := serviceNew(t)
	before, e := s.Stats()
	if e != nil {
		t.Fatal(e)
	}
	n := s.manager.z.natural
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
	if after.NaturalPayloads != before.NaturalPayloads+3 || after.LivePayloads != before.LivePayloads+3 {
		t.Fatal("actual u/v/ms overflow roots absent from Stats", before, after)
	}
	if n.division[3].raw != (memory.Block{}) || s.manager.z.words.Stats().Live != 0 {
		t.Fatal("inline or obsolete Words counted as actual overflow")
	}
}

func TestNaturalSourceAddbackRefusalAfterQuotientDecrement(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	// Scale the fixed three-limb subtraction/add-back source fixture by B^12.
	// u(16), v(15), ms(16) remain inline; only conditional ab(17) allocates.
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
	failure, ok := e.(memory.Failure)
	if !ok || failure.ErrorCode() != 101 || q[0] != 0 {
		t.Fatal("quotient estimate then decrement must precede real ab allocation", e, q)
	}
	for _, v := range r {
		if v != 99 {
			t.Fatal("remainder published before failed add-back", r)
		}
	}
	for i := uint32(0); i < 16; i++ {
		want := uint32(0)
		if i >= 12 {
			want = ^uint32(0)
		}
		if got := n.sourceDivRead(0, i); got != want {
			t.Fatal("actual modular subtraction prefix", i, got, want)
		}
	}
	if n.divCapacity != [4]uint32{16, 16, 16, 16} || n.division[3].size() != 16 || n.RuntimeOwner().Stats().LiveBlocks != 0 || budget.Used() != baseline {
		t.Fatal("failed source ab first expansion roots", n.divCapacity, n.RuntimeOwner().Stats())
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
