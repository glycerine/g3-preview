// Licensed under the MIT license.
// Source symbol interning byte-input boundary used by actual AST name buffers.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"math"
	"strconv"
)

// InternBytes borrows only during this call. It preserves source strlen,
// prepare-before-duplicate, hash-prefix/CString region allocation and publication.
// A nil slice denotes an empty CString; use Symbol{} for the null constructor.
func (m *SymbolManager) InternBytes(text []byte) (Symbol, error) {
	return m.internCStringSource(utilhash.BorrowBytesCString(text))
}
func (m *SymbolManager) InternRuntimeCString(r *memory.Runtime, b memory.Block, offset, extent uint64) (Symbol, error) {
	return m.internCStringSource(utilhash.BorrowRuntimeCString(r, b, offset, extent))
}
func (m *SymbolManager) internCStringSource(text utilhash.CStringByteSource) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Symbol{}, arena.ErrClosed
	}
	length, e := text.Length()
	if e != nil {
		return Symbol{}, e
	}
	entry, found, e := m.table.PrepareFindCStringByteSource(text)
	if e != nil {
		return Symbol{}, e
	}
	if found {
		k, e := entry.Key()
		if e != nil {
			return Symbol{}, e
		}
		return m.symbol(k), nil
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
	k, e := m.table.CString(r, m.pointerWidth)
	if e != nil {
		return Symbol{}, e
	}
	_, entry, e = m.table.InsertIfNotThereCore(k)
	if e != nil {
		return Symbol{}, e
	}
	k, e = entry.Key()
	if e != nil {
		return Symbol{}, e
	}
	return m.symbol(k), nil
}

// Source string_buffer<64> keeps its first64 bytes on the stack, doubles on
// overflow and appends a NUL via c_str(). The byte interner avoids a Go string
// intermediate even for long names. All overflow arrays have lexical ownership.
func (m *ASTManager) modelValueName(sortName Symbol, index int32) (Symbol, error) {
	var initial [64]byte
	buffer := initial[:0]
	var charged uint64
	defer func() { m.budget.Release(charged) }()
	appendByte := func(v byte) error {
		if len(buffer) == cap(buffer) {
			n := uint64(cap(buffer)) * 2
			if n > uint64(^uint(0)>>1) {
				return arena.ErrBounds
			}
			if e := m.budget.Reserve(n); e != nil {
				return e
			}
			next := make([]byte, len(buffer), int(n))
			copy(next, buffer)
			m.budget.Release(charged)
			charged = n
			buffer = next
		}
		buffer = append(buffer, v)
		return nil
	}
	e := func() error {
		owner := m.symbols
		owner.mu.Lock()
		defer owner.mu.Unlock()
		if e := owner.checkPlain(sortName); e != nil {
			return e
		}
		if sortName.IsNumerical() || sortName.value == 0 {
			return ErrASTDomain
		} // source bare_str/strlen invalid continuation
		key, e := owner.key(sortName)
		if e != nil {
			return e
		}
		for i := uint32(0); i < key.Length(); i++ {
			b, e := owner.region.Read(key.Allocation(), key.Offset()+i)
			if e != nil {
				return e
			}
			if e = appendByte(b); e != nil {
				return e
			}
		}
		return nil
	}()
	if e != nil {
		return Symbol{}, e
	}
	for i := 0; i < len("!val!"); i++ {
		if e = appendByte("!val!"[i]); e != nil {
			return Symbol{}, e
		}
	}
	var digits [24]byte
	number := strconv.AppendInt(digits[:0], int64(index), 10)
	for _, v := range number {
		if e = appendByte(v); e != nil {
			return Symbol{}, e
		}
	}
	if e = appendByte(0); e != nil {
		return Symbol{}, e
	}
	return m.symbols.InternBytes(buffer)
}
