// Source: src/util/obj_hashtable.h:58-105,144-159;
// src/util/hashtable.h:155-227,244-256,272-274,286-308,377-396,494-510;
// src/ast/rewriter/poly_rewriter.h:32; src/ast/rewriter/poly_rewriter_def.h:570-586.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

// Concrete specialization for m_expr2pos. The polynomial arguments own the
// expressions; this table never acquires or releases expression references.
// Its consumer uses no erase operation, so there are no deleted buckets.
type bvPolynomialPositionEntry struct {
	key      ast.ASTHandle
	position uint32
}

type bvPolynomialPositions struct {
	manager         *ast.ASTManager
	entries         []bvPolynomialPositionEntry
	capacity, size  uint32
	header, bytes   uint64
	stopped, closed bool
}

func bvPolynomialPositionEntries(m *ast.ASTManager, capacity uint32) ([]bvPolynomialPositionEntry, uint64, error) {
	bytes := uint64(capacity) * uint64(reflect.TypeOf(bvPolynomialPositionEntry{}).Size())
	if capacity == 0 || bytes > uint64(^uint(0)>>1) {
		return nil, 0, arena.ErrBounds
	}
	if e := m.Budget().Reserve(bytes); e != nil {
		return nil, 0, e
	}
	return make([]bvPolynomialPositionEntry, capacity), bytes, nil
}

func newBVPolynomialPositions(m *ast.ASTManager) (*bvPolynomialPositions, error) {
	header := uint64(reflect.TypeOf(bvPolynomialPositions{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	entries, bytes, e := bvPolynomialPositionEntries(m, 8)
	if e != nil {
		m.Budget().Release(header)
		return nil, e
	}
	return &bvPolynomialPositions{manager: m, entries: entries, capacity: 8, header: header, bytes: bytes}, nil
}

func (p *bvPolynomialPositions) ready() error {
	if p == nil || p.closed || p.stopped {
		return ast.ErrASTPending
	}
	return nil
}

func (p *bvPolynomialPositions) find(key ast.ASTHandle) (uint32, bool, error) {
	if e := p.ready(); e != nil {
		return 0, false, e
	}
	hash, e := p.manager.GetHash(key)
	if e != nil {
		return 0, false, e
	}
	for offset := uint32(0); offset < p.capacity; offset++ {
		entry := p.entries[(hash+offset)&(p.capacity-1)]
		if entry.key.IsNull() {
			return 0, false, nil
		}
		stored, e := p.manager.GetHash(entry.key)
		if e != nil {
			return 0, false, e
		}
		if stored == hash && entry.key == key {
			return entry.position, true, nil
		}
	}
	return 0, false, nil
}

func (p *bvPolynomialPositions) insert(key ast.ASTHandle, position uint32) error {
	if e := p.ready(); e != nil {
		return e
	}
	// Original threshold and ordering: expansion happens before duplicate lookup.
	if p.size<<2 > p.capacity*3 {
		capacity := p.capacity << 1
		entries, bytes, e := bvPolynomialPositionEntries(p.manager, capacity)
		if e != nil {
			return e
		}
		for _, entry := range p.entries {
			if entry.key.IsNull() {
				continue
			}
			hash, e := p.manager.GetHash(entry.key)
			if e != nil {
				p.manager.Budget().Release(bytes)
				return e
			}
			index := hash & (capacity - 1)
			for !entries[index].key.IsNull() {
				index = (index + 1) & (capacity - 1)
			}
			entries[index] = entry
		}
		p.manager.Budget().Release(p.bytes)
		p.entries, p.bytes, p.capacity = entries, bytes, capacity
	}
	hash, e := p.manager.GetHash(key)
	if e != nil {
		return e
	}
	for offset := uint32(0); offset < p.capacity; offset++ {
		entry := &p.entries[(hash+offset)&(p.capacity-1)]
		if entry.key.IsNull() {
			*entry = bvPolynomialPositionEntry{key: key, position: position}
			p.size++
			return nil
		}
		stored, e := p.manager.GetHash(entry.key)
		if e != nil {
			return e
		}
		if stored == hash && entry.key == key {
			entry.position = position
			return nil
		}
	}
	return ast.ErrASTDomain
}

func (p *bvPolynomialPositions) reset() error {
	if e := p.ready(); e != nil {
		return e
	}
	if p.size == 0 {
		return nil
	}
	var overhead uint32
	for i := range p.entries {
		if p.entries[i].key.IsNull() {
			overhead++
		} else {
			p.entries[i].key = ast.ASTHandle{}
		}
	}
	if p.capacity > 16 && overhead<<2 > p.capacity*3 {
		// delete_table precedes the replacement allocation in the source.
		p.manager.Budget().Release(p.bytes)
		p.entries, p.bytes = nil, 0
		p.capacity >>= 1
		entries, bytes, e := bvPolynomialPositionEntries(p.manager, p.capacity)
		if e != nil {
			p.stopped = true
			return e
		}
		p.entries, p.bytes = entries, bytes
	}
	p.size = 0
	return nil
}

func (p *bvPolynomialPositions) close() {
	if p == nil || p.closed {
		return
	}
	p.manager.Budget().Release(p.bytes)
	p.manager.Budget().Release(p.header)
	p.entries = nil
	p.bytes, p.header = 0, 0
	p.closed = true
}
