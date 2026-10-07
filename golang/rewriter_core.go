// Source: src/ast/rewriter/rewriter.cpp:23-204;
// src/ast/rewriter/rewriter.h:31-119; src/ast/ast.h:845-853;
// src/util/vector.h:59-99; src/util/ref_vector.h:45-72,140-172.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Concrete source frame. The flags retain the original 1/1/2/2/26-bit fields;
// storing a state/depth/index truncates exactly as the unsigned bitfields do.
// All hot frame and scope records contain handles/scalars, never Go pointers.
type rewriterFrame struct {
	curr        ASTHandle
	flags, spos uint32
}

func (f *rewriterFrame) cacheResult() bool { return f.flags&1 != 0 }
func (f *rewriterFrame) newChild() bool    { return f.flags&2 != 0 }
func (f *rewriterFrame) state() uint32     { return (f.flags >> 2) & 3 }
func (f *rewriterFrame) maxDepth() uint32  { return (f.flags >> 4) & 3 }
func (f *rewriterFrame) index() uint32     { return f.flags >> 6 }
func (f *rewriterFrame) setState(s uint32) { f.flags = (f.flags &^ 12) | ((s & 3) << 2) }
func (f *rewriterFrame) setDepth(d uint32) { f.flags = (f.flags &^ 48) | ((d & 3) << 4) }
func (f *rewriterFrame) setIndex(i uint32) { f.flags = (f.flags & 63) | (i << 6) }

type rewriterScope struct {
	oldRoot     ASTHandle
	oldNumQVars uint32
}

// Cold owner pointer vectors use the original ptr_vector growth. Each new
// cache is retained in pendingCache until its pointer append succeeds.
type rewriterCacheStack struct {
	values []*astActivityCache
	bytes  uint64
}

func (s *rewriterCacheStack) push(m *ASTManager, c *astActivityCache) error {
	if len(s.values) == cap(s.values) {
		n := uint64(2)
		if cap(s.values) != 0 {
			n = (3*uint64(cap(s.values)) + 1) >> 1
		}
		width := uint64(reflect.TypeOf((*astActivityCache)(nil)).Size())
		if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := m.budget.Reserve(bytes); e != nil {
			return e
		}
		v := make([]*astActivityCache, len(s.values), int(n))
		copy(v, s.values)
		m.budget.Release(s.bytes)
		s.values = v
		s.bytes = bytes
	}
	s.values = append(s.values, c)
	return nil
}
func (s *rewriterCacheStack) close(m *ASTManager) error {
	for i, c := range s.values {
		if c != nil {
			if e := c.Close(); e != nil {
				return e
			}
			s.values[i] = nil
		}
	}
	m.budget.Release(s.bytes)
	s.values = nil
	s.bytes = 0
	return nil
}

type rewriterCore struct {
	self                           *rewriterCore
	manager                        *ASTManager
	proofGen, cancelCheck          bool
	cacheStack                     rewriterCacheStack
	cache                          *astActivityCache
	frames                         []rewriterFrame
	results                        ASTRefVector
	proofCacheStack                rewriterCacheStack
	proofCache                     *astActivityCache
	proofResults                   ASTRefVector
	root                           ASTHandle
	numQVars                       uint32
	scopes                         []rewriterScope
	pendingCache                   *astActivityCache
	header, frameBytes, scopeBytes uint64
	ready, closed                  bool
	cause                          error
}

func newRewriterCore(m *ASTManager, proofGen bool) (*rewriterCore, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(rewriterCore{}).Size())
	if e := m.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	c := &rewriterCore{header: bytes}
	e := c.init(m, proofGen)
	return c, e
}

// Embedded derived objects charge their full header once, then construct this
// base in place with header=0. Standalone construction retains its own charge.
func (c *rewriterCore) init(m *ASTManager, proofGen bool) error {
	c.manager, c.proofGen, c.cancelCheck = m, proofGen, true
	c.self = c
	var e error
	// Both owning result-vector members are constructed, even without proofs.
	c.results, e = m.NewRefVector()
	if e != nil {
		return e
	}
	c.proofResults, e = m.NewRefVector()
	if e != nil {
		return e
	}
	if e = c.initCacheStack(); e != nil {
		return e
	}
	c.ready = true
	return nil
}
func (c *rewriterCore) open() error {
	if c == nil || c.self != c || c.closed {
		return ErrASTHandle
	}
	if !c.ready {
		return ErrASTDomain
	}
	if c.cause != nil {
		return c.cause
	}
	_, e := c.manager.ProofsEnabled()
	return e
}
func (c *rewriterCore) pushCache(stack *rewriterCacheStack) (*astActivityCache, error) {
	v, e := newASTActivityCache(c.manager, nil)
	if e != nil {
		return nil, e
	}
	c.pendingCache = v
	if e = stack.push(c.manager, v); e != nil {
		return nil, e
	}
	c.pendingCache = nil
	return v, nil
}
func (c *rewriterCore) initCacheStack() error {
	if len(c.cacheStack.values) != 0 {
		return ErrASTDomain
	}
	v, e := c.pushCache(&c.cacheStack)
	if e != nil {
		return e
	}
	c.cache = v
	if c.proofGen {
		v, e = c.pushCache(&c.proofCacheStack)
		if e != nil {
			return e
		}
		c.proofCache = v
	}
	return nil
}
func (c *rewriterCore) delCacheStack() error {
	if c.pendingCache != nil {
		if e := c.pendingCache.Close(); e != nil {
			return e
		}
		c.pendingCache = nil
	}
	if e := c.cacheStack.close(c.manager); e != nil {
		return e
	}
	c.cache = nil
	if c.proofGen {
		if e := c.proofCacheStack.close(c.manager); e != nil {
			return e
		}
		c.proofCache = nil
	}
	return nil
}
func (c *rewriterCore) cacheResult(k, v ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	if astSourceDebug {
		if c.proofGen {
			return ErrASTDomain
		}
		a, e := c.manager.GetSort(k)
		if e != nil {
			return e
		}
		b, e := c.manager.GetSort(v)
		if e != nil {
			return e
		}
		if a != b {
			return ErrASTDomain
		}
	}
	return c.cache.Insert(k, v)
}
func (c *rewriterCore) cacheResultProof(k, v, pr ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	if e := c.cache.Insert(k, v); e != nil {
		return e
	}
	if !c.proofGen {
		return ErrASTDomain
	}
	return c.proofCache.Insert(k, pr)
}
func (c *rewriterCore) getCached(k ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	return c.cache.Find(k)
}
func (c *rewriterCore) getCachedProof(k ASTHandle) (ASTHandle, error) {
	if e := c.open(); e != nil {
		return ASTHandle{}, e
	}
	if !c.proofGen {
		return ASTHandle{}, ErrASTDomain
	}
	return c.proofCache.Find(k)
}
func (c *rewriterCore) getCacheSize() (uint32, error) {
	if e := c.open(); e != nil {
		return 0, e
	}
	return c.cache.Size()
}
func (c *rewriterCore) resetCache() error {
	if len(c.cacheStack.values) == 0 {
		return ErrASTDomain
	}
	c.cache = c.cacheStack.values[0]
	if e := c.cache.Reset(); e != nil {
		return e
	}
	if c.proofGen {
		if len(c.proofCacheStack.values) == 0 {
			return ErrASTDomain
		}
		c.proofCache = c.proofCacheStack.values[0]
		if e := c.proofCache.Reset(); e != nil {
			return e
		}
	}
	return nil
}
func (c *rewriterCore) pushScope() error {
	if len(c.scopes) == cap(c.scopes) {
		n := uint64(2)
		if cap(c.scopes) != 0 {
			n = (3*uint64(cap(c.scopes)) + 1) >> 1
		}
		width := uint64(reflect.TypeOf(rewriterScope{}).Size())
		if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := c.manager.budget.Reserve(bytes); e != nil {
			return e
		}
		v := make([]rewriterScope, len(c.scopes), int(n))
		copy(v, c.scopes)
		c.manager.budget.Release(c.scopeBytes)
		c.scopes = v
		c.scopeBytes = bytes
	}
	c.scopes = append(c.scopes, rewriterScope{c.root, c.numQVars})
	return nil
}
func (c *rewriterCore) beginScope() (err error) {
	if e := c.open(); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if e := c.pushScope(); e != nil {
		return e
	}
	level := len(c.scopes)
	if level > len(c.cacheStack.values) || c.proofGen && len(c.proofCacheStack.values) != len(c.cacheStack.values) {
		return ErrASTDomain
	}
	if level == len(c.cacheStack.values) {
		if _, e := c.pushCache(&c.cacheStack); e != nil {
			return e
		}
		if c.proofGen {
			if _, e := c.pushCache(&c.proofCacheStack); e != nil {
				return e
			}
		}
	}
	c.cache = c.cacheStack.values[level]
	if e := c.cache.Reset(); e != nil {
		return e
	}
	if c.proofGen {
		c.proofCache = c.proofCacheStack.values[level]
		if e := c.proofCache.Reset(); e != nil {
			return e
		}
	}
	return nil
}
func (c *rewriterCore) endScope() (err error) {
	if e := c.open(); e != nil {
		return e
	}
	if len(c.scopes) == 0 {
		return ErrASTDomain
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if e := c.cache.Reset(); e != nil {
		return e
	}
	if c.proofGen {
		if e := c.proofCache.Reset(); e != nil {
			return e
		}
	}
	s := c.scopes[len(c.scopes)-1]
	c.root = s.oldRoot
	c.numQVars = s.oldNumQVars
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.cache = c.cacheStack.values[len(c.scopes)]
	if c.proofGen {
		c.proofCache = c.proofCacheStack.values[len(c.scopes)]
	}
	return nil
}
func (c *rewriterCore) mustCache(t ASTHandle) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	refs, e := c.manager.GetRefCount(t)
	if e != nil || refs <= 1 || t == c.root {
		return false, e
	}
	kind, e := c.manager.GetKind(t)
	if e != nil {
		return false, e
	}
	if kind == ASTQuantifier {
		return true, nil
	}
	if kind != ASTApp {
		return false, nil
	}
	n, e := c.manager.GetNumArgs(t)
	return n > 0, e
}
func (c *rewriterCore) pushFrameCore(t ASTHandle, cache bool, state, depth uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	n, e := c.manager.RefVectorSize(c.results)
	if e != nil {
		return e
	}
	if astSourceDebug && c.proofGen {
		p, e := c.manager.RefVectorSize(c.proofResults)
		if e != nil {
			return e
		}
		if n != p {
			return ErrASTDomain
		}
	}
	if len(c.frames) == cap(c.frames) {
		size := uint64(2)
		if cap(c.frames) != 0 {
			size = (3*uint64(cap(c.frames)) + 1) >> 1
		}
		width := uint64(reflect.TypeOf(rewriterFrame{}).Size())
		if size > uint64(^uint32(0)) || size > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := size * width
		if e := c.manager.budget.Reserve(bytes); e != nil {
			return e
		}
		v := make([]rewriterFrame, len(c.frames), int(size))
		copy(v, c.frames)
		c.manager.budget.Release(c.frameBytes)
		c.frames = v
		c.frameBytes = bytes
	}
	fr := rewriterFrame{curr: t, spos: n}
	fr.setState(state)
	fr.setDepth(depth)
	if cache {
		fr.flags |= 1
	}
	c.frames = append(c.frames, fr)
	return nil
}
func (c *rewriterCore) pushFrame(t ASTHandle, state uint32) error {
	cache, e := c.mustCache(t)
	if e != nil {
		return e
	}
	return c.pushFrameCore(t, cache, state, 3)
}
func (c *rewriterCore) isChildOfTopFrame(t ASTHandle) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if len(c.frames) == 0 {
		return true, nil
	}
	parent := c.frames[len(c.frames)-1].curr
	m := c.manager
	kind, e := m.GetKind(parent)
	if e != nil {
		return false, e
	}
	switch kind {
	case ASTApp:
		n, e := m.GetNumArgs(parent)
		if e != nil {
			return false, e
		}
		for i := uint32(0); i < n; i++ {
			child, e := m.GetArg(parent, i)
			if e != nil {
				return false, e
			}
			if child == t {
				return true, nil
			}
		}
	case ASTQuantifier:
		child, e := m.GetQuantifierBody(parent)
		if e != nil {
			return false, e
		}
		if child == t {
			return true, nil
		}
		n, e := m.GetNumPatterns(parent)
		if e != nil {
			return false, e
		}
		for i := uint32(0); i < n; i++ {
			child, e := m.GetPattern(parent, i)
			if e != nil {
				return false, e
			}
			if child == t {
				return true, nil
			}
		}
		n, e = m.GetNumNoPatterns(parent)
		if e != nil {
			return false, e
		}
		for i := uint32(0); i < n; i++ {
			child, e := m.GetNoPattern(parent, i)
			if e != nil {
				return false, e
			}
			if child == t {
				return true, nil
			}
		}
	}
	return false, nil
}
func (c *rewriterCore) setNewChild(old ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	if astSourceDebug {
		yes, e := c.isChildOfTopFrame(old)
		if e != nil {
			return e
		}
		if !yes {
			return ErrASTDomain
		}
	}
	if len(c.frames) != 0 {
		c.frames[len(c.frames)-1].flags |= 2
	}
	return nil
}
func (c *rewriterCore) setNewChildIfDifferent(old, new ASTHandle) error {
	if old == new {
		return c.open()
	}
	return c.setNewChild(old)
}
func (c *rewriterCore) elimReflexProofs(spos uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if astSourceDebug && !c.proofGen {
		return ErrASTDomain
	}
	size, e := c.manager.RefVectorSize(c.proofResults)
	if e != nil {
		return e
	}
	if spos > size {
		return ErrASTDomain
	}
	j := spos
	for i := spos; i < size; i++ {
		pr, e := c.manager.RefVectorGet(c.proofResults, i)
		if e != nil {
			return e
		}
		if !pr.IsNull() {
			if i != j {
				if e = c.manager.RefVectorSet(c.proofResults, j, pr); e != nil {
					return e
				}
			}
			j++
		}
	}
	return c.manager.RefVectorShrink(c.proofResults, j)
}
func (c *rewriterCore) SetCancelCheck(f bool) error {
	if e := c.open(); e != nil {
		return e
	}
	c.cancelCheck = f
	return nil
}
func (c *rewriterCore) Reset() (err error) {
	if e := c.open(); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if e := c.resetCache(); e != nil {
		return e
	}
	c.frames = c.frames[:0]
	if e := c.manager.RefVectorReset(c.results); e != nil {
		return e
	}
	if c.proofGen {
		if e := c.manager.RefVectorReset(c.proofResults); e != nil {
			return e
		}
	}
	c.root = ASTHandle{}
	c.numQVars = 0
	c.scopes = c.scopes[:0]
	return nil
}
func (c *rewriterCore) freeMemory() error {
	if e := c.delCacheStack(); e != nil {
		return e
	}
	c.manager.budget.Release(c.frameBytes)
	c.frames = nil
	c.frameBytes = 0
	if e := c.manager.RefVectorFinalize(c.results); e != nil {
		return e
	}
	c.manager.budget.Release(c.scopeBytes)
	c.scopes = nil
	c.scopeBytes = 0
	// Source deliberately does not finalize the proof result-stack here.
	return nil
}
func (c *rewriterCore) Cleanup() (err error) {
	if e := c.open(); e != nil {
		return e
	}
	defer func() {
		if err != nil {
			c.cause = err
		}
	}()
	if e := c.freeMemory(); e != nil {
		return e
	}
	if e := c.initCacheStack(); e != nil {
		return e
	}
	c.root = ASTHandle{}
	c.numQVars = 0
	return nil
}
func (c *rewriterCore) Close() error {
	if c == nil || c.self != c {
		return ErrASTHandle
	}
	if c.closed {
		return nil
	}
	if e := c.delCacheStack(); e != nil {
		return e
	}
	// Destructor body precedes member destruction in reverse declaration order.
	c.manager.budget.Release(c.scopeBytes)
	c.scopes = nil
	c.scopeBytes = 0
	if c.proofResults != (ASTRefVector{}) {
		if e := c.manager.DelRefVector(&c.proofResults); e != nil {
			return e
		}
	}
	if c.results != (ASTRefVector{}) {
		if e := c.manager.DelRefVector(&c.results); e != nil {
			return e
		}
	}
	c.manager.budget.Release(c.frameBytes)
	c.frames = nil
	c.frameBytes = 0
	c.manager.budget.Release(c.header)
	c.header = 0
	c.closed = true
	return nil
}
