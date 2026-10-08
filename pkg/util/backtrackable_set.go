// Source: src/util/backtrackable_set.h:25-111.
package util

import "errors"

var ErrBacktrackableSetScope = errors.New("g3: backtrackable set scope underflow")

// BacktrackableSetEvent is called while an insertion or deletion is undone.
// inserted is true when an insertion is being undone and false when a deletion
// is being undone, matching the source event handler.
type BacktrackableSetEvent func(value any, inserted bool)

type backtrackableSetTrail struct {
	value    any
	inserted bool
}

// BacktrackableSet stores comparable Go values. Its retained C++ consumer uses
// pointer identity, which maps directly to comparable Go pointers.
type BacktrackableSet struct {
	values map[any]struct{}
	trail  []backtrackableSetTrail
	scopes []uint32
	event  BacktrackableSetEvent
}

func NewBacktrackableSet(event BacktrackableSetEvent) *BacktrackableSet {
	return &BacktrackableSet{values: make(map[any]struct{}), event: event}
}

func (s *BacktrackableSet) Insert(value any) {
	if len(s.scopes) == 0 {
		s.values[value] = struct{}{}
		return
	}
	if _, exists := s.values[value]; exists {
		return
	}
	s.values[value] = struct{}{}
	s.trail = append(s.trail, backtrackableSetTrail{value: value, inserted: true})
}

func (s *BacktrackableSet) Erase(value any) {
	if len(s.scopes) == 0 {
		delete(s.values, value)
		return
	}
	if _, exists := s.values[value]; !exists {
		return
	}
	delete(s.values, value)
	s.trail = append(s.trail, backtrackableSetTrail{value: value})
}

func (s *BacktrackableSet) Contains(value any) bool {
	_, exists := s.values[value]
	return exists
}

func (s *BacktrackableSet) Empty() bool { return len(s.values) == 0 }

func (s *BacktrackableSet) PushScope() {
	s.scopes = append(s.scopes, uint32(len(s.trail)))
}

func (s *BacktrackableSet) PopScope() error {
	if len(s.scopes) == 0 {
		return ErrBacktrackableSetScope
	}
	lastScope := len(s.scopes) - 1
	oldSize := s.scopes[lastScope]
	s.scopes = s.scopes[:lastScope]
	for len(s.trail) > int(oldSize) {
		last := len(s.trail) - 1
		entry := s.trail[last]
		if s.event != nil {
			s.event(entry.value, entry.inserted)
		}
		if entry.inserted {
			delete(s.values, entry.value)
		} else {
			s.values[entry.value] = struct{}{}
		}
		s.trail[last] = backtrackableSetTrail{}
		s.trail = s.trail[:last]
	}
	return nil
}

func (s *BacktrackableSet) Reset() {
	clear(s.values)
	clear(s.trail)
	clear(s.scopes)
	s.trail = s.trail[:0]
	s.scopes = s.scopes[:0]
}

// Values returns the current set contents. As with the source hash-set
// iterators, callers must not depend on iteration order.
func (s *BacktrackableSet) Values() []any {
	result := make([]any, 0, len(s.values))
	for value := range s.values {
		result = append(result, value)
	}
	return result
}
