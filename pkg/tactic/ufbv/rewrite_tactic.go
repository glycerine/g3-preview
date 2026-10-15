// Source: src/tactic/ufbv/ufbv_rewriter_tactic.cpp:26-113.
package ufbv

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

var ErrRewriterUnsatCores = errors.New("g3: ufbv-rewriter does not support unsat cores")

// RewriterTactic owns the configured Boolean reducer used by the concrete
// UFBV formula processor. Its AST and parameter services are borrowed.
type RewriterTactic struct {
	self    *RewriterTactic
	manager *ast.ASTManager
	context *uparams.ParamsContext
	globals *uparams.GlobalParamsService
	boolean *rewriter.BoolRewriter
	header  uint64
	closed  bool
}

// NewRewriterTactic applies the source's fixed elim_and=true setting to its
// internal Boolean reducer. Other tactic parameters do not change this pass.
func NewRewriterTactic(m *ast.ASTManager, c *uparams.ParamsContext, g *uparams.GlobalParamsService) (*RewriterTactic, error) {
	if m == nil || c == nil || g == nil || c.SymbolsField != m.Symbols || g.Context != c {
		return nil, uparams.ErrParamsOwner
	}
	n := uint64(reflect.TypeOf(RewriterTactic{}).Size())
	if e := m.Budget().Reserve(n); e != nil {
		return nil, e
	}
	t := &RewriterTactic{manager: m, context: c, globals: g, header: n}
	t.self = t
	local, e := c.NewValue()
	if e != nil {
		return t, e
	}
	key := "elim_and"
	if e = c.SetBoolName(&local, &key, true); e != nil {
		return t, errors.Join(e, c.Del(&local))
	}
	t.boolean, e = rewriter.NewBoolRewriter(m, c, g, &local)
	return t, errors.Join(e, c.Del(&local))
}

func (t *RewriterTactic) check() error {
	if t == nil || t.self != t || t.closed || t.manager == nil || t.boolean == nil {
		return ast.ErrASTHandle
	}
	_, e := t.manager.ProofsEnabled()
	return e
}

// Apply follows the source's goal reset, assertion and depth transition. It
// returns the same goal with an independent reference, matching the source
// goal_ref_buffer push. The caller releases the result with DecRef.
func (t *RewriterTactic) Apply(goal *tactic.Goal) (_ *tactic.Goal, result error) {
	if e := t.check(); e != nil {
		return nil, e
	}
	if goal == nil || goal.Manager() != t.manager {
		return nil, ast.ErrASTHandle
	}
	if e := goal.CheckOwner(); e != nil {
		return nil, e
	}
	report, e := tactic.NewTacticReport("ufbv-rewriter", goal)
	if e != nil {
		return nil, e
	}
	defer func() { result = errors.Join(result, report.Close()) }()
	cores, e := goal.UnsatCoresEnabled()
	if e != nil {
		return nil, e
	}
	if cores {
		return nil, ErrRewriterUnsatCores
	}
	proofs, e := goal.ProofsEnabled()
	if e != nil {
		return nil, e
	}
	n, e := goal.Size()
	if e != nil {
		return nil, e
	}
	forms := make([]ast.ASTHandle, n)
	formPins := 0
	defer func() {
		// The source forms ref_vector releases its entries after the younger
		// proof and rewritten-formula owners, in insertion order.
		for i := 0; i < formPins; i++ {
			result = errors.Join(result, t.manager.DecRef(forms[i]))
		}
	}()
	var proofRefs []ast.ASTRef
	if proofs {
		proofRefs = make([]ast.ASTRef, 0, n)
	}
	defer func() {
		for i := len(proofRefs) - 1; i >= 0; i-- {
			result = errors.Join(result, t.manager.DelRef(&proofRefs[i]))
		}
	}()
	for i := range forms {
		forms[i], e = goal.Form(uint32(i))
		if e != nil {
			return nil, e
		}
		if e = t.manager.IncRef(forms[i]); e != nil {
			return nil, e
		}
		formPins++
		if proofs {
			pr, e := goal.Proof(uint32(i))
			if e != nil {
				return nil, e
			}
			ref, e := t.manager.NewRef(pr)
			if e != nil {
				return nil, e
			}
			proofRefs = append(proofRefs, ref)
		}
	}
	processor, e := newFormulaProcessor(t.manager, t.boolean)
	if e != nil {
		if processor != nil {
			result = errors.Join(result, processor.close())
		}
		return nil, errors.Join(e, result)
	}
	output, e := processor.run(forms)
	if e != nil {
		return nil, errors.Join(e, processor.close())
	}
	defer func() {
		for i := len(output) - 1; i >= 0; i-- {
			result = errors.Join(result, t.manager.DelRef(&output[i]))
		}
	}()
	if e := processor.close(); e != nil {
		return nil, e
	}
	if e := goal.Reset(); e != nil {
		return nil, e
	}
	for i, ref := range output {
		h, e := t.manager.RefGet(ref)
		if e != nil {
			return nil, e
		}
		if proofs {
			pr, e := t.manager.RefGet(proofRefs[i])
			if e != nil {
				return nil, e
			}
			if e := goal.AssertWithProof(h, pr, ast.ASTDependency{}); e != nil {
				return nil, e
			}
		} else if e := goal.Assert(h); e != nil {
			return nil, e
		}
	}
	if e := goal.IncDepth(); e != nil {
		return nil, e
	}
	if e := goal.IncRef(); e != nil {
		return nil, e
	}
	return goal, nil
}

func (t *RewriterTactic) Close() error {
	if t == nil || t.self != t {
		return ast.ErrASTHandle
	}
	if t.closed {
		return nil
	}
	if t.boolean != nil {
		if e := t.boolean.Close(); e != nil {
			return e
		}
		t.boolean = nil
	}
	t.manager.Budget().Release(t.header)
	t.header = 0
	t.closed = true
	return nil
}
