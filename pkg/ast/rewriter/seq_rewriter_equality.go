// Source: src/ast/rewriter/seq_rewriter.cpp:1572-1931.
package rewriter

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type sequenceEqualityReduction struct {
	rewriter *SequenceRewriter
	refs     []ast.ASTRef
	left     []ast.ASTHandle
	right    []ast.ASTHandle
}

func (s *sequenceEqualityReduction) close() (err error) {
	for i := len(s.refs) - 1; i >= 0; i-- {
		err = errors.Join(err, s.rewriter.manager.DelRef(&s.refs[i]))
	}
	return err
}

func (s *sequenceEqualityReduction) hold(h ast.ASTHandle) error {
	ref, err := s.rewriter.manager.NewRef(h)
	if err != nil {
		return err
	}
	s.refs = append(s.refs, ref)
	return nil
}

func (s *sequenceEqualityReduction) add(left, right ast.ASTHandle) {
	s.left = append(s.left, left)
	s.right = append(s.right, right)
}

func (s *sequenceEqualityReduction) stringValue(text string) (ast.ASTHandle, error) {
	h, err := s.rewriter.manager.MkString(text)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = s.hold(h); err != nil {
		return ast.ASTHandle{}, err
	}
	return h, nil
}

func (s *sequenceEqualityReduction) character(value byte) (ast.ASTHandle, error) {
	h, err := s.rewriter.manager.BitVectorNumeralUnsigned(uint32(value), 8)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = s.hold(h); err != nil {
		return ast.ASTHandle{}, err
	}
	return h, nil
}

func (s *sequenceEqualityReduction) concat(parts []ast.ASTHandle, like ast.ASTHandle) (ast.ASTHandle, error) {
	ref, h, err := s.rewriter.makeSequencePartsRef(parts, like)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	s.refs = append(s.refs, ref)
	return h, nil
}

func (s *sequenceEqualityReduction) setEmpty(parts []ast.ASTHandle, all bool) (bool, error) {
	var empty ast.ASTHandle
	for _, part := range parts {
		unit, err := s.rewriter.manager.IsSequenceUnit(part)
		if err != nil {
			return false, err
		}
		if unit {
			if all {
				return false, nil
			}
			continue
		}
		isEmpty, err := s.rewriter.manager.IsSequenceEmpty(part)
		if err != nil {
			return false, err
		}
		if isEmpty {
			continue
		}
		text, stringValue, err := s.rewriter.manager.SequenceStringValue(part)
		if err != nil {
			return false, err
		}
		if stringValue {
			if all && text != "" {
				return false, nil
			}
			continue
		}
		if empty == (ast.ASTHandle{}) {
			sort, err := s.rewriter.manager.GetSort(part)
			if err != nil {
				return false, err
			}
			empty, err = s.rewriter.manager.MkSequenceEmpty(sort)
			if err != nil {
				return false, err
			}
			if err = s.hold(empty); err != nil {
				return false, err
			}
		}
		s.add(empty, part)
	}
	return true, nil
}

func (s *sequenceEqualityReduction) publish(out ast.ASTRef) (status RewriteStatus, err error) {
	if len(s.left) != len(s.right) {
		return RewriteFailed, ast.ErrASTDomain
	}
	equalities := make([]ast.ASTHandle, 0, len(s.left))
	for i := range s.left {
		equality, err := s.rewriter.manager.MkEq(s.left[i], s.right[i])
		if err != nil {
			return RewriteFailed, err
		}
		if err = s.hold(equality); err != nil {
			return RewriteFailed, err
		}
		equalities = append(equalities, equality)
	}
	result, err := s.rewriter.manager.MkAndArray(equalities)
	if err != nil {
		return RewriteFailed, err
	}
	if err = s.rewriter.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return Rewrite3, nil
}

// RewriteEquality ports the boundary cancellation, concrete String splitting,
// unit alignment, and empty-residual portion of seq_rewriter::mk_eq_core.
func (r *SequenceRewriter) RewriteEquality(left, right ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	ls, err := r.manager.SequenceFlattenConcat(left)
	if err != nil {
		return RewriteFailed, err
	}
	rs, err := r.manager.SequenceFlattenConcat(right)
	if err != nil {
		return RewriteFailed, err
	}
	state := sequenceEqualityReduction{rewriter: r}
	defer func() { err = errors.Join(err, state.close()) }()
	changed := false
	localChange := false

	for len(ls) > 0 && len(rs) > 0 {
		l, rr := ls[len(ls)-1], rs[len(rs)-1]
		_, rightUnit, e := r.manager.SequenceUnitArgument(rr)
		if e != nil {
			return RewriteFailed, e
		}
		_, leftString, e := r.manager.SequenceStringValue(l)
		if e != nil {
			return RewriteFailed, e
		}
		if rightUnit && leftString {
			ls, rs = rs, ls
			l, rr = ls[len(ls)-1], rs[len(rs)-1]
		}
		if l == rr {
			ls, rs = ls[:len(ls)-1], rs[:len(rs)-1]
			changed, localChange = true, true
			continue
		}
		leftElement, leftUnit, e := r.manager.SequenceUnitArgument(l)
		if e != nil {
			return RewriteFailed, e
		}
		rightElement, rightUnit, e := r.manager.SequenceUnitArgument(rr)
		if e != nil {
			return RewriteFailed, e
		}
		if leftUnit && rightUnit {
			distinct, e := r.manager.AreDistinct(leftElement, rightElement)
			if e != nil {
				return RewriteFailed, e
			}
			if distinct {
				return r.assignBoolean(false, out)
			}
			state.add(leftElement, rightElement)
			ls, rs = ls[:len(ls)-1], rs[:len(rs)-1]
			changed, localChange = true, true
			continue
		}
		text, stringValue, e := r.manager.SequenceStringValue(rr)
		if e != nil {
			return RewriteFailed, e
		}
		if !leftUnit || !stringValue || text == "" {
			break
		}
		character, e := state.character(text[len(text)-1])
		if e != nil {
			return RewriteFailed, e
		}
		state.add(character, leftElement)
		ls = ls[:len(ls)-1]
		if len(text) == 1 {
			rs = rs[:len(rs)-1]
		} else {
			rs[len(rs)-1], e = state.stringValue(text[:len(text)-1])
			if e != nil {
				return RewriteFailed, e
			}
		}
		changed, localChange = true, true
	}

	headLeft, headRight := 0, 0
	for headLeft < len(ls) && headRight < len(rs) {
		l, rr := ls[headLeft], rs[headRight]
		_, rightUnit, e := r.manager.SequenceUnitArgument(rr)
		if e != nil {
			return RewriteFailed, e
		}
		_, leftString, e := r.manager.SequenceStringValue(l)
		if e != nil {
			return RewriteFailed, e
		}
		if rightUnit && leftString {
			ls, rs = rs, ls
			headLeft, headRight = headRight, headLeft
			l, rr = ls[headLeft], rs[headRight]
		}
		if l == rr {
			headLeft++
			headRight++
			changed, localChange = true, true
			continue
		}
		leftElement, leftUnit, e := r.manager.SequenceUnitArgument(l)
		if e != nil {
			return RewriteFailed, e
		}
		rightElement, rightUnit, e := r.manager.SequenceUnitArgument(rr)
		if e != nil {
			return RewriteFailed, e
		}
		if leftUnit && rightUnit {
			distinct, e := r.manager.AreDistinct(leftElement, rightElement)
			if e != nil {
				return RewriteFailed, e
			}
			if distinct {
				return r.assignBoolean(false, out)
			}
			state.add(leftElement, rightElement)
			headLeft++
			headRight++
			changed, localChange = true, true
			continue
		}
		text, stringValue, e := r.manager.SequenceStringValue(rr)
		if e != nil {
			return RewriteFailed, e
		}
		if !leftUnit || !stringValue || text == "" {
			break
		}
		character, e := state.character(text[0])
		if e != nil {
			return RewriteFailed, e
		}
		state.add(character, leftElement)
		headLeft++
		if len(text) == 1 {
			headRight++
		} else {
			rs[headRight], e = state.stringValue(text[1:])
			if e != nil {
				return RewriteFailed, e
			}
		}
		changed, localChange = true, true
	}

	for headLeft < len(ls) && headRight < len(rs) {
		leftText, leftString, e := r.manager.SequenceStringValue(ls[headLeft])
		if e != nil {
			return RewriteFailed, e
		}
		rightText, rightString, e := r.manager.SequenceStringValue(rs[headRight])
		if e != nil {
			return RewriteFailed, e
		}
		if !leftString || !rightString {
			break
		}
		common := len(leftText)
		if len(rightText) < common {
			common = len(rightText)
		}
		if leftText[:common] != rightText[:common] {
			return r.assignBoolean(false, out)
		}
		if common == len(leftText) {
			headLeft++
		} else {
			ls[headLeft], e = state.stringValue(leftText[common:])
			if e != nil {
				return RewriteFailed, e
			}
		}
		if common == len(rightText) {
			headRight++
		} else {
			rs[headRight], e = state.stringValue(rightText[common:])
			if e != nil {
				return RewriteFailed, e
			}
		}
		changed, localChange = true, true
	}

	for headLeft < len(ls) && headRight < len(rs) {
		leftText, leftString, e := r.manager.SequenceStringValue(ls[len(ls)-1])
		if e != nil {
			return RewriteFailed, e
		}
		rightText, rightString, e := r.manager.SequenceStringValue(rs[len(rs)-1])
		if e != nil {
			return RewriteFailed, e
		}
		if !leftString || !rightString {
			break
		}
		common := len(leftText)
		if len(rightText) < common {
			common = len(rightText)
		}
		if leftText[len(leftText)-common:] != rightText[len(rightText)-common:] {
			return r.assignBoolean(false, out)
		}
		ls, rs = ls[:len(ls)-1], rs[:len(rs)-1]
		if common < len(leftText) {
			remaining, e := state.stringValue(leftText[:len(leftText)-common])
			if e != nil {
				return RewriteFailed, e
			}
			ls = append(ls, remaining)
		}
		if common < len(rightText) {
			remaining, e := state.stringValue(rightText[:len(rightText)-common])
			if e != nil {
				return RewriteFailed, e
			}
			rs = append(rs, remaining)
		}
		changed, localChange = true, true
	}

	remainingLeft, remainingRight := ls[headLeft:], rs[headRight:]
	if len(remainingLeft) == 0 && len(remainingRight) == 0 {
		if !changed {
			return RewriteFailed, nil
		}
		return state.publish(out)
	}
	if len(remainingLeft) == 0 || len(remainingRight) == 0 {
		parts := remainingRight
		if len(parts) == 0 {
			parts = remainingLeft
		}
		before := len(state.left)
		satisfiable, e := state.setEmpty(parts, true)
		if e != nil {
			return RewriteFailed, e
		}
		if !satisfiable {
			return r.assignBoolean(false, out)
		}
		if len(parts) == 1 && !localChange {
			state.left = state.left[:before]
			state.right = state.right[:before]
			return RewriteFailed, nil
		}
		changed = changed || len(parts) > 1
		if !changed {
			return RewriteFailed, nil
		}
		return state.publish(out)
	}
	if !localChange {
		return RewriteFailed, nil
	}
	leftResidual, e := state.concat(remainingLeft, left)
	if e != nil {
		return RewriteFailed, e
	}
	rightResidual, e := state.concat(remainingRight, right)
	if e != nil {
		return RewriteFailed, e
	}
	state.add(leftResidual, rightResidual)
	return state.publish(out)
}
