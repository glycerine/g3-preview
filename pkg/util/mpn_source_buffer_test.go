package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestNaturalSourceInline16AndActualOverflow(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var s SourceLimbBuffer
	initial := r.State()
	for i := uint32(0); i < 16; i++ {
		if e = s.push(r, 100+i); e != nil {
			t.Fatal(e)
		}
	}
	if s.capacity() != 16 || r.State() != initial || r.Stats().LiveBlocks != 0 {
		t.Fatal("inline source buffer allocated runtime storage")
	}
	if e = s.push(r, 116); e != nil {
		t.Fatal(e)
	}
	if s.capacity() != 32 || s.size() != 17 || r.Stats().LiveBlocks != 1 || r.Stats().PhysicalBytes != 136 {
		t.Fatalf("first expansion topology %+v", r.Stats())
	}
	for i := uint32(0); i < 17; i++ {
		v, e := s.read(r, i)
		if e != nil || v != 100+i {
			t.Fatalf("prefix%d %d %v", i, v, e)
		}
	}
	if e = s.Finalize(r); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("overflow leaked")
	}
}

func TestNaturalSourceBufferFailedExpansionRetainsOld(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var s SourceLimbBuffer
	for i := uint32(0); i < 32; i++ {
		if e = s.push(r, i+1); e != nil {
			t.Fatal(e)
		}
	}
	old := r.Stats()
	b.FailAfter(0)
	if e = s.push(r, 33); !errors.Is(e, Failure{Kind: FailureMemout}) {
		t.Fatalf("actual overflow refusal %v", e)
	}
	count, ce := r.AllocationCount()
	extent, se := r.AllocationSize()
	if ce != nil || se != nil || count != 2 || extent != 400 {
		t.Fatalf("failed real malloc lost nonTLS source prefix count%d extent%d: %v %v", count, extent, ce, se)
	}
	if s.size() != 32 || s.capacity() != 32 || r.Stats() != old {
		t.Fatal("failed allocation discarded source old buffer")
	}
	for i := uint32(0); i < 32; i++ {
		v, e := s.read(r, i)
		if e != nil || v != i+1 {
			t.Fatal("old prefix lost")
		}
	}
	b.FailAfter(-1)
	s.reset()
	if s.size() != 0 || s.capacity() != 32 || r.Stats() != old {
		t.Fatal("source reset freed retained overflow")
	}
	if e = s.Finalize(r); e != nil {
		t.Fatal(e)
	}
	if s.capacity() != 16 || s.size() != 0 || r.Stats().LiveBlocks != 0 {
		t.Fatal("source finalize did not restore inline storage")
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("cold roots leaked")
	}
}

func TestNaturalSourceLateTLSRefusalRetainsBothActualRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = r.SetMaxSize(100000); e != nil {
		t.Fatal(e)
	}
	var s SourceLimbBuffer
	var failure error
	for i := uint32(0); i < 32769; i++ {
		failure = s.push(r, i+1)
		if failure != nil {
			break
		}
	}
	if !errors.Is(failure, Failure{Kind: FailureMemout}) {
		t.Fatalf("actual TLS limit path %v", failure)
	}
	if s.Pending == (Block{}) || s.Raw == (Block{}) || s.capacity() != 16384 || s.size() != 16384 || r.Stats().LiveBlocks != 2 {
		t.Fatalf("completed allocation/old buffer roots lost: size%d cap%d %+v", s.size(), s.capacity(), r.Stats())
	}
	if got, e := s.read(r, 16383); e != nil || got != 16384 {
		t.Fatalf("old last initialized word%d %v", got, e)
	}
	// No fabricated free event/rollback: both actual blocks remain charged until
	// explicit cleanup, including the unpublished new allocation after TLS sync.
	used := b.Used()
	if used <= r.Stats().PhysicalBytes {
		t.Fatal("real metadata/initialization charges disappeared")
	}
	if e = s.Finalize(r); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 || s.Pending != (Block{}) || s.Raw != (Block{}) {
		t.Fatal("late allocation cleanup lost an actual root")
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("late TLS owner charge leak")
	}
}

func TestNaturalSourceInlinePointerFreedomAndWarmNoAlloc(t *testing.T) {
	if e := arena.ValidateRecordType(reflect.TypeOf(SourceLimbBuffer{})); e != nil {
		t.Fatal(e)
	}
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var s SourceLimbBuffer
	baseline := b.Used()
	state := r.State()
	var failure error
	if n := testing.AllocsPerRun(100, func() {
		s.reset()
		for i := uint32(0); i < 16; i++ {
			if e := s.push(r, i); e != nil {
				failure = e
				return
			}
		}
	}); n != 0 {
		t.Fatalf("inline heap allocations %v", n)
	}
	if failure != nil || b.Used() != baseline || r.State() != state {
		t.Fatal("inline warm path changed source memory", failure)
	}
	if e = s.Finalize(r); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("inline owner cleanup charge")
	}
}

// A corrupted scalar source position can demand an actual out-of-inline copy.
// The allocation prefix is real; the guard belongs at that endangered read.
func TestNaturalSourceCopyGuardRetainsAllocatedPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var s SourceLimbBuffer
	for i := uint32(0); i < 16; i++ {
		if e = s.push(r, i+1); e != nil {
			t.Fatal(e)
		}
	}
	s.pos = 17 // bounded undefined-source continuation fixture, not valid sbuffer state
	if e = s.expand(r); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("actual endangered copy read %v", e)
	}
	if s.Pending == (Block{}) || s.Raw != (Block{}) || s.cap != 0 || r.Stats().LiveBlocks != 1 || r.Stats().PhysicalBytes != 136 {
		t.Fatal("copy guard rolled back actual allocation prefix")
	}
	for i := uint64(0); i < 16; i++ {
		if v, e := r.ReadUint32(s.Pending, i*4); e != nil || v != uint32(i+1) {
			t.Fatal("prior actual copy stores lost", v, e)
		}
	}
	if e = s.Finalize(r); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("copy guard pending root leak")
	}
}
