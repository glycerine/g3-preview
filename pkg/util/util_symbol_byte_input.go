// Source: src/util/symbol.cpp.
package util

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	// InternInitializedBytes reads a small inline C string while preserving the
	// runtime initialization checks of its producer.
)

import "github.com/glycerine/g3/pkg/arena"

func InternSymbolInitializedBytes(m *usymbol.Manager, text []byte, initialized uint16) (usymbol.Symbol, error) {
	for i, b := range text {
		if i >= 16 {
			return usymbol.Symbol{}, arena.ErrBounds
		}
		if initialized&(uint16(1)<<i) == 0 {
			return usymbol.Symbol{}, umemory.ErrRuntimeUninitialized
		}
		if b == 0 {
			return m.Intern(string(text[:i]))
		}
	}
	return usymbol.Symbol{}, arena.ErrBounds
}

func InternSymbolRuntimeCString(m *usymbol.Manager, r *umemory.Runtime, block umemory.Block, offset, extent uint64) (usymbol.Symbol, error) {
	if r == nil {
		return usymbol.Symbol{}, umemory.ErrRuntimeOwner
	}
	text := make([]byte, 0, 32)
	for i := uint64(0); i < extent; i++ {
		if offset+i < offset {
			return usymbol.Symbol{}, arena.ErrBounds
		}
		b, e := r.Read(block, offset+i)
		if e != nil {
			return usymbol.Symbol{}, e
		}
		if b == 0 {
			return m.Intern(string(text))
		}
		text = append(text, b)
	}
	return usymbol.Symbol{}, arena.ErrBounds
}
