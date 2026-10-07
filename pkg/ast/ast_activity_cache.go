// Source: src/ast/act_cache.cpp:56-225; src/ast/act_cache.h:23-53;
// src/util/chashtable.h:58-266,312-348,406-419,443-479,484-522;
// src/ast/ast.h:1598; src/util/vector.h:35-143.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Concrete cmap<expr*, tagged expr*> specialization. Encoded link indices and
// an explicit activity bit replace source pointers/tags. Cached AST hashes are
// immutable. Table iteration remains ascending buckets, then head-first chains.
type astActivityCell struct {
	key, value                          ASTHandle
	hash, next                          uint32
	accessed, dummy, keyHeld, valueHeld bool
}
type astActivityTable struct {
	cells                                     []astActivityCell
	slots, next, free, size, used, collisions uint32
}

func activityTableBytes(n uint32) uint64 {
	return uint64(n) * uint64(reflect.TypeOf(astActivityCell{}).Size())
}
func activityCells(m *ASTManager, slots, cellar uint32) ([]astActivityCell, error) {
	n := uint64(slots) + uint64(cellar)
	if slots == 0 || slots&(slots-1) != 0 || cellar == 0 || n >= uint64(util.ChainEnd) || n > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(astActivityCell{}).Size()) {
		return nil, arena.ErrBounds
	}
	if e := m.BudgetField.Reserve(activityTableBytes(uint32(n))); e != nil {
		return nil, e
	}
	cells := make([]astActivityCell, int(n))
	for i := range cells {
		cells[i].next = util.ChainFree
	}
	return cells, nil
}
func (t *astActivityTable) init(m *ASTManager) error {
	cells, e := activityCells(m, 8, 2)
	if e != nil {
		return e
	}
	*t = astActivityTable{cells: cells, slots: 8, next: 8, free: util.ChainEnd, collisions: t.collisions}
	return nil
}
func (t *astActivityTable) find(k ASTHandle, hash uint32) (uint32, bool) {
	if t.slots == 0 {
		return util.ChainEnd, false
	}
	i := hash & (t.slots - 1)
	if t.cells[i].next == util.ChainFree {
		return util.ChainEnd, false
	}
	for ; i != util.ChainEnd; i = t.cells[i].next {
		if t.cells[i].key == k {
			return i, true
		}
		t.collisions++
	}
	return util.ChainEnd, false
}
func (t *astActivityTable) expand(m *ASTManager) error {
	slots, cellar := uint64(t.slots)*2, uint64(len(t.cells)-int(t.slots))*2
	for {
		if slots >= uint64(util.ChainEnd) || cellar >= uint64(util.ChainEnd) {
			return arena.ErrBounds
		}
		cells, e := activityCells(m, uint32(slots), uint32(cellar))
		if e != nil {
			return e
		}
		next := uint32(slots)
		t.used = 0
		fits := true
		for b := uint32(0); b < t.slots && fits; b++ {
			if t.cells[b].next == util.ChainFree {
				continue
			}
			for i := b; i != util.ChainEnd; i = t.cells[i].next {
				idx := t.cells[i].hash & (uint32(slots) - 1)
				if cells[idx].next == util.ChainFree {
					cells[idx] = t.cells[i]
					cells[idx].next = util.ChainEnd
					t.used++
				} else {
					if next == uint32(len(cells)) {
						fits = false
						break
					}
					cells[next] = cells[idx]
					cells[idx] = t.cells[i]
					cells[idx].next = next
					next++
				}
			}
		}
		if fits {
			m.BudgetField.Release(activityTableBytes(uint32(len(t.cells))))
			t.cells = cells
			t.slots = uint32(slots)
			t.next = next
			t.free = util.ChainEnd
			return nil
		}
		m.BudgetField.Release(activityTableBytes(uint32(len(cells))))
		cellar *= 2
	}
}
func (t *astActivityTable) insertDummy(m *ASTManager, k ASTHandle, hash uint32) (uint32, error) {
	// Source grows before checking whether this key is already present.
	if t.free == util.ChainEnd && t.next == uint32(len(t.cells)) {
		if e := t.expand(m); e != nil {
			return util.ChainEnd, e
		}
	}
	if i, found := t.find(k, hash); found {
		return i, nil
	}
	idx := hash & (t.slots - 1)
	entry := astActivityCell{key: k, hash: hash, next: util.ChainEnd, dummy: true}
	if t.cells[idx].next == util.ChainFree {
		t.used++
	} else {
		j := t.free
		if j != util.ChainEnd {
			t.free = t.cells[j].next
		} else {
			j = t.next
			t.next++
		}
		t.cells[j] = t.cells[idx]
		entry.next = j
	}
	t.cells[idx] = entry
	t.size++
	return idx, nil
}
func (t *astActivityTable) erase(k ASTHandle, hash uint32) {
	i := hash & (t.slots - 1)
	if t.cells[i].next == util.ChainFree {
		return
	}
	prev := uint32(util.ChainEnd)
	for i != util.ChainEnd {
		c := t.cells[i]
		if c.key == k {
			t.size--
			if prev == util.ChainEnd {
				if c.next == util.ChainEnd {
					t.used--
					t.cells[i].next = util.ChainFree
				} else {
					t.cells[i] = t.cells[c.next]
					t.cells[c.next].next = t.free
					t.free = c.next
				}
			} else {
				t.cells[prev].next = c.next
				t.cells[i].next = t.free
				t.free = i
			}
			return
		}
		t.collisions++
		prev = i
		i = c.next
	}
}
func (t *astActivityTable) release(m *ASTManager) {
	m.BudgetField.Release(activityTableBytes(uint32(len(t.cells))))
	*t = astActivityTable{free: util.ChainEnd, collisions: t.collisions}
}

// The cache is an owning cold object borrowing its AST manager. The queue is
// the source ptr_vector (initial capacity2, growth ceil(3*capacity/2)), not an
// inline buffer. Its handles borrow the references owned by live map entries.
type ActivityCache struct {
	self                     *ActivityCache
	manager                  *ASTManager
	table                    astActivityTable
	queue                    []ASTHandle
	qhead, unused, maxUnused uint32
	header, queueBytes       uint64
	retiring                 astActivityCell
	incoming                 ASTHandle
	incomingHeld             bool
	pending                  bool
	cause                    error
	closed                   bool
}

func NewActivityCache(m *ASTManager, maxUnused *uint32) (*ActivityCache, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if e := m.lock(); e != nil {
		return nil, e
	}
	n := m.table.size
	m.mu.Unlock()
	if maxUnused != nil {
		n = *maxUnused
	}
	if n < 1024 {
		n = 1024
	}
	bytes := uint64(reflect.TypeOf(ActivityCache{}).Size())
	if e := m.BudgetField.Reserve(bytes); e != nil {
		return nil, e
	}
	c := &ActivityCache{manager: m, maxUnused: n, header: bytes}
	c.self = c
	if e := c.table.init(m); e != nil {
		m.BudgetField.Release(bytes)
		return nil, e
	}
	return c, nil
}
func (c *ActivityCache) open() error {
	if c == nil || c.self != c || c.closed {
		return ErrASTHandle
	}
	if c.cause != nil {
		return c.cause
	}
	_, e := c.manager.ProofsEnabled()
	return e
}
func (c *ActivityCache) pushQueue(k ASTHandle) error {
	if len(c.queue) == cap(c.queue) {
		n := uint64(2)
		if cap(c.queue) != 0 {
			n = (3*uint64(cap(c.queue)) + 1) >> 1
		}
		width := uint64(reflect.TypeOf(ASTHandle{}).Size())
		if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := c.manager.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		q := make([]ASTHandle, len(c.queue), int(n))
		copy(q, c.queue)
		c.manager.BudgetField.Release(c.queueBytes)
		c.queue = q
		c.queueBytes = bytes
	}
	c.queue = append(c.queue, k)
	return nil
}
func (c *ActivityCache) dec(h ASTHandle) (bool, error) {
	m := c.manager
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	if h.IsNull() {
		return true, nil
	}
	r, e := m.internal(h)
	if e != nil {
		return false, e
	}
	return m.decNode(r)
}
func (c *ActivityCache) retire(entry *astActivityCell) error {
	if entry.keyHeld {
		consumed, e := c.dec(entry.key)
		if consumed {
			entry.keyHeld = false
		}
		if e != nil {
			return e
		}
	}
	if entry.valueHeld {
		consumed, e := c.dec(entry.value)
		if consumed {
			entry.valueHeld = false
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (c *ActivityCache) finishPending() error {
	if !c.pending {
		return nil
	}
	if e := c.retire(&c.retiring); e != nil {
		return e
	}
	c.pending = false
	c.retiring = astActivityCell{}
	return nil
}
func (c *ActivityCache) deleteUnused() error {
	size := uint32(len(c.queue))
	for c.qhead < size {
		k := c.queue[c.qhead]
		c.qhead++
		hash, e := c.manager.GetHash(k)
		if e != nil {
			return e
		}
		i, found := c.table.find(k, hash)
		if !found {
			return ErrASTDomain
		}
		if !c.table.cells[i].accessed {
			c.unused--
			c.retiring = c.table.cells[i]
			c.pending = true
			c.table.erase(k, hash)
			if e = c.finishPending(); e != nil {
				return e
			}
			break
		}
	}
	if c.qhead == size {
		c.queue = c.queue[:0]
		c.qhead = 0
	} else if c.qhead > c.maxUnused {
		copy(c.queue, c.queue[c.qhead:])
		c.queue = c.queue[:size-c.qhead]
		c.qhead = 0
	}
	return nil
}
func (c *ActivityCache) Insert(k, v ASTHandle) (err error) {
	if e := c.open(); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if k.IsNull() {
		return ErrASTDomain
	}
	// Checked handles replace invalid source pointers, without changing the
	// statement order for valid expressions.
	m := c.manager
	if _, e := m.GetSort(k); e != nil {
		return e
	}
	if !v.IsNull() {
		if _, e := m.GetSort(v); e != nil {
			return e
		}
	}
	if c.unused >= c.maxUnused {
		if e := c.deleteUnused(); e != nil {
			return e
		}
	}
	hash, e := m.GetHash(k)
	if e != nil {
		return e
	}
	i, e := c.table.insertDummy(m, k, hash)
	if e != nil {
		return e
	}
	entry := &c.table.cells[i]
	if entry.dummy {
		if e = m.IncRef(k); e != nil {
			return e
		}
		entry.keyHeld = true
		if e = m.IncRef(v); e != nil {
			return e
		}
		entry.valueHeld = !v.IsNull()
		entry.value = v
		entry.dummy = false
		if e = c.pushQueue(k); e != nil {
			return e
		}
		c.unused++
	} else if entry.value != v {
		if e = m.IncRef(v); e != nil {
			return e
		}
		// Publish after the old-value DEC, preserving source statement order.
		c.incoming, c.incomingHeld = v, !v.IsNull()
		consumed, e := c.dec(entry.value)
		if consumed {
			entry.valueHeld = false
		}
		if e != nil {
			return e
		}
		entry.value, entry.valueHeld = v, c.incomingHeld
		c.incoming, c.incomingHeld = ASTHandle{}, false
		entry.accessed = false
		// Preserve the source quirk: replacing an accessed entry does not increment
		// unused and does not append its key to the queue.
	}
	return nil
}
func (c *ActivityCache) Find(k ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	if k.IsNull() {
		return ASTHandle{}, ErrASTDomain
	}
	hash, e := c.manager.GetHash(k)
	if e != nil {
		return ASTHandle{}, e
	}
	i, found := c.table.find(k, hash)
	if !found {
		return ASTHandle{}, nil
	}
	entry := &c.table.cells[i]
	if !entry.accessed {
		entry.accessed = true
		if ASTSourceDebug && c.unused == 0 {
			return ASTHandle{}, ErrASTDomain
		}
		c.unused--
	}
	return entry.value, nil
}
func (c *ActivityCache) decRefs() error {
	if c.incomingHeld {
		consumed, e := c.dec(c.incoming)
		if consumed {
			c.incomingHeld = false
			c.incoming = ASTHandle{}
		}
		if e != nil {
			return e
		}
	}
	if e := c.finishPending(); e != nil {
		return e
	}
	for b := uint32(0); b < c.table.slots; b++ {
		if c.table.cells[b].next == util.ChainFree {
			continue
		}
		for i := b; i != util.ChainEnd; i = c.table.cells[i].next {
			if e := c.retire(&c.table.cells[i]); e != nil {
				return e
			}
		}
	}
	return nil
}
func (c *ActivityCache) clear(cleanup bool) (err error) {
	if c == nil || c.self != c || c.closed {
		return ErrASTHandle
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if e := c.decRefs(); e != nil {
		return e
	}
	if cleanup || c.table.size != 0 || c.table.slots == 0 {
		// Source finalize deletes the old array before reinitializing at8+2.
		c.table.release(c.manager)
		if e := c.table.init(c.manager); e != nil {
			return e
		}
	}
	c.queue = c.queue[:0]
	if cleanup {
		c.manager.BudgetField.Release(c.queueBytes)
		c.queue = nil
		c.queueBytes = 0
	}
	c.unused = 0
	c.qhead = 0
	c.cause = nil
	return nil
}
func (c *ActivityCache) Reset() error         { return c.clear(false) }
func (c *ActivityCache) Cleanup() error       { return c.clear(true) }
func (c *ActivityCache) CheckInvariant() bool { return true }
func (c *ActivityCache) Size() (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	return c.table.size, nil
}
func (c *ActivityCache) Capacity() (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	return uint32(len(c.table.cells)), nil
}
func (c *ActivityCache) Empty() (bool, error) {
	n, e := c.Size()
	return n == 0, e
}
func (c *ActivityCache) Close() error {
	if c == nil || c.self != c {
		return ErrASTHandle
	}
	if c.closed {
		return nil
	}
	if e := c.decRefs(); e != nil {
		return e
	}
	c.table.release(c.manager)
	c.manager.BudgetField.Release(c.queueBytes + c.header)
	c.queue = nil
	c.queueBytes = 0
	c.header = 0
	c.closed = true
	return nil
}
