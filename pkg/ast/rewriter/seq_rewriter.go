// Source: src/ast/rewriter/seq_rewriter.cpp:483-519.
package rewriter

import (
	"errors"
	"fmt"
	"math"
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

func (r *SequenceRewriter) rationalValue(h ast.ASTHandle) (*big.Rat, bool, error) {
	text, _, matched, err := r.manager.ArithmeticNumeralText(h)
	if err != nil || !matched {
		return nil, false, err
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, false, ast.ErrASTDomain
	}
	return value, true, nil
}

func (r *SequenceRewriter) assignInteger(value *big.Int, out ast.ASTRef) (status RewriteStatus, err error) {
	result, err := r.makeInteger(value)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, nil
}

func (r *SequenceRewriter) makeInteger(value *big.Int) (result ast.ASTHandle, err error) {
	service := r.manager.NumeralService()
	if service == nil || value == nil {
		return result, ast.ErrASTHandle
	}
	numeral, err := service.NewValue()
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, service.Del(&numeral)) }()
	if err = service.SetString(&numeral, value.String()); err != nil {
		return result, err
	}
	return r.manager.ArithmeticNumeral(&numeral, true)
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

func (r *SequenceRewriter) RewriteLength(a ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	leaves, err := r.manager.SequenceFlattenConcat(a)
	if err != nil {
		return RewriteFailed, err
	}
	length := new(big.Int)
	unknown := make([]ast.ASTHandle, 0, len(leaves))
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
			unknown = append(unknown, leaf)
			continue
		}
		length.Add(length, big.NewInt(1))
	}
	if len(unknown) == 0 {
		return r.assignInteger(length, out)
	}
	if len(unknown) == 1 && length.Sign() == 0 && len(leaves) == 1 {
		return RewriteFailed, nil
	}
	terms := make([]ast.ASTHandle, 0, len(unknown)+1)
	refs := make([]ast.ASTRef, 0, len(unknown)+1)
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			err = errors.Join(err, r.manager.DelRef(&refs[i]))
		}
	}()
	for _, leaf := range unknown {
		term, e := r.manager.MkSequenceLength(leaf)
		if e != nil {
			return RewriteFailed, e
		}
		ref, e := r.manager.NewRef(term)
		if e != nil {
			return RewriteFailed, e
		}
		refs = append(refs, ref)
		terms = append(terms, term)
	}
	if length.Sign() != 0 {
		term, e := r.makeInteger(length)
		if e != nil {
			return RewriteFailed, e
		}
		ref, e := r.manager.NewRef(term)
		if e != nil {
			return RewriteFailed, e
		}
		refs = append(refs, ref)
		terms = append(terms, term)
	}
	family, err := r.manager.ArithmeticFamilyID()
	if err != nil {
		return RewriteFailed, err
	}
	result, err := r.manager.MkFamilyApp(family, 6, nil, terms)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, nil
}

func (r *SequenceRewriter) assignSequenceParts(parts []ast.ASTHandle, like ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if len(parts) == 0 {
		return r.assignEmptyLike(like, out)
	}
	if len(parts) == 1 {
		return RewriteDone, r.manager.SetRef(out, parts[0])
	}
	current := parts[len(parts)-1]
	refs := make([]ast.ASTRef, 0, len(parts)-1)
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			err = errors.Join(err, r.manager.DelRef(&refs[i]))
		}
	}()
	for i := len(parts) - 2; i >= 0; i-- {
		next, e := r.manager.MkSequenceConcat(parts[i], current)
		if e != nil {
			return RewriteFailed, e
		}
		ref, e := r.manager.NewRef(next)
		if e != nil {
			return RewriteFailed, e
		}
		refs = append(refs, ref)
		current = next
	}
	if err = r.manager.SetRef(out, current); err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, nil
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
	if err != nil {
		return RewriteFailed, err
	}
	if constant && posNumeral && sizeNumeral {
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
	if !posNumeral || !sizeNumeral || !pos.IsUint64() || !size.IsUint64() {
		return RewriteFailed, nil
	}
	leaves, err := r.manager.SequenceFlattenConcat(a)
	if err != nil {
		return RewriteFailed, err
	}
	for _, leaf := range leaves {
		unit, e := r.manager.IsSequenceUnit(leaf)
		if e != nil {
			return RewriteFailed, e
		}
		if !unit {
			return RewriteFailed, nil
		}
	}
	start, count := pos.Uint64(), size.Uint64()
	if start >= uint64(len(leaves)) {
		return r.assignEmptyLike(a, out)
	}
	end := start + count
	if end < start || end > uint64(len(leaves)) {
		end = uint64(len(leaves))
	}
	return r.assignSequenceParts(leaves[start:end], a, out)
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
	leaves, err := r.manager.SequenceFlattenConcat(a)
	if err != nil {
		return RewriteFailed, err
	}
	knownLength := uint64(0)
	bounded := true
	for _, leaf := range leaves {
		text, stringValue, e := r.manager.SequenceStringValue(leaf)
		if e != nil {
			return RewriteFailed, e
		}
		if stringValue {
			knownLength += uint64(len(text))
			continue
		}
		unit, e := r.manager.IsSequenceUnit(leaf)
		if e != nil {
			return RewriteFailed, e
		}
		if unit {
			knownLength++
		} else {
			bounded = false
		}
	}
	if bounded && (!position.IsUint64() || position.Uint64() >= knownLength) {
		return r.assignEmptyLike(a, out)
	}
	text, constant, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	if constant {
		if !position.IsUint64() || position.Uint64() >= uint64(len(text)) {
			return r.assignEmptyLike(a, out)
		}
		result, err := r.manager.MkString(text[position.Uint64() : position.Uint64()+1])
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	indexValue := position.Uint64()
	for _, leaf := range leaves {
		unit, e := r.manager.IsSequenceUnit(leaf)
		if e != nil {
			return RewriteFailed, e
		}
		if !unit {
			return RewriteFailed, nil
		}
		if indexValue == 0 {
			return RewriteDone, r.manager.SetRef(out, leaf)
		}
		indexValue--
	}
	return RewriteFailed, nil
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

func (r *SequenceRewriter) RewriteIntToString(a ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	value, matched, err := r.rationalValue(a)
	if err != nil || !matched {
		return RewriteFailed, err
	}
	text := ""
	if value.IsInt() && value.Sign() >= 0 {
		text = value.Num().String()
	}
	result, err := r.manager.MkString(text)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) RewriteStringToInt(a ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	text, stringValue, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, err
	}
	if stringValue {
		valid := text != ""
		for i := 0; valid && i < len(text); i++ {
			valid = '0' <= text[i] && text[i] <= '9'
		}
		if !valid {
			return r.assignInteger(big.NewInt(-1), out)
		}
		value, ok := new(big.Int).SetString(text, 10)
		if !ok {
			return RewriteFailed, ast.ErrASTDomain
		}
		return r.assignInteger(value, out)
	}
	integer, matched, err := r.manager.SequenceIntToStringArgument(a)
	if err != nil || !matched {
		return RewriteFailed, err
	}
	zero, err := r.makeInteger(big.NewInt(0))
	if err != nil {
		return RewriteFailed, err
	}
	refs := make([]ast.ASTRef, 0, 3)
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			err = errors.Join(err, r.manager.DelRef(&refs[i]))
		}
	}()
	zeroRef, err := r.manager.NewRef(zero)
	if err != nil {
		return RewriteFailed, err
	}
	refs = append(refs, zeroRef)
	minusOne, err := r.makeInteger(big.NewInt(-1))
	if err != nil {
		return RewriteFailed, err
	}
	minusOneRef, err := r.manager.NewRef(minusOne)
	if err != nil {
		return RewriteFailed, err
	}
	refs = append(refs, minusOneRef)
	family, err := r.manager.ArithmeticFamilyID()
	if err != nil {
		return RewriteFailed, err
	}
	ge, err := r.manager.MkFamilyApp(family, 3, nil, []ast.ASTHandle{integer, zero})
	if err != nil {
		return RewriteFailed, err
	}
	geRef, err := r.manager.NewRef(ge)
	if err != nil {
		return RewriteFailed, err
	}
	refs = append(refs, geRef)
	result, err := r.manager.MkIte(ge, integer, minusOne)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, nil
}

func (r *SequenceRewriter) sequenceOperation(h ast.ASTHandle, kind int32) (bool, error) {
	return r.manager.IsSequenceOperation(h, kind)
}

func (r *SequenceRewriter) makeRegexEpsilon(regex ast.ASTHandle) (result ast.ASTHandle, err error) {
	regexSort, err := r.manager.GetSort(regex)
	if err != nil {
		return result, err
	}
	sequenceSort, matched, err := r.manager.RegexSequenceSort(regexSort)
	if err != nil || !matched {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return result, err
	}
	empty, err := r.manager.MkSequenceEmpty(sequenceSort)
	if err != nil {
		return result, err
	}
	emptyRef, err := r.manager.NewRef(empty)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, r.manager.DelRef(&emptyRef)) }()
	return r.manager.MkSequenceToRegex(empty)
}

func (r *SequenceRewriter) RewriteRegexConcat(a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	fullA, err := r.sequenceOperation(a, 23)
	if err != nil {
		return RewriteFailed, err
	}
	fullB, err := r.sequenceOperation(b, 23)
	if err != nil {
		return RewriteFailed, err
	}
	if fullA && fullB {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	emptyA, err := r.sequenceOperation(a, 22)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyA {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	emptyB, err := r.sequenceOperation(b, 22)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyB {
		return RewriteDone, r.manager.SetRef(out, b)
	}
	epsilonA, err := r.manager.IsRegexEpsilon(a)
	if err != nil {
		return RewriteFailed, err
	}
	if epsilonA {
		return RewriteDone, r.manager.SetRef(out, b)
	}
	epsilonB, err := r.manager.IsRegexEpsilon(b)
	if err != nil {
		return RewriteFailed, err
	}
	if epsilonB {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) RewriteRegexUnion(a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	if a == b {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	for _, row := range []struct {
		h    ast.ASTHandle
		kind int32
		out  ast.ASTHandle
	}{{a, 22, b}, {b, 22, a}, {a, 23, a}, {b, 23, b}} {
		matched, err := r.sequenceOperation(row.h, row.kind)
		if err != nil {
			return RewriteFailed, err
		}
		if matched {
			return RewriteDone, r.manager.SetRef(out, row.out)
		}
	}
	starA, err := r.sequenceOperation(a, 14)
	if err != nil {
		return RewriteFailed, err
	}
	epsilonB, err := r.manager.IsRegexEpsilon(b)
	if err != nil {
		return RewriteFailed, err
	}
	if starA && epsilonB {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	starB, err := r.sequenceOperation(b, 14)
	if err != nil {
		return RewriteFailed, err
	}
	epsilonA, err := r.manager.IsRegexEpsilon(a)
	if err != nil {
		return RewriteFailed, err
	}
	if starB && epsilonA {
		return RewriteDone, r.manager.SetRef(out, b)
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) RewriteRegexIntersect(a, b ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	if a == b {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	for _, row := range []struct {
		h    ast.ASTHandle
		kind int32
		out  ast.ASTHandle
	}{{a, 22, a}, {b, 22, b}, {a, 23, b}, {b, 23, a}} {
		matched, err := r.sequenceOperation(row.h, row.kind)
		if err != nil {
			return RewriteFailed, err
		}
		if matched {
			return RewriteDone, r.manager.SetRef(out, row.out)
		}
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) RewriteRegexPlus(a ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	for _, kind := range []int32{22, 23, 13, 14} {
		matched, e := r.sequenceOperation(a, kind)
		if e != nil {
			return RewriteFailed, e
		}
		if matched {
			return RewriteDone, r.manager.SetRef(out, a)
		}
	}
	epsilon, err := r.manager.IsRegexEpsilon(a)
	if err != nil {
		return RewriteFailed, err
	}
	if epsilon {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	star, err := r.manager.MkRegexStar(a)
	if err != nil {
		return RewriteFailed, err
	}
	starRef, err := r.manager.NewRef(star)
	if err != nil {
		return RewriteFailed, err
	}
	defer func() { err = errors.Join(err, r.manager.DelRef(&starRef)) }()
	result, err := r.manager.MkRegexConcat(a, star)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, nil
}

func (r *SequenceRewriter) RewriteRegexOptional(a ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	epsilon, err := r.makeRegexEpsilon(a)
	if err != nil {
		return RewriteFailed, err
	}
	epsilonRef, err := r.manager.NewRef(epsilon)
	if err != nil {
		return RewriteFailed, err
	}
	defer func() { err = errors.Join(err, r.manager.DelRef(&epsilonRef)) }()
	result, err := r.manager.MkRegexUnion(epsilon, a)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return Rewrite1, nil
}

func (r *SequenceRewriter) RewriteRegexComplement(a ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
		return RewriteFailed, err
	}
	for _, row := range []struct {
		from, to int32
	}{{19, 18}, {18, 19}} {
		left, right, matched, e := r.manager.SequenceBinaryOperationArguments(a, row.from)
		if e != nil {
			return RewriteFailed, e
		}
		if !matched {
			continue
		}
		leftComplement, e := r.manager.MkRegexComplement(left)
		if e != nil {
			return RewriteFailed, e
		}
		leftRef, e := r.manager.NewRef(leftComplement)
		if e != nil {
			return RewriteFailed, e
		}
		defer func() { err = errors.Join(err, r.manager.DelRef(&leftRef)) }()
		rightComplement, e := r.manager.MkRegexComplement(right)
		if e != nil {
			return RewriteFailed, e
		}
		rightRef, e := r.manager.NewRef(rightComplement)
		if e != nil {
			return RewriteFailed, e
		}
		defer func() { err = errors.Join(err, r.manager.DelRef(&rightRef)) }()
		var result ast.ASTHandle
		if row.to == 18 {
			result, e = r.manager.MkRegexUnion(leftComplement, rightComplement)
		} else {
			result, e = r.manager.MkRegexIntersect(leftComplement, rightComplement)
		}
		if e != nil {
			return RewriteFailed, e
		}
		if err = r.manager.SetRef(out, result); err != nil {
			return RewriteFailed, err
		}
		return Rewrite2, nil
	}
	sort, err := r.manager.GetSort(a)
	if err != nil {
		return RewriteFailed, err
	}
	empty, err := r.sequenceOperation(a, 22)
	if err != nil {
		return RewriteFailed, err
	}
	if empty {
		result, e := r.manager.MkRegexFull(sort)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	full, err := r.sequenceOperation(a, 23)
	if err != nil {
		return RewriteFailed, err
	}
	if full {
		result, e := r.manager.MkRegexEmpty(sort)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) assignRegexStar(arg ast.ASTHandle, out ast.ASTRef, status RewriteStatus) (RewriteStatus, error) {
	result, err := r.manager.MkRegexStar(arg)
	if err != nil {
		return RewriteFailed, err
	}
	return status, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) assignRegexStarUnion(left, right ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	union, err := r.manager.MkRegexUnion(left, right)
	if err != nil {
		return RewriteFailed, err
	}
	unionRef, err := r.manager.NewRef(union)
	if err != nil {
		return RewriteFailed, err
	}
	defer func() { err = errors.Join(err, r.manager.DelRef(&unionRef)) }()
	result, err := r.manager.MkRegexStar(union)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, nil
}

func (r *SequenceRewriter) RewriteRegexStar(a ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	for _, kind := range []int32{14, 23} {
		matched, err := r.sequenceOperation(a, kind)
		if err != nil {
			return RewriteFailed, err
		}
		if matched {
			return RewriteDone, r.manager.SetRef(out, a)
		}
	}
	fullChar, err := r.sequenceOperation(a, 24)
	if err != nil {
		return RewriteFailed, err
	}
	if fullChar {
		sort, err := r.manager.GetSort(a)
		if err != nil {
			return RewriteFailed, err
		}
		result, err := r.manager.MkRegexFull(sort)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	empty, err := r.sequenceOperation(a, 22)
	if err != nil {
		return RewriteFailed, err
	}
	if empty {
		epsilon, err := r.makeRegexEpsilon(a)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, epsilon)
	}
	body, plus, err := r.manager.SequenceUnaryOperationArgument(a, 13)
	if err != nil {
		return RewriteFailed, err
	}
	if plus {
		return r.assignRegexStar(body, out, RewriteDone)
	}
	left, right, union, err := r.manager.SequenceBinaryOperationArguments(a, 18)
	if err != nil {
		return RewriteFailed, err
	}
	if union {
		leftBody, leftStar, e := r.manager.SequenceUnaryOperationArgument(left, 14)
		if e != nil {
			return RewriteFailed, e
		}
		if leftStar {
			return r.assignRegexStarUnion(leftBody, right, out)
		}
		rightBody, rightStar, e := r.manager.SequenceUnaryOperationArgument(right, 14)
		if e != nil {
			return RewriteFailed, e
		}
		if rightStar {
			return r.assignRegexStarUnion(left, rightBody, out)
		}
		leftEpsilon, e := r.manager.IsRegexEpsilon(left)
		if e != nil {
			return RewriteFailed, e
		}
		if leftEpsilon {
			return r.assignRegexStar(right, out, Rewrite2)
		}
		rightEpsilon, e := r.manager.IsRegexEpsilon(right)
		if e != nil {
			return RewriteFailed, e
		}
		if rightEpsilon {
			return r.assignRegexStar(left, out, Rewrite2)
		}
	}
	left, right, concat, err := r.manager.SequenceBinaryOperationArguments(a, 17)
	if err != nil {
		return RewriteFailed, err
	}
	if concat {
		leftBody, leftStar, e := r.manager.SequenceUnaryOperationArgument(left, 14)
		if e != nil {
			return RewriteFailed, e
		}
		rightBody, rightStar, e := r.manager.SequenceUnaryOperationArgument(right, 14)
		if e != nil {
			return RewriteFailed, e
		}
		if leftStar && rightStar {
			return r.assignRegexStarUnion(leftBody, rightBody, out)
		}
	}
	return RewriteFailed, nil
}

func (r *SequenceRewriter) RewriteRegexLoop(args []ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	if len(args) != 2 && len(args) != 3 {
		return RewriteFailed, nil
	}
	lower, matched, err := r.integerValue(args[1])
	if err != nil || !matched || lower.Sign() < 0 || !lower.IsUint64() || lower.Uint64() > math.MaxInt32 {
		return RewriteFailed, err
	}
	var result ast.ASTHandle
	if len(args) == 2 {
		result, err = r.manager.MkRegexLoop(args[0], int32(lower.Uint64()))
	} else {
		upper, upperMatched, e := r.integerValue(args[2])
		if e != nil || !upperMatched || upper.Sign() < 0 || !upper.IsUint64() || upper.Uint64() > math.MaxInt32 {
			return RewriteFailed, e
		}
		result, err = r.manager.MkRegexLoop(args[0], int32(lower.Uint64()), int32(upper.Uint64()))
	}
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *SequenceRewriter) RewriteStringInRegex(_ ast.ASTHandle, regex ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	empty, err := r.sequenceOperation(regex, 22)
	if err != nil {
		return RewriteFailed, err
	}
	if empty {
		return r.assignBoolean(false, out)
	}
	full, err := r.sequenceOperation(regex, 23)
	if err != nil {
		return RewriteFailed, err
	}
	if full {
		return r.assignBoolean(true, out)
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
