// Source: src/util/sorting_network.h:28-34,87-94; src/util/vector.h:102-120,159-167,178-189.
package util

import (
	"errors"
	"reflect"
)

var ErrUnsignedSortingNetworkUnconstructed = errors.New("g3: sorting-network source object construction did not complete")

// Clone implements the generated copy constructor: vector members construct in
// declaration order, then scalar pointers copy their original addresses. Those
// pointers borrow the original object; they do not own the copied vectors.
func (s *RuntimeUnsignedSortingNetwork) Clone() (*RuntimeUnsignedSortingNetwork, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	charge := uint64(reflect.TypeOf(RuntimeUnsignedSortingNetwork{}).Size())
	if e := s.budget.Reserve(charge); e != nil {
		return nil, e
	}
	out := &RuntimeUnsignedSortingNetwork{budget: s.budget, header: charge, debug: s.debug}
	out.self = out
	out.currentv.Bind(s.currentv.Column.Raw.Runtime, s.budget)
	out.nextv.Bind(s.nextv.Column.Raw.Runtime, s.budget)
	if e := out.currentv.copyConstructDebug(&s.currentv, s.debug); e != nil {
		out.constructorCause = e
		return out, e // Failed member has no source destructor.
	}
	out.constructorCompleted = 1
	if e := out.nextv.copyConstructDebug(&s.nextv, s.debug); e != nil {
		out.constructorCause = e
		// The first vector completed before the second constructor body refused.
		// Its actual source destructor runs before returning the abort context.
		if cleanup := out.currentv.SourceFinalize(); cleanup != nil {
			return out, errors.Join(e, cleanup)
		}
		return out, e
	}
	out.constructorCompleted = 3
	out.current = s.current
	out.next = s.next
	out.constructed = true
	return out, nil
}
