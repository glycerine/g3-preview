// Source: src/smt/arith_eq_adapter.h and src/smt/arith_eq_adapter.cpp retained
// arithmetic equality-to-bound adapter.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type smtArithmeticEqualityPair struct {
	first, second ast.ASTHandle
}

// smtArithmeticEqualityAdapter converts equality and disequality events from
// congruence closure into the arithmetic equality atom handled by the owning
// theory. InternalizeEquality then creates the paired <= and >= atoms and the
// triangle clauses. Processed pairs follow theory scopes.
type smtArithmeticEqualityAdapter struct {
	owner     *SMTTheoryBase
	processed map[smtArithmeticEqualityPair]struct{}
	trail     []smtArithmeticEqualityPair
	limits    []int
	restart   []smtArithmeticEqualityPair
	numAxioms uint64
}

func newSMTArithmeticEqualityAdapter(owner *SMTTheoryBase) (*smtArithmeticEqualityAdapter, error) {
	if owner == nil {
		return nil, ast.ErrASTHandle
	}
	return &smtArithmeticEqualityAdapter{
		owner:     owner,
		processed: make(map[smtArithmeticEqualityPair]struct{}),
	}, nil
}

func (a *smtArithmeticEqualityAdapter) pair(left, right *SMTEnode) smtArithmeticEqualityPair {
	first, second := left.Owner(), right.Owner()
	if first.Slot() > second.Slot() || first.Slot() == second.Slot() && first.Generation() > second.Generation() {
		first, second = second, first
	}
	return smtArithmeticEqualityPair{first: first, second: second}
}

func (a *smtArithmeticEqualityAdapter) MkAxioms(left, right SMTTheoryVar) error {
	if a == nil || a.owner == nil || left == right {
		return ast.ErrASTDomain
	}
	leftNode, err := a.owner.Enode(left)
	if err != nil {
		return err
	}
	rightNode, err := a.owner.Enode(right)
	if err != nil {
		return err
	}
	if leftNode == rightNode || leftNode.Owner() == rightNode.Owner() {
		return nil
	}
	state, err := a.owner.State()
	if err != nil {
		return err
	}
	leftValue, err := state.manager.IsValue(leftNode.Owner())
	if err != nil {
		return err
	}
	rightValue, err := state.manager.IsValue(rightNode.Owner())
	if err != nil {
		return err
	}
	if leftValue && rightValue {
		return nil
	}
	pair := a.pair(leftNode, rightNode)
	if _, found := a.processed[pair]; found {
		return nil
	}
	a.processed[pair] = struct{}{}
	a.trail = append(a.trail, pair)
	equality, err := state.manager.MkEq(pair.first, pair.second)
	if err == nil {
		err = state.Internalize(equality, true)
	}
	if err != nil {
		delete(a.processed, pair)
		a.trail = a.trail[:len(a.trail)-1]
		return err
	}
	a.numAxioms++
	if state.searching && state.scopeLevel > state.searchBaseLevel &&
		leftNode.InternalScopeLevel() <= state.searchBaseLevel && rightNode.InternalScopeLevel() <= state.searchBaseLevel {
		a.restart = append(a.restart, pair)
	}
	return nil
}

func (a *smtArithmeticEqualityAdapter) InitializeSearch() error {
	if a == nil || a.owner == nil {
		return ast.ErrASTHandle
	}
	a.restart = nil
	return nil
}

func (a *smtArithmeticEqualityAdapter) Restart() error {
	if a == nil || a.owner == nil {
		return ast.ErrASTHandle
	}
	state, err := a.owner.State()
	if err != nil {
		return err
	}
	pairs := append([]smtArithmeticEqualityPair(nil), a.restart...)
	a.restart = nil
	for _, pair := range pairs {
		if state.inconsistent {
			break
		}
		left := state.context.enodes[pair.first]
		right := state.context.enodes[pair.second]
		if left == nil || right == nil {
			return ast.ErrASTDomain
		}
		leftVar := left.TheoryVar(a.owner.FamilyID())
		rightVar := right.TheoryVar(a.owner.FamilyID())
		if leftVar == SMTNullTheoryVar || rightVar == SMTNullTheoryVar {
			return ast.ErrASTDomain
		}
		if err = a.MkAxioms(leftVar, rightVar); err != nil {
			return err
		}
	}
	return nil
}

func (a *smtArithmeticEqualityAdapter) PushScope() error {
	if a == nil || a.owner == nil {
		return ast.ErrASTHandle
	}
	a.limits = append(a.limits, len(a.trail))
	return nil
}

func (a *smtArithmeticEqualityAdapter) PopScopes(count uint32) error {
	if a == nil || count == 0 || int(count) > len(a.limits) {
		return ast.ErrASTDomain
	}
	level := len(a.limits) - int(count)
	limit := a.limits[level]
	for index := len(a.trail); index > limit; index-- {
		delete(a.processed, a.trail[index-1])
	}
	a.trail = a.trail[:limit]
	a.limits = a.limits[:level]
	return nil
}

func (a *smtArithmeticEqualityAdapter) NumAxioms() uint64 {
	if a == nil {
		return 0
	}
	return a.numAxioms
}
