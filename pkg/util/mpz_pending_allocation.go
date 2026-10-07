// Source: src/util/mpz.h:124-149; src/util/mpz.cpp:142-161.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

// CleanupPendingAllocation explicitly releases physical storage whose source
// allocation completed before Go descriptor preparation failed. It preserves
// published numerals, failed tokens, scratch holes and source counter history.
// It does not finalize the source manager or make a failed numeral readable.

func (m *IntegerManager) CleanupPendingAllocation() error {
	if m == nil || m.self != m {
		return ErrRuntimeOwner
	}
	if m.closed {
		return arena.ErrClosed
	}
	if e := m.cleanupPendingAllocation(); e != nil {
		return e
	}
	if m.Allocator != nil {
		return m.Allocator.CleanupPending()
	}
	return nil
}

func (m *IntegerManager) cleanupPendingAllocation() error {
	if m.PendingCell.Object != (ObjectRef{}) {
		if m.ConstructorUnwound || m.SourceFinalized {
			if m.PendingCell.Offset != 0 {
				_, e := m.Memory.ReadUint32(m.PendingCell.Raw, m.PendingCell.Offset+4)
				if e != ErrRuntimeOwner {
					if e != nil {
						return e
					}
					return arena.ErrInvalidSlot
				}
				m.PendingCell = SourceCellBacking{}
				return nil
			}
			raw := m.PendingCell.Raw
			e := m.Memory.Deallocate(&raw)
			if raw == (Block{}) {
				m.PendingCell = SourceCellBacking{}
			}
			return e
		}
		progress, e := m.PendingCell.deallocate(m.Allocator)
		if progress.PhysicalReleased || progress.PooledLinked {
			m.PendingCell = SourceCellBacking{}
		}
		if e != nil {
			return e
		}
		m.PendingCell = SourceCellBacking{}
	}
	return nil
}
