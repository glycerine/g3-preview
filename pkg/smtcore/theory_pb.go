// Source: src/smt/theory_pb.cpp and src/smt/theory_pb.h retained exact
// pseudo-Boolean atom semantics.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
)

type smtPseudoBooleanStatistics struct {
	predicates uint64
	clauses    uint64
}

// SMTPseudoBooleanTheory defines each reached pseudo-Boolean atom with exact
// SAT clauses. Each clause excludes one Boolean argument valuation paired with
// the wrong atom value.
type SMTPseudoBooleanTheory struct {
	*SMTTheoryBase
	statistics smtPseudoBooleanStatistics
}

func NewSMTPseudoBooleanTheory(family ast.FamilyID) (*SMTPseudoBooleanTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTPseudoBooleanTheory{SMTTheoryBase: base}, nil
}

func (t *SMTPseudoBooleanTheory) operation(expression ast.ASTHandle) (int32, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return 0, false, err
	}
	op, err := state.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func smtPseudoBooleanRational(manager *ast.ASTManager, value unumeral.OwnedRational) (*big.Rat, error) {
	service := manager.NumeralService()
	text, textErr := service.String(&value)
	err := errors.Join(textErr, service.Del(&value))
	if err != nil {
		return nil, err
	}
	result, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, ast.ErrASTDomain
	}
	return result, nil
}

func smtPseudoBooleanRelation(op int32, sum, bound *big.Rat) (bool, error) {
	comparison := sum.Cmp(bound)
	switch op {
	case ast.ASTPBOpAtMost, ast.ASTPBOpLE:
		return comparison <= 0, nil
	case ast.ASTPBOpAtLeast, ast.ASTPBOpGE:
		return comparison >= 0, nil
	case ast.ASTPBOpEQ:
		return comparison == 0, nil
	default:
		return false, ast.ErrASTDomain
	}
}

func (t *SMTPseudoBooleanTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	op, own, err := t.operation(expression)
	if err != nil || !own || op < ast.ASTPBOpAtMost || op > ast.ASTPBOpAuxBool {
		return false, err
	}
	state, _ := t.State()
	variable, found := state.boolVar[expression]
	if !found {
		variable, err = state.MkBoolVar(expression)
		if err != nil {
			return false, err
		}
	}
	data, err := state.BooleanData(variable)
	if err != nil {
		return false, err
	}
	if !data.IsTheoryAtom() {
		if err = state.SetVariableTheory(variable, t.FamilyID()); err != nil {
			return false, err
		}
	}
	if op == ast.ASTPBOpAuxBool || found {
		return true, nil
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	arguments := make([]SMTLiteral, count)
	coefficients := make([]*big.Rat, count)
	for i := uint32(0); i < count; i++ {
		argument, argumentErr := state.manager.GetArg(expression, i)
		if argumentErr != nil {
			return false, argumentErr
		}
		if argumentErr = state.Internalize(argument, true); argumentErr != nil {
			return false, argumentErr
		}
		arguments[i], argumentErr = state.Literal(argument)
		if argumentErr != nil {
			return false, argumentErr
		}
		coefficient, coefficientErr := state.manager.PseudoBooleanCoefficient(expression, i)
		if coefficientErr != nil {
			return false, coefficientErr
		}
		coefficients[i], coefficientErr = smtPseudoBooleanRational(state.manager, coefficient)
		if coefficientErr != nil {
			return false, coefficientErr
		}
	}
	boundValue, err := state.manager.PseudoBooleanBound(expression)
	if err != nil {
		return false, err
	}
	bound, err := smtPseudoBooleanRational(state.manager, boundValue)
	if err != nil {
		return false, err
	}
	selected := make([]bool, count)
	atom := NewSMTLiteral(variable, false)
	var emit func(int) error
	emit = func(index int) error {
		if index != len(selected) {
			selected[index] = false
			if branchErr := emit(index + 1); branchErr != nil {
				return branchErr
			}
			selected[index] = true
			return emit(index + 1)
		}
		sum := new(big.Rat)
		for i, yes := range selected {
			if yes {
				sum.Add(sum, coefficients[i])
			}
		}
		holds, relationErr := smtPseudoBooleanRelation(op, sum, bound)
		if relationErr != nil {
			return relationErr
		}
		clause := make([]SMTLiteral, 0, len(arguments)+1)
		for i, literal := range arguments {
			if selected[i] {
				literal = literal.Negated()
			}
			clause = append(clause, literal)
		}
		if !holds {
			clause = append(clause, atom.Negated())
		} else {
			clause = append(clause, atom)
		}
		if relationErr = state.addTheoryAxiom(t.FamilyID(), clause); relationErr != nil {
			return relationErr
		}
		t.statistics.clauses++
		return nil
	}
	if err = emit(0); err != nil {
		return false, err
	}
	t.statistics.predicates++
	return true, nil
}

func (*SMTPseudoBooleanTheory) InternalizeTerm(ast.ASTHandle) (bool, error)        { return false, nil }
func (*SMTPseudoBooleanTheory) InternalizeEquality(ast.ASTHandle, int32) error     { return nil }
func (*SMTPseudoBooleanTheory) ApplySortConstraint(*SMTEnode, ast.ASTHandle) error { return nil }
func (*SMTPseudoBooleanTheory) UseDisequalities() bool                             { return false }
func (*SMTPseudoBooleanTheory) NewEquality(SMTTheoryVar, SMTTheoryVar) error       { return nil }
func (*SMTPseudoBooleanTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error {
	return ast.ErrASTDomain
}
func (*SMTPseudoBooleanTheory) Assign(int32, bool) error { return nil }
func (*SMTPseudoBooleanTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	return SMTFinalCheckDone, nil
}
func (*SMTPseudoBooleanTheory) Name() string { return "pb" }

func (t *SMTPseudoBooleanTheory) CollectStatistics(target *statistics.Statistics) error {
	if t == nil || target == nil {
		return ast.ErrASTHandle
	}
	target.UpdateUint64("pb predicates", t.statistics.predicates)
	target.UpdateUint64("pb clauses", t.statistics.clauses)
	return nil
}

func (t *SMTPseudoBooleanTheory) ResetStatistics() {
	if t != nil {
		t.statistics = smtPseudoBooleanStatistics{}
	}
}
