// Source: src/smt/smt_kernel.h and src/smt/smt_kernel.cpp retained assertion,
// scope, setup, check, and context-access facade.
package smtcore

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type SMTKernelAssertion struct {
	Formula ast.ASTHandle
	Proof   ast.ASTHandle
}

type SMTKernel struct {
	manager          *ast.ASTManager
	checker          *SMTCheckerContext
	relevancy        *SMTRelevancyPropagator
	internalizer     *SMTInternalizerState
	assertions       []SMTKernelAssertion
	assertedHead     int
	scopes           []int
	unsatCore        []ast.ASTHandle
	unsatProof       ast.ASTRef
	assumptions      map[SMTLiteral]ast.ASTHandle
	lastFailure      SMTFailure
	unknown          string
	labels           []usymbol.Symbol
	assignments      []ast.ASTHandle
	relevantLiterals []ast.ASTHandle
	relevantLabeled  []ast.ASTHandle
	guessed          []ast.ASTHandle
	callback         solver.ProgressCallback
	modelConfig      model.ModelEvaluatorConfig
	generated        *model.Model
	preprocessor     *rewriter.TheoryRewriter
	paramContext     *uparams.ParamsContext
	paramGlobals     *uparams.GlobalParamsService
	smtParams        *SMTParams
	smtState         SMTParamsState
	preprocessState  SMTPreprocessorState
	produceModel     bool
	preprocess       bool
	timerClock       func() (uint64, error)
	closed           bool
}

func (k *SMTKernel) readPreprocessorState(context *uparams.ParamsContext, globals *uparams.GlobalParamsService, params *uparams.ParamsRef) (state SMTPreprocessorState, result error) {
	value, err := NewSMTPreprocessorParams(context, params, globals, k.manager.Budget())
	if err != nil {
		if value != nil {
			result = errors.Join(err, value.Close())
			return state, result
		}
		return state, err
	}
	defer func() { result = errors.Join(result, value.Close()) }()
	return value.State()
}

func NewSMTKernel(manager *ast.ASTManager, quantifiers *SMTQuantifierRegistry) (*SMTKernel, error) {
	checker, err := NewSMTCheckerContext(manager)
	if err != nil {
		return nil, err
	}
	relevancy, err := NewSMTRelevancyPropagator(checker, true)
	if err != nil {
		return nil, err
	}
	internalizer, err := NewSMTInternalizerState(manager, checker, quantifiers)
	if err != nil {
		return nil, err
	}
	proofs, err := manager.ProofsEnabled()
	if err != nil {
		return nil, errors.Join(err, internalizer.Close())
	}
	if !proofs {
		if err = internalizer.SetBinaryClauseOptimization(true); err != nil {
			return nil, errors.Join(err, internalizer.Close())
		}
	}
	if err = internalizer.SetRelevancy(relevancy); err != nil {
		return nil, errors.Join(err, internalizer.Close())
	}
	proof, err := manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return nil, errors.Join(err, internalizer.Close())
	}
	return &SMTKernel{
		manager: manager, checker: checker, relevancy: relevancy, internalizer: internalizer, unsatProof: proof,
		smtState: SMTParamsState{
			MaxConflicts: ^uint32(0), InvDecay: 1.052, RandomInitialActivity: 1, RandomVarFreq: 0.01, CaseSplitStrategy: 1, RestartStrategy: 1, RestartInitial: 100, RestartFactor: 1.1,
			RestartAdaptive: true, AgilityFactor: 0.9999, RestartAgilityThreshold: 0.18, SimplifyClauses: true,
		},
	}, nil
}

// ConfigurePreprocessing installs the source theory-rewrite pass used before
// asserted formulas enter the retained formula stack. It must be configured
// before the first assertion is stored.
func (k *SMTKernel) ConfigurePreprocessing(context *uparams.ParamsContext, globals *uparams.GlobalParamsService, params *uparams.ParamsRef) error {
	if err := k.open(); err != nil {
		return err
	}
	if len(k.assertions) != 0 || k.assertedHead != 0 || len(k.scopes) != 0 || k.preprocessor != nil || k.smtParams != nil {
		return ast.ErrASTDomain
	}
	if context == nil || globals == nil || context.SymbolsField != k.manager.Symbols || globals.Context != context {
		return uparams.ErrParamsOwner
	}
	k.paramContext = context
	k.paramGlobals = globals
	full, err := NewSMTParams(context, params, globals, k.manager.Budget())
	if err != nil {
		if full != nil {
			err = errors.Join(err, full.Close())
		}
		return err
	}
	// smt_solver constructs smt_params and its context first, then invokes the
	// full updt_params overload. That second step updates embedded bases such as
	// dyn_ack_params, whose constructors intentionally saw an empty params_ref.
	if err = full.Update(params); err != nil {
		return errors.Join(err, full.Close())
	}
	fullState, err := full.State()
	if err != nil {
		return errors.Join(err, full.Close())
	}
	dynamic, err := full.DynamicAckState()
	if err != nil {
		return errors.Join(err, full.Close())
	}
	proofs, err := k.manager.ProofsEnabled()
	if err != nil {
		return errors.Join(err, full.Close())
	}
	if err = k.internalizer.SetBinaryClauseOptimization(fullState.BinaryClauseOpt && !proofs); err != nil {
		return errors.Join(err, full.Close())
	}
	if err = k.internalizer.ConfigureDynamicAck(dynamic); err != nil {
		return errors.Join(err, full.Close())
	}
	maintenance := SMTClauseMaintenanceConfig{
		Strategy: fullState.LemmaGcStrategy, Half: fullState.LemmaGcHalf,
		Recent: fullState.RecentLemmasSize, Initial: fullState.LemmaGcInitial,
		Factor: fullState.LemmaGcFactor, InverseDecay: fullState.InvClauseDecay,
		NewOldRatio: fullState.NewOldRatio, NewActivity: fullState.NewClauseActivity,
		OldActivity: fullState.OldClauseActivity, NewRelevancy: fullState.NewClauseRelevancy,
		OldRelevancy: fullState.OldClauseRelevancy,
	}
	if err = k.internalizer.ConfigureClauseMaintenance(maintenance); err != nil {
		return errors.Join(err, full.Close())
	}
	if err = k.internalizer.configureBooleanHeuristics(fullState); err != nil {
		return errors.Join(err, full.Close())
	}
	k.relevancy.enabled = fullState.RelevancyLvl > 0
	k.smtParams = full
	k.smtState = fullState
	state, err := k.readPreprocessorState(context, globals, params)
	if err != nil {
		return err
	}
	k.preprocessState = state
	name := "preprocess"
	enabled, err := context.GetBoolName(params, &name, true)
	if err != nil {
		return err
	}
	if !enabled {
		k.preprocess = false
		return nil
	}
	value, err := rewriter.NewTheoryRewriter(k.manager, context, globals, params)
	if err != nil {
		return err
	}
	k.preprocessor = value
	k.preprocess = true
	return nil
}

// UpdateParams refreshes the assertion rewriter and the hidden preprocessing
// switch using the kernel's borrowed parameter services.
func (k *SMTKernel) UpdateParams(params *uparams.ParamsRef) error {
	if err := k.open(); err != nil {
		return err
	}
	if k.paramContext == nil || k.paramGlobals == nil || k.smtParams == nil {
		return uparams.ErrParamsOwner
	}
	if err := k.smtParams.Update(params); err != nil {
		return err
	}
	paramsState, err := k.smtParams.State()
	if err != nil {
		return err
	}
	dynamic, err := k.smtParams.DynamicAckState()
	if err != nil {
		return err
	}
	if err = k.internalizer.UpdateDynamicAck(dynamic); err != nil {
		return err
	}
	if err = k.internalizer.configureBooleanHeuristics(paramsState); err != nil {
		return err
	}
	k.relevancy.enabled = paramsState.RelevancyLvl > 0
	k.smtState = paramsState
	name := "preprocess"
	enabled, err := k.paramContext.GetBoolName(params, &name, true)
	if err != nil {
		return err
	}
	state, err := k.readPreprocessorState(k.paramContext, k.paramGlobals, params)
	if err != nil {
		return err
	}
	if !enabled {
		if k.preprocessor != nil {
			if err = k.preprocessor.Close(); err != nil {
				return err
			}
			k.preprocessor = nil
		}
		k.preprocess = false
		k.preprocessState = state
		return nil
	}
	if k.preprocessor == nil {
		k.preprocessor, err = rewriter.NewTheoryRewriter(k.manager, k.paramContext, k.paramGlobals, params)
	} else {
		err = k.preprocessor.UpdateParams(params)
	}
	if err == nil {
		k.preprocess = true
		k.preprocessState = state
	}
	return err
}

func (k *SMTKernel) open() error {
	if k == nil || k.manager == nil || k.internalizer == nil || k.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (k *SMTKernel) Internalizer() (*SMTInternalizerState, error) {
	if err := k.open(); err != nil {
		return nil, err
	}
	return k.internalizer, nil
}

func (k *SMTKernel) RegisterTheory(theory SMTTheoryInternalizer) error {
	if err := k.open(); err != nil {
		return err
	}
	return k.internalizer.RegisterTheory(theory)
}

func (k *SMTKernel) Assert(formula, proof ast.ASTHandle) error {
	if err := k.open(); err != nil {
		return err
	}
	sort, err := k.manager.GetSort(formula)
	if err != nil {
		return err
	}
	boolean, err := k.manager.MkBoolSort()
	if err != nil {
		return err
	}
	if sort != boolean {
		return ast.ErrASTDomain
	}
	if !proof.IsNull() {
		if isProof, err := k.manager.IsProof(proof); err != nil || !isProof {
			if err != nil {
				return err
			}
			return ast.ErrASTDomain
		}
	}
	if proof.IsNull() {
		proofs, err := k.manager.ProofsEnabled()
		if err != nil {
			return err
		}
		if proofs {
			proof, err = k.manager.MkProofAsserted(formula)
			if err != nil {
				return err
			}
		}
	}
	formula, proof, err = k.preprocessAssertion(formula, proof)
	if err != nil {
		return err
	}
	if k.preprocess {
		defer k.manager.DecRef(formula)
		if !proof.IsNull() {
			defer k.manager.DecRef(proof)
		}
	}
	limit := len(k.assertions)
	if err = k.pushAssertion(formula, proof); err == nil {
		return nil
	}
	for index := len(k.assertions); index > limit; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			err = errors.Join(err, k.manager.DecRef(assertion.Proof))
		}
		err = errors.Join(err, k.manager.DecRef(assertion.Formula))
	}
	k.assertions = k.assertions[:limit]
	return err
}

func (k *SMTKernel) Assertions() []SMTKernelAssertion {
	if k == nil {
		return nil
	}
	return append([]SMTKernelAssertion(nil), k.assertions...)
}

func (k *SMTKernel) internalizePending() error {
	if err := k.propagatePendingAssertions(); err != nil {
		return err
	}
	for k.assertedHead < len(k.assertions) {
		assertion := k.assertions[k.assertedHead]
		if err := k.internalizer.Assert(assertion.Formula, assertion.Proof, 0); err != nil {
			return err
		}
		k.assertedHead++
	}
	return nil
}

func (k *SMTKernel) Push() error {
	if err := k.open(); err != nil {
		return err
	}
	if err := k.internalizePending(); err != nil {
		return err
	}
	if err := k.internalizer.PushScope(); err != nil {
		return err
	}
	k.scopes = append(k.scopes, len(k.assertions))
	return nil
}

func (k *SMTKernel) Pop(count uint32) error {
	if err := k.open(); err != nil {
		return err
	}
	if count == 0 || count > uint32(len(k.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(k.scopes) - int(count)
	limit := k.scopes[level]
	if err := k.internalizer.PopScopes(count); err != nil {
		return err
	}
	for index := len(k.assertions); index > limit; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			if err := k.manager.DecRef(assertion.Proof); err != nil {
				return err
			}
		}
		if err := k.manager.DecRef(assertion.Formula); err != nil {
			return err
		}
	}
	k.assertions = k.assertions[:limit]
	if k.assertedHead > limit {
		k.assertedHead = limit
	}
	k.scopes = k.scopes[:level]
	return nil
}

func (k *SMTKernel) ScopeLevel() uint32 {
	if k == nil {
		return 0
	}
	return uint32(len(k.scopes))
}

func (k *SMTKernel) prepareCheck() (*SMTActivityCaseSplitQueue, error) {
	if err := k.open(); err != nil {
		return nil, err
	}
	if err := k.internalizer.configureBooleanHeuristics(k.smtState); err != nil {
		return nil, fmt.Errorf("smt kernel configure Boolean heuristics: %w", err)
	}
	if err := k.internalizer.SetupTheories(); err != nil {
		return nil, fmt.Errorf("smt kernel setup theories: %w", err)
	}
	if err := k.internalizePending(); err != nil {
		return nil, fmt.Errorf("smt kernel internalize assertions: %w", err)
	}
	if err := k.internalizer.InitializeTheorySearch(); err != nil {
		return nil, fmt.Errorf("smt kernel initialize theory search: %w", err)
	}
	queue, err := NewSMTActivityCaseSplitQueue(k.internalizer)
	if err != nil {
		return nil, fmt.Errorf("smt kernel case split queue: %w", err)
	}
	if err = queue.ConfigurePhases(k.smtState); err != nil {
		return nil, fmt.Errorf("smt kernel case split phases: %w", err)
	}
	k.internalizer.statistics.NumChecks++
	return queue, nil
}

func (k *SMTKernel) Check() (lbool.LBool, error) {
	if err := k.clearLiteralResults(); err != nil {
		return lbool.Undef, err
	}
	if err := k.clearGeneratedModel(); err != nil {
		return lbool.Undef, err
	}
	if err := k.clearUnsatCore(); err != nil {
		return lbool.Undef, err
	}
	if err := k.manager.ResetRef(k.unsatProof); err != nil {
		return lbool.Undef, err
	}
	k.lastFailure = SMTFailureOK
	k.labels = nil
	queue, err := k.prepareCheck()
	if err != nil {
		return lbool.Undef, err
	}
	return k.search(queue)
}

func (k *SMTKernel) ConfigureModelGeneration(enabled bool, config model.ModelEvaluatorConfig) error {
	if err := k.open(); err != nil {
		return err
	}
	k.produceModel = enabled
	k.modelConfig = config
	return nil
}

func (k *SMTKernel) buildGeneratedModel() error {
	if !k.produceModel {
		return nil
	}
	generator, err := NewSMTModelGenerator(k.manager)
	if err != nil {
		return err
	}
	fail := func(cause error) error { return errors.Join(cause, generator.Close()) }
	if err = generator.SetContext(k.checker); err != nil {
		return fail(err)
	}
	if err = generator.InitializeModel(k.modelConfig); err != nil {
		return fail(err)
	}
	if err = generator.BuildStateValues(k.internalizer); err != nil {
		return fail(err)
	}
	generated, err := generator.FinalModel()
	if err != nil {
		return fail(err)
	}
	if err = generator.Close(); err != nil {
		return errors.Join(err, generated.Close())
	}
	if err = generated.Core().IncRef(); err != nil {
		return errors.Join(err, generated.Close())
	}
	k.generated = generated
	return nil
}

func (k *SMTKernel) clearGeneratedModel() error {
	if k == nil || k.generated == nil {
		return nil
	}
	consumed, err := k.generated.Core().DecRefConsumed()
	if consumed {
		k.generated = nil
	}
	return err
}

func (k *SMTKernel) GeneratedModel() *model.ModelCore {
	if k == nil || k.generated == nil {
		return nil
	}
	return k.generated.Core()
}

func (k *SMTKernel) LastFailure() (SMTFailure, error) {
	if err := k.open(); err != nil {
		return SMTFailureUnknown, err
	}
	return k.lastFailure, nil
}

func (k *SMTKernel) ReasonUnknown() (string, error) {
	if err := k.open(); err != nil {
		return "", err
	}
	switch k.lastFailure {
	case SMTFailureOK, SMTFailureUnknown:
		return k.unknown, nil
	case SMTFailureTimeout:
		return "timeout", nil
	case SMTFailureMemory:
		return "memout", nil
	case SMTFailureCanceled:
		return "canceled", nil
	case SMTFailureConflictLimit:
		return "max-conflicts-reached", nil
	case SMTFailureTheory:
		var reason strings.Builder
		reason.WriteString("(incomplete (theory")
		for _, theory := range k.internalizer.IncompleteTheories() {
			if named, ok := theory.(interface{ Name() string }); ok {
				if name := named.Name(); name != "" {
					reason.WriteByte(' ')
					reason.WriteString(name)
				}
			}
		}
		reason.WriteString("))")
		return reason.String(), nil
	case SMTFailureResourceLimit:
		return "(resource limits reached)", nil
	case SMTFailureQuantifiers:
		return "(incomplete quantifiers)", nil
	default:
		return "", ast.ErrASTDomain
	}
}

func (k *SMTKernel) SetReasonUnknown(message string) error {
	if err := k.open(); err != nil {
		return err
	}
	k.unknown = message
	return nil
}

func (k *SMTKernel) captureRelevantLabels() error {
	k.labels = nil
	for variable := int32(SMTFirstBoolVar); int(variable) < len(k.internalizer.boolExpr); variable++ {
		expression := k.internalizer.boolExpr[variable]
		if !k.relevancy.IsRelevant(expression) {
			continue
		}
		value, err := k.internalizer.Assignment(NewSMTLiteral(variable, false))
		if err != nil {
			return err
		}
		if value != lbool.True {
			continue
		}
		var names []usymbol.Symbol
		label, err := k.manager.IsLabelLit(expression, &names)
		if err != nil {
			return err
		}
		if label {
			k.labels = append(k.labels, names...)
		}
	}
	return nil
}

func (k *SMTKernel) Labels() ([]usymbol.Symbol, error) {
	if err := k.open(); err != nil {
		return nil, err
	}
	return append([]usymbol.Symbol(nil), k.labels...), nil
}

func (k *SMTKernel) SetProgressCallback(callback solver.ProgressCallback) error {
	if err := k.open(); err != nil {
		return err
	}
	k.callback = callback
	return nil
}

// Proof returns the kernel-owned proof from the latest unsatisfiable check.
// It is null when proof generation is disabled or the latest check did not
// establish unsatisfiability.
func (k *SMTKernel) Proof() (ast.ASTHandle, error) {
	if err := k.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return k.manager.RefGet(k.unsatProof)
}

func (k *SMTKernel) Inconsistent() bool {
	return k != nil && k.internalizer != nil && k.internalizer.Inconsistent()
}

func (k *SMTKernel) Statistics() SMTContextStatistics {
	if k == nil || k.internalizer == nil {
		return SMTContextStatistics{}
	}
	return k.internalizer.Statistics()
}

// ResetStatistics clears the cumulative counters retained by the kernel.
func (k *SMTKernel) ResetStatistics() error {
	if err := k.open(); err != nil {
		return err
	}
	k.internalizer.statistics.Reset()
	return nil
}

// Canceled performs the same resource-limit check used by the source context.
// The check accounts for one resource step before reporting the cancel state.
func (k *SMTKernel) Canceled() (bool, error) {
	if err := k.open(); err != nil {
		return false, err
	}
	return k.manager.Canceled()
}

// Display renders the retained asserted formulas in the source kernel format.
func (k *SMTKernel) Display(out io.Writer) error {
	if err := k.open(); err != nil {
		return err
	}
	if out == nil {
		return ast.ErrASTHandle
	}
	printer, err := ast.NewLegacySMTPrinter(k.manager)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(out, "(kernel"); err != nil {
		return err
	}
	for _, assertion := range k.assertions {
		if _, err = io.WriteString(out, "\n  "); err != nil {
			return err
		}
		if err = printer.DisplayAST(out, assertion.Formula, 2, nil); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, ")")
	return err
}

func (k *SMTKernel) Close() error {
	if k == nil || k.manager == nil {
		return ast.ErrASTHandle
	}
	if k.closed {
		return nil
	}
	var result error
	result = errors.Join(result, k.clearLiteralResults())
	result = errors.Join(result, k.clearGeneratedModel())
	result = errors.Join(result, k.clearUnsatCore())
	result = errors.Join(result, k.manager.DelRef(&k.unsatProof))
	if k.preprocessor != nil {
		if err := k.preprocessor.Close(); err != nil {
			result = errors.Join(result, err)
		} else {
			k.preprocessor = nil
		}
	}
	result = errors.Join(result, k.internalizer.Close())
	if k.smtParams != nil {
		result = errors.Join(result, k.smtParams.Close())
	}
	for index := len(k.assertions); index > 0; index-- {
		assertion := &k.assertions[index-1]
		if !assertion.Proof.IsNull() {
			result = errors.Join(result, k.manager.DecRef(assertion.Proof))
		}
		result = errors.Join(result, k.manager.DecRef(assertion.Formula))
	}
	if result == nil {
		k.assertions = nil
		k.internalizer = nil
		k.checker = nil
		k.relevancy = nil
		k.paramContext = nil
		k.paramGlobals = nil
		k.smtParams = nil
		k.smtState = SMTParamsState{}
		k.preprocessState = SMTPreprocessorState{}
		k.manager = nil
		k.closed = true
	}
	return result
}
