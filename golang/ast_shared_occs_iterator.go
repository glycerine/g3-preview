// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

// Iterator borrows the collector. A scalar epoch checks the Go lifetime
// adaptation of source entry pointers invalidated by backing replacement/free.
type ASTSharedOccurrencesIterator struct {
	owner    *ASTSharedOccurrences
	epoch    uint64
	index    uint32
	endpoint uint32
}

func (s *ASTSharedOccurrences) BeginShared() (ASTSharedOccurrencesIterator, error) {
	m, e := s.owner.lock()
	if e != nil {
		return ASTSharedOccurrencesIterator{}, e
	}
	defer m.mu.Unlock()
	if s.table.pending != 0 {
		return ASTSharedOccurrencesIterator{}, ErrASTHandle
	}
	it := ASTSharedOccurrencesIterator{owner: s, epoch: s.table.epoch, endpoint: s.table.capacity}
	it.skip()
	return it, nil
}
func (s *ASTSharedOccurrences) EndShared() (ASTSharedOccurrencesIterator, error) {
	it, e := s.BeginShared()
	if e != nil {
		return it, e
	}
	it.index = s.table.capacity
	return it, nil
}
func (it *ASTSharedOccurrencesIterator) skip() {
	for it.index < it.endpoint && !astExprUsed(it.owner.table.entries[it.index]) {
		it.index++
	}
}
func (it *ASTSharedOccurrencesIterator) check() error {
	if it == nil || it.owner == nil {
		return ErrASTHandle
	}
	if e := it.owner.owner.check(); e != nil {
		return e
	}
	if it.owner.owner.manager.closed {
		return ErrASTHandle
	}
	if it.epoch != it.owner.table.epoch || it.owner.table.pending != 0 {
		return ErrASTHandle
	}
	return nil
}
func (it *ASTSharedOccurrencesIterator) Done() (bool, error) {
	if e := it.check(); e != nil {
		return false, e
	}
	return it.index == it.endpoint, nil
}
func (it *ASTSharedOccurrencesIterator) Value() (ASTHandle, error) {
	if e := it.check(); e != nil {
		return ASTHandle{}, e
	}
	if it.index >= it.endpoint {
		return ASTHandle{}, ErrASTHandle
	}
	return it.owner.owner.manager.external(it.owner.table.entries[it.index]), nil
}
func (it *ASTSharedOccurrencesIterator) Next() error {
	if e := it.check(); e != nil {
		return e
	}
	if it.index >= it.endpoint {
		return ErrASTHandle
	}
	it.index++
	it.skip()
	return nil
}
func (it *ASTSharedOccurrencesIterator) Equal(other ASTSharedOccurrencesIterator) (bool, error) {
	if e := it.check(); e != nil {
		return false, e
	}
	if e := other.check(); e != nil {
		return false, e
	}
	return it.owner == other.owner && it.index == other.index, nil
}
