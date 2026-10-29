package fp

// Source: src/muz/pdr/pdr_dl_interface.cpp query setup and PDR dispatch.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/pdr"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (e *PropositionalEngine) queryPDR(declaration ast.ASTHandle) (lbool.LBool, error) {
	if e.params == nil || e.globals == nil {
		return lbool.Undef, fmt.Errorf("muz/fp: PDR requires parameter and global services")
	}
	if e.spacer != nil {
		if err := e.spacer.Close(); err != nil {
			return lbool.Undef, err
		}
		e.spacer = nil
		e.spacerRevision = 0
	}
	finalized, err := e.set.IsFinalized()
	if err != nil {
		return lbool.Undef, err
	}
	if !finalized {
		valid, x := e.set.Finalize()
		if x != nil {
			return lbool.Undef, x
		}
		if !valid {
			return lbool.Undef, fmt.Errorf("muz/fp: negation is not stratified")
		}
	}
	if err = e.normalizeSimpleJoins(); err != nil {
		return lbool.Undef, err
	}
	if err = e.validateFragment(); err != nil {
		return lbool.Undef, err
	}
	if e.pdr != nil && e.pdrRevision == e.revision {
		return e.pdr.Solve(declaration)
	}
	if e.pdr != nil {
		if err = e.pdr.Close(); err != nil {
			return lbool.Undef, err
		}
		e.pdr = nil
		e.pdrRevision = 0
	}
	fixedpoint, err := e.globals.SourceGetModuleName("fixedpoint")
	if err != nil {
		return lbool.Undef, err
	}
	maximumName := "pdr.max_num_contexts"
	maximumContexts, err := e.globals.Context.GetUIntName(&fixedpoint, &maximumName, 500)
	if err != nil {
		return lbool.Undef, err
	}
	cacheName := "pdr.cache_mode"
	cacheValue, err := e.globals.Context.GetUIntName(&fixedpoint, &cacheName, uint32(pdr.ReachableNoCache))
	if err != nil {
		return lbool.Undef, err
	}
	cacheMode := pdr.ReachableCacheMode(cacheValue)
	if cacheMode > pdr.ReachableConstraintCache {
		return lbool.Undef, fmt.Errorf("muz/fp: invalid pdr.cache_mode %d", cacheValue)
	}
	context, err := pdr.NewContext(e.manager, nil, maximumContexts, cacheMode, smtcore.SMTSolverConfig{Params: e.params, Globals: e.globals})
	if err != nil {
		return lbool.Undef, err
	}
	farkasName := "pdr.farkas"
	useFarkas, err := e.globals.Context.GetBoolName(&fixedpoint, &farkasName, true)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetUseFarkas(useFarkas); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	inductiveName := "pdr.use_inductive_generalizer"
	useInductive, err := e.globals.Context.GetBoolName(&fixedpoint, &inductiveName, true)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetUseInductiveGeneralizer(useInductive); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	modelGeneralizerName := "pdr.use_model_generalizer"
	useModelGeneralizer, err := e.globals.Context.GetBoolName(&fixedpoint, &modelGeneralizerName, false)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetUseModelGeneralizer(useModelGeneralizer); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	breadthFirstName := "pdr.bfs_model_search"
	breadthFirst, err := e.globals.Context.GetBoolName(&fixedpoint, &breadthFirstName, true)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetBreadthFirstModelSearch(breadthFirst); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	flexibleName := "pdr.flexible_trace"
	flexible, err := e.globals.Context.GetBoolName(&fixedpoint, &flexibleName, false)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetFlexibleTrace(flexible); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	auxiliary := make([]ast.ASTHandle, 0, len(e.auxiliary))
	for declaration := range e.auxiliary {
		auxiliary = append(auxiliary, declaration)
	}
	if err = context.SetAuxiliaryPredicates(auxiliary); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	background, err := e.manager.RefGet(e.background)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetBackground(background); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.Initialize(e.set); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	e.pdr = context
	e.pdrRevision = e.revision
	return context.Solve(declaration)
}

func (e *PropositionalEngine) pdrContext() (*pdr.Context, error) {
	if err := e.open(); err != nil {
		return nil, err
	}
	if e.backend != "pdr" || e.pdr == nil || e.pdrRevision != e.revision {
		return nil, ErrPropositionalEngine
	}
	return e.pdr, nil
}

// NumLevels returns the finite PDR frame count for a relation after a query.
func (e *PropositionalEngine) NumLevels(declaration ast.ASTHandle) (uint32, error) {
	if err := e.open(); err != nil {
		return 0, err
	}
	if e.backend == "spacer" && e.spacer != nil && e.spacerRevision == e.revision {
		return e.spacer.NumLevels(declaration)
	}
	context, err := e.pdrContext()
	if err != nil {
		return 0, err
	}
	return context.NumLevels(declaration)
}

// CoverDelta returns an owning reference to a PDR or Spacer cover frame after
// a query.
func (e *PropositionalEngine) CoverDelta(level int, declaration ast.ASTHandle) (ast.ASTRef, error) {
	if err := e.open(); err != nil {
		return ast.ASTRef{}, err
	}
	if e.backend == "spacer" && e.spacer != nil && e.spacerRevision == e.revision {
		return e.spacer.CoverDelta(level, declaration)
	}
	context, err := e.pdrContext()
	if err != nil {
		return ast.ASTRef{}, err
	}
	return context.CoverDelta(level, declaration)
}

// AddCover installs a bound-variable property in a retained PDR or Spacer
// frame.
func (e *PropositionalEngine) AddCover(level int, declaration, property ast.ASTHandle) error {
	if err := e.open(); err != nil {
		return err
	}
	if e.backend == "spacer" && e.spacer != nil && e.spacerRevision == e.revision {
		return e.spacer.AddCover(level, declaration, property)
	}
	context, err := e.pdrContext()
	if err != nil {
		return err
	}
	return context.AddCover(level, declaration, property)
}

// Reachable returns the accumulated Spacer reach summary for a predicate.
func (e *PropositionalEngine) Reachable(declaration ast.ASTHandle) (ast.ASTRef, error) {
	if err := e.open(); err != nil {
		return ast.ASTRef{}, err
	}
	if e.backend != "spacer" || e.spacer == nil || e.spacerRevision != e.revision {
		return ast.ASTRef{}, ErrPropositionalEngine
	}
	return e.spacer.Reachable(declaration)
}
