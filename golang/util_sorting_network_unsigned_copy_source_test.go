// Source: src/util/sorting_network.h:28-34,87-94; src/util/vector.h:102-120,159-167,178-189.
// Source author: Nikolaj Bjorner (2013-11-07); Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUnsignedSortingNetworkGeneratedCopyMemberStorageAndPointerAliases(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	in, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	out, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	values := []uint32{9, 2, 8, 1, 7, 3, 6, 4, 5}
	for _, x := range values {
		paramsMust(t, in.Push(x))
	}
	paramsMust(t, src.Sort(in, out))
	dst, e := src.Clone()
	paramsMust(t, e)
	if dst.current != src.current || dst.next != src.next {
		t.Fatal("compiler copy preserves original member addresses")
	}
	for _, pair := range [][2]*RuntimeUnsignedVector{{&src.currentv, &dst.currentv}, {&src.nextv, &dst.nextv}} {
		a, z := pair[0], pair[1]
		ac, e := a.Capacity()
		paramsMust(t, e)
		zc, e := z.Capacity()
		paramsMust(t, e)
		an, e := a.Size()
		paramsMust(t, e)
		zn, e := z.Size()
		paramsMust(t, e)
		if ac != zc || an != zn || a.column.raw.block == z.column.raw.block {
			t.Fatal("literal vector copy exact capacity and distinct allocation", ac, zc, an, zn)
		}
		if an < ac {
			if _, e := satAssignmentReadWord(z.column.raw.runtime, z.column.raw.block, 8+4*uint64(an)); e != memory.ErrRuntimeUninitialized {
				t.Fatal("copy initializes SIZE elements, not spare capacity", e)
			}
		}
		for i := uint32(0); i < an; i++ {
			x, e := a.At(i)
			paramsMust(t, e)
			y, e := z.At(i)
			paramsMust(t, e)
			if x != y {
				t.Fatal(i, x, y)
			}
		}
	}
	paramsMust(t, dst.Sort(in, out))
	sortingUnsignedOracle(t, out, values)
	n, e := dst.currentv.Size()
	paramsMust(t, e)
	if n != 32 {
		t.Fatal("copied members append despite borrowed algorithm pointers", n)
	}
	paramsMust(t, dst.Close())
	paramsMust(t, src.Sort(in, out))
	sortingUnsignedOracle(t, out, values)
	paramsMust(t, src.Close())
	paramsMust(t, out.Close())
	paramsMust(t, in.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedSortingNetworkEveryGeneratedCopyReservationAndMemberUnwind(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		success := int64(-1)
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			b := arena.NewBudget(1 << 26)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: bits})
			paramsMust(t, e)
			src, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
			paramsMust(t, e)
			in, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			out, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			for i := uint32(9); i > 0; i-- {
				paramsMust(t, in.Push(i))
			}
			paramsMust(t, src.Sort(in, out))
			sourceA, sourceB := src.currentv.column.raw.block, src.nextv.column.raw.block
			b.FailAfter(ordinal)
			dst, cause := src.Clone()
			vecs := []*RuntimeUnsignedVector{&src.currentv, &src.nextv, in, out}
			cold := src.header
			if dst != nil {
				vecs = append(vecs, &dst.currentv, &dst.nextv)
				cold += dst.header
				if cause != nil {
					if dst.constructed || dst.current != nil || dst.next != nil || dst.constructorCause != cause {
						t.Fatal("failed outer constructor cannot publish scalar pointer members", ordinal, cause)
					}
					if dst.constructorCompleted == 1 && (!dst.currentv.constructed || !dst.currentv.sourceDone || dst.currentv.column.raw.block != (memory.Block{}) || dst.nextv.constructed) {
						t.Fatal("completed first member source DT before return", ordinal)
					}
					if dst.constructorCompleted == 0 && (dst.currentv.constructed || dst.currentv.sourceDone || dst.nextv.constructed) {
						t.Fatal("failed first member has no source DT", ordinal)
					}
					if dst.SourceFinalize() != ErrUnsignedSortingNetworkUnconstructed {
						t.Fatal("uncompleted outer class has no source destructor")
					}
				} else {
					if !dst.constructed || dst.constructorCompleted != 3 || dst.current != src.current || dst.next != src.next {
						t.Fatal("member completion then scalar pointer copy")
					}
				}
			}
			unsignedGrowthGraph(t, b, r, vecs, cold)
			if src.currentv.column.raw.block != sourceA || src.nextv.column.raw.block != sourceB {
				t.Fatal("copy does not consume original members")
			}
			b.FailAfter(-1)
			if dst != nil {
				paramsMust(t, dst.Close())
				if dst.constructorCause != cause {
					t.Fatal("own abort must preserve source constructor cause")
				}
			}
			paramsMust(t, src.Close())
			paramsMust(t, out.Close())
			paramsMust(t, in.Close())
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(bits, ordinal, b.Used())
			}
			if cause == nil {
				success = ordinal
				break
			}
		}
		if success < 0 {
			t.Fatal("copy did not reach first success", bits)
		}
		t.Log("actual copy first-success ordinal", bits, success)
	}
}

func TestUnsignedSortingNetworkCopiedPointersBorrowOriginalLifetimeOnlyAtDereference(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	in, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	out, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	empty, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	values := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1}
	for _, x := range values {
		paramsMust(t, in.Push(x))
	}
	paramsMust(t, src.Sort(in, out))
	dst, e := src.Clone()
	paramsMust(t, e)
	a, z := dst.currentv.column.raw.block, dst.nextv.column.raw.block
	paramsMust(t, src.Close())
	paramsMust(t, dst.Sort(empty, out)) // Source early return erases the ended borrowed pointers.
	n, e := dst.currentv.Size()
	paramsMust(t, e)
	if n != 16 {
		t.Fatal("unused member mappings unchanged", n)
	}
	if e = dst.Sort(in, out); e != arena.ErrClosed {
		t.Fatal("actual reached original-member lifetime guard", e)
	}
	if dst.currentv.column.raw.block == a || dst.nextv.column.raw.block == z {
		t.Fatal("own persistent index members append/grow before reached borrowed guard")
	}
	size, e := out.Size()
	paramsMust(t, e)
	if size != 16 {
		t.Fatal("caller copy/padding prefix before closed borrowed access", size)
	}
	for i, x := range values {
		y, e := out.At(uint32(i))
		paramsMust(t, e)
		if x != y {
			t.Fatal(i, x, y)
		}
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&src.currentv, &src.nextv, &dst.currentv, &dst.nextv, in, out, empty}, dst.header)
	paramsMust(t, dst.Close())
	paramsMust(t, empty.Close())
	paramsMust(t, out.Close())
	paramsMust(t, in.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedSortingNetworkGeneratedCopyClosedAssertionPrefixAndEmptyMembers(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "populated"}[populated], func(t *testing.T) {
			p, d, _, _, _ := permutationActionFixture(t, "T")
			src, e := NewRuntimeUnsignedSortingNetwork(p.forward.column.raw.runtime, d.budget, d)
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, src.Close()) })
			if populated {
				paramsMust(t, src.currentv.Resize(4, 13))
				paramsMust(t, src.nextv.Resize(4, 17))
			}
			paramsMust(t, d.Close())
			dst, cause := src.Clone()
			if dst == nil {
				t.Fatal("actual constructor placement header", cause)
			}
			if satStorageSourceDebug {
				if cause != arena.ErrClosed || dst.constructed || dst.constructorCompleted != 0 || dst.currentv.constructed || dst.currentv.sourceDone {
					t.Fatal("first vector constructor has not completed", cause)
				}
				if populated {
					h := dst.currentv.column.raw.block
					if h == (memory.Block{}) {
						t.Fatal("copy_core publishes header/pointer before source assertion113")
					}
					n, e := satAssignmentReadWord(dst.currentv.column.raw.runtime, h, 4)
					paramsMust(t, e)
					if n != 4 {
						t.Fatal(n)
					}
					if _, e = satAssignmentReadWord(dst.currentv.column.raw.runtime, h, 8); e != memory.ErrRuntimeUninitialized {
						t.Fatal("closed switch after pointer publication before first payload placement", e)
					}
				} else if dst.currentv.column.raw.block != (memory.Block{}) || dst.currentv.copyAcquisitions != nil {
					t.Fatal("null source skips copy_core then reaches constructor assertion167")
				}
			} else {
				paramsMust(t, cause)
				if !dst.constructed || dst.constructorCompleted != 3 {
					t.Fatal("release erases every closed assertion borrow")
				}
			}
			paramsMust(t, dst.Close())
		})
	}
}

func TestUnsignedSortingNetworkEmptyGeneratedCopyAndAccidentalOwnerCopy(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	before := r.State()
	dst, e := src.Clone()
	paramsMust(t, e)
	if r.State() != before || r.Stats().LiveBlocks != 0 || dst.current != src.current || dst.next != src.next || !dst.constructed {
		t.Fatal("empty source copy constructors allocate no arrays")
	}
	unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&src.currentv, &src.nextv, &dst.currentv, &dst.nextv}, src.header+dst.header)
	accidental := *dst
	used := b.Used()
	if bad, e := accidental.Clone(); bad != nil || e != ErrSATOwner || b.Used() != used {
		t.Fatal("accidental Go copy is not actual source copy construction", bad, e)
	}
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestUnsignedSortingNetworkGeneratedCopyPostphysicalFailuresRetainExactChild(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		for _, second := range []bool{false, true} {
			b := arena.NewBudget(1 << 26)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: bits})
			paramsMust(t, e)
			src, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
			paramsMust(t, e)
			in, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			out, e := NewRuntimeUnsignedVector(r, b)
			paramsMust(t, e)
			for i := uint32(9); i > 0; i-- {
				paramsMust(t, in.Push(i))
			}
			paramsMust(t, src.Sort(in, out))
			target := int64(99990)
			if second {
				target = 99900
			}
			request := target - r.State().LocalSize - int64(bits/8)
			if request <= 0 {
				t.Fatal("bounded real TLS priming", r.State())
			}
			anchor, e := r.Allocate(uint64(request))
			paramsMust(t, e)
			paramsMust(t, r.SetMaxSize(1))
			failed, cause := src.Clone()
			if cause == nil || failed == nil || !r.State().OutOfMemory || failed.constructed || failed.current != nil || failed.next != nil {
				t.Fatal("real postphysical constructor failure", cause, r.State())
			}
			child := &failed.currentv
			if second {
				child = &failed.nextv
				if failed.constructorCompleted != 1 || !failed.currentv.sourceDone || failed.currentv.column.raw.block != (memory.Block{}) {
					t.Fatal("first completed member DT before returning second failed member")
				}
			} else if failed.constructorCompleted != 0 || failed.currentv.sourceDone {
				t.Fatal("first failed member has no source DT")
			}
			a := child.copyAcquisitions
			if a == nil || !a.called || !a.acquired || a.published || a.block == (memory.Block{}) || a.sourceCause != cause || child.column.raw.block != (memory.Block{}) {
				t.Fatal("actual returned unjoined child token", a, cause)
			}
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&src.currentv, &src.nextv, &failed.currentv, &failed.nextv, in, out}, src.header+failed.header, anchor)
			paramsMust(t, r.SetMaxSize(0))
			later, e := src.Clone()
			paramsMust(t, e)
			unsignedGrowthGraph(t, b, r, []*RuntimeUnsignedVector{&src.currentv, &src.nextv, &failed.currentv, &failed.nextv, &later.currentv, &later.nextv, in, out}, src.header+failed.header+later.header, anchor)
			paramsMust(t, later.Close())
			paramsMust(t, failed.Close())
			paramsMust(t, src.Close())
			paramsMust(t, out.Close())
			paramsMust(t, in.Close())
			paramsMust(t, r.Deallocate(&anchor))
			paramsMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(bits, second, b.Used())
			}
		}
	}
}
