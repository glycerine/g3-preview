// Source: src/tactic/aig/aig_tactic.cpp:65-113.
package aig

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Pass applies the source AIG tactic's two goal-rewriting branches.
type Pass struct {
	DefaultGateEncoding bool
	PerAssertion        bool
	maxMemory           uint64
	cooperation         *cooperate.Cooperation
	paramsSymbols       *usymbol.Manager
}

func NewPass() *Pass {
	return &Pass{DefaultGateEncoding: true, PerAssertion: true, maxMemory: math.MaxUint64}
}

// UpdateParams follows aig_tactic::updt_params, including the UINT_MAX
// sentinel used by megabytes_to_bytes on the selected 64-bit profile.
func (p *Pass) UpdateParams(c *uparams.ParamsContext, value *uparams.ParamsRef) error {
	if p == nil || c == nil || c.SymbolsField == nil {
		return ErrOwner
	}
	memoryName, encodingName, perName := "max_memory", "aig_default_gate_encoding", "aig_per_assertion"
	mb, err := c.GetUIntName(value, &memoryName, math.MaxUint32)
	if err != nil {
		return err
	}
	encoding, err := c.GetBoolName(value, &encodingName, true)
	if err != nil {
		return err
	}
	per, err := c.GetBoolName(value, &perName, true)
	if err != nil {
		return err
	}
	limit := uint64(mb) << 20
	if mb == math.MaxUint32 {
		limit = math.MaxUint64
	}
	p.maxMemory, p.DefaultGateEncoding, p.PerAssertion, p.paramsSymbols = limit, encoding, per, c.SymbolsField
	return nil
}

// BindCooperation borrows the process cooperation owner used at AIG checkpoints.
func (p *Pass) BindCooperation(owner *cooperate.Cooperation) error {
	if p == nil || owner == nil {
		return ErrOwner
	}
	if _, err := owner.Enabled(); err != nil {
		return err
	}
	p.cooperation = owner
	return nil
}

func (p *Pass) closeManager(m *Manager) error {
	if err := m.Close(); err != nil {
		return err
	}
	m.traceTacticDeallocated()
	return nil
}

func (p *Pass) rewriteOne(m *Manager, g *tactic.Goal, i uint32) (err error) {
	form, err := g.Form(i)
	if err != nil {
		return err
	}
	r, err := m.MkAIG(form)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.Close()) }()
	shared, err := m.MaxSharing(r)
	if err != nil {
		return err
	}
	closeErr := r.Close()
	r = shared
	if closeErr != nil {
		return closeErr
	}
	newForm, err := m.ToFormula(r)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.releaseAST(&newForm)) }()
	h, err := m.ast.RefGet(newForm)
	if err != nil {
		return err
	}
	dep, err := g.Dependency(i)
	if err != nil {
		return err
	}
	return g.Update(i, h, ast.ASTHandle{}, dep)
}

func (p *Pass) rewriteGoal(m *Manager, g *tactic.Goal) (err error) {
	if p.PerAssertion {
		for i := uint32(0); ; i++ {
			size, e := g.Size()
			if e != nil || i >= size {
				return e
			}
			if e = p.rewriteOne(m, g, i); e != nil {
				return e
			}
		}
	}
	cores, err := g.UnsatCoresEnabled()
	if err != nil {
		return err
	}
	if cores {
		return tactic.NewTacticException("aig does not support unsat core production")
	}
	r, err := m.MkAIGGoal(g)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.Close()) }()
	if err = g.Reset(); err != nil {
		return err
	}
	shared, err := m.MaxSharing(r)
	if err != nil {
		return err
	}
	closeErr := r.Close()
	r = shared
	if closeErr != nil {
		return closeErr
	}
	return m.ToGoal(r, g)
}

// Apply rewrites the input goal in place and returns an additional owning
// goal reference, matching goal_ref_buffer::push_back in the source tactic.
func (p *Pass) Apply(g *tactic.Goal) (out *tactic.Goal, err error) {
	defer func() {
		if err != nil && out != nil {
			err = errors.Join(err, out.DecRef())
			out = nil
		}
	}()
	if p == nil || g == nil {
		return nil, ErrOwner
	}
	if err = g.CheckOwner(); err != nil {
		return nil, err
	}
	if p.paramsSymbols != nil && p.paramsSymbols != g.Manager().Symbols {
		return nil, ErrOwner
	}
	proofs, err := g.ProofsEnabled()
	if err != nil {
		return nil, err
	}
	if proofs {
		return nil, tactic.NewTacticException("aig does not support proof production")
	}
	report, err := tactic.NewTacticReport("aig", g)
	if err != nil {
		return nil, err
	}
	if report != nil {
		defer func() { err = errors.Join(err, report.Close()) }()
	}
	m, err := newManagerWithGateEncoding(g.Manager(), p.DefaultGateEncoding, true)
	if err != nil {
		return nil, err
	}
	if err = m.SetMaxMemory(p.maxMemory); err != nil {
		return nil, errors.Join(err, p.closeManager(m))
	}
	if p.cooperation != nil {
		if err = m.BindCooperation(p.cooperation); err != nil {
			return nil, errors.Join(err, p.closeManager(m))
		}
	}
	if err = p.rewriteGoal(m, g); err != nil {
		return nil, errors.Join(err, p.closeManager(m))
	}
	if err = p.closeManager(m); err != nil {
		return nil, err
	}
	if err = g.IncDepth(); err != nil {
		return nil, err
	}
	if err = g.IncRef(); err != nil {
		return nil, err
	}
	return g, nil
}
