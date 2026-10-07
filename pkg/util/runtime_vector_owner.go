package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

const RuntimeVectorSourceDebug = sourceDebug

func (d *DebugContext) Budget() *arena.Budget {
	if d == nil {
		return nil
	}
	return d.BudgetField
}
func (d *DebugContext) ValidOwner() bool { return d != nil && d.Self == d }
