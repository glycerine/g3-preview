// Source: src/util/mpf.cpp:61-67; src/util/mpf.h:241-259.
// Source author: Christoph Wintersteiger (2011-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import "sync/atomic"

// NewArbitraryFloatManager constructs an independent MPF member using the
// service's actual allocator namespace. The returned owner borrows the Runtime;
// it does not own this service or any of its rational values. This is the
// explicit Go binding for the source default/global allocator namespace.
func (s *RationalService) NewArbitraryFloatManager() (*ArbitraryFloatManager, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	c := s.initializationConfig.Integers
	c.Memory = s.manager.RuntimeOwner()
	atomic.StoreUint32(&s.sourceRuntimeExecuting, 1)
	defer atomic.StoreUint32(&s.sourceRuntimeExecuting, 0)
	return NewArbitraryFloatManager(c)
}
