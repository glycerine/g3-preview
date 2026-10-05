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
	debugContext                      *DebugContext
	constructorCause, destructorCause error
	destructorStage                   uint8
	completedMembers                  uint8
	closed                            bool
}

func NewRuntimePermutation(r *memory.Runtime, b *arena.Budget, n uint32) (*RuntimePermutation, error) {
	return NewRuntimePermutationWithAssertions(r, b, n, nil)
}

// The assertion provider is borrowed configuration; it is read only at macro sites.
func NewRuntimePermutationWithAssertions(r *memory.Runtime, b *arena.Budget, n uint32, d *DebugContext) (*RuntimePermutation, error) {
	if d != nil && (d.self != d || d.budget != b) {
		return nil, arena.ErrConfig
	}
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
	p := &RuntimePermutation{budget: b, header: h, debugContext: d}
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
	return p.read(&p.forward, i)
}
func (p *RuntimePermutation) Inv(i uint32) (uint32, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	return p.read(&p.inverse, i)
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
		if e := p.push(&p.forward, i); e != nil {
			return e
		}
		if e := p.push(&p.inverse, i); e != nil {
			return e
		}
	}
	return nil
}
func (p *RuntimePermutation) Swap(i, j uint32) error {
	if e := p.open(); e != nil {
		return e
	}
	a, e := p.readMutable(&p.forward, i)
	if e != nil {
		return e
	}
	b, e := p.readMutable(&p.forward, j)
	if e != nil {
		return e
	}
	// Source std::swap arguments acquire both actual lvalues before its body.
	left, e := p.acquireMutable(&p.forward, i)
	if e != nil {
		return e
	}
	right, e := p.acquireMutable(&p.forward, j)
	if e != nil {
		return e
	}
	if e = permutationSwapWords(left, right); e != nil {
		return e
	}
	left, e = p.acquireMutable(&p.inverse, a)
	if e != nil {
		return e
	}
	right, e = p.acquireMutable(&p.inverse, b)
	if e != nil {
		return e
	}
	return permutationSwapWords(left, right)
}
func (p *RuntimePermutation) MoveAfter(i, j uint32) error {
	if e := p.open(); e != nil {
		return e
	}
	if i >= j {
		return nil
	}
	a, e := p.readMutable(&p.forward, i)
	if e != nil {
		return e
	}
	for k := i; k < j; k++ {
		v, e := p.readMutable(&p.forward, k+1)
		if e != nil {
			return e
		}
		if e = p.write(&p.forward, k, v); e != nil {
			return e
		}
		v, e = p.readMutable(&p.forward, k)
		if e != nil {
			return e
		}
		if e = p.write(&p.inverse, v, k); e != nil {
			return e
		}
	}
	if e = p.write(&p.forward, j, a); e != nil {
		return e
	}
	if e = p.write(&p.inverse, a, j); e != nil {
		return e
	}
	on, e := permutationAssertions(p.debugContext)
	if e != nil {
		return e
	}
	if on {
		_, e = p.CheckInvariant()
		return e
	}
	return nil
}
func (p *RuntimePermutation) CheckInvariant() (bool, error) {
	if e := p.open(); e != nil {
		return false, e
	}
	on, e := permutationAssertions(p.debugContext)
	if e != nil {
		return false, e
	}
	if on {
		a, e := p.forward.Size()
		if e != nil {
			return false, e
		}
		b, e := p.inverse.Size()
		if e != nil {
			return false, e
		}
		if a != b {
			if e := debugAssertionFailure(p.debugContext, "src/util/permutation.cpp", 67, "m_p.size() == m_inv_p.size()"); e != nil {
				return false, e
			}
		}
	}
	n, e := p.forward.Size()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < n; i++ {
		on, e = permutationAssertions(p.debugContext)
		if e != nil {
			return false, e
		}
		if on {
			a, e := p.read(&p.forward, i)
			if e != nil {
				return false, e
			}
			if a >= n {
				if e := debugAssertionFailure(p.debugContext, "src/util/permutation.cpp", 70, "m_p[i] < n"); e != nil {
					return false, e
				}
			}
		}
		on, e = permutationAssertions(p.debugContext)
		if e != nil {
			return false, e
		}
		if on {
			a, e := p.read(&p.inverse, i)
			if e != nil {
				return false, e
			}
			if a >= n {
				if e := debugAssertionFailure(p.debugContext, "src/util/permutation.cpp", 71, "m_inv_p[i] < n"); e != nil {
					return false, e
				}
			}
		}
		on, e = permutationAssertions(p.debugContext)
		if e != nil {
			return false, e
		}
		if on {
			a, e := p.read(&p.inverse, i)
			if e != nil {
				return false, e
			}
			b, e := p.read(&p.forward, a)
			if e != nil {
				return false, e
			}
			if b != i {
				if e := debugAssertionFailure(p.debugContext, "src/util/permutation.cpp", 72, "m_p[m_inv_p[i]] == i"); e != nil {
					return false, e
				}
			}
		}
		on, e = permutationAssertions(p.debugContext)
		if e != nil {
			return false, e
		}
		if on {
			a, e := p.read(&p.forward, i)
			if e != nil {
				return false, e
			}
			b, e := p.read(&p.inverse, a)
			if e != nil {
				return false, e
			}
			if b != i {
				if e := debugAssertionFailure(p.debugContext, "src/util/permutation.cpp", 73, "m_inv_p[m_p[i]] == i"); e != nil {
					return false, e
				}
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
		v, e := p.read(&p.forward, i)
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
