package spacer

// Source: src/muz/spacer/spacer_generalizers.cpp
// lemma_bool_inductive_generalizer and src/muz/spacer/spacer_context.cpp
// pred_transformer::check_inductive.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (p *PredicateTransformer) checkInductiveCube(cube []ast.ASTHandle, level uint32) (_ []ast.ASTHandle, usesLevel uint32, inductive bool, resultErr error) {
	conjunction, err := p.manager.MkAndArray(cube)
	if err != nil {
		return nil, 0, false, err
	}
	states, err := p.manager.PushNot(conjunction)
	if err != nil {
		return nil, 0, false, err
	}
	head, err := p.Head()
	if err != nil {
		return nil, 0, false, err
	}
	assumptionRefs, err := p.makeAssumptions(head, states)
	if err != nil {
		return nil, 0, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, releaseNormalizeRefs(p.manager, assumptionRefs)) }()
	background := make([]ast.ASTHandle, 0, len(assumptionRefs)+1)
	for i := range assumptionRefs {
		handle, getErr := p.manager.RefGet(assumptionRefs[i])
		if getErr != nil {
			return nil, 0, false, getErr
		}
		background = append(background, handle)
	}
	extension, err := p.manager.RefGet(p.extendLiteral)
	if err != nil {
		return nil, 0, false, err
	}
	background = append(background, extension)
	check, err := p.solver.CheckAssumptions(cube, nil, PropCheckOptions{
		Background: background, NeedCore: true, SubsetBasedCore: true, Level: &level,
	})
	if err != nil {
		return nil, 0, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, check.Close(p.manager)) }()
	if check.Status != lbool.False {
		return nil, check.UsesLevel, false, nil
	}
	core := make([]ast.ASTHandle, len(check.Core))
	for i := range check.Core {
		core[i], err = p.manager.RefGet(check.Core[i])
		if err != nil {
			return nil, 0, false, err
		}
	}
	return core, check.UsesLevel, true, nil
}

func containsGeneralizedLiteral(values []ast.ASTHandle, value ast.ASTHandle) bool {
	for _, prior := range values {
		if prior == value {
			return true
		}
	}
	return false
}

// expandInductiveLiteral ports the live arithmetic branch of expand_literals.
// The datatype and bit-vector branches remain separate retained work.
func (p *PredicateTransformer) expandInductiveLiteral(literal ast.ASTHandle) ([]ast.ASTHandle, error) {
	family, err := p.manager.GetFamilyID(literal)
	if err != nil || family != p.manager.BasicFamilyID() {
		return []ast.ASTHandle{literal}, err
	}
	kind, err := p.manager.GetDeclKind(literal)
	if err != nil || kind != ast.ASTOpEq {
		return []ast.ASTHandle{literal}, err
	}
	left, err := p.manager.GetArg(literal, 0)
	if err != nil {
		return nil, err
	}
	right, err := p.manager.GetArg(literal, 1)
	if err != nil {
		return nil, err
	}
	sort, err := p.manager.GetSort(left)
	if err != nil {
		return nil, err
	}
	sortFamily, err := p.manager.GetFamilyID(sort)
	if err != nil {
		return nil, err
	}
	arithmetic, err := p.manager.ArithmeticFamilyID()
	if err != nil || sortFamily != arithmetic {
		return []ast.ASTHandle{literal}, err
	}
	le, err := p.manager.MkArithmeticLE(left, right)
	if err != nil {
		return nil, err
	}
	ge, err := p.manager.MkArithmeticGE(left, right)
	if err != nil {
		return nil, err
	}
	return []ast.ASTHandle{le, ge}, nil
}

func (p *PredicateTransformer) generalizeInductiveLemma(obligation *ProofObligation, lemma *Lemma) (*Lemma, error) {
	cube, err := lemma.Cube()
	if err != nil || len(cube) == 0 {
		return lemma, err
	}
	level, err := lemma.Level()
	if err != nil {
		return lemma, err
	}
	truth, err := p.manager.MkTrue()
	if err != nil {
		return lemma, err
	}
	processed := make([]ast.ASTHandle, 0, len(cube))
	dirty := false
	usesLevel := level
	for index := 0; index < len(cube); {
		literal := cube[index]
		cube[index] = truth
		var core []ast.ASTHandle
		var used uint32
		var inductive bool
		if len(cube) > 1 {
			core, used, inductive, err = p.checkInductiveCube(cube, level)
			if err != nil {
				return lemma, err
			}
			if inductive {
				cube = core
				usesLevel = used
				dirty = true
				index = 0
				for index < len(cube) && containsGeneralizedLiteral(processed, cube[index]) {
					index++
				}
				continue
			}
		}
		expanded, expandErr := p.expandInductiveLiteral(literal)
		if expandErr != nil {
			return lemma, expandErr
		}
		found := false
		if len(expanded) > 1 || expanded[0] != literal {
			for _, replacement := range expanded {
				cube[index] = replacement
				core, used, inductive, err = p.checkInductiveCube(cube, level)
				if err != nil {
					return lemma, err
				}
				if inductive {
					cube = core
					usesLevel = used
					dirty = true
					found = true
					processed = append(processed, replacement)
					index = 0
					for index < len(cube) && containsGeneralizedLiteral(processed, cube[index]) {
						index++
					}
					break
				}
			}
		}
		if found {
			continue
		}
		cube[index] = literal
		processed = append(processed, literal)
		index++
	}
	if !dirty {
		return lemma, nil
	}
	return NewLemmaFromProofObligation(obligation, cube, usesLevel)
}
