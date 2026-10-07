// Source: src/ast/bv_decl_plugin.cpp:132-141,437-468;
// src/util/vector.h:407-436,504-512.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
	"reflect"
)

func (p *astBitVectorPluginOwner) resizeBitRows(needed uint32) error {
	if needed <= p.bit2boolSize {
		return nil
	}
	recordBytes := uint64(reflect.TypeOf(astFormatRefVector{}).Size())
	for uint64(len(p.bit2bool)) < uint64(needed) {
		n := (uint64(len(p.bit2bool))*3 + 1) / 2
		if n < 2 {
			n = 2
		}
		if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/recordBytes {
			return arena.ErrBounds
		}
		if e := p.owner.BudgetField.Reserve(n * recordBytes); e != nil {
			return e
		}
		rows := make([]astFormatRefVector, int(n))
		copy(rows, p.bit2bool[:p.bit2boolSize])
		p.owner.BudgetField.Release(uint64(len(p.bit2bool)) * recordBytes)
		p.bit2bool = rows
	}
	// Default ptr_vector construction has no child allocation. Logical size is
	// published only after growth, preserving completed old row ownership.
	p.bit2boolSize = needed
	return nil
}

func (m *ASTManager) bitVectorMKBV(domain []ASTHandle) (astNodeRef, error) {
	p := m.bitVectorPlugin
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	for _, sort := range domain {
		r, e := m.requireKind(sort, ASTSort)
		if e != nil {
			return astNodeRef{}, e
		}
		if r != m.boolSort {
			return astNodeRef{}, bvDeclarationError("invalid mkbv operator")
		}
	}
	if uint64(len(domain)) >= math.MaxUint32 {
		return astNodeRef{}, ErrASTDomain
	}
	if p.fixedPending.slot != 0 {
		return astNodeRef{}, ErrASTPending
	}
	width := uint32(len(domain))
	v := &p.fixed[56]
	if e := v.resizeNull(m.BudgetField, width+1); e != nil {
		return astNodeRef{}, e
	}
	r, e := v.at(width)
	if e != nil || r.slot != 0 {
		return r, e
	}
	result, e := p.getSort(width)
	if e != nil {
		return astNodeRef{}, e
	}
	r, e = m.pluginDecl(p.names[9], p.family, 56, nil, domain, result, ASTFuncFlags{}, false)
	if e != nil {
		return astNodeRef{}, e
	}
	return r, p.publishFixed(56, width, r)
}

func (m *ASTManager) bitVectorBit2Bool(width uint32, ps []ASTParameter, domain []ASTHandle) (astNodeRef, error) {
	p := m.bitVectorPlugin
	if len(ps) != 1 {
		return astNodeRef{}, bvDeclarationError("invalid bit2bool declaration")
	}
	parameter, e := m.parameter(ps[0])
	if e != nil {
		return astNodeRef{}, e
	}
	if parameter.Kind != ASTParameterInt || len(domain) != 1 || parameter.integer >= int32(width) {
		return astNodeRef{}, bvDeclarationError("invalid bit2bool declaration")
	}
	if width == math.MaxUint32 {
		return astNodeRef{}, ErrASTDomain
	}
	if p.fixedPending.slot != 0 {
		return astNodeRef{}, ErrASTPending
	}
	if e = p.resizeBitRows(width + 1); e != nil {
		return astNodeRef{}, e
	}
	v := &p.bit2bool[width]
	if e = v.resizeNull(m.BudgetField, width); e != nil {
		return astNodeRef{}, e
	}
	index := uint32(parameter.integer)
	// Negative indices pass the original signed comparison, then convert to
	// unsigned and read outside the source vector. Reject that undefined access
	// after preserving the reached outer/inner resize effects.
	if index >= width {
		return astNodeRef{}, ErrASTDomain
	}
	r, e := v.at(index)
	if e != nil || r.slot != 0 {
		return r, e
	}
	r, e = m.pluginDecl(p.names[8], p.family, 55, ps, domain, m.boolSort, ASTFuncFlags{}, false)
	if e != nil {
		return astNodeRef{}, e
	}
	v.words[2+index*2], v.words[3+index*2] = r.slot, r.generation
	p.fixedPending, p.fixedPendingKind, p.fixedPendingWidth, p.fixedPendingIndex = r, 55, width, index
	if e = m.incNode(r); e != nil {
		return r, e
	}
	p.fixedPending = astNodeRef{}
	return r, nil
}

func (p *astBitVectorPluginOwner) booleanBitsBytes() uint64 {
	n := uint64(len(p.bit2bool))*uint64(reflect.TypeOf(astFormatRefVector{}).Size()) + p.fixed[56].bytes()
	for i := uint32(0); i < p.bit2boolSize; i++ {
		n += p.bit2bool[i].bytes()
	}
	return n
}
func (p *astBitVectorPluginOwner) releaseBooleanBits() {
	for i := uint32(0); i < p.bit2boolSize; i++ {
		p.bit2bool[i].close(p.owner.BudgetField)
	}
	p.owner.BudgetField.Release(uint64(len(p.bit2bool)) * uint64(reflect.TypeOf(astFormatRefVector{}).Size()))
	p.bit2bool = nil
	p.bit2boolSize = 0
	p.fixed[56].close(p.owner.BudgetField)
}
func (p *astBitVectorPluginOwner) finalizeBooleanBits() error {
	for width := uint32(0); width < p.bit2boolSize; width++ {
		if e := p.finalizeBitVectorCache(&p.bit2bool[width]); e != nil {
			return e
		}
	}
	return p.finalizeBitVectorCache(&p.fixed[56])
}
