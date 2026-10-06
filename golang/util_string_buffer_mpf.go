// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
)

// AppendFloatCString follows string_buffer.h:67-76 through the borrowed
// formatting result: strlen, expansion, memcpy, then position publication.
func (b *StringBuffer64) AppendFloatCString(v mpnum.ArbitraryFloatFormattedCString) error {
	if b == nil {
		return ErrStringBufferOwner
	}
	b.state.mu.Lock()
	defer b.state.mu.Unlock()
	if err := b.state.open(b.self == b); err != nil {
		return err
	}
	n, err := v.Length()
	if err != nil {
		return err
	}
	if n > ^uint64(0)-b.state.pos {
		return arena.ErrBounds
	}
	next := b.state.pos + n
	for next > b.state.capacity {
		if err := b.state.expand(b.inline[:]); err != nil {
			return err
		}
	}
	for i := uint64(0); i < n; i++ {
		c, err := v.Byte(i)
		if err != nil {
			return err
		}
		b.state.data(b.inline[:])[b.state.pos+i] = c
	}
	b.state.pos = next
	return nil
}
