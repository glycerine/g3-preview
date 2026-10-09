// Source: src/util/symbol.cpp.
package util

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
	length, e := text.Length()
	if e != nil {
		return Symbol{}, e
	}
	b := make([]byte, length)
	for i := uint32(0); i < length; i++ {
		v, re := text.Byte(uint64(i))
		if re != nil {
			return Symbol{}, re
		}
		b[i] = v
	}
	return m.Intern(string(b))
}
