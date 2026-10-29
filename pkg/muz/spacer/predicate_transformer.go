package spacer

// Source: the initialized solver, lemma/frame, formula, and invariant paths in
// src/muz/spacer/spacer_context.cpp pred_transformer.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/statistics"
)

type PredicateTransformer struct {
	owner        *Manager
	manager      *ast.ASTManager
	solver       *PropSolver
	frames       *Frames
	initial      ast.ASTRef
	transition   ast.ASTRef
	propagations uint64
	invariants   uint64
	closed       bool
}

// NewPredicateTransformer creates the live solver/frame bridge after rule
// conversion has produced the initial-state and transition formulas.
func NewPredicateTransformer(owner *Manager, name string, initial, transition ast.ASTHandle, config PropSolverConfig) (_ *PredicateTransformer, resultErr error) {
	if owner == nil || owner.open() != nil {
		return nil, ErrManagerOwner
	}
	p := &PredicateTransformer{owner: owner, manager: owner.manager}
	if config.Itp.Params == nil && config.Itp.Globals == nil {
		config.Itp.Params = owner.contexts[0].config.Params
		config.Itp.Globals = owner.contexts[0].config.Globals
	}
	committed := false
	defer func() {
		if !committed {
			resultErr = errors.Join(resultErr, p.Close())
		}
	}()
	var err error
	if p.initial, err = p.manager.NewRef(initial); err != nil {
		return nil, err
	}
	if p.transition, err = p.manager.NewRef(transition); err != nil {
		return nil, err
	}
	if p.solver, err = owner.NewPropSolver(name, config); err != nil {
		return nil, err
	}
	if p.frames, err = NewFrames(p.manager, p); err != nil {
		return nil, err
	}
	if err = p.solver.Assert(transition); err != nil {
		return nil, err
	}
	if err = p.solver.AssertAtLevel(initial, 0); err != nil {
		return nil, err
	}
	if err = p.frames.AddFrame(); err != nil {
		return nil, err
	}
	committed = true
	return p, nil
}

func (p *PredicateTransformer) open() error {
	if p == nil || p.closed || p.owner == nil || p.manager == nil || p.solver == nil || p.frames == nil {
		return ErrManagerOwner
	}
	return nil
}

func (p *PredicateTransformer) EnsureLevel(level uint32) error {
	if err := p.open(); err != nil {
		return err
	}
	if level == InfinityLevel {
		return nil
	}
	size, err := p.frames.Size()
	if err != nil {
		return err
	}
	for size <= level {
		if err = p.frames.AddFrame(); err != nil {
			return err
		}
		if err = p.solver.EnsureLevel(size); err != nil {
			return err
		}
		size++
	}
	return nil
}

func (p *PredicateTransformer) AddLemmaCore(lemma *Lemma) error {
	if err := p.open(); err != nil {
		return err
	}
	if lemma == nil || lemma.manager != p.manager {
		return ErrManagerOwner
	}
	body, err := lemma.Body()
	if err != nil {
		return err
	}
	ground, err := p.manager.IsGround(body)
	if err != nil {
		return err
	}
	level, err := lemma.Level()
	if err != nil {
		return err
	}
	if level == InfinityLevel {
		p.invariants++
	}
	if !ground {
		return nil
	}
	if level == InfinityLevel {
		return p.solver.Assert(body)
	}
	if err = p.EnsureLevel(level); err != nil {
		return err
	}
	return p.solver.AssertAtLevel(body, level)
}

func (p *PredicateTransformer) IsInvariant(level uint32, formula ast.ASTHandle) (bool, uint32, error) {
	if err := p.open(); err != nil {
		return false, 0, err
	}
	negated, err := p.manager.MkNotSimplified(formula)
	if err != nil {
		return false, 0, err
	}
	result, err := p.solver.CheckAssumptions([]ast.ASTHandle{negated}, nil, PropCheckOptions{
		SubsetBasedCore: true,
		Level:           &level,
	})
	if err != nil {
		return false, 0, err
	}
	defer func() { _ = result.Close(p.manager) }()
	if result.Status != lbool.False {
		return false, 0, nil
	}
	return true, result.UsesLevel, nil
}

// AddLemma transfers ownership of the new lemma to the transformer's frames.
func (p *PredicateTransformer) AddLemma(formula ast.ASTHandle, level uint32) (bool, error) {
	if err := p.open(); err != nil {
		return false, err
	}
	lemma, err := NewLemma(p.owner, formula, level)
	if err != nil {
		return false, err
	}
	added, err := p.frames.AddLemma(lemma)
	if err != nil {
		_ = lemma.Close()
	}
	return added, err
}

func (p *PredicateTransformer) PropagateToNextLevel(level uint32) (bool, error) {
	if err := p.open(); err != nil {
		return false, err
	}
	p.propagations++
	return p.frames.PropagateToNextLevel(level)
}

func (p *PredicateTransformer) Formulas(level uint32, addAxioms bool) (ast.ASTHandle, error) {
	if err := p.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	var formulas []ast.ASTHandle
	if addAxioms {
		background, err := p.owner.Background()
		if err != nil {
			return ast.ASTHandle{}, err
		}
		formulas = append(formulas, background)
		var state ast.ASTHandle
		if level == 0 {
			state, err = p.manager.RefGet(p.initial)
		} else {
			state, err = p.manager.RefGet(p.transition)
		}
		if err != nil {
			return ast.ASTHandle{}, err
		}
		formulas = append(formulas, state)
	}
	lemmas, err := p.frames.FrameGEQLemmas(level)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	formulas = append(formulas, lemmas...)
	return p.manager.MkAndArray(formulas)
}

func (p *PredicateTransformer) CollectStatistics(destination *statistics.Statistics) error {
	if err := p.open(); err != nil {
		return err
	}
	if destination == nil {
		return ast.ErrASTHandle
	}
	if err := p.solver.CollectStatistics(destination); err != nil {
		return err
	}
	lemmas, err := p.frames.LemmaSize()
	if err != nil {
		return err
	}
	destination.UpdateUint64("SPACER num propagations", p.propagations)
	destination.UpdateUint("SPACER num properties", uint32(lemmas))
	destination.UpdateUint64("SPACER num invariants", p.invariants)
	return nil
}

func (p *PredicateTransformer) Close() (result error) {
	if p == nil {
		return ErrManagerOwner
	}
	if p.closed {
		return nil
	}
	if p.frames != nil {
		result = errors.Join(result, p.frames.Close())
	}
	if p.solver != nil {
		result = errors.Join(result, p.solver.Close())
	}
	if p.manager != nil {
		if p.transition != (ast.ASTRef{}) {
			result = errors.Join(result, p.manager.DelRef(&p.transition))
		}
		if p.initial != (ast.ASTRef{}) {
			result = errors.Join(result, p.manager.DelRef(&p.initial))
		}
	}
	if result == nil {
		p.frames = nil
		p.solver = nil
		p.owner = nil
		p.manager = nil
		p.closed = true
	}
	return result
}
