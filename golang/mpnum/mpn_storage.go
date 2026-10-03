// Copyright (c) 2011 Microsoft Corporation
// Source: src/util/mpn.h manager scratch and util/buffer.h growth policy.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// NaturalManager owns source division scratch and retained formatting workspace.
// Operands/output slices are borrowed only during a call. A manager may not be
// copied or used concurrently. Close invalidates the service. Requested source
// capacities are kept separate from arena geometric allocation classes.
type NaturalManager struct {
	words             *arena.Words
	divScratch        [4]arena.VectorRef
	divCapacity       [4]uint32
	formatScratch     [3]arena.VectorRef
	formatCapacity    [3]uint32
	text              *arena.SmallBytes
	textRef           arena.SmallBytesRef
	textCapacity      uint32
	config            arena.VectorConfig
	budget            *arena.Budget
	headerBytes       uint64
	ownsWords, closed bool
}

// NewNaturalManager charges all cold owner headers and the source four inline
// 16-limb buffers. Arena pages replace C++ inline storage, changing allocation
// timing while retaining source requested capacity/doubling policies.
func NewNaturalManager(c arena.VectorConfig) (*NaturalManager, error) {
	words, e := arena.NewWords(c)
	if e != nil {
		return nil, e
	}
	m, e := newNaturalManager(c, words, true)
	if e != nil {
		words.Close()
	}
	return m, e
}
func newNaturalManager(c arena.VectorConfig, words *arena.Words, owns bool) (*NaturalManager, error) {
	header := uint64(reflect.TypeOf(NaturalManager{}).Size())
	if e := c.Descriptors.Budget.Reserve(header); e != nil {
		return nil, e
	}
	m := &NaturalManager{words: words, config: c, budget: c.Descriptors.Budget, headerBytes: header, ownsWords: owns}
	for i := range m.divScratch {
		r, e := words.Allocate(0, 16)
		if e != nil {
			m.Close()
			return nil, e
		}
		m.divScratch[i] = r
		m.divCapacity[i] = 16
	}
	return m, nil
}
func (m *NaturalManager) check() error {
	if m == nil || m.closed || m.words == nil {
		return arena.ErrClosed
	}
	return nil
}
func (m *NaturalManager) Close() {
	if m == nil || m.closed {
		return
	}
	if m.words != nil {
		for _, r := range m.divScratch {
			if r != (arena.VectorRef{}) {
				m.words.Free(r)
			}
		}
		for _, r := range m.formatScratch {
			if r != (arena.VectorRef{}) {
				m.words.Free(r)
			}
		}
	}
	if m.text != nil {
		m.text.Close()
		m.text = nil
	}
	if m.ownsWords && m.words != nil {
		m.words.Close()
	}
	m.divScratch = [4]arena.VectorRef{}
	m.formatScratch = [3]arena.VectorRef{}
	m.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
}
func sourceCapacity(current, needed uint32) (uint32, error) {
	if current == 0 {
		current = 16
	}
	for current < needed {
		if current > ^uint32(0)/2 {
			return 0, arena.ErrBounds
		}
		current *= 2
	}
	return current, nil
}
func (m *NaturalManager) resizeScratch(ref *arena.VectorRef, capacity *uint32, size uint32) error {
	c, e := sourceCapacity(*capacity, size)
	if e != nil {
		return e
	}
	var r arena.VectorRef
	if *ref == (arena.VectorRef{}) {
		r, e = m.words.Allocate(size, c)
	} else {
		r, e = m.words.Reserve(*ref, c)
		if e == nil {
			*ref = r
			r, e = m.words.Resize(r, size)
		}
	}
	if e != nil {
		return e
	}
	*ref = r
	*capacity = c
	return nil
}
func (m *NaturalManager) resizeDiv(i int, n uint32) error {
	return m.resizeScratch(&m.divScratch[i], &m.divCapacity[i], n)
}
func (m *NaturalManager) divRead(r arena.VectorRef, i uint32) uint32 {
	v, e := m.words.Read(r, i)
	if e != nil {
		panic(e)
	}
	return v
}
func (m *NaturalManager) divWrite(r arena.VectorRef, i, v uint32) {
	if e := m.words.Write(r, i, v); e != nil {
		panic(e)
	}
}
