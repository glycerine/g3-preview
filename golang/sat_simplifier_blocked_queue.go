// Source: src/sat/sat_simplifier.cpp:875–919. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package g3

import "errors"

func newSATSimplifierBlockedQueue(s *satSimplifier) (*satVarQueue, error) {
	if s == nil || s.self != s {
		return nil, ErrSATOwner
	}
	if s.lifecycle == nil || s.lifecycle.propagation == nil || s.lifecycle.propagation.memory == nil || s.lifecycle.budget != s.lifecycle.propagation.memory.Budget() {
		return nil, ErrSATOwner
	}
	q, e := newSATVarQueue(nil, s.lifecycle.budget)
	if e != nil {
		return nil, e
	}
	q.literalSource = s
	q.bindRuntime(s.lifecycle.propagation.memory)
	if e = q.initialize(); e != nil {
		if cleanup := q.Close(); cleanup != nil {
			return q, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return q, nil
}
func (s *satSimplifier) literalWeight(idx uint32) (uint32, error) {
	u, e := s.useList.Get(satLiteral(idx).negated())
	if e != nil {
		return 0, e
	}
	n, e := u.Size()
	if e != nil {
		return 0, e
	}
	w, e := s.lifecycle.propagation.variables.watches.Get(idx)
	if e != nil {
		return 0, e
	}
	count, e := w.Size()
	if e != nil {
		return 0, e
	}
	return 2*n + count, nil // source unsigned32 multiplication/addition wraps
}
func (s *satSimplifier) blockedDecreased(q *satVarQueue, l satLiteral) error {
	v := satBoolVar(uint32(l))
	yes, e := q.containsChecked(v)
	if e != nil {
		return e
	}
	if yes {
		return q.ActivityIncreased(v)
	}
	return q.insert(v)
}
