// Copyright (c) 2006, 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrSATSelfAppend = errors.New("sat: nonterminating source self append")

func satVectorNextCapacity(old, stride uint64) (uint64, error) {
	return satVectorNextCapacityFor(old, stride, stride)
}
func satVectorNextCapacityFor(old, stride, sourceStride uint64) (uint64, error) {
	n := uint64(2)
	if old != 0 {
		if old > uint64(^uint32(0)) {
			return 0, arena.ErrBounds
		}
		n = (3*old + 1) / 2
	}
	if n <= old || n > (uint64(^uint32(0))-8)/sourceStride || n > uint64(^uint(0)>>1)/stride {
		return 0, arena.ErrBounds
	}
	return n, nil
}

type satWatchList struct {
	self              *satWatchList
	data              []satWatched
	initialized       []bool
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
}

func NewSATWatchList(b *arena.Budget) (*satWatchList, error) { return newsatWatchList(b) }
func newsatWatchList(b *arena.Budget) (*satWatchList, error) {
	n := uint64(reflect.TypeOf(satWatchList{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &satWatchList{budget: b, ownerBytes: n, epoch: 1}
	v.self = v
	return v, nil
}
func (v *satWatchList) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *satWatchList) change() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (v *satWatchList) Size() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *satWatchList) Capacity() uint64 {
	if v == nil || v.closed {
		return 0
	}
	return uint64(cap(v.data))
}
func (v *satWatchList) Empty() bool { return v.Size() == 0 }
func (v *satWatchList) grow(need uint64) error {
	for need > uint64(cap(v.data)) {
		n, e := satVectorNextCapacityFor(uint64(cap(v.data)), uint64(reflect.TypeOf(satWatched{}).Size()), 8)
		if e != nil {
			return e
		}
		bytes := n * (uint64(reflect.TypeOf(satWatched{}).Size()) + uint64(reflect.TypeOf(false).Size()))
		if e = v.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satWatched, len(v.data), int(n))
		copy(next[:cap(v.data)], v.data[:cap(v.data)])
		life := make([]bool, int(n))
		copy(life, v.initialized)
		v.budget.Release(v.backingBytes())
		v.data = next
		v.initialized = life
		if v.epoch != ^uint64(0) {
			v.epoch++
		}
	}
	return nil
}
func (v *satWatchList) Get(i uint32) (satWatched, error) {
	var z satWatched
	if e := v.open(); e != nil {
		return z, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return z, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) || !v.initialized[i] {
		return z, arena.ErrBounds
	}
	return v.data[:cap(v.data)][i], nil
}
func (v *satWatchList) GetDefault(i uint32, d satWatched) (satWatched, error) {
	if e := v.open(); e != nil {
		return d, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return d, nil
	}
	return v.data[i], nil
}
func (v *satWatchList) Set(i uint32, x satWatched) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) || !v.initialized[i] {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satWatchList) SetX(i uint32, x, d satWatched) error {
	if e := v.change(); e != nil {
		return e
	}

	if uint64(i) >= uint64(len(v.data)) {
		if e := v.ResizeFill(i+1, d); e != nil {
			return e
		}
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satWatchList) Back() (satWatched, error) {
	if e := v.open(); e != nil {
		var z satWatched
		return z, e
	}
	if len(v.data) == 0 {
		var z satWatched
		if satStorageSourceDebug {
			return z, ErrSATAssertion
		}
		return z, arena.ErrBounds
	}
	return v.Get(uint32(len(v.data) - 1))
}
func (v *satWatchList) Push(x satWatched) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	v.initialized[len(v.data)-1] = true
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) Pop() (satWatched, error) {
	x, e := v.Back()
	if e != nil {
		return x, e
	}
	if e = v.change(); e != nil {
		return x, e
	}
	v.initialized[len(v.data)-1] = false
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satWatchList) ResizeFill(n uint32, x satWatched) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(n)); e != nil {
		return e
	}
	old := len(v.data)
	if int(n) < old {
		clear(v.initialized[int(n):old])
	}
	v.data = v.data[:int(n)]
	for i := old; i < int(n); i++ {
		v.data[i] = x
		v.initialized[i] = true
	}
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) ReserveFill(n uint32, x satWatched) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return v.ResizeFill(n, x)
	}
	return nil
}
func (v *satWatchList) Shrink(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(n) > uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if v.data == nil {
		return nil
	}
	if uint64(n) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	clear(v.initialized[int(n):len(v.data)])
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) SetEnd(n uint32) error { return v.Shrink(n) }
func (v *satWatchList) InsertAt(i uint32, x satWatched) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	v.initialized[len(v.data)-1] = true
	copy(v.data[int(i)+1:], v.data[int(i):len(v.data)-1])
	v.data[i] = x
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) Contains(x satWatched) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	for _, a := range v.data {
		b := x
		if a.Equal(b) {
			return true, nil
		}
	}
	return false, nil
}
func (v *satWatchList) Erase(x satWatched) (bool, error) {
	if e := v.change(); e != nil {
		return false, e
	}
	for i, a := range v.data {
		b := x
		if a.Equal(b) {
			copy(v.data[i:], v.data[i+1:])
			v.data = v.data[:len(v.data)-1]
			if v.epoch != ^uint64(0) {
				v.epoch++
			}
			return true, nil
		}
	}
	return false, nil
}
func (v *satWatchList) Append(src *satWatchList) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src && len(v.data) != 0 {
		return ErrSATSelfAppend
	}
	for _, x := range src.data {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satWatchList) AppendTo(dst []satWatched) ([]satWatched, error) {
	if e := v.open(); e != nil {
		return dst, e
	}
	return append(dst, v.data...), nil
}
func (v *satWatchList) Fill(x satWatched) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := range v.data {
		v.data[i] = x
	}
	return nil
}
func (v *satWatchList) Reverse() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, j := 0, len(v.data)-1; i < j; i, j = i+1, j-1 {
		v.data[i], v.data[j] = v.data[j], v.data[i]
	}
	return nil
}
func (v *satWatchList) Reset() error {
	if e := v.change(); e != nil {
		return e
	}
	clear(v.initialized[:len(v.data)])
	v.data = v.data[:0]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) Finalize() error {
	if e := v.change(); e != nil {
		return e
	}
	v.budget.Release(v.backingBytes())
	v.data = nil
	v.initialized = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satWatchList) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.budget.Release(v.ownerBytes + v.backingBytes())
	v.data = nil
	v.initialized = nil
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *satWatchList) copyCore(src *satWatchList) error {
	n := uint64(cap(src.data))
	bytes := n * (uint64(reflect.TypeOf(satWatched{}).Size()) + uint64(reflect.TypeOf(false).Size()))
	if n == 0 {
		return nil
	}
	if e := v.budget.Reserve(bytes); e != nil {
		return e
	}
	v.data = make([]satWatched, len(src.data), int(n))
	copy(v.data, src.data)
	v.initialized = make([]bool, int(n))
	for i := range src.data {
		v.initialized[i] = true
	}
	return nil
}
func (v *satWatchList) Clone(b *arena.Budget) (*satWatchList, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	x, e := newsatWatchList(b)
	if e != nil {
		return nil, e
	}
	if e = x.copyCore(v); e != nil {
		x.Close()
		return nil, e
	}
	return x, nil
}
func (v *satWatchList) Assign(src *satWatchList) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	if e := v.Finalize(); e != nil {
		return e
	}
	return v.copyCore(src)
}
func (v *satWatchList) MoveClone(b *arena.Budget) (*satWatchList, error) {
	if e := v.change(); e != nil {
		return nil, e
	}
	x, e := newsatWatchList(b)
	if e != nil {
		return nil, e
	}
	n := v.backingBytes()
	if b != v.budget {
		if e = b.Reserve(n); e != nil {
			x.Close()
			return nil, e
		}
		v.budget.Release(n)
	}
	x.data = v.data
	x.initialized = v.initialized
	v.data = nil
	v.initialized = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satWatchList) MoveAssign(src *satWatchList) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.change(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	n := src.backingBytes()
	if v.budget != src.budget {
		if e := v.budget.Reserve(n); e != nil {
			return e
		}
	}
	if e := v.Finalize(); e != nil {
		if v.budget != src.budget {
			v.budget.Release(n)
		}
		return e
	}
	if v.budget != src.budget {
		src.budget.Release(n)
	}
	v.data = src.data
	v.initialized = src.initialized
	src.data = nil
	src.initialized = nil
	if src.epoch != ^uint64(0) {
		src.epoch++
	}
	return nil
}

type satWatchListIterator struct {
	owner   *satWatchList
	epoch   uint64
	index   int
	reverse bool
}

func (v *satWatchList) Iter() satWatchListIterator {
	return satWatchListIterator{owner: v, epoch: v.epoch}
}
func (v *satWatchList) ReverseIter() satWatchListIterator {
	return satWatchListIterator{owner: v, epoch: v.epoch, index: len(v.data) - 1, reverse: true}
}
func (i *satWatchListIterator) Next() (satWatched, bool, error) {
	var z satWatched
	if e := i.owner.open(); e != nil {
		return z, false, e
	}
	if i.epoch != i.owner.epoch {
		return z, false, arena.ErrInvalidSlot
	}
	if i.index < 0 || i.index >= len(i.owner.data) {
		return z, false, nil
	}
	x := i.owner.data[i.index]
	if i.reverse {
		i.index--
	} else {
		i.index++
	}
	return x, true, nil
}

type satClauseVector struct {
	self              *satClauseVector
	data              []satStoredClauseRef
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
	allocator         *satClauseAllocator
	bindings          []satAllocatorBinding
}

func NewSATClauseVector(a *satClauseAllocator, b *arena.Budget) (*satClauseVector, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	return newsatClauseVector(a, b)
}
func newsatClauseVector(a *satClauseAllocator, b *arena.Budget) (*satClauseVector, error) {
	n := uint64(reflect.TypeOf(satClauseVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &satClauseVector{allocator: a, budget: b, ownerBytes: n, epoch: 1}
	v.self = v
	return v, nil
}
func (v *satClauseVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *satClauseVector) change() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (v *satClauseVector) Size() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *satClauseVector) Capacity() uint64 {
	if v == nil || v.closed {
		return 0
	}
	return uint64(cap(v.data))
}
func (v *satClauseVector) Empty() bool { return v.Size() == 0 }
func (v *satClauseVector) grow(need uint64) error {
	for need > uint64(cap(v.data)) {
		n, e := satVectorNextCapacityFor(uint64(cap(v.data)), uint64(reflect.TypeOf(satStoredClauseRef{}).Size()), 8)
		if e != nil {
			return e
		}
		bytes := n * uint64(reflect.TypeOf(satStoredClauseRef{}).Size())
		if e = v.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satStoredClauseRef, len(v.data), int(n))
		// Source trivial reallocate preserves the entire physical backing,
		// including initialized scalar values retained after shrinking.
		copy(next[:cap(v.data)], v.data[:cap(v.data)])
		v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
		v.data = next
		if v.epoch != ^uint64(0) {
			v.epoch++
		}
	}
	return nil
}
func (v *satClauseVector) Get(i uint32) (satStoredClauseRef, error) {
	var z satStoredClauseRef
	if e := v.open(); e != nil {
		return z, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return z, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return z, arena.ErrBounds
	}
	return v.data[:cap(v.data)][i], nil
}
func (v *satClauseVector) GetDefault(i uint32, d satStoredClauseRef) (satStoredClauseRef, error) {
	if e := v.open(); e != nil {
		return d, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return d, nil
	}
	return v.data[i], nil
}
func (v *satClauseVector) Set(i uint32, x satStoredClauseRef) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satClauseVector) SetX(i uint32, x, d satStoredClauseRef) error {
	if e := v.change(); e != nil {
		return e
	}

	if uint64(i) >= uint64(len(v.data)) {
		if e := v.ResizeFill(i+1, d); e != nil {
			return e
		}
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satClauseVector) Back() (satStoredClauseRef, error) {
	if e := v.open(); e != nil {
		var z satStoredClauseRef
		return z, e
	}
	if len(v.data) == 0 {
		var z satStoredClauseRef
		if satStorageSourceDebug {
			return z, ErrSATAssertion
		}
		return z, arena.ErrBounds
	}
	return v.Get(uint32(len(v.data) - 1))
}
func (v *satClauseVector) Push(x satStoredClauseRef) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) Pop() (satStoredClauseRef, error) {
	x, e := v.Back()
	if e != nil {
		return x, e
	}
	if e = v.change(); e != nil {
		return x, e
	}
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satClauseVector) ResizeFill(n uint32, x satStoredClauseRef) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(n)); e != nil {
		return e
	}
	old := len(v.data)
	v.data = v.data[:int(n)]
	for i := old; i < int(n); i++ {
		v.data[i] = x
	}
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) ReserveFill(n uint32, x satStoredClauseRef) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return v.ResizeFill(n, x)
	}
	return nil
}
func (v *satClauseVector) Shrink(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(n) > uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if len(v.data) == 0 && cap(v.data) == 0 {
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) SetEnd(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	// Source set_end has no size assertion for existing scalar backing.
	// Unlike shrink, it can expose retained initialized elements in debug.
	if cap(v.data) == 0 {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) InsertAt(i uint32, x satStoredClauseRef) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	copy(v.data[int(i)+1:], v.data[int(i):len(v.data)-1])
	v.data[i] = x
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) Contains(x satStoredClauseRef) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	for _, a := range v.data {
		b := x
		if a == b {
			return true, nil
		}
	}
	return false, nil
}
func (v *satClauseVector) Erase(x satStoredClauseRef) (bool, error) {
	if e := v.change(); e != nil {
		return false, e
	}
	for i, a := range v.data {
		b := x
		if a == b {
			copy(v.data[i:], v.data[i+1:])
			v.data = v.data[:len(v.data)-1]
			if v.epoch != ^uint64(0) {
				v.epoch++
			}
			return true, nil
		}
	}
	return false, nil
}
func (v *satClauseVector) Append(src *satClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src && len(v.data) != 0 {
		return ErrSATSelfAppend
	}
	for _, x := range src.data {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseVector) AppendTo(dst []satStoredClauseRef) ([]satStoredClauseRef, error) {
	if e := v.open(); e != nil {
		return dst, e
	}
	return append(dst, v.data...), nil
}
func (v *satClauseVector) Fill(x satStoredClauseRef) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := range v.data {
		v.data[i] = x
	}
	return nil
}
func (v *satClauseVector) Reverse() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, j := 0, len(v.data)-1; i < j; i, j = i+1, j-1 {
		v.data[i], v.data[j] = v.data[j], v.data[i]
	}
	return nil
}
func (v *satClauseVector) Reset() error {
	if e := v.change(); e != nil {
		return e
	}
	v.data = v.data[:0]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) Finalize() error {
	if e := v.change(); e != nil {
		return e
	}
	v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.budget.Release(v.ownerBytes + uint64(cap(v.data))*uint64(reflect.TypeOf(satStoredClauseRef{}).Size()))
	v.budget.Release(uint64(cap(v.bindings)) * uint64(reflect.TypeOf(satAllocatorBinding{}).Size()))
	v.bindings = nil
	v.data = nil
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *satClauseVector) copyCore(src *satClauseVector) error {
	if e := v.adoptBindings(src); e != nil {
		return e
	}
	n := uint64(cap(src.data))
	bytes := n * uint64(reflect.TypeOf(satStoredClauseRef{}).Size())
	if n == 0 {
		return nil
	}
	if e := v.budget.Reserve(bytes); e != nil {
		return e
	}
	v.data = make([]satStoredClauseRef, len(src.data), int(n))
	copy(v.data, src.data)
	return nil
}
func (v *satClauseVector) Clone(b *arena.Budget) (*satClauseVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	x, e := newsatClauseVector(v.allocator, b)
	if e != nil {
		return nil, e
	}
	if e = x.copyCore(v); e != nil {
		x.Close()
		return nil, e
	}
	return x, nil
}
func (v *satClauseVector) Assign(src *satClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	if e := v.Finalize(); e != nil {
		return e
	}
	return v.copyCore(src)
}
func (v *satClauseVector) MoveClone(b *arena.Budget) (*satClauseVector, error) {
	if e := v.change(); e != nil {
		return nil, e
	}
	x, e := newsatClauseVector(v.allocator, b)
	if e != nil {
		return nil, e
	}
	if e = x.adoptBindings(v); e != nil {
		x.Close()
		return nil, e
	}
	n := uint64(cap(v.data)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size())
	if b != v.budget {
		if e = b.Reserve(n); e != nil {
			x.Close()
			return nil, e
		}
		v.budget.Release(n)
	}
	x.data = v.data
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satClauseVector) MoveAssign(src *satClauseVector) error { return v.Assign(src) }

type satClauseVectorIterator struct {
	owner   *satClauseVector
	epoch   uint64
	index   int
	reverse bool
}

func (v *satClauseVector) Iter() satClauseVectorIterator {
	return satClauseVectorIterator{owner: v, epoch: v.epoch}
}
func (v *satClauseVector) ReverseIter() satClauseVectorIterator {
	return satClauseVectorIterator{owner: v, epoch: v.epoch, index: len(v.data) - 1, reverse: true}
}
func (i *satClauseVectorIterator) Next() (satStoredClauseRef, bool, error) {
	var z satStoredClauseRef
	if e := i.owner.open(); e != nil {
		return z, false, e
	}
	if i.epoch != i.owner.epoch {
		return z, false, arena.ErrInvalidSlot
	}
	if i.index < 0 || i.index >= len(i.owner.data) {
		return z, false, nil
	}
	x := i.owner.data[i.index]
	if i.reverse {
		i.index--
	} else {
		i.index++
	}
	return x, true, nil
}

type satClauseWrapperVector struct {
	self              *satClauseWrapperVector
	data              []satClauseWrapper
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
	allocator         *satClauseAllocator
	bindings          []satAllocatorBinding
}

func NewSATClauseWrapperVector(a *satClauseAllocator, b *arena.Budget) (*satClauseWrapperVector, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	return newsatClauseWrapperVector(a, b)
}
func newsatClauseWrapperVector(a *satClauseAllocator, b *arena.Budget) (*satClauseWrapperVector, error) {
	n := uint64(reflect.TypeOf(satClauseWrapperVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &satClauseWrapperVector{allocator: a, budget: b, ownerBytes: n, epoch: 1}
	v.self = v
	return v, nil
}
func (v *satClauseWrapperVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *satClauseWrapperVector) change() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (v *satClauseWrapperVector) Size() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *satClauseWrapperVector) Capacity() uint64 {
	if v == nil || v.closed {
		return 0
	}
	return uint64(cap(v.data))
}
func (v *satClauseWrapperVector) Empty() bool { return v.Size() == 0 }
func (v *satClauseWrapperVector) grow(need uint64) error {
	for need > uint64(cap(v.data)) {
		n, e := satVectorNextCapacityFor(uint64(cap(v.data)), uint64(reflect.TypeOf(satClauseWrapper{}).Size()), 16)
		if e != nil {
			return e
		}
		bytes := n * uint64(reflect.TypeOf(satClauseWrapper{}).Size())
		if e = v.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satClauseWrapper, len(v.data), int(n))
		// Source trivial reallocate preserves the entire physical backing,
		// including initialized scalar values retained after shrinking.
		copy(next[:cap(v.data)], v.data[:cap(v.data)])
		v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
		v.data = next
		if v.epoch != ^uint64(0) {
			v.epoch++
		}
	}
	return nil
}
func (v *satClauseWrapperVector) Get(i uint32) (satClauseWrapper, error) {
	var z satClauseWrapper
	if e := v.open(); e != nil {
		return z, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return z, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return z, arena.ErrBounds
	}
	return v.data[:cap(v.data)][i], nil
}
func (v *satClauseWrapperVector) GetDefault(i uint32, d satClauseWrapper) (satClauseWrapper, error) {
	if e := v.open(); e != nil {
		return d, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return d, nil
	}
	return v.data[i], nil
}
func (v *satClauseWrapperVector) Set(i uint32, x satClauseWrapper) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satClauseWrapperVector) SetX(i uint32, x, d satClauseWrapper) error {
	if e := v.change(); e != nil {
		return e
	}

	if uint64(i) >= uint64(len(v.data)) {
		if e := v.ResizeFill(i+1, d); e != nil {
			return e
		}
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satClauseWrapperVector) Back() (satClauseWrapper, error) {
	if e := v.open(); e != nil {
		var z satClauseWrapper
		return z, e
	}
	if len(v.data) == 0 {
		var z satClauseWrapper
		if satStorageSourceDebug {
			return z, ErrSATAssertion
		}
		return z, arena.ErrBounds
	}
	return v.Get(uint32(len(v.data) - 1))
}
func (v *satClauseWrapperVector) Push(x satClauseWrapper) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) Pop() (satClauseWrapper, error) {
	x, e := v.Back()
	if e != nil {
		return x, e
	}
	if e = v.change(); e != nil {
		return x, e
	}
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satClauseWrapperVector) ResizeFill(n uint32, x satClauseWrapper) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(n)); e != nil {
		return e
	}
	old := len(v.data)
	v.data = v.data[:int(n)]
	for i := old; i < int(n); i++ {
		v.data[i] = x
	}
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) ReserveFill(n uint32, x satClauseWrapper) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return v.ResizeFill(n, x)
	}
	return nil
}
func (v *satClauseWrapperVector) Shrink(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(n) > uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if len(v.data) == 0 && cap(v.data) == 0 {
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) SetEnd(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	// Source set_end has no size assertion for existing scalar backing.
	// Unlike shrink, it can expose retained initialized elements in debug.
	if cap(v.data) == 0 {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) InsertAt(i uint32, x satClauseWrapper) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	copy(v.data[int(i)+1:], v.data[int(i):len(v.data)-1])
	v.data[i] = x
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) Append(src *satClauseWrapperVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src && len(v.data) != 0 {
		return ErrSATSelfAppend
	}
	for _, x := range src.data {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseWrapperVector) AppendTo(dst []satClauseWrapper) ([]satClauseWrapper, error) {
	if e := v.open(); e != nil {
		return dst, e
	}
	return append(dst, v.data...), nil
}
func (v *satClauseWrapperVector) Fill(x satClauseWrapper) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := range v.data {
		v.data[i] = x
	}
	return nil
}
func (v *satClauseWrapperVector) Reverse() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, j := 0, len(v.data)-1; i < j; i, j = i+1, j-1 {
		v.data[i], v.data[j] = v.data[j], v.data[i]
	}
	return nil
}
func (v *satClauseWrapperVector) Reset() error {
	if e := v.change(); e != nil {
		return e
	}
	v.data = v.data[:0]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) Finalize() error {
	if e := v.change(); e != nil {
		return e
	}
	v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.budget.Release(v.ownerBytes + uint64(cap(v.data))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()))
	v.budget.Release(uint64(cap(v.bindings)) * uint64(reflect.TypeOf(satAllocatorBinding{}).Size()))
	v.bindings = nil
	v.data = nil
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *satClauseWrapperVector) copyCore(src *satClauseWrapperVector) error {
	if e := v.adoptBindings(src); e != nil {
		return e
	}
	n := uint64(cap(src.data))
	bytes := n * uint64(reflect.TypeOf(satClauseWrapper{}).Size())
	if n == 0 {
		return nil
	}
	if e := v.budget.Reserve(bytes); e != nil {
		return e
	}
	v.data = make([]satClauseWrapper, len(src.data), int(n))
	copy(v.data, src.data)
	return nil
}
func (v *satClauseWrapperVector) Clone(b *arena.Budget) (*satClauseWrapperVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	x, e := newsatClauseWrapperVector(v.allocator, b)
	if e != nil {
		return nil, e
	}
	if e = x.copyCore(v); e != nil {
		x.Close()
		return nil, e
	}
	return x, nil
}
func (v *satClauseWrapperVector) Assign(src *satClauseWrapperVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	if e := v.Finalize(); e != nil {
		return e
	}
	return v.copyCore(src)
}
func (v *satClauseWrapperVector) MoveClone(b *arena.Budget) (*satClauseWrapperVector, error) {
	if e := v.change(); e != nil {
		return nil, e
	}
	x, e := newsatClauseWrapperVector(v.allocator, b)
	if e != nil {
		return nil, e
	}
	if e = x.adoptBindings(v); e != nil {
		x.Close()
		return nil, e
	}
	n := uint64(cap(v.data)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size())
	if b != v.budget {
		if e = b.Reserve(n); e != nil {
			x.Close()
			return nil, e
		}
		v.budget.Release(n)
	}
	x.data = v.data
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satClauseWrapperVector) MoveAssign(src *satClauseWrapperVector) error { return v.Assign(src) }

type satClauseWrapperVectorIterator struct {
	owner   *satClauseWrapperVector
	epoch   uint64
	index   int
	reverse bool
}

func (v *satClauseWrapperVector) Iter() satClauseWrapperVectorIterator {
	return satClauseWrapperVectorIterator{owner: v, epoch: v.epoch}
}
func (v *satClauseWrapperVector) ReverseIter() satClauseWrapperVectorIterator {
	return satClauseWrapperVectorIterator{owner: v, epoch: v.epoch, index: len(v.data) - 1, reverse: true}
}
func (i *satClauseWrapperVectorIterator) Next() (satClauseWrapper, bool, error) {
	var z satClauseWrapper
	if e := i.owner.open(); e != nil {
		return z, false, e
	}
	if i.epoch != i.owner.epoch {
		return z, false, arena.ErrInvalidSlot
	}
	if i.index < 0 || i.index >= len(i.owner.data) {
		return z, false, nil
	}
	x := i.owner.data[i.index]
	if i.reverse {
		i.index--
	} else {
		i.index++
	}
	return x, true, nil
}

type satActivityDomain struct {
	self              *satActivityDomain
	data              []uint32
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
}

func NewSATActivityDomain(b *arena.Budget) (*satActivityDomain, error) {
	return newsatActivityDomain(b)
}
func newsatActivityDomain(b *arena.Budget) (*satActivityDomain, error) {
	n := uint64(reflect.TypeOf(satActivityDomain{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &satActivityDomain{budget: b, ownerBytes: n, epoch: 1}
	v.self = v
	return v, nil
}
func (v *satActivityDomain) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *satActivityDomain) change() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (v *satActivityDomain) Size() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *satActivityDomain) Capacity() uint64 {
	if v == nil || v.closed {
		return 0
	}
	return uint64(cap(v.data))
}
func (v *satActivityDomain) Empty() bool { return v.Size() == 0 }
func (v *satActivityDomain) grow(need uint64) error {
	for need > uint64(cap(v.data)) {
		n, e := satVectorNextCapacityFor(uint64(cap(v.data)), uint64(reflect.TypeOf(uint32(0)).Size()), 4)
		if e != nil {
			return e
		}
		bytes := n * uint64(reflect.TypeOf(uint32(0)).Size())
		if e = v.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]uint32, len(v.data), int(n))
		// Source trivial reallocate preserves the entire physical backing,
		// including initialized scalar values retained after shrinking.
		copy(next[:cap(v.data)], v.data[:cap(v.data)])
		v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(uint32(0)).Size()))
		v.data = next
		if v.epoch != ^uint64(0) {
			v.epoch++
		}
	}
	return nil
}
func (v *satActivityDomain) Get(i uint32) (uint32, error) {
	var z uint32
	if e := v.open(); e != nil {
		return z, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return z, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return z, arena.ErrBounds
	}
	return v.data[:cap(v.data)][i], nil
}
func (v *satActivityDomain) GetDefault(i uint32, d uint32) (uint32, error) {
	if e := v.open(); e != nil {
		return d, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return d, nil
	}
	return v.data[i], nil
}
func (v *satActivityDomain) Set(i uint32, x uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satActivityDomain) SetX(i uint32, x, d uint32) error {
	if e := v.change(); e != nil {
		return e
	}

	if uint64(i) >= uint64(len(v.data)) {
		if e := v.ResizeFill(i+1, d); e != nil {
			return e
		}
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satActivityDomain) Back() (uint32, error) {
	if e := v.open(); e != nil {
		var z uint32
		return z, e
	}
	if len(v.data) == 0 {
		var z uint32
		if satStorageSourceDebug {
			return z, ErrSATAssertion
		}
		return z, arena.ErrBounds
	}
	return v.Get(uint32(len(v.data) - 1))
}
func (v *satActivityDomain) Push(x uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) Pop() (uint32, error) {
	x, e := v.Back()
	if e != nil {
		return x, e
	}
	if e = v.change(); e != nil {
		return x, e
	}
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satActivityDomain) ResizeFill(n uint32, x uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(n)); e != nil {
		return e
	}
	old := len(v.data)
	v.data = v.data[:int(n)]
	for i := old; i < int(n); i++ {
		v.data[i] = x
	}
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) ReserveFill(n uint32, x uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return v.ResizeFill(n, x)
	}
	return nil
}
func (v *satActivityDomain) Shrink(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(n) > uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if len(v.data) == 0 && cap(v.data) == 0 {
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) SetEnd(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	// Source set_end has no size assertion for existing scalar backing.
	// Unlike shrink, it can expose retained initialized elements in debug.
	if cap(v.data) == 0 {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) InsertAt(i uint32, x uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	copy(v.data[int(i)+1:], v.data[int(i):len(v.data)-1])
	v.data[i] = x
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) Contains(x uint32) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	for _, a := range v.data {
		b := x
		if a == b {
			return true, nil
		}
	}
	return false, nil
}
func (v *satActivityDomain) Erase(x uint32) (bool, error) {
	if e := v.change(); e != nil {
		return false, e
	}
	for i, a := range v.data {
		b := x
		if a == b {
			copy(v.data[i:], v.data[i+1:])
			v.data = v.data[:len(v.data)-1]
			if v.epoch != ^uint64(0) {
				v.epoch++
			}
			return true, nil
		}
	}
	return false, nil
}
func (v *satActivityDomain) Append(src *satActivityDomain) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src && len(v.data) != 0 {
		return ErrSATSelfAppend
	}
	for _, x := range src.data {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satActivityDomain) AppendTo(dst []uint32) ([]uint32, error) {
	if e := v.open(); e != nil {
		return dst, e
	}
	return append(dst, v.data...), nil
}
func (v *satActivityDomain) Fill(x uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := range v.data {
		v.data[i] = x
	}
	return nil
}
func (v *satActivityDomain) Reverse() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, j := 0, len(v.data)-1; i < j; i, j = i+1, j-1 {
		v.data[i], v.data[j] = v.data[j], v.data[i]
	}
	return nil
}
func (v *satActivityDomain) Reset() error {
	if e := v.change(); e != nil {
		return e
	}
	v.data = v.data[:0]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) Finalize() error {
	if e := v.change(); e != nil {
		return e
	}
	v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(uint32(0)).Size()))
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satActivityDomain) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.budget.Release(v.ownerBytes + uint64(cap(v.data))*uint64(reflect.TypeOf(uint32(0)).Size()))
	v.data = nil
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *satActivityDomain) copyCore(src *satActivityDomain) error {
	n := uint64(cap(src.data))
	bytes := n * uint64(reflect.TypeOf(uint32(0)).Size())
	if n == 0 {
		return nil
	}
	if e := v.budget.Reserve(bytes); e != nil {
		return e
	}
	v.data = make([]uint32, len(src.data), int(n))
	copy(v.data, src.data)
	return nil
}
func (v *satActivityDomain) Clone(b *arena.Budget) (*satActivityDomain, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	x, e := newsatActivityDomain(b)
	if e != nil {
		return nil, e
	}
	if e = x.copyCore(v); e != nil {
		x.Close()
		return nil, e
	}
	return x, nil
}
func (v *satActivityDomain) Assign(src *satActivityDomain) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	if e := v.Finalize(); e != nil {
		return e
	}
	return v.copyCore(src)
}
func (v *satActivityDomain) MoveClone(b *arena.Budget) (*satActivityDomain, error) {
	if e := v.change(); e != nil {
		return nil, e
	}
	x, e := newsatActivityDomain(b)
	if e != nil {
		return nil, e
	}
	n := uint64(cap(v.data)) * uint64(reflect.TypeOf(uint32(0)).Size())
	if b != v.budget {
		if e = b.Reserve(n); e != nil {
			x.Close()
			return nil, e
		}
		v.budget.Release(n)
	}
	x.data = v.data
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satActivityDomain) MoveAssign(src *satActivityDomain) error { return v.Assign(src) }

type satActivityDomainIterator struct {
	owner   *satActivityDomain
	epoch   uint64
	index   int
	reverse bool
}

func (v *satActivityDomain) Iter() satActivityDomainIterator {
	return satActivityDomainIterator{owner: v, epoch: v.epoch}
}
func (v *satActivityDomain) ReverseIter() satActivityDomainIterator {
	return satActivityDomainIterator{owner: v, epoch: v.epoch, index: len(v.data) - 1, reverse: true}
}
func (i *satActivityDomainIterator) Next() (uint32, bool, error) {
	var z uint32
	if e := i.owner.open(); e != nil {
		return z, false, e
	}
	if i.epoch != i.owner.epoch {
		return z, false, arena.ErrInvalidSlot
	}
	if i.index < 0 || i.index >= len(i.owner.data) {
		return z, false, nil
	}
	x := i.owner.data[i.index]
	if i.reverse {
		i.index--
	} else {
		i.index++
	}
	return x, true, nil
}

type satBinClauseVector struct {
	self              *satBinClauseVector
	data              []satBinClause
	budget            *arena.Budget
	ownerBytes, epoch uint64
	closed            bool
}

func NewSATBinClauseVector(b *arena.Budget) (*satBinClauseVector, error) {
	return newsatBinClauseVector(b)
}
func newsatBinClauseVector(b *arena.Budget) (*satBinClauseVector, error) {
	n := uint64(reflect.TypeOf(satBinClauseVector{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	v := &satBinClauseVector{budget: b, ownerBytes: n, epoch: 1}
	v.self = v
	return v, nil
}
func (v *satBinClauseVector) open() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *satBinClauseVector) change() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.epoch == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	return nil
}
func (v *satBinClauseVector) Size() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.data))
}
func (v *satBinClauseVector) Capacity() uint64 {
	if v == nil || v.closed {
		return 0
	}
	return uint64(cap(v.data))
}
func (v *satBinClauseVector) Empty() bool { return v.Size() == 0 }
func (v *satBinClauseVector) grow(need uint64) error {
	for need > uint64(cap(v.data)) {
		n, e := satVectorNextCapacityFor(uint64(cap(v.data)), uint64(reflect.TypeOf(satBinClause{}).Size()), 8)
		if e != nil {
			return e
		}
		bytes := n * uint64(reflect.TypeOf(satBinClause{}).Size())
		if e = v.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satBinClause, len(v.data), int(n))
		// Source trivial reallocate preserves the entire physical backing,
		// including initialized scalar values retained after shrinking.
		copy(next[:cap(v.data)], v.data[:cap(v.data)])
		v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satBinClause{}).Size()))
		v.data = next
		if v.epoch != ^uint64(0) {
			v.epoch++
		}
	}
	return nil
}
func (v *satBinClauseVector) Get(i uint32) (satBinClause, error) {
	var z satBinClause
	if e := v.open(); e != nil {
		return z, e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return z, ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return z, arena.ErrBounds
	}
	return v.data[:cap(v.data)][i], nil
}
func (v *satBinClauseVector) GetDefault(i uint32, d satBinClause) (satBinClause, error) {
	if e := v.open(); e != nil {
		return d, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return d, nil
	}
	return v.data[i], nil
}
func (v *satBinClauseVector) Set(i uint32, x satBinClause) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(i) >= uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satBinClauseVector) SetX(i uint32, x, d satBinClause) error {
	if e := v.change(); e != nil {
		return e
	}

	if uint64(i) >= uint64(len(v.data)) {
		if e := v.ResizeFill(i+1, d); e != nil {
			return e
		}
	}
	if uint64(i) >= uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data[:cap(v.data)][i] = x
	return nil
}
func (v *satBinClauseVector) Back() (satBinClause, error) {
	if e := v.open(); e != nil {
		var z satBinClause
		return z, e
	}
	if len(v.data) == 0 {
		var z satBinClause
		if satStorageSourceDebug {
			return z, ErrSATAssertion
		}
		return z, arena.ErrBounds
	}
	return v.Get(uint32(len(v.data) - 1))
}
func (v *satBinClauseVector) Push(x satBinClause) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) Pop() (satBinClause, error) {
	x, e := v.Back()
	if e != nil {
		return x, e
	}
	if e = v.change(); e != nil {
		return x, e
	}
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satBinClauseVector) ResizeFill(n uint32, x satBinClause) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := v.grow(uint64(n)); e != nil {
		return e
	}
	old := len(v.data)
	v.data = v.data[:int(n)]
	for i := old; i < int(n); i++ {
		v.data[i] = x
	}
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) ReserveFill(n uint32, x satBinClause) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return v.ResizeFill(n, x)
	}
	return nil
}
func (v *satBinClauseVector) Shrink(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if satStorageSourceDebug && uint64(n) > uint64(len(v.data)) {
		return ErrSATAssertion
	}
	if len(v.data) == 0 && cap(v.data) == 0 {
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) SetEnd(n uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	// Source set_end has no size assertion for existing scalar backing.
	// Unlike shrink, it can expose retained initialized elements in debug.
	if cap(v.data) == 0 {
		if satStorageSourceDebug && n != 0 {
			return ErrSATAssertion
		}
		return nil
	}
	if uint64(n) > uint64(cap(v.data)) {
		return arena.ErrBounds
	}
	v.data = v.data[:int(n)]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) InsertAt(i uint32, x satBinClause) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	if e := v.grow(uint64(len(v.data)) + 1); e != nil {
		return e
	}
	v.data = append(v.data, x)
	copy(v.data[int(i)+1:], v.data[int(i):len(v.data)-1])
	v.data[i] = x
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) Contains(x satBinClause) (bool, error) {
	if e := v.open(); e != nil {
		return false, e
	}
	for _, a := range v.data {
		b := x
		if a.Equal(b) {
			return true, nil
		}
	}
	return false, nil
}
func (v *satBinClauseVector) Erase(x satBinClause) (bool, error) {
	if e := v.change(); e != nil {
		return false, e
	}
	for i, a := range v.data {
		b := x
		if a.Equal(b) {
			copy(v.data[i:], v.data[i+1:])
			v.data = v.data[:len(v.data)-1]
			if v.epoch != ^uint64(0) {
				v.epoch++
			}
			return true, nil
		}
	}
	return false, nil
}
func (v *satBinClauseVector) Append(src *satBinClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src && len(v.data) != 0 {
		return ErrSATSelfAppend
	}
	for _, x := range src.data {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satBinClauseVector) AppendTo(dst []satBinClause) ([]satBinClause, error) {
	if e := v.open(); e != nil {
		return dst, e
	}
	return append(dst, v.data...), nil
}
func (v *satBinClauseVector) Fill(x satBinClause) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := range v.data {
		v.data[i] = x
	}
	return nil
}
func (v *satBinClauseVector) Reverse() error {
	if e := v.open(); e != nil {
		return e
	}
	for i, j := 0, len(v.data)-1; i < j; i, j = i+1, j-1 {
		v.data[i], v.data[j] = v.data[j], v.data[i]
	}
	return nil
}
func (v *satBinClauseVector) Reset() error {
	if e := v.change(); e != nil {
		return e
	}
	v.data = v.data[:0]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) Finalize() error {
	if e := v.change(); e != nil {
		return e
	}
	v.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(satBinClause{}).Size()))
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}
func (v *satBinClauseVector) Close() error {
	if v == nil || v.self != v {
		return ErrSATOwner
	}
	if v.closed {
		return nil
	}
	v.budget.Release(v.ownerBytes + uint64(cap(v.data))*uint64(reflect.TypeOf(satBinClause{}).Size()))
	v.data = nil
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (v *satBinClauseVector) copyCore(src *satBinClauseVector) error {
	n := uint64(cap(src.data))
	bytes := n * uint64(reflect.TypeOf(satBinClause{}).Size())
	if n == 0 {
		return nil
	}
	if e := v.budget.Reserve(bytes); e != nil {
		return e
	}
	v.data = make([]satBinClause, len(src.data), int(n))
	copy(v.data, src.data)
	return nil
}
func (v *satBinClauseVector) Clone(b *arena.Budget) (*satBinClauseVector, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	x, e := newsatBinClauseVector(b)
	if e != nil {
		return nil, e
	}
	if e = x.copyCore(v); e != nil {
		x.Close()
		return nil, e
	}
	return x, nil
}
func (v *satBinClauseVector) Assign(src *satBinClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if v == src {
		return nil
	}
	if e := v.Finalize(); e != nil {
		return e
	}
	return v.copyCore(src)
}
func (v *satBinClauseVector) MoveClone(b *arena.Budget) (*satBinClauseVector, error) {
	if e := v.change(); e != nil {
		return nil, e
	}
	x, e := newsatBinClauseVector(b)
	if e != nil {
		return nil, e
	}
	n := uint64(cap(v.data)) * uint64(reflect.TypeOf(satBinClause{}).Size())
	if b != v.budget {
		if e = b.Reserve(n); e != nil {
			x.Close()
			return nil, e
		}
		v.budget.Release(n)
	}
	x.data = v.data
	v.data = nil
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return x, nil
}
func (v *satBinClauseVector) MoveAssign(src *satBinClauseVector) error { return v.Assign(src) }

type satBinClauseVectorIterator struct {
	owner   *satBinClauseVector
	epoch   uint64
	index   int
	reverse bool
}

func (v *satBinClauseVector) Iter() satBinClauseVectorIterator {
	return satBinClauseVectorIterator{owner: v, epoch: v.epoch}
}
func (v *satBinClauseVector) ReverseIter() satBinClauseVectorIterator {
	return satBinClauseVectorIterator{owner: v, epoch: v.epoch, index: len(v.data) - 1, reverse: true}
}
func (i *satBinClauseVectorIterator) Next() (satBinClause, bool, error) {
	var z satBinClause
	if e := i.owner.open(); e != nil {
		return z, false, e
	}
	if i.epoch != i.owner.epoch {
		return z, false, arena.ErrInvalidSlot
	}
	if i.index < 0 || i.index >= len(i.owner.data) {
		return z, false, nil
	}
	x := i.owner.data[i.index]
	if i.reverse {
		i.index--
	} else {
		i.index++
	}
	return x, true, nil
}
func (v *satActivityDomain) Len() uint32           { return v.Size() }
func (v *satActivityDomain) Resize(n uint32) error { return v.ResizeFill(n, 0) }
func (v *satClauseVector) Resize(n uint32) error   { return v.ResizeFill(n, satStoredClauseRef{}) }
func (v *satClauseVector) DisplayInto(out *NumericTextStream) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, r := range v.data {
		a, e := v.resolve(r.owner)
		if e != nil {
			return e
		}
		if e := a.DisplayInto(r, out); e != nil {
			return e
		}
		if e := out.AppendCString("\n"); e != nil {
			return e
		}
	}
	return nil
}
func (v *satWatchList) EraseClauseWatch(o satClauseOffset) (bool, error) {
	if e := v.change(); e != nil {
		return false, e
	}
	for i, w := range v.data {
		if w.Clause() {
			got, e := w.ClauseOffset()
			if e != nil {
				return false, e
			}
			if got == o {
				copy(v.data[i:], v.data[i+1:])
				v.data = v.data[:len(v.data)-1]
				if v.epoch != ^uint64(0) {
					v.epoch++
				}
				return true, nil
			}
		}
	}
	return false, nil
}
func (v *satWatchList) EraseTernaryWatch(a, b satLiteral) error {
	w, e := newSATTernaryWatch(a, b)
	if e != nil {
		return e
	}
	_, e = v.Erase(w)
	return e
}
func (v *satWatchList) SortByKind() error {
	if e := v.open(); e != nil {
		return e
	}
	for i := 1; i < len(v.data); i++ {
		x := v.data[i]
		j := i
		for j > 0 && satWatchedLess(x, v.data[j-1]) {
			v.data[j] = v.data[j-1]
			j--
		}
		v.data[j] = x
	}
	return nil
}
func (v *satWatchList) DisplayInto(a *satClauseAllocator, s *NumericTextStream) error {
	if e := v.open(); e != nil {
		return e
	}
	for i, w := range v.data {
		if i != 0 {
			if e := s.AppendCString(" "); e != nil {
				return e
			}
		}
		switch w.Kind() {
		case 0:
			l, e := w.Literal()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if w.LearnedUnchecked() {
				if e = s.AppendCString("*"); e != nil {
					return e
				}
			}
		case 1:
			if e := s.AppendCString("("); e != nil {
				return e
			}
			l, e := w.Literal1()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(" "); e != nil {
				return e
			}
			l, e = w.Literal2()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(")"); e != nil {
				return e
			}
		case 2:
			if e := s.AppendCString("("); e != nil {
				return e
			}
			l, e := w.BlockedLiteral()
			if e != nil {
				return e
			}
			if e = satLiteralInto(s, l); e != nil {
				return e
			}
			if e = s.AppendCString(" "); e != nil {
				return e
			}
			o, e := w.ClauseOffset()
			if e != nil {
				return e
			}
			r, e := a.GetClause(o)
			if e != nil {
				return e
			}
			if e = a.DisplayInto(r, s); e != nil {
				return e
			}
			if e = s.AppendCString(")"); e != nil {
				return e
			}
		case 3:
			n, e := w.ExternalIndex()
			if e != nil {
				return e
			}
			if e = s.AppendUint32(n); e != nil {
				return e
			}
		}
	}
	return nil
}

func (v *satWatchList) AppendArray(xs []satWatched) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satWatchList) Swap(other *satWatchList) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := other.change(); e != nil {
		return e
	}
	if v == other {
		return nil
	}
	a, b := v.backingBytes(), other.backingBytes()
	if v.budget != other.budget {
		if e := v.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			v.budget.Release(b)
			return e
		}
		v.budget.Release(a)
		other.budget.Release(b)
	}
	v.data, other.data = other.data, v.data
	v.initialized, other.initialized = other.initialized, v.initialized
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	if other.epoch != ^uint64(0) {
		other.epoch++
	}
	return nil
}
func (v *satWatchList) EraseAt(i uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	copy(v.data[i:], v.data[i+1:])
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}

func (v *satClauseVector) AppendArray(xs []satStoredClauseRef) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseVector) Swap(other *satClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := other.change(); e != nil {
		return e
	}
	if v == other {
		return nil
	}
	if e := v.adoptBindings(other); e != nil {
		return e
	}
	if e := other.adoptBindings(v); e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf(satStoredClauseRef{}).Size())
	a, b := uint64(cap(v.data))*stride, uint64(cap(other.data))*stride
	if v.budget != other.budget {
		if e := v.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			v.budget.Release(b)
			return e
		}
		v.budget.Release(a)
		other.budget.Release(b)
	}
	v.data, other.data = other.data, v.data
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	if other.epoch != ^uint64(0) {
		other.epoch++
	}
	return nil
}
func (v *satClauseVector) EraseAt(i uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	copy(v.data[i:], v.data[i+1:])
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}

func (v *satClauseWrapperVector) AppendArray(xs []satClauseWrapper) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseWrapperVector) Swap(other *satClauseWrapperVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := other.change(); e != nil {
		return e
	}
	if v == other {
		return nil
	}
	if e := v.adoptBindings(other); e != nil {
		return e
	}
	if e := other.adoptBindings(v); e != nil {
		return e
	}
	stride := uint64(reflect.TypeOf(satClauseWrapper{}).Size())
	a, b := uint64(cap(v.data))*stride, uint64(cap(other.data))*stride
	if v.budget != other.budget {
		if e := v.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			v.budget.Release(b)
			return e
		}
		v.budget.Release(a)
		other.budget.Release(b)
	}
	v.data, other.data = other.data, v.data
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	if other.epoch != ^uint64(0) {
		other.epoch++
	}
	return nil
}
func (v *satClauseWrapperVector) EraseAt(i uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	copy(v.data[i:], v.data[i+1:])
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}

func (v *satActivityDomain) AppendArray(xs []uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satActivityDomain) Swap(other *satActivityDomain) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := other.change(); e != nil {
		return e
	}
	if v == other {
		return nil
	}
	stride := uint64(reflect.TypeOf(uint32(0)).Size())
	a, b := uint64(cap(v.data))*stride, uint64(cap(other.data))*stride
	if v.budget != other.budget {
		if e := v.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			v.budget.Release(b)
			return e
		}
		v.budget.Release(a)
		other.budget.Release(b)
	}
	v.data, other.data = other.data, v.data
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	if other.epoch != ^uint64(0) {
		other.epoch++
	}
	return nil
}
func (v *satActivityDomain) EraseAt(i uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	copy(v.data[i:], v.data[i+1:])
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}

func (v *satBinClauseVector) AppendArray(xs []satBinClause) error {
	if e := v.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if e := v.Push(x); e != nil {
			return e
		}
	}
	return nil
}
func (v *satBinClauseVector) Swap(other *satBinClauseVector) error {
	if e := v.change(); e != nil {
		return e
	}
	if e := other.change(); e != nil {
		return e
	}
	if v == other {
		return nil
	}
	stride := uint64(reflect.TypeOf(satBinClause{}).Size())
	a, b := uint64(cap(v.data))*stride, uint64(cap(other.data))*stride
	if v.budget != other.budget {
		if e := v.budget.Reserve(b); e != nil {
			return e
		}
		if e := other.budget.Reserve(a); e != nil {
			v.budget.Release(b)
			return e
		}
		v.budget.Release(a)
		other.budget.Release(b)
	}
	v.data, other.data = other.data, v.data
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	if other.epoch != ^uint64(0) {
		other.epoch++
	}
	return nil
}
func (v *satBinClauseVector) EraseAt(i uint32) error {
	if e := v.change(); e != nil {
		return e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return arena.ErrBounds
	}
	copy(v.data[i:], v.data[i+1:])
	v.data = v.data[:len(v.data)-1]
	if v.epoch != ^uint64(0) {
		v.epoch++
	}
	return nil
}

func (v *satClauseVector) bindRetained(a *satClauseAllocator) error {
	if a == nil || a.self != a || a.origin == 0 {
		return ErrSATOwner
	}
	if a == v.allocator {
		return nil
	}
	for _, x := range v.bindings {
		if x.origin == a.origin {
			return nil
		}
	}
	if len(v.bindings) == cap(v.bindings) {
		stride := uint64(reflect.TypeOf(satAllocatorBinding{}).Size())
		n, e := satVectorNextCapacity(uint64(cap(v.bindings)), stride)
		if e != nil {
			return e
		}
		if e = v.budget.Reserve(n * stride); e != nil {
			return e
		}
		next := make([]satAllocatorBinding, len(v.bindings), int(n))
		copy(next, v.bindings)
		v.budget.Release(uint64(cap(v.bindings)) * stride)
		v.bindings = next
	}
	v.bindings = append(v.bindings, satAllocatorBinding{a.origin, a})
	return nil
}
func (v *satClauseVector) BindAllocator(a *satClauseAllocator) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := a.open(); e != nil {
		return e
	}
	return v.bindRetained(a)
}
func (v *satClauseVector) adoptBindings(src *satClauseVector) error {
	if e := v.bindRetained(src.allocator); e != nil {
		return e
	}
	for _, x := range src.bindings {
		if e := v.bindRetained(x.owner); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseVector) resolve(origin uint64) (*satClauseAllocator, error) {
	if v.allocator.origin == origin {
		return v.allocator, nil
	}
	for _, x := range v.bindings {
		if x.origin == origin {
			return x.owner, nil
		}
	}
	return nil, ErrSATOwner
}

func (v *satClauseWrapperVector) bindRetained(a *satClauseAllocator) error {
	if a == nil || a.self != a || a.origin == 0 {
		return ErrSATOwner
	}
	if a == v.allocator {
		return nil
	}
	for _, x := range v.bindings {
		if x.origin == a.origin {
			return nil
		}
	}
	if len(v.bindings) == cap(v.bindings) {
		stride := uint64(reflect.TypeOf(satAllocatorBinding{}).Size())
		n, e := satVectorNextCapacity(uint64(cap(v.bindings)), stride)
		if e != nil {
			return e
		}
		if e = v.budget.Reserve(n * stride); e != nil {
			return e
		}
		next := make([]satAllocatorBinding, len(v.bindings), int(n))
		copy(next, v.bindings)
		v.budget.Release(uint64(cap(v.bindings)) * stride)
		v.bindings = next
	}
	v.bindings = append(v.bindings, satAllocatorBinding{a.origin, a})
	return nil
}
func (v *satClauseWrapperVector) BindAllocator(a *satClauseAllocator) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := a.open(); e != nil {
		return e
	}
	return v.bindRetained(a)
}
func (v *satClauseWrapperVector) adoptBindings(src *satClauseWrapperVector) error {
	if e := v.bindRetained(src.allocator); e != nil {
		return e
	}
	for _, x := range src.bindings {
		if e := v.bindRetained(x.owner); e != nil {
			return e
		}
	}
	return nil
}
func (v *satClauseWrapperVector) resolve(origin uint64) (*satClauseAllocator, error) {
	if v.allocator.origin == origin {
		return v.allocator, nil
	}
	for _, x := range v.bindings {
		if x.origin == origin {
			return x.owner, nil
		}
	}
	return nil, ErrSATOwner
}

type satAllocatorBinding struct {
	origin uint64
	owner  *satClauseAllocator
}

func NewSATWatchListFill(n uint32, x satWatched, b *arena.Budget) (*satWatchList, error) {
	v, e := NewSATWatchList(b)
	if e != nil {
		return nil, e
	}
	if e = v.ResizeFill(n, x); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func NewSATWatchListArray(xs []satWatched, b *arena.Budget) (*satWatchList, error) {
	v, e := NewSATWatchList(b)
	if e != nil {
		return nil, e
	}
	if e = v.AppendArray(xs); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (v *satWatchList) Insert(x satWatched) error { return v.Push(x) }
func (v *satWatchList) Stats() satStorageVectorStats {
	return satStorageVectorStats{v.ownerBytes, v.backingBytes(), 0}
}

func NewSATClauseVectorFill(a *satClauseAllocator, n uint32, x satStoredClauseRef, b *arena.Budget) (*satClauseVector, error) {
	v, e := NewSATClauseVector(a, b)
	if e != nil {
		return nil, e
	}
	if e = v.ResizeFill(n, x); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func NewSATClauseVectorArray(a *satClauseAllocator, xs []satStoredClauseRef, b *arena.Budget) (*satClauseVector, error) {
	v, e := NewSATClauseVector(a, b)
	if e != nil {
		return nil, e
	}
	if e = v.AppendArray(xs); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (v *satClauseVector) Insert(x satStoredClauseRef) error { return v.Push(x) }
func (v *satClauseVector) Stats() satStorageVectorStats {
	return satStorageVectorStats{v.ownerBytes, uint64(cap(v.data)) * uint64(reflect.TypeOf(satStoredClauseRef{}).Size()), uint64(cap(v.bindings)) * uint64(reflect.TypeOf(satAllocatorBinding{}).Size())}
}

func NewSATClauseVectorSized(a *satClauseAllocator, n uint32, b *arena.Budget) (*satClauseVector, error) {
	v, e := NewSATClauseVector(a, b)
	if e != nil {
		return nil, e
	}
	stride := uint64(reflect.TypeOf(satStoredClauseRef{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		v.Close()
		return nil, arena.ErrBounds
	}
	if n != 0 {
		if e = b.Reserve(uint64(n) * stride); e != nil {
			v.Close()
			return nil, e
		}
		v.data = make([]satStoredClauseRef, int(n))
	}
	return v, nil
}

// Source fill(size,element) recurses into itself after resize; retain that
// completed resize and report a typed nontermination adapter.
func (v *satClauseVector) FillSize(n uint32, x satStoredClauseRef) error {
	if e := v.Resize(n); e != nil {
		return e
	}
	return ErrSATNonterminating
}

func NewSATClauseWrapperVectorFill(a *satClauseAllocator, n uint32, x satClauseWrapper, b *arena.Budget) (*satClauseWrapperVector, error) {
	v, e := NewSATClauseWrapperVector(a, b)
	if e != nil {
		return nil, e
	}
	if e = v.ResizeFill(n, x); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func NewSATClauseWrapperVectorArray(a *satClauseAllocator, xs []satClauseWrapper, b *arena.Budget) (*satClauseWrapperVector, error) {
	v, e := NewSATClauseWrapperVector(a, b)
	if e != nil {
		return nil, e
	}
	if e = v.AppendArray(xs); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (v *satClauseWrapperVector) Insert(x satClauseWrapper) error { return v.Push(x) }
func (v *satClauseWrapperVector) Stats() satStorageVectorStats {
	return satStorageVectorStats{v.ownerBytes, uint64(cap(v.data)) * uint64(reflect.TypeOf(satClauseWrapper{}).Size()), uint64(cap(v.bindings)) * uint64(reflect.TypeOf(satAllocatorBinding{}).Size())}
}

func NewSATActivityDomainFill(n uint32, x uint32, b *arena.Budget) (*satActivityDomain, error) {
	v, e := NewSATActivityDomain(b)
	if e != nil {
		return nil, e
	}
	if e = v.ResizeFill(n, x); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func NewSATActivityDomainArray(xs []uint32, b *arena.Budget) (*satActivityDomain, error) {
	v, e := NewSATActivityDomain(b)
	if e != nil {
		return nil, e
	}
	if e = v.AppendArray(xs); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (v *satActivityDomain) Insert(x uint32) error { return v.Push(x) }
func (v *satActivityDomain) Stats() satStorageVectorStats {
	return satStorageVectorStats{v.ownerBytes, uint64(cap(v.data)) * uint64(reflect.TypeOf(uint32(0)).Size()), 0}
}

func NewSATActivityDomainSized(n uint32, b *arena.Budget) (*satActivityDomain, error) {
	v, e := NewSATActivityDomain(b)
	if e != nil {
		return nil, e
	}
	stride := uint64(reflect.TypeOf(uint32(0)).Size())
	if uint64(n) > uint64(^uint(0)>>1)/stride {
		v.Close()
		return nil, arena.ErrBounds
	}
	if n != 0 {
		if e = b.Reserve(uint64(n) * stride); e != nil {
			v.Close()
			return nil, e
		}
		v.data = make([]uint32, int(n))
	}
	return v, nil
}

// Source fill(size,element) recurses into itself after resize; retain that
// completed resize and report a typed nontermination adapter.
func (v *satActivityDomain) FillSize(n uint32, x uint32) error {
	if e := v.Resize(n); e != nil {
		return e
	}
	return ErrSATNonterminating
}

func NewSATBinClauseVectorFill(n uint32, x satBinClause, b *arena.Budget) (*satBinClauseVector, error) {
	v, e := NewSATBinClauseVector(b)
	if e != nil {
		return nil, e
	}
	if e = v.ResizeFill(n, x); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func NewSATBinClauseVectorArray(xs []satBinClause, b *arena.Budget) (*satBinClauseVector, error) {
	v, e := NewSATBinClauseVector(b)
	if e != nil {
		return nil, e
	}
	if e = v.AppendArray(xs); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (v *satBinClauseVector) Insert(x satBinClause) error { return v.Push(x) }
func (v *satBinClauseVector) Stats() satStorageVectorStats {
	return satStorageVectorStats{v.ownerBytes, uint64(cap(v.data)) * uint64(reflect.TypeOf(satBinClause{}).Size()), 0}
}

type satStorageVectorStats struct{ OwnerBytes, BackingBytes, BindingBytes uint64 }

var ErrSATNonterminating = errors.New("sat: nonterminating source operation")

// The bool backing tracks actual C++ object lifetimes after erase, separately
// from logical size. One aggregate reservation charges both actual arrays;
// growth preserves their old+new physical peak before replacing either root.
func (v *satWatchList) backingBytes() uint64 {
	return uint64(cap(v.data))*uint64(reflect.TypeOf(satWatched{}).Size()) + uint64(cap(v.initialized))*uint64(reflect.TypeOf(false).Size())
}
