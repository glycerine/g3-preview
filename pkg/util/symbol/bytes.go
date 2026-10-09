// Source: src/util/symbol.cpp.
package symbol

func (m *Manager) InternBytes(text []byte) (Symbol, error) {
	for i, b := range text {
		if b == 0 {
			text = text[:i]
			break
		}
	}
	return m.Intern(string(text))
}
