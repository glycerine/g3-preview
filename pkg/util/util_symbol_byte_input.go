// Source: src/util/symbol.cpp.
package util

import "github.com/glycerine/g3/pkg/arena"

func (m *SymbolManager) InternBytes(text []byte) (Symbol, error) {
	for i, b := range text {
		if b == 0 {
			text = text[:i]
			break
		}
	}
	return m.Intern(string(text))
}

// InternInitializedBytes reads a small inline C string while preserving the
// runtime initialization checks of its producer.
func (m *SymbolManager) InternInitializedBytes(text []byte, initialized uint16) (Symbol, error) {
	for i, b := range text {
		if i >= 16 {
			return Symbol{}, arena.ErrBounds
		}
		if initialized&(uint16(1)<<i) == 0 {
			return Symbol{}, ErrRuntimeUninitialized
		}
		if b == 0 {
			return m.Intern(string(text[:i]))
		}
	}
	return Symbol{}, arena.ErrBounds
}

func (m *SymbolManager) InternRuntimeCString(r *Runtime, block Block, offset, extent uint64) (Symbol, error) {
	if r == nil {
		return Symbol{}, ErrRuntimeOwner
	}
	text := make([]byte, 0, 32)
	for i := uint64(0); i < extent; i++ {
		if offset+i < offset {
			return Symbol{}, arena.ErrBounds
		}
		b, e := r.Read(block, offset+i)
		if e != nil {
			return Symbol{}, e
		}
		if b == 0 {
			return m.Intern(string(text))
		}
		text = append(text, b)
	}
	return Symbol{}, arena.ErrBounds
}
