// Source: src/util/sorting_network.h:28-115; src/test/sorting_network.cpp:41-57.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrUnsignedSortingNetworkExecuting = errors.New("g3: sorting-network vector acquisition is executing")

// RuntimeUnsignedSortingNetwork is the concrete unsigned_ext specialization.
// Its two svectors own actual Runtime bytes, their default headers embedded in
// this cold owner. Input/output and assertion providers are borrowed.
type RuntimeUnsignedSortingNetwork struct {
	self                 *RuntimeUnsignedSortingNetwork
	currentv, nextv      RuntimeUnsignedVector
	current, next        *RuntimeUnsignedVector
	budget               *arena.Budget
	header               uint64
	debug                *DebugContext
	destructorCause      error
	constructorCause     error
	constructorCompleted uint8
	constructed          bool
	destructorStage      uint8
	closed               bool
	executing            bool
}

func NewRuntimeUnsignedSortingNetwork(r *Runtime, b *arena.Budget, d *DebugContext) (*RuntimeUnsignedSortingNetwork, error) {
	if r == nil || b == nil || r.Budget() != b || d != nil && (d.Self != d || d.BudgetField != b) {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	charge := uint64(reflect.TypeOf(RuntimeUnsignedSortingNetwork{}).Size())
	if e := b.Reserve(charge); e != nil {
		return nil, e
	}
	s := &RuntimeUnsignedSortingNetwork{budget: b, header: charge, debug: d, constructed: true, constructorCompleted: 3}
	s.self = s
	s.currentv.Bind(r, b)
	s.currentv.Constructed = true
	s.nextv.Bind(r, b)
	s.nextv.Constructed = true
	s.current = &s.currentv
	s.next = &s.nextv
	return s, nil
}
func (s *RuntimeUnsignedSortingNetwork) open() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.executing {
		return ErrUnsignedSortingNetworkExecuting
	}
	if !s.constructed || s.closed || s.destructorStage != 0 || s.destructorCause != nil {
		return arena.ErrClosed
	}
	return nil
}
func (s *RuntimeUnsignedSortingNetwork) Sort(in, out *RuntimeUnsignedVector) error {
	if e := s.open(); e != nil {
		return e
	}
	// Identity/readiness is a Go borrowing boundary, not an operand read.
	if in == nil || out == nil || in.self != in || out.self != out || in.Column.Raw.Runtime != s.currentv.Column.Raw.Runtime || out.Column.Raw.Runtime != s.currentv.Column.Raw.Runtime {
		return arena.ErrConfig
	}
	if e := out.Reset(); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := in.Size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		x, e := s.read(in, i, 309)
		if e != nil {
			return e
		}
		if e = s.push(out, x); e != nil {
			return e
		}
	}
	n, e := in.Size()
	if e != nil {
		return e
	}
	if n <= 1 {
		return nil
	}
	for {
		n, e = out.Size()
		if e != nil {
			return e
		}
		if unsignedNetworkPower2(n) {
			break
		}
		if e = s.push(out, 0); e != nil {
			return e
		}
	}
	for i := uint32(0); ; i++ {
		n, e = out.Size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		if e = s.push(&s.currentv, i); e != nil {
			return e
		}
		if e = s.push(&s.nextv, i); e != nil {
			return e
		}
	}
	for k := uint32(2); ; k *= 2 {
		n, e = out.Size()
		if e != nil {
			return e
		}
		if k > n {
			break
		}
		if e = s.sort(k, out); e != nil {
			return e
		}
	}
	return nil
}
func unsignedNetworkPower2(n uint32) bool { return n != 0 && (n-1)&n == 0 }
func (s *RuntimeUnsignedSortingNetwork) exchange(i, j uint32, out *RuntimeUnsignedVector) error {
	on, e := PermutationAssertions(s.debug)
	if e != nil {
		return e
	}
	if on && i > j {
		if e = DebugAssertionFailure(s.debug, "src/util/sorting_network.h", 40, "i <= j"); e != nil {
			return e
		}
	}
	if i < j {
		ei, e := s.read(out, i, 314)
		if e != nil {
			return e
		}
		ej, e := s.read(out, j, 314)
		if e != nil {
			return e
		}
		low := ej
		if ei <= ej {
			low = ei
		}
		if e = s.write(out, i, low, 329); e != nil {
			return e
		}
		high := ej
		if ej <= ei {
			high = ei
		}
		if e = s.write(out, j, high, 329); e != nil {
			return e
		}
	}
	return nil
}
func (s *RuntimeUnsignedSortingNetwork) sort(k uint32, out *RuntimeUnsignedVector) error {
	on, e := PermutationAssertions(s.debug)
	if e != nil {
		return e
	}
	if on && !(unsignedNetworkPower2(k) && k > 0) {
		if e = DebugAssertionFailure(s.debug, "src/util/sorting_network.h", 50, "is_power_of2(k) && k > 0"); e != nil {
			return e
		}
	}
	if k == 2 {
		for i := uint32(0); ; i++ {
			n, e := out.Size()
			if e != nil {
				return e
			}
			if i >= n/2 {
				break
			}
			a, e := s.read(s.current, 2*i, 304)
			if e != nil {
				return e
			}
			b, e := s.read(s.current, 2*i+1, 304)
			if e != nil {
				return e
			}
			if e = s.exchange(a, b, out); e != nil {
				return e
			}
			a, e = s.read(s.current, 2*i, 304)
			if e != nil {
				return e
			}
			if e = s.write(s.next, 2*i, a, 304); e != nil {
				return e
			}
			b, e = s.read(s.current, 2*i+1, 304)
			if e != nil {
				return e
			}
			if e = s.write(s.next, 2*i+1, b, 304); e != nil {
				return e
			}
		}
		s.current, s.next = s.next, s.current
	} else {
		for i := uint32(0); ; i++ {
			n, e := out.Size()
			if e != nil {
				return e
			}
			if k == 0 {
				return arena.ErrBounds
			} // Reached unsigned division-by-zero guard after size read.
			if i >= n/k {
				break
			}
			ki := k * i
			for j := uint32(0); j < k/2; j++ {
				a, e := s.read(s.current, ki+2*j, 304)
				if e != nil {
					return e
				}
				if e = s.write(s.next, ki+j, a, 304); e != nil {
					return e
				}
				a, e = s.read(s.current, ki+2*j+1, 304)
				if e != nil {
					return e
				}
				if e = s.write(s.next, ki+k/2+j, a, 304); e != nil {
					return e
				}
			}
		}
		s.current, s.next = s.next, s.current
		if e = s.sort(k/2, out); e != nil {
			return e
		}
		for i := uint32(0); ; i++ {
			n, e := out.Size()
			if e != nil {
				return e
			}
			if k == 0 {
				return arena.ErrBounds
			} // Reached unsigned division-by-zero guard after size read.
			if i >= n/k {
				break
			}
			ki := k * i
			for j := uint32(0); j < k/2; j++ {
				a, e := s.read(s.current, ki+j, 304)
				if e != nil {
					return e
				}
				if e = s.write(s.next, ki+2*j, a, 304); e != nil {
					return e
				}
				a, e = s.read(s.current, ki+k/2+j, 304)
				if e != nil {
					return e
				}
				if e = s.write(s.next, ki+2*j+1, a, 304); e != nil {
					return e
				}
			}
			for j := uint32(0); j < k/2-1; j++ {
				a, e := s.read(s.next, ki+2*j+1, 304)
				if e != nil {
					return e
				}
				b, e := s.read(s.next, ki+2*(j+1), 304)
				if e != nil {
					return e
				}
				if e = s.exchange(a, b, out); e != nil {
					return e
				}
			}
		}
		s.current, s.next = s.next, s.current
	}
	return nil
}
func (s *RuntimeUnsignedSortingNetwork) SourceFinalize() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.executing {
		return ErrUnsignedSortingNetworkExecuting
	}
	if !s.constructed {
		return ErrUnsignedSortingNetworkUnconstructed
	}
	if s.destructorCause != nil {
		return s.destructorCause
	}
	if s.closed || s.destructorStage == 2 {
		return nil
	}
	if s.destructorStage == 0 {
		if e := s.nextv.SourceFinalize(); e != nil {
			s.destructorCause = e
			return e
		}
		s.destructorStage = 1
	}
	if e := s.currentv.SourceFinalize(); e != nil {
		s.destructorCause = e
		return e
	}
	s.destructorStage = 2
	return nil
}
func (s *RuntimeUnsignedSortingNetwork) closeOwning() error {
	if e := s.nextv.closeOwning(); e != nil {
		return e
	}
	if e := s.currentv.closeOwning(); e != nil {
		return e
	}
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
func (s *RuntimeUnsignedSortingNetwork) Close() error {
	if s == nil || s.self != s {
		return ErrRuntimeVectorOwner
	}
	if s.closed {
		return nil
	}
	if !s.constructed {
		return s.closeOwning() // Explicit failed-constructor owning abort, no class/member source DT.
	}
	if e := s.SourceFinalize(); e != nil {
		return e
	}
	return s.closeOwning()
}
func (s *RuntimeUnsignedSortingNetwork) CleanupStopped() error {
	if s == nil || s.self != s || s.destructorCause == nil {
		return ErrRuntimeVectorOwner
	}
	if s.closed {
		return nil
	}
	return s.closeOwning()
}
