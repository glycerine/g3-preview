// Source: src/tactic/aig/aig_tactic.cpp:65-113.
package aig

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

// Pass applies the source AIG tactic's two goal-rewriting branches. Its
// resource-limit parameters are kept separate until the source checkpoints
// in aig.cpp are ported.
type Pass struct {
	DefaultGateEncoding bool
	PerAssertion        bool
}

func NewPass() *Pass { return &Pass{DefaultGateEncoding: true, PerAssertion: true} }

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
	m, err := NewManagerWithGateEncoding(g.Manager(), p.DefaultGateEncoding)
	if err != nil {
		return nil, err
	}
	if err = p.rewriteGoal(m, g); err != nil {
		return nil, errors.Join(err, m.Close())
	}
	if err = m.Close(); err != nil {
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
