// Licensed under the MIT license.
// Concrete int32 configurations of src/util/parray.h.
package g3

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"sync/atomic"

	"github.com/glycerine/g3/golang/arena"
)

var ErrPArrayRef = errors.New("g3: persistent array source reference precondition")
var ErrPArrayLive = errors.New("g3: persistent array manager has live cells")
var pArrayIdentity atomic.Uint64

const (
	pArraySet uint32 = iota
	pArrayPush
	pArrayPop
	pArrayRoot
)
const pArrayMaxRC uint32 = (1 << 30) - 1
const pArrayCellBytes uint32 = 24

var pArrayCapacityHeader = uint32(reflect.TypeOf(uintptr(0)).Size())

type Int32PArrayConfig struct {
	Budget        *arena.Budget
	Allocator     *arena.SmallBytes
	PreserveRoots bool
}

// Ref is manually owned. Use Copy to acquire ownership; ordinary Go copies are
// borrowed aliases. Get can mutate this particular reference during rerooting.
type Int32PArrayRef struct {
	owner   uint64
	cell    arena.SmallBytesRef
	updates uint32
}
type pArrayCell struct {
	rc, kind, idx uint32
	elem          int32
	link          arena.SmallBytesRef
}
type Int32PArrayStats struct {
	LiveCells, LiveValueBlocks                      uint32
	OwnerBytes, ScratchBytes, CellBytes, ValueBytes uint64
}

// Allocator is borrowed, must outlive the manager, and must be configured with
// the same Budget. Callers serialize the shared owner graph; no budget identity
// getter is invented. All encoded records/values are charged by that allocator.
type Int32PArrayManager struct {
	allocator             *arena.SmallBytes
	budget                *arena.Budget
	id                    uint64
	getTmp, rerootTmp     []arena.SmallBytesRef
	liveCells, liveBlocks uint32
	valueBytes            uint64
	preserve, closed      bool
}

var pArrayOwnerBytes = uint64(reflect.TypeOf(Int32PArrayManager{}).Size())

func NewInt32PArrayManager(c Int32PArrayConfig) (*Int32PArrayManager, error) {
	if c.Allocator == nil {
		return nil, ErrPArrayRef
	}
	if e := c.Budget.Reserve(pArrayOwnerBytes); e != nil {
		return nil, e
	}
	var id uint64
	for {
		old := pArrayIdentity.Load()
		if old == math.MaxUint64 {
			c.Budget.Release(pArrayOwnerBytes)
			return nil, arena.ErrSlotLimit
		}
		if pArrayIdentity.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	return &Int32PArrayManager{allocator: c.Allocator, budget: c.Budget, id: id, preserve: c.PreserveRoots}, nil
}
func (m *Int32PArrayManager) open() error {
	if m == nil || m.closed {
		return arena.ErrClosed
	}
	return nil
}
func pArrayNull(r arena.SmallBytesRef) bool { return r == (arena.SmallBytesRef{}) }
func (m *Int32PArrayManager) valid(r *Int32PArrayRef) error {
	if e := m.open(); e != nil {
		return e
	}
	if r == nil {
		return ErrPArrayRef
	}
	if pArrayNull(r.cell) {
		if r.owner != 0 && r.owner != m.id {
			return ErrPArrayRef
		}
		return nil
	}
	if r.owner != m.id || !m.allocator.IsLive(r.cell) {
		return ErrPArrayRef
	}
	return nil
}
func (m *Int32PArrayManager) read(r arena.SmallBytesRef, offset, width uint32) (uint64, error) {
	var v uint64
	for i := uint32(0); i < width; i++ {
		b, e := m.allocator.Read(r, offset+i)
		if e != nil {
			return 0, ErrPArrayRef
		}
		v |= uint64(b) << (8 * i)
	}
	return v, nil
}
func (m *Int32PArrayManager) write(r arena.SmallBytesRef, offset, width uint32, v uint64) error {
	for i := uint32(0); i < width; i++ {
		if e := m.allocator.Write(r, offset+i, byte(v>>(8*i))); e != nil {
			return ErrPArrayRef
		}
	}
	return nil
}

// Explicit 24-byte encoding: RC/kind0, index4, element8, padding12,
// link descriptor slot16/generation20. No raw address or Go pointer is encoded.
func (m *Int32PArrayManager) cell(r arena.SmallBytesRef) (pArrayCell, error) {
	if !m.allocator.IsLive(r) {
		return pArrayCell{}, ErrPArrayRef
	}
	n, e := m.allocator.Size(r)
	if e != nil || n != pArrayCellBytes {
		return pArrayCell{}, ErrPArrayRef
	}
	a, e := m.read(r, 0, 4)
	if e != nil {
		return pArrayCell{}, e
	}
	i, e := m.read(r, 4, 4)
	if e != nil {
		return pArrayCell{}, e
	}
	v, e := m.read(r, 8, 4)
	if e != nil {
		return pArrayCell{}, e
	}
	s, e := m.read(r, 16, 4)
	if e != nil {
		return pArrayCell{}, e
	}
	g, e := m.read(r, 20, 4)
	if e != nil {
		return pArrayCell{}, e
	}
	return pArrayCell{uint32(a) & pArrayMaxRC, uint32(a) >> 30, uint32(i), int32(v), arena.SmallBytesRef{Slot: arena.Slot(s), Generation: arena.Generation(g)}}, nil
}
func (m *Int32PArrayManager) store(r arena.SmallBytesRef, c pArrayCell) error {
	if e := m.write(r, 0, 4, uint64((c.kind<<30)|(c.rc&pArrayMaxRC))); e != nil {
		return e
	}
	if e := m.write(r, 4, 4, uint64(c.idx)); e != nil {
		return e
	}
	if e := m.write(r, 8, 4, uint64(uint32(c.elem))); e != nil {
		return e
	}
	if e := m.write(r, 12, 4, 0); e != nil {
		return e
	}
	if e := m.write(r, 16, 4, uint64(c.link.Slot)); e != nil {
		return e
	}
	return m.write(r, 20, 4, uint64(c.link.Generation))
}
func (m *Int32PArrayManager) mkCell(kind uint32) (arena.SmallBytesRef, error) {
	r, e := m.allocator.Allocate(pArrayCellBytes)
	if e != nil {
		return r, e
	}
	m.liveCells++
	if e = m.store(r, pArrayCell{rc: 1, kind: kind}); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *Int32PArrayManager) freeCell(r arena.SmallBytesRef) {
	if e := m.allocator.Deallocate(pArrayCellBytes, r); e != nil {
		panic(e)
	}
	m.liveCells--
}
func (m *Int32PArrayManager) inc(r arena.SmallBytesRef) error {
	if pArrayNull(r) {
		return nil
	}
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	c.rc = (c.rc + 1) & pArrayMaxRC
	return m.store(r, c)
}
func (m *Int32PArrayManager) dec(r arena.SmallBytesRef) error {
	for !pArrayNull(r) {
		c, e := m.cell(r)
		if e != nil || c.rc == 0 {
			return ErrPArrayRef
		}
		c.rc--
		if c.rc != 0 {
			return m.store(r, c)
		}
		next := c.link
		if c.kind == pArrayRoot {
			if !pArrayNull(c.link) {
				m.freeValues(c.link)
			}
			next = arena.SmallBytesRef{}
		}
		m.freeCell(r)
		r = next
	}
	return nil
}
func (m *Int32PArrayManager) capacity(r arena.SmallBytesRef) (uint64, error) {
	if pArrayNull(r) {
		return 0, nil
	}
	return m.read(r, 0, pArrayCapacityHeader)
}
func (m *Int32PArrayManager) allocateValues(capacity uint64) (arena.SmallBytesRef, error) {
	if capacity > (math.MaxUint32-uint64(pArrayCapacityHeader))/4 {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	bytes := uint32(capacity*4) + pArrayCapacityHeader
	r, e := m.allocator.Allocate(bytes)
	if e != nil {
		return r, e
	}
	m.liveBlocks++
	m.valueBytes += uint64(bytes)
	if e = m.write(r, 0, pArrayCapacityHeader, capacity); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *Int32PArrayManager) freeValues(r arena.SmallBytesRef) {
	n, e := m.allocator.Size(r)
	if e != nil {
		panic(e)
	}
	if e = m.allocator.Deallocate(n, r); e != nil {
		panic(e)
	}
	m.liveBlocks--
	m.valueBytes -= uint64(n)
}
func (m *Int32PArrayManager) value(r arena.SmallBytesRef, i uint32) (int32, error) {
	v, e := m.read(r, pArrayCapacityHeader+4*i, 4)
	return int32(v), e
}
func (m *Int32PArrayManager) setValue(r arena.SmallBytesRef, i uint32, v int32) error {
	return m.write(r, pArrayCapacityHeader+4*i, 4, uint64(uint32(v)))
}
func (m *Int32PArrayManager) expand(r *arena.SmallBytesRef) error {
	old, e := m.capacity(*r)
	if e != nil {
		return e
	}
	n := uint64(2)
	if old != 0 {
		if old > (math.MaxUint64-1)/3 {
			return arena.ErrBounds
		}
		n = (3*old + 1) >> 1
	}
	next, e := m.allocateValues(n)
	if e != nil {
		return e
	}
	for i := uint32(0); uint64(i) < old; i++ {
		v, e := m.value(*r, i)
		if e != nil {
			m.freeValues(next)
			return e
		}
		if e = m.setValue(next, i, v); e != nil {
			m.freeValues(next)
			return e
		}
	}
	if !pArrayNull(*r) {
		m.freeValues(*r)
	}
	*r = next
	return nil
}
func (m *Int32PArrayManager) rootPush(c *pArrayCell, v int32) error {
	cap, e := m.capacity(c.link)
	if e != nil {
		return e
	}
	if uint64(c.idx) == cap {
		if e = m.expand(&c.link); e != nil {
			return e
		}
	}
	if c.idx == math.MaxUint32 {
		return arena.ErrBounds
	}
	if e = m.setValue(c.link, c.idx, v); e != nil {
		return e
	}
	c.idx++
	return nil
}
func (m *Int32PArrayManager) growScratch(a *[]arena.SmallBytesRef) error {
	if len(*a) < cap(*a) {
		return nil
	}
	n := uint64(2)
	if cap(*a) != 0 {
		n = (3*uint64(cap(*a)) + 1) >> 1
	}
	if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/8 {
		return arena.ErrBounds
	}
	if e := m.budget.Reserve(n * 8); e != nil {
		return e
	}
	next := make([]arena.SmallBytesRef, len(*a), int(n))
	copy(next, *a)
	m.budget.Release(uint64(cap(*a)) * 8)
	*a = next
	return nil
}
func (m *Int32PArrayManager) scratchPush(a *[]arena.SmallBytesRef, r arena.SmallBytesRef) error {
	if e := m.growScratch(a); e != nil {
		return e
	}
	*a = append(*a, r)
	return nil
}
func (m *Int32PArrayManager) sizeCell(r arena.SmallBytesRef) (uint32, error) {
	for !pArrayNull(r) {
		c, e := m.cell(r)
		if e != nil {
			return 0, e
		}
		switch c.kind {
		case pArraySet:
			r = c.link
		case pArrayPush:
			return c.idx + 1, nil
		case pArrayPop:
			return c.idx - 1, nil
		case pArrayRoot:
			return c.idx, nil
		}
	}
	return 0, nil
}
func (m *Int32PArrayManager) Mk(out *Int32PArrayRef) error {
	if e := m.valid(out); e != nil {
		return e
	}
	if e := m.dec(out.cell); e != nil {
		return e
	}
	*out = Int32PArrayRef{} // clear the source dangling-output exception continuation
	r, e := m.mkCell(pArrayRoot)
	if e != nil {
		return e
	}
	*out = Int32PArrayRef{owner: m.id, cell: r}
	return nil
}
func (m *Int32PArrayManager) Del(out *Int32PArrayRef) error {
	if e := m.valid(out); e != nil {
		return e
	}
	if e := m.dec(out.cell); e != nil {
		return e
	}
	*out = Int32PArrayRef{}
	return nil
}
func (m *Int32PArrayManager) Copy(src, out *Int32PArrayRef) error {
	if e := m.valid(src); e != nil {
		return e
	}
	if e := m.valid(out); e != nil {
		return e
	}
	r := src.cell
	if e := m.inc(r); e != nil {
		return e
	}
	if e := m.dec(out.cell); e != nil {
		_ = m.dec(r)
		return e
	}
	*out = Int32PArrayRef{cell: r}
	if !pArrayNull(r) {
		out.owner = m.id
	}
	return nil
}
func (m *Int32PArrayManager) Size(ref Int32PArrayRef) (uint32, error) {
	if e := m.valid(&ref); e != nil {
		return 0, e
	}
	return m.sizeCell(ref.cell)
}
func (m *Int32PArrayManager) Empty(ref Int32PArrayRef) (bool, error) {
	n, e := m.Size(ref)
	return n == 0, e
}
func (m *Int32PArrayManager) Get(ref *Int32PArrayRef, index uint32) (int32, error) {
	if e := m.valid(ref); e != nil {
		return 0, e
	}
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return 0, e
	}
	if index >= n {
		return 0, arena.ErrBounds
	}
	r := ref.cell
	var trail uint32
	for {
		if trail > 8 {
			if e = m.Reroot(ref); e != nil {
				return 0, e
			}
			c, e := m.cell(ref.cell)
			if e != nil {
				return 0, e
			}
			return m.value(c.link, index)
		}
		c, e := m.cell(r)
		if e != nil {
			return 0, e
		}
		if c.kind == pArrayRoot {
			return m.value(c.link, index)
		}
		if (c.kind == pArraySet || c.kind == pArrayPush) && c.idx == index {
			return c.elem, nil
		}
		trail++
		r = c.link
	}
}
func (m *Int32PArrayManager) SetFrom(src *Int32PArrayRef, i uint32, v int32, out *Int32PArrayRef) error {
	if e := m.valid(src); e != nil {
		return e
	}
	n, e := m.Size(*src)
	if e != nil {
		return e
	}
	if i >= n {
		return arena.ErrBounds
	}
	if src == out {
		return m.Set(out, i, v)
	}
	if e = m.Copy(src, out); e != nil {
		return e
	}
	return m.Set(out, i, v)
}
func (m *Int32PArrayManager) PushBackFrom(src *Int32PArrayRef, v int32, out *Int32PArrayRef) error {
	if src == out {
		return m.PushBack(out, v)
	}
	if e := m.Copy(src, out); e != nil {
		return e
	}
	return m.PushBack(out, v)
}
func (m *Int32PArrayManager) PopBackFrom(src, out *Int32PArrayRef) error {
	if e := m.valid(src); e != nil {
		return e
	}
	n, e := m.Size(*src)
	if e != nil {
		return e
	}
	if n == 0 {
		return arena.ErrBounds
	}
	if src == out {
		return m.PopBack(out)
	}
	if e = m.Copy(src, out); e != nil {
		return e
	}
	return m.PopBack(out)
}

// Shared ROOT reversal publishes the new root before its final requested update.
func (m *Int32PArrayManager) reverseRoot(ref *Int32PArrayRef, c pArrayCell, operation, i uint32) (pArrayCell, error) {
	ref.updates++
	r, e := m.mkCell(pArrayRoot)
	if e != nil {
		return pArrayCell{}, e
	}
	root := pArrayCell{rc: 2, kind: pArrayRoot, idx: c.idx, link: c.link}
	if e = m.store(r, root); e != nil {
		panic(e)
	}
	switch operation {
	case pArraySet:
		c.kind = pArraySet
		c.idx = i
		c.elem, e = m.value(c.link, i)
	case pArrayPush:
		c.kind = pArrayPop
		c.idx = root.idx + 1
	case pArrayPop:
		c.kind = pArrayPush
		c.idx = root.idx - 1
		c.elem, e = m.value(c.link, c.idx)
	}
	if e != nil {
		m.freeCell(r)
		return pArrayCell{}, e
	}
	c.link = r
	c.rc--
	if e = m.store(ref.cell, c); e != nil {
		panic(e)
	}
	ref.cell = r
	return root, nil
}
func (m *Int32PArrayManager) Set(ref *Int32PArrayRef, i uint32, v int32) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return e
	}
	if i >= n {
		return arena.ErrBounds
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		if c.rc == 1 {
			return m.setValue(c.link, i, v)
		}
		if m.preserve {
			if ref.updates > n {
				if e = m.Unshare(ref); e != nil {
					return e
				}
				return m.Set(ref, i, v)
			}
			root, e := m.reverseRoot(ref, c, pArraySet, i)
			if e != nil {
				return e
			}
			return m.setValue(root.link, i, v)
		}
	}
	r, e := m.mkCell(pArraySet)
	if e != nil {
		return e
	}
	if e = m.store(r, pArrayCell{rc: 1, kind: pArraySet, idx: i, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	ref.owner = m.id
	return nil
}
func (m *Int32PArrayManager) PushBack(ref *Int32PArrayRef, v int32) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if pArrayNull(ref.cell) {
		if e := m.Mk(ref); e != nil {
			return e
		}
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		if c.rc == 1 {
			if e = m.rootPush(&c, v); e != nil {
				return e
			}
			return m.store(ref.cell, c)
		}
		if m.preserve {
			if ref.updates > c.idx {
				if e = m.Unshare(ref); e != nil {
					return e
				}
				return m.PushBack(ref, v)
			}
			root, e := m.reverseRoot(ref, c, pArrayPush, 0)
			if e != nil {
				return e
			}
			if e = m.rootPush(&root, v); e != nil {
				return e
			}
			return m.store(ref.cell, root)
		}
	}
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return e
	}
	r, e := m.mkCell(pArrayPush)
	if e != nil {
		return e
	}
	if e = m.store(r, pArrayCell{rc: 1, kind: pArrayPush, idx: n, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *Int32PArrayManager) PopBack(ref *Int32PArrayRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return e
	}
	if n == 0 {
		return arena.ErrBounds
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		if c.rc == 1 {
			c.idx--
			return m.store(ref.cell, c)
		}
		if m.preserve {
			if ref.updates > n {
				if e = m.Unshare(ref); e != nil {
					return e
				}
				return m.PopBack(ref)
			}
			root, e := m.reverseRoot(ref, c, pArrayPop, 0)
			if e != nil {
				return e
			}
			root.idx--
			return m.store(ref.cell, root)
		}
	}
	r, e := m.mkCell(pArrayPop)
	if e != nil {
		return e
	}
	if e = m.store(r, pArrayCell{rc: 1, kind: pArrayPop, idx: n, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *Int32PArrayManager) getValues(r arena.SmallBytesRef) (block arena.SmallBytesRef, size uint32, err error) {
	m.getTmp = m.getTmp[:0]
	for {
		c, e := m.cell(r)
		if e != nil {
			return block, 0, e
		}
		if c.kind == pArrayRoot {
			size = c.idx
			cap, e := m.capacity(c.link)
			if e != nil {
				return block, 0, e
			}
			block, e = m.allocateValues(cap)
			if e != nil {
				return block, 0, e
			}
			for i := uint32(0); i < size; i++ {
				v, e := m.value(c.link, i)
				if e != nil {
					m.freeValues(block)
					return arena.SmallBytesRef{}, 0, e
				}
				if e = m.setValue(block, i, v); e != nil {
					m.freeValues(block)
					return arena.SmallBytesRef{}, 0, e
				}
			}
			break
		}
		if e = m.scratchPush(&m.getTmp, r); e != nil {
			return block, 0, e
		}
		r = c.link
	}
	// The temporary block remains lexical ownership until complete publication.
	defer func() {
		if err != nil && !pArrayNull(block) {
			m.freeValues(block)
			block = arena.SmallBytesRef{}
		}
	}()
	for i := len(m.getTmp); i > 0; {
		i--
		c, e := m.cell(m.getTmp[i])
		if e != nil {
			return block, size, e
		}
		switch c.kind {
		case pArraySet:
			err = m.setValue(block, c.idx, c.elem)
		case pArrayPush:
			root := pArrayCell{idx: size, link: block}
			err = m.rootPush(&root, c.elem)
			block = root.link
			size = root.idx
		case pArrayPop:
			if size == 0 {
				return block, size, ErrPArrayRef
			}
			size--
		}
		if err != nil {
			return block, size, err
		}
	}
	return block, size, nil
}
func (m *Int32PArrayManager) unfoldCell(r arena.SmallBytesRef) error {
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		return nil
	}
	block, n, e := m.getValues(r)
	if e != nil {
		return e
	}
	if e = m.dec(c.link); e != nil {
		m.freeValues(block)
		return e
	}
	c.kind = pArrayRoot
	c.idx = n
	c.link = block
	return m.store(r, c)
}
func (m *Int32PArrayManager) Unfold(ref *Int32PArrayRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if pArrayNull(ref.cell) {
		return ErrPArrayRef
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		return nil
	}
	if e = m.unfoldCell(ref.cell); e != nil {
		return e
	}
	ref.updates = 0
	return nil
}
func (m *Int32PArrayManager) Unshare(ref *Int32PArrayRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if pArrayNull(ref.cell) {
		return ErrPArrayRef
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot && c.rc == 1 {
		return nil
	}
	r, e := m.mkCell(pArrayRoot)
	if e != nil {
		return e
	}
	block, n, e := m.getValues(ref.cell)
	if e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, pArrayCell{rc: 1, kind: pArrayRoot, idx: n, link: block}); e != nil {
		panic(e)
	}
	if e = m.dec(ref.cell); e != nil {
		m.freeValues(block)
		m.freeCell(r)
		return e
	}
	ref.cell = r
	ref.updates = 0
	return nil
}
func (m *Int32PArrayManager) Reroot(ref *Int32PArrayRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if pArrayNull(ref.cell) {
		return ErrPArrayRef
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == pArrayRoot {
		return nil
	}
	m.rerootTmp = m.rerootTmp[:0]
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return e
	}
	r := ref.cell
	for i := uint32(0); i < n/2; i++ {
		c, e = m.cell(r)
		if e != nil {
			return e
		}
		if c.kind == pArrayRoot {
			break
		}
		if e = m.scratchPush(&m.rerootTmp, r); e != nil {
			return e
		}
		r = c.link
	}
	c, e = m.cell(r)
	if e != nil {
		return e
	}
	if c.kind != pArrayRoot {
		if e = m.unfoldCell(r); e != nil {
			return e
		}
		c, e = m.cell(r)
		if e != nil {
			return e
		}
	}
	for i := len(m.rerootTmp); i > 0; {
		i--
		p := m.rerootTmp[i]
		pc, e := m.cell(p)
		if e != nil {
			return e
		}
		size := c.idx
		block := c.link
		// Preflight before source's destructive ROOT->POP tag change. That source
		// exception continuation would otherwise interpret a values pointer as next.
		if pc.kind == pArrayPush {
			cap, e := m.capacity(block)
			if e != nil {
				return e
			}
			if uint64(size) == cap {
				if e = m.expand(&block); e != nil {
					return e
				}
				c.link = block
				if e = m.store(r, c); e != nil {
					panic(e)
				}
			}
		}
		switch pc.kind {
		case pArraySet:
			c.kind = pArraySet
			c.idx = pc.idx
			c.elem, e = m.value(block, pc.idx)
			if e != nil {
				return e
			}
			if e = m.setValue(block, pc.idx, pc.elem); e != nil {
				return e
			}
		case pArrayPush:
			c.kind = pArrayPop
			c.idx = size
			if e = m.setValue(block, size, pc.elem); e != nil {
				return e
			}
			size++
		case pArrayPop:
			if size == 0 {
				return ErrPArrayRef
			}
			size--
			c.kind = pArrayPush
			c.idx = size
			c.elem, e = m.value(block, size)
			if e != nil {
				return e
			}
		default:
			return ErrPArrayRef
		}
		pc.rc = (pc.rc + 1) & pArrayMaxRC
		if e = m.store(p, pc); e != nil {
			panic(e)
		}
		c.link = p
		if e = m.store(r, c); e != nil {
			panic(e)
		}
		if e = m.dec(r); e != nil {
			return e
		}
		pc, e = m.cell(p)
		if e != nil {
			return e
		}
		pc.kind = pArrayRoot
		pc.idx = size
		pc.link = block
		if e = m.store(p, pc); e != nil {
			panic(e)
		}
		r = p
		c = pc
	}
	ref.updates = 0
	return nil
}
func (m *Int32PArrayManager) Stats() Int32PArrayStats {
	owner := pArrayOwnerBytes
	if m.closed {
		owner = 0
	}
	return Int32PArrayStats{m.liveCells, m.liveBlocks, owner, uint64(cap(m.getTmp)+cap(m.rerootTmp)) * 8, uint64(m.liveCells) * uint64(pArrayCellBytes), m.valueBytes}
}
func (m *Int32PArrayManager) Close() error {
	if m == nil || m.closed {
		return nil
	}
	if m.liveCells != 0 {
		return ErrPArrayLive
	}
	m.budget.Release(pArrayOwnerBytes + uint64(cap(m.getTmp)+cap(m.rerootTmp))*8)
	m.getTmp = nil
	m.rerootTmp = nil
	m.closed = true
	return nil
}

// Checked unsigned calculations precede every conversion back to physical host
// int. The formatter has no invented uint32 size restriction.
func pArrayFormatCapacity(old, length, extra, limit uint64) (need, capacity uint64, err error) {
	if old > limit || length > limit || extra > limit-length {
		return 0, 0, arena.ErrBounds
	}
	need = length + extra
	capacity = old
	if need <= old {
		return need, capacity, nil
	}
	if old > limit/2 {
		return 0, 0, arena.ErrBounds
	}
	capacity = old * 2
	if capacity < 64 {
		capacity = 64
	}
	if capacity > limit {
		return 0, 0, arena.ErrBounds
	}
	for capacity < need {
		if capacity > limit/2 {
			return 0, 0, arena.ErrBounds
		}
		capacity *= 2
	}
	return need, capacity, nil
}

// Addresses become stable scalar slot/generation spellings. This caller-output
// adapter uses a budgeted lexical buffer to avoid uncharged intermediate strings.
func (m *Int32PArrayManager) DisplayInfo(ref Int32PArrayRef) (string, error) {
	if e := m.valid(&ref); e != nil {
		return "", e
	}
	if pArrayNull(ref.cell) {
		return "<null>", nil
	}
	// Formatting is a separate explicitly charged lexical buffer; its final
	// returned string belongs to the caller. Source has no retained string cache.
	var out []byte
	var charged uint64
	defer func() { m.budget.Release(charged) }()
	appendBytes := func(v []byte) error {
		need, n, e := pArrayFormatCapacity(uint64(cap(out)), uint64(len(out)), uint64(len(v)), uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if need > uint64(cap(out)) {
			if e := m.budget.Reserve(uint64(n)); e != nil {
				return e
			}
			next := make([]byte, len(out), int(n))
			copy(next, out)
			m.budget.Release(charged)
			out = next
			charged = uint64(n)
		}
		out = append(out, v...)
		return nil
	}
	var digits [32]byte
	r := ref.cell
	for {
		c, e := m.cell(r)
		if e != nil {
			return string(out), e
		}
		if e = appendBytes([]byte("cell[")); e != nil {
			return string(out), e
		}
		v := strconv.AppendUint(digits[:0], uint64(r.Slot), 10)
		if e = appendBytes(v); e != nil {
			return string(out), e
		}
		if e = appendBytes([]byte("/")); e != nil {
			return string(out), e
		}
		v = strconv.AppendUint(digits[:0], uint64(r.Generation), 10)
		if e = appendBytes(v); e != nil {
			return string(out), e
		}
		name := []string{"set, ", "push, ", "pop, ", "root, "}[c.kind]
		if e = appendBytes([]byte(", ")); e != nil {
			return string(out), e
		}
		if e = appendBytes([]byte(name)); e != nil {
			return string(out), e
		}
		v = strconv.AppendUint(digits[:0], uint64(c.idx), 10)
		if e = appendBytes(v); e != nil {
			return string(out), e
		}
		if c.kind == pArrayRoot {
			cap, e := m.capacity(c.link)
			if e != nil {
				return string(out), e
			}
			if e = appendBytes([]byte(", ")); e != nil {
				return string(out), e
			}
			v = strconv.AppendUint(digits[:0], cap, 10)
			if e = appendBytes(v); e != nil {
				return string(out), e
			}
		}
		if e = appendBytes([]byte("]#")); e != nil {
			return string(out), e
		}
		v = strconv.AppendUint(digits[:0], uint64(c.rc), 10)
		if e = appendBytes(v); e != nil {
			return string(out), e
		}
		if c.kind == pArrayRoot {
			break
		}
		if e = appendBytes([]byte(" -> ")); e != nil {
			return string(out), e
		}
		r = c.link
	}
	return string(out), nil
}
