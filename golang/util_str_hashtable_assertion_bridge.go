// Source: src/util/debug.cpp:28-36; src/util/debug.h:60-65.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "github.com/glycerine/g3/golang/arena"

// Cold binding identity only; no pointed assertion/action readiness is read.
func (d *DebugContext) StrHashtableAssertionBudget() *arena.Budget {
	if d == nil || d.self != d {
		return nil
	}
	return d.budget
}
