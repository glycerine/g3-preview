package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

const RuntimeVectorSourceDebug = sourceDebug

var ErrRuntimeUnsignedGrowthExecuting = errors.New("g3: unsigned-vector acquisition is executing")

func (d *DebugContext) Budget() *arena.Budget {
	if d == nil {
		return nil
	}
	return d.BudgetField
}
func (d *DebugContext) ValidOwner() bool { return d != nil && d.Self == d }
