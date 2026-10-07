// Source: src/util/symbol.cpp. Licensed under the MIT license.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// InternBytes borrows only during this call. It preserves source strlen,
// prepare-before-duplicate, hash-prefix/CString region allocation and publication.
// A nil slice denotes an empty CString; use Symbol{} for the null constructor.
func (m *SymbolManager) InternBytes(text []byte) (Symbol, error) {
	return m.InternCStringSource(BorrowBytesCString(text))
}
func (m *SymbolManager) InternRuntimeCString(r *Runtime, b Block, offset, extent uint64) (Symbol, error) {
	return m.InternCStringSource(BorrowRuntimeCString(r, b, offset, extent))
}
func (m *SymbolManager) InternCStringSource(text CStringByteSource) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Symbol{}, arena.ErrClosed
	}
	length, e := text.Length()
	if e != nil {
		return Symbol{}, e
	}
	entry, found, e := m.Table.PrepareFindCStringByteSource(text)
	if e != nil {
		return Symbol{}, e
	}
	if found {
		k, e := entry.Key()
		if e != nil {
			return Symbol{}, e
		}
		return m.Symbol(k), nil
	}
	wide := uint64(length) + 1 + uint64(m.pointerWidth)
	if wide > math.MaxUint32 {
		return Symbol{}, arena.ErrBounds
	}
	r, e := m.region.Allocate(uint32(wide))
	if e != nil {
		return Symbol{}, e
	}
	hash, e := text.Hash()
	if e != nil {
		return Symbol{}, e
	}
	for i := uint32(0); i < m.pointerWidth; i++ {
		v := byte(0)
		if i < 4 {
			v = byte(hash >> (8 * i))
		}
		if e = m.region.Write(r, i, v); e != nil {
			return Symbol{}, e
		}
	}
	for i := uint32(0); i < length; i++ {
		v, re := text.Byte(uint64(i))
		if re != nil {
			return Symbol{}, re
		}
		if e = m.region.Write(r, m.pointerWidth+i, v); e != nil {
			return Symbol{}, e
		}
	}
	if e = m.region.Write(r, m.pointerWidth+length, 0); e != nil {
		return Symbol{}, e
	}
	k, e := m.Table.CString(r, m.pointerWidth)
	if e != nil {
		return Symbol{}, e
	}
	_, entry, e = m.Table.InsertIfNotThereCore(k)
	if e != nil {
		return Symbol{}, e
	}
	k, e = entry.Key()
	if e != nil {
		return Symbol{}, e
	}
	return m.Symbol(k), nil
}
