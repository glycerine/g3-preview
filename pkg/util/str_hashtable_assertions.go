// Source: src/util/hashtable.h:159-215,249,300-303,371,395,422,451,559,643-644;
// src/util/debug.h:60-65,76-89.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

// One cold borrowed source assertion owner, never a hot entry or condition
// closure. Its flag lookup does not query the tag table being operated on.
type StrHashtableAssertionContext interface {
	StrHashtableAssertionBudget() *arena.Budget
	AssertionsEnabled() (bool, error)
	NotifyAssertionViolation(string, int32, string) error
	InvokeAssertionDebugger() error
	UnreachableSource(string, int32) error
}

var ErrStrHashtableAssertion = errors.New("g3: string-table source assertion")
var ErrStrHashtableUnreachable = errors.New("g3: string-table GNU unreachable guard")

func (h *StrHashtable) assertionsEnabled() (bool, error) {
	if !strHashtableSourceDebug {
		return false, nil
	}
	if h.config.Assertions == nil {
		return true, nil
	}
	return h.config.Assertions.AssertionsEnabled()
}
func (h *StrHashtable) assertionFailure(line int32, condition string) error {
	if h.config.Assertions == nil {
		return ErrStrHashtableAssertion
	}
	if e := h.config.Assertions.NotifyAssertionViolation("src/util/hashtable.h", line, condition); e != nil {
		return e
	}
	return h.config.Assertions.InvokeAssertionDebugger()
}
func (h *StrHashtable) unreachable(line int32) error {
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
func sourceProbeIndex(start, step, capacity uint32) uint32 {
	if step >= capacity-start {
		return step - (capacity - start)
	}
	return start + step
}
