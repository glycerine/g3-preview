// Source: src/ast/format.cpp:147-152; src/util/string_buffer.h:62-65.
package ast

import (
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// FormatBuffer64 evaluates the source c_str argument before reaching mk_string.
// Its bytes remain borrowed from the actual buffer throughout symbol creation.
func (m *ASTManager) FormatBuffer64(b *util.StringBuffer64) (ASTHandle, error) {
	if b == nil {
		return ASTHandle{}, util.ErrStringBufferOwner
	}
	text, e := b.CStr()
	if e != nil {
		return ASTHandle{}, e
	}
	f, e := m.lockFormat()
	if e != nil {
		return ASTHandle{}, e
	}
	defer f.unlockFormat()
	name, e := f.Symbols.InternBytes([]byte(text))
	if e != nil {
		return ASTHandle{}, e
	}
	p := astParameterRecord{Kind: ASTParameterSymbol, symbol: name}
	r, e := f.formatApp(ASTFormatString, &p, nil)
	return f.external(r), e
}

// Source: src/ast/ast_smt2_pp.cpp:226-229; src/ast/format.cpp:147-152.
// Capture the actual sbuffer<char> pointer argument without an assembly copy.
func (m *ASTManager) formatBVCharacters(b *astBVCharBuffer16) (ASTHandle, error) {
	if b == nil || b.sourceDone || b.rawConsumed {
		return ASTHandle{}, ErrASTHandle
	}
	f, e := m.lockFormat()
	if e != nil {
		return ASTHandle{}, e
	}
	defer f.unlockFormat()
	var name usymbol.Symbol
	if b.cap == 0 {
		name, e = util.InternSymbolInitializedBytes(f.Symbols, b.inline[:], b.initialized)
	} else {
		name, e = util.InternSymbolRuntimeCString(f.Symbols, m.numerals.RuntimeOwner(), b.raw, 0, uint64(b.capacity()))
	}
	if e != nil {
		return ASTHandle{}, e
	}
	p := astParameterRecord{Kind: ASTParameterSymbol, symbol: name}
	r, e := f.formatApp(ASTFormatString, &p, nil)
	return f.external(r), e
}
