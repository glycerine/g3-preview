package util

import (
	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

// Source: src/util/string_buffer.h:67-76.
// AppendTo copies the borrowed formatting result into Go-managed storage.
func (v ArbitraryFloatFormattedCString) AppendTo(b *utext.Buffer) error {
	if b == nil {
		return utext.ErrOwner
	}
	if b.Closed() {
		return arena.ErrClosed
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
