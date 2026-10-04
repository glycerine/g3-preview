// Licensed under the MIT license.
// Translation of src/util/util.h/.cpp::escaped, Leonardo de Moura.
package g3

import "github.com/glycerine/g3/golang/arena"

func escapedEnd(text string, trim bool) int {
	n := cSymbolLength(text)
	if trim {
		for n > 0 && text[n-1] == '\n' {
			n--
		}
	}
	return n
}

// AppendEscapedCString translates ostream byte actions into a concrete owned
// destination. Only quote is escaped; retained LF emits indent following spaces.
// Successful byte prefixes remain live on budget failure. This byte adapter has
// a host-int representability bound; the complete uint32 indentation domain is valid.
// Source ostream custom state/badbit behavior is a separate unported surface.
func AppendEscapedCString(dst *StringBuffer64, text *string, trim bool, indent uint32) error {
	if dst == nil {
		return ErrStringBufferOwner
	}
	dst.state.mu.Lock()
	defer dst.state.mu.Unlock()
	if e := dst.state.open(dst.self == dst); e != nil {
		return e
	}
	if text == nil {
		return nil
	}
	s := *text
	n := escapedEnd(s, trim)
	count := dst.state.pos
	for i := 0; i < n; i++ {
		extra := uint64(1)
		if s[i] == '"' {
			extra++
		}
		if s[i] == '\n' {
			extra += uint64(indent)
		}
		if count > uint64(^uint(0)>>1) || extra > uint64(^uint(0)>>1)-count {
			return arena.ErrBounds
		}
		count += extra
	}
	for i := 0; i < n; i++ {
		c := s[i]
		if c == '"' {
			if e := dst.state.appendByte(dst.inline[:], '\\'); e != nil {
				return e
			}
		}
		if e := dst.state.appendByte(dst.inline[:], c); e != nil {
			return e
		}
		if c == '\n' {
			for j := uint32(0); j < indent; j++ {
				if e := dst.state.appendByte(dst.inline[:], ' '); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

// EscapedCString owns only temporary budgeted storage; its output is one final
// caller-owned string. Error discards the temporary prefix and releases storage.
func EscapedCString(text *string, trim bool, indent uint32, budget *arena.Budget) (string, error) {
	b, e := NewStringBuffer64(StringBufferConfig{Budget: budget})
	if e != nil {
		return "", e
	}
	defer b.Close()
	if e = AppendEscapedCString(b, text, trim, indent); e != nil {
		return "", e
	}
	return b.CStr()
}
