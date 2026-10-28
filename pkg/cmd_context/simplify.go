package cmd_context

// Source: src/cmd_context/simplify_cmd.cpp default term-rewrite execution.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/stopwatch"
)

type SimplifyDetails struct {
	Seconds                                                float64
	Steps, CacheSize, NodesBefore, SharedNodes, NodesAfter uint32
	MemoryBytes, MaximumMemoryBytes                        uint64
}

// Simplify rewrites one term with command-local theory-rewriter parameters.
// The returned reference is owned by the caller.
func (c *Context) Simplify(expression ast.ASTHandle, params *uparams.ParamsRef) (result ast.ASTRef, resultErr error) {
	result, _, resultErr = c.SimplifyWithProof(expression, params, false)
	return result, resultErr
}

// SimplifyWithProof optionally returns the rewrite proof in a second caller-
// owned reference. Proof production requires a proof-enabled AST manager.
func (c *Context) SimplifyWithProof(expression ast.ASTHandle, params *uparams.ParamsRef, produceProof bool) (result, proof ast.ASTRef, resultErr error) {
	result, proof, _, resultErr = c.SimplifyWithDetails(expression, params, produceProof, false)
	return result, proof, resultErr
}

// SimplifyWithDetails optionally captures the source command's statistics
// fields while the simplified term remains rooted for node accounting.
func (c *Context) SimplifyWithDetails(expression ast.ASTHandle, params *uparams.ParamsRef, produceProof, collectStatistics bool) (result, proof ast.ASTRef, details SimplifyDetails, resultErr error) {
	if err := c.valid(); err != nil {
		return result, proof, details, err
	}
	m, err := c.Manager()
	if err != nil {
		return result, proof, details, err
	}
	if collectStatistics {
		details.NodesBefore, err = m.GetNumExprs(expression)
		if err != nil {
			return result, proof, details, err
		}
	}
	value, err := rewriter.NewTheoryRewriter(m, c.config.Params, c.config.Globals, params)
	if err != nil {
		if value != nil {
			err = errors.Join(err, value.Close())
		}
		return result, proof, details, err
	}
	defer func() { resultErr = errors.Join(resultErr, value.Close()) }()
	result, err = m.NewRef(ast.ASTHandle{})
	if err != nil {
		return result, proof, details, err
	}
	published := false
	defer func() {
		if !published {
			if proof != (ast.ASTRef{}) {
				resultErr = errors.Join(resultErr, m.DelRef(&proof))
			}
			resultErr = errors.Join(resultErr, m.DelRef(&result))
		}
	}()
	if produceProof {
		proof, err = m.NewRef(ast.ASTHandle{})
		if err != nil {
			return result, proof, details, err
		}
	}
	watch := stopwatch.New(nil)
	if err = watch.Start(); err != nil {
		return result, proof, details, err
	}
	if produceProof {
		err = value.RewriteWithProof(expression, result, proof)
	} else {
		err = value.Rewrite(expression, result)
	}
	err = errors.Join(err, watch.Stop())
	if err != nil {
		return result, proof, details, err
	}
	if collectStatistics {
		if details.Seconds, err = watch.Seconds(); err != nil {
			return result, proof, details, err
		}
		if details.Steps, err = value.NumSteps(); err != nil {
			return result, proof, details, err
		}
		if details.CacheSize, err = value.CacheSize(); err != nil {
			return result, proof, details, err
		}
		h, e := m.RefGet(result)
		if e != nil {
			return result, proof, details, e
		}
		if details.NodesAfter, e = m.GetNumExprs(h); e != nil {
			return result, proof, details, e
		}
		shared, e := ast.NewASTSharedOccurrences(m)
		if e != nil {
			return result, proof, details, e
		}
		if e = shared.Collect(h); e == nil {
			details.SharedNodes = shared.NumShared()
		}
		e = errors.Join(e, shared.Close())
		if e != nil {
			return result, proof, details, e
		}
		runtime := c.config.Params.NumeralsField.RuntimeOwner()
		if details.MemoryBytes, err = runtime.AllocationSize(); err != nil {
			return result, proof, details, err
		}
		if details.MaximumMemoryBytes, err = runtime.MaximumUsed(); err != nil {
			return result, proof, details, err
		}
	}
	published = true
	return result, proof, details, nil
}
