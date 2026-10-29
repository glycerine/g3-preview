package fp

// Source: src/muz/spacer/spacer_dl_interface.cpp query setup and dispatch.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/spacer"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (e *PropositionalEngine) querySpacer(declaration ast.ASTHandle) (lbool.LBool, error) {
	if e.params == nil || e.globals == nil {
		return lbool.Undef, fmt.Errorf("muz/fp: Spacer requires parameter and global services")
	}
	if e.pdr != nil {
		if err := e.pdr.Close(); err != nil {
			return lbool.Undef, err
		}
		e.pdr = nil
		e.pdrRevision = 0
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
	// The retained Context currently owns one completed proof-obligation
	// search. Rebuild it for a subsequent query until source reset_query is
	// ported; the rule set and AST expressions remain borrowed by the engine.
	if e.spacer != nil {
		if err = e.spacer.Close(); err != nil {
			return lbool.Undef, err
		}
		e.spacer = nil
		e.spacerRevision = 0
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
	maximumLevelName := "spacer.max_level"
	maximumLevels, err := e.globals.Context.GetUIntName(&fixedpoint, &maximumLevelName, ^uint32(0))
	if err != nil {
		return lbool.Undef, err
	}
	context, err := spacer.NewContext(e.manager, nil, maximumContexts, smtcore.SMTSolverConfig{Params: e.params, Globals: e.globals})
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
	if err = context.InitializePredicates(e.set, e.predicateOrder); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	if err = context.SetQuery(declaration); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	e.spacer = context
	e.spacerRevision = e.revision
	return context.Solve(0, maximumLevels)
}
