// Licensed under the MIT license.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Caller-confined cold owner for separate source positions/substs output vectors. Their
// allocation and publication sequences remain independent. Premises and
// conclusion are separately supplied actual AST owning references.
type ASTProofHyperOutput struct {
	self                                         *ASTProofHyperOutput
	manager                                      *ASTManager
	positions                                    []ASTProofPosition
	substitutions                                []ASTRefVector
	ownerBytes, positionBytes, substitutionBytes uint64
	closed                                       bool
}

func NewASTProofHyperOutput(m *ASTManager) (*ASTProofHyperOutput, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	n := uint64(reflect.TypeOf(ASTProofHyperOutput{}).Size())
	if e := m.BudgetField.Reserve(n); e != nil {
		return nil, e
	}
	v := &ASTProofHyperOutput{manager: m, ownerBytes: n}
	v.self = v
	return v, nil
}
func (v *ASTProofHyperOutput) check(m *ASTManager) error {
	if v == nil || v.self != v || v.manager != m {
		return ErrASTHandle
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *ASTProofHyperOutput) PositionsLen() uint32 {
	if v == nil || v.closed || v.self != v {
		return 0
	}
	return uint32(len(v.positions))
}
func (v *ASTProofHyperOutput) SubstitutionsLen() uint32 {
	if v == nil || v.closed || v.self != v {
		return 0
	}
	return uint32(len(v.substitutions))
}
func (v *ASTProofHyperOutput) Position(i uint32) (ASTProofPosition, error) {
	if v == nil {
		return ASTProofPosition{}, ErrASTHandle
	}
	if e := v.check(v.manager); e != nil {
		return ASTProofPosition{}, e
	}
	if e := v.manager.lock(); e != nil {
		return ASTProofPosition{}, e
	}
	defer v.manager.mu.Unlock()
	if uint64(i) >= uint64(len(v.positions)) {
		return ASTProofPosition{}, arena.ErrBounds
	}
	return v.positions[i], nil
}

// Substitution returns a borrowed scalar vector token; the output owns its
// destruction. It can be queried/mutated through the actual AST manager.
func (v *ASTProofHyperOutput) Substitution(i uint32) (ASTRefVector, error) {
	if v == nil {
		return ASTRefVector{}, ErrASTHandle
	}
	if e := v.check(v.manager); e != nil {
		return ASTRefVector{}, e
	}
	if e := v.manager.lock(); e != nil {
		return ASTRefVector{}, e
	}
	defer v.manager.mu.Unlock()
	if uint64(i) >= uint64(len(v.substitutions)) {
		return ASTRefVector{}, arena.ErrBounds
	}
	return v.substitutions[i], nil
}
func hyperVectorCapacity(old int, width uint64, sourceWidth uint64) (int, error) {
	n := uint64(2)
	if old != 0 {
		n = (3*uint64(old) + 1) / 2
	}
	if n <= uint64(old) || n > (math.MaxUint32-8)/sourceWidth || n > uint64(^uint(0)>>1)/width {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
func (v *ASTProofHyperOutput) pushPosition(m *ASTManager, p ASTProofPosition) error {
	if len(v.positions) == cap(v.positions) {
		width := uint64(reflect.TypeOf(ASTProofPosition{}).Size())
		n, e := hyperVectorCapacity(cap(v.positions), width, 8)
		if e != nil {
			return e
		}
		bytes := uint64(n) * width
		if e = m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		next := make([]ASTProofPosition, len(v.positions), n)
		copy(next[:cap(next)], v.positions[:cap(v.positions)])
		m.BudgetField.Release(v.positionBytes)
		v.positionBytes = bytes
		v.positions = next
	}
	v.positions = append(v.positions, p)
	return nil
}
func (v *ASTProofHyperOutput) pushSubstitution(m *ASTManager) error {
	if len(v.substitutions) == cap(v.substitutions) {
		width := uint64(reflect.TypeOf(ASTRefVector{}).Size())
		n, e := hyperVectorCapacity(cap(v.substitutions), width, 16)
		if e != nil {
			return e
		}
		bytes := uint64(n) * width
		if e = m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		next := make([]ASTRefVector, len(v.substitutions), n)
		copy(next, v.substitutions)
		m.BudgetField.Release(v.substitutionBytes)
		v.substitutionBytes = bytes
		v.substitutions = next
	}
	// Source constructs an empty stack temporary before push (no heap or
	// AST-reference action), grows the outer vector, then move-constructs
	// its element. The extra cold scalar owner record reserved after growth
	// is the checked Go stable-owner adapter, not a source allocation.
	inner, e := m.newRefVector()
	if e != nil {
		return e
	}
	v.substitutions = append(v.substitutions, inner)
	return nil
}
func (v *ASTProofHyperOutput) Close() error {
	if v == nil || v.self != v {
		return ErrASTHandle
	}
	if v.closed {
		return nil
	}
	m := v.manager
	if m == nil || m.Self != m {
		return ErrASTHandle
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var failure error
	if !m.Closed {
		for _, inner := range v.substitutions {
			if e := m.finalizeVector(inner); failure == nil {
				failure = e
			}
			if e := m.allocator.Deallocate(astRefVectorRecordBytes, inner.record); failure == nil {
				failure = e
			}
		}
	}
	// A closed manager has already force-released AST nodes and bulk-freed
	// scalar vector owners. Refund caller-owned outer storage without calling
	// refcount callbacks against that freed graph (undefined source lifetime).
	m.BudgetField.Release(v.positionBytes)
	m.BudgetField.Release(v.substitutionBytes)
	m.BudgetField.Release(v.ownerBytes)
	v.positions = nil
	v.substitutions = nil
	v.positionBytes = 0
	v.substitutionBytes = 0
	v.ownerBytes = 0
	v.closed = true
	return failure
}
