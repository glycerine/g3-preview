// Concrete source specialization: src/util/map.h:24-196; src/util/hashtable.h:44-63.
// Source: src/util/hashtable.h:159-215,249,300-303,371,395,422,451,559,643-644;
// src/util/debug.h:60-65,76-89.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

func (h *StrInt32Map) assertionsEnabled() (bool, error) {
	if !strHashtableSourceDebug {
		return false, nil
	}
	if h.config.Assertions == nil {
		return true, nil
	}
	return h.config.Assertions.AssertionsEnabled()
}
func (h *StrInt32Map) assertionFailure(line int32, condition string) error {
	if h.config.Assertions == nil {
		return ErrStrHashtableAssertion
	}
	if e := h.config.Assertions.NotifyAssertionViolation("src/util/hashtable.h", line, condition); e != nil {
		return e
	}
	return h.config.Assertions.InvokeAssertionDebugger()
}
func (h *StrInt32Map) unreachable(line int32) error {
	if !strHashtableSourceDebug {
		return ErrStrHashtableUnreachable
	}
	if h.config.Assertions == nil {
		return ErrStrHashtableUnreachable
	}
	return h.config.Assertions.UnreachableSource("src/util/hashtable.h", line)
}

// Two source loops start at hash&mask, run to the actual end, then restart at
// begin. This preserves the continued SASSERT non-power-of-two domain too.
