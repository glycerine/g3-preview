// Source: src/ast/ast.cpp:1297-1479; src/ast/ast.h:1454-1490.
package ast

import (
	"reflect"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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
	s := ASTManagerStats{LiveNodes: m.liveNodes, LiveParameters: m.liveParameters, LiveInfos: m.liveInfos, LiveReferences: m.liveReferences, ArrayLiveCells: m.exprArrays.liveCells, OwnerBytes: m.ownerBytes, PageBytes: m.pageBytes, DirectoryBytes: m.directoryBytes}
	s.AllocatedBytes = m.datatypePluginBytes() + m.auxiliaryPluginBytes() + m.floatingPointPluginBytes() + m.sequencePluginBytes() + m.plugins.bytes() + m.arrayCardinalityBytes() + m.arrayPluginBytes() + m.arithmeticPluginBytes() + m.bitVectorPluginBytes() + m.formatPluginBytes() + m.formatFramesBytes() + m.ppFramesBytes() + s.OwnerBytes + s.PageBytes + s.DirectoryBytes + uint64(cap(m.builtin.userNames))*uint64(reflect.TypeOf(usymbol.Symbol{}).Size())
	for _, state := range m.scopedExprSubstitutions {
		s.AllocatedBytes += state.header
	}
	for _, state := range m.exprSubstitutions {
		s.AllocatedBytes += state.header
	}
	s.AllocatedBytes += m.Limit.StorageBytes()
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
