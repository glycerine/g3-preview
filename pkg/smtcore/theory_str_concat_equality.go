// Source: src/smt/theory_str.cpp infer_len_concat_arg,
// infer_len_concat_equality and the quick paths of simplify_concat_equality.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTStringTheory) inferStringConcatArgument(concat ast.ASTHandle, total *big.Int) (result error) {
	if total.Sign() < 0 {
		return nil
	}
	m := t.state.manager
	first, second, isConcat, err := m.SequenceConcatParts(concat)
	if err != nil || !isConcat {
		return err
	}
	left, lk, err := t.lengthValue(first)
	if err != nil {
		return err
	}
	right, rk, err := t.lengthValue(second)
	if err != nil || lk == rk {
		return err
	}
	missing, known, value := first, second, right
	if lk {
		missing, known, value = second, first, left
	}
	difference := new(big.Int).Sub(total, value)
	if difference.Sign() < 0 {
		return nil
	}
	premises := smtStringLengthPremises{theory: t}
	defer func() { result = errors.Join(result, premises.close()) }()
	if err := premises.length(concat, total); err != nil {
		return err
	}
	if err := premises.length(known, value); err != nil {
		return err
	}
	conclusion := smtStringLengthPremises{theory: t}
	defer func() { result = errors.Join(result, conclusion.close()) }()
	if err := conclusion.length(missing, difference); err != nil {
		return err
	}
	return premises.imply(conclusion.values[0])
}

func (t *SMTStringTheory) inferStringConcatEquality(left, right ast.ASTHandle) error {
	total, known, err := t.lengthValue(left)
	if err != nil {
		return err
	}
	if !known {
		total, known, err = t.lengthValue(right)
		if err != nil {
			return err
		}
	}
	// Native infer_len_concat's assertion branch is unreachable with the
	// recursive get_len_value: known children imply a known concat sum.
	if !known {
		return nil
	}
	if err := t.inferStringConcatArgument(left, total); err != nil {
		return err
	}
	return t.inferStringConcatArgument(right, total)
}

// Only a unique pair is selected until overlap-aware pair selection and
// arrangement cut histories are ported. Native shared-concat groups skip
// step 1 entirely, even if they contain other distinct concat members.
func (t *SMTStringTheory) simplifyStringConcatGroups(left, right smtStringClassGroups) error {
	for _, l := range left.concats {
		for _, r := range right.concats {
			if l == r {
				return nil
			}
		}
	}
	if len(left.concats) == 1 && len(right.concats) == 1 {
		return t.simplifyStringConcatEquality(left.concats[0], right.concats[0])
	}
	return nil
}

func (t *SMTStringTheory) simplifyStringConcatEquality(left, right ast.ASTHandle) (result error) {
	m := t.state.manager
	x, y, lc, err := m.SequenceConcatParts(left)
	if err != nil {
		return err
	}
	a, b, rc, err := m.SequenceConcatParts(right)
	if err != nil || !lc || !rc {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return err
	}
	// Snapshot all four lengths before inference, as in the source.
	lengths := [4]*big.Int{}
	known := [4]bool{}
	for i, expression := range []ast.ASTHandle{x, y, a, b} {
		lengths[i], known[i], err = t.lengthValue(expression)
		if err != nil {
			return err
		}
	}
	if err := t.inferStringConcatEquality(left, right); err != nil {
		return err
	}
	premises := smtStringLengthPremises{theory: t}
	defer func() { result = errors.Join(result, premises.close()) }()
	if err := premises.equate(left, right); err != nil {
		return err
	}
	conclusions := smtStringLengthPremises{theory: t}
	defer func() { result = errors.Join(result, conclusions.close()) }()
	temporaries := smtStringLengthPremises{theory: t}
	defer func() { result = errors.Join(result, temporaries.close()) }()
	lengthEquality := func(first, second ast.ASTHandle, target *smtStringLengthPremises) error {
		l, err := m.MkSequenceLength(first)
		if err != nil {
			return err
		}
		if err := temporaries.add(l); err != nil {
			return err
		}
		r, err := m.MkSequenceLength(second)
		if err != nil {
			return err
		}
		if err := temporaries.add(r); err != nil {
			return err
		}
		return target.equate(l, r)
	}
	cancel := func(first, second ast.ASTHandle) error {
		if err := conclusions.equate(first, second); err != nil {
			return err
		}
		if err := lengthEquality(first, second, &conclusions); err != nil {
			return err
		}
		conclusion, err := m.MkAndArray(conclusions.values)
		if err != nil {
			return err
		}
		return premises.imply(conclusion)
	}
	if x == a {
		same, err := t.sameStringClass(y, b)
		if err != nil || same {
			return err
		}
		return cancel(y, b)
	}
	if y == b {
		same, err := t.sameStringClass(x, a)
		if err != nil || same {
			return err
		}
		return cancel(x, a)
	}
	for i, pair := range [][4]ast.ASTHandle{{x, a, y, b}, {y, b, x, a}} {
		same, err := t.sameStringClass(pair[0], pair[1])
		if err != nil {
			return err
		}
		if same {
			if i == 0 {
				otherSame, err := t.sameStringClass(pair[2], pair[3])
				if err != nil || otherSame {
					return err
				}
			}
			if err := premises.equate(pair[0], pair[1]); err != nil {
				return err
			}
			return cancel(pair[2], pair[3])
		}
	}
	for i, pair := range [][2]ast.ASTHandle{{x, a}, {y, b}} {
		if !known[i] || !known[i+2] || lengths[i].Cmp(lengths[i+2]) != 0 {
			continue
		}
		if err := lengthEquality(pair[0], pair[1], &premises); err != nil {
			return err
		}
		if err := conclusions.equate(x, a); err != nil {
			return err
		}
		if err := conclusions.equate(y, b); err != nil {
			return err
		}
		conclusion, err := m.MkAndArray(conclusions.values)
		if err != nil {
			return err
		}
		return premises.imply(conclusion)
	}
	// The following concat re-simplification and six arrangement families
	// remain pending; FinalCheck continues to report GiveUp.
	return nil
}
