// Source: src/ast/pattern/pattern_inference.h:253-259 and
// src/ast/pattern/pattern_inference.cpp:722-725.
package pattern

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
)

type inferenceConfiguration struct {
	rewriter.DefaultConfiguration
	owner *InferenceRewriter
}

func (c *inferenceConfiguration) ReduceQuantifier(q, body ast.ASTHandle, _ ast.AppArguments, noPatterns ast.AppArguments, out, _ ast.ASTRef) (bool, error) {
	r := c.owner
	if r == nil || r.closed {
		return false, ast.ErrASTHandle
	}
	exclusions := make([]ast.ASTHandle, noPatterns.Count())
	for i := range exclusions {
		h, err := noPatterns.At(uint32(i))
		if err != nil {
			return false, err
		}
		exclusions[i] = h
	}
	result, changed, err := r.candidates.reduceQuantifierNoDatabase(q, body, exclusions, r.options)
	if err != nil || !changed {
		return false, err
	}
	quantifier, err := r.manager.RefGet(result)
	if err == nil {
		err = r.manager.SetRef(out, quantifier)
	}
	if cleanupErr := r.manager.DelRef(&result); cleanupErr != nil {
		if result != (ast.ASTRef{}) {
			r.candidates.pendingPatternRefs = append(r.candidates.pendingPatternRefs, result)
		}
		err = errors.Join(err, cleanupErr)
	}
	return err == nil, err
}

// InferenceRewriter owns the source pattern-inference configuration and its
// AST rewriter traversal. The current no-proof path uses the source defaults.
type InferenceRewriter struct {
	self       *InferenceRewriter
	manager    *ast.ASTManager
	candidates *patternCandidates
	traversal  *rewriter.Traversal
	config     inferenceConfiguration
	options    quantifierInferenceOptions
	closed     bool
}

func NewInferenceRewriter(m *ast.ASTManager) (*InferenceRewriter, error) {
	return NewInferenceRewriterWithOptions(m, DefaultInferenceOptions())
}

func NewInferenceRewriterWithOptions(m *ast.ASTManager, options InferenceOptions) (*InferenceRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	proofs, err := m.ProofsEnabled()
	if err != nil {
		return nil, err
	}
	if proofs {
		return nil, ast.ErrASTPending
	}
	candidates, err := newPatternCandidates(m, 0, nil, options.BlockLoopPatterns)
	if err != nil {
		return nil, err
	}
	candidates.avoidSkolems = options.AvoidSkolems
	r := &InferenceRewriter{
		manager: m, candidates: candidates,
		options: options.internal(),
	}
	r.self = r
	r.config.owner = r
	r.traversal, err = rewriter.NewTraversal(m, &r.config)
	if err != nil {
		return r, err
	}
	return r, nil
}

func (r *InferenceRewriter) UpdateOptions(options InferenceOptions) error {
	if r == nil || r.self != r || r.closed {
		return ast.ErrASTHandle
	}
	r.options = options.internal()
	r.candidates.blockLoops = options.BlockLoopPatterns
	r.candidates.avoidSkolems = options.AvoidSkolems
	return nil
}

func (r *InferenceRewriter) Rewrite(body ast.ASTHandle, out ast.ASTRef) error {
	if r == nil || r.self != r || r.closed || r.traversal == nil {
		return ast.ErrASTHandle
	}
	return r.traversal.Rewrite(body, out)
}

func (r *InferenceRewriter) Resume(out ast.ASTRef) error {
	if r == nil || r.self != r || r.closed || r.traversal == nil {
		return ast.ErrASTHandle
	}
	return r.traversal.Resume(out)
}

func (r *InferenceRewriter) RegisterPreferred(decl ast.ASTHandle) error {
	if r == nil || r.self != r || r.closed {
		return ast.ErrASTHandle
	}
	return r.candidates.registerPreferred(decl)
}

func (r *InferenceRewriter) Close() error {
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
	if err := r.candidates.close(); err != nil {
		return err
	}
	r.closed = true
	return nil
}
