// Source: src/api/api_context.cpp:35-55,109-119,215-246;
// src/api/api_context.h:78-84; src/api/api_util.h:37-49; src/util/ref.h:79-85.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package api

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

var errAPIObjectOwner = errors.New("api: invalid, copied or retired context result/object owner")

// This is the result/object member portion of api::context, borrowing the real
// AST manager. It is not a replacement for the unported context constructor,
// which must also own its services, plugin utilities and tactic registry.
// The scalar u_map preserves source bucket order; its values index a cold
// directory of API wrappers, never an interface in each hash-table entry.
type apiContextResults struct {
	self                          *apiContextResults
	manager                       *ast.ASTManager
	lastResult, astTrail          ast.ASTRefVector
	lastObject, pendingDestroy    *apiObject
	allocated                     util.UintMap
	objects                       []*apiObject
	freeIDs                       []uint32
	ownerBytes                    uint64
	userRefCount, closing, closed bool
}

// One cold virtual destructor per API wrapper translates api::object's virtual
// destructor. Derived goals/models/solvers retain their own source AST roots.
type apiObjectDestructor interface{ apiDestroy() error }
type apiObject struct {
	self           *apiObject
	context        *apiContextResults
	destructor     apiObjectDestructor
	id, references uint32
	ownerBytes     uint64
	closed         bool
}

func newAPIContextResults(m *ast.ASTManager, rc bool) (c *apiContextResults, err error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err = m.ProofsEnabled(); err != nil {
		return nil, err
	}
	n := uint64(reflect.TypeOf(apiContextResults{}).Size())
	if err = m.Budget().Reserve(n); err != nil {
		return nil, err
	}
	c = &apiContextResults{manager: m, userRefCount: rc, ownerBytes: n}
	c.self = c
	// The source declares both vectors before its object table.
	if c.lastResult, err = m.NewRefVector(); err != nil {
		m.Budget().Release(n)
		return nil, err
	}
	if c.astTrail, err = m.NewRefVector(); err != nil {
		cleanup := m.DelRefVector(&c.lastResult)
		m.Budget().Release(n)
		return nil, errors.Join(err, cleanup)
	}
	if err = util.InitializeInlineUintMap(&c.allocated, util.UintMapConfig{Budget: m.Budget(), OutOfMemory: m.IsOutOfMemory}); err != nil {
		cleanup := errors.Join(m.DelRefVector(&c.astTrail), m.DelRefVector(&c.lastResult))
		m.Budget().Release(n)
		return nil, errors.Join(err, cleanup)
	}
	return c, nil
}

func (c *apiContextResults) open() error {
	if c == nil || c.self != c {
		return errAPIObjectOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	if c.closing {
		return ast.ErrASTPending
	}
	return nil
}

func (c *apiContextResults) saveASTTrail(h ast.ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	if !c.userRefCount {
		return c.manager.RefVectorPush(c.astTrail, h)
	}
	// Keep an alias alive across reset, exactly as context.cpp:220-225.
	if e := c.manager.IncRef(h); e != nil {
		return e
	}
	if e := c.manager.RefVectorReset(c.lastResult); e != nil {
		return e
	}
	if e := c.manager.RefVectorPush(c.lastResult, h); e != nil {
		return e
	}
	return c.manager.DecRef(h)
}

func (c *apiContextResults) saveMultipleASTTrail(h ast.ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	if c.userRefCount {
		return c.manager.RefVectorPush(c.lastResult, h)
	}
	return c.manager.RefVectorPush(c.astTrail, h)
}

func (c *apiContextResults) resetLastResult() error {
	if e := c.open(); e != nil {
		return e
	}
	if c.userRefCount {
		if e := c.manager.RefVectorReset(c.lastResult); e != nil {
			return e
		}
	}
	return c.assignLastObject(nil)
}

func (c *apiContextResults) saveObject(o *apiObject) error {
	if e := c.open(); e != nil {
		return e
	}
	return c.assignLastObject(o)
}

func (c *apiContextResults) assignLastObject(o *apiObject) error {
	if o != nil {
		if o.context != c {
			return errAPIObjectOwner
		}
		if e := o.incRef(); e != nil {
			return e
		}
	}
	if c.lastObject != nil {
		if e := c.lastObject.decRef(); e != nil {
			return e
		}
	}
	c.lastObject = o
	return nil
}

func (c *apiContextResults) growObjectDirectory(id uint32) error {
	if uint64(id) < uint64(len(c.objects)) {
		return nil
	}
	stride := uint64(reflect.TypeOf((*apiObject)(nil)).Size())
	for uint64(id) >= uint64(cap(c.objects)) {
		n, e := util.SourceVectorCapacity(cap(c.objects), stride)
		if e != nil {
			return e
		}
		if e = c.manager.Budget().Reserve(uint64(n) * stride); e != nil {
			return e
		}
		v := make([]*apiObject, len(c.objects), n)
		copy(v, c.objects)
		c.manager.Budget().Release(uint64(cap(c.objects)) * stride)
		c.objects = v
	}
	c.objects = c.objects[:int(id)+1]
	return nil
}

func (c *apiContextResults) addObject(o *apiObject) (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	id := c.allocated.Size()
	if n := len(c.freeIDs); n != 0 {
		id = c.freeIDs[n-1]
		c.freeIDs = c.freeIDs[:n-1]
	}
	if e := c.growObjectDirectory(id); e != nil {
		return 0, e
	}
	if e := c.allocated.Insert(id, id); e != nil {
		return 0, e
	}
	c.objects[id] = o
	return id, nil
}

func (c *apiContextResults) pushFreeID(id uint32) error {
	if len(c.freeIDs) == cap(c.freeIDs) {
		n, e := util.SourceVectorCapacity(cap(c.freeIDs), 4)
		if e != nil {
			return e
		}
		if e = c.manager.Budget().Reserve(uint64(n) * 4); e != nil {
			return e
		}
		v := make([]uint32, len(c.freeIDs), n)
		copy(v, c.freeIDs)
		c.manager.Budget().Release(uint64(cap(c.freeIDs)) * 4)
		c.freeIDs = v
	}
	c.freeIDs = append(c.freeIDs, id)
	return nil
}

func (c *apiContextResults) delObject(o *apiObject) error {
	// Source publication order is free-list push, table removal, deallocation.
	if e := c.pushFreeID(o.id); e != nil {
		return e
	}
	c.pendingDestroy = o
	if e := c.allocated.Remove(o.id); e != nil {
		return e
	}
	return c.destroyObject(o)
}

func (c *apiContextResults) destroyObject(o *apiObject) error {
	c.pendingDestroy = o
	if e := o.destroy(); e != nil {
		c.pendingDestroy = o
		return e
	}
	c.objects[o.id] = nil
	c.pendingDestroy = nil
	return nil
}

func (o *apiObject) open() error {
	if o == nil || o.self != o || o.closed || o.context == nil || o.context.self != o.context || o.context.closed {
		return errAPIObjectOwner
	}
	if uint64(o.id) >= uint64(len(o.context.objects)) || o.context.objects[o.id] != o || !o.context.allocated.Contains(o.id) {
		return errAPIObjectOwner
	}
	return nil
}
func (o *apiObject) incRef() error {
	if e := o.open(); e != nil {
		return e
	}
	o.references++
	return nil
}
func (o *apiObject) decRef() error {
	if e := o.open(); e != nil {
		return e
	}
	if o.references == 0 {
		return errAPIObjectOwner
	}
	o.references--
	if o.references == 0 {
		return o.context.delObject(o)
	}
	return nil
}
func (o *apiObject) destroy() error {
	if o.closed {
		return nil
	}
	if e := o.destructor.apiDestroy(); e != nil {
		return e
	}
	o.closed = true
	o.context.manager.Budget().Release(o.ownerBytes)
	o.ownerBytes = 0
	return nil
}

func (c *apiContextResults) Close() error {
	if c == nil || c.self != c {
		return errAPIObjectOwner
	}
	if c.closed {
		return nil
	}
	c.closing = true
	if c.pendingDestroy != nil {
		if e := c.destroyObject(c.pendingDestroy); e != nil {
			return e
		}
	}
	// A returning Go failure may have consumed the source DEC before its
	// free-list push or derived destructor refused. Owning teardown drains that
	// zero-ref wrapper from the registry rather than repeating the DEC.
	if c.lastObject != nil && (c.lastObject.closed || c.lastObject.references == 0) {
		c.lastObject = nil
	}
	if e := c.assignLastObject(nil); e != nil {
		return e
	}
	// Restart begin after each removal: a derived destructor may release other
	// registered objects, and the source hash table can compact during removal.
	for !c.allocated.Empty() {
		entry, e := c.allocated.Begin().Entry()
		if e != nil {
			return e
		}
		o := c.objects[entry.Value]
		c.pendingDestroy = o
		if e = c.allocated.Remove(entry.Key); e != nil {
			return e
		}
		if e = c.destroyObject(o); e != nil {
			return e
		}
	}
	c.manager.Budget().Release(uint64(cap(c.freeIDs)) * 4)
	c.freeIDs = nil
	c.allocated.Close()
	c.manager.Budget().Release(uint64(cap(c.objects)) * uint64(reflect.TypeOf((*apiObject)(nil)).Size()))
	c.objects = nil
	if c.astTrail != (ast.ASTRefVector{}) {
		if e := c.manager.DelRefVector(&c.astTrail); e != nil {
			return e
		}
	}
	if c.lastResult != (ast.ASTRefVector{}) {
		if e := c.manager.DelRefVector(&c.lastResult); e != nil {
			return e
		}
	}
	c.manager.Budget().Release(c.ownerBytes)
	c.ownerBytes = 0
	c.closed = true
	return nil
}
