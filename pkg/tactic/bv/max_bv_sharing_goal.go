// Source: src/tactic/bv/max_bv_sharing_tactic.cpp:238-271.
package bv

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

var ErrSharingMaxSteps = errors.New("max. steps exceeded")
var ErrSharingMaxMemory = errors.New("max. memory exceeded")

// SharingGoalPass applies the concrete BV sharing reducer to goal formulas.
// Parameter/resource limits and lifecycle remain in the enclosing tactic's
// unfinished scope.
type SharingGoalPass struct {
	manager     *ast.ASTManager
	memory      *memory.Runtime
	cooperation *cooperate.Cooperation
	family      ast.FamilyID
	maxArgs     uint32
	maxSteps    uint32
	maxMemory   uint64
}

func NewSharingGoalPass(m *ast.ASTManager, maxArgs uint32) (*SharingGoalPass, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	numerals := m.NumeralService()
	if numerals == nil || numerals.RuntimeOwner() == nil {
		return nil, ast.ErrASTHandle
	}
	runtime := numerals.RuntimeOwner()
	if _, e := runtime.AllocationSize(); e != nil {
		return nil, e
	}
	fid, e := m.RegisterBitVectorPlugin()
	if e != nil {
		return nil, e
	}
	return &SharingGoalPass{manager: m, memory: runtime, family: fid, maxArgs: maxArgs, maxSteps: math.MaxUint32, maxMemory: math.MaxUint64}, nil
}

// BindCooperation borrows the source process owner for sharing checkpoints.
func (p *SharingGoalPass) BindCooperation(owner *cooperate.Cooperation) error {
	if p == nil || p.manager == nil {
		return ast.ErrASTHandle
	}
	if owner == nil {
		return arena.ErrConfig
	}
	if _, err := owner.Enabled(); err != nil {
		return err
	}
	p.cooperation = owner
	return nil
}

// UpdateParams reads the three unsigned settings used by rw_cfg::updt_params.
// The memory limit reads the source process allocation counter.
func (p *SharingGoalPass) UpdateParams(c *uparams.ParamsContext, value *uparams.ParamsRef) error {
	if p == nil || p.manager == nil || p.memory == nil || c == nil || c.SymbolsField != p.manager.Symbols {
		return ast.ErrASTHandle
	}
	paramsNumerals := c.Numerals()
	if paramsNumerals == nil || paramsNumerals.RuntimeOwner() == nil {
		return ast.ErrASTHandle
	}
	process, paramsProcess := p.memory.ProcessOwner(), paramsNumerals.RuntimeOwner().ProcessOwner()
	if (process != nil || paramsProcess != nil) && process != paramsProcess {
		return ast.ErrASTHandle
	}
	memName, stepsName, argsName := "max_memory", "max_steps", "max_args"
	mem, e := c.GetUIntName(value, &memName, math.MaxUint32)
	if e != nil {
		return e
	}
	steps, e := c.GetUIntName(value, &stepsName, math.MaxUint32)
	if e != nil {
		return e
	}
	args, e := c.GetUIntName(value, &argsName, 128)
	if e != nil {
		return e
	}
	limit := uint64(mem) << 20
	if mem == math.MaxUint32 {
		limit = math.MaxUint64
	}
	p.maxMemory, p.maxSteps, p.maxArgs = limit, steps, args
	return nil
}

func NewSharingGoalPassWithParams(m *ast.ASTManager, c *uparams.ParamsContext, value *uparams.ParamsRef) (*SharingGoalPass, error) {
	p, e := NewSharingGoalPass(m, 128)
	if e != nil {
		return nil, e
	}
	if e = p.UpdateParams(c, value); e != nil {
		return nil, e
	}
	return p, nil
}

type sharingWalk struct {
	pass  *SharingGoalPass
	steps uint32
}

func (w *sharingWalk) advance() error {
	w.steps++
	if w.pass.cooperation != nil {
		if err := w.pass.cooperation.Cooperate("max bv sharing"); err != nil {
			return err
		}
	}
	used, e := w.pass.memory.AllocationSize()
	if e != nil {
		return e
	}
	if used > w.pass.maxMemory {
		return ErrSharingMaxMemory
	}
	if w.steps > w.pass.maxSteps {
		return ErrSharingMaxSteps
	}
	return nil
}

// Apply follows the source's sequential goal updates: pairs found in an
// earlier formula can guide the next formula. The result is the input goal.
func (p *SharingGoalPass) Apply(goal *tactic.Goal) (_ *tactic.Goal, result error) {
	if p == nil || p.manager == nil || goal == nil || goal.Manager() != p.manager {
		return nil, ast.ErrASTHandle
	}
	if e := goal.CheckOwner(); e != nil {
		return nil, e
	}
	report, e := tactic.NewTacticReport("max-bv-sharing", goal)
	if e != nil {
		return nil, e
	}
	defer func() { result = errors.Join(result, report.Close()) }()
	proofs, e := goal.ProofsEnabled()
	if e != nil {
		return nil, e
	}
	r, e := NewSharingReducer(p.manager, p.family, p.maxArgs)
	if e != nil {
		return nil, e
	}
	defer func() { result = errors.Join(result, r.Close()) }()
	memo := make(map[ast.ASTHandle]ast.ASTHandle)
	proofMemo := make(map[ast.ASTHandle]sharingProofResult)
	n, e := goal.Size()
	if e != nil {
		return nil, e
	}
	for i := uint32(0); i < n; i++ {
		walk := sharingWalk{pass: p}
		current, e := goal.Size()
		if e != nil {
			return nil, e
		}
		if i >= current { // A preceding update may have made the goal inconsistent.
			break
		}
		form, e := goal.Form(i)
		if e != nil {
			return nil, e
		}
		dep, e := goal.Dependency(i)
		if e != nil {
			return nil, e
		}
		var rewritten, rewrittenProof ast.ASTHandle
		if proofs {
			var step sharingProofResult
			step, e = p.rewriteWithProof(form, r, proofMemo, &walk)
			if e != nil {
				return nil, e
			}
			rewritten = step.expr
			rewrittenProof, e = goal.Proof(i)
			if e != nil {
				return nil, e
			}
			if !step.proof.IsNull() {
				rewrittenProof, e = p.manager.MkProofModusPonens(rewrittenProof, step.proof)
				if e != nil {
					return nil, e
				}
				if e = r.keep(rewrittenProof); e != nil {
					return nil, e
				}
			}
		} else {
			rewritten, e = p.rewriteApplication(form, r, memo, &walk)
			if e != nil {
				return nil, e
			}
		}
		if e := goal.Update(i, rewritten, rewrittenProof, dep); e != nil {
			return nil, e
		}
	}
	if e := goal.IncDepth(); e != nil {
		return nil, e
	}
	return goal, nil
}

func (p *SharingGoalPass) rewriteApplication(h ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]ast.ASTHandle, walk *sharingWalk) (ast.ASTHandle, error) {
	if e := walk.advance(); e != nil {
		return ast.ASTHandle{}, e
	}
	original := h
	if mapped, ok := memo[h]; ok {
		return mapped, nil
	}
	kind, e := p.manager.GetKind(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if kind != ast.ASTApp {
		if kind == ast.ASTQuantifier {
			return p.rewriteQuantifier(h, r, memo, walk)
		}
		memo[h] = h
		return h, nil
	}
	decl, e := p.manager.GetDecl(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	n, e := p.manager.GetNumArgs(h)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	args := make([]ast.ASTHandle, n)
	changed := false
	start := 0
	if n == 3 {
		fid, e := p.manager.GetFamilyID(decl)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		k, e := p.manager.GetDeclKind(decl)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if fid == p.manager.BasicFamilyID() && k == ast.ASTOpIte {
			cond, e := p.manager.GetArg(h, 0)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			args[0], e = p.rewriteApplication(cond, r, memo, walk)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if e := walk.advance(); e != nil {
				return ast.ASTHandle{}, e
			}
			selected := uint32(0)
			if p.manager.IsTrue(args[0]) {
				selected = 1
			}
			if p.manager.IsFalse(args[0]) {
				selected = 2
			}
			if selected != 0 {
				branch, e := p.manager.GetArg(h, selected)
				if e != nil {
					return ast.ASTHandle{}, e
				}
				result, e := p.rewriteApplication(branch, r, memo, walk)
				if e != nil {
					return ast.ASTHandle{}, e
				}
				if e := walk.advance(); e != nil {
					return ast.ASTHandle{}, e
				}
				memo[original] = result
				return result, nil
			}
			changed = args[0] != cond
			start = 1
		}
	}
	for i := start; i < len(args); i++ {
		arg, e := p.manager.GetArg(h, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		args[i], e = p.rewriteApplication(arg, r, memo, walk)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e := walk.advance(); e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || args[i] != arg
	}
	if replacement, done, e := r.ReduceApp(decl, args); e != nil {
		return ast.ASTHandle{}, e
	} else if done {
		memo[h] = replacement
		return replacement, nil
	}
	if changed {
		h, e = p.manager.MkApp(decl, args)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e := r.keep(h); e != nil {
			return ast.ASTHandle{}, e
		}
	}
	memo[original] = h
	return h, nil
}

func (p *SharingGoalPass) rewriteQuantifier(q ast.ASTHandle, r *SharingReducer, memo map[ast.ASTHandle]ast.ASTHandle, walk *sharingWalk) (ast.ASTHandle, error) {
	body, e := p.manager.GetQuantifierBody(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	newBody, e := p.rewriteApplication(body, r, memo, walk)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if e := walk.advance(); e != nil {
		return ast.ASTHandle{}, e
	}
	changed := newBody != body
	nPatterns, e := p.manager.GetNumPatterns(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	patterns := make([]ast.ASTHandle, nPatterns)
	for i := range patterns {
		old, e := p.manager.GetPattern(q, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		patterns[i], e = p.rewriteApplication(old, r, memo, walk)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e := walk.advance(); e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || patterns[i] != old
	}
	nNoPatterns, e := p.manager.GetNumNoPatterns(q)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	noPatterns := make([]ast.ASTHandle, nNoPatterns)
	for i := range noPatterns {
		old, e := p.manager.GetNoPattern(q, uint32(i))
		if e != nil {
			return ast.ASTHandle{}, e
		}
		noPatterns[i], e = p.rewriteApplication(old, r, memo, walk)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if e := walk.advance(); e != nil {
			return ast.ASTHandle{}, e
		}
		changed = changed || noPatterns[i] != old
	}
	if !changed {
		memo[q] = q
		return q, nil
	}
	updated, e := p.manager.UpdateQuantifierAnnotations(q, patterns, noPatterns, newBody)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if e := r.keep(updated); e != nil {
		return ast.ASTHandle{}, e
	}
	memo[q] = updated
	return updated, nil
}
