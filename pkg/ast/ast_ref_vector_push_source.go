// Source: src/util/ref_vector.h:96-100; src/ast/ast.h:2273-2277.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// This is an optional actual cold caller frame, not a hot pointer entry. It
// stamps only the real successful source INC and the real vector publication.
type ASTRefVectorPushReceipt struct {
	self                                        *ASTRefVectorPushReceipt
	owner                                       *ASTManager
	vector                                      ASTRefVector
	node                                        astNodeRef
	invocation                                  uint64
	acquired, published, consumed, acknowledged bool
	cause, cleanupCause                         error
}

func (r *ASTRefVectorPushReceipt) prepare(m *ASTManager, v ASTRefVector) error {
	if r == nil {
		return ErrASTHandle
	}
	if r.self != nil && r.self != r {
		return ErrASTHandle
	}
	if r.acquired && !r.acknowledged {
		return ErrASTPending
	}
	if m.refVectorPushSerial == math.MaxUint64 {
		return arena.ErrSlotLimit
	}
	return nil
}
func (m *ASTManager) RefVectorPushTracked(v ASTRefVector, h ASTHandle, r *ASTRefVectorPushReceipt) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if r == nil {
		return ErrASTHandle
	}
	node, e := m.vectorInput(h)
	if e != nil {
		return e
	}
	e = m.pushVectorTracked(v, node, r)
	return e
}

// CompleteRefVectorPush releases only this exact untransferred reference. The
// actual decNode consumed result is authority; an exceptional underlying node
// deletion remains named by the borrowed manager's own continuation fields.
func (m *ASTManager) CompleteRefVectorPush(v ASTRefVector, h ASTHandle, r *ASTRefVectorPushReceipt) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if r == nil || r.self != r || r.owner != m || r.vector != v || h.origin != m.origin || r.node != (astNodeRef{h.slot, h.generation}) || r.invocation == 0 || r.invocation > m.refVectorPushSerial || !r.acquired || r.published || r.acknowledged {
		return ErrASTHandle
	}
	if !r.consumed {
		consumed, e := m.decNode(r.node)
		if consumed {
			r.consumed = true
		}
		if e != nil {
			if r.cleanupCause == nil {
				r.cleanupCause = e
			}
			return e
		}
	}
	r.acknowledged = true
	return nil
}

// Unpublished reports this receipt's actual acquired reference awaiting caller
// cleanup. Published references belong to the vector, and empty receipts own none.
func (r *ASTRefVectorPushReceipt) Unpublished() (bool, error) {
	if r == nil {
		return false, ErrASTHandle
	}
	if r.self == nil {
		return false, nil
	}
	if r.self != r {
		return false, ErrASTHandle
	}
	return r.acquired && !r.published && !r.acknowledged, nil
}
