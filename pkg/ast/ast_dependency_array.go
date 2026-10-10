// Concrete actual ast_manager::dep_array_cfg specialization of parray.h.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Source capacity prefix is8 bytes. Go additionally charges a4-byte initialized
// high-water mark to distinguish retained pointers from never-written storage.
const astDependencyArrayHeaderBytes uint32 = 12

type ASTDependencyArray struct {
	origin  uint64
	cell    arena.SmallBytesRef
	updates uint32
}

type astDependencyArrayCell struct {
	rc, kind, idx uint32
	elem          arena.SmallBytesRef
	link          arena.SmallBytesRef
}

// Embedded in ASTManager: its physical header is included exactly once in the
// owner charge. Both value callbacks invoke this actual AST owner.
type astDependencyArrayManager struct {
	owner                 *ASTManager
	allocator             *arena.SmallBytes
	budget                *arena.Budget
	id                    uint64
	getTmp, rerootTmp     []arena.SmallBytesRef
	liveCells, liveBlocks uint32
	valueBytes            uint64
	preserve, closed      bool
}

func (m *astDependencyArrayManager) open() error {
	if m == nil || m.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *astDependencyArrayManager) valid(r *ASTDependencyArray) error {
	if e := m.open(); e != nil {
		return e
	}
	if r == nil {
		return ErrASTHandle
	}
	if astPArrayNull(r.cell) {
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
func (m *astDependencyArrayManager) read(r arena.SmallBytesRef, offset, width uint32) (uint64, error) {
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
func (m *astDependencyArrayManager) write(r arena.SmallBytesRef, offset, width uint32, v uint64) error {
	for i := uint32(0); i < width; i++ {
		if e := m.allocator.Write(r, offset+i, byte(v>>(8*i))); e != nil {
			return ErrASTHandle
		}
	}
	return nil
}

// Explicit 24-byte encoding: RC/kind0, index4, element8, padding12,
// link descriptor slot16/generation20. No raw address or Go pointer is encoded.
func (m *astDependencyArrayManager) cell(r arena.SmallBytesRef) (astDependencyArrayCell, error) {
	if !m.allocator.IsLive(r) {
		return astDependencyArrayCell{}, ErrASTHandle
	}
	n, e := m.allocator.Size(r)
	if e != nil || n != astPArrayCellBytes {
		return astDependencyArrayCell{}, ErrASTHandle
	}
	a, e := m.read(r, 0, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	i, e := m.read(r, 4, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	v, e := m.read(r, 8, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	vg, e := m.read(r, 12, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	s, e := m.read(r, 16, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	g, e := m.read(r, 20, 4)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	return astDependencyArrayCell{uint32(a) & astPArrayMaxRC, uint32(a) >> 30, uint32(i), arena.SmallBytesRef{Slot: arena.Slot(v), Generation: arena.Generation(vg)}, arena.SmallBytesRef{Slot: arena.Slot(s), Generation: arena.Generation(g)}}, nil
}
func (m *astDependencyArrayManager) store(r arena.SmallBytesRef, c astDependencyArrayCell) error {
	if e := m.write(r, 0, 4, uint64((c.kind<<30)|(c.rc&astPArrayMaxRC))); e != nil {
		return e
	}
	if e := m.write(r, 4, 4, uint64(c.idx)); e != nil {
		return e
	}
	if e := m.write(r, 8, 4, uint64(uint32(c.elem.Slot))); e != nil {
		return e
	}
	if e := m.write(r, 12, 4, uint64(uint32(c.elem.Generation))); e != nil {
		return e
	}
	if e := m.write(r, 16, 4, uint64(c.link.Slot)); e != nil {
		return e
	}
	return m.write(r, 20, 4, uint64(c.link.Generation))
}
func (m *astDependencyArrayManager) mkCell(kind uint32) (arena.SmallBytesRef, error) {
	r, e := m.allocator.Allocate(astPArrayCellBytes)
	if e != nil {
		return r, e
	}
	m.liveCells++
	if e = m.store(r, astDependencyArrayCell{rc: 1, kind: kind}); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *astDependencyArrayManager) freeCell(r arena.SmallBytesRef) {
	if e := m.allocator.Deallocate(astPArrayCellBytes, r); e != nil {
		panic(e)
	}
	m.liveCells--
}
func (m *astDependencyArrayManager) inc(r arena.SmallBytesRef) error {
	if astPArrayNull(r) {
		return nil
	}
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	c.rc = (c.rc + 1) & astPArrayMaxRC
	return m.store(r, c)
}
func (m *astDependencyArrayManager) dec(r arena.SmallBytesRef) error {
	var failure error
	for !astPArrayNull(r) {
		c, e := m.cell(r)
		if e != nil {
			return e
		}
		if ASTSourceDebug && c.rc == 0 {
			return ErrASTDomain
		}
		c.rc = (c.rc - 1) & astPArrayMaxRC
		if c.rc != 0 {
			if e = m.store(r, c); e != nil {
				return e
			}
			return failure
		}
		next := c.link
		if c.kind == astPArrayRoot {
			if !astPArrayNull(c.link) {
				if e = m.dropOwnedValues(c.link, c.idx); failure == nil {
					failure = e
				}
			}
			next = arena.SmallBytesRef{}
		}
		if c.kind == astPArraySet || c.kind == astPArrayPush {
			_, e = m.valueDec(c.elem)
			if failure == nil {
				failure = e
			}
		}
		m.freeCell(r)
		r = next
	}
	return failure
}
func (m *astDependencyArrayManager) capacity(r arena.SmallBytesRef) (uint64, error) {
	if astPArrayNull(r) {
		return 0, nil
	}
	return m.read(r, 0, astPArrayCapacityHeader)
}
func (m *astDependencyArrayManager) allocateValues(capacity uint64) (arena.SmallBytesRef, error) {
	if capacity > (math.MaxUint32-uint64(astDependencyArrayHeaderBytes))/8 {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	bytes := uint32(capacity*8) + astDependencyArrayHeaderBytes
	r, e := m.allocator.Allocate(bytes)
	if e != nil {
		return r, e
	}
	m.liveBlocks++
	m.valueBytes += uint64(bytes)
	if e = m.write(r, 0, astPArrayCapacityHeader, capacity); e != nil {
		panic(e)
	}
	if e = m.write(r, astPArrayCapacityHeader, 4, 0); e != nil {
		panic(e)
	}
	return r, nil
}
func (m *astDependencyArrayManager) freeValues(r arena.SmallBytesRef) {
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
func (m *astDependencyArrayManager) value(r arena.SmallBytesRef, i uint32) (arena.SmallBytesRef, error) {
	initialized, e := m.read(r, astPArrayCapacityHeader, 4)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	if uint64(i) >= initialized {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	off, e := astDependencyArrayOffset(i)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	v, e := m.read(r, off, 4)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	g, e := m.read(r, off+4, 4)
	return arena.SmallBytesRef{Slot: arena.Slot(v), Generation: arena.Generation(g)}, e
}
func (m *astDependencyArrayManager) setValue(r arena.SmallBytesRef, i uint32, v arena.SmallBytesRef) error {
	off, e := astDependencyArrayOffset(i)
	if e != nil {
		return e
	}
	if e = m.write(r, off, 4, uint64(uint32(v.Slot))); e != nil {
		return e
	}
	if e = m.write(r, off+4, 4, uint64(uint32(v.Generation))); e != nil {
		return e
	}
	initialized, e := m.read(r, astPArrayCapacityHeader, 4)
	if e != nil {
		return e
	}
	if uint64(i) >= initialized {
		return m.write(r, astPArrayCapacityHeader, 4, uint64(i)+1)
	}
	return nil
}
func (m *astDependencyArrayManager) expand(r *arena.SmallBytesRef) error {
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
	if !astPArrayNull(*r) {
		for offset := uint32(0); uint64(offset) < old*8; offset++ {
			b, e := m.read(*r, astDependencyArrayHeaderBytes+offset, 1)
			if e != nil {
				m.freeValues(next)
				return e
			}
			if e = m.write(next, astDependencyArrayHeaderBytes+offset, 1, b); e != nil {
				m.freeValues(next)
				return e
			}
		}
		initialized, e := m.read(*r, astPArrayCapacityHeader, 4)
		if e != nil {
			m.freeValues(next)
			return e
		}
		if e = m.write(next, astPArrayCapacityHeader, 4, initialized); e != nil {
			m.freeValues(next)
			return e
		}
	}
	if !astPArrayNull(*r) {
		m.freeValues(*r)
	}
	*r = next
	return nil
}
func (m *astDependencyArrayManager) rootPush(c *astDependencyArrayCell, v arena.SmallBytesRef) error {
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
	if e = m.owner.dependencies.inc(v); e != nil {
		return e
	}
	if e = m.setValue(c.link, c.idx, v); e != nil {
		return e
	}
	c.idx++
	return nil
}
func (m *astDependencyArrayManager) sizeCell(r arena.SmallBytesRef) (uint32, error) {
	for !astPArrayNull(r) {
		c, e := m.cell(r)
		if e != nil {
			return 0, e
		}
		switch c.kind {
		case astPArraySet:
			r = c.link
		case astPArrayPush:
			return c.idx + 1, nil
		case astPArrayPop:
			return c.idx - 1, nil
		case astPArrayRoot:
			return c.idx, nil
		}
	}
	return 0, nil
}
func (m *astDependencyArrayManager) Mk(out *ASTDependencyArray) error {
	if e := m.valid(out); e != nil {
		return e
	}
	consumed, err := m.decConsumed(out.cell)
	if !consumed {
		return err
	}
	*out = ASTDependencyArray{} // clear the source dangling-output exception continuation
	if err != nil {
		return err
	}
	r, e := m.mkCell(astPArrayRoot)
	if e != nil {
		return e
	}
	*out = ASTDependencyArray{origin: m.id, cell: r}
	return nil
}
func (m *astDependencyArrayManager) Del(out *ASTDependencyArray) error {
	if e := m.valid(out); e != nil {
		return e
	}
	consumed, err := m.decConsumed(out.cell)
	if consumed {
		*out = ASTDependencyArray{}
	}
	return err
}
func (m *astDependencyArrayManager) Copy(src, out *ASTDependencyArray) error {
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
	*out = ASTDependencyArray{cell: r}
	if !astPArrayNull(r) {
		out.origin = m.id
	}
	return err
}
func (m *astDependencyArrayManager) Size(ref ASTDependencyArray) (uint32, error) {
	if e := m.valid(&ref); e != nil {
		return 0, e
	}
	return m.sizeCell(ref.cell)
}
func (m *astDependencyArrayManager) Empty(ref ASTDependencyArray) (bool, error) {
	n, e := m.Size(ref)
	return n == 0, e
}
func (m *astDependencyArrayManager) Get(ref *ASTDependencyArray, index uint32) (arena.SmallBytesRef, error) {
	if e := m.valid(ref); e != nil {
		return arena.SmallBytesRef{}, e
	}
	if ASTSourceDebug {
		n, e := m.sizeCell(ref.cell)
		if e != nil {
			return arena.SmallBytesRef{}, e
		}
		if index >= n {
			return arena.SmallBytesRef{}, ErrASTDomain
		}
	}
	var e error
	r := ref.cell
	var trail uint32
	for {
		if trail > 16 {
			if e = m.Reroot(ref); e != nil {
				return arena.SmallBytesRef{}, e
			}
			c, e := m.cell(ref.cell)
			if e != nil {
				return arena.SmallBytesRef{}, e
			}
			return m.value(c.link, index)
		}
		c, e := m.cell(r)
		if e != nil {
			return arena.SmallBytesRef{}, e
		}
		if c.kind == astPArrayRoot {
			return m.value(c.link, index)
		}
		if (c.kind == astPArraySet || c.kind == astPArrayPush) && c.idx == index {
			return c.elem, nil
		}
		trail++
		r = c.link
	}
}
func (m *astDependencyArrayManager) SetFrom(src *ASTDependencyArray, i uint32, v arena.SmallBytesRef, out *ASTDependencyArray) error {
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
func (m *astDependencyArrayManager) PushBackFrom(src *ASTDependencyArray, v arena.SmallBytesRef, out *ASTDependencyArray) error {
	if src == out {
		return m.PushBack(out, v)
	}
	if e := m.Copy(src, out); e != nil {
		return e
	}
	return m.PushBack(out, v)
}
func (m *astDependencyArrayManager) PopBackFrom(src, out *ASTDependencyArray) error {
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
func (m *astDependencyArrayManager) reverseRoot(ref *ASTDependencyArray, c astDependencyArrayCell, operation, i uint32) (astDependencyArrayCell, error) {
	ref.updates++
	r, e := m.mkCell(astPArrayRoot)
	if e != nil {
		return astDependencyArrayCell{}, e
	}
	root := astDependencyArrayCell{rc: 2, kind: astPArrayRoot, idx: c.idx, link: c.link}
	if e = m.store(r, root); e != nil {
		panic(e)
	}
	switch operation {
	case astPArraySet:
		c.kind = astPArraySet
		c.idx = i
		c.elem, e = m.value(c.link, i)
	case astPArrayPush:
		c.kind = astPArrayPop
		c.idx = root.idx + 1
	case astPArrayPop:
		c.kind = astPArrayPush
		c.idx = root.idx - 1
		c.elem, e = m.value(c.link, c.idx)
	}
	if e != nil {
		m.freeCell(r)
		return astDependencyArrayCell{}, e
	}
	if c.kind == astPArraySet || c.kind == astPArrayPush {
		if e = m.owner.dependencies.inc(c.elem); e != nil {
			m.freeCell(r)
			return astDependencyArrayCell{}, e
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
func (m *astDependencyArrayManager) Set(ref *ASTDependencyArray, i uint32, v arena.SmallBytesRef) error {
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
	if c.kind == astPArrayRoot {
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
			root, e := m.reverseRoot(ref, c, astPArraySet, i)
			if e != nil {
				return e
			}
			return m.rootSet(root.link, i, v)
		}
	}
	r, e := m.mkCell(astPArraySet)
	if e != nil {
		return e
	}
	if e = m.owner.dependencies.inc(v); e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astDependencyArrayCell{rc: 1, kind: astPArraySet, idx: i, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	ref.origin = m.id
	return nil
}
func (m *astDependencyArrayManager) PushBack(ref *ASTDependencyArray, v arena.SmallBytesRef) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if astPArrayNull(ref.cell) {
		if e := m.Mk(ref); e != nil {
			return e
		}
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot {
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
			root, e := m.reverseRoot(ref, c, astPArrayPush, 0)
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
	r, e := m.mkCell(astPArrayPush)
	if e != nil {
		return e
	}
	if e = m.owner.dependencies.inc(v); e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astDependencyArrayCell{rc: 1, kind: astPArrayPush, idx: n, elem: v, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *astDependencyArrayManager) PopBack(ref *ASTDependencyArray) error {
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
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot {
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
			root, e := m.reverseRoot(ref, c, astPArrayPop, 0)
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
	r, e := m.mkCell(astPArrayPop)
	if e != nil {
		return e
	}
	if e = m.store(r, astDependencyArrayCell{rc: 1, kind: astPArrayPop, idx: n, link: ref.cell}); e != nil {
		panic(e)
	}
	ref.cell = r
	return nil
}
func (m *astDependencyArrayManager) getValues(r arena.SmallBytesRef) (block arena.SmallBytesRef, size uint32, err error) {
	m.getTmp = m.getTmp[:0]
	for {
		c, e := m.cell(r)
		if e != nil {
			return block, 0, e
		}
		if c.kind == astPArrayRoot {
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
					m.dropOwnedValues(block, i)
					return arena.SmallBytesRef{}, 0, e
				}
				// Source copy_values publishes the scalar before its INC.
				if e = m.setValue(block, i, v); e != nil {
					m.dropOwnedValues(block, i)
					return arena.SmallBytesRef{}, 0, e
				}
				if e = m.owner.dependencies.inc(v); e != nil {
					// The current scalar has not acquired ownership; only the
					// preceding successful INC prefix must be decremented.
					m.dropOwnedValues(block, i)
					return arena.SmallBytesRef{}, 0, e
				}
			}
			break
		}
		m.getTmp = append(m.getTmp, r)
		r = c.link
	}
	// The temporary block remains lexical ownership until complete publication.
	defer func() {
		if err != nil && !astPArrayNull(block) {
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
		case astPArraySet:
			err = m.rootSet(block, c.idx, c.elem)
		case astPArrayPush:
			root := astDependencyArrayCell{idx: size, link: block}
			err = m.rootPush(&root, c.elem)
			block = root.link
			size = root.idx
		case astPArrayPop:
			if size == 0 {
				return block, size, ErrASTHandle
			}
			root := astDependencyArrayCell{idx: size, link: block}
			err = m.rootPop(&root)
			size = root.idx
		}
		if err != nil {
			return block, size, err
		}
	}
	return block, size, nil
}
func (m *astDependencyArrayManager) unfoldCell(r arena.SmallBytesRef) error {
	c, e := m.cell(r)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot {
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
	if c.kind == astPArraySet || c.kind == astPArrayPush {
		_, e = m.valueDec(c.elem)
		if failure == nil {
			failure = e
		}
	}
	c.kind = astPArrayRoot
	c.idx = n
	c.link = block
	if e = m.store(r, c); e != nil {
		return e
	}
	return failure
}
func (m *astDependencyArrayManager) Unfold(ref *ASTDependencyArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if astPArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot {
		return nil
	}
	if e = m.unfoldCell(ref.cell); e != nil {
		return e
	}
	ref.updates = 0
	return nil
}
func (m *astDependencyArrayManager) Unshare(ref *ASTDependencyArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if astPArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot && c.rc == 1 {
		return nil
	}
	r, e := m.mkCell(astPArrayRoot)
	if e != nil {
		return e
	}
	block, n, e := m.getValues(ref.cell)
	if e != nil {
		m.freeCell(r)
		return e
	}
	if e = m.store(r, astDependencyArrayCell{rc: 1, kind: astPArrayRoot, idx: n, link: block}); e != nil {
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
func (m *astDependencyArrayManager) Reroot(ref *ASTDependencyArray) error {
	if e := m.valid(ref); e != nil {
		return e
	}
	if astPArrayNull(ref.cell) {
		return ErrASTHandle
	}
	c, e := m.cell(ref.cell)
	if e != nil {
		return e
	}
	if c.kind == astPArrayRoot {
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
		if c.kind == astPArrayRoot {
			break
		}
		m.rerootTmp = append(m.rerootTmp, r)
		r = c.link
	}
	c, e = m.cell(r)
	if e != nil {
		return e
	}
	if c.kind != astPArrayRoot {
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
		if pc.kind == astPArrayPush {
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
		case astPArraySet:
			c.kind = astPArraySet
			c.idx = pc.idx
			c.elem, e = m.value(block, pc.idx)
			if e != nil {
				return e
			}
			if e = m.setValue(block, pc.idx, pc.elem); e != nil {
				return e
			}
		case astPArrayPush:
			c.kind = astPArrayPop
			c.idx = size
			if e = m.setValue(block, size, pc.elem); e != nil {
				return e
			}
			size++
		case astPArrayPop:
			if size == 0 {
				return ErrASTHandle
			}
			size--
			c.kind = astPArrayPush
			c.idx = size
			c.elem, e = m.value(block, size)
			if e != nil {
				return e
			}
		default:
			return ErrASTHandle
		}
		pc.rc = (pc.rc + 1) & astPArrayMaxRC
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
		pc.kind = astPArrayRoot
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
