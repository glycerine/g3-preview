// Source: src/util/debug.cpp:28-36; src/util/debug.h:60-65.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Cold binding identity only; no pointed assertion/action readiness is read.

func (d *DebugContext) StrHashtableAssertionBudget() *arena.Budget {
	if d == nil || d.Self != d {
		return nil
	}
	return d.BudgetField
}
