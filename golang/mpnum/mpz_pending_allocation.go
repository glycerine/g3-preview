package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// CleanupPendingAllocation explicitly releases physical storage whose source
// allocation completed before Go descriptor preparation failed. It preserves
// published numerals, failed tokens, scratch holes and source counter history.
// It does not finalize the source manager or make a failed numeral readable.
func (m *IntegerManager) CleanupPendingAllocation() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if e := m.cleanupPendingAllocation(); e != nil {
		return e
	}
	if m.allocator != nil {
		return m.allocator.CleanupPending()
	}
	return nil
}

func (m *IntegerManager) cleanupPendingAllocation() error {
	if m.pendingCell.object != (memory.ObjectRef{}) {
		progress, e := m.allocator.DeallocateTracked(m.pendingCell.requested, m.pendingCell.object)
		if progress.PhysicalReleased || progress.PooledLinked {
			m.pendingCell = sourceCellBacking{}
		}
		if e != nil {
			return e
		}
		m.pendingCell = sourceCellBacking{}
	}
	return nil
}
