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
	if e.pdr != nil {
		if err = e.pdr.Close(); err != nil {
			return lbool.Undef, err
		}
		e.pdr = nil
	}
	context, err := pdr.NewContext(e.manager, nil, 4, pdr.ReachableHashCache, smtcore.SMTSolverConfig{Params: e.params, Globals: e.globals})
	if err != nil {
		return lbool.Undef, err
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
	return context.Solve(declaration)
}
