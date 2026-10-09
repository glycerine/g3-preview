package util

import "github.com/glycerine/g3/pkg/arena"

// AppendFloatCString follows string_buffer.h:67-76 through the borrowed
// formatting result: strlen followed by a copy into Go-managed storage.

func (b *StringBuffer64) AppendFloatCString(v ArbitraryFloatFormattedCString) error {
	if err := b.open(); err != nil {
		return err
	}
	n, err := v.Length()
	if err != nil {
		return err
	}
	if n > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	text := make([]byte, int(n))
	for i := uint64(0); i < n; i++ {
		c, err := v.Byte(i)
		if err != nil {
			return err
		}
		text[i] = c
	}
	return b.AppendBytes(text)
}
