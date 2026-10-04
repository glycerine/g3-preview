// Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Failed construction destroys completed members in reverse declaration order.
// mpz/raw pointer fields do not own destructors; no six-cell del body runs.
func (m *IntegerManager) unwindConstructorMembers() error {
	for m.constructorUnwindStage < 2 {
		if m.constructorUnwindStage == 0 {
			if m.natural != nil {
				if e := m.natural.CloseChecked(); e != nil {
					return e
				}
				m.natural = nil
			}
		} else if m.allocator != nil {
			e := m.allocator.Close()
			if m.cells != nil {
				if mark := m.markSourceFreedPools(); mark != nil {
					return mark
				}
			}
			if e != nil {
				return e
			}
		}
		m.constructorUnwindStage++
	}
	m.constructorUnwound = true
	m.cleanupCause = nil
	return nil
}
func (m *IntegerManager) hasConstructorPhysicalRoots() bool {
	if m.pendingCell.object.OwnerID() != 0 {
		if m.pendingCell.offset == 0 {
			return true
		}
		// A saved pooled token is not physical storage after member chunk free.
		_, e := m.memory.ReadUint32(m.pendingCell.raw, m.pendingCell.offset+4)
		if e != memory.ErrRuntimeOwner {
			return true
		}
	}
	if m.cells != nil {
		for n := uint64(1); n <= uint64(m.cells.Stats().Slots); n++ {
			c, e := m.cells.Read(arena.Slot(n))
			if e == nil && !c.released {
				return true
			}
		}
	}
	return false
}
