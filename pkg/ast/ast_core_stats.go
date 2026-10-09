// Source: src/ast/ast.cpp:1297-1479; src/ast/ast.h:1454-1490.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

func (m *ASTManager) Stats() ASTManagerStats {
	if m == nil || m.Self != m {
		return ASTManagerStats{}
	}
	if m.traceExecuting || m.formatExecuting {
		panic(ErrASTPending)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return ASTManagerStats{}
	}
	s := ASTManagerStats{LiveNodes: m.liveNodes, LiveParameters: m.liveParameters, LiveInfos: m.liveInfos, LiveReferences: m.liveReferences, ArrayLiveCells: m.exprArrays.liveCells, OwnerBytes: m.ownerBytes, PageBytes: m.pageBytes, DirectoryBytes: m.directoryBytes, TableSlots: m.table.slots, TableUsedSlots: m.table.used, TableCollisions: m.table.collisions}
	s.AllocatedBytes = m.datatypePluginBytes() + m.auxiliaryPluginBytes() + m.floatingPointPluginBytes() + m.sequencePluginBytes() + m.plugins.bytes() + m.arrayCardinalityBytes() + m.arrayPluginBytes() + m.arithmeticPluginBytes() + m.bitVectorPluginBytes() + m.formatPluginBytes() + m.formatFramesBytes() + m.ppFramesBytes() + m.LowLevel.marks.exprs.bytes + m.LowLevel.marks.decls.bytes + s.OwnerBytes + s.PageBytes + s.DirectoryBytes + astChainBytes(uint32(len(m.table.cells))) + m.builtin.userTable.StorageBytes() + uint64(cap(m.builtin.userNames))*uint64(reflect.TypeOf(util.Symbol{}).Size())
	s.AllocatedBytes += m.Limit.StorageBytes()
	s.AllocatedBytes += m.appCoercion.args.charged
	for f := m.traceAcquisitionHistory; f != nil; f = f.historyNext {
		s.AllocatedBytes += f.bytes
	}
	for f := m.traceAcquisitions; f != nil; f = f.previous {
		s.AllocatedBytes += f.bytes
	}
	if f := m.traceAcquisitionExecuting; f != nil {
		s.AllocatedBytes += f.bytes
	}
	s.AllocatedBytes += m.exprVisit.bytes + m.exprVisit.localFast.heapBytes + m.exprVisit.localBits.bytes
	for h := m.exprVisitHistory; h != nil; h = h.next {
		s.AllocatedBytes += h.bytes
	}
	width := uint64(reflect.TypeOf(astNodeRef{}).Size())
	s.AllocatedBytes += uint64(cap(m.basic.eq)+cap(m.basic.oeq)+cap(m.basic.ite)) * width
	for _, v := range m.basic.proofCaches {
		s.AllocatedBytes += uint64(cap(v)) * width
	}
	s.AllocatedBytes += uint64(m.ExprIDs.FreeCapacity()+m.declIDs.FreeCapacity())*4 + uint64(cap(m.exprArrays.getTmp)+cap(m.exprArrays.rerootTmp))*8
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
		s.AllocatedBytes += f.OwnerBytes
	}
	if m.Format != nil {
		s.AllocatedBytes += m.Format.Stats().AllocatedBytes
	}
	if m.constructorPendingFormat != nil {
		s.AllocatedBytes += m.constructorPendingFormat.Stats().AllocatedBytes
	}
	for _, file := range []*astTraceFile{m.TraceFile, m.constructorPendingFile} {
		if file != nil {
			x := file.stream.Stats()
			s.AllocatedBytes += file.header + x.ScratchBytes + x.OutputOwnerBytes + x.OutputOverflowBytes
		}
	}
	return s
}
