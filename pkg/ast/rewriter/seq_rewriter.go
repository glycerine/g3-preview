// Source: src/ast/rewriter/seq_rewriter.cpp:483-519.
package rewriter

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
)

// SequenceRewriter borrows its AST manager and implements the retained theory
// reductions independently of traversal.
type SequenceRewriter struct {
	manager *ast.ASTManager
}

func NewSequenceRewriter(m *ast.ASTManager) (*SequenceRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	return &SequenceRewriter{manager: m}, nil
}

func (r *SequenceRewriter) open() error {
	if r == nil || r.manager == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

func (r *SequenceRewriter) assignBoolean(value bool, out ast.ASTRef) (RewriteStatus, error) {
	var result ast.ASTHandle
	var err error
	if value {
		result, err = r.manager.MkTrue()
	} else {
		result, err = r.manager.MkFalse()
	}
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) integerValue(h ast.ASTHandle) (*big.Int, bool, error) {
	text, integer, matched, err := r.manager.ArithmeticNumeralText(h)
	if err != nil || !matched || !integer {
		return nil, false, err
	}
	value, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return nil, false, ast.ErrASTDomain
	}
	return value, true, nil
}

func (r *SequenceRewriter) assignInteger(value *big.Int, out ast.ASTRef) (status RewriteStatus, err error) {
	service := r.manager.NumeralService()
	if service == nil || value == nil {
		return RewriteFailed, ast.ErrASTHandle
	}
	numeral, err := service.NewValue()
	if err != nil {
		return RewriteFailed, err
	}
	defer func() { err = errors.Join(err, service.Del(&numeral)) }()
	if err = service.SetString(&numeral, value.String()); err != nil {
		return RewriteFailed, err
	}
	result, err := r.manager.ArithmeticNumeral(&numeral, true)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, nil
}

func (r *SequenceRewriter) assignEmptyLike(h ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	sort, err := r.manager.GetSort(h)
	if err != nil {
		return RewriteFailed, err
	}
	empty, err := r.manager.MkSequenceEmpty(sort)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, empty)
}

func (r *SequenceRewriter) RewriteLength(a ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	leaves, err := r.manager.SequenceFlattenConcat(a)
	if err != nil {
		return RewriteFailed, err
	}
	length := new(big.Int)
	for _, leaf := range leaves {
		text, stringValue, err := r.manager.SequenceStringValue(leaf)
		if err != nil {
			return RewriteFailed, err
		}
		if stringValue {
			length.Add(length, new(big.Int).SetUint64(uint64(len(text))))
			continue
		}
		unit, err := r.manager.IsSequenceUnit(leaf)
		if err != nil {
			return RewriteFailed, err
		}
		if !unit {
			return RewriteFailed, nil
		}
		length.Add(length, big.NewInt(1))
	}
	return r.assignInteger(length, out)
}

func (r *SequenceRewriter) RewriteExtract(a, position, length ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	pos, posNumeral, err := r.integerValue(position)
	if err != nil {
		return RewriteFailed, err
	}
	size, sizeNumeral, err := r.integerValue(length)
	if err != nil {
		return RewriteFailed, err
	}
	if posNumeral && pos.Sign() < 0 || sizeNumeral && size.Sign() <= 0 {
		return r.assignEmptyLike(a, out)
	}
	text, constant, err := r.manager.SequenceStringValue(a)
	if err != nil || !constant || !posNumeral || !sizeNumeral {
		return RewriteFailed, err
	}
	textLength := new(big.Int).SetUint64(uint64(len(text)))
	if pos.Cmp(textLength) >= 0 {
		return r.assignEmptyLike(a, out)
	}
	end := new(big.Int).Add(new(big.Int).Set(pos), size)
	if end.Cmp(textLength) > 0 {
		end.Set(textLength)
	}
	startIndex := int(pos.Uint64())
	endIndex := int(end.Uint64())
	result, err := r.manager.MkString(text[startIndex:endIndex])
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) RewriteAt(a, index ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	position, numeral, err := r.integerValue(index)
	if err != nil || !numeral {
		return RewriteFailed, err
	}
	if position.Sign() < 0 {
		return r.assignEmptyLike(a, out)
	}
	text, constant, err := r.manager.SequenceStringValue(a)
	if err != nil || !constant {
		return RewriteFailed, err
	}
	if !position.IsUint64() || position.Uint64() >= uint64(len(text)) {
		return r.assignEmptyLike(a, out)
	}
	result, err := r.manager.MkString(text[position.Uint64() : position.Uint64()+1])
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) RewriteIndex(a, needle, offset ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	position, numeral, err := r.integerValue(offset)
	if err != nil || !numeral {
		return RewriteFailed, err
	}
	if position.Sign() < 0 {
		return r.assignInteger(big.NewInt(-1), out)
	}
	base, baseString, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	pattern, patternString, err := r.manager.SequenceStringValue(needle)
	if err != nil {
		return RewriteFailed, err
	}
	if baseString && patternString {
		baseLength := new(big.Int).SetUint64(uint64(len(base)))
		if position.Cmp(baseLength) > 0 || position.Cmp(baseLength) == 0 && pattern != "" {
			return r.assignInteger(big.NewInt(-1), out)
		}
		start := position.Uint64()
		found := strings.Index(base[start:], pattern)
		if found < 0 {
			return r.assignInteger(big.NewInt(-1), out)
		}
		answer := new(big.Int).Add(position, new(big.Int).SetUint64(uint64(found)))
		return r.assignInteger(answer, out)
	}
	empty, err := r.manager.IsSequenceEmpty(needle)
	if err != nil {
		return RewriteFailed, err
	}
	if empty && position.Sign() == 0 {
		return RewriteDone, r.manager.SetRef(out, offset)
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) RewriteContains(a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	left, leftString, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	right, rightString, err := r.manager.SequenceStringValue(b)
	if err != nil || !leftString || !rightString {
		return RewriteFailed, err
	}
	return r.assignBoolean(strings.Contains(left, right), out)
}

func (r *SequenceRewriter) RewritePrefix(a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	left, leftString, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	right, rightString, err := r.manager.SequenceStringValue(b)
	if err != nil {
		return RewriteFailed, err
	}
	if leftString && rightString {
		return r.assignBoolean(strings.HasPrefix(right, left), out)
	}
	empty, err := r.manager.IsSequenceEmpty(a)
	if err != nil || !empty {
		return RewriteFailed, err
	}
	return r.assignBoolean(true, out)
}

func (r *SequenceRewriter) RewriteSuffix(a, b ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	if a == b {
		return r.assignBoolean(true, out)
	}
	emptyA, err := r.manager.IsSequenceEmpty(a)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyA {
		return r.assignBoolean(true, out)
	}
	emptyB, err := r.manager.IsSequenceEmpty(b)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyB {
		sort, err := r.manager.GetSort(a)
		if err != nil {
			return RewriteFailed, err
		}
		empty, err := r.manager.MkSequenceEmpty(sort)
		if err != nil {
			return RewriteFailed, err
		}
		emptyRef, err := r.manager.NewRef(empty)
		if err != nil {
			return RewriteFailed, err
		}
		defer func() { err = errors.Join(err, r.manager.DelRef(&emptyRef)) }()
		result, err := r.manager.MkEq(empty, a)
		if err != nil {
			return RewriteFailed, err
		}
		if err = r.manager.SetRef(out, result); err != nil {
			return RewriteFailed, err
		}
		return Rewrite3, nil
	}
	left, leftString, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	right, rightString, err := r.manager.SequenceStringValue(b)
	if err != nil || !leftString || !rightString {
		return RewriteFailed, err
	}
	return r.assignBoolean(strings.HasSuffix(right, left), out)
}

func (r *SequenceRewriter) RewriteReplace(a, b, replacement ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	base, baseString, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	pattern, patternString, err := r.manager.SequenceStringValue(b)
	if err != nil {
		return RewriteFailed, err
	}
	insert, replacementString, err := r.manager.SequenceStringValue(replacement)
	if err != nil {
		return RewriteFailed, err
	}
	if baseString && patternString && replacementString {
		result, err := r.manager.MkString(strings.Replace(base, pattern, insert, 1))
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	if b == replacement {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	if patternString && pattern == "" {
		result, err := r.manager.MkSequenceConcat(replacement, a)
		if err != nil {
			return RewriteFailed, err
		}
		return Rewrite1, r.manager.SetRef(out, result)
	}
	if baseString && base == "" {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	return RewriteFailed, nil
}

// RewriteUnit ports the byte-valued branch of seq_rewriter::mk_seq_unit.
func (r *SequenceRewriter) RewriteUnit(element ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	value, matched, err := r.manager.SequenceUnitByteElementValue(element)
	if err != nil || !matched {
		return RewriteFailed, err
	}
	result, err := r.manager.MkString(string([]byte{value}))
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

// RewriteConcat ports seq_rewriter::mk_seq_concat. The caller owns traversal
// and applies the returned revisit depth.
func (r *SequenceRewriter) RewriteConcat(a, b ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	s1, stringA, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat left string: %w", err)
	}
	s2, stringB, err := r.manager.SequenceStringValue(b)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat right string: %w", err)
	}
	if stringA && stringB {
		result, err := r.manager.MkString(s1 + s2)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	c, d, concatA, err := r.manager.SequenceConcatParts(a)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat left parts: %w", err)
	}
	if concatA {
		right, err := r.manager.MkSequenceConcat(d, b)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat suffix: %w", err)
		}
		rightRef, err := r.manager.NewRef(right)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat suffix reference: %w", err)
		}
		defer func() { err = errors.Join(err, r.manager.DelRef(&rightRef)) }()
		result, err := r.manager.MkSequenceConcat(c, right)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat reassociation: %w", err)
		}
		if err = r.manager.SetRef(out, result); err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat result: %w", err)
		}
		return Rewrite2, nil
	}
	emptyA, err := r.manager.IsSequenceEmpty(a)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyA {
		return RewriteDone, r.manager.SetRef(out, b)
	}
	emptyB, err := r.manager.IsSequenceEmpty(b)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyB {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	if stringA {
		c, d, concatB, err := r.manager.SequenceConcatParts(b)
		if err != nil {
			return RewriteFailed, err
		}
		if concatB {
			s2, stringC, err := r.manager.SequenceStringValue(c)
			if err != nil {
				return RewriteFailed, err
			}
			if stringC {
				prefix, err := r.manager.MkString(s1 + s2)
				if err != nil {
					return RewriteFailed, err
				}
				prefixRef, err := r.manager.NewRef(prefix)
				if err != nil {
					return RewriteFailed, err
				}
				defer func() { err = errors.Join(err, r.manager.DelRef(&prefixRef)) }()
				result, err := r.manager.MkSequenceConcat(prefix, d)
				if err != nil {
					return RewriteFailed, err
				}
				return RewriteDone, r.manager.SetRef(out, result)
			}
		}
	}
	return RewriteFailed, nil
}
