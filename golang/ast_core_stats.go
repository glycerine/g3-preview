// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "reflect"

func (m *ASTManager) Stats() ASTManagerStats {
	if m == nil || m.self != m {
		return ASTManagerStats{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ASTManagerStats{}
	}
	s := ASTManagerStats{LiveNodes: m.liveNodes, LiveParameters: m.liveParameters, LiveInfos: m.liveInfos, LiveReferences: m.liveReferences, ArrayLiveCells: m.exprArrays.liveCells, OwnerBytes: m.ownerBytes, PageBytes: m.pageBytes, DirectoryBytes: m.directoryBytes, TableSlots: m.table.slots, TableUsedSlots: m.table.used, TableCollisions: m.table.collisions}
	s.AllocatedBytes = s.OwnerBytes + s.PageBytes + s.DirectoryBytes + astChainBytes(uint32(len(m.table.cells))) + m.builtin.userTable.bytes + uint64(cap(m.builtin.userNames))*uint64(reflect.TypeOf(Symbol{}).Size())
	width := uint64(reflect.TypeOf(astNodeRef{}).Size())
	s.AllocatedBytes += uint64(cap(m.basic.eq)+cap(m.basic.oeq)+cap(m.basic.ite)) * width
	for _, v := range m.basic.proofCaches {
		s.AllocatedBytes += uint64(cap(v)) * width
	}
	s.AllocatedBytes += uint64(cap(m.exprIDs.free)+cap(m.declIDs.free))*4 + uint64(cap(m.exprArrays.getTmp)+cap(m.exprArrays.rerootTmp))*8
	if m.allocator != nil {
		a := m.allocator.Stats()
		s.AllocatedBytes += a.OwnerBytes + a.PayloadBytes + a.MetadataBytes + a.DirectoryBytes
	}
	if m.vectorAllocator != nil {
		a := m.vectorAllocator.Stats()
		s.AllocatedBytes += a.OwnerBytes + a.PayloadBytes + a.MetadataBytes + a.DirectoryBytes
	}
	if m.families != nil {
		f := m.families.Stats()
		s.AllocatedBytes += f.OwnerBytes + f.NamesBytes + f.Table.OwnerBytes + f.Table.TableBytes + f.Table.TrailBytes + f.Table.ScopeBytes
	}
	if m.format != nil {
		s.AllocatedBytes += m.format.Stats().AllocatedBytes
	}
	return s
}
