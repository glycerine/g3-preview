// Licensed under the MIT license.
// Concrete actual ast_manager::expr_array_cfg specialization of parray.h.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Source capacity prefix is8 bytes. Go additionally charges a4-byte initialized
// high-water mark to distinguish retained pointers from never-written storage.
const astExprArrayHeaderBytes uint32 = 12

type astExprArrayCell struct {
	rc, kind, idx uint32
	elem          astNodeRef
	link          arena.SmallBytesRef
}

// Embedded in ASTManager: its physical header is included exactly once in the
// owner charge. Both value callbacks invoke this actual AST owner.
type astExprArrayManager struct {
	owner                 *ASTManager
	allocator             *arena.SmallBytes
	budget                *arena.Budget
	id                    uint64
	getTmp, rerootTmp     []arena.SmallBytesRef
	liveCells, liveBlocks uint32
	valueBytes            uint64
	preserve, closed      bool
}

func (m *astExprArrayManager) open() error {
	if m == nil || m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *astExprArrayManager) valid(r *ASTExprArray) error {
	if e := m.open(); e != nil {
		return e
	}
	if r == nil {
		return ErrASTHandle
	}
	if util.PArrayNull(r.cell) {
		if r.origin != 0 && r.origin != m.id {
			return ErrASTHandle
		}
		return nil
	}
	if r.origin != m.id || !m.allocator.IsLive(r.cell) {
		return ErrASTHandle
	}
	return nil
}
func (m *astExprArrayManager) read(r arena.SmallBytesRef, offset, width uint32) (uint64, error) {
	var v uint64
	for i := uint32(0); i < width; i++ {
		b, e := m.allocator.Read(r, offset+i)
		if e != nil {
			return 0, ErrASTHandle
		}
		v |= uint64(b) << (8 * i)
	}
	return v, nil
}
func (m *astExprArrayManager) write(r arena.SmallBytesRef, offset, width uint32, v uint64) error {
	for i := uint32(0); i < width; i++ {
		if e := m.allocator.Write(r, offset+i, byte(v>>(8*i))); e != nil {
			return ErrASTHandle
		}
	}
	return nil
}

// Explicit 24-byte encoding: RC/kind0, index4, element8, padding12,
// link descriptor slot16/generation20. No raw address or Go pointer is encoded.
func (m *astExprArrayManager) cell(r arena.SmallBytesRef) (astExprArrayCell, error) {
	if !m.allocator.IsLive(r) {
		return astExprArrayCell{}, ErrASTHandle
	}
	n, e := m.allocator.Size(r)
	if e != nil || n != util.PArrayCellBytes {
		return astExprArrayCell{}, ErrASTHandle
	}
	a, e := m.read(r, 0, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	i, e := m.read(r, 4, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	v, e := m.read(r, 8, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	vg, e := m.read(r, 12, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	s, e := m.read(r, 16, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	g, e := m.read(r, 20, 4)
	if e != nil {
		return astExprArrayCell{}, e
	}
	return astExprArrayCell{uint32(a) & util.PArrayMaxRC, uint32(a) >> 30, uint32(i), astNodeRef{uint32(v), uint32(vg)}, arena.SmallBytesRef{Slot: arena.Slot(s), Generation: arena.Generation(g)}}, nil
}
func (m *astExprArrayManager) store(r arena.SmallBytesRef, c astExprArrayCell) error {
	if e := m.write(r, 0, 4, uint64((c.kind<<30)|(c.rc&util.PArrayMaxRC))); e != nil {
		return e
	}
	if e := m.write(r, 4, 4, uint64(c.idx)); e != nil {
		return e
	}
	if e := m.write(r, 8, 4, uint64(c.elem.slot)); e != nil {
		return e
	}
	if e := m.write(r, 12, 4, uint64(c.elem.generation)); e != nil {
		return e
	}
	if e := m.write(r, 16, 4, uint64(c.link.Slot)); e != nil {
		return e
	}
	return m.write(r, 20, 4, uint64(c.link.Generation))
}
func (m *astExprArrayManager) mkCell(kind uint32) (arena.SmallBytesRef, error) {
	r, e := m.allocator.Allocate(util.PArrayCellBytes)
	if e != nil {
		return r, e
	}
	m.liveCells++
	if e = m.store(r, astExprArrayCell{rc: 1, kind: kind}); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *astExprArrayManager) freeCell(r arena.SmallBytesRef) {
	if e := m.allocator.Deallocate(util.PArrayCellBytes, r); e != nil {
		panic(e)
	}
	m.liveCells--
}
func (m *astExprArrayManager) inc(r arena.SmallBytesRef) error {
	if util.PArrayNull(r) {
		return nil
	}
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	c.rc = (c.rc + 1) & util.PArrayMaxRC
	return m.store(r, c)
}
func (m *astExprArrayManager) dec(r arena.SmallBytesRef) error {
	var failure error
	for !util.PArrayNull(r) {
		c, e := m.cell(r)
		if e != nil || c.rc == 0 {
			return ErrASTHandle
		}
		c.rc--
		if c.rc != 0 {
			if e = m.store(r, c); e != nil {
				return e
			}
			return failure
		}
		next := c.link
		if c.kind == util.PArrayRoot {
			if !util.PArrayNull(c.link) {
				if e = m.dropOwnedValues(c.link, c.idx); failure == nil {
					failure = e
				}
			}
			next = arena.SmallBytesRef{}
		}
		if c.kind == util.PArraySet || c.kind == util.PArrayPush {
			_, e = m.owner.decNode(c.elem)
			if failure == nil {
				failure = e
			}
		}
		m.freeCell(r)
		r = next
	}
	return failure
}
func (m *astExprArrayManager) capacity(r arena.SmallBytesRef) (uint64, error) {
	if util.PArrayNull(r) {
		return 0, nil
	}
	return m.read(r, 0, util.PArrayCapacityHeader)
}
func (m *astExprArrayManager) allocateValues(capacity uint64) (arena.SmallBytesRef, error) {
	if capacity > (math.MaxUint32-uint64(astExprArrayHeaderBytes))/8 {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	bytes := uint32(capacity*8) + astExprArrayHeaderBytes
	r, e := m.allocator.Allocate(bytes)
	if e != nil {
		return r, e
	}
	m.liveBlocks++
	m.valueBytes += uint64(bytes)
	if e = m.write(r, 0, util.PArrayCapacityHeader, capacity); e != nil {
		panic(e)
	}
	if e = m.write(r, util.PArrayCapacityHeader, 4, 0); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *astExprArrayManager) freeValues(r arena.SmallBytesRef) {
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
func (m *astExprArrayManager) value(r arena.SmallBytesRef, i uint32) (astNodeRef, error) {
	initialized, e := m.read(r, util.PArrayCapacityHeader, 4)
	if e != nil {
		return astNodeRef{}, e
	}
	if uint64(i) >= initialized {
		return astNodeRef{}, arena.ErrBounds
	}
	off, e := astExprArrayOffset(i)
	if e != nil {
		return astNodeRef{}, e
	}
	v, e := m.read(r, off, 4)
	if e != nil {
		return astNodeRef{}, e
	}
	g, e := m.read(r, off+4, 4)
	return astNodeRef{uint32(v), uint32(g)}, e
}
func (m *astExprArrayManager) setValue(r arena.SmallBytesRef, i uint32, v astNodeRef) error {
	off, e := astExprArrayOffset(i)
	if e != nil {
		return e
	}
	if e = m.write(r, off, 4, uint64(v.slot)); e != nil {
		return e
	}
	if e = m.write(r, off+4, 4, uint64(v.generation)); e != nil {
		return e
	}
	initialized, e := m.read(r, util.PArrayCapacityHeader, 4)
	if e != nil {
		return e
	}
	if uint64(i) >= initialized {
		return m.write(r, util.PArrayCapacityHeader, 4, uint64(i)+1)
	}
	return nil
}
func (m *astExprArrayManager) expand(r *arena.SmallBytesRef) error {
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
	// Source copies the entire old capacity, including retained/uninitialized
	// tail bytes. Copy raw encoding without inventing initialized pointers.
	if !util.PArrayNull(*r) {
		for offset := uint32(0); uint64(offset) < old*8; offset++ {
			b, e := m.read(*r, astExprArrayHeaderBytes+offset, 1)
			if e != nil {
				m.freeValues(next)
				return e
			}
			if e = m.write(next, astExprArrayHeaderBytes+offset, 1, b); e != nil {
				m.freeValues(next)
				return e
			}
		}
		initialized, e := m.read(*r, util.PArrayCapacityHeader, 4)
		if e != nil {
			m.freeValues(next)
			return e
		}
		if e = m.write(next, util.PArrayCapacityHeader, 4, initialized); e != nil {
			m.freeValues(next)
			return e
		}
	}
	if !util.PArrayNull(*r) {
		m.freeValues(*r)
	}
	*r = next
	return nil
}
func (m *astExprArrayManager) rootPush(c *astExprArrayCell, v astNodeRef) error {
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
	if e = m.owner.incNode(v); e != nil {
		return e
	}
	if e = m.setValue(c.link, c.idx, v); e != nil {
		return e
	}
	c.idx++
	return nil
}
func (m *astExprArrayManager) growScratch(a *[]arena.SmallBytesRef) error {
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
func (m *astExprArrayManager) scratchPush(a *[]arena.SmallBytesRef, r arena.SmallBytesRef) error {
	if e := m.growScratch(a); e != nil {
		return e
	}
	*a = append(*a, r)
	return nil
}
func (m *astExprArrayManager) sizeCell(r arena.SmallBytesRef) (uint32, error) {
	for !util.PArrayNull(r) {
		c, e := m.cell(r)
		if e != nil {
			return 0, e
		}
		switch c.kind {
		case util.PArraySet:
			r = c.link
		case util.PArrayPush:
			return c.idx + 1, nil
		case util.PArrayPop:
			return c.idx - 1, nil
		case util.PArrayRoot:
			return c.idx, nil
		}
	}
	return 0, nil
}
func (m *astExprArrayManager) Mk(out *ASTExprArray) error {
	if e := m.valid(out); e != nil {
		return e
	}
	consumed, err := m.decConsumed(out.cell)
	if !consumed {
		return err
	}
	*out = ASTExprArray{} // clear the source dangling-output exception continuation
	if err != nil {
		return err
	}
	r, e := m.mkCell(util.PArrayRoot)
	if e != nil {
		return e
	}
	*out = ASTExprArray{origin: m.id, cell: r}
	return nil
}
func (m *astExprArrayManager) Del(out *ASTExprArray) error {
	if e := m.valid(out); e != nil {
		return e
	}
	consumed, err := m.decConsumed(out.cell)
	if consumed {
		*out = ASTExprArray{}
	}
	return err
}
func (m *astExprArrayManager) Copy(src, out *ASTExprArray) error {
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
	consumed, err := m.decConsumed(out.cell)
	if !consumed {
		_ = m.dec(r)
		return err
	}
	*out = ASTExprArray{cell: r}
	if !util.PArrayNull(r) {
		out.origin = m.id
	}
	return err
}
func (m *astExprArrayManager) Size(ref ASTExprArray) (uint32, error) {
	if e := m.valid(&ref); e != nil {
		return 0, e
	}
	return m.sizeCell(ref.cell)
}
func (m *astExprArrayManager) Empty(ref ASTExprArray) (bool, error) {
	n, e := m.Size(ref)
	return n == 0, e
}
func (m *astExprArrayManager) Get(ref *ASTExprArray, index uint32) (astNodeRef, error) {
	if e := m.valid(ref); e != nil {
		return astNodeRef{}, e
	}
	if ASTSourceDebug {
		n, e := m.sizeCell(ref.cell)
		if e != nil {
			return astNodeRef{}, e
		}
		if index >= n {
			return astNodeRef{}, ErrASTDomain
		}
	}
	var e error
	r := ref.cell
	var trail uint32
	for {
		if trail > 16 {
			if e = m.Reroot(ref); e != nil {
				return astNodeRef{}, e
			}
			c, e := m.cell(ref.cell)
			if e != nil {
				return astNodeRef{}, e
			}
			return m.value(c.link, index)
		}
		c, e := m.cell(r)
		if e != nil {
			return astNodeRef{}, e
		}
		if c.kind == util.PArrayRoot {
			return m.value(c.link, index)
		}
		if (c.kind == util.PArraySet || c.kind == util.PArrayPush) && c.idx == index {
			return c.elem, nil
		}
		trail++
		r = c.link
	}
}
func (m *astExprArrayManager) SetFrom(src *ASTExprArray, i uint32, v astNodeRef, out *ASTExprArray) error {
	if e := m.valid(src); e != nil {
		return e
	}
	if ASTSourceDebug {
		n, e := m.Size(*src)
		if e != nil {
			return e
		}
		if i >= n {
			return ErrASTDomain
		}
	}
	if src == out {
		return m.Set(out, i, v)
	}
	if e := m.Copy(src, out); e != nil {
		return e
	}
	return m.Set(out, i, v)
}
func (m *astExprArrayManager) PushBackFrom(src *ASTExprArray, v astNodeRef, out *ASTExprArray) error {
	if src == out {
		return m.PushBack(out, v)
	}
	if e := m.Copy(src, out); e != nil {
		return e
	}
	return m.PushBack(out, v)
}
func (m *astExprArrayManager) PopBackFrom(src, out *ASTExprArray) error {
	if e := m.valid(src); e != nil {
		return e
	}
	n, e := m.Size(*src)
	if e != nil {
		return e
	}
	if ASTSourceDebug && n == 0 {
		return ErrASTDomain
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
func (m *astExprArrayManager) reverseRoot(ref *ASTExprArray, c astExprArrayCell, operation, i uint32) (astExprArrayCell, error) {
	ref.updates++
	r, e := m.mkCell(util.PArrayRoot)
	if e != nil {
		return astExprArrayCell{}, e
	}
	root := astExprArrayCell{rc: 2, kind: util.PArrayRoot, idx: c.idx, link: c.link}
	if e = m.store(r, root); e != nil {
		panic(e)
	}
	switch operation {
	case util.PArraySet:
		c.kind = util.PArraySet
		c.idx = i
		c.elem, e = m.value(c.link, i)
	case util.PArrayPush:
		c.kind = util.PArrayPop
		c.idx = root.idx + 1
	case util.PArrayPop:
		c.kind = util.PArrayPush
		c.idx = root.idx - 1
		c.elem, e = m.value(c.link, c.idx)
	}
	if e != nil {
		m.freeCell(r)
		return astExprArrayCell{}, e
	}
	if c.kind == util.PArraySet || c.kind == util.PArrayPush {
		if e = m.owner.incNode(c.elem); e != nil {
			m.freeCell(r)
			return astExprArrayCell{}, e
		}
	}
	c.link = r
	c.rc--
	if e = m.store(ref.cell, c); e != nil {
		panic(e)
	}
	ref.cell = r
	return root, nil
}
func (m *astExprArrayManager) Set(ref *ASTExprArray, i uint32, v astNodeRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if ASTSourceDebug {
		n, e := m.sizeCell(ref.cell)
		if e != nil {
			return e
		}
		if i >= n {
			return ErrASTDomain
		}
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
		if c.rc == 1 {
			return m.rootSet(c.link, i, v)
		}
		if m.preserve {
			if ref.updates > c.idx {
				if e = m.Unshare(ref); e != nil {
					return e
				}
				return m.Set(ref, i, v)
			}
			root, e := m.reverseRoot(ref, c, util.PArraySet, i)
			if e != nil {
				return e
			}
			return m.rootSet(root.link, i, v)
		}
	}
	r, e := m.mkCell(util.PArraySet)
	if e != nil {
		return e
	}
	if e = m.owner.incNode(v); e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astExprArrayCell{rc: 1, kind: util.PArraySet, idx: i, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	ref.origin = m.id
	return nil
}
func (m *astExprArrayManager) PushBack(ref *ASTExprArray, v astNodeRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if util.PArrayNull(ref.cell) {
		if e := m.Mk(ref); e != nil {
			return e
		}
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
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
			root, e := m.reverseRoot(ref, c, util.PArrayPush, 0)
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
	r, e := m.mkCell(util.PArrayPush)
	if e != nil {
		return e
	}
	if e = m.owner.incNode(v); e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astExprArrayCell{rc: 1, kind: util.PArrayPush, idx: n, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *astExprArrayManager) PopBack(ref *ASTExprArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	n, e := m.sizeCell(ref.cell)
	if e != nil {
		return e
	}
	if ASTSourceDebug && n == 0 {
		return ErrASTDomain
	}
	if util.PArrayNull(ref.cell) {
		return arena.ErrBounds
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
		if n == 0 {
			return arena.ErrBounds
		}
		if c.rc == 1 {
			err := m.rootPop(&c)
			if e = m.store(ref.cell, c); e != nil {
				return e
			}
			return err
		}
		if m.preserve {
			if ref.updates > n {
				if e = m.Unshare(ref); e != nil {
					return e
				}
				return m.PopBack(ref)
			}
			root, e := m.reverseRoot(ref, c, util.PArrayPop, 0)
			if e != nil {
				return e
			}
			err := m.rootPop(&root)
			if e = m.store(ref.cell, root); e != nil {
				return e
			}
			return err
		}
	}
	r, e := m.mkCell(util.PArrayPop)
	if e != nil {
		return e
	}
	if e = m.store(r, astExprArrayCell{rc: 1, kind: util.PArrayPop, idx: n, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *astExprArrayManager) getValues(r arena.SmallBytesRef) (block arena.SmallBytesRef, size uint32, err error) {
	m.getTmp = m.getTmp[:0]
	for {
		c, e := m.cell(r)
		if e != nil {
			return block, 0, e
		}
		if c.kind == util.PArrayRoot {
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
				if e = m.owner.incNode(v); e != nil {
					m.dropOwnedValues(block, i)
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
		if err != nil && !util.PArrayNull(block) {
			m.dropOwnedValues(block, size)
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
		case util.PArraySet:
			err = m.rootSet(block, c.idx, c.elem)
		case util.PArrayPush:
			root := astExprArrayCell{idx: size, link: block}
			err = m.rootPush(&root, c.elem)
			block = root.link
			size = root.idx
		case util.PArrayPop:
			if size == 0 {
				return block, size, ErrASTHandle
			}
			root := astExprArrayCell{idx: size, link: block}
			err = m.rootPop(&root)
			size = root.idx
		}
		if err != nil {
			return block, size, err
		}
	}
	return block, size, nil
}
func (m *astExprArrayManager) unfoldCell(r arena.SmallBytesRef) error {
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
		return nil
	}
	block, n, e := m.getValues(r)
	if e != nil {
		return e
	}
	consumed, failure := m.decConsumed(c.link)
	if !consumed {
		m.dropOwnedValues(block, n)
		return failure
	}
	if c.kind == util.PArraySet || c.kind == util.PArrayPush {
		_, e = m.owner.decNode(c.elem)
		if failure == nil {
			failure = e
		}
	}
	c.kind = util.PArrayRoot
	c.idx = n
	c.link = block
	if e = m.store(r, c); e != nil {
		return e
	}
	return failure
}
func (m *astExprArrayManager) Unfold(ref *ASTExprArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if util.PArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
		return nil
	}
	if e = m.unfoldCell(ref.cell); e != nil {
		return e
	}
	ref.updates = 0
	return nil
}
func (m *astExprArrayManager) Unshare(ref *ASTExprArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if util.PArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot && c.rc == 1 {
		return nil
	}
	r, e := m.mkCell(util.PArrayRoot)
	if e != nil {
		return e
	}
	block, n, e := m.getValues(ref.cell)
	if e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astExprArrayCell{rc: 1, kind: util.PArrayRoot, idx: n, link: block}); e != nil {
		panic(e)
	}
	consumed, err := m.decConsumed(ref.cell)
	if !consumed {
		m.dropOwnedValues(block, n)
		m.freeCell(r)
		return err
	}
	ref.cell = r
	ref.updates = 0
	return err
}
func (m *astExprArrayManager) Reroot(ref *ASTExprArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if util.PArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == util.PArrayRoot {
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
		if c.kind == util.PArrayRoot {
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
	if c.kind != util.PArrayRoot {
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
		if pc.kind == util.PArrayPush {
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
		case util.PArraySet:
			c.kind = util.PArraySet
			c.idx = pc.idx
			c.elem, e = m.value(block, pc.idx)
			if e != nil {
				return e
			}
			if e = m.setValue(block, pc.idx, pc.elem); e != nil {
				return e
			}
		case util.PArrayPush:
			c.kind = util.PArrayPop
			c.idx = size
			if e = m.setValue(block, size, pc.elem); e != nil {
				return e
			}
			size++
		case util.PArrayPop:
			if size == 0 {
				return ErrASTHandle
			}
			size--
			c.kind = util.PArrayPush
			c.idx = size
			c.elem, e = m.value(block, size)
			if e != nil {
				return e
			}
		default:
			return ErrASTHandle
		}
		pc.rc = (pc.rc + 1) & util.PArrayMaxRC
		if e = m.store(p, pc); e != nil {
			panic(e)
		}
		c.link = p
		if e = m.store(r, c); e != nil {
			panic(e)
		}
		consumed, releaseErr := m.decConsumed(r)
		if !consumed {
			return releaseErr
		}
		pc, e = m.cell(p)
		if e != nil {
			return e
		}
		pc.kind = util.PArrayRoot
		pc.idx = size
		pc.link = block
		if e = m.store(p, pc); e != nil {
			panic(e)
		}
		r = p
		c = pc
		if releaseErr != nil {
			return releaseErr
		}
	}
	ref.updates = 0
	return nil
}
