// Source: src/ast/format.cpp:135-136,147-151; src/util/symbol.cpp:50-74.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

// The source pointer is an actual immutable spelling owned by this exact
// interner. Duplicate insertion performs the same prepare/grow/hash/probe
// leaves, with reached bytes read in place, and never copies a Go string.
func (m *SymbolManager) internBorrowedSymbolCString(s Symbol) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return Symbol{}, e
	}
	if s.IsNumerical() {
		if astSourceDebug {
			return Symbol{}, ErrInvalidSymbol
		}
		// symbol.h:96 numerical bare_str is the actual empty string in release.
		// Avoid recursive lock acquisition for that source literal constructor.
		m.mu.Unlock()
		result, e := m.Intern("")
		m.mu.Lock()
		return result, e
	}
	if s.value == 0 {
		return Symbol{}, ErrInvalidSymbol
	} // actual null char-pointer guard
	key, e := m.key(s)
	if e != nil {
		return Symbol{}, e
	}
	_, entry, e := m.table.InsertIfNotThereCore(key)
	if e != nil {
		return Symbol{}, e
	}
	result, e := entry.Key()
	if e != nil {
		return Symbol{}, e
	}
	return m.symbol(result), nil
}
