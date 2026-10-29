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
	var slicedPredicates map[ast.ASTHandle]ast.ASTHandle
	var slicedColumns map[ast.ASTHandle][]bool
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
	// rule_manager::mk_query registers the synthetic/direct query predicate as
	// an output before apply_default_transformation runs.
	if err = prepared.SetOutputPredicate(query); err != nil {
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
	if err = transformer.Register(transforms.NewConePlugin(0)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	firstSimplifier, err := transforms.NewInterpTailSimplifierPlugin(e.manager, e.rules, e.params, e.globals, 40000)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(firstSimplifier); err != nil {
		return nil, errors.Join(err, firstSimplifier.Close(), transformer.Close())
	}
	fixedpoint, err := e.globals.SourceGetModuleName("fixedpoint")
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	subsumptionName := "datalog.subsumption"
	useSubsumption, err := e.params.GetBoolName(&fixedpoint, &subsumptionName, true)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	checkerName := "xform.subsumption_checker"
	useChecker, err := e.params.GetBoolName(&fixedpoint, &checkerName, true)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	inlineEagerName := "xform.inline_eager"
	inlineEager, err := e.params.GetBoolName(&fixedpoint, &inlineEagerName, true)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	inlineLinearName := "xform.inline_linear"
	inlineLinear, err := e.params.GetBoolName(&fixedpoint, &inlineLinearName, true)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	unfoldName := "xform.unfold_rules"
	unfoldRounds, err := e.params.GetUIntName(&fixedpoint, &unfoldName, 0)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	coalesceName := "xform.coalesce_rules"
	coalesceRules, err := e.params.GetBoolName(&fixedpoint, &coalesceName, false)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	sliceName := "xform.slice"
	sliceRules, err := e.params.GetBoolName(&fixedpoint, &sliceName, true)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	transformArraysName := "xform.transform_arrays"
	transformArrays, err := e.params.GetBoolName(&fixedpoint, &transformArraysName, false)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	instantiateQuantifiersName := "xform.instantiate_quantifiers"
	instantiateQuantifiers, err := e.params.GetBoolName(&fixedpoint, &instantiateQuantifiersName, false)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	quantifyArraysName := "xform.quantify_arrays"
	quantifyArrays, err := e.params.GetBoolName(&fixedpoint, &quantifyArraysName, false)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	newInliner := func(priority uint32) base.RuleTransformPlugin {
		return transforms.NewRuleInlinerPluginWithServices(e.manager, e.rules, []ast.ASTHandle{query}, inlineEager, inlineLinear, e.params, e.globals, priority)
	}
	if useSubsumption {
		if err = transformer.Register(transforms.NewSubsumptionCheckerPlugin(useChecker, 35005)); err != nil {
			return nil, errors.Join(err, transformer.Close())
		}
	}
	if instantiateQuantifiers {
		if err = transformer.Register(transforms.NewQuantifierInstantiationPlugin(e.manager, e.rules, 37000)); err != nil {
			return nil, errors.Join(err, transformer.Close())
		}
	}
	var quantifierAbstraction *transforms.QuantifierAbstractionPlugin
	var quantifiedPredicates map[ast.ASTHandle]ast.ASTHandle
	if quantifyArrays {
		quantifierAbstraction = transforms.NewQuantifierAbstractionPlugin(e.manager, e.rules, 38000)
		if err = transformer.Register(quantifierAbstraction); err != nil {
			return nil, errors.Join(err, transformer.Close())
		}
	}
	if err = transformer.Register(newInliner(35000)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if transformArrays {
		arrayRewrite, makeErr := transforms.NewArrayEqRewritePlugin(e.manager, e.rules, e.params, e.globals, 34998)
		if makeErr != nil {
			return nil, errors.Join(makeErr, transformer.Close())
		}
		if err = transformer.Register(arrayRewrite); err != nil {
			return nil, errors.Join(err, arrayRewrite.Close(), transformer.Close())
		}
	}
	if err = transformer.Register(transforms.NewConePlugin(34990)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	secondSimplifier, err := transforms.NewInterpTailSimplifierPlugin(e.manager, e.rules, e.params, e.globals, 34980)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(secondSimplifier); err != nil {
		return nil, errors.Join(err, secondSimplifier.Close(), transformer.Close())
	}
	if useSubsumption {
		if err = transformer.Register(transforms.NewSubsumptionCheckerPlugin(useChecker, 34975)); err != nil {
			return nil, errors.Join(err, transformer.Close())
		}
	}
	if err = transformer.Register(newInliner(34970)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(transforms.NewConePlugin(34960)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	thirdSimplifier, err := transforms.NewInterpTailSimplifierPlugin(e.manager, e.rules, e.params, e.globals, 34950)
	if err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	if err = transformer.Register(thirdSimplifier); err != nil {
		return nil, errors.Join(err, thirdSimplifier.Close(), transformer.Close())
	}
	if useSubsumption {
		for _, stage := range []struct {
			priority uint32
			inlining bool
		}{
			{priority: 34940},
			{priority: 34930, inlining: true},
			{priority: 34920},
			{priority: 34910, inlining: true},
			{priority: 34900},
			{priority: 34890, inlining: true},
			{priority: 34880},
		} {
			var plugin base.RuleTransformPlugin
			if stage.inlining {
				plugin = newInliner(stage.priority)
			} else {
				plugin = transforms.NewSubsumptionCheckerPlugin(useChecker, stage.priority)
			}
			if err = transformer.Register(plugin); err != nil {
				return nil, errors.Join(err, plugin.Close(), transformer.Close())
			}
		}
	} else if err = transformer.Register(newInliner(34930)); err != nil {
		return nil, errors.Join(err, transformer.Close())
	}
	_, err = transformer.Apply(prepared)
	if err == nil && quantifierAbstraction != nil {
		quantifiedPredicates = quantifierAbstraction.FreshPredicates()
		for _, declaration := range quantifiedPredicates {
			state, stateErr := e.predicateState(declaration)
			if stateErr != nil {
				err = stateErr
				break
			}
			e.predicates[declaration] = state
		}
	}
	err = errors.Join(err, transformer.Close())
	if err != nil {
		return nil, fmt.Errorf("muz/fp: Spacer rule preprocessing: %w", err)
	}
	if sliceRules {
		slicer := base.NewRuleTransformer()
		plugin := transforms.NewRuleSlicePlugin(e.manager, e.rules, 1)
		if err = slicer.Register(plugin); err != nil {
			return nil, errors.Join(err, plugin.Close(), slicer.Close())
		}
		_, err = slicer.Apply(prepared)
		if err == nil {
			slicedPredicates, slicedColumns = plugin.PredicateSlices()
		}
		err = errors.Join(err, slicer.Close())
		if err != nil {
			return nil, fmt.Errorf("muz/fp: Spacer rule slicing: %w", err)
		}
	}
	if unfoldRounds > 0 && coalesceRules {
		coalescer := base.NewRuleTransformer()
		plugin := transforms.NewRuleCoalescePlugin(e.manager, e.rules, 50)
		if err = coalescer.Register(plugin); err != nil {
			return nil, errors.Join(err, plugin.Close(), coalescer.Close())
		}
		_, err = coalescer.Apply(prepared)
		err = errors.Join(err, coalescer.Close())
		if err != nil {
			return nil, fmt.Errorf("muz/fp: Spacer rule coalescing: %w", err)
		}
	}
	for round := uint32(0); round < unfoldRounds; round++ {
		unfolder := base.NewRuleTransformer()
		plugin := transforms.NewRuleUnfoldPlugin(e.manager, e.rules, e.params, e.globals, 100)
		if err = unfolder.Register(plugin); err != nil {
			return nil, errors.Join(err, plugin.Close(), unfolder.Close())
		}
		_, err = unfolder.Apply(prepared)
		err = errors.Join(err, unfolder.Close())
		if err != nil {
			return nil, fmt.Errorf("muz/fp: Spacer rule unfolding round %d: %w", round+1, err)
		}
	}
	e.spacerPredicates = slicedPredicates
	e.spacerSlices = slicedColumns
	e.spacerQuantified = quantifiedPredicates
	keep = true
	return prepared, nil
}

func (e *PropositionalEngine) querySpacer(declaration ast.ASTHandle) (_ lbool.LBool, resultErr error) {
	return e.querySpacerFromLevel(declaration, 0)
}

func (e *PropositionalEngine) querySpacerFromLevel(declaration ast.ASTHandle, fromLevel uint32) (_ lbool.LBool, resultErr error) {
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
	queryDeclaration, err := prepared.DerivedPredicate(declaration)
	if err != nil {
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
	if err = context.InitializePredicates(prepared, e.predicateOrder); err != nil {
		return lbool.Undef, errors.Join(fmt.Errorf("muz/fp: initialize preprocessed Spacer rules: %w", err), context.Close())
	}
	if err = context.SetQuery(queryDeclaration); err != nil {
		return lbool.Undef, errors.Join(err, context.Close())
	}
	e.spacer = context
	e.spacerRevision = e.revision
	return context.Solve(fromLevel, maximumLevels)
}
