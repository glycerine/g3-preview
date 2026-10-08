// Source: src/util/stack.h:24-49; src/util/stack.cpp:24-137.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrStackOwner = errors.New("g3: invalid stack owner or allocation")
var ErrStackEmpty = errors.New("g3: empty stack")

// StackRef is a checked replacement for a raw pointer returned by stack.
type StackRef struct {
	owner  *Stack
	region arena.BytesRegionRef
	serial uint64
}

type stackEntry struct {
	ref  StackRef
	mark arena.BytesRegionMark
	size uint32
}

// Stack is a LIFO byte allocator backed by source-shaped recyclable pages.
type Stack struct {
	self    *Stack
	region  *arena.BytesRegion
	entries []stackEntry
	serial  uint64
	closed  bool
}

func NewStack() (*Stack, error) { return NewStackWithConfig(arena.BytesRegionConfig{}) }

func NewStackWithConfig(config arena.BytesRegionConfig) (*Stack, error) {
	region, err := arena.NewBytesRegion(config)
	if err != nil {
		return nil, err
	}
	stack := &Stack{region: region}
	stack.self = stack
	return stack, nil
}

func (stack *Stack) requireOpen() error {
	if stack == nil || stack.self != stack {
		return ErrStackOwner
	}
	if stack.closed {
		return arena.ErrClosed
	}
	return nil
}

func (stack *Stack) allocation(ref StackRef) (stackEntry, error) {
	if err := stack.requireOpen(); err != nil {
		return stackEntry{}, err
	}
	if ref.owner != stack || ref.serial == 0 || !stack.region.IsLive(ref.region) {
		return stackEntry{}, ErrStackOwner
	}
	for index := len(stack.entries) - 1; index >= 0; index-- {
		if stack.entries[index].ref.serial == ref.serial {
			return stack.entries[index], nil
		}
	}
	return stackEntry{}, ErrStackOwner
}

func (stack *Stack) Allocate(size uint32) (StackRef, error) {
	if err := stack.requireOpen(); err != nil {
		return StackRef{}, err
	}
	if stack.serial == ^uint64(0) {
		return StackRef{}, arena.ErrSlotLimit
	}
	mark, err := stack.region.PushScope()
	if err != nil {
		return StackRef{}, err
	}
	regionRef, err := stack.region.Allocate(size)
	if err != nil {
		_ = stack.region.Rollback(mark)
		return StackRef{}, err
	}
	stack.serial++
	ref := StackRef{owner: stack, region: regionRef, serial: stack.serial}
	stack.entries = append(stack.entries, stackEntry{ref: ref, mark: mark, size: size})
	return ref, nil
}

func (stack *Stack) Empty() bool {
	if err := stack.requireOpen(); err != nil {
		panic(err)
	}
	return len(stack.entries) == 0
}

func (stack *Stack) Top() (StackRef, error) {
	if err := stack.requireOpen(); err != nil {
		return StackRef{}, err
	}
	if len(stack.entries) == 0 {
		return StackRef{}, ErrStackEmpty
	}
	return stack.entries[len(stack.entries)-1].ref, nil
}

func (stack *Stack) Size(ref StackRef) (uint32, error) {
	entry, err := stack.allocation(ref)
	return entry.size, err
}

func (stack *Stack) Read(ref StackRef, index uint32) (byte, error) {
	if _, err := stack.allocation(ref); err != nil {
		return 0, err
	}
	return stack.region.Read(ref.region, index)
}

func (stack *Stack) Write(ref StackRef, index uint32, value byte) error {
	if _, err := stack.allocation(ref); err != nil {
		return err
	}
	return stack.region.Write(ref.region, index, value)
}

func (stack *Stack) Deallocate() error {
	if err := stack.requireOpen(); err != nil {
		return err
	}
	if len(stack.entries) == 0 {
		return ErrStackEmpty
	}
	last := len(stack.entries) - 1
	entry := stack.entries[last]
	if err := stack.region.Rollback(entry.mark); err != nil {
		return err
	}
	stack.entries[last] = stackEntry{}
	stack.entries = stack.entries[:last]
	return nil
}

// DeallocateRef implements checked placement delete and requires ref to be top.
func (stack *Stack) DeallocateRef(ref StackRef) error {
	top, err := stack.Top()
	if err != nil {
		return err
	}
	if ref.owner != stack || ref.serial != top.serial || ref.region != top.region {
		return ErrStackOwner
	}
	return stack.Deallocate()
}

// DeallocateValue invokes the caller's destructor before popping ref.
func (stack *Stack) DeallocateValue(ref StackRef, destroy func()) error {
	top, err := stack.Top()
	if err != nil {
		return err
	}
	if ref.owner != stack || ref.serial != top.serial || ref.region != top.region {
		return ErrStackOwner
	}
	if destroy == nil {
		return ErrStackOwner
	}
	destroy()
	return stack.Deallocate()
}

func (stack *Stack) Reset() error {
	if err := stack.requireOpen(); err != nil {
		return err
	}
	for len(stack.entries) != 0 {
		if err := stack.Deallocate(); err != nil {
			return err
		}
	}
	return nil
}

func (stack *Stack) Stats() (arena.BytesRegionStats, error) {
	if err := stack.requireOpen(); err != nil {
		return arena.BytesRegionStats{}, err
	}
	return stack.region.Stats(), nil
}

func (stack *Stack) Close() error {
	if stack == nil || stack.self != stack {
		return ErrStackOwner
	}
	if stack.closed {
		return nil
	}
	if err := stack.Reset(); err != nil {
		return err
	}
	if err := stack.region.Close(); err != nil {
		return err
	}
	stack.entries = nil
	stack.region = nil
	stack.closed = true
	return nil
}
