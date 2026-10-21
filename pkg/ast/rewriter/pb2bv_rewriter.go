// Source: src/ast/rewriter/pb2bv_rewriter.cpp;
// src/ast/rewriter/pb2bv_rewriter.h.
package rewriter

import (
	"errors"
	"math/big"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

type pb2BVConfiguration struct {
	DefaultConfiguration
	owner *PB2BVRewriter
}

func (c *pb2BVConfiguration) RewritePatterns() bool { return false }
func (c *pb2BVConfiguration) FlatAssoc(ast.ASTHandle) (bool, error) {
	return false, nil
}
func (c *pb2BVConfiguration) ReduceApp(decl ast.ASTHandle, args ast.AppArguments, out, proof ast.ASTRef) (RewriteStatus, error) {
	if err := c.owner.manager.SetRef(proof, ast.ASTHandle{}); err != nil {
		return RewriteFailed, err
	}
	return c.owner.rewriteApp(decl, args, out)
}

// PB2BVRewriter owns the concrete pseudo-Boolean traversal, sorting network,
// auxiliary trail, declarations and side constraints. The AST manager is
// borrowed. Do not copy this owner.
type PB2BVRewriter struct {
	self          *PB2BVRewriter
	manager       *ast.ASTManager
	pseudo        *PseudoBooleanRewriter
	runtime       *umemory.Runtime
	context       *ast.PsortASTContext
	network       *ast.PsortASTNetwork
	configuration pb2BVConfiguration
	traversal     *Traversal
	fresh         []ast.ASTHandle // owning declaration references
	freshLimits   []uint32
	freshSeen     uint32
	numTranslated uint32
	header        uint64
	closed        bool
}

// NewPB2BVRewriter constructs the source implementation graph. A nonnil
// partial owner returned with an error must be closed.
func NewPB2BVRewriter(m *ast.ASTManager) (*PB2BVRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	n := uint64(reflect.TypeOf(PB2BVRewriter{}).Size())
	if err := m.Budget().Reserve(n); err != nil {
		return nil, err
	}
	r := &PB2BVRewriter{manager: m, header: n}
	r.self = r
	var err error
	if r.pseudo, err = NewPseudoBooleanRewriter(m); err != nil {
		return r, err
	}
	if r.runtime, err = umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.Budget()}); err != nil {
		return r, err
	}
	if r.context, err = ast.NewPsortASTContext(m); err != nil {
		return r, err
	}
	if err = r.context.SetFreshPrefix("sn"); err != nil {
		return r, err
	}
	if r.network, err = ast.NewPsortASTNetwork(r.context, r.runtime, m.Budget(), nil); err != nil {
		return r, err
	}
	r.configuration.owner = r
	if r.traversal, err = NewTraversal(m, &r.configuration); err != nil {
		return r, err
	}
	return r, nil
}

func (r *PB2BVRewriter) open() error {
	if r == nil || r.self != r || r.closed || r.manager == nil || r.traversal == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

func pb2BVUnsigned(q *big.Rat) (uint32, bool) {
	if q == nil || q.Sign() < 0 || !q.IsInt() || !q.Num().IsUint64() {
		return 0, false
	}
	v := q.Num().Uint64()
	if v > uint64(^uint32(0)) {
		return 0, false
	}
	return uint32(v), true
}

func (r *PB2BVRewriter) syncFresh() error {
	n, err := r.context.FreshCount()
	if err != nil {
		return err
	}
	for r.freshSeen < n {
		app, err := r.context.FreshApplication(r.freshSeen)
		if err != nil {
			return err
		}
		decl, err := r.manager.GetDecl(app)
		if err != nil {
			return err
		}
		if err = r.manager.IncRef(decl); err != nil {
			return err
		}
		r.fresh = append(r.fresh, decl)
		r.freshSeen++
	}
	return nil
}

func (r *PB2BVRewriter) cardinality(full bool, kind int32, k uint32, args ast.AppArguments) (result ast.ASTHandle, err error) {
	v, err := ast.NewASTLiteralVector(r.manager.Budget())
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, v.Close()) }()
	for i := uint32(0); i < args.Count(); i++ {
		h, e := args.At(i)
		if e != nil {
			return result, e
		}
		if e = v.Push(h); e != nil {
			return result, e
		}
	}
	span, err := v.Span()
	if err != nil {
		return result, err
	}
	switch kind {
	case ast.ASTPBOpAtMost, ast.ASTPBOpLE:
		result, err = r.network.Le(full, k, args.Count(), span)
	case ast.ASTPBOpAtLeast, ast.ASTPBOpGE:
		result, err = r.network.Ge(full, k, args.Count(), span)
	case ast.ASTPBOpEQ:
		result, err = r.network.Eq(full, k, args.Count(), span)
	default:
		return result, ast.ErrASTDomain
	}
	if err == nil {
		err = r.syncFresh()
	}
	return result, err
}

func (r *PB2BVRewriter) rewriteApp(decl ast.ASTHandle, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	family, err := r.manager.GetFamilyID(decl)
	if err != nil {
		return RewriteFailed, err
	}
	if family != r.pseudo.family {
		return RewriteFailed, nil
	}
	kind, err := r.manager.GetDeclKind(decl)
	if err != nil || kind < ast.ASTPBOpAtMost || kind > ast.ASTPBOpEQ {
		return RewriteFailed, err
	}
	bound, err := r.pseudo.bound(decl)
	if err != nil {
		return RewriteFailed, err
	}
	k, unsigned := pb2BVUnsigned(bound)
	if (kind == ast.ASTPBOpAtLeast || kind == ast.ASTPBOpGE) && bound.Cmp(big.NewRat(1, 1)) == 0 {
		handles := make([]ast.ASTHandle, args.Count())
		for i := range handles {
			handles[i], err = args.At(uint32(i))
			if err != nil {
				return RewriteFailed, err
			}
		}
		result, err := r.manager.MkOrArgs(handles)
		if err != nil {
			return RewriteFailed, err
		}
		r.numTranslated++
		return RewriteDone, r.manager.SetRef(out, result)
	}
	unit := kind == ast.ASTPBOpAtMost || kind == ast.ASTPBOpAtLeast
	if !unit {
		unit, err = r.manager.PseudoBooleanHasUnitCoefficients(decl)
		if err != nil {
			return RewriteFailed, err
		}
	}
	if !unsigned || !unit {
		// Weighted conversion and arithmetic-ITE recognition are the next
		// retained source slice; leave them visible to the caller for now.
		return RewriteFailed, nil
	}
	result, err := r.cardinality(true, kind, k, args)
	if err != nil {
		return RewriteFailed, err
	}
	r.numTranslated++
	return RewriteDone, r.manager.SetRef(out, result)
}

// Rewrite traverses an expression and converts reached cardinality constraints.
func (r *PB2BVRewriter) Rewrite(expression ast.ASTHandle, out ast.ASTRef) error {
	if err := r.open(); err != nil {
		return err
	}
	return r.traversal.Rewrite(expression, out)
}

func (r *PB2BVRewriter) NumSteps() (uint32, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return r.traversal.NumSteps()
}

func (r *PB2BVRewriter) NumTranslated() (uint32, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return r.numTranslated, nil
}

// FreshConstants returns borrowed declaration handles retained by this owner.
func (r *PB2BVRewriter) FreshConstants() ([]ast.ASTHandle, error) {
	if err := r.open(); err != nil {
		return nil, err
	}
	return append([]ast.ASTHandle(nil), r.fresh...), nil
}

// FlushSideConstraints transfers owning clause references to the caller.
func (r *PB2BVRewriter) FlushSideConstraints() ([]ast.ASTHandle, error) {
	if err := r.open(); err != nil {
		return nil, err
	}
	return r.context.TakeClauses()
}

func (r *PB2BVRewriter) Push() error {
	if err := r.open(); err != nil {
		return err
	}
	r.freshLimits = append(r.freshLimits, uint32(len(r.fresh)))
	return nil
}

func (r *PB2BVRewriter) Pop(scopes uint32) error {
	if err := r.open(); err != nil {
		return err
	}
	clauses, err := r.context.ClauseCount()
	if err != nil {
		return err
	}
	if clauses != 0 || scopes > uint32(len(r.freshLimits)) {
		return ast.ErrASTDomain
	}
	if scopes != 0 {
		newSize := uint32(len(r.freshLimits)) - scopes
		limit := r.freshLimits[newSize]
		for i := len(r.fresh) - 1; i >= int(limit); i-- {
			if err = r.manager.DecRef(r.fresh[i]); err != nil {
				return err
			}
			r.fresh[i] = ast.ASTHandle{}
		}
		r.fresh = r.fresh[:limit]
		r.freshLimits = r.freshLimits[:newSize]
	}
	return r.traversal.Reset()
}

func (r *PB2BVRewriter) Statistics() (ast.PsortASTStats, error) {
	if err := r.open(); err != nil {
		return ast.PsortASTStats{}, err
	}
	return r.network.Stats()
}

func (r *PB2BVRewriter) Close() error {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	if r.traversal != nil {
		if err := r.traversal.Close(); err != nil {
			return err
		}
		r.traversal = nil
	}
	if r.network != nil {
		if err := r.network.Close(); err != nil {
			return err
		}
		r.network = nil
	}
	if r.context != nil {
		if err := r.context.Close(); err != nil {
			return err
		}
		r.context = nil
	}
	for i := len(r.fresh) - 1; i >= 0; i-- {
		if err := r.manager.DecRef(r.fresh[i]); err != nil {
			return err
		}
		r.fresh[i] = ast.ASTHandle{}
	}
	r.fresh = nil
	r.freshLimits = nil
	if r.runtime != nil {
		if err := r.runtime.Close(); err != nil {
			return err
		}
		r.runtime = nil
	}
	if r.pseudo != nil {
		if err := r.pseudo.Close(); err != nil {
			return err
		}
		r.pseudo = nil
	}
	r.manager.Budget().Release(r.header)
	r.header = 0
	r.closed = true
	return nil
}
