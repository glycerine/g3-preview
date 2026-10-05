// Source: src/util/permutation.cpp:21-76; src/util/permutation.h:25-44.
// Source author: Leonardo de Moura (2011-06-10).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"io"
	"reflect"
	"strconv"
)

type RuntimePermutation struct {
	self                              *RuntimePermutation
	forward, inverse                  RuntimeUnsignedVector
	budget                            *arena.Budget
	header                            uint64
	constructorCause, destructorCause error
	destructorStage                   uint8
	completedMembers                  uint8
	closed                            bool
}

func NewRuntimePermutation(r *memory.Runtime, b *arena.Budget, n uint32) (*RuntimePermutation, error) {
	if r == nil || b == nil || r.Budget() != b {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(RuntimePermutation{}).Size())
	if e := b.Reserve(h); e != nil {
		return nil, e
	}
	p := &RuntimePermutation{budget: b, header: h}
	p.self = p
	p.forward.bind(r, b)
	p.inverse.bind(r, b)
	p.forward.constructed = true
	p.inverse.constructed = true
	p.completedMembers = 3
	if e := p.Reset(n); e != nil {
		p.constructorCause = e
		if d := p.SourceFinalize(); d != nil {
			return p, errors.Join(e, d)
		}
		return p, e
	}
	return p, nil
}
func (p *RuntimePermutation) open() error {
	if p == nil || p.self != p {
		return ErrSATOwner
	}
	if p.closed || p.completedMembers != 3 || p.constructorCause != nil || p.destructorStage != 0 || p.destructorCause != nil {
		return arena.ErrClosed
	}
	return nil
}
func (p *RuntimePermutation) Size() (uint32, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	return p.forward.Size()
}
func (p *RuntimePermutation) At(i uint32) (uint32, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	return p.forward.At(i)
}
func (p *RuntimePermutation) Inv(i uint32) (uint32, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	return p.inverse.At(i)
}
func (p *RuntimePermutation) Reset(n uint32) error {
	if e := p.open(); e != nil {
		return e
	}
	if e := p.forward.Reset(); e != nil {
		return e
	}
	if e := p.inverse.Reset(); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if e := p.forward.Push(i); e != nil {
			return e
		}
		if e := p.inverse.Push(i); e != nil {
			return e
		}
	}
	return nil
}
func (p *RuntimePermutation) Swap(i, j uint32) error {
	if e := p.open(); e != nil {
		return e
	}
	a, e := p.forward.At(i)
	if e != nil {
		return e
	}
	b, e := p.forward.At(j)
	if e != nil {
		return e
	}
	// std::swap first reads its actual operands, then publishes them in order.
	x, e := p.forward.At(i)
	if e != nil {
		return e
	}
	y, e := p.forward.At(j)
	if e != nil {
		return e
	}
	if e = p.forward.Set(i, y); e != nil {
		return e
	}
	if e = p.forward.Set(j, x); e != nil {
		return e
	}
	x, e = p.inverse.At(a)
	if e != nil {
		return e
	}
	y, e = p.inverse.At(b)
	if e != nil {
		return e
	}
	if e = p.inverse.Set(a, y); e != nil {
		return e
	}
	return p.inverse.Set(b, x)
}
func (p *RuntimePermutation) MoveAfter(i, j uint32) error {
	if e := p.open(); e != nil {
		return e
	}
	if i >= j {
		return nil
	}
	a, e := p.forward.At(i)
	if e != nil {
		return e
	}
	for k := i; k < j; k++ {
		v, e := p.forward.At(k + 1)
		if e != nil {
			return e
		}
		if e = p.forward.Set(k, v); e != nil {
			return e
		}
		v, e = p.forward.At(k)
		if e != nil {
			return e
		}
		if e = p.inverse.Set(v, k); e != nil {
			return e
		}
	}
	if e = p.forward.Set(j, a); e != nil {
		return e
	}
	if e = p.inverse.Set(a, j); e != nil {
		return e
	}
	if satStorageSourceDebug {
		_, e = p.CheckInvariant()
		return e
	}
	return nil
}
func (p *RuntimePermutation) CheckInvariant() (bool, error) {
	if e := p.open(); e != nil {
		return false, e
	}
	if satStorageSourceDebug {
		a, e := p.forward.Size()
		if e != nil {
			return false, e
		}
		b, e := p.inverse.Size()
		if e != nil {
			return false, e
		}
		if a != b {
			return false, ErrSATAssertion
		}
	}
	n, e := p.forward.Size()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		if satStorageSourceDebug {
			a, e := p.forward.At(i)
			if e != nil {
				return false, e
			}
			if a >= n {
				return false, ErrSATAssertion
			}
			b, e := p.inverse.At(i)
			if e != nil {
				return false, e
			}
			if b >= n {
				return false, ErrSATAssertion
			}
			b, e = p.inverse.At(i)
			if e != nil {
				return false, e
			}
			a, e = p.forward.At(b)
			if e != nil {
				return false, e
			}
			if a != i {
				return false, ErrSATAssertion
			}
			a, e = p.forward.At(i)
			if e != nil {
				return false, e
			}
			b, e = p.inverse.At(a)
			if e != nil {
				return false, e
			}
			if b != i {
				return false, ErrSATAssertion
			}
		}
	}
	return true, nil
}
func (p *RuntimePermutation) Display(w io.Writer) error {
	n, e := p.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			if _, e = io.WriteString(w, " "); e != nil {
				return e
			}
		}
		if _, e = io.WriteString(w, strconv.FormatUint(uint64(i), 10)); e != nil {
			return e
		}
		if _, e = io.WriteString(w, ":"); e != nil {
			return e
		}
		v, e := p.forward.At(i)
		if e != nil {
			return e
		}
		if _, e = io.WriteString(w, strconv.FormatUint(uint64(v), 10)); e != nil {
			return e
		}
	}
	return nil
}
func (p *RuntimePermutation) SourceFinalize() error {
	if p == nil || p.self != p {
		return ErrSATOwner
	}
	if p.closed || p.destructorStage == 2 {
		return nil
	}
	if p.destructorCause != nil {
		return p.destructorCause
	}
	if p.destructorStage == 0 {
		if p.completedMembers&2 != 0 {
			if e := p.inverse.SourceFinalize(); e != nil {
				p.destructorCause = e
				return e
			}
		}
		p.destructorStage = 1
	}
	if p.completedMembers&1 != 0 {
		if e := p.forward.SourceFinalize(); e != nil {
			p.destructorCause = e
			return e
		}
	}
	p.destructorStage = 2
	return nil
}
func (p *RuntimePermutation) closeOwning() error {
	if e := p.inverse.closeOwning(); e != nil {
		return e
	}
	if e := p.forward.closeOwning(); e != nil {
		return e
	}
	p.budget.Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
func (p *RuntimePermutation) Close() error {
	if p == nil || p.self != p {
		return ErrSATOwner
	}
	if p.closed {
		return nil
	}
	if e := p.SourceFinalize(); e != nil {
		return e
	}
	return p.closeOwning()
}
func (p *RuntimePermutation) CleanupStopped() error {
	if p == nil || p.self != p || p.destructorCause == nil {
		return ErrSATOwner
	}
	if p.closed {
		return nil
	}
	return p.closeOwning()
}
