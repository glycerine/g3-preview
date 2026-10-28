package fp

// Source: src/muz/fp/horn_tactic.cpp retained finite verification path.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type HornTactic struct {
	manager    *ast.ASTManager
	context    *uparams.ParamsContext
	globals    *uparams.GlobalParamsService
	references uint32
	closed     bool
}

func NewHornTactic(manager *ast.ASTManager, context *uparams.ParamsContext, globals *uparams.GlobalParamsService) (*HornTactic, error) {
	if manager == nil || context == nil || globals == nil {
		return nil, tactic.NewTacticException("horn tactic services are unavailable")
	}
	return &HornTactic{manager: manager, context: context, globals: globals}, nil
}

func (t *HornTactic) open() error {
	if t == nil || t.closed || t.manager == nil || t.context == nil || t.globals == nil {
		return tactic.NewTacticException("horn tactic is closed")
	}
	return nil
}

func (t *HornTactic) IncRef() error {
	if err := t.open(); err != nil {
		return err
	}
	t.references++
	return nil
}

func (t *HornTactic) DecRef() error {
	if err := t.open(); err != nil {
		return err
	}
	if t.references == 0 {
		return tactic.NewTacticException("horn tactic reference underflow")
	}
	t.references--
	if t.references == 0 {
		t.closed = true
		t.manager, t.context, t.globals = nil, nil, nil
	}
	return nil
}

func (t *HornTactic) predicate(expression ast.ASTHandle) (ast.ASTHandle, bool, error) {
	kind, err := t.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return ast.ASTHandle{}, false, err
	}
	declaration, err := t.manager.GetDecl(expression)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	family, err := t.manager.GetFamilyID(declaration)
	if err != nil || family != ast.NullFamilyID {
		return ast.ASTHandle{}, false, err
	}
	sort, err := t.manager.GetSort(expression)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	boolean, err := t.manager.MkBoolSort()
	return declaration, err == nil && sort == boolean, err
}

func (t *HornTactic) registerPredicates(engine *PropositionalEngine, root ast.ASTHandle, seen map[ast.ASTHandle]bool) error {
	if declaration, yes, err := t.predicate(root); err != nil {
		return err
	} else if yes && !seen[declaration] {
		if err = engine.RegisterPredicate(declaration); err != nil {
			return err
		}
		seen[declaration] = true
	}
	kind, err := t.manager.GetKind(root)
	if err != nil || kind != ast.ASTApp {
		return err
	}
	count, err := t.manager.GetNumArgs(root)
	if err != nil {
		return err
	}
	for index := uint32(0); index < count; index++ {
		child, x := t.manager.GetArg(root, index)
		if x != nil {
			return x
		}
		if x = t.registerPredicates(engine, child, seen); x != nil {
			return x
		}
	}
	return nil
}

func (t *HornTactic) classify(formula ast.ASTHandle) (rule bool, query ast.ASTHandle, err error) {
	if _, yes, x := t.predicate(formula); x != nil {
		return false, query, x
	} else if yes {
		return true, query, nil
	}
	operator, err := t.manager.GetDeclKind(formula)
	if err != nil {
		return false, query, err
	}
	if operator == ast.ASTOpImplies {
		count, x := t.manager.GetNumArgs(formula)
		if x != nil || count != 2 {
			return false, query, errors.Join(x, tactic.NewTacticException("formula is not in Horn fragment"))
		}
		head, x := t.manager.GetArg(formula, 1)
		if x != nil {
			return false, query, x
		}
		_, yes, x := t.predicate(head)
		if x != nil || !yes {
			return false, query, errors.Join(x, tactic.NewTacticException("formula is not in Horn fragment"))
		}
		return true, query, nil
	}
	if operator == ast.ASTOpNot {
		argument, x := t.manager.GetArg(formula, 0)
		if x != nil {
			return false, query, x
		}
		if _, yes, x := t.predicate(argument); x != nil || !yes {
			return false, query, errors.Join(x, tactic.NewTacticException("Horn query must negate a predicate"))
		}
		return false, argument, nil
	}
	return false, query, tactic.NewTacticException("formula is not in Horn fragment")
}

func (t *HornTactic) Apply(goal *tactic.Goal, out *tactic.TacticResult) (resultErr error) {
	if err := t.open(); err != nil || goal == nil || goal.Manager() != t.manager {
		return errors.Join(err, tactic.NewTacticException("invalid Horn goal"))
	}
	if err := out.Prepare(t.manager); err != nil {
		return err
	}
	engine, err := NewPropositionalEngineWithServices(t.manager, t.context, t.globals)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, engine.Close()) }()
	seen := make(map[ast.ASTHandle]bool)
	var query ast.ASTHandle
	size, err := goal.Size()
	if err != nil {
		return err
	}
	for index := uint32(0); index < size; index++ {
		formula, x := goal.Form(index)
		if x != nil {
			return x
		}
		if x = t.registerPredicates(engine, formula, seen); x != nil {
			return x
		}
		isRule, candidate, x := t.classify(formula)
		if x != nil {
			return x
		}
		if isRule {
			if x = engine.AddRule(formula, "horn-tactic"); x != nil {
				return x
			}
		} else if !query.IsNull() {
			return tactic.NewTacticException("finite Horn tactic expects one query")
		} else {
			query = candidate
		}
	}
	if query.IsNull() {
		return tactic.NewTacticException("finite Horn tactic expects one query")
	}
	declaration, _, err := t.predicate(query)
	if err != nil {
		return err
	}
	status, err := engine.QueryPredicate(declaration)
	if err != nil {
		return err
	}
	if err = goal.IncDepth(); err != nil {
		return err
	}
	if status == lbool.True {
		falsity, x := t.manager.MkFalse()
		if x != nil {
			return x
		}
		if x = goal.Assert(falsity); x != nil {
			return x
		}
	} else if status == lbool.False {
		if err = goal.Reset(); err != nil {
			return err
		}
	}
	return out.AddGoal(goal)
}

func (t *HornTactic) UpdateParams(*uparams.ParamsRef) error         { return t.open() }
func (t *HornTactic) CollectParamDescrs(*uparams.ParamDescrs) error { return t.open() }
func (t *HornTactic) CollectStatistics() map[string]float64         { return nil }
func (t *HornTactic) ResetStatistics() error                        { return t.open() }
func (t *HornTactic) Cleanup() error                                { return t.open() }
func (t *HornTactic) Reset() error                                  { return t.open() }
func (t *HornTactic) SetLogic(usymbol.Symbol) error                 { return t.open() }
func (t *HornTactic) SetProgressCallback(func(uint32)) error        { return t.open() }
func (t *HornTactic) Translate(manager *ast.ASTManager) (tactic.Tactic, error) {
	if err := t.open(); err != nil {
		return nil, err
	}
	if manager != t.manager {
		return nil, fmt.Errorf("horn tactic translation requires matching services")
	}
	return NewHornTactic(manager, t.context, t.globals)
}
