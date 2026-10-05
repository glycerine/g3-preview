// Source: src/util/memory_manager.cpp:55-72; src/util/small_object_allocator.cpp:172-242.
// Licensed under the MIT license. See LICENSE.txt.
// Source author: Nikolaj Bjorner (nbjorner), 2007.
package g3

import "errors"

// The selected file provider uses direct Go writes. It borrows the file and
// represents default nonthrowing ostream badbit; native buffering and locale
// profiles remain outside this adapter.
var ErrNumericStreamFile = errors.New("g3: file stream has no string buffer")

func (s *NumericTextStream) Bad() (bool, error) {
	if e := s.lock(); e != nil {
		return false, e
	}
	defer s.mu.Unlock()
	return s.fileBad, nil
}
