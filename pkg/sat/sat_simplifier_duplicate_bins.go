// Source: src/sat/sat_simplifier.cpp:736–784.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

func satSimplifierBinaryLess(a, b satWatched) (bool, error) {
	if !a.Binary() {
		return false, nil
	}
	if !b.Binary() {
		return true, nil
	}
	x, e := a.Literal()
	if e != nil {
		return false, e
	}
	y, e := b.Literal()
	if e != nil {
		return false, e
	}
	if uint32(x) < uint32(y) {
		return true, nil
	}
	if x == y {
		learned, e := a.Learned()
		if e != nil {
			return false, e
		}
		if !learned {
			other, e := b.Learned()
			if e != nil {
				return false, e
			}
			if other {
				return true, nil
			}
		}
	}
	return false, nil
}
func (s *satSimplifier) sortBinaryWatches(w satSolverWatchListView) error {
	it, e := s.watchIterator(w)
	if e != nil {
		return e
	}
	n := it.end
	if n < 2 {
		return nil
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(satWatched{}).Size())
	if bytes > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e = s.lifecycle.budget.Reserve(bytes); e != nil {
		return e
	}
	scratch := make([]satWatched, n)
	defer s.lifecycle.budget.Release(bytes)
	// Stable bottom-up merge over the actual captured mutable payload. The
	// temporary scalar records are nonowning borrows, charged Go sort workspace.
	for width := uint64(1); width < uint64(n); width *= 2 {
		for base := uint64(0); base < uint64(n); base += 2 * width {
			mid := base + width
			if mid > uint64(n) {
				mid = uint64(n)
			}
			end := base + 2*width
			if end > uint64(n) {
				end = uint64(n)
			}
			left, right := base, mid
			for j := base; j < end; j++ {
				chooseRight := false
				var a, b satWatched
				if left == mid {
					chooseRight = true
				} else if right < end {
					a, e = satPropagationCursorRead(&it, uint32(right))
					if e != nil {
						return e
					}
					b, e = satPropagationCursorRead(&it, uint32(left))
					if e != nil {
						return e
					}
					chooseRight, e = satSimplifierBinaryLess(a, b)
					if e != nil {
						return e
					}
				}
				idx := left
				if chooseRight {
					idx = right
				}
				scratch[j], e = satPropagationCursorRead(&it, uint32(idx))
				if e != nil {
					return e
				}
				if chooseRight {
					right++
				} else {
					left++
				}
			}
			for j := base; j < end; j++ {
				if e = satPropagationCursorWrite(&it, uint32(j), scratch[j]); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func (s *satSimplifier) eliminateDuplicateBinaries() error {
	p := s.lifecycle.propagation
	ws := p.variables.watches
	elim := uint32(0)
	end := uint32(len(ws.lists))
	for idx := uint32(0); idx < end; idx++ {
		if e := p.checkpoint(); e != nil {
			return e
		}
		w, e := ws.Get(idx)
		if e != nil {
			return e
		}
		if e = s.sortBinaryWatches(w); e != nil {
			return e
		}
		it, e := s.watchIterator(w)
		if e != nil {
			return e
		}
		last := DefaultLiteral()
		out := uint32(0)
		for i := uint32(0); i < it.end; i++ {
			x, e := satPropagationCursorRead(&it, i)
			if e != nil {
				return e
			}
			if !x.Binary() {
				if e = satPropagationCursorWrite(&it, out, x); e != nil {
					return e
				}
				out++
				continue
			}
			l, e := x.Literal()
			if e != nil {
				return e
			}
			if l == last {
				elim++
			} else {
				last, e = x.Literal()
				if e != nil {
					return e
				}
				if e = satPropagationCursorWrite(&it, out, x); e != nil {
					return e
				}
				out++
			}
		}
		if e = p.finishPropagationList(&it, w, out); e != nil {
			return e
		}
	}
	s.numSubsumed += elim / 2
	return nil
}
