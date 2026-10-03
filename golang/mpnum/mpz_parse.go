// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.cpp set(mpz &, char const *), _MP_INTERNAL backend.
package mpnum

// numeralByte gives the finite Go-string boundary the source C-string's NUL
// semantics. Embedded NUL ends parsing; bytes past it are never consumed.
func numeralByte(text string, i int) byte {
	if i >= len(text) {
		return 0
	}
	return text[i]
}

// SetString translates the source permissive byte parser: initial ASCII spaces
// are skipped for sign detection, while all nondigits are ignored thereafter.
// Input is borrowed only for the call. Work is staged before owner publication.
func (m *IntegerManager) SetString(out *Int, text string) error {
	if e := m.check(out); e != nil {
		return e
	}
	var work, tmp Int
	defer m.Del(&work)
	defer m.Del(&tmp)
	if e := m.Reset(&work); e != nil {
		return e
	}
	ten := NewInt(10)
	i := 0
	for numeralByte(text, i) == ' ' {
		i++
	}
	negative := numeralByte(text, i) == '-'
	for ; numeralByte(text, i) != 0; i++ {
		b := numeralByte(text, i)
		if b >= '0' && b <= '9' {
			digit := NewInt(int32(b - '0'))
			if e := m.Mul(&work, &ten, &tmp); e != nil {
				return e
			}
			if e := m.Add(&tmp, &digit, &work); e != nil {
				return e
			}
		}
	}
	if e := m.Del(&tmp); e != nil {
		return e
	}
	if negative {
		if e := m.Neg(&work); e != nil {
			return e
		}
	}
	return m.Move(out, &work)
}
