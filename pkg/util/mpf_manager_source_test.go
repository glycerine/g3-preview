// Source: src/util/mpf.cpp:61-67; src/util/mpf.h:65-73,241-259.
// Source author: Christoph Wintersteiger (2011-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatSourceConstructorActualFourPowerMaps(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	cfg := IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	m, e := NewArbitraryFloatManager(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if m.integer != m.rational.Z || m.rational.Synchronized {
		t.Fatal("actual unsynchronized mpq and borrowed same mpz", m)
	}
	var maps uint64
	for i := range m.powers {
		p := &m.powers[i]
		if p.Capacity() != 8 || !p.Empty() {
			t.Fatal("actual default unsigned-pointer map", i, p.Capacity(), p.Size())
		}
		maps += p.StorageBytes()
	}
	if b.Used() != m.header+m.rational.HeaderBytes+integerActualOwnedBytes(m.integer)+maps {
		t.Fatal("exact constructor physical/cold graph", b.Used(), m.header, maps)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != r.Stats().TotalBytes {
		t.Fatal("borrowed Runtime survived", b.Used(), r.Stats())
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("actual source manager owner end", b.Used())
	}
}

func arbitraryFloatSourceCold(t *testing.T, m *ArbitraryFloatManager, r *Runtime, foreign ...Block) uint64 {
	t.Helper()
	n := r.Stats().TotalBytes + m.header
	var z *IntegerManager
	if m.rational != nil {
		n += m.rational.HeaderBytes
		z = m.rational.Z
	} else {
		var failed *IntegerManager
		if errors.As(m.failedChild, &failed) {
			z = failed
		}
	}
	if z != nil {
		n += z.HeaderBytes + z.Allocator.Stats().TotalBytes
		if z.Cells != nil {
			v := z.Cells.Stats()
			n += v.OwnerBytes + v.RecordBytes + v.MetadataBytes + v.DirectoryBytes
		}
		if z.Words != nil {
			v := z.Words.Stats()
			n += v.OwnerBytes + v.PayloadBytes + v.BlockMetadataBytes + v.DirectoryBytes + v.Descriptors.OwnerBytes + v.Descriptors.RecordBytes + v.Descriptors.MetadataBytes + v.Descriptors.DirectoryBytes
		}
		if z.Natural != nil {
			n += z.Natural.HeaderBytes
			if z.Natural.Text != nil {
				v := z.Natural.Text.Stats()
				n += v.OwnerBytes + v.PayloadBytes + v.MetadataBytes + v.DirectoryBytes
			}
			integerPhysicalAuditWithRaw(t, z, foreign)
		} else {
			sourcePreparedEarlyIntegerPhysical(t, z)
		}
	}
	for i := range m.powers {
		n += m.powers[i].StorageBytes()
	}
	return n
}
func TestArbitraryFloatEveryActualManagerAndMapConstructorPrefix(t *testing.T) {
	done, failed := false, false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 28)
		r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
		if e != nil {
			t.Fatal(e)
		}
		cfg := IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
		b.FailAfter(ordinal)
		m, call := NewArbitraryFloatManager(cfg)
		if call != nil {
			failed = true
			var retained *ArbitraryFloatManager
			if errors.As(call, &retained) {
				m = retained
				if m.constructorComplete || m.bodyCause == nil {
					t.Fatal("unfinished constructor became callable", ordinal, m)
				}
			}
		}
		if m != nil {
			if b.Used() != arbitraryFloatSourceCold(t, m, r) {
				t.Fatal("before lift exact reached constructor graph", ordinal, b.Used(), arbitraryFloatSourceCold(t, m, r))
			}
		}
		if m == nil {
			if !errors.Is(call, arena.ErrInjectedFailure) || b.Used() != r.Stats().TotalBytes || r.Stats().LiveBlocks != 0 {
				t.Fatal("before lift refused outer header left only actual borrowed Runtime", ordinal, call, b.Used(), r.Stats())
			}
		}
		if call == nil {
			done = true
		}
		b.FailAfter(-1)
		if m != nil {
			if e = m.Close(); e != nil {
				t.Fatal(ordinal, e)
			}
		}
		if b.Used() != r.Stats().TotalBytes {
			t.Fatal("own only acquired children", ordinal, b.Used(), r.Stats())
		}
		if e = r.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("completed own context", ordinal, b.Used())
		}
		if done {
			t.Log("first full MPF manager/powers2 constructor", ordinal)
			break
		}
	}
	if !done || !failed {
		t.Fatal("whole actual constructor horizon", done, failed)
	}
}
func TestArbitraryFloatActualSourceFinalizeThenOwnClose(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := r.Allocate(19)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.Write(foreign, 0, 73); e != nil {
		t.Fatal(e)
	}
	if e = m.SourceFinalize(); e != nil {
		t.Fatal(e)
	}
	if !m.sourceEnded || m.header == 0 || m.rational.HeaderBytes == 0 {
		t.Fatal("actual source members vs own/borrowed lifetimes", m, r.Stats())
	}
	if b.Used() != arbitraryFloatSourceCold(t, m, r, foreign) {
		t.Fatal("charged source-ended contexts", b.Used(), arbitraryFloatSourceCold(t, m, r, foreign))
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if value, read := r.Read(foreign, 0); read != nil || value != 73 {
		t.Fatal("actual foreign payload after source/own", value, read)
	}
	if b.Used() != r.Stats().TotalBytes {
		t.Fatal("foreign raw block survived", b.Used(), r.Stats())
	}
	if e = r.Deallocate(&foreign); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("context end", b.Used())
	}
}

func TestArbitraryFloatActualBorrowedServiceFactory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	q, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := q.NewValue()
	if e != nil {
		t.Fatal(e)
	}
	if e = q.SetUint64(&foreign, 37); e != nil {
		t.Fatal(e)
	}
	m, e := q.NewArbitraryFloatManager()
	if e != nil {
		t.Fatal(e)
	}
	if m.integer == q.Manager.Z || m.memory != r {
		t.Fatal("new actual unsynchronized member, borrowed shared Runtime", m)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	value, e := q.Uint64(&foreign)
	if e != nil || value != 37 {
		t.Fatal("unrelated source service survived", value, e)
	}
	if e = q.Del(&foreign); e != nil {
		t.Fatal(e)
	}
	if e = q.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("actual factory own completion", b.Used())
	}
}

// This tests the actual returning Go transport profile, not a C++ callback.
func TestArbitraryFloatServiceFactoryActualTransportExclusion(t *testing.T) {
	if os.Getenv("G3_MPF_FACTORY_CALLBACK") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestArbitraryFloatServiceFactoryActualTransportExclusion$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_MPF_FACTORY_CALLBACK=1")
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("actual factory transport child: %v %s", e, out)
		}
		return
	}
	b := arena.NewBudget(1 << 28)
	var q *RationalService
	active, calls := false, 0
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if active {
			calls++
			if e := q.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("borrowed parent close", e)
			}
			if e := q.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("borrowed parent source DT", e)
			}
			if _, e := q.Stats(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("borrowed parent read", e)
			}
			if _, e := q.NewArbitraryFloatManager(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("recursive factory", e)
			}
		}
		return f
	}})
	if e != nil {
		t.Fatal(e)
	}
	q, e = NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	old := q.Manager
	padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	if e != nil {
		t.Fatal(e)
	}
	if e = r.SetMaxSize(1); e != nil {
		t.Fatal(e)
	}
	active = true
	_, call := q.NewArbitraryFloatManager()
	active = false
	var failed *ArbitraryFloatManager
	if calls == 0 || !errors.As(call, &failed) || q.Manager != old || r.FailureTransportActive() {
		t.Fatal("actual postphysical failed constructor", calls, call)
	}
	// Exact named old member tokens and the returned unfinished child roots.
	// Direct profile makes each reached numeric object's raw address independent.
	raw := map[Block]bool{padding: true}
	add := func(z *IntegerManager) {
		refs := append([]NumericCellRef{}, z.Tmp[:]...)
		refs = append(refs, z.Arg[:]...)
		refs = append(refs, z.IntMin.Cell, z.Two64.Cell)
		for _, ref := range refs {
			if ref == 0 || z.IsFailureRef(ref) {
				continue
			}
			c := z.Cell(ref)
			if c.Released {
				continue
			}
			if c.Backing.Raw != (Block{}) {
				raw[c.Backing.Raw] = true
			}
		}
		if z.PendingCell.Raw != (Block{}) {
			raw[z.PendingCell.Raw] = true
		}
		p, e := z.Allocator.PendingRoots()
		if e != nil {
			t.Fatal(e)
		}
		for _, block := range p {
			if block != (Block{}) {
				raw[block] = true
			}
		}
	}
	add(q.Manager.Z)
	var child *IntegerManager
	if !errors.As(failed.failedChild, &child) {
		t.Fatal("actual returned failed Integer owner", failed.failedChild)
	}
	add(child)
	for block := range raw {
		if _, e := r.Size(block); e != nil {
			t.Fatal("named acquisition lifetime", block, e)
		}
	}
	if uint32(len(raw)) != r.Stats().LiveBlocks {
		t.Fatal("immediate exact physical graph", len(raw), r.Stats())
	}
	if e = r.SetMaxSize(0); e != nil {
		t.Fatal(e)
	}
	if e = failed.Close(); e != nil {
		t.Fatal(e)
	}
	if q.Manager != old {
		t.Fatal("failed child adopted borrowed service")
	}
	if e = q.Close(); e != nil {
		t.Fatal(e)
	}
	if e = r.Deallocate(&padding); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("exact caller context end", b.Used())
	}
}
