// Source: src/ast/rewriter/rewriter.h:365-408;
// src/ast/rewriter/bool_rewriter.h:181-200.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "reflect"

// Configurations are cold owners; frames, bindings and results use concrete
// handle storage. This is the source's no-proof traversal specialization.
type rewriterConfiguration interface {
	cacheAllResults() bool
	cacheResults() bool
	flatAssoc(ASTHandle) (bool, error)
	rewritePatterns() bool
	maxScopesExceeded(uint32) bool
	maxFramesExceeded(uint32) bool
	maxStepsExceeded(uint32) bool
	preVisit(ASTHandle) (bool, error)
	reduceApp(ASTHandle, astAppArguments, ASTRef, ASTRef) (brStatus, error)
	reduceVar(ASTHandle, ASTRef, ASTRef) (bool, error)
	reduceQuantifier(ASTHandle, ASTHandle, astAppArguments, astAppArguments, ASTRef, ASTRef) (bool, error)
	getMacro(ASTHandle) (ASTHandle, ASTHandle, ASTHandle, bool, error)
	getSubst(ASTHandle) (ASTHandle, ASTHandle, bool, error)
	reset()
	cleanup()
}
type defaultRewriterConfig struct{ manager *ASTManager }

func (*defaultRewriterConfig) cacheAllResults() bool             { return false }
func (*defaultRewriterConfig) cacheResults() bool                { return true }
func (*defaultRewriterConfig) flatAssoc(ASTHandle) (bool, error) { return false, nil }
func (*defaultRewriterConfig) rewritePatterns() bool             { return true }
func (*defaultRewriterConfig) maxScopesExceeded(uint32) bool     { return false }
func (*defaultRewriterConfig) maxFramesExceeded(uint32) bool     { return false }
func (*defaultRewriterConfig) maxStepsExceeded(uint32) bool      { return false }
func (*defaultRewriterConfig) preVisit(ASTHandle) (bool, error)  { return true, nil }
func (*defaultRewriterConfig) reduceApp(ASTHandle, astAppArguments, ASTRef, ASTRef) (brStatus, error) {
	return brFailed, nil
}
func (*defaultRewriterConfig) reduceVar(ASTHandle, ASTRef, ASTRef) (bool, error) { return false, nil }
func (*defaultRewriterConfig) reduceQuantifier(ASTHandle, ASTHandle, astAppArguments, astAppArguments, ASTRef, ASTRef) (bool, error) {
	return false, nil
}
func (*defaultRewriterConfig) getMacro(ASTHandle) (ASTHandle, ASTHandle, ASTHandle, bool, error) {
	return ASTHandle{}, ASTHandle{}, ASTHandle{}, false, nil
}
func (*defaultRewriterConfig) getSubst(ASTHandle) (ASTHandle, ASTHandle, bool, error) {
	return ASTHandle{}, ASTHandle{}, false, nil
}
func (*defaultRewriterConfig) reset()   {}
func (*defaultRewriterConfig) cleanup() {}

type betaReducerConfig struct{ defaultRewriterConfig }

func (c *betaReducerConfig) preVisit(t ASTHandle) (bool, error) {
	g, e := c.manager.IsGround(t)
	return !g, e
}

type betaReducer struct {
	engine rewriterEngine
	cfg    betaReducerConfig
}

func newBetaReducer(m *ASTManager) (*betaReducer, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(betaReducer{}).Size())
	if e := m.budget.Reserve(n); e != nil {
		return nil, e
	}
	r := &betaReducer{}
	r.cfg.manager = m
	r.engine.header = n
	return r, r.engine.init(m, &r.cfg)
}
func (r *betaReducer) Rewrite(t ASTHandle, out ASTRef) error { return r.engine.Rewrite(t, out) }
func (r *betaReducer) Close() error                          { return r.engine.Close() }

type boolRewriterStarConfig struct {
	defaultRewriterConfig
	rw boolRewriter
}

func (c *boolRewriterStarConfig) rewritePatterns() bool { return false }
func (c *boolRewriterStarConfig) flatAssoc(d ASTHandle) (bool, error) {
	if !c.rw.flat {
		return false, nil
	}
	fid, e := c.manager.GetFamilyID(d)
	if e != nil || fid != 0 {
		return false, e
	}
	op, e := c.manager.GetDeclKind(d)
	return op == ASTOpAnd || op == ASTOpOr, e
}
func (c *boolRewriterStarConfig) reduceApp(d ASTHandle, args astAppArguments, out, pr ASTRef) (brStatus, error) {
	if e := c.manager.ResetRef(pr); e != nil {
		return brFailed, e
	}
	fid, e := c.manager.GetFamilyID(d)
	if e != nil || fid != 0 {
		return brFailed, e
	}
	return c.rw.mkAppCoreArguments(d, args, out)
}

type boolRewriterStar struct {
	engine rewriterEngine
	cfg    boolRewriterStarConfig
}

func newBoolRewriterStar(m *ASTManager, c *ParamsContext, g *GlobalParamsService, p *ParamsRef) (*boolRewriterStar, error) {
	if m == nil || c == nil || g == nil || c.symbols != m.symbols || g.context != c {
		return nil, ErrParamsOwner
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(boolRewriterStar{}).Size())
	if e := m.budget.Reserve(n); e != nil {
		return nil, e
	}
	r := &boolRewriterStar{}
	r.cfg.manager = m
	r.engine.header = n
	if e := r.engine.init(m, &r.cfg); e != nil {
		return r, e
	}
	return r, r.cfg.rw.init(m, c, g, p)
}
func (r *boolRewriterStar) Rewrite(t ASTHandle, out ASTRef) error { return r.engine.Rewrite(t, out) }
func (r *boolRewriterStar) Close() error {
	if r.cfg.rw.self != nil {
		if e := r.cfg.rw.Close(); e != nil {
			return e
		}
	}
	return r.engine.Close()
}

func astSliceArguments(args []ASTHandle) astAppArguments {
	return astAppArguments{slice: args, count: uint32(len(args))}
}
func (rw *boolRewriter) mkAppCore(decl ASTHandle, args []ASTHandle, out ASTRef) (brStatus, error) {
	return rw.mkAppCoreArguments(decl, astSliceArguments(args), out)
}
func (rw *boolRewriter) mkAppCoreArguments(decl ASTHandle, args astAppArguments, out ASTRef) (brStatus, error) {
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
	case ASTOpAnd:
		if rw.elimAnd {
			return brDone, rw.mkAndAsOrArguments(args, out)
		}
		if rw.flat {
			return rw.mkFlatCore(args, out, false)
		}
		return rw.mkNFlatCore(args, out, false)
	case ASTOpOr:
		if rw.flat {
			return rw.mkFlatCore(args, out, true)
		}
		return rw.mkNFlatCore(args, out, true)
	case ASTOpDistinct:
		return rw.mkDistinctCoreArguments(args, out)
	}
	// Scalar callbacks have at most three arguments, read without a heap vector.
	if args.count > 3 {
		switch op {
		case ASTOpEq, ASTOpIff, ASTOpNot, ASTOpIte, ASTOpImplies, ASTOpXor:
			return brFailed, ErrASTDomain
		}
		return brFailed, nil
	}
	var scalar [3]ASTHandle
	for i := uint32(0); i < args.count; i++ {
		scalar[i], e = args.at(i)
		if e != nil {
			return brFailed, e
		}
	}
	switch op {
	case ASTOpEq, ASTOpIff:
		if args.count != 2 {
			return brFailed, ErrASTDomain
		}
		return rw.mkEqCore(scalar[0], scalar[1], out)
	case ASTOpNot:
		if args.count != 1 {
			return brFailed, ErrASTDomain
		}
		return rw.mkNotCore(scalar[0], out)
	case ASTOpIte:
		if args.count != 3 {
			return brFailed, ErrASTDomain
		}
		return rw.mkITECore(scalar[0], scalar[1], scalar[2], out)
	case ASTOpImplies:
		if args.count != 2 {
			return brFailed, ErrASTDomain
		}
		return brDone, rw.mkImplies(scalar[0], scalar[1], out)
	case ASTOpXor:
		switch args.count {
		case 0:
			return brFailed, nil
		case 1:
			return brDone, rw.manager.SetRef(out, scalar[0])
		case 2:
			return brDone, rw.mkXor(scalar[0], scalar[1], out)
		default:
			return brFailed, ErrASTDomain
		}
	default:
		return brFailed, nil
	}
}
