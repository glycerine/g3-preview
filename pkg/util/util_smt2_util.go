// Translation of src/util/smt2_util.h/.cpp, Leonardo de Moura.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrSMT2NeedsQuoting = errors.New("g3: SMT2 symbol does not satisfy quoting precondition")

func IsSMT2SimpleSymbolChar(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' || c == '!' || c == '@' || c == '$' || c == '%' || c == '^' || c == '&' || c == '*' || c == '_' || c == '-' || c == '+' || c == '=' || c == '<' || c == '>' || c == '.' || c == '?' || c == '/'
}

// IsSMT2QuotedCString is the source needs-quoting predicate, not a quoted parser.
// The input is borrowed only for this call; nil is the source null C pointer.
func IsSMT2QuotedCString(text *string) bool {
	if text == nil {
		return false
	}
	s := *text
	n := cSymbolLength(s)
	s = s[:n]
	if n > 0 && s[0] >= '0' && s[0] <= '9' {
		return true
	}
	if n >= 2 && s[0] == '|' && s[n-1] == '|' {
		for i := 1; i+1 < n; i++ {
			if s[i] == '\\' && i+2 < n && (s[i+1] == '\\' || s[i+1] == '|') {
				i++
			} else if s[i] == '\\' || s[i] == '|' {
				return true
			}
		}
		return false
	}
	for i := 0; i < n; i++ {
		if !IsSMT2SimpleSymbolChar(s[i]) {
			return true
		}
	}
	return false
}
func smt2QuotedSymbolLocked(m *SymbolManager, s Symbol) (bool, error) {
	if e := m.checkPlain(s); e != nil {
		return false, e
	}
	if s.IsNumerical() || s.Value == 0 {
		return false, nil
	}
	text, e := m.spelling(s)
	if e != nil {
		return false, e
	}
	return IsSMT2QuotedCString(&text), nil
}
func IsSMT2QuotedSymbol(m *SymbolManager, s Symbol) (bool, error) {
	if m == nil {
		return false, ErrInvalidSymbol
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return smt2QuotedSymbolLocked(m, s)
}

// MkSMT2QuotedSymbol holds only a symbol-manager lock and an unpublished local
// buffer. Its private scalar buffer operations take no shared-buffer mutex, so
// they cannot invert AppendSymbol's shared-buffer -> symbol-manager lock order.
// Temporary owned storage is budgeted; the sole returned string is caller-owned.
func MkSMT2QuotedSymbol(m *SymbolManager, s Symbol, _ *arena.Budget) (string, error) {
	if m == nil {
		return "", ErrInvalidSymbol
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	needed, e := smt2QuotedSymbolLocked(m, s)
	if e != nil {
		return "", e
	}
	if !needed {
		return "", ErrSMT2NeedsQuoting
	}
	b, e := NewStringBuffer64(StringBufferConfig{})
	if e != nil {
		return "", e
	}
	defer b.Close()
	if e = b.AppendByte('|'); e != nil {
		return "", e
	}
	text, e := m.spelling(s)
	if e != nil {
		return "", e
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '|' || c == '\\' {
			if e = b.AppendByte('\\'); e != nil {
				return "", e
			}
		}
		if e = b.AppendByte(c); e != nil {
			return "", e
		}
	}
	if e = b.AppendByte('|'); e != nil {
		return "", e
	}
	return b.CStr()
}
