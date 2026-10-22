// Source: src/smt/smt_solver.h; src/smt/smt_solver.cpp retained kernel-backed
// solver facade. Model, proof-result, consequence, mutex, core-extension, and
// cross-manager kernel-copy operations remain with their kernel prerequisites.
package smtcore

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

type SMTSolverConfig struct {
	Params       *uparams.ParamsContext
	Quantifiers  *SMTQuantifierRegistry
	TargetConfig func(*ast.ASTManager) (SMTSolverConfig, error)
}

type smtTrackedAssertion struct {
	tag     ast.ASTHandle
	formula ast.ASTRef
}

// SMTSolver exposes the retained SMT kernel through solver.Solver. The result
// object is replaced on each check, while kernel statistics remain cumulative.
type SMTSolver struct {
	self          *SMTSolver
	manager       *ast.ASTManager
	config        SMTSolverConfig
	kernel        *SMTKernel
	named         *solver.NamedAssertions
	result        *solver.SimpleCheckSatResult
	params        uparams.ParamsRef
	logic         usymbol.Symbol
	tracked       []smtTrackedAssertion
	trackedScopes []uint32
	callback      solver.ProgressCallback
	references    uint32
	produceModels bool
	closed        bool
}

var _ solver.Solver = (*SMTSolver)(nil)

func NewSMTSolver(m *ast.ASTManager, p *uparams.ParamsRef, logic string, config SMTSolverConfig) (*SMTSolver, error) {
	if m == nil || config.Params == nil {
		return nil, uparams.ErrParamsOwner
	}
	if err := config.Params.CheckServices(m.Symbols, m.NumeralService()); err != nil {
		return nil, err
	}
	value := config.Params.EmptyValue()
	if p != nil {
		if err := config.Params.Copy(&value, p); err != nil {
			return nil, err
		}
	}
	logicSymbol := usymbol.Symbol{}
	var err error
	if logic != "" {
		logicSymbol, err = m.Symbols.Intern(logic)
		if err != nil {
			_ = config.Params.Del(&value)
			return nil, err
		}
	}
	kernel, err := NewSMTKernel(m, config.Quantifiers)
	if err != nil {
		_ = config.Params.Del(&value)
		return nil, err
	}
	s := &SMTSolver{manager: m, config: config, kernel: kernel, params: value, logic: logicSymbol}
	s.self = s
	s.named, err = solver.NewNamedAssertions(m, s)
	if err != nil {
		_ = kernel.Close()
		_ = config.Params.Del(&value)
		return nil, err
	}
	return s, nil
}

func (s *SMTSolver) open() error {
	if s == nil || s.self != s {
		return ast.ErrASTHandle
	}
	if s.closed {
		return arena.ErrClosed
	}
	return s.config.Params.Check(&s.params)
}

func (s *SMTSolver) IncRef() error {
	if err := s.open(); err != nil {
		return err
	}
	s.references++
	return nil
}

func (s *SMTSolver) DecRef() error {
	if err := s.open(); err != nil {
		return err
	}
	if s.references == 0 {
		return arena.ErrBounds
	}
	s.references--
	if s.references == 0 {
		return s.Close()
	}
	return nil
}

func (s *SMTSolver) clearResult() error {
	if s.result == nil {
		return nil
	}
	err := s.result.Close()
	if err == nil {
		s.result = nil
	}
	return err
}

func (s *SMTSolver) replaceResult(next *solver.SimpleCheckSatResult) error {
	if err := s.clearResult(); err != nil {
		return err
	}
	s.result = next
	return nil
}

func (s *SMTSolver) Status() (lbool.LBool, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, err
	}
	if s.result == nil {
		return lbool.Undef, nil
	}
	return s.result.Status()
}

func (s *SMTSolver) SetStatus(v lbool.LBool) error {
	if err := s.open(); err != nil {
		return err
	}
	if s.result == nil {
		return ast.ErrASTHandle
	}
	return s.result.SetStatus(v)
}

func collectSMTKernelStatistics(dst *statistics.Statistics, value SMTContextStatistics) {
	dst.UpdateUint("conflicts", value.NumConflicts)
	dst.UpdateUint("decisions", value.NumDecisions)
	dst.UpdateUint("propagations", value.NumPropagations+value.NumBinPropagations)
	dst.UpdateUint("binary propagations", value.NumBinPropagations)
	dst.UpdateUint("restarts", value.NumRestarts)
	dst.UpdateUint("final checks", value.NumFinalChecks)
	dst.UpdateUint("added eqs", value.NumAddEq)
	dst.UpdateUint("mk clause", value.NumMkClause)
	dst.UpdateUint("del clause", value.NumDelClause)
	dst.UpdateUint("dyn ack", value.NumDynAck)
	dst.UpdateUint("interface eqs", value.NumInterfaceEqs)
	dst.UpdateUint("max generation", value.MaxGeneration)
	dst.UpdateUint("minimized lits", value.NumMinimizedLits)
	dst.UpdateUint("num checks", value.NumChecks)
	dst.UpdateUint("mk bool var", value.NumMkBoolVar)
}

func (s *SMTSolver) CollectStatistics(dst *statistics.Statistics) error {
	if err := s.open(); err != nil {
		return err
	}
	if dst == nil {
		return arena.ErrInvalidSlot
	}
	collectSMTKernelStatistics(dst, s.kernel.Statistics())
	return nil
}

func (s *SMTSolver) UnsatCore() ([]ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if s.result == nil {
		return []ast.ASTHandle{}, nil
	}
	return s.result.UnsatCore()
}

func (s *SMTSolver) AcquireModel() (*model.ModelCore, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if s.result == nil {
		return nil, nil
	}
	return s.result.AcquireModel()
}

func (s *SMTSolver) Proof() (ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if s.result == nil {
		return ast.ASTHandle{}, nil
	}
	return s.result.Proof()
}

func (s *SMTSolver) ReasonUnknown() (string, error) {
	if err := s.open(); err != nil {
		return "", err
	}
	if s.result == nil {
		return s.kernel.ReasonUnknown()
	}
	return s.result.ReasonUnknown()
}

func (s *SMTSolver) SetReasonUnknown(v string) error {
	if err := s.open(); err != nil {
		return err
	}
	if err := s.kernel.SetReasonUnknown(v); err != nil {
		return err
	}
	if s.result == nil {
		return nil
	}
	return s.result.SetReasonUnknown(v)
}

func (s *SMTSolver) Labels() ([]usymbol.Symbol, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if s.result == nil {
		return []usymbol.Symbol{}, nil
	}
	return s.result.Labels()
}

func (s *SMTSolver) Manager() (*ast.ASTManager, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	return s.manager, nil
}

func (s *SMTSolver) UpdateParams(p *uparams.ParamsRef) error {
	if err := s.open(); err != nil {
		return err
	}
	if p == nil {
		empty := s.config.Params.EmptyValue()
		p = &empty
	}
	return s.config.Params.Copy(&s.params, p)
}

func (s *SMTSolver) Params() (*uparams.ParamsRef, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	return &s.params, nil
}

func (s *SMTSolver) CollectParamDescrs(d *uparams.ParamDescrs) error {
	if err := s.open(); err != nil {
		return err
	}
	if d == nil {
		return uparams.ErrParamDescrsPrecondition
	}
	return catalogCollectSmtParamsHelper(d)
}

func (s *SMTSolver) SetProduceModels(value bool) error {
	if err := s.open(); err != nil {
		return err
	}
	s.produceModels = value
	return nil
}

func (s *SMTSolver) AssertFormula(formula ast.ASTHandle) error {
	if err := s.open(); err != nil {
		return err
	}
	if err := s.kernel.Assert(formula, ast.ASTHandle{}); err != nil {
		return err
	}
	return s.clearResult()
}

func (s *SMTSolver) Assert(formula ast.ASTHandle) error { return s.AssertFormula(formula) }

func (s *SMTSolver) AssertTracked(formula, tag ast.ASTHandle) error {
	if err := s.open(); err != nil {
		return err
	}
	if tag.IsNull() {
		return s.AssertFormula(formula)
	}
	for _, assertion := range s.tracked {
		if assertion.tag == tag {
			return z3exception.NewDefaultException("named assertion defined twice")
		}
	}
	ref, err := s.manager.NewRef(formula)
	if err != nil {
		return err
	}
	if err = s.named.AssertTracked(formula, tag); err != nil {
		return errors.Join(err, s.manager.DelRef(&ref))
	}
	s.tracked = append(s.tracked, smtTrackedAssertion{tag: tag, formula: ref})
	return nil
}

func (s *SMTSolver) PushCore() error {
	if err := s.open(); err != nil {
		return err
	}
	if err := s.kernel.Push(); err != nil {
		return err
	}
	s.trackedScopes = append(s.trackedScopes, uint32(len(s.tracked)))
	return s.clearResult()
}

func (s *SMTSolver) Push() error { return s.named.Push() }

func (s *SMTSolver) restoreTracked(size uint32) error {
	for uint32(len(s.tracked)) > size {
		index := len(s.tracked) - 1
		if err := s.manager.DelRef(&s.tracked[index].formula); err != nil {
			return err
		}
		s.tracked = s.tracked[:index]
	}
	return nil
}

func (s *SMTSolver) PopCore(count uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	if count > uint32(len(s.trackedScopes)) {
		return arena.ErrBounds
	}
	level := uint32(len(s.trackedScopes)) - count
	if err := s.kernel.Pop(count); err != nil {
		return err
	}
	if err := s.restoreTracked(s.trackedScopes[level]); err != nil {
		return err
	}
	s.trackedScopes = s.trackedScopes[:level]
	return s.clearResult()
}

func (s *SMTSolver) Pop(count uint32) error { return s.named.Pop(count) }

func (s *SMTSolver) ScopeLevel() (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	return s.named.ScopeLevel()
}

func (s *SMTSolver) CheckSatCore(assumptions []ast.ASTHandle) (lbool.LBool, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, err
	}
	result, err := solver.NewSimpleCheckSatResult(s.manager)
	if err != nil {
		return lbool.Undef, err
	}
	if err = s.replaceResult(result); err != nil {
		_ = result.Close()
		return lbool.Undef, err
	}
	status, err := s.kernel.CheckAssumptions(assumptions)
	if err != nil {
		return lbool.Undef, err
	}
	if err = result.SetStatus(status); err != nil {
		return lbool.Undef, err
	}
	if status == lbool.Undef {
		reason, reasonErr := s.kernel.ReasonUnknown()
		if reasonErr != nil {
			return lbool.Undef, reasonErr
		}
		if err = result.SetReasonUnknown(reason); err != nil {
			return lbool.Undef, err
		}
	}
	labels, labelsErr := s.kernel.Labels()
	if labelsErr != nil {
		return lbool.Undef, labelsErr
	}
	if err = result.SetLabels(labels); err != nil {
		return lbool.Undef, err
	}
	if status == lbool.False {
		if err = result.SetUnsatCore(s.kernel.UnsatCore()); err != nil {
			return lbool.Undef, err
		}
		proof, proofErr := s.kernel.Proof()
		if proofErr != nil {
			return lbool.Undef, proofErr
		}
		if err = result.SetProof(proof); err != nil {
			return lbool.Undef, err
		}
	}
	stats, err := result.Statistics()
	if err != nil {
		return lbool.Undef, err
	}
	collectSMTKernelStatistics(stats, s.kernel.Statistics())
	return status, nil
}

func (s *SMTSolver) CheckSat(assumptions []ast.ASTHandle) (lbool.LBool, error) {
	return s.named.CheckSat(assumptions)
}

func (s *SMTSolver) NumAssertions() (uint32, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	return uint32(len(s.kernel.Assertions())), nil
}

func (s *SMTSolver) Assertion(index uint32) (ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	assertions := s.kernel.Assertions()
	if uint64(index) >= uint64(len(assertions)) {
		return ast.ASTHandle{}, arena.ErrBounds
	}
	return assertions[index].Formula, nil
}

func (s *SMTSolver) NumAssumptions() (uint32, error) { return s.named.NumAssumptions() }
func (s *SMTSolver) Assumption(index uint32) (ast.ASTHandle, error) {
	return s.named.Assumption(index)
}

func (s *SMTSolver) GetConsequences([]ast.ASTHandle, []ast.ASTHandle) (lbool.LBool, []ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, nil, err
	}
	return lbool.Undef, nil, ast.ErrASTDomain
}

func (s *SMTSolver) FindMutexes([]ast.ASTHandle) (lbool.LBool, [][]ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, nil, err
	}
	return lbool.Undef, nil, ast.ErrASTDomain
}

func (s *SMTSolver) PreferredSAT([]ast.ASTHandle) (lbool.LBool, [][]ast.ASTHandle, error) {
	return solver.DefaultPreferredSAT(s)
}

func (s *SMTSolver) SetProgressCallback(callback solver.ProgressCallback) error {
	if err := s.open(); err != nil {
		return err
	}
	s.callback = callback
	return nil
}

func (s *SMTSolver) DisplayCore(out io.Writer) error { return solver.DefaultDisplay(s, out, nil) }
func (s *SMTSolver) Display(out io.Writer, assumptions []ast.ASTHandle) error {
	return solver.DefaultDisplay(s, out, assumptions)
}

func (s *SMTSolver) Translate(manager *ast.ASTManager, params *uparams.ParamsRef) (solver.Solver, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if manager == nil {
		return nil, ast.ErrASTHandle
	}
	if level, err := s.ScopeLevel(); err != nil {
		return nil, err
	} else if level != 0 {
		return nil, z3exception.NewDefaultException("translation of contexts is only supported at base level")
	}
	config := s.config
	if err := config.Params.CheckServices(manager.Symbols, manager.NumeralService()); err != nil {
		if config.TargetConfig == nil {
			return nil, err
		}
		config, err = config.TargetConfig(manager)
		if err != nil {
			return nil, err
		}
	}
	logic := ""
	var err error
	if s.logic != (usymbol.Symbol{}) {
		logic, err = s.manager.Symbols.String(s.logic)
		if err != nil {
			return nil, err
		}
	}
	next, err := NewSMTSolver(manager, params, logic, config)
	if err != nil {
		return nil, err
	}
	next.produceModels = s.produceModels
	translate := func(value ast.ASTHandle) (ast.ASTHandle, error) { return value, nil }
	var translation *ast.ASTTranslation
	if manager != s.manager {
		translation, err = ast.NewASTTranslation(s.manager, manager, false)
		if err != nil {
			_ = next.Close()
			return nil, err
		}
		defer translation.Close()
		translate = translation.Translate
	}
	for _, assertion := range s.kernel.Assertions() {
		formula, copyErr := translate(assertion.Formula)
		if copyErr != nil {
			_ = next.Close()
			return nil, copyErr
		}
		proof, copyErr := translate(assertion.Proof)
		if copyErr != nil {
			_ = next.Close()
			return nil, copyErr
		}
		if copyErr = next.kernel.Assert(formula, proof); copyErr != nil {
			_ = next.Close()
			return nil, copyErr
		}
	}
	for _, assertion := range s.tracked {
		formula, getErr := s.manager.RefGet(assertion.formula)
		if getErr != nil {
			_ = next.Close()
			return nil, getErr
		}
		formula, getErr = translate(formula)
		if getErr != nil {
			_ = next.Close()
			return nil, getErr
		}
		tag, getErr := translate(assertion.tag)
		if getErr != nil {
			_ = next.Close()
			return nil, getErr
		}
		if getErr = next.AssertTracked(formula, tag); getErr != nil {
			_ = next.Close()
			return nil, getErr
		}
	}
	return next, nil
}

func (s *SMTSolver) Close() error {
	if s == nil || s.self != s {
		return ast.ErrASTHandle
	}
	if s.closed {
		return nil
	}
	var result error
	result = errors.Join(result, s.clearResult())
	result = errors.Join(result, s.named.Close())
	result = errors.Join(result, s.restoreTracked(0))
	result = errors.Join(result, s.kernel.Close())
	result = errors.Join(result, s.config.Params.Del(&s.params))
	if result == nil {
		s.named = nil
		s.kernel = nil
		s.callback = nil
		s.manager = nil
		s.closed = true
	}
	return result
}

type smtSolverFactory struct {
	config SMTSolverConfig
	closed bool
}

func NewSMTSolverFactory(config SMTSolverConfig) (solver.Factory, error) {
	if config.Params == nil {
		return nil, uparams.ErrParamsOwner
	}
	return &smtSolverFactory{config: config}, nil
}

func (f *smtSolverFactory) New(m *ast.ASTManager, p *uparams.ParamsRef, _, models, _ bool, logic string) (solver.Solver, error) {
	if f == nil || f.closed {
		return nil, arena.ErrClosed
	}
	value, err := NewSMTSolver(m, p, logic, f.config)
	if err != nil {
		return nil, err
	}
	value.produceModels = models
	return value, nil
}

func (f *smtSolverFactory) Close() error {
	if f == nil || f.closed {
		return nil
	}
	f.closed = true
	return nil
}
