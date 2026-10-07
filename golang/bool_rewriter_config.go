// Source: src/ast/rewriter/bool_rewriter.cpp:23-87;
// src/ast/rewriter/bool_rewriter.h:85-107,113-126,132-156;
// src/ast/rewriter/bool_rewriter_params.hpp:6-30;
// src/util/ref_buffer.h:35-68.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"
)

// The standalone cold constructor charges its complete header once. A failed
// reached parameter constructor returns its owner for explicit Close, following
// the existing generated-helper lifetime engine. AST/parameter services borrow.
func newBoolRewriter(m *ASTManager, c *ParamsContext, g *GlobalParamsService, p *ParamsRef) (*boolRewriter, error) {
	if m == nil || c == nil || g == nil || c.symbols != m.symbols || g.context != c {
		return nil, ErrParamsOwner
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(boolRewriter{}).Size())
	if e := m.budget.Reserve(n); e != nil {
		return nil, e
	}
	rw := &boolRewriter{manager: m, header: n}
	rw.self = rw
	rw.params = smtModuleParamsOwner{budget: g.config.Budget}
	rw.params.self = &rw.params
	if e := rw.params.bind(c, g); e != nil {
		return rw, e
	}
	e := rw.updateParams(p)
	if e == nil {
		rw.ready = true
	}
	return rw, e
}
func (rw *boolRewriter) configuredOpen() error {
	if rw == nil || rw.self != rw || rw.closed {
		return ErrParamsOwner
	}
	if !rw.ready {
		return ErrParamsPrecondition
	}
	_, e := rw.manager.ProofsEnabled()
	return e
}
func (rw *boolRewriter) UpdtParams(p *ParamsRef) error {
	if e := rw.configuredOpen(); e != nil {
		return e
	}
	return rw.updateParams(p)
}
func (rw *boolRewriter) updateParams(local *ParamsRef) (err error) {
	o := &rw.params
	if o.self != o || o.running || o.destructorCause != nil {
		return ErrParamsPrecondition
	}
	if local == nil {
		empty, e := o.context.NewValue()
		if e != nil {
			return e
		}
		local = &empty
		defer func() { err = errors.Join(err, o.context.Del(&empty)) }()
	}
	if e := o.context.check(local); e != nil {
		return e
	}
	o.running = true
	defer func() { o.running = false }()
	f, e := o.beginHelper(local, "rewriter")
	if f != nil {
		defer func() { err = o.finishHelper(f, err) }()
	}
	if e != nil {
		return e
	}
	b, e := o.frameBool(f, "flat", true)
	if e != nil {
		return e
	}
	rw.flat = b
	b, e = o.frameBool(f, "elim_and", false)
	if e != nil {
		return e
	}
	rw.elimAnd = b
	b, e = o.frameBool(f, "local_ctx", false)
	if e != nil {
		return e
	}
	rw.localContext = b
	u, e := o.frameUInt(f, "local_ctx_limit", ^uint32(0))
	if e != nil {
		return e
	}
	rw.localContextLimit = u
	b, e = o.frameBool(f, "blast_distinct", false)
	if e != nil {
		return e
	}
	rw.blastDistinct = b
	u, e = o.frameUInt(f, "blast_distinct_threshold", ^uint32(0))
	if e != nil {
		return e
	}
	rw.blastDistinctThreshold = u
	b, e = o.frameBool(f, "ite_extra_rules", false)
	if e != nil {
		return e
	}
	rw.iteExtraRules = b
	return nil
}
func (rw *boolRewriter) ResetLocalContextCost() { rw.localContextCost = 0 }
func (rw *boolRewriter) Close() error {
	if rw == nil || rw.self != rw {
		return ErrParamsOwner
	}
	if rw.closed {
		return nil
	}
	if e := rw.params.closeHelpers(); e != nil {
		return e
	}
	rw.manager.budget.Release(rw.header)
	rw.header = 0
	rw.closed = true
	return nil
}

func (rw *boolRewriter) mkAndCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	if rw.elimAnd {
		return brDone, rw.mkAndAsOr(args, out)
	}
	if rw.flat {
		return rw.mkFlatAndCore(args, out)
	}
	return rw.mkNFlatAndCore(args, out)
}
func (rw *boolRewriter) mkOrCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	if rw.flat {
		return rw.mkFlatOrCore(args, out)
	}
	return rw.mkNFlatOrCore(args, out)
}

// Concrete expr_ref_buffer: the underlying inline16/doubling pointer storage
// carries owning AST references here, including repeated entries.
type boolRewriterRefBuffer struct{ values astProofArguments }

func (b *boolRewriterRefBuffer) push(m *ASTManager, h ASTHandle) error {
	if e := m.IncRef(h); e != nil {
		return e
	}
	if e := b.values.push(m, h); e != nil {
		return errors.Join(e, m.DecRef(h))
	}
	return nil
}
func (b *boolRewriterRefBuffer) close(m *ASTManager) error {
	for i, h := range b.values.slice() {
		if !h.IsNull() {
			if e := m.DecRef(h); e != nil {
				return e
			}
			b.values.slice()[i] = ASTHandle{}
		}
	}
	b.values.close(m)
	return nil
}
func (rw *boolRewriter) mkAndAsOr(args []ASTHandle, out ASTRef) (err error) {
	m := rw.manager
	var newArgs boolRewriterRefBuffer
	defer func() { err = errors.Join(err, newArgs.close(m)) }()
	for _, arg := range args {
		e := func() (err error) {
			tmp, e := m.NewRef(ASTHandle{})
			if e != nil {
				return e
			}
			defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
			if e = rw.mkNot(arg, tmp); e != nil {
				return e
			}
			h, e := m.RefGet(tmp)
			if e != nil {
				return e
			}
			return newArgs.push(m, h)
		}()
		if e != nil {
			return e
		}
	}
	tmp, e := m.NewRef(ASTHandle{})
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
	if e = rw.mkOr(newArgs.values.slice(), tmp); e != nil {
		return e
	}
	h, e := m.RefGet(tmp)
	if e != nil {
		return e
	}
	return rw.mkNot(h, out)
}

func (rw *boolRewriter) mkAppCore(decl ASTHandle, args []ASTHandle, out ASTRef) (brStatus, error) {
	if rw.self != nil {
		if e := rw.configuredOpen(); e != nil {
			return brFailed, e
		}
	}
	fid, e := rw.manager.GetFamilyID(decl)
	if e != nil {
		return brFailed, e
	}
	if astSourceDebug && fid != 0 {
		return brFailed, ErrASTDomain
	}
	op, e := rw.manager.GetDeclKind(decl)
	if e != nil {
		return brFailed, e
	}
	switch op {
	case ASTOpEq, ASTOpIff:
		if len(args) != 2 {
			return brFailed, ErrASTDomain
		}
		return rw.mkEqCore(args[0], args[1], out)
	case ASTOpDistinct:
		return rw.mkDistinctCore(args, out)
	case ASTOpAnd:
		return rw.mkAndCore(args, out)
	case ASTOpOr:
		return rw.mkOrCore(args, out)
	case ASTOpNot:
		if len(args) != 1 {
			return brFailed, ErrASTDomain
		}
		return rw.mkNotCore(args[0], out)
	case ASTOpIte:
		if len(args) != 3 {
			return brFailed, ErrASTDomain
		}
		return rw.mkITECore(args[0], args[1], args[2], out)
	case ASTOpImplies:
		if len(args) != 2 {
			return brFailed, ErrASTDomain
		}
		return brDone, rw.mkImplies(args[0], args[1], out)
	case ASTOpXor:
		switch len(args) {
		case 0:
			return brFailed, nil
		case 1:
			return brDone, rw.manager.SetRef(out, args[0])
		case 2:
			return brDone, rw.mkXor(args[0], args[1], out)
		default:
			return brFailed, ErrASTDomain
		}
	default:
		return brFailed, nil
	}
}
func (rw *boolRewriter) mkApp(decl ASTHandle, args []ASTHandle, out ASTRef) error {
	status, e := rw.mkAppCore(decl, args, out)
	if e != nil || status != brFailed {
		return e
	}
	h, e := rw.manager.MkApp(decl, args)
	if e != nil {
		return e
	}
	return rw.manager.SetRef(out, h)
}
func (rw *boolRewriter) GetParamDescrs(d *ParamDescrs) error {
	return catalogCollectBoolRewriterParams(d)
}
