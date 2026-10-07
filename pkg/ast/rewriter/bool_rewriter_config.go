// Source: src/ast/rewriter/bool_rewriter.cpp:23-87;
// src/ast/rewriter/bool_rewriter.h:85-107,113-126,132-156;
// src/ast/rewriter/bool_rewriter_params.hpp:6-30;
// src/util/ref_buffer.h:35-68.
package rewriter

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// The standalone cold constructor charges its complete header once. A failed
// reached parameter constructor returns its owner for explicit Close, following
// the existing generated-helper lifetime engine. AST/parameter services borrow.

func newBoolRewriter(m *ast.ASTManager, c *util.ParamsContext, g *util.GlobalParamsService, p *util.ParamsRef) (*boolRewriter, error) {
	if m == nil || c == nil || g == nil || c.SymbolsField != m.Symbols || g.Context != c {
		return nil, util.ErrParamsOwner
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(boolRewriter{}).Size())
	if e := m.BudgetField.Reserve(n); e != nil {
		return nil, e
	}
	rw := &boolRewriter{header: n}
	return rw, rw.init(m, c, g, p)
}
func (rw *boolRewriter) init(m *ast.ASTManager, c *util.ParamsContext, g *util.GlobalParamsService, p *util.ParamsRef) error {
	rw.manager = m
	rw.self = rw
	rw.params = util.ModuleParamsOwner{Budget: g.Config.Budget}
	rw.params.Self = &rw.params
	if e := rw.params.Bind(c, g); e != nil {
		return e
	}
	e := rw.updateParams(p)
	if e == nil {
		rw.ready = true
	}
	return e
}
func (rw *boolRewriter) configuredOpen() error {
	if rw == nil || rw.self != rw || rw.closed {
		return util.ErrParamsOwner
	}
	if !rw.ready {
		return util.ErrParamsPrecondition
	}
	_, e := rw.manager.ProofsEnabled()
	return e
}
func (rw *boolRewriter) UpdtParams(p *util.ParamsRef) error {
	if e := rw.configuredOpen(); e != nil {
		return e
	}
	return rw.updateParams(p)
}
func (rw *boolRewriter) updateParams(local *util.ParamsRef) (err error) {
	o := &rw.params
	if o.Self != o || o.Running || o.DestructorCause != nil {
		return util.ErrParamsPrecondition
	}
	if local == nil {
		empty, e := o.Context.NewValue()
		if e != nil {
			return e
		}
		local = &empty
		defer func() { err = errors.Join(err, o.Context.Del(&empty)) }()
	}
	if e := o.Context.Check(local); e != nil {
		return e
	}
	o.Running = true
	defer func() { o.Running = false }()
	f, e := o.BeginHelper(local, "rewriter")
	if f != nil {
		defer func() { err = o.FinishHelper(f, err) }()
	}
	if e != nil {
		return e
	}
	b, e := o.FrameBool(f, "flat", true)
	if e != nil {
		return e
	}
	rw.flat = b
	b, e = o.FrameBool(f, "elim_and", false)
	if e != nil {
		return e
	}
	rw.elimAnd = b
	b, e = o.FrameBool(f, "local_ctx", false)
	if e != nil {
		return e
	}
	rw.localContext = b
	u, e := o.FrameUInt(f, "local_ctx_limit", ^uint32(0))
	if e != nil {
		return e
	}
	rw.localContextLimit = u
	b, e = o.FrameBool(f, "blast_distinct", false)
	if e != nil {
		return e
	}
	rw.blastDistinct = b
	u, e = o.FrameUInt(f, "blast_distinct_threshold", ^uint32(0))
	if e != nil {
		return e
	}
	rw.blastDistinctThreshold = u
	b, e = o.FrameBool(f, "ite_extra_rules", false)
	if e != nil {
		return e
	}
	rw.iteExtraRules = b
	return nil
}
func (rw *boolRewriter) ResetLocalContextCost() { rw.localContextCost = 0 }
func (rw *boolRewriter) Close() error {
	if rw == nil || rw.self != rw {
		return util.ErrParamsOwner
	}
	if rw.closed {
		return nil
	}
	if e := rw.params.CloseHelpers(); e != nil {
		return e
	}
	rw.manager.BudgetField.Release(rw.header)
	rw.header = 0
	rw.closed = true
	return nil
}

func (rw *boolRewriter) mkAndCore(args []ast.ASTHandle, out ast.ASTRef) (brStatus, error) {
	if rw.elimAnd {
		return brDone, rw.mkAndAsOr(args, out)
	}
	if rw.flat {
		return rw.mkFlatAndCore(args, out)
	}
	return rw.mkNFlatAndCore(args, out)
}
func (rw *boolRewriter) mkOrCore(args []ast.ASTHandle, out ast.ASTRef) (brStatus, error) {
	if rw.flat {
		return rw.mkFlatOrCore(args, out)
	}
	return rw.mkNFlatOrCore(args, out)
}

// Concrete expr_ref_buffer: the underlying inline16/doubling pointer storage
// carries owning AST references here, including repeated entries.
type boolRewriterRefBuffer struct {
	values util.PointerBuffer[ast.ASTHandle]
}

func (b *boolRewriterRefBuffer) push(m *ast.ASTManager, h ast.ASTHandle) error {
	if e := m.IncRef(h); e != nil {
		return e
	}
	if e := b.values.Push(m.Budget(), h); e != nil {
		return errors.Join(e, m.DecRef(h))
	}
	return nil
}
func (b *boolRewriterRefBuffer) close(m *ast.ASTManager) error {
	for i, h := range b.values.Slice() {
		if !h.IsNull() {
			if e := m.DecRef(h); e != nil {
				return e
			}
			b.values.Slice()[i] = ast.ASTHandle{}
		}
	}
	b.values.Close(m.Budget())
	return nil
}
func (rw *boolRewriter) mkAndAsOr(args []ast.ASTHandle, out ast.ASTRef) (err error) {
	return rw.mkAndAsOrArguments(astSliceArguments(args), out)
}
func (rw *boolRewriter) mkAndAsOrArguments(args ast.AppArguments, out ast.ASTRef) (err error) {
	m := rw.manager
	var newArgs boolRewriterRefBuffer
	defer func() { err = errors.Join(err, newArgs.close(m)) }()
	for i := uint32(0); i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return e
		}
		e = func() (err error) {
			tmp, e := m.NewRef(ast.ASTHandle{})
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
	tmp, e := m.NewRef(ast.ASTHandle{})
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
	if e = rw.mkOr(newArgs.values.Slice(), tmp); e != nil {
		return e
	}
	h, e := m.RefGet(tmp)
	if e != nil {
		return e
	}
	return rw.mkNot(h, out)
}

func (rw *boolRewriter) mkApp(decl ast.ASTHandle, args []ast.ASTHandle, out ast.ASTRef) error {
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
func (rw *boolRewriter) GetParamDescrs(d *util.ParamDescrs) error {
	return catalogCollectBoolRewriterParams(d)
}
