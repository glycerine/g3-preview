package fp

// Source: src/muz/spacer/spacer_dl_interface.cpp query setup and dispatch.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/spacer"
	"github.com/glycerine/g3/pkg/muz/transforms"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (e *PropositionalEngine) preprocessHornRules(query ast.ASTHandle) (_ *base.RuleSet, resultErr error) {
	prepared, err := base.NewRuleSet(e.rules)
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			resultErr = errors.Join(resultErr, prepared.Close())
		}
	}()
	if err = prepared.AddRules(e.set); err != nil {
		return nil, err
	}
	valid, err := prepared.Finalize()
	if err != nil || !valid {
		if err == nil {
			err = fmt.Errorf("muz/fp: negation is not stratified")
		}
		return nil, err
	}
	transformer := base.NewRuleTransformer()
	firstSimplifier, err := transforms.NewInterpTailSimplifierPlugin(e.manager, e.rules, e.params, e.globals, 40000)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(firstSimplifier); err != nil {
		return nil, errors.Join(err, firstSimplifier.Close(), transformer.Close())
	}
	if err = transformer.Register(transforms.NewRuleInlinerPlugin(e.manager, e.rules, []ast.ASTHandle{query}, 35000)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	secondSimplifier, err := transforms.NewInterpTailSimplifierPlugin(e.manager, e.rules, e.params, e.globals, 30000)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(secondSimplifier); err != nil {
		return nil, errors.Join(err, secondSimplifier.Close(), transformer.Close())
	}
	if err = transformer.Register(transforms.NewRuleInlinerPlugin(e.manager, e.rules, []ast.ASTHandle{query}, 25000)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	_, err = transformer.Apply(prepared)
	err = errors.Join(err, transformer.Close())
	if err != nil {
		return nil, fmt.Errorf("muz/fp: Spacer rule preprocessing: %w", err)
	}
	keep = true
	return prepared, nil
}

func (e *PropositionalEngine) querySpacer(declaration ast.ASTHandle) (_ lbool.LBool, resultErr error) {
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
	prepared, err := e.preprocessHornRules(declaration)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { resultErr = errors.Join(resultErr, prepared.Close()) }()
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
	if err = context.InitializePredicates(prepared, e.predicateOrder); err != nil {
		return lbool.Undef, errors.Join(fmt.Errorf("muz/fp: initialize preprocessed Spacer rules: %w", err), context.Close())
	}
	if err = context.SetQuery(declaration); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	e.spacer = context
	e.spacerRevision = e.revision
	return context.Solve(0, maximumLevels)
}
