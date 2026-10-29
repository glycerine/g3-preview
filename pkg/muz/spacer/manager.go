package spacer

// Source: the retained state-symbol, background, solver-factory, and
// statistics paths in src/muz/spacer/spacer_manager.h and spacer_manager.cpp.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/cooperate"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

var ErrManagerOwner = errors.New("muz/spacer: invalid or closed manager")

type Manager struct {
	manager     *ast.ASTManager
	mux         *SymMux
	background  ast.ASTRef
	contexts    [3]*SMTContextManager
	cooperation *cooperate.Cooperation
	unique      uint32
	closed      bool
}

func NewManager(manager *ast.ASTManager, params *uparams.ParamsRef, maximumContexts uint32, config smtcore.SMTSolverConfig) (_ *Manager, resultErr error) {
	if manager == nil || config.Params == nil || config.Globals == nil {
		return nil, ErrManagerOwner
	}
	mux, err := NewSymMux(manager, []string{"_n"})
	if err != nil {
		return nil, err
	}
	truth, err := manager.MkTrue()
	if err != nil {
		return nil, errors.Join(err, mux.Close())
	}
	background, err := manager.NewRef(truth)
	if err != nil {
		return nil, errors.Join(err, mux.Close())
	}
	result := &Manager{manager: manager, mux: mux, background: background}
	committed := false
	defer func() {
		if !committed {
			resultErr = errors.Join(resultErr, result.Close())
		}
	}()
	if config.ModelEvaluator.Context == nil {
		config.ModelEvaluator.Context = config.Params
	}
	if config.ModelEvaluator.Globals == nil {
		config.ModelEvaluator.Globals = config.Globals
	}
	if config.ModelEvaluator.Memory == nil {
		config.ModelEvaluator.Memory = manager.NumeralService().RuntimeOwner()
	}
	if config.ModelEvaluator.Cooperation == nil {
		result.cooperation, err = cooperate.NewCooperation(manager.Budget())
		if err != nil {
			return nil, err
		}
		config.ModelEvaluator.Cooperation = result.cooperation
	}
	for index := range result.contexts {
		result.contexts[index], err = NewSMTContextManager(manager, params, maximumContexts, config)
		if err != nil {
			return nil, err
		}
	}
	committed = true
	return result, nil
}

func (m *Manager) open() error {
	if m == nil || m.closed || m.manager == nil || m.mux == nil {
		return ErrManagerOwner
	}
	return nil
}

func (m *Manager) Mux() (*SymMux, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux, nil
}

func (m *Manager) addState(prefix ast.ASTHandle) error {
	arity, err := m.manager.GetArity(prefix)
	if err != nil {
		return err
	}
	if arity != 0 {
		return ast.ErrASTDomain
	}
	_, err = m.mux.CreateTuple(prefix, 2)
	return err
}

func (m *Manager) NewPredicate(prefix ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	value, found, err := m.mux.ByPrefix(prefix, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !found {
		if err = m.addState(prefix); err != nil {
			return ast.ASTHandle{}, err
		}
		value, found, err = m.mux.ByPrefix(prefix, 0)
	}
	if err != nil || !found {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return ast.ASTHandle{}, err
	}
	return value, nil
}

func (m *Manager) OldPredicate(prefix ast.ASTHandle, oldIndex uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	value, found, err := m.mux.ByPrefix(prefix, oldIndex+1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !found {
		if err = m.addState(prefix); err != nil {
			return ast.ASTHandle{}, err
		}
		value, found, err = m.mux.ByPrefix(prefix, oldIndex+1)
	}
	if err != nil || !found {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return ast.ASTHandle{}, err
	}
	return value, nil
}

func (m *Manager) OldToNew(declaration ast.ASTHandle, oldIndex uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.mux.ConvertDeclaration(declaration, oldIndex+1, 0)
}

func (m *Manager) NewToOld(declaration ast.ASTHandle, oldIndex uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.mux.ConvertDeclaration(declaration, 0, oldIndex+1)
}

func (m *Manager) OldToOld(declaration ast.ASTHandle, source, target uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.mux.ConvertDeclaration(declaration, source+1, target+1)
}

func (m *Manager) FormulaNewToOld(formula ast.ASTHandle, oldIndex uint32, homogeneous bool) (ast.ASTRef, error) {
	if err := m.open(); err != nil {
		return ast.ASTRef{}, err
	}
	return m.mux.ConvertFormula(formula, 0, oldIndex+1, homogeneous)
}

func (m *Manager) FormulaOldToNew(formula ast.ASTHandle, oldIndex uint32, homogeneous bool) (ast.ASTRef, error) {
	if err := m.open(); err != nil {
		return ast.ASTRef{}, err
	}
	return m.mux.ConvertFormula(formula, oldIndex+1, 0, homogeneous)
}

func (m *Manager) IsOldFormula(formula ast.ASTHandle) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	containsNew, err := m.mux.Contains(formula, 0)
	return !containsNew, err
}

func (m *Manager) IsOldFormulaAt(formula ast.ASTHandle, oldIndex uint32) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	return m.mux.IsHomogeneousAt(formula, oldIndex+1)
}

func (m *Manager) IsNewFormula(formula ast.ASTHandle) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	return m.mux.IsHomogeneousAt(formula, 0)
}

func (m *Manager) IsHomogeneousFormula(formula ast.ASTHandle) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	return m.mux.IsHomogeneous(formula)
}

func (m *Manager) CollectStateIndices(formula ast.ASTHandle) ([]uint32, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux.CollectIndices(formula)
}

func (m *Manager) CollectStateVariables(formula ast.ASTHandle) ([][]ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux.CollectVariables(formula)
}

func (m *Manager) HaveDifferentStateKinds(left, right ast.ASTHandle) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	leftIndex, leftMuxed, err := m.mux.Index(left)
	if err != nil {
		return false, err
	}
	rightIndex, rightMuxed, err := m.mux.Index(right)
	return leftMuxed && rightMuxed && leftIndex != rightIndex, err
}

func (m *Manager) FilterOldAtoms(formulas []ast.ASTHandle, oldIndex uint32) ([]ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux.FilterIndex(formulas, oldIndex+1)
}

func (m *Manager) FilterNewAtoms(formulas []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux.FilterIndex(formulas, 0)
}

func (m *Manager) PartitionOldAtoms(formulas []ast.ASTHandle, oldIndex uint32) ([]ast.ASTHandle, []ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, nil, err
	}
	return m.mux.PartitionIndex(formulas, oldIndex+1)
}

func (m *Manager) FilterNonModelAtoms(formulas []ast.ASTHandle) ([]ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	return m.mux.FilterNonModel(formulas)
}

func (m *Manager) ModelString(modelCore *model.ModelCore) (string, error) {
	if err := m.open(); err != nil {
		return "", err
	}
	return m.mux.ModelString(modelCore)
}

func (m *Manager) SetBackground(background ast.ASTHandle) error {
	if err := m.open(); err != nil {
		return err
	}
	return m.manager.SetRef(m.background, background)
}

func (m *Manager) Background() (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.manager.RefGet(m.background)
}

func (m *Manager) Unique() (uint32, error) {
	if err := m.open(); err != nil {
		return 0, err
	}
	value := m.unique
	m.unique++
	return value, nil
}

func (m *Manager) Fresh(pool uint32) (*VirtualSolver, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	if pool >= uint32(len(m.contexts)) {
		return nil, ast.ErrASTDomain
	}
	return m.contexts[pool].Fresh()
}

func (m *Manager) NewPropSolver(name string, config PropSolverConfig) (_ *PropSolver, resultErr error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	var concrete [2]*VirtualSolver
	var backends [2]solver.Solver
	for index := range concrete {
		created, err := m.contexts[index].Fresh()
		if err != nil {
			for prior := index - 1; prior >= 0; prior-- {
				resultErr = errors.Join(resultErr, concrete[prior].Close())
			}
			return nil, errors.Join(err, resultErr)
		}
		concrete[index], backends[index] = created, created
	}
	background, err := m.Background()
	if err != nil {
		return nil, errors.Join(err, concrete[1].Close(), concrete[0].Close())
	}
	result, err := NewPropSolver(m.manager, backends, background, name, config)
	if err != nil {
		return nil, errors.Join(err, concrete[1].Close(), concrete[0].Close())
	}
	result.ownedBackends = concrete
	return result, nil
}

func (m *Manager) CollectStatistics(destination *statistics.Statistics) error {
	if err := m.open(); err != nil {
		return err
	}
	for _, context := range m.contexts {
		if err := context.CollectStatistics(destination); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ResetStatistics() error {
	if err := m.open(); err != nil {
		return err
	}
	for _, context := range m.contexts {
		if err := context.ResetStatistics(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Close() (result error) {
	if m == nil || m.manager == nil {
		return ErrManagerOwner
	}
	if m.closed {
		return nil
	}
	for _, context := range m.contexts {
		if context == nil {
			continue
		}
		for _, factory := range context.factories {
			for _, virtual := range factory.solvers {
				if virtual != nil && !virtual.closed {
					return ErrManagerOwner
				}
			}
		}
	}
	for index := len(m.contexts) - 1; index >= 0; index-- {
		if m.contexts[index] != nil {
			result = errors.Join(result, m.contexts[index].Close())
		}
	}
	result = errors.Join(result, m.mux.Close(), m.manager.DelRef(&m.background))
	if m.cooperation != nil {
		result = errors.Join(result, m.cooperation.Close())
	}
	if result == nil {
		m.contexts = [3]*SMTContextManager{}
		m.mux = nil
		m.cooperation = nil
		m.manager = nil
		m.closed = true
	}
	return result
}
