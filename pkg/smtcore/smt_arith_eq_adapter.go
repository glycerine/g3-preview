// Source: src/smt/arith_eq_adapter.h and src/smt/arith_eq_adapter.cpp retained
// arithmetic equality-to-bound adapter.
package smtcore

import (
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

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
	lazy      bool
	trace     *diagnostic.TraceTags
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

func (a *smtArithmeticEqualityAdapter) pair(left, right *SMTEnode) (smtArithmeticEqualityPair, error) {
	first, err := left.OwnerID()
	if err != nil {
		return smtArithmeticEqualityPair{}, err
	}
	second, err := right.OwnerID()
	if err != nil {
		return smtArithmeticEqualityPair{}, err
	}
	if first > second {
		left, right = right, left
	}
	return smtArithmeticEqualityPair{first: left.Owner(), second: right.Owner()}, nil
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
	pair, err := a.pair(leftNode, rightNode)
	if err != nil {
		return err
	}
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
	if !a.lazy && state.searching && state.scopeLevel > state.searchBaseLevel &&
		leftNode.InternalScopeLevel() <= state.searchBaseLevel && rightNode.InternalScopeLevel() <= state.searchBaseLevel {
		a.restart = append(a.restart, pair)
	}
	return nil
}

func (a *smtArithmeticEqualityAdapter) AddRelevancy(first, second ast.ASTHandle, targets ...ast.ASTHandle) error {
	if a == nil || a.owner == nil {
		return ast.ErrASTHandle
	}
	state, err := a.owner.State()
	if err != nil {
		return err
	}
	if state.relevancy == nil {
		return nil
	}
	for _, target := range targets {
		if err = state.relevancy.AddPairDependency(first, second, target); err != nil {
			return err
		}
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

func (a *smtArithmeticEqualityAdapter) ResetStatistics() {
	if a != nil {
		a.numAxioms = 0
	}
}

// Source arith_eq_adapter.cpp:269-279. Its native hash-table iteration order
// is unspecified; only each canonical pair's literal record is prescribed.
// Context enodes retain the borrowed AST handles through the pair's scope.
func (a *smtArithmeticEqualityAdapter) DisplayAlreadyProcessed(out io.Writer) error {
	if a == nil || a.owner == nil || out == nil {
		return ast.ErrASTHandle
	}
	state, err := a.owner.State()
	if err != nil {
		return err
	}
	for pair := range a.processed {
		first, err := state.manager.GetASTID(pair.first)
		if err != nil {
			return err
		}
		second, err := state.manager.GetASTID(pair.second)
		if err != nil {
			return err
		}
		if err = smtWriteTheoryText(out, fmt.Sprintf("eq_adapter: #%d #%d\n", first, second)); err != nil {
			return err
		}
	}
	return nil
}

// Source theory_arith_core.h:1475, before propagate_core in final_check_eh.
func (a *smtArithmeticEqualityAdapter) traceFinalCheckPairs() error {
	if a == nil {
		return ast.ErrASTHandle
	}
	tags := a.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	var operandErr error
	err := tags.EmitConditional("arith_eq_adapter_info", "final_check_eh", "src/smt/theory_arith_core.h", 1475, func() bool { return true }, func(out io.Writer) error {
		operandErr = a.DisplayAlreadyProcessed(assertedTraceWriter{out})
		return operandErr
	})
	if operandErr != nil {
		return operandErr
	}
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}
