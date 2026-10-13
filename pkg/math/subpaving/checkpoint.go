// Source: src/math/subpaving/subpaving_t_def.h:461-467.
package subpaving

import (
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/messages"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

var ErrSubpavingMaxMemory = z3exception.NewDefaultException(messages.MaxMemory)

// sourceCheckpoint preserves the source order: resource increment precedes the
// process-global allocation getter. An unbound context cannot query that
// source process yet; the enclosing process owner must bind its runtimes.
func sourceCheckpoint(limit *control.Reslimit, process *memory.ProcessAccounting, maxMemory uint64, canceled error) error {
	if !limit.Inc() {
		return canceled
	}
	if process != nil && process.AllocationSize() > maxMemory {
		return ErrSubpavingMaxMemory
	}
	return nil
}
