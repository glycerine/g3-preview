// Source: src/test/sorting_network.cpp:150-181; src/util/ref_vector.h:53,96-100.
package ast

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// PsortASTContext is the actual ast_ext2 specialization. Manager ownership
// remains borrowed. Only clauses, trail and acquired unpublished INC frames
// are owned here; factory-returned zero-reference pointers stay manager-owned.
type PsortASTContext struct {
	self                              *PsortASTContext
	manager                           *ASTManager
	budget                            *arena.Budget
	header                            uint64
	clauses, trail                    ASTRefVector
	completed                         uint8
	constructed, closed, executing    bool
	stage                             uint8
	constructorCause, destructorCause error
	pending                           *psortASTPushFrame
}
type psortASTPushFrame struct {
	next    *psortASTPushFrame
	owner   *PsortASTContext
	header  uint64
	receipt ASTRefVectorPushReceipt
}

func NewPsortASTContext(m *ASTManager) (*PsortASTContext, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if e := m.lock(); e != nil {
		return nil, e
	}
	m.mu.Unlock()
	b := m.BudgetField
	bytes := uint64(reflect.TypeOf(PsortASTContext{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	c := &PsortASTContext{manager: m, budget: b, header: bytes}
	c.self = c
	var e error
	c.clauses, e = m.NewRefVector()
	if e == nil {
		c.completed = 1
		c.trail, e = m.NewRefVector()
	}
	if e == nil {
		c.completed = 3
		c.constructed = true
		return c, nil
	}
	c.constructorCause = e
	// Incomplete outer construction unwinds only completed source members.
	unwind := c.finalizeMembers()
	return c, errors.Join(e, unwind)
}
func (c *PsortASTContext) identity() error {
	if c == nil || c.self != c {
		return ErrASTHandle
	}
	if c.closed {
		return arena.ErrClosed
	}
	if c.executing {
		return ErrASTPending
	}
	return nil
}
func (c *PsortASTContext) open() error {
	if e := c.identity(); e != nil {
		return e
	}
	if !c.constructed || c.destructorCause != nil || c.stage != 0 {
		return ErrASTPending
	}
	return nil
}
func (c *PsortASTContext) push(v ASTRefVector, h ASTHandle) error {
	// Real cold frame preparation precedes INC; it owns no reference yet.
	bytes := uint64(reflect.TypeOf(psortASTPushFrame{}).Size())
	if e := c.budget.Reserve(bytes); e != nil {
		return e
	}
	f := &psortASTPushFrame{owner: c, header: bytes, next: c.pending}
	c.pending = f
	e := c.manager.RefVectorPushTracked(v, h, &f.receipt)
	if !f.receipt.acquired || f.receipt.published {
		c.pending = f.next
		c.budget.Release(f.header)
		f.header = 0
	}
	return e
}
func (c *PsortASTContext) False() (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	return c.manager.MkFalse()
}
func (c *PsortASTContext) True() (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	return c.manager.MkTrue()
}
func (c *PsortASTContext) Fresh() (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	c.executing = true
	defer func() { c.executing = false }()
	sort, e := c.manager.MkBoolSort()
	if e != nil {
		return ASTHandle{}, e
	}
	h, e := c.manager.MkFreshConst("x", sort)
	if e != nil {
		return h, e
	}
	return h, c.push(c.trail, h)
}
func (c *PsortASTContext) Min(a, b ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	c.executing = true
	defer func() { c.executing = false }()
	h, e := c.manager.MkAnd(a, b)
	if e != nil {
		return h, e
	}
	return h, c.push(c.trail, h)
}
func (c *PsortASTContext) Max(a, b ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	c.executing = true
	defer func() { c.executing = false }()
	h, e := c.manager.MkOr(a, b)
	if e != nil {
		return h, e
	}
	return h, c.push(c.trail, h)
}
func (c *PsortASTContext) Not(a ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	c.executing = true
	defer func() { c.executing = false }()
	yes, child, e := c.manager.IsNot(a)
	if e != nil {
		return ASTHandle{}, e
	}
	if yes {
		return child, nil
	}
	h, e := c.manager.MkNot(a)
	if e != nil {
		return h, e
	}
	return h, c.push(c.trail, h)
}
func (c *PsortASTContext) Clause(n uint32, span ASTLiteralVectorSpan) error {
	if e := c.open(); e != nil {
		return e
	}
	c.executing = true
	defer func() { c.executing = false }()
	h, e := c.manager.MkOrSpan(n, span)
	if e != nil {
		return e
	}
	return c.push(c.clauses, h)
}
func (c *PsortASTContext) finalizeMembers() error {
	if c.stage == 0 {
		if c.completed&2 != 0 {
			if e := c.manager.DelRefVector(&c.trail); e != nil {
				return e
			}
			c.completed &^= 2
		}
		c.stage = 1
	}
	if c.stage == 1 {
		if c.completed&1 != 0 {
			if e := c.manager.DelRefVector(&c.clauses); e != nil {
				return e
			}
			c.completed &^= 1
		}
		c.stage = 2
	}
	return nil
}
func (c *PsortASTContext) SourceFinalize() error {
	if e := c.identity(); e != nil {
		return e
	}
	if c.destructorCause != nil {
		return c.destructorCause
	}
	if !c.constructed {
		return nil
	}
	c.executing = true
	defer func() { c.executing = false }()
	if e := c.finalizeMembers(); e != nil {
		c.destructorCause = e
		return e
	}
	return nil
}
func (c *PsortASTContext) Close() error {
	if c != nil && c.self == c && c.closed {
		return nil
	}
	if e := c.identity(); e != nil {
		return e
	}
	c.executing = true
	defer func() { c.executing = false }()
	// Explicit owning continuation may resume the child adapter's consumed
	// reference warning. Ordinary source teardown remains at its sticky cause.
	if e := c.finalizeMembers(); e != nil {
		if c.destructorCause == nil {
			c.destructorCause = e
		}
		return e
	}
	for c.pending != nil {
		f := c.pending
		if f.owner != c {
			return ErrASTHandle
		}
		r := &f.receipt
		h := ASTHandle{origin: c.manager.origin, slot: r.node.slot, generation: r.node.generation}
		if e := c.manager.CompleteRefVectorPush(r.vector, h, r); e != nil {
			return e
		}
		c.pending = f.next
		c.budget.Release(f.header)
		f.header = 0
	}
	c.budget.Release(c.header)
	c.header = 0
	c.closed = true
	return nil
}
