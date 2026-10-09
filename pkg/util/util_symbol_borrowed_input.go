// Source: src/util/symbol.cpp:50-74.
package util

func (m *SymbolManager) InternBorrowedSymbolCString(s Symbol) (Symbol, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(s); e != nil {
		return Symbol{}, e
	}
	if s.IsNumerical() {
		if sourceDebug {
			return Symbol{}, ErrInvalidSymbol
		}
		// symbol.h:96 numerical bare_str is the actual empty string in release.
		// Avoid recursive lock acquisition for that source literal constructor.
		m.mu.Unlock()
		result, e := m.Intern("")
		m.mu.Lock()
		return result, e
	}
	if s.Value == 0 {
		return Symbol{}, ErrInvalidSymbol
	} // actual null char-pointer guard
	return s, nil
}
