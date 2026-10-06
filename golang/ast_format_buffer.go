// Source: src/ast/format.cpp:147-152; src/util/string_buffer.h:62-65.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "github.com/glycerine/g3/golang/utilhash"

// FormatBuffer64 evaluates the source c_str argument before reaching mk_string.
// Its bytes remain borrowed from the actual buffer throughout symbol creation.
func (m *ASTManager) FormatBuffer64(b *StringBuffer64) (ASTHandle, error) {
	if b == nil {
		return ASTHandle{}, ErrStringBufferOwner
	}
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if e := b.state.open(b.self == b); e != nil {
		return ASTHandle{}, e
	}
	bytes, e := b.state.cstr(b.inline[:])
	if e != nil {
		return ASTHandle{}, e
	}
	f, e := m.lockFormat()
	if e != nil {
		return ASTHandle{}, e
	}
	defer f.unlockFormat()
	name, e := f.symbols.InternBytes(bytes)
	if e != nil {
		return ASTHandle{}, e
	}
	p := astParameterRecord{kind: ASTParameterSymbol, symbol: name}
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
	var name Symbol
	if b.cap == 0 {
		name, e = f.symbols.internCStringSource(utilhash.BorrowInitializedBytesCString(b.inline[:], &b.initialized))
	} else {
		name, e = f.symbols.InternRuntimeCString(m.numerals.RuntimeOwner(), b.raw, 0, uint64(b.capacity()))
	}
	if e != nil {
		return ASTHandle{}, e
	}
	p := astParameterRecord{kind: ASTParameterSymbol, symbol: name}
	r, e := f.formatApp(ASTFormatString, &p, nil)
	return f.external(r), e
}
