// Source: src/math/subpaving/tactic/subpaving_tactic.cpp:193-263;
// concrete MPFF core tactic, excluding the public simplify composition.
package tactic

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

var ErrFastCorePrintNodes = errors.New("subpaving tactic: MPFF goal variable printer is not yet ported")

// FastCoreTactic owns the selected engine and expression converter. The goal,
// managers, and parameter service are borrowed. The returned goal is borrowed.
type FastCoreTactic struct {
	self      *FastCoreTactic
	manager   *ast.ASTManager
	engine    *FastEngine
	converter *FastExprConverter
	stats     statistics.Statistics
	closed    bool
}

func NewFastCoreTactic(m *ast.ASTManager, numbers *numeral.FastFloatManager, rational *numeral.RationalManager, pc *params.ParamsContext, p *params.ParamsRef) (*FastCoreTactic, error) {
	if m == nil || numbers == nil || rational == nil {
		return nil, arena.ErrInvalidSlot
	}
	engine, err := NewFastEngine(m.ResourceLimit(), numbers, rational, pc, p)
	if err != nil {
		if engine != nil {
			err = errors.Join(err, engine.Close())
		}
		return nil, err
	}
	converter, err := NewFastExprConverter(m, engine, rational)
	if err != nil {
		return nil, errors.Join(err, engine.Close())
	}
	t := &FastCoreTactic{manager: m, engine: engine, converter: converter}
	t.self = t
	return t, nil
}

func (t *FastCoreTactic) open() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return arena.ErrClosed
	}
	return t.converter.check()
}

// Apply internalizes and searches the input in place. As in the source core
// tactic, a successful result contains exactly the original goal; it creates
// no model, proof, or unsat-core converter.
func (t *FastCoreTactic) Apply(g *goaltactic.Goal) (*goaltactic.Goal, error) {
	if err := t.open(); err != nil {
		return nil, err
	}
	if g == nil {
		return nil, arena.ErrInvalidSlot
	}
	if err := g.CheckOwner(); err != nil {
		return nil, err
	}
	if g.Manager() != t.manager {
		return nil, ast.ErrASTHandle
	}
	options, err := t.engine.Options()
	if err != nil {
		return nil, err
	}
	if options.PrintNodes {
		return nil, ErrFastCorePrintNodes
	}
	if err := t.converter.SearchGoal(g); err != nil {
		return nil, err
	}
	shell, err := t.engine.Shell()
	if err != nil {
		return nil, err
	}
	var current statistics.Statistics
	if err := shell.CollectStatistics(&current); err != nil {
		return nil, err
	}
	t.stats.Copy(current)
	return g, nil
}

func (t *FastCoreTactic) CollectStatistics(dst *statistics.Statistics) error {
	if err := t.open(); err != nil {
		return err
	}
	if dst == nil {
		return arena.ErrInvalidSlot
	}
	dst.Copy(t.stats)
	return nil
}

func (t *FastCoreTactic) ResetStatistics() error {
	if err := t.open(); err != nil {
		return err
	}
	t.stats.Reset()
	return nil
}

func (t *FastCoreTactic) Close() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return nil
	}
	if t.converter != nil {
		if err := t.converter.Close(); err != nil {
			return err
		}
		t.converter = nil
	}
	if err := t.engine.Close(); err != nil {
		return err
	}
	t.engine = nil
	t.closed = true
	return nil
}
