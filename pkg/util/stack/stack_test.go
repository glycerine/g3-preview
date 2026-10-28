package stack

// Source: src/util/stack.h:24-49; src/util/stack.cpp:24-137.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func mustStackAllocate(t *testing.T, stack *Stack, size uint32) StackRef {
	t.Helper()
	ref, err := stack.Allocate(size)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestStackSmallBigZeroLIFOAndReset(t *testing.T) {
	stack, err := NewStack()
	if err != nil {
		t.Fatal(err)
	}
	zero, err := stack.Allocate(0)
	if err != nil {
		t.Fatal(err)
	}
	small := mustStackAllocate(t, stack, 17)
	big := mustStackAllocate(t, stack, 9000)
	if size, _ := stack.Size(zero); size != 0 {
		t.Fatalf("zero size = %d", size)
	}
	if err = stack.Write(small, 16, 0xa5); err != nil {
		t.Fatal(err)
	}
	if value, err := stack.Read(small, 16); err != nil || value != 0xa5 {
		t.Fatalf("read = %x, %v", value, err)
	}
	if err = stack.Write(big, 8999, 0x5a); err != nil {
		t.Fatal(err)
	}
	if top, _ := stack.Top(); top != big {
		t.Fatal("big allocation is not top")
	}
	if err = stack.Deallocate(); err != nil {
		t.Fatal(err)
	}
	if _, err = stack.Read(big, 0); !errors.Is(err, ErrStackOwner) {
		t.Fatal("popped allocation remained live", err)
	}
	if top, _ := stack.Top(); top != small {
		t.Fatal("small allocation was not restored")
	}
	if err = stack.Reset(); err != nil || !stack.Empty() {
		t.Fatal("reset", err)
	}
	if _, err = stack.Top(); !errors.Is(err, ErrStackEmpty) {
		t.Fatal("empty top", err)
	}
	if err = stack.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStackPageRecyclingAndBigPageRelease(t *testing.T) {
	stack, err := NewStack()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if _, err = stack.Allocate(512); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = stack.Allocate(9000); err != nil {
		t.Fatal(err)
	}
	before, err := stack.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if before.DefaultPages < 2 || before.BigPages != 1 {
		t.Fatalf("pages before reset = %#v", before)
	}
	if err = stack.Reset(); err != nil {
		t.Fatal(err)
	}
	after, err := stack.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if after.BigPages != 0 || after.FreePages == 0 || after.Live != 0 {
		t.Fatalf("pages after reset = %#v", after)
	}
	free := after.FreePages
	for i := 0; i < 40; i++ {
		if _, err = stack.Allocate(512); err != nil {
			t.Fatal(err)
		}
		current, statsErr := stack.Stats()
		if statsErr != nil {
			t.Fatal(statsErr)
		}
		if current.FreePages < free {
			break
		}
	}
	reused, err := stack.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if reused.FreePages >= free {
		t.Fatalf("default page was not reused: before %d after %d", free, reused.FreePages)
	}
	_ = stack.Close()
}

func TestStackPlacementDeleteAndDestructorOrder(t *testing.T) {
	stack, err := NewStack()
	if err != nil {
		t.Fatal(err)
	}
	first := mustStackAllocate(t, stack, 8)
	second := mustStackAllocate(t, stack, 8)
	if err := stack.DeallocateRef(first); !errors.Is(err, ErrStackOwner) {
		t.Fatalf("non-top placement delete = %v", err)
	}
	destroyed := false
	if err := stack.DeallocateValue(second, func() { destroyed = true }); err != nil || !destroyed {
		t.Fatal("typed deallocation", err, destroyed)
	}
	if top, _ := stack.Top(); top != first {
		t.Fatal("wrong top after typed deallocation")
	}
	if err := stack.DeallocateRef(first); err != nil {
		t.Fatal(err)
	}
	if err := stack.Deallocate(); !errors.Is(err, ErrStackEmpty) {
		t.Fatal("empty deallocation", err)
	}
	_ = stack.Close()
}

func TestStackOwnerBoundsAndClosePreconditions(t *testing.T) {
	stack, err := NewStack()
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewStack()
	if err != nil {
		t.Fatal(err)
	}
	ref := mustStackAllocate(t, stack, 2)
	if err := stack.Write(ref, 2, 1); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("bounds", err)
	}
	if _, err := other.Read(ref, 0); !errors.Is(err, ErrStackOwner) {
		t.Fatal("foreign owner", err)
	}
	copyOfStack := *stack
	if _, err := copyOfStack.Top(); !errors.Is(err, ErrStackOwner) {
		t.Fatal("copied owner", err)
	}
	if err := stack.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := stack.Allocate(1); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
	if err := stack.Close(); err != nil {
		t.Fatal("repeated close", err)
	}
	_ = other.Close()
}
