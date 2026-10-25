// Source: src/smt/theory_seq.cpp branch_variable, find_branch_candidate and
// assume_equality retained equality case splitting for non-fixed words.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// The source sequence theory owns a substitution-free th_rewriter. The
// kernel's preprocessor also carries assertion substitutions and cannot be
// borrowed for unconditional equality simplification here.
type smtSequenceRewriterOwner struct {
	context  *uparams.ParamsContext
	catalog  *uparams.GlobalParamsCatalog
	globals  *uparams.GlobalParamsService
	rewriter *rewriter.TheoryRewriter
}

func (r *smtSequenceRewriterOwner) close() (result error) {
	if r.rewriter != nil {
		result = errors.Join(result, r.rewriter.Close())
	}
	if r.globals != nil {
		result = errors.Join(result, r.globals.Close())
	}
	if r.catalog != nil {
		result = errors.Join(result, r.catalog.Close())
	}
	if r.context != nil {
		result = errors.Join(result, r.context.Close())
	}
	return result
}

func (t *SMTSequenceTheory) equalityRewriter() (_ *rewriter.TheoryRewriter, result error) {
	if t.rewrite != nil {
		return t.rewrite.rewriter, nil
	}
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	m := state.manager
	r := &smtSequenceRewriterOwner{}
	defer func() {
		if result != nil {
			result = errors.Join(result, r.close())
		}
	}()
	r.context, err = uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	if err != nil {
		return nil, err
	}
	r.catalog, err = uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	if err != nil {
		return nil, err
	}
	r.globals, err = uparams.NewGlobalParamsService(m.Symbols, r.context, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: r.catalog})
	if err != nil {
		return nil, err
	}
	if err = r.globals.InitializeSource(); err != nil {
		return nil, err
	}
	r.rewriter, err = rewriter.NewTheoryRewriter(m, r.context, r.globals, nil)
	if err != nil {
		return nil, err
	}
	t.rewrite = r
	return r.rewriter, nil
}

func (t *SMTSequenceTheory) assumeSequenceEquality(left, right ast.ASTHandle) (value lbool.LBool, result error) {
	state, err := t.State()
	if err != nil {
		return lbool.Undef, err
	}
	if t.exclusions.Contains(left, right) {
		return lbool.False, nil
	}
	equality, err := state.manager.MkEq(left, right)
	if err != nil {
		return lbool.Undef, err
	}
	original, err := state.manager.NewRef(equality)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { result = errors.Join(result, state.manager.DelRef(&original)) }()
	ref, err := state.manager.NewRef(equality)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { result = errors.Join(result, state.manager.DelRef(&ref)) }()
	rw, err := t.equalityRewriter()
	if err != nil {
		return lbool.Undef, err
	}
	if err = rw.Rewrite(equality, ref); err != nil {
		return lbool.Undef, err
	}
	rewritten, err := state.manager.RefGet(ref)
	if err != nil {
		return lbool.Undef, err
	}
	if state.manager.IsTrue(rewritten) {
		return lbool.True, nil
	}
	if state.manager.IsFalse(rewritten) {
		return lbool.False, nil
	}
	if err = state.Internalize(left, false); err != nil {
		return lbool.Undef, err
	}
	if err = state.Internalize(right, false); err != nil {
		return lbool.Undef, err
	}
	first, second := state.context.enodes[left], state.context.enodes[right]
	if first == nil || second == nil {
		return lbool.Undef, ast.ErrASTDomain
	}
	if first.Root() == second.Root() {
		return lbool.True, nil
	}
	if state.context.isDisequal(first, second) {
		return lbool.False, nil
	}
	if err = state.markRelevant(left); err != nil {
		return lbool.Undef, err
	}
	if err = state.markRelevant(right); err != nil {
		return lbool.Undef, err
	}
	// Sequence inherits the source base mk_eq_atom AST-ID order.
	firstID, err := first.OwnerID()
	if err != nil {
		return lbool.Undef, err
	}
	secondID, err := second.OwnerID()
	if err != nil {
		return lbool.Undef, err
	}
	if firstID > secondID {
		first, second = second, first
	}
	accepted, err := state.assumeEquality(first, second)
	if err != nil {
		return lbool.Undef, err
	}
	if !accepted {
		return lbool.False, nil
	}
	equality, err = state.manager.MkEq(first.Owner(), second.Owner())
	if err != nil {
		return lbool.Undef, err
	}
	literal, err := state.Literal(equality)
	if err != nil {
		return lbool.Undef, err
	}
	return state.Assignment(literal)
}

func (t *SMTSequenceTheory) wordHasUncoveredPart(parts []ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	for _, part := range parts {
		if variable, variableErr := t.sequenceVariable(part); variableErr != nil {
			return false, variableErr
		} else if variable {
			continue
		}
		if _, fixed, textErr := state.manager.SequenceStringValue(part); textErr != nil {
			return false, textErr
		} else if fixed {
			continue
		}
		if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
			return false, unitErr
		} else if !unit {
			return true, nil
		}
	}
	return false, nil
}

func (t *SMTSequenceTheory) sequenceWordsCanBeEqual(left, right []ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	index := 0
	for index < len(left) && index < len(right) {
		distinct, exact, distinctErr := t.concreteDistinct(left[index], right[index])
		if distinctErr != nil {
			return false, distinctErr
		}
		if exact && distinct {
			return false, nil
		}
		equal, _ := t.equalWordParts(left[index], right[index], nil)
		if !equal {
			break
		}
		index++
	}
	if index == len(right) {
		left, right = right, left
	}
	if index == len(left) && index < len(right) {
		for _, part := range right[index:] {
			if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
				return false, unitErr
			} else if unit {
				return false, nil
			}
		}
	}
	return true, nil
}

func (t *SMTSequenceTheory) branchVariablePrefixEquality(left, right []ast.ASTHandle, dependency *smtSequenceDependency) (progress bool, result error) {
	if len(left) == 0 || len(right) == 0 {
		return false, nil
	}
	state, err := t.State()
	if err != nil {
		return false, err
	}
	var candidateRefs []ast.ASTRef
	defer func() {
		for i := len(candidateRefs) - 1; i >= 0; i-- {
			result = errors.Join(result, state.manager.DelRef(&candidateRefs[i]))
		}
	}()
	for pass := 0; pass < 2; pass++ {
		variable, err := t.sequenceVariable(left[0])
		if err != nil {
			return false, err
		}
		uncovered, err := t.wordHasUncoveredPart(right)
		if err != nil {
			return false, err
		}
		allUnits := true
		for _, part := range right {
			if _, unit, unitErr := state.manager.SequenceUnitArgument(part); unitErr != nil {
				return false, unitErr
			} else if !unit {
				allUnits = false
				break
			}
		}
		if variable && (uncovered || allUnits) {
			sort, sortErr := state.manager.GetSort(left[0])
			if sortErr != nil {
				return false, sortErr
			}
			empty, makeErr := state.manager.MkSequenceEmpty(sort)
			if makeErr != nil {
				return false, makeErr
			}
			emptyRef, makeErr := state.manager.NewRef(empty)
			if makeErr != nil {
				return false, makeErr
			}
			candidateRefs = append(candidateRefs, emptyRef)
			possible, possibleErr := t.sequenceWordsCanBeEqual(left[1:], right)
			if possibleErr != nil {
				return false, possibleErr
			}
			if possible {
				assignment, assumeErr := t.assumeSequenceEquality(left[0], empty)
				if assumeErr != nil {
					return false, assumeErr
				}
				if assignment != lbool.False {
					return true, nil
				}
			}
			candidates := []ast.ASTHandle{empty}
			for index, part := range right {
				if part == left[0] {
					break
				}
				possible, possibleErr = t.sequenceWordsCanBeEqual(left[1:], right[index+1:])
				if possibleErr != nil {
					return false, possibleErr
				}
				if !possible {
					continue
				}
				prefix, prefixErr := t.makeWord(right[:index+1], sort)
				if prefixErr != nil {
					return false, prefixErr
				}
				prefixRef, prefixErr := state.manager.NewRef(prefix)
				if prefixErr != nil {
					return false, prefixErr
				}
				candidateRefs = append(candidateRefs, prefixRef)
				assignment, assumeErr := t.assumeSequenceEquality(left[0], prefix)
				if assumeErr != nil {
					return false, assumeErr
				}
				if assignment != lbool.False {
					return true, nil
				}
				candidates = append(candidates, prefix)
			}
			if allUnits {
				conflictDependency := dependency
				for _, candidate := range candidates {
					equality, equalityErr := state.manager.MkEq(left[0], candidate)
					if equalityErr != nil {
						return false, equalityErr
					}
					if equalityErr = state.Internalize(equality, false); equalityErr != nil {
						return false, equalityErr
					}
					literal, equalityErr := state.Literal(equality)
					if equalityErr != nil {
						return false, equalityErr
					}
					assignment, equalityErr := state.Assignment(literal)
					if equalityErr != nil {
						return false, equalityErr
					}
					if assignment != lbool.False {
						data, dataErr := state.BooleanData(literal.Var())
						if dataErr != nil {
							return false, dataErr
						}
						data.SetTrueFirst()
						return true, nil
					}
					conflictDependency = t.dependencies.Join(conflictDependency,
						t.dependencies.Leaf(smtSequenceAssumption{literal: literal.Negated()}))
				}
				if conflictErr := t.setDependencyConflict(conflictDependency); conflictErr != nil {
					return false, conflictErr
				}
				return true, nil
			}
		}
		left, right = right, left
	}
	return false, nil
}
