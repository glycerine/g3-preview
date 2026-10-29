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
