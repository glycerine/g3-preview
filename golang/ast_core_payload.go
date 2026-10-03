// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Scalar payload encoding. No unsafe or borrowed slice escapes.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

const astRefBytes uint32 = 8
const astSymbolBytes uint32 = 24

func astPayloadOffset(index, stride uint32) (uint32, error) {
	n := uint64(index) * uint64(stride)
	if n+uint64(stride) > math.MaxUint32 || n+uint64(stride) > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func astPayloadBytes(count, stride uint32) (uint32, error) {
	n := uint64(count) * uint64(stride)
	if n > math.MaxUint32 || n > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func (m *ASTManager) writeWord(p arena.SmallBytesRef, offset, value uint32) error {
	if uint64(offset)+4 > math.MaxUint32 {
		return arena.ErrBounds
	}
	for i := uint32(0); i < 4; i++ {
		if e := m.allocator.Write(p, offset+i, byte(value>>(8*i))); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) readWord(p arena.SmallBytesRef, offset uint32) (uint32, error) {
	if uint64(offset)+4 > math.MaxUint32 {
		return 0, arena.ErrBounds
	}
	var v uint32
	for i := uint32(0); i < 4; i++ {
		b, e := m.allocator.Read(p, offset+i)
		if e != nil {
			return 0, e
		}
		v |= uint32(b) << (8 * i)
	}
	return v, nil
}
func (m *ASTManager) writeInternal(p arena.SmallBytesRef, index uint32, r astNodeRef) error {
	offset, e := astPayloadOffset(index, astRefBytes)
	if e != nil {
		return e
	}
	if e = m.writeWord(p, offset, r.slot); e != nil {
		return e
	}
	return m.writeWord(p, offset+4, r.generation)
}
func (m *ASTManager) readInternal(p arena.SmallBytesRef, index uint32) (astNodeRef, error) {
	offset, e := astPayloadOffset(index, astRefBytes)
	if e != nil {
		return astNodeRef{}, e
	}
	s, e := m.readWord(p, offset)
	if e != nil {
		return astNodeRef{}, e
	}
	g, e := m.readWord(p, offset+4)
	return astNodeRef{s, g}, e
}
func (m *ASTManager) writeSymbol(p arena.SmallBytesRef, index uint32, s Symbol) error {
	// Symbol is local to package g3. Its scalar fields are encoded by name;
	// opaque mpnum numerals are never encoded through this representation.
	off, e := astPayloadOffset(index, astSymbolBytes)
	if e != nil {
		return e
	}
	if _, e = m.symbols.FromToken(s.Token()); e != nil {
		return e
	}
	for i, v := range [6]uint32{uint32(s.owner), uint32(s.owner >> 32), s.value, uint32(s.generation), s.tag, 0} {
		if e := m.writeWord(p, off+uint32(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}
func (m *ASTManager) readSymbol(p arena.SmallBytesRef, index uint32) (Symbol, error) {
	offset, e := astPayloadOffset(index, astSymbolBytes)
	if e != nil {
		return Symbol{}, e
	}
	var v [5]uint32
	for i := range v {
		w, e := m.readWord(p, offset+uint32(i)*4)
		if e != nil {
			return Symbol{}, e
		}
		v[i] = w
	}
	return m.symbols.FromToken(Symbol{owner: uint64(v[0]) | uint64(v[1])<<32, value: v[2], generation: arena.Generation(v[3]), tag: v[4]}.Token())
}
func (m *ASTManager) newASTPayload(hs []ASTHandle) (arena.SmallBytesRef, error) {
	if len(hs) == 0 {
		return arena.SmallBytesRef{}, nil
	}
	if uint64(len(hs)) > math.MaxUint32 {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	n, e := astPayloadBytes(uint32(len(hs)), astRefBytes)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	p, e := m.allocator.Allocate(n)
	if e != nil {
		return p, e
	}
	for i, h := range hs {
		r, e := m.internal(h)
		if e == nil {
			e = m.writeInternal(p, uint32(i), r)
		}
		if e != nil {
			m.allocator.Deallocate(n, p)
			return arena.SmallBytesRef{}, e
		}
	}
	return p, nil
}
func (m *ASTManager) newSymbolPayload(ss []Symbol) (arena.SmallBytesRef, error) {
	if len(ss) == 0 {
		return arena.SmallBytesRef{}, nil
	}
	if uint64(len(ss)) > math.MaxUint32 {
		return arena.SmallBytesRef{}, arena.ErrBounds
	}
	n, e := astPayloadBytes(uint32(len(ss)), astSymbolBytes)
	if e != nil {
		return arena.SmallBytesRef{}, e
	}
	p, e := m.allocator.Allocate(n)
	if e != nil {
		return p, e
	}
	for i, s := range ss {
		if _, e = m.symbols.Hash(s); e == nil {
			e = m.writeSymbol(p, uint32(i), s)
		}
		if e != nil {
			m.allocator.Deallocate(n, p)
			return arena.SmallBytesRef{}, e
		}
	}
	return p, nil
}
func (m *ASTManager) freePayload(p arena.SmallBytesRef, n, stride uint32) error {
	if p.Slot == 0 {
		return nil
	}
	bytes, e := astPayloadBytes(n, stride)
	if e != nil {
		return e
	}
	return m.allocator.Deallocate(bytes, p)
}
