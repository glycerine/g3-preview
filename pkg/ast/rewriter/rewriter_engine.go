// Source: src/ast/rewriter/rewriter_def.h:23-750;
// src/ast/rewriter/rewriter.h:194-363; src/util/vector.h:59-99.
package rewriter

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

type rewriterException struct{ message string }

func (e rewriterException) Error() string { return e.message }

// The no-proof source specialization retains the original iterative states,
// scopes, bounded revisit depths and borrowed binding vectors. Proof-generating
// traversal remains a separate port obligation.
type rewriterEngine struct {
	self                             *rewriterEngine
	core                             rewriterCore
	cfg                              rewriterConfiguration
	numSteps                         uint32
	bindings                         []ast.ASTHandle
	shifter, invShifter              variableShifter
	r, pr, pr2                       ast.ASTRef
	shifts                           []uint32
	header, bindingBytes, shiftBytes uint64
	ready, closed                    bool
}

func newRewriterEngine(m *ast.ASTManager, cfg rewriterConfiguration) (*rewriterEngine, error) {
	if m == nil || cfg == nil {
		return nil, ast.ErrASTHandle
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(rewriterEngine{}).Size())
	if e := m.BudgetField.Reserve(n); e != nil {
		return nil, e
	}
	r := &rewriterEngine{header: n}
	return r, r.init(m, cfg)
}
func (r *rewriterEngine) init(m *ast.ASTManager, cfg rewriterConfiguration) error {
	r.self = r
	r.cfg = cfg
	if e := r.core.init(m, false); e != nil {
		return e
	}
	if e := r.shifter.init(m, false); e != nil {
		return e
	}
	if e := r.invShifter.init(m, true); e != nil {
		return e
	}
	var e error
	r.r, e = m.NewRef(ast.ASTHandle{})
	if e != nil {
		return e
	}
	r.pr, e = m.NewRef(ast.ASTHandle{})
	if e != nil {
		return e
	}
	r.pr2, e = m.NewRef(ast.ASTHandle{})
	if e != nil {
		return e
	}
	r.ready = true
	return nil
}
func (r *rewriterEngine) open() error {
	if r == nil || r.self != r || r.closed || !r.ready {
		return ast.ErrASTHandle
	}
	return r.core.open()
}
func rewriterVectorCapacity(capacity int, width uint64) (uint64, uint64, error) {
	n := uint64(2)
	if capacity != 0 {
		n = (3*uint64(capacity) + 1) >> 1
	}
	if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/width {
		return 0, 0, arena.ErrBounds
	}
	return n, n * width, nil
}
func (r *rewriterEngine) pushBinding(h ast.ASTHandle, shift uint32) error {
	m := r.core.manager
	if len(r.bindings) == cap(r.bindings) {
		n, bytes, e := rewriterVectorCapacity(cap(r.bindings), uint64(reflect.TypeOf(ast.ASTHandle{}).Size()))
		if e != nil {
			return e
		}
		if e = m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		v := make([]ast.ASTHandle, len(r.bindings), int(n))
		copy(v, r.bindings)
		m.BudgetField.Release(r.bindingBytes)
		r.bindings = v
		r.bindingBytes = bytes
	}
	r.bindings = append(r.bindings, h)
	if len(r.shifts) == cap(r.shifts) {
		n, bytes, e := rewriterVectorCapacity(cap(r.shifts), 4)
		if e != nil {
			return e
		}
		if e = m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		v := make([]uint32, len(r.shifts), int(n))
		copy(v, r.shifts)
		m.BudgetField.Release(r.shiftBytes)
		r.shifts = v
		r.shiftBytes = bytes
	}
	r.shifts = append(r.shifts, shift)
	return nil
}
func (r *rewriterEngine) SetBindings(values []ast.ASTHandle) error {
	return r.setBindings(values, false)
}
func (r *rewriterEngine) SetInvBindings(values []ast.ASTHandle) error {
	return r.setBindings(values, true)
}
func (r *rewriterEngine) setBindings(values []ast.ASTHandle, inverse bool) error {
	if e := r.open(); e != nil {
		return e
	}
	if len(r.core.frames) != 0 || uint64(len(values)) > uint64(^uint32(0)) {
		return ast.ErrASTDomain
	}
	r.bindings = r.bindings[:0]
	r.shifts = r.shifts[:0]
	n := uint32(len(values))
	for i := uint32(0); i < n; i++ {
		j := n - i - 1
		if inverse {
			j = i
		}
		if e := r.pushBinding(values[j], n); e != nil {
			return e
		}
	}
	return nil
}
func (r *rewriterEngine) Reset() error {
	if e := r.open(); e != nil {
		return e
	}
	r.cfg.reset()
	if e := r.core.Reset(); e != nil {
		return e
	}
	r.bindings = r.bindings[:0]
	r.shifts = r.shifts[:0]
	if e := r.shifter.core.Reset(); e != nil {
		return e
	}
	return r.invShifter.core.Reset()
}
func (r *rewriterEngine) Cleanup() error {
	if e := r.open(); e != nil {
		return e
	}
	r.cfg.cleanup()
	if e := r.core.Cleanup(); e != nil {
		return e
	}
	m := r.core.manager
	m.BudgetField.Release(r.bindingBytes)
	r.bindingBytes = 0
	r.bindings = nil
	if e := r.shifter.core.Cleanup(); e != nil {
		return e
	}
	m.BudgetField.Release(r.shiftBytes)
	r.shiftBytes = 0
	r.shifts = nil
	return r.invShifter.core.Cleanup()
}
func (r *rewriterEngine) Close() error {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	m := r.core.manager
	m.BudgetField.Release(r.shiftBytes)
	r.shifts = nil
	r.shiftBytes = 0
	for _, ref := range []*ast.ASTRef{&r.pr2, &r.pr, &r.r} {
		if *ref != (ast.ASTRef{}) {
			if e := m.DelRef(ref); e != nil {
				return e
			}
		}
	}
	if r.invShifter.self != nil {
		if e := r.invShifter.Close(); e != nil {
			return e
		}
	}
	if r.shifter.self != nil {
		if e := r.shifter.Close(); e != nil {
			return e
		}
	}
	m.BudgetField.Release(r.bindingBytes)
	r.bindings = nil
	r.bindingBytes = 0
	if e := r.core.Close(); e != nil {
		return e
	}
	m.BudgetField.Release(r.header)
	r.header = 0
	r.closed = true
	return nil
}
func (r *rewriterEngine) pushFrame(t ast.ASTHandle, cache bool, depth uint32) error {
	if r.cfg.maxFramesExceeded(uint32(len(r.core.frames))) {
		return rewriterException{util.Z3MaxFramesMessage}
	}
	return r.core.pushFrameCore(t, cache, 0, depth)
}
func (r *rewriterEngine) beginScope() error {
	if r.cfg.maxScopesExceeded(uint32(len(r.core.scopes))) {
		return rewriterException{util.Z3MaxScopesMessage}
	}
	return r.core.beginScope()
}
func (r *rewriterEngine) mustCache(t ast.ASTHandle) (bool, error) {
	c := &r.core
	if r.cfg.cacheAllResults() {
		if t == c.root {
			return false, nil
		}
		kind, e := c.manager.GetKind(t)
		if e != nil {
			return false, e
		}
		if kind == ast.ASTQuantifier {
			return true, nil
		}
		if kind != ast.ASTApp {
			return false, nil
		}
		n, e := c.manager.GetNumArgs(t)
		return n > 0, e
	}
	if r.cfg.cacheResults() {
		return c.mustCache(t)
	}
	return false, nil
}
func (r *rewriterEngine) cache(fr rewriterFrame, value ast.ASTHandle) error {
	if fr.cacheResult() {
		return r.core.cacheResult(fr.curr, value)
	}
	return nil
}
func (r *rewriterEngine) sameSort(old, value ast.ASTHandle) error {
	if !ast.ASTSourceDebug {
		return nil
	}
	m := r.core.manager
	a, e := m.GetSort(old)
	if e != nil {
		return e
	}
	b, e := m.GetSort(value)
	if e != nil {
		return e
	}
	if a != b {
		return ast.ErrASTDomain
	}
	return nil
}
func (r *rewriterEngine) visit(t ast.ASTHandle, depth uint32) (bool, error) {
	c := &r.core
	m := c.manager
	sub, _, found, e := r.cfg.getSubst(t)
	if e != nil {
		return false, e
	}
	if found {
		if e = r.sameSort(t, sub); e != nil {
			return false, e
		}
		if e = m.RefVectorPush(c.results, sub); e != nil {
			return false, e
		}
		return true, r.setNewChildIfDifferent(t, sub)
	}
	if depth == 0 {
		return true, m.RefVectorPush(c.results, t)
	}
	cache, e := r.mustCache(t)
	if e != nil {
		return false, e
	}
	if cache {
		h, e := c.getCached(t)
		if e != nil {
			return false, e
		}
		if !h.IsNull() {
			if e = r.sameSort(t, h); e != nil {
				return false, e
			}
			if e = m.RefVectorPush(c.results, h); e != nil {
				return false, e
			}
			return true, r.setNewChildIfDifferent(t, h)
		}
	}
	allowed, e := r.cfg.preVisit(t)
	if e != nil {
		return false, e
	}
	if !allowed {
		return true, m.RefVectorPush(c.results, t)
	}
	kind, e := m.GetKind(t)
	if e != nil {
		return false, e
	}
	switch kind {
	case ast.ASTVar:
		return true, r.processVar(t)
	case ast.ASTApp:
		n, e := m.GetNumArgs(t)
		if e != nil {
			return false, e
		}
		if n == 0 {
			return true, r.processConst(t)
		}
	case ast.ASTQuantifier:
	default:
		return false, ast.ErrASTDomain
	}
	if depth != 3 {
		depth--
	}
	return false, r.pushFrame(t, cache, depth)
}
func (r *rewriterEngine) processVar(t ast.ASTHandle) (err error) {
	c := &r.core
	m := c.manager
	done, e := r.cfg.reduceVar(t, r.r, r.pr)
	if e != nil {
		return e
	}
	if done {
		h, e := m.RefGet(r.r)
		if e != nil {
			return e
		}
		if e = m.RefVectorPush(c.results, h); e != nil {
			return e
		}
		if e = r.sameSort(t, h); e != nil {
			return e
		}
		if e = r.setNewChild(t); e != nil {
			return e
		}
		return m.ResetRef(r.r)
	}
	index, e := m.GetVarIndex(t)
	if e != nil {
		return e
	}
	if index < uint32(len(r.bindings)) {
		i := uint32(len(r.bindings)) - index - 1
		h := r.bindings[i]
		if !h.IsNull() {
			if e = r.sameSort(t, h); e != nil {
				return e
			}
			ground, e := m.IsGround(h)
			if e != nil {
				return e
			}
			if !ground && r.shifts[i] != uint32(len(r.bindings)) {
				e = func() (err error) {
					tmp, e := m.NewRef(ast.ASTHandle{})
					if e != nil {
						return e
					}
					defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
					if e = r.shifter.ShiftBy(h, uint32(len(r.bindings))-r.shifts[i], tmp); e != nil {
						return e
					}
					value, e := m.RefGet(tmp)
					if e != nil {
						return e
					}
					return m.RefVectorPush(c.results, value)
				}()
				if e != nil {
					return e
				}
				return r.setNewChild(t)
			}
			if e = m.RefVectorPush(c.results, h); e != nil {
				return e
			}
			return r.setNewChild(t)
		}
	}
	return m.RefVectorPush(c.results, t)
}
func (r *rewriterEngine) processConst(t ast.ASTHandle) error {
	c := &r.core
	m := c.manager
	d, e := m.GetDecl(t)
	if e != nil {
		return e
	}
	st, e := r.cfg.reduceApp(d, ast.AppArguments{}, r.r, r.pr)
	if e != nil {
		return e
	}
	if st == brDone {
		h, e := m.RefGet(r.r)
		if e != nil {
			return e
		}
		if e = r.sameSort(t, h); e != nil {
			return e
		}
		if e = m.RefVectorPush(c.results, h); e != nil {
			return e
		}
		if e = m.ResetRef(r.r); e != nil {
			return e
		}
		return r.setNewChild(t)
	}
	if ast.ASTSourceDebug && st != brFailed {
		return ast.ErrASTDomain
	}
	return m.RefVectorPush(c.results, t)
}

// Complete one frame while preserving the source's explicit temporary-ref
// lifetime: delayed builtin and macro states retain m_r until its next use.
func (r *rewriterEngine) finish(fr rewriterFrame, unconditional, clear bool) error {
	c := &r.core
	m := c.manager
	h, e := m.RefGet(r.r)
	if e != nil {
		return e
	}
	if e = m.RefVectorShrink(c.results, fr.spos); e != nil {
		return e
	}
	if e = m.RefVectorPush(c.results, h); e != nil {
		return e
	}
	if e = r.cache(fr, h); e != nil {
		return e
	}
	c.frames = c.frames[:len(c.frames)-1]
	if unconditional {
		e = r.setNewChild(fr.curr)
	} else {
		e = r.setNewChildIfDifferent(fr.curr, h)
	}
	if e != nil {
		return e
	}
	if clear {
		return m.ResetRef(r.r)
	}
	return nil
}
func (r *rewriterEngine) rewritten(fr rewriterFrame, clear bool) error {
	c, m := &r.core, r.core.manager
	h, e := m.RefVectorBack(c.results)
	if e != nil {
		return e
	}
	if e = m.SetRef(r.r, h); e != nil {
		return e
	}
	if e = m.RefVectorPop(c.results); e != nil {
		return e
	}
	if e = m.RefVectorPop(c.results); e != nil {
		return e
	}
	if e = m.RefVectorPush(c.results, h); e != nil {
		return e
	}
	if e = r.cache(fr, h); e != nil {
		return e
	}
	c.frames = c.frames[:len(c.frames)-1]
	if e = r.setNewChild(fr.curr); e != nil {
		return e
	}
	if clear {
		return m.ResetRef(r.r)
	}
	return nil
}
func (r *rewriterEngine) constantFold(frameIndex int) (bool, error) {
	c := &r.core
	m := c.manager
	fr := c.frames[frameIndex]
	if fr.index() != 1 {
		return false, nil
	}
	helper := boolRewriter{manager: m}
	ite, e := helper.isOperator(fr.curr, ast.ASTOpIte)
	if e != nil || !ite {
		return false, e
	}
	cond, e := m.RefVectorGet(c.results, fr.spos)
	if e != nil {
		return false, e
	}
	truth, e := helper.isOperator(cond, ast.ASTOpTrue)
	if e != nil {
		return false, e
	}
	falsity, e := helper.isOperator(cond, ast.ASTOpFalse)
	if e != nil {
		return false, e
	}
	if !truth && !falsity {
		return false, nil
	}
	i := uint32(2)
	if truth {
		i = 1
	}
	arg, e := m.GetArg(fr.curr, i)
	if e != nil {
		return false, e
	}
	if e = m.RefVectorShrink(c.results, fr.spos); e != nil {
		return false, e
	}
	if e = m.RefVectorPush(c.results, arg); e != nil {
		return false, e
	}
	c.frames[frameIndex].setState(1)
	done, e := r.visit(arg, fr.maxDepth())
	if e != nil {
		return false, e
	}
	if done {
		if e = r.rewritten(fr, false); e != nil {
			return false, e
		}
	}
	return true, m.ResetRef(r.r)
}

func (r *rewriterEngine) processApp() (err error) {
	c := &r.core
	m := c.manager
	fi := len(c.frames) - 1
	fr := c.frames[fi]
	t := fr.curr
	n, e := m.GetNumArgs(t)
	if e != nil {
		return e
	}
	switch fr.state() {
	case 0:
		for c.frames[fi].index() < n {
			folded, e := r.constantFold(fi)
			if e != nil {
				return e
			}
			if folded {
				return nil
			}
			i := c.frames[fi].index()
			arg, e := m.GetArg(t, i)
			if e != nil {
				return e
			}
			c.frames[fi].setIndex(i + 1)
			done, e := r.visit(arg, fr.maxDepth())
			if e != nil {
				return e
			}
			if !done {
				return nil
			}
		}
		fr = c.frames[fi]
		d, e := m.GetDecl(t)
		if e != nil {
			return e
		}
		assoc, e := m.RewriterAssociative(d)
		if e != nil {
			return e
		}
		rc, e := m.GetRefCount(t)
		if e != nil {
			return e
		}
		if assoc && rc <= 1 && fi > 0 {
			prev := c.frames[fi-1]
			kind, e := m.GetKind(prev.curr)
			if e != nil {
				return e
			}
			if kind == ast.ASTApp && prev.state() == 0 {
				pd, e := m.GetDecl(prev.curr)
				if e != nil {
					return e
				}
				if pd == d {
					flat, e := r.cfg.flatAssoc(d)
					if e != nil {
						return e
					}
					if flat {
						c.frames = c.frames[:fi]
						return r.setNewChild(t)
					}
				}
			}
		}
		size, e := m.RefVectorSize(c.results)
		if e != nil {
			return e
		}
		args, e := c.resultArguments(fr.spos, size-fr.spos)
		if e != nil {
			return e
		}
		st, e := r.cfg.reduceApp(d, args, r.r, r.pr2)
		if e != nil {
			return e
		}
		if st != brFailed {
			h, e := m.RefGet(r.r)
			if e != nil {
				return e
			}
			if e = r.sameSort(t, h); e != nil {
				return e
			}
			if e = m.RefVectorShrink(c.results, fr.spos); e != nil {
				return e
			}
			if e = m.RefVectorPush(c.results, h); e != nil {
				return e
			}
			if st == brDone {
				return r.finish(fr, true, true)
			}
			if st > brRewriteFull {
				return ast.ErrASTDomain
			}
			c.frames[fi].setState(1)
			depth := uint32(st)
			if depth != 3 {
				depth++
			}
			done, e := r.visit(h, depth)
			if e != nil {
				return e
			}
			if done {
				return r.rewritten(fr, true)
			}
			return m.ResetRef(r.r)
		}
		def, _, _, macro, e := r.cfg.getMacro(d)
		if e != nil {
			return e
		}
		if macro {
			if e = r.sameSort(t, def); e != nil {
				return e
			}
			ground, e := m.IsGround(def)
			if e != nil {
				return e
			}
			if ground {
				if e = m.SetRef(r.r, def); e != nil {
					return e
				}
			} else {
				if args.Count() != n {
					return ast.ErrASTDomain
				}
				c.frames[fi].setState(2)
				sz := uint32(len(r.bindings))
				for i := n; i > 0; {
					i--
					h, e := args.At(i)
					if e != nil {
						return e
					}
					if e = r.pushBinding(h, sz); e != nil {
						return e
					}
				}
				if e = m.RefVectorPush(c.results, def); e != nil {
					return e
				}
				if e = r.beginScope(); e != nil {
					return e
				}
				c.numQVars += n
				c.root = def
				return r.pushFrame(def, false, 3)
			}
		} else {
			h := t
			if fr.newChild() {
				h, e = m.RewriterMakeApp(d, args)
				if e != nil {
					return e
				}
			}
			if e = m.SetRef(r.r, h); e != nil {
				return e
			}
		}
		return r.finish(fr, false, true)
	case 1:
		return r.rewritten(fr, false)
	case 2:
		tmp, e := m.NewRef(ast.ASTHandle{})
		if e != nil {
			return e
		}
		defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
		if n > uint32(len(r.bindings)) || n > uint32(len(r.shifts)) {
			return ast.ErrASTDomain
		}
		r.bindings = r.bindings[:uint32(len(r.bindings))-n]
		r.shifts = r.shifts[:uint32(len(r.shifts))-n]
		c.numQVars -= n
		if e = c.endScope(); e != nil {
			return e
		}
		h, e := m.RefVectorBack(c.results)
		if e != nil {
			return e
		}
		if e = m.SetRef(r.r, h); e != nil {
			return e
		}
		ground, e := m.IsGround(h)
		if e != nil {
			return e
		}
		if !ground {
			if e = r.invShifter.InverseShift(h, n, tmp); e != nil {
				return e
			}
			h, e = m.RefGet(tmp)
			if e != nil {
				return e
			}
			if e = m.SetRef(r.r, h); e != nil {
				return e
			}
		}
		return r.finish(fr, true, false)
	default:
		return ast.ErrASTDomain // Source REWRITE_RULE is NOT_IMPLEMENTED_YET.
	}
}

func (r *rewriterEngine) processQuantifier() (err error) {
	c := &r.core
	m := c.manager
	fi := len(c.frames) - 1
	fr := c.frames[fi]
	q := fr.curr
	nd, e := m.GetNumDeclarations(q)
	if e != nil {
		return e
	}
	np, e := m.GetNumPatterns(q)
	if e != nil {
		return e
	}
	nn, e := m.GetNumNoPatterns(q)
	if e != nil {
		return e
	}
	body, e := m.GetQuantifierBody(q)
	if e != nil {
		return e
	}
	if fr.index() == 0 {
		if e = r.beginScope(); e != nil {
			return e
		}
		c.root = body
		sz := uint32(len(r.bindings))
		for i := uint32(0); i < nd; i++ {
			if e = r.pushBinding(ast.ASTHandle{}, sz); e != nil {
				return e
			}
		}
		c.numQVars += nd
	}
	children := uint32(1)
	if r.cfg.rewritePatterns() {
		children += np + nn
	}
	for c.frames[fi].index() < children {
		i := c.frames[fi].index()
		h := body
		if i > 0 {
			if i <= np {
				h, e = m.GetPattern(q, i-1)
			} else {
				h, e = m.GetNoPattern(q, i-1-np)
			}
			if e != nil {
				return e
			}
		}
		c.frames[fi].setIndex(i + 1)
		done, e := r.visit(h, fr.maxDepth())
		if e != nil {
			return e
		}
		if !done {
			return nil
		}
	}
	fr = c.frames[fi]
	pats, e := m.NewRefVector()
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelRefVector(&pats)) }()
	nopats, e := m.NewRefVector()
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelRefVector(&nopats)) }()
	for i := uint32(0); i < np; i++ {
		h, e := m.GetPattern(q, i)
		if e != nil {
			return e
		}
		if e = m.RefVectorPush(pats, h); e != nil {
			return e
		}
	}
	for i := uint32(0); i < nn; i++ {
		h, e := m.GetNoPattern(q, i)
		if e != nil {
			return e
		}
		if e = m.RefVectorPush(nopats, h); e != nil {
			return e
		}
	}
	if r.cfg.rewritePatterns() {
		for i := uint32(0); i < np+nn; i++ {
			h, e := m.RefVectorGet(c.results, fr.spos+1+i)
			if e != nil {
				return e
			}
			pattern, e := m.RewriterPattern(h)
			if e != nil {
				return e
			}
			if pattern {
				v, j := pats, i
				if i >= np {
					v, j = nopats, i-np
				}
				if e = m.RefVectorSet(v, j, h); e != nil {
					return e
				}
			}
		}
	}
	newBody, e := m.RefVectorGet(c.results, fr.spos)
	if e != nil {
		return e
	}
	pa, e := m.RewriterVectorArguments(pats)
	if e != nil {
		return e
	}
	na, e := m.RewriterVectorArguments(nopats)
	if e != nil {
		return e
	}
	done, e := r.cfg.reduceQuantifier(q, newBody, pa, na, r.r, r.pr)
	if e != nil {
		return e
	}
	if !done {
		h := q
		if fr.newChild() {
			h, e = m.UpdateQuantifierArguments(q, newBody, pa, na)
			if e != nil {
				return e
			}
		}
		if e = m.SetRef(r.r, h); e != nil {
			return e
		}
	}
	h, e := m.RefGet(r.r)
	if e != nil {
		return e
	}
	if e = m.RefVectorShrink(c.results, fr.spos); e != nil {
		return e
	}
	if e = m.RefVectorPush(c.results, h); e != nil {
		return e
	}
	if nd > uint32(len(r.bindings)) || nd > uint32(len(r.shifts)) {
		return ast.ErrASTDomain
	}
	r.bindings = r.bindings[:uint32(len(r.bindings))-nd]
	r.shifts = r.shifts[:uint32(len(r.shifts))-nd]
	if e = c.endScope(); e != nil {
		return e
	}
	if e = r.cache(fr, h); e != nil {
		return e
	}
	if e = m.ResetRef(r.r); e != nil {
		return e
	}
	c.frames = c.frames[:fi]
	// Source compares q with m_r after clearing m_r, so even reuse marks parent.
	return r.setNewChild(q)
}
func (r *rewriterEngine) checkCancel() error {
	if !r.core.cancelCheck {
		return nil
	}
	canceled, e := r.core.manager.Canceled()
	if e != nil {
		return e
	}
	if !canceled {
		return nil
	}
	message := r.core.manager.Limit.CancelMessage()
	if e = r.Reset(); e != nil {
		return e
	}
	return rewriterException{message}
}
func (r *rewriterEngine) Rewrite(t ast.ASTHandle, out ast.ASTRef) error {
	if e := r.open(); e != nil {
		return e
	}
	if e := r.checkCancel(); e != nil {
		return e
	}
	c := &r.core
	if len(c.frames) != 0 || c.cache != c.cacheStack.values[0] {
		return ast.ErrASTDomain
	}
	c.root = t
	c.numQVars = 0
	r.numSteps = 0
	done, e := r.visit(t, 3)
	if e != nil {
		return e
	}
	if done {
		return r.publish(out)
	}
	return r.Resume(out)
}
func (r *rewriterEngine) Resume(out ast.ASTRef) error {
	if e := r.open(); e != nil {
		return e
	}
	c := &r.core
	m := c.manager
	if len(c.frames) == 0 {
		return ast.ErrASTDomain
	}
	for len(c.frames) != 0 {
		if e := r.checkCancel(); e != nil {
			return e
		}
		r.numSteps++
		exceeded, err := r.cfg.maxStepsExceeded(r.numSteps)
		if err != nil {
			return err
		}
		if exceeded {
			return rewriterException{util.Z3MaxStepsMessage}
		}
		fr := c.frames[len(c.frames)-1]
		t := fr.curr
		if fr.state() == 0 && fr.index() == 0 && fr.cacheResult() {
			h, e := c.getCached(t)
			if e != nil {
				return e
			}
			if !h.IsNull() {
				if e = m.RefVectorPush(c.results, h); e != nil {
					return e
				}
				c.frames = c.frames[:len(c.frames)-1]
				if e = r.setNewChildIfDifferent(t, h); e != nil {
					return e
				}
				continue
			}
		}
		kind, e := m.GetKind(t)
		if e != nil {
			return e
		}
		switch kind {
		case ast.ASTApp:
			e = r.processApp()
		case ast.ASTQuantifier:
			e = r.processQuantifier()
		case ast.ASTVar:
			c.frames = c.frames[:len(c.frames)-1]
			e = r.processVar(t)
		default:
			e = ast.ErrASTDomain
		}
		if e != nil {
			return e
		}
	}
	return r.publish(out)
}
func (r *rewriterEngine) publish(out ast.ASTRef) error {
	m := r.core.manager
	h, e := m.RefVectorBack(r.core.results)
	if e != nil {
		return e
	}
	if e = m.SetRef(out, h); e != nil {
		return e
	}
	return m.RefVectorPop(r.core.results)
}

// rewriter_tpl overrides the base child assertion: builtin/macro frames may
// receive replacements that are not children of their original application.
func (r *rewriterEngine) setNewChild(old ast.ASTHandle) error {
	c := &r.core
	if e := c.open(); e != nil {
		return e
	}
	if len(c.frames) == 0 || c.frames[len(c.frames)-1].state() == 0 {
		return c.setNewChild(old)
	}
	c.frames[len(c.frames)-1].flags |= 2
	return nil
}
func (r *rewriterEngine) setNewChildIfDifferent(old, new ast.ASTHandle) error {
	if old == new {
		return r.core.open()
	}
	return r.setNewChild(old)
}
