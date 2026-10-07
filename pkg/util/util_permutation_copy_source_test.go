// Source: src/util/permutation.h:25-40; src/util/vector.h:102-120,159-167,191-202,536-541.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRuntimePermutationCopyMemberOrderAndIndependentDestination(t *testing.T) {
	p, _, _ := utilityRuntimePermutationFixture(t, 4)
	paramsMust(t, p.Swap(0, 3))
	out, e := p.Clone()
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	permutationExpect(t, out, []uint32{3, 1, 2, 0})
	paramsMust(t, out.MoveAfter(0, 2))
	permutationExpect(t, p, []uint32{3, 1, 2, 0})
	permutationExpect(t, out, []uint32{1, 2, 3, 0})
	paramsMust(t, p.Assign(out))
	permutationExpect(t, p, []uint32{1, 2, 3, 0})
	paramsMust(t, p.Assign(p))
	permutationExpect(t, p, []uint32{1, 2, 3, 0})
}
func TestRuntimePermutationFailedInverseCopyDestructsOnlyCompletedForward(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 2)
	b.FailAfter(5)
	out, e := p.Clone()
	if out != nil {
		t.Cleanup(func() { b.FailAfter(-1); paramsMust(t, out.Close()) })
	}
	if e == nil || out == nil {
		t.Fatal("actual inverse copy refusal", e)
	}
	if out.completedMembers != 1 || !out.forward.sourceDone || out.inverse.sourceDone {
		t.Fatal("completed forward only source unwind", out.completedMembers, out.destructorStage)
	}
	if r.Stats().LiveBlocks != 2 {
		t.Fatal("source forward unwind before returning failed ctor", r.Stats())
	}
	if !errors.Is(e, arena.ErrInjectedFailure) {
		var f Failure
		if !errors.As(e, &f) || f.ErrorCode() != 101 {
			t.Fatal(e)
		}
	}
	b.FailAfter(-1)
	paramsMust(t, out.Close())
	permutationExpect(t, p, []uint32{0, 1})
}

func permutationCopyPhysical(t *testing.T, r *Runtime, b *arena.Budget, owners ...*RuntimePermutation) {
	t.Helper()
	var cold uint64
	for _, p := range owners {
		if p != nil {
			cold += p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse)
		}
	}
	// Use the actual caller source column as primary; other roots come only
	// from explicitly passed owner fields, never live-record ownership inference.
	primary := &owners[0].forward.Column.Raw
	var roots []Block
	for _, p := range owners {
		if p == nil {
			continue
		}
		for _, v := range []*RuntimeUnsignedVector{&p.forward, &p.inverse} {
			for a := v.copyAcquisitions; a != nil; a = a.next {
				if a.block != (Block{}) {
					if !a.acquired || a.published {
						t.Fatal("actual copy acquisition transfer")
					}
					roots = append(roots, a.block)
				}
			}
			if &v.Column.Raw == primary {
				continue
			}
			if !v.assignmentDestroyed && !v.Column.Raw.PendingReplaces && v.Column.Raw.Block != (Block{}) {
				roots = append(roots, v.Column.Raw.Block)
			}
			if v.Column.Raw.Pending != (Block{}) {
				roots = append(roots, v.Column.Raw.Pending)
			}
		}
	}
	runtimeVectorAuditBlocks(t, primary, roots...)
	if b.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("actual source/destination cold/raw sum", b.Used(), r.Stats().TotalBytes, cold)
	}
}
func TestRuntimePermutationCopyEveryReservationAndConstructorUnwind(t *testing.T) {
	for _, n := range []uint32{0, 1, 9} {
		t.Run(string(rune('a'+n)), func(t *testing.T) {
			success := -1
			for ordinal := int64(0); ordinal < 50; ordinal++ {
				p, r, b := utilityRuntimePermutationFixture(t, n)
				b.FailAfter(ordinal)
				out, e := p.Clone()
				permutationCopyPhysical(t, r, b, p, out)
				if e != nil {
					permutationCopyRefusal(t, e, r)
				}
				if e == nil {
					success = int(ordinal)
					want := make([]uint32, n)
					for i := range want {
						want[i] = uint32(i)
					}
					permutationExpect(t, out, want)
				} else if out != nil {
					if out.completedMembers == 3 {
						t.Fatal("failure cannot complete both members")
					}
					if out.completedMembers&1 != 0 && !out.forward.sourceDone {
						t.Fatal("source completed forward must unwind before return")
					}
					if out.inverse.sourceDone {
						t.Fatal("failed inverse ctor has no source destructor")
					}
				}
				b.FailAfter(-1)
				if out != nil {
					paramsMust(t, out.Close())
				}
				permutationCopyPhysical(t, r, b, p)
				if success >= 0 {
					break
				}
			}
			if success < 0 {
				t.Fatal("no copy first success")
			}
			t.Log("copy first success", n, success)
		})
	}
}
func TestRuntimePermutationAssignEveryReservationAndLiteralMemberPrefix(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 50; ordinal++ {
		p, r, b := utilityRuntimePermutationFixture(t, 9)
		paramsMust(t, p.Swap(0, 8))
		dst, e := NewRuntimePermutation(r, b, 3)
		paramsMust(t, e)
		b.FailAfter(ordinal)
		e = dst.Assign(p)
		permutationCopyPhysical(t, r, b, p, dst)
		if e != nil {
			permutationCopyRefusal(t, e, r)
		}
		if e == nil {
			success = int(ordinal)
			permutationExpect(t, dst, []uint32{8, 1, 2, 3, 4, 5, 6, 7, 0})
		} else {
			if !dst.forward.assignmentDestroyed {
				v, x := dst.At(0)
				paramsMust(t, x)
				if v != 8 {
					t.Fatal("forward copied before inverse destroy/copy refusal", ordinal, v)
				}
			}
			// Implicit self-assignment evaluates member pointer identity and skips
			// destruction; no attempt to reconstruct or read a dangling member.
			state := r.State()
			used := b.Used()
			paramsMust(t, dst.Assign(dst))
			if r.State() != state || b.Used() != used {
				t.Fatal("self assignment changed failure prefix")
			}
		}
		permutationExpect(t, p, []uint32{8, 1, 2, 3, 4, 5, 6, 7, 0})
		b.FailAfter(-1)
		if x := dst.Close(); x != nil {
			paramsMust(t, dst.CleanupStopped())
		}
		permutationCopyPhysical(t, r, b, p)
		if success >= 0 {
			break
		}
	}
	if success < 0 {
		t.Fatal("no assignment first success")
	}
	t.Log("assignment first success", success)
}

func TestRuntimePermutationFailedCopyReadHasNoUncompletedMemberDestructor(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 2)
	// Source-undefined guarded domain: source header claims a third element
	// beyond its actual two-element backing. No source field repair on failure.
	paramsMust(t, WriteRuntimeWord(r, p.forward.Column.Raw.Block, 4, 3))
	out, e := p.Clone()
	if !errors.Is(e, arena.ErrBounds) || out == nil {
		t.Fatal("reached third iterator dereference", e)
	}
	if out.completedMembers != 0 || out.forward.sourceDone || out.inverse.sourceDone || out.forward.Column.Raw.Block == (Block{}) || out.inverse.Column.Raw.Block != (Block{}) {
		t.Fatal("failed first member ctor has no vector DT")
	}
	permutationCopyPhysical(t, r, b, p, out)
	for i := uint32(0); i < 2; i++ {
		v, e := ReadRuntimeWord(r, out.forward.Column.Raw.Block, 8+4*uint64(i))
		paramsMust(t, e)
		if v != i {
			t.Fatal("completed source copy placements", i, v)
		}
	}
	paramsMust(t, out.Close())
	permutationCopyPhysical(t, r, b, p)
}
func TestRuntimePermutationAssignNullSourceAndSelfUseNoAllocation(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 3)
	empty, e := NewRuntimePermutation(r, b, 0)
	paramsMust(t, e)
	b.FailAfter(0)
	paramsMust(t, p.Assign(empty))
	paramsMust(t, p.Assign(p))
	if p.forward.Column.Raw.Block != (Block{}) || p.inverse.Column.Raw.Block != (Block{}) {
		t.Fatal("actual null-source publication")
	}
	permutationCopyPhysical(t, r, b, p, empty)
	b.FailAfter(-1)
	paramsMust(t, empty.Close())
}

func permutationCopyRefusal(t *testing.T, e error, r *Runtime) {
	t.Helper()
	if errors.Is(e, arena.ErrInjectedFailure) {
		return
	}
	var f Failure
	if !errors.As(e, &f) || f.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal("actual same Runtime allocation refusal", e, r.State())
	}
}

func TestRuntimePermutationAssignStoppedDeallocateUsesExactConsumedFieldReceipt(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	src, e := NewRuntimePermutation(r, b, 30000)
	paramsMust(t, e)
	dst, e := NewRuntimePermutation(r, b, 30000)
	paramsMust(t, e)
	old := dst.forward.Column.Raw.Block
	inverse := dst.inverse.Column.Raw.Block
	contextTLSNearMinimum(t, r)
	first := dst.Assign(src)
	if !errors.Is(first, arena.ErrBounds) {
		t.Fatal("reached actual destroy physical free/TLS stop", first)
	}
	if !dst.forward.assignmentDestroyed || dst.forward.Column.Raw.Block != old || dst.inverse.Column.Raw.Block != inverse || dst.forward.assignmentCause != first {
		t.Fatal("literal source dead pointer and exact consumed receipt")
	}
	permutationCopyPhysical(t, r, b, src, dst)
	dtor := dst.Close()
	if !errors.Is(dtor, arena.ErrBounds) || dst.destructorStage != 0 || dst.inverse.Column.Raw.Block != (Block{}) {
		t.Fatal("inverse source destructor stop before older dead forward", dtor)
	}
	permutationCopyPhysical(t, r, b, src, dst)
	paramsMust(t, dst.CleanupStopped())
	if dst.forward.assignmentCause != first || dst.destructorCause != dtor {
		t.Fatal("owning acknowledgment repaired source causes")
	}
	permutationCopyPhysical(t, r, b, src)
	if e = src.Close(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual remaining caller inverse physical stop", e)
	}
	e = src.CleanupStopped()
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual remaining forward physical stop", e)
	}
	paramsMust(t, src.CleanupStopped())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("exact once-only actual owner refund", b.Used())
	}
}

func TestRuntimePermutationFailedSecondCopyReadRetainsUncompletedInverse(t *testing.T) {
	p, r, b := utilityRuntimePermutationFixture(t, 2)
	// Guarded malformed source size: first member completes, second member
	// allocates/publishes/copies two words then its third dereference refuses.
	paramsMust(t, WriteRuntimeWord(r, p.inverse.Column.Raw.Block, 4, 3))
	out, e := p.Clone()
	if !errors.Is(e, arena.ErrBounds) || out == nil {
		t.Fatal("actual second member read guard", e)
	}
	if out.completedMembers != 1 || !out.forward.sourceDone || out.inverse.sourceDone || out.forward.Column.Raw.Block != (Block{}) || out.inverse.Column.Raw.Block == (Block{}) {
		t.Fatal("completed forward DT, uncompleted inverse retained")
	}
	permutationCopyPhysical(t, r, b, p, out)
	for i := uint32(0); i < 2; i++ {
		v, e := ReadRuntimeWord(r, out.inverse.Column.Raw.Block, 8+4*uint64(i))
		paramsMust(t, e)
		if v != i {
			t.Fatal("inverse completed placement prefix", i, v)
		}
	}
	paramsMust(t, out.Close())
	permutationCopyPhysical(t, r, b, p)
}
