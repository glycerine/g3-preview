package pdr

// Source: the Farkas-lemma recognition and constraint-combination paths in
// src/muz/pdr/pdr_farkas_learner.h and pdr_farkas_learner.cpp.

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// FarkasLearner owns the arithmetic utility used to combine proof premises.
// Returned ASTRef values are owned by the caller and must be released with
// ASTManager.DelRef.
type FarkasLearner struct {
	manager             *ast.ASTManager
	utility             *smtcore.SMTFarkasUtil
	combineCoefficients bool
	closed              bool
}

func NewFarkasLearner(manager *ast.ASTManager, context *uparams.ParamsContext, globals *uparams.GlobalParamsService) (*FarkasLearner, error) {
	if manager == nil || context == nil || globals == nil {
		return nil, ast.ErrASTDomain
	}
	utility, err := smtcore.NewSMTFarkasUtil(manager, context, globals)
	if err != nil {
		return nil, err
	}
	return &FarkasLearner{manager: manager, utility: utility, combineCoefficients: true}, nil
}

func (f *FarkasLearner) open() error {
	if f == nil || f.closed || f.manager == nil || f.utility == nil {
		return ast.ErrASTHandle
	}
	return nil
}

func (f *FarkasLearner) Close() error {
	if f == nil {
		return ast.ErrASTHandle
	}
	if f.closed {
		return nil
	}
	f.closed = true
	err := f.utility.Close()
	f.utility = nil
	return err
}

// SetCombineCoefficients selects the source learner's coefficient-combination
// mode. When disabled, CombineConstraints returns the conjunction of the
// negated literals, represented by not(or(literals)).
func (f *FarkasLearner) SetCombineCoefficients(enabled bool) error {
	if err := f.open(); err != nil {
		return err
	}
	f.combineCoefficients = enabled
	return nil
}

func (f *FarkasLearner) parameterIsSymbol(parameter ast.ASTParameter, want string) (bool, error) {
	symbol, err := f.manager.ParameterSymbol(parameter)
	if err != nil {
		return false, err
	}
	text, present, err := f.manager.Symbols.BareString(symbol)
	return present && text == want, err
}

// IsFarkasLemma recognizes arithmetic theory lemmas carrying the source
// "arith", "farkas", and per-parent coefficient parameter layout.
func (f *FarkasLearner) IsFarkasLemma(proof ast.ASTHandle) (_ bool, resultErr error) {
	if err := f.open(); err != nil {
		return false, err
	}
	isTheoryLemma, err := f.manager.IsProofKind(proof, ast.ASTPrTheoryLemma)
	if err != nil || !isTheoryLemma {
		return false, err
	}
	declaration, err := f.manager.GetDecl(proof)
	if err != nil {
		return false, err
	}
	parameters, err := f.manager.GetDeclParametersCopy(declaration)
	if err != nil {
		return false, err
	}
	defer func() {
		for i := range parameters {
			resultErr = errors.Join(resultErr, f.manager.DelParameter(&parameters[i]))
		}
	}()
	parents, err := f.manager.GetProofNumParents(proof)
	if err != nil {
		return false, err
	}
	if len(parameters) < 2 || uint32(len(parameters)) < parents+2 {
		return false, nil
	}
	arithmetic, err := f.parameterIsSymbol(parameters[0], "arith")
	if err != nil || !arithmetic {
		return false, err
	}
	farkas, err := f.parameterIsSymbol(parameters[1], "farkas")
	return farkas, err
}

func (f *FarkasLearner) rationalParameter(parameter ast.ASTParameter) (_ *big.Rat, resultErr error) {
	service := f.manager.NumeralService()
	value, err := service.NewValue()
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, service.Del(&value)) }()
	matched, err := f.manager.ParameterRationalCopy(parameter, &value)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, ast.ErrASTDomain
	}
	text, err := service.String(&value)
	if err != nil {
		return nil, err
	}
	coefficient, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, ast.ErrASTDomain
	}
	return coefficient, nil
}

// FarkasPremises returns the borrowed fact of each proof parent and its
// coefficient. Coefficients are made nonnegative, matching the source
// learner's workaround before it combines constraints.
func (f *FarkasLearner) FarkasPremises(proof ast.ASTHandle) (_ []ast.ASTHandle, _ []*big.Rat, _ bool, resultErr error) {
	matched, err := f.IsFarkasLemma(proof)
	if err != nil || !matched {
		return nil, nil, matched, err
	}
	declaration, err := f.manager.GetDecl(proof)
	if err != nil {
		return nil, nil, false, err
	}
	parameters, err := f.manager.GetDeclParametersCopy(declaration)
	if err != nil {
		return nil, nil, false, err
	}
	defer func() {
		for i := range parameters {
			resultErr = errors.Join(resultErr, f.manager.DelParameter(&parameters[i]))
		}
	}()
	count, err := f.manager.GetProofNumParents(proof)
	if err != nil {
		return nil, nil, false, err
	}
	facts := make([]ast.ASTHandle, 0, count)
	coefficients := make([]*big.Rat, 0, count)
	for i := uint32(0); i < count; i++ {
		parent, err := f.manager.GetProofParent(proof, i)
		if err != nil {
			return nil, nil, false, err
		}
		fact, err := f.manager.GetProofFact(parent)
		if err != nil {
			return nil, nil, false, err
		}
		coefficient, err := f.rationalParameter(parameters[i+2])
		if err != nil {
			return nil, nil, false, err
		}
		coefficient.Abs(coefficient)
		facts = append(facts, fact)
		coefficients = append(coefficients, coefficient)
	}
	return facts, coefficients, true, nil
}

// CombineFarkasLemma performs the theory-lemma branch of get_lemmas after the
// proof traversal has classified each parent. A true bPure entry removes that
// B-derived parent from the arithmetic combination. Coefficients attached to
// the lemma conclusion are applied to the negated conclusion literals.
func (f *FarkasLearner) CombineFarkasLemma(proof ast.ASTHandle, bPure []bool) (_ ast.ASTRef, _ bool, resultErr error) {
	facts, coefficients, matched, err := f.FarkasPremises(proof)
	if err != nil || !matched {
		return ast.ASTRef{}, false, err
	}
	if len(bPure) != len(facts) {
		return ast.ASTRef{}, false, ast.ErrASTDomain
	}
	literals := make([]ast.ASTHandle, 0, len(facts))
	selectedCoefficients := make([]*big.Rat, 0, len(coefficients))
	bPureCount := 0
	for i := range facts {
		if bPure[i] {
			bPureCount++
			continue
		}
		literals = append(literals, facts[i])
		selectedCoefficients = append(selectedCoefficients, coefficients[i])
	}
	if bPureCount == 0 {
		return ast.ASTRef{}, false, nil
	}

	declaration, err := f.manager.GetDecl(proof)
	if err != nil {
		return ast.ASTRef{}, false, err
	}
	parameters, err := f.manager.GetDeclParametersCopy(declaration)
	if err != nil {
		return ast.ASTRef{}, false, err
	}
	defer func() {
		for i := range parameters {
			resultErr = errors.Join(resultErr, f.manager.DelParameter(&parameters[i]))
		}
	}()
	parentParameterEnd := len(facts) + 2
	if len(parameters) > parentParameterEnd {
		fact, err := f.manager.GetProofFact(proof)
		if err != nil {
			return ast.ASTRef{}, false, err
		}
		conclusionLiterals := []ast.ASTHandle{fact}
		family, err := f.manager.GetFamilyID(fact)
		if err != nil {
			return ast.ASTRef{}, false, err
		}
		kind, err := f.manager.GetDeclKind(fact)
		if err != nil {
			return ast.ASTRef{}, false, err
		}
		if family == f.manager.BasicFamilyID() && kind == ast.ASTOpOr {
			count, err := f.manager.GetNumArgs(fact)
			if err != nil {
				return ast.ASTRef{}, false, err
			}
			conclusionLiterals = make([]ast.ASTHandle, count)
			for i := uint32(0); i < count; i++ {
				conclusionLiterals[i], err = f.manager.GetArg(fact, i)
				if err != nil {
					return ast.ASTRef{}, false, err
				}
			}
		}
		if len(parameters) != parentParameterEnd+len(conclusionLiterals) {
			return ast.ASTRef{}, false, ast.ErrASTDomain
		}
		negationReferences := make([]ast.ASTRef, 0, len(conclusionLiterals))
		defer func() {
			for i := range negationReferences {
				resultErr = errors.Join(resultErr, f.manager.DelRef(&negationReferences[i]))
			}
		}()
		for i, literal := range conclusionLiterals {
			negation, err := f.manager.MkNot(literal)
			if err != nil {
				return ast.ASTRef{}, false, err
			}
			negationReference, err := f.manager.NewRef(negation)
			if err != nil {
				return ast.ASTRef{}, false, err
			}
			negationReferences = append(negationReferences, negationReference)
			coefficient, err := f.rationalParameter(parameters[parentParameterEnd+i])
			if err != nil {
				return ast.ASTRef{}, false, err
			}
			coefficient.Abs(coefficient)
			literals = append(literals, negation)
			selectedCoefficients = append(selectedCoefficients, coefficient)
		}
	}
	if len(literals) == 0 {
		return ast.ASTRef{}, false, ast.ErrASTDomain
	}
	combined, err := f.CombineConstraints(literals, selectedCoefficients)
	return combined, err == nil, err
}

// CombineConstraints applies the source learner's two combination modes.
func (f *FarkasLearner) CombineConstraints(literals []ast.ASTHandle, coefficients []*big.Rat) (_ ast.ASTRef, resultErr error) {
	if err := f.open(); err != nil {
		return ast.ASTRef{}, err
	}
	if len(literals) != len(coefficients) {
		return ast.ASTRef{}, ast.ErrASTDomain
	}
	if f.combineCoefficients {
		if err := f.utility.Reset(); err != nil {
			return ast.ASTRef{}, err
		}
		for i := range literals {
			if coefficients[i] == nil {
				return ast.ASTRef{}, ast.ErrASTDomain
			}
			if err := f.utility.Add(coefficients[i], literals[i]); err != nil {
				return ast.ASTRef{}, err
			}
		}
		out, err := f.manager.NewRef(ast.ASTHandle{})
		if err != nil {
			return ast.ASTRef{}, err
		}
		if err = f.utility.Get(out); err != nil {
			resultErr = errors.Join(err, f.manager.DelRef(&out))
			return ast.ASTRef{}, resultErr
		}
		return out, nil
	}

	disjunction, err := f.manager.MkOrArray(literals)
	if err != nil {
		return ast.ASTRef{}, err
	}
	disjunctionRef, err := f.manager.NewRef(disjunction)
	if err != nil {
		return ast.ASTRef{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, f.manager.DelRef(&disjunctionRef)) }()
	negation, err := f.manager.MkNot(disjunction)
	if err != nil {
		return ast.ASTRef{}, err
	}
	return f.manager.NewRef(negation)
}
