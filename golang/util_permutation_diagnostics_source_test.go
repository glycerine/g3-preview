// Source: src/util/permutation.cpp:21-76; src/util/debug.h:60-65.
// Source author: Leonardo de Moura (2011-06-10).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"io"
	"os"
	"strings"
	"testing"
)

func permutationDiagnosticFixture(t *testing.T, n uint32) (*RuntimePermutation, *DebugContext, *memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	d, e := NewDebugContext(region, b)
	paramsMust(t, e)
	p, e := NewRuntimePermutationWithAssertions(r, b, n, d)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, p.Close())
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("unnamed diagnostic owner", b.Used())
		}
	})
	return p, d, r, b
}
func TestRuntimePermutationAssertionsActualPartialResetAndScalarReads(t *testing.T) {
	p, d, _, b := permutationDiagnosticFixture(t, 2)
	b.FailAfter(1)
	if e := p.Reset(3); e == nil {
		t.Fatal("actual inverse growth must refuse")
	}
	permutationDiagnosticPhysical(t, d, p.forward.column.raw.runtime, b, nil, p)
	a, e := p.forward.Size()
	paramsMust(t, e)
	z, e := p.inverse.Size()
	paramsMust(t, e)
	if a != 3 || z != 2 {
		t.Fatal("completed alternating prefix", a, z)
	}
	paramsMust(t, d.EnableAssertions(false))
	yes, e := p.CheckInvariant()
	paramsMust(t, e)
	if !yes {
		t.Fatal("erased source conditions")
	}
	paramsMust(t, d.EnableAssertions(true))
	_, e = p.CheckInvariant()
	if satStorageSourceDebug && !errors.Is(e, ErrSATAssertion) {
		t.Fatal("actual size equality", e)
	}
	b.FailAfter(-1)
	paramsMust(t, p.Reset(2))
	paramsMust(t, p.forward.Reset())
	paramsMust(t, d.EnableAssertions(false))
	v, e := p.At(1)
	paramsMust(t, e)
	if v != 1 {
		t.Fatal("initialized spare", v)
	}
	paramsMust(t, d.EnableAssertions(true))
	_, e = p.At(1)
	if satStorageSourceDebug && !errors.Is(e, ErrSATAssertion) {
		t.Fatal("source indexed assertion", e)
	}
	paramsMust(t, d.Close())
	paramsMust(t, p.MoveAfter(7, 3)) // source early return erases all switch reads.
	_, e = p.At(0)
	if satStorageSourceDebug && !errors.Is(e, arena.ErrClosed) {
		t.Fatal("switch before indexed condition", e)
	}
}
func TestRuntimePermutationClosedAssertionCopyPublication(t *testing.T) {
	p, d, r, b := permutationDiagnosticFixture(t, 2)
	paramsMust(t, d.Close())
	out, e := p.Clone()
	if out == nil {
		t.Fatal("real copy context", e)
	}
	t.Cleanup(func() { paramsMust(t, out.Close()) })
	permutationDiagnosticPhysical(t, d, r, b, nil, p, out)
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrClosed) || out.completedMembers != 0 || out.forward.column.raw.block == (memory.Block{}) || out.forward.sourceDone {
			t.Fatal("copy pointer before switch, incomplete member no destructor", e, out.completedMembers)
		}
		if r.Stats().LiveBlocks != 3 {
			t.Fatal("retained published failed member", r.Stats())
		}
	} else {
		paramsMust(t, e)
	}
}
func TestRuntimePermutationDisplayIntoExactFormattingAndBadbitReads(t *testing.T) {
	p, d, _, b := permutationDiagnosticFixture(t, 3)
	paramsMust(t, p.Swap(0, 2))
	s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, s.Close()) })
	paramsMust(t, s.SetIntegerBase(16))
	paramsMust(t, s.SetShowBase(true))
	paramsMust(t, s.SetUppercase(true))
	paramsMust(t, s.SetWidth(6))
	paramsMust(t, s.SetFill('_'))
	paramsMust(t, p.DisplayInto(s))
	got, e := s.String()
	paramsMust(t, e)
	if got != "_____0:0X2 0X1:0X1 0X2:0" {
		t.Fatal(got)
	}
	f, e := os.CreateTemp(".", "permutation-display-*")
	paramsMust(t, e)
	t.Cleanup(func() { f.Close(); os.Remove(f.Name()) })
	fs, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, fs.Close()) })
	b.FailAfter(0)
	paramsMust(t, p.DisplayInto(fs))
	b.FailAfter(-1)
	_, e = f.Seek(0, io.SeekStart)
	paramsMust(t, e)
	bytes, e := io.ReadAll(f)
	paramsMust(t, e)
	if string(bytes) != "0:2 1:1 2:0" {
		t.Fatal(string(bytes))
	}
	paramsMust(t, f.Close())
	paramsMust(t, d.Close())
	e = p.DisplayInto(fs)
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("nonthrowing badbit must still evaluate indexed operand", e)
		}
	} else {
		paramsMust(t, e)
	}
	bad, e := fs.Bad()
	paramsMust(t, e)
	if !bad {
		t.Fatal("real closed file badbit")
	}
}
func TestRuntimePermutationEmptyDisplayLeavesStreamUnread(t *testing.T) {
	p, d, _, _ := permutationDiagnosticFixture(t, 0)
	paramsMust(t, d.Close())
	paramsMust(t, p.DisplayInto(nil))
}

func permutationDiagnosticPhysical(t *testing.T, d *DebugContext, r *memory.Runtime, b *arena.Budget, streams []*NumericTextStream, owners ...*RuntimePermutation) {
	t.Helper()
	var cold uint64
	for _, p := range owners {
		if p != nil {
			cold += p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse)
		}
	}
	// Use the actual caller source column as primary; other roots come only
	// from explicitly passed owner fields, never live-record ownership inference.
	primary := &owners[0].forward.column.raw
	var roots []memory.Block
	for _, p := range owners {
		if p == nil {
			continue
		}
		for _, v := range []*RuntimeUnsignedVector{&p.forward, &p.inverse} {
			for a := v.copyAcquisitions; a != nil; a = a.next {
				if a.block != (memory.Block{}) {
					if !a.acquired || a.published {
						t.Fatal("actual copy acquisition transfer")
					}
					roots = append(roots, a.block)
				}
			}
			if &v.column.raw == primary {
				continue
			}
			if !v.assignmentDestroyed && !v.column.raw.pendingReplaces && v.column.raw.block != (memory.Block{}) {
				roots = append(roots, v.column.raw.block)
			}
			if v.column.raw.pending != (memory.Block{}) {
				roots = append(roots, v.column.raw.pending)
			}
		}
	}
	satAssignmentAuditBlocks(t, primary, roots...)
	x := d.region.Stats()
	a := x.Descriptors
	cold += x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if !d.closed {
		cold += d.ownerBytes
	}
	if d.tags != nil {
		h := d.tags.Stats()
		cold += h.OwnerBytes + h.TableBytes + h.DirectoryBytes
	}
	for _, s := range streams {
		if s != nil {
			h := s.Stats()
			cold += h.OwnerBytes + h.ScratchBytes + h.OutputOwnerBytes + h.OutputOverflowBytes
		}
	}
	if b.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("actual source/destination cold/raw sum", b.Used(), r.Stats().TotalBytes, cold)
	}
}

func TestRuntimePermutationAssertionConstructorReachedGrowthAndUnwind(t *testing.T) {
	base, d, r, b := permutationDiagnosticFixture(t, 0)
	paramsMust(t, d.Close())
	p, e := NewRuntimePermutationWithAssertions(r, b, 3, d)
	if p == nil {
		t.Fatal("closed borrow must not preflight constructor", e)
	}
	t.Cleanup(func() { paramsMust(t, p.Close()) })
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrClosed) || !p.inverse.sourceDone || !p.forward.sourceDone || r.Stats().LiveBlocks != 0 {
			t.Fatal("completed members reverse source unwind", e, r.Stats())
		}
	} else {
		paramsMust(t, e)
	}
	permutationDiagnosticPhysical(t, d, r, b, nil, base, p)
}
func TestRuntimePermutationDiagnosticConstructorEveryActualReservation(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 40; ordinal++ {
		base, d, r, b := permutationDiagnosticFixture(t, 0)
		b.FailAfter(ordinal)
		p, e := NewRuntimePermutationWithAssertions(r, b, 9, d)
		permutationDiagnosticPhysical(t, d, r, b, nil, base, p)
		if e == nil {
			success = int(ordinal)
			permutationExpect(t, p, []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8})
		}
		b.FailAfter(-1)
		if p != nil {
			paramsMust(t, p.Close())
		}
		if e == nil {
			break
		}
	}
	if success < 0 {
		t.Fatal("no source constructor success frontier")
	}
	t.Log("first successful actual reservation ordinal", success)
}
func TestRuntimePermutationFormattedDisplayEveryOutputReservation(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		p, d, r, b := permutationDiagnosticFixture(t, 3)
		s, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		t.Cleanup(func() { b.FailAfter(-1); paramsMust(t, s.Close()) })
		paramsMust(t, s.SetWidth(150))
		paramsMust(t, s.SetFill('_'))
		b.FailAfter(ordinal)
		e = p.DisplayInto(s)
		permutationDiagnosticPhysical(t, d, r, b, []*NumericTextStream{s}, p)
		got, se := s.String()
		paramsMust(t, se)
		expected := strings.Repeat("_", 149) + "0:0 1:1 2:2"
		if !strings.HasPrefix(expected, got) {
			t.Fatal("literal output prefix", len(got), got)
		}
		if e == nil {
			if got != expected {
				t.Fatal("complete output", got)
			}
			success = int(ordinal)
		}
		b.FailAfter(-1)
		paramsMust(t, s.Close())
		if e == nil {
			break
		}
	}
	if success < 0 {
		t.Fatal("no display success frontier")
	}
	t.Log("first successful output reservation ordinal", success)
}
func TestRuntimePermutationBindingRejectsForeignCohortBeforeReservation(t *testing.T) {
	p, d, r, b := permutationDiagnosticFixture(t, 2)
	foreign, _, _ := debugFixture(t)
	before := b.Used()
	_, e := NewRuntimePermutationWithAssertions(r, b, 2, foreign)
	if !errors.Is(e, arena.ErrConfig) || b.Used() != before {
		t.Fatal("foreign cohort before constructor reservation", e)
	}
	if e = p.BindAssertions(foreign); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
	if p.debugContext != d {
		t.Fatal("borrow replaced on rejection")
	}
}
