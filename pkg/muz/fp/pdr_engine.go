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
	context, err := pdr.NewContext(e.manager, nil, 4, pdr.ReachableHashCache, smtcore.SMTSolverConfig{Params: e.params, Globals: e.globals})
	if err != nil {
		return lbool.Undef, err
	}
	fixedpoint, err := e.globals.SourceGetModuleName("fixedpoint")
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	farkasName := "pdr.farkas"
	useFarkas, err := e.globals.Context.GetBoolName(&fixedpoint, &farkasName, true)
	if err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetUseFarkas(useFarkas); err != nil {
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
	context, err := e.pdrContext()
	if err != nil {
		return 0, err
	}
	return context.NumLevels(declaration)
}

// CoverDelta returns an owning reference to a PDR cover frame after a query.
func (e *PropositionalEngine) CoverDelta(level int, declaration ast.ASTHandle) (ast.ASTRef, error) {
	context, err := e.pdrContext()
	if err != nil {
		return ast.ASTRef{}, err
	}
	return context.CoverDelta(level, declaration)
}

// AddCover installs a bound-variable property in a retained PDR frame.
func (e *PropositionalEngine) AddCover(level int, declaration, property ast.ASTHandle) error {
	context, err := e.pdrContext()
	if err != nil {
		return err
	}
	return context.AddCover(level, declaration, property)
}
