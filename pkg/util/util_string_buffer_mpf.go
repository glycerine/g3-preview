package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// AppendFloatCString follows string_buffer.h:67-76 through the borrowed
// formatting result: strlen, expansion, memcpy, then position publication.

func (b *StringBuffer64) AppendFloatCString(v ArbitraryFloatFormattedCString) error {
	if b == nil {
		return ErrStringBufferOwner
	}
	b.State.Mu.Lock()
	defer b.State.Mu.Unlock()
	if err := b.State.Open(b.Self == b); err != nil {
		return err
	}
	n, err := v.Length()
	if err != nil {
		return err
	}
	if n > ^uint64(0)-b.State.Pos {
		return arena.ErrBounds
	}
	next := b.State.Pos + n
	for next > b.State.Capacity {
		if err := b.State.Expand(b.Inline[:]); err != nil {
			return err
		}
	}
	for i := uint64(0); i < n; i++ {
		c, err := v.Byte(i)
		if err != nil {
			return err
		}
		b.State.Data(b.Inline[:])[b.State.Pos+i] = c
	}
	b.State.Pos = next
	return nil
}
