// Source: src/math/subpaving/tactic/subpaving_tactic.cpp:193-263;
// concrete MPFF core tactic, excluding the public simplify composition.
package tactic

import (
	"errors"
	"io"
	"os"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

var ErrFastCorePrintNodes = errors.New("subpaving tactic: SMT2 display for this mapped expression is not yet ported")

// FastCoreTactic owns the selected engine, expression converter and a shared
// parameter snapshot. The goal, managers and parameter service are borrowed.
// The returned goal is borrowed.
type FastCoreTactic struct {
	self      *FastCoreTactic
	manager   *ast.ASTManager
	numbers   *numeral.FastFloatManager
	rational  *numeral.RationalManager
	pc        *params.ParamsContext
	params    params.ParamsRef
	engine    *FastEngine
	converter *FastExprConverter
	stats     statistics.Statistics
	closed    bool
}

func NewFastCoreTactic(m *ast.ASTManager, numbers *numeral.FastFloatManager, rational *numeral.RationalManager, pc *params.ParamsContext, p *params.ParamsRef) (*FastCoreTactic, error) {
	if m == nil || numbers == nil || rational == nil {
		return nil, arena.ErrInvalidSlot
	}
	if pc == nil || p == nil {
		return nil, params.ErrParamsOwner
	}
	astNumerals := m.NumeralService()
	parameterNumerals := pc.Numerals()
	if astNumerals == nil || parameterNumerals == nil {
		return nil, arena.ErrInvalidSlot
	}
	owners := []*memory.Runtime{numbers.RuntimeOwner(), rational.RuntimeOwner(), astNumerals.RuntimeOwner(), parameterNumerals.RuntimeOwner()}
	for _, owner := range owners {
		if owner == nil {
			return nil, arena.ErrInvalidSlot
		}
	}
	process := owners[0].ProcessOwner()
	for _, owner := range owners[1:] {
		if other := owner.ProcessOwner(); (process != nil || other != nil) && process != other {
			return nil, arena.ErrConfig
		}
	}
	owned, err := pc.NewValue()
	if err != nil {
		return nil, err
	}
	if err := pc.Assign(&owned, p); err != nil {
		return nil, errors.Join(err, pc.Del(&owned))
	}
	t := &FastCoreTactic{manager: m, numbers: numbers, rational: rational, pc: pc, params: owned}
	t.self = t
	if err := t.rebuild(); err != nil {
		return nil, errors.Join(err, t.Close())
	}
	return t, nil
}

func (t *FastCoreTactic) rebuild() error {
	engine, err := NewFastEngine(t.manager.ResourceLimit(), t.numbers, t.rational, t.pc, &t.params)
	if err != nil {
		if engine != nil {
			err = errors.Join(err, engine.Close())
		}
		return err
	}
	converter, err := NewFastExprConverter(t.manager, engine, t.rational)
	if err != nil {
		return errors.Join(err, engine.Close())
	}
	t.engine, t.converter = engine, converter
	return nil
}

func (t *FastCoreTactic) open() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return arena.ErrClosed
	}
	if t.converter == nil || t.engine == nil {
		return arena.ErrInvalidSlot
	}
	return t.converter.check()
}

// Apply internalizes and searches the input in place. As in the source core
// tactic, a successful result contains exactly the original goal; it creates
// no model, proof, or unsat-core converter.
func (t *FastCoreTactic) Apply(g *goaltactic.Goal) (*goaltactic.Goal, error) {
	return t.ApplyWithOutput(g, os.Stdout)
}

// ApplyWithOutput routes the source optional print_nodes output to a caller
// writer. Nil is valid only when print_nodes is disabled.
func (t *FastCoreTactic) ApplyWithOutput(g *goaltactic.Goal, out io.Writer) (*goaltactic.Goal, error) {
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
	if options.PrintNodes && out == nil {
		return nil, arena.ErrInvalidSlot
	}
	if err := t.converter.InternalizeGoal(g); err != nil {
		return nil, err
	}
	shell, err := t.engine.Shell()
	if err != nil {
		return nil, err
	}
	display, err := t.converter.VariableDisplay(t.pc.SymbolsField)
	if err != nil {
		return nil, err
	}
	if err := shell.SetVariableDisplay(display); err != nil {
		return nil, err
	}
	if err := shell.Search(); err != nil {
		return nil, err
	}
	if options.PrintNodes {
		if err := shell.DisplayConstraints(out, true); err != nil {
			return nil, err
		}
		if err := fastWriteDisplay(out, "bounds at leaves: \n"); err != nil {
			return nil, err
		}
		if err := shell.DisplayBounds(out); err != nil {
			return nil, err
		}
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

// UpdateParams retains the caller's parameter value before updating the live
// selected MPFF context. Other numeral engines require their own concrete
// tactic owners and leave this MPFF owner unchanged.
func (t *FastCoreTactic) UpdateParams(p *params.ParamsRef) error {
	if err := t.open(); err != nil {
		return err
	}
	selected, err := ReadOptions(t.pc, p)
	if err != nil {
		return err
	}
	if selected.Engine != EngineMPFF {
		return ErrFastEngineSelection
	}
	if err := t.pc.Assign(&t.params, p); err != nil {
		return err
	}
	return t.engine.UpdateParamsRef(t.pc, &t.params)
}

// Cleanup releases the old converter before its context, then constructs a
// fresh implementation from the tactic's retained parameter snapshot.
// Accumulated outer statistics survive, as in the source tactic.
func (t *FastCoreTactic) Cleanup() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return arena.ErrClosed
	}
	if t.converter != nil {
		if err := t.converter.Close(); err != nil {
			return err
		}
		t.converter = nil
	}
	if t.engine != nil {
		if err := t.engine.Close(); err != nil {
			return err
		}
		t.engine = nil
	}
	return t.rebuild()
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
	if t.engine != nil {
		if err := t.engine.Close(); err != nil {
			return err
		}
		t.engine = nil
	}
	if err := t.pc.Del(&t.params); err != nil {
		return err
	}
	t.closed = true
	return nil
}
