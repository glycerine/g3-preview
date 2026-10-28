package fp

// Source: src/muz/base/dl_context.cpp add_rule/query orchestration and
// src/muz/base/dl_engine_base.h query contract. This is the first exact
// end-to-end engine slice: positive, zero-arity Horn programs only.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/dataflow"
	"github.com/glycerine/g3/pkg/util/lbool"
)

var (
	ErrPropositionalEngine = errors.New("muz/fp: invalid or closed propositional engine")
	ErrPropositionalHorn   = errors.New("muz/fp: positive zero-arity Horn fragment required")
)

// PropositionalEngine owns a rule manager and rule set for the exact positive
// zero-arity Horn fragment. Richer fixedpoint programs are refused explicitly.
type PropositionalEngine struct {
	manager    *ast.ASTManager
	rules      *base.RuleManager
	set        *base.RuleSet
	predicates map[ast.ASTHandle]bool
	closed     bool
}

func NewPropositionalEngine(manager *ast.ASTManager) (*PropositionalEngine, error) {
	if manager == nil {
		return nil, ErrPropositionalEngine
	}
	rules, err := base.NewRuleManager(manager)
	if err != nil {
		return nil, err
	}
	set, err := base.NewRuleSet(rules)
	if err != nil {
		return nil, errors.Join(err, rules.Close())
	}
	return &PropositionalEngine{
		manager: manager, rules: rules, set: set, predicates: make(map[ast.ASTHandle]bool),
	}, nil
}

func (e *PropositionalEngine) open() error {
	if e == nil || e.closed || e.manager == nil || e.rules == nil || e.set == nil {
		return ErrPropositionalEngine
	}
	return nil
}

func (e *PropositionalEngine) RegisterPredicate(declaration ast.ASTHandle) error {
	if err := e.open(); err != nil {
		return err
	}
	kind, err := e.manager.GetKind(declaration)
	if err != nil {
		return err
	}
	if kind != ast.ASTFuncDecl {
		return ErrPropositionalHorn
	}
	arity, err := e.manager.GetArity(declaration)
	if err != nil {
		return err
	}
	rangeSort, err := e.manager.GetRange(declaration)
	if err != nil {
		return err
	}
	boolean, err := e.manager.MkBoolSort()
	if err != nil {
		return err
	}
	if arity != 0 || rangeSort != boolean {
		return ErrPropositionalHorn
	}
	if err = e.rules.RegisterPredicate(declaration); err != nil {
		return err
	}
	e.predicates[declaration] = true
	return nil
}

func (e *PropositionalEngine) AddRule(formula ast.ASTHandle, name string) error {
	if err := e.open(); err != nil {
		return err
	}
	finalized, err := e.set.IsFinalized()
	if err != nil {
		return err
	}
	if finalized {
		if err = e.set.Reopen(); err != nil {
			return err
		}
	}
	return e.rules.AddFormula(e.set, formula, name)
}

func (e *PropositionalEngine) validateFragment() error {
	count, err := e.set.Len()
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		rule, x := e.set.Rule(i)
		if x != nil {
			return x
		}
		head, x := rule.HeadDeclaration()
		if x == nil {
			var arity uint32
			arity, x = e.manager.GetArity(head)
			if x == nil && arity != 0 {
				x = ErrPropositionalHorn
			}
		}
		positive, pErr := rule.PositiveTailSize()
		uninterpreted, uErr := rule.UninterpretedTailSize()
		tailSize, tErr := rule.TailSize()
		x = errors.Join(x, pErr, uErr, tErr)
		if x == nil && (positive != uninterpreted || uninterpreted != tailSize) {
			x = ErrPropositionalHorn
		}
		for j := 0; x == nil && j < uninterpreted; j++ {
			declaration, y := rule.TailDeclaration(j)
			if y != nil {
				x = y
				break
			}
			arity, y := e.manager.GetArity(declaration)
			if y != nil {
				x = y
			} else if arity != 0 {
				x = ErrPropositionalHorn
			}
		}
		x = errors.Join(x, rule.Close())
		if x != nil {
			return x
		}
	}
	return nil
}

// Query returns true exactly when the registered zero-arity query predicate is
// derivable by the positive Horn program.
func (e *PropositionalEngine) Query(query ast.ASTHandle) (lbool.LBool, error) {
	if err := e.open(); err != nil {
		return lbool.Undef, err
	}
	kind, err := e.manager.GetKind(query)
	if err != nil {
		return lbool.Undef, err
	}
	if kind != ast.ASTApp {
		return lbool.Undef, ErrPropositionalHorn
	}
	declaration, err := e.manager.GetDecl(query)
	if err != nil {
		return lbool.Undef, err
	}
	arguments, err := e.manager.GetNumArgs(query)
	if err != nil {
		return lbool.Undef, err
	}
	if arguments != 0 || !e.predicates[declaration] {
		return lbool.Undef, ErrPropositionalHorn
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
	if err = e.validateFragment(); err != nil {
		return lbool.Undef, err
	}
	reachable, err := dataflow.BottomUp(e.set)
	if err != nil {
		return lbool.Undef, err
	}
	return lbool.FromBool(reachable.IsReachable(declaration)), nil
}

func (e *PropositionalEngine) Close() (result error) {
	if e == nil {
		return ErrPropositionalEngine
	}
	if e.closed {
		return nil
	}
	if e.set != nil {
		result = errors.Join(result, e.set.Close())
		e.set = nil
	}
	if e.rules != nil {
		result = errors.Join(result, e.rules.Close())
		e.rules = nil
	}
	e.predicates = nil
	e.closed = true
	return result
}
