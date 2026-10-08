// Source: src/util/scoped_numeral_buffer.h; src/util/buffer.h:29-59,146-151,210-242.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// ScopedNumeralBuffer is the non-generic Go form of
// _scoped_numeral_buffer. It owns every numeral in values and borrows manager.
type ScopedNumeralBuffer struct {
	self    *ScopedNumeralBuffer
	manager *ScopedNumeralManager
	values  []any
	closed  bool
}

func NewScopedNumeralBuffer(manager *ScopedNumeralManager) (*ScopedNumeralBuffer, error) {
	if !manager.valid() {
		return nil, ErrScopedOwner
	}
	buffer := &ScopedNumeralBuffer{manager: manager, values: make([]any, 0, 16)}
	buffer.self = buffer
	return buffer, nil
}

func (buffer *ScopedNumeralBuffer) requireOpen() error {
	if buffer == nil || buffer.self != buffer {
		return ErrScopedCopy
	}
	if buffer.closed {
		return arena.ErrClosed
	}
	return nil
}

func (buffer *ScopedNumeralBuffer) Manager() (*ScopedNumeralManager, error) {
	if err := buffer.requireOpen(); err != nil {
		return nil, err
	}
	return buffer.manager, nil
}

func (buffer *ScopedNumeralBuffer) Size() uint32 {
	if err := buffer.requireOpen(); err != nil {
		panic(err)
	}
	return uint32(len(buffer.values))
}

func (buffer *ScopedNumeralBuffer) Empty() bool { return buffer.Size() == 0 }

func (buffer *ScopedNumeralBuffer) Get(index uint32) (*any, error) {
	if err := buffer.requireOpen(); err != nil {
		return nil, err
	}
	if uint64(index) >= uint64(len(buffer.values)) {
		return nil, arena.ErrBounds
	}
	return &buffer.values[index], nil
}

// CPtr returns the live Go slice corresponding to the source raw buffer. It is
// invalidated by structural edits.
func (buffer *ScopedNumeralBuffer) CPtr() ([]any, error) {
	if err := buffer.requireOpen(); err != nil {
		return nil, err
	}
	return buffer.values, nil
}

// PushBack appends a default numeral before copying value into it, preserving
// the source publication order.
func (buffer *ScopedNumeralBuffer) PushBack(value any) error {
	if err := buffer.requireOpen(); err != nil {
		return err
	}
	buffer.values = append(buffer.values, buffer.manager.New())
	return buffer.manager.Set(&buffer.values[len(buffer.values)-1], value)
}

// Reset deletes values from first to last and retains backing capacity.
func (buffer *ScopedNumeralBuffer) Reset() error {
	if err := buffer.requireOpen(); err != nil {
		return err
	}
	for index := range buffer.values {
		if err := buffer.manager.Delete(&buffer.values[index]); err != nil {
			return err
		}
	}
	clear(buffer.values)
	buffer.values = buffer.values[:0]
	return nil
}

func (buffer *ScopedNumeralBuffer) Shrink(size uint32) error {
	if err := buffer.requireOpen(); err != nil {
		return err
	}
	if uint64(size) > uint64(len(buffer.values)) {
		return arena.ErrBounds
	}
	target := int(size)
	if target == len(buffer.values) {
		return nil
	}
	for index := target; index < len(buffer.values); index++ {
		if err := buffer.manager.Delete(&buffer.values[index]); err != nil {
			return err
		}
	}
	clear(buffer.values[target:])
	buffer.values = buffer.values[:target]
	return nil
}

func (buffer *ScopedNumeralBuffer) Resize(size uint32) error {
	if err := buffer.requireOpen(); err != nil {
		return err
	}
	if uint64(size) <= uint64(len(buffer.values)) {
		return buffer.Shrink(size)
	}
	for uint32(len(buffer.values)) < size {
		zero := buffer.manager.New()
		if err := buffer.manager.SetInt(&zero, 0); err != nil {
			_ = buffer.manager.Delete(&zero)
			return err
		}
		buffer.values = append(buffer.values, zero)
	}
	return nil
}

func (buffer *ScopedNumeralBuffer) Close() error {
	if buffer == nil || buffer.self != buffer {
		return ErrScopedCopy
	}
	if buffer.closed {
		return nil
	}
	if err := buffer.Reset(); err != nil {
		return err
	}
	buffer.values = nil
	buffer.manager = nil
	buffer.closed = true
	return nil
}

// Concrete polynomial mpzzp/128 specialization. The enclosing source scope
// owns these cold fields; only expansion acquires Runtime storage. Expanded
// numerals use the existing twelve-byte pointer-free Int codec, not Go padding.
// Relocation copies references without acquiring new numeral ownership.

type scopedIntegerModularBuffer128 struct {
	manager                              *IntegerModularManager
	inline                               [128]Int
	Raw, Pending                         Block
	count, capacity, initialized, copied uint32
	numeric                              NumericAllocationRoots
	bodyCause, sourceCause               error
	Constructed, closed, freeStarted     bool
}

func (b *scopedIntegerModularBuffer128) construct(m *IntegerModularManager) {
	*b = scopedIntegerModularBuffer128{manager: m, capacity: 128, Constructed: true}
}
func (b *scopedIntegerModularBuffer128) open() error {
	if !b.Constructed || b.closed {
		return arena.ErrClosed
	}
	return b.manager.check()
}
func (b *scopedIntegerModularBuffer128) physical(i uint32) (Int, error) {
	if i >= b.initialized {
		return Int{}, arena.ErrBounds
	}
	if b.Raw == (Block{}) {
		if i >= 128 {
			return Int{}, arena.ErrBounds
		}
		return b.inline[i], nil
	}
	return (IntegerArrayView{Runtime: b.manager.manager.Memory, Raw: b.Raw, Count: b.initialized, Stride: 12}).At(i)
}
func (b *scopedIntegerModularBuffer128) get(i uint32) (Int, error) {
	if e := b.open(); e != nil {
		return Int{}, e
	}
	if modularSourceDebug && i >= b.count {
		return Int{}, ErrNumericAssertion
	}
	return b.physical(i)
}
func (b *scopedIntegerModularBuffer128) store(i uint32, v Int) error {
	if i >= b.capacity {
		return arena.ErrBounds
	}
	if b.Raw == (Block{}) {
		if i >= 128 {
			return arena.ErrBounds
		}
		b.inline[i] = v
		return nil
	}
	return (IntegerArrayView{Runtime: b.manager.manager.Memory, Raw: b.Raw, Count: b.capacity, Stride: 12}).store(i, v)
}
func (b *scopedIntegerModularBuffer128) expand() error {
	nc := b.capacity << 1
	var e error
	b.Pending, e = b.manager.manager.Memory.Allocate(uint64(nc) * 12)
	if e != nil {
		return e
	}
	dst := IntegerArrayView{Runtime: b.manager.manager.Memory, Raw: b.Pending, Count: nc, Stride: 12}
	for b.copied < b.count {
		v, e := b.physical(b.copied)
		if e != nil {
			return e
		}
		if e = dst.store(b.copied, v); e != nil {
			return e
		}
		b.copied++
	}
	if b.Raw != (Block{}) {
		b.freeStarted = true
		if e = b.manager.manager.Memory.Deallocate(&b.Raw); e != nil {
			return e
		}
	}
	b.Raw = b.Pending
	b.Pending = Block{}
	b.capacity = nc
	b.copied = 0
	b.freeStarted = false
	return nil
}
func (b *scopedIntegerModularBuffer128) pushZero() error {
	if b.count >= b.capacity {
		if e := b.expand(); e != nil {
			return e
		}
	}
	if e := b.store(b.count, Int{}); e != nil {
		return e
	}
	b.count++
	if b.count > b.initialized {
		b.initialized = b.count
	}
	return nil
}
func (b *scopedIntegerModularBuffer128) set(i uint32, v *Int) (err error) {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.bodyCause != nil {
		return b.bodyCause
	}
	if modularSourceDebug && i >= b.count {
		return ErrNumericAssertion
	}
	a, e := b.physical(i)
	if e != nil {
		return e
	}
	before, e := CaptureNumericAllocationRoots(b.manager.manager)
	if e != nil {
		return e
	}
	err = b.manager.Copy(&a, v)
	after := b.numeric.After(b.manager.manager, before)
	stored := b.store(i, a)
	err = JoinNumericErrors(err, JoinNumericErrors(after, stored))
	if err != nil && b.bodyCause == nil {
		b.bodyCause = err
	}
	return err
}
func (b *scopedIntegerModularBuffer128) push(v *Int) (err error) {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.bodyCause != nil {
		return b.bodyCause
	}
	if e := b.pushZero(); e != nil {
		b.bodyCause = e
		return e
	}
	return b.set(b.count-1, v)
}
func (b *scopedIntegerModularBuffer128) del(i uint32, owning bool) error {
	a, e := b.physical(i)
	if e != nil {
		return e
	}
	if owning {
		e = b.manager.Del(&a)
	} else {
		_, e = SourceNumeralDel(b.manager.manager, &a)
	}
	return JoinNumericErrors(e, b.store(i, a))
}
func (b *scopedIntegerModularBuffer128) shrink(n uint32) error {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if modularSourceDebug && n > b.count {
		return ErrNumericAssertion
	}
	old := b.count
	if old == n {
		return nil
	}
	for i := n; i < old; i++ {
		if e := b.del(i, false); e != nil {
			b.sourceCause = e
			return e
		}
	}
	for i := n; i < old; i++ {
		b.count--
	}
	return nil
}
func (b *scopedIntegerModularBuffer128) reset() error { return b.shrink(0) }
func (b *scopedIntegerModularBuffer128) Resize(n uint32) (err error) {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.bodyCause != nil {
		return b.bodyCause
	}
	old := b.count
	if n <= old {
		if e := b.shrink(n); e != nil {
			return e
		}
	}
	for b.count < n {
		if e := b.pushZero(); e != nil {
			b.bodyCause = e
			return e
		}
	}
	return nil
}
func (b *scopedIntegerModularBuffer128) sourceClose() error {
	if !b.Constructed || b.closed {
		return nil
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if b.freeStarted {
		return b.bodyCause
	}
	if e := b.reset(); e != nil {
		return e
	}
	if b.Raw != (Block{}) {
		b.freeStarted = true
		if e := b.manager.manager.Memory.Deallocate(&b.Raw); e != nil {
			b.sourceCause = e
			return e
		}
	}
	b.closed = true
	return nil
}
func (b *scopedIntegerModularBuffer128) cleanup() error {
	if !b.Constructed {
		return nil
	}
	if e := b.numeric.Cleanup(b.manager.manager); e != nil {
		return e
	}
	// The physical old buffer can be consumed before reporting a growth free
	// failure. Its fully copied pending relocation still names the same owned
	// numerals; it is not a second set of owning elements.
	if b.freeStarted && b.Raw == (Block{}) && b.Pending != (Block{}) {
		b.Raw = b.Pending
		b.Pending = Block{}
		b.capacity = b.capacity << 1
	}
	for i := uint32(0); i < b.count; i++ {
		if e := b.del(i, true); e != nil {
			return e
		}
	}
	b.count = 0
	for _, raw := range []*Block{&b.Raw, &b.Pending} {
		if *raw != (Block{}) {
			if e := b.manager.manager.Memory.Deallocate(raw); e != nil {
				return e
			}
		}
	}
	b.closed = true
	return nil
}

// Source c_ptr/size borrows the current physical buffer. Inline values are the
// actual cold fields; expanded values use the common codec directly. Neither
// route clones numerals or acquires their ownership.
func (b *scopedIntegerModularBuffer128) view() (IntegerArrayView, error) {
	if e := b.open(); e != nil {
		return IntegerArrayView{}, e
	}
	if b.Raw == (Block{}) {
		return IntegerArrayView{Values: b.inline[:], Count: b.count}, nil
	}
	return IntegerArrayView{Runtime: b.manager.manager.Memory, Raw: b.Raw, Count: b.count, Stride: 12}, nil
}
