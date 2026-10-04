// Source: src/util/scoped_numeral_buffer.h; src/util/buffer.h:29-59,146-151,210-242.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012-01-23.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Concrete polynomial mpzzp/128 specialization. The enclosing source scope
// owns these cold fields; only expansion acquires Runtime storage. Expanded
// numerals use the existing twelve-byte pointer-free Int codec, not Go padding.
// Relocation copies references without acquiring new numeral ownership.
type scopedIntegerModularBuffer128 struct {
	manager                              *IntegerModularManager
	inline                               [128]Int
	raw, pending                         memory.Block
	count, capacity, initialized, copied uint32
	numeric                              polynomialNumericRoots
	bodyCause, sourceCause               error
	constructed, closed, freeStarted     bool
}

func (b *scopedIntegerModularBuffer128) construct(m *IntegerModularManager) {
	*b = scopedIntegerModularBuffer128{manager: m, capacity: 128, constructed: true}
}
func (b *scopedIntegerModularBuffer128) open() error {
	if !b.constructed || b.closed {
		return arena.ErrClosed
	}
	return b.manager.check()
}
func (b *scopedIntegerModularBuffer128) physical(i uint32) (Int, error) {
	if i >= b.initialized {
		return Int{}, arena.ErrBounds
	}
	if b.raw == (memory.Block{}) {
		if i >= 128 {
			return Int{}, arena.ErrBounds
		}
		return b.inline[i], nil
	}
	return (integerArrayView{runtime: b.manager.manager.memory, raw: b.raw, count: b.initialized, stride: 12}).at(i)
}
func (b *scopedIntegerModularBuffer128) get(i uint32) (Int, error) {
	if e := b.open(); e != nil {
		return Int{}, e
	}
	if modularSourceDebug && i >= b.count {
		return Int{}, ErrPolynomialAssertion
	}
	return b.physical(i)
}
func (b *scopedIntegerModularBuffer128) store(i uint32, v Int) error {
	if i >= b.capacity {
		return arena.ErrBounds
	}
	if b.raw == (memory.Block{}) {
		if i >= 128 {
			return arena.ErrBounds
		}
		b.inline[i] = v
		return nil
	}
	return (integerArrayView{runtime: b.manager.manager.memory, raw: b.raw, count: b.capacity, stride: 12}).store(i, v)
}
func (b *scopedIntegerModularBuffer128) expand() error {
	nc := b.capacity << 1
	var e error
	b.pending, e = b.manager.manager.memory.Allocate(uint64(nc) * 12)
	if e != nil {
		return e
	}
	dst := integerArrayView{runtime: b.manager.manager.memory, raw: b.pending, count: nc, stride: 12}
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
	if b.raw != (memory.Block{}) {
		b.freeStarted = true
		if e = b.manager.manager.memory.Deallocate(&b.raw); e != nil {
			return e
		}
	}
	b.raw = b.pending
	b.pending = memory.Block{}
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
		return ErrPolynomialAssertion
	}
	a, e := b.physical(i)
	if e != nil {
		return e
	}
	before, e := polynomialNumericBefore(b.manager.manager)
	if e != nil {
		return e
	}
	err = b.manager.Copy(&a, v)
	after := b.numeric.after(b.manager.manager, before)
	stored := b.store(i, a)
	err = polynomialNewtonJoin(err, polynomialNewtonJoin(after, stored))
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
		_, e = upolynomialSourceNumeralDel(b.manager.manager, &a)
	}
	return polynomialNewtonJoin(e, b.store(i, a))
}
func (b *scopedIntegerModularBuffer128) shrink(n uint32) error {
	if e := b.open(); e != nil {
		return e
	}
	if b.sourceCause != nil {
		return b.sourceCause
	}
	if modularSourceDebug && n > b.count {
		return ErrPolynomialAssertion
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
func (b *scopedIntegerModularBuffer128) resize(n uint32) (err error) {
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
	if !b.constructed || b.closed {
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
	if b.raw != (memory.Block{}) {
		b.freeStarted = true
		if e := b.manager.manager.memory.Deallocate(&b.raw); e != nil {
			b.sourceCause = e
			return e
		}
	}
	b.closed = true
	return nil
}
func (b *scopedIntegerModularBuffer128) cleanup() error {
	if !b.constructed {
		return nil
	}
	if e := b.numeric.cleanup(b.manager.manager); e != nil {
		return e
	}
	// The physical old buffer can be consumed before reporting a growth free
	// failure. Its fully copied pending relocation still names the same owned
	// numerals; it is not a second set of owning elements.
	if b.freeStarted && b.raw == (memory.Block{}) && b.pending != (memory.Block{}) {
		b.raw = b.pending
		b.pending = memory.Block{}
		b.capacity = b.capacity << 1
	}
	for i := uint32(0); i < b.count; i++ {
		if e := b.del(i, true); e != nil {
			return e
		}
	}
	b.count = 0
	for _, raw := range []*memory.Block{&b.raw, &b.pending} {
		if *raw != (memory.Block{}) {
			if e := b.manager.manager.memory.Deallocate(raw); e != nil {
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
func (b *scopedIntegerModularBuffer128) view() (integerArrayView, error) {
	if e := b.open(); e != nil {
		return integerArrayView{}, e
	}
	if b.raw == (memory.Block{}) {
		return integerArrayView{values: b.inline[:], count: b.count}, nil
	}
	return integerArrayView{runtime: b.manager.manager.memory, raw: b.raw, count: b.count, stride: 12}, nil
}
