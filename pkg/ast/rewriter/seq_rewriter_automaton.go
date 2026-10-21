// Source: src/ast/rewriter/seq_rewriter.cpp:26-193,203-342,1147-1326.
package rewriter

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type sequenceAutomatonLabelKind uint8

const (
	sequenceLabelCharacter sequenceAutomatonLabelKind = iota
	sequenceLabelRange
	sequenceLabelAny
)

type sequenceAutomatonLabel struct {
	kind       sequenceAutomatonLabelKind
	character  ast.ASTHandle
	lowerBound ast.ASTHandle
	upperBound ast.ASTHandle
}

type sequenceAutomatonMove struct {
	source, destination int
	label               *sequenceAutomatonLabel
}

type sequenceAutomaton struct {
	initial int
	finals  map[int]bool
	moves   [][]sequenceAutomatonMove
}

func newSequenceAutomaton(states int) *sequenceAutomaton {
	if states < 1 {
		states = 1
	}
	return &sequenceAutomaton{finals: make(map[int]bool), moves: make([][]sequenceAutomatonMove, states)}
}

func sequenceEmptyAutomaton() *sequenceAutomaton { return newSequenceAutomaton(1) }

func sequenceEpsilonAutomaton() *sequenceAutomaton {
	a := newSequenceAutomaton(1)
	a.finals[0] = true
	return a
}

func sequenceLabelAutomaton(label *sequenceAutomatonLabel) *sequenceAutomaton {
	a := newSequenceAutomaton(2)
	a.finals[1] = true
	a.moves[0] = append(a.moves[0], sequenceAutomatonMove{source: 0, destination: 1, label: label})
	return a
}

func sequenceLoopAutomaton(label *sequenceAutomatonLabel) *sequenceAutomaton {
	a := newSequenceAutomaton(1)
	a.finals[0] = true
	a.moves[0] = append(a.moves[0], sequenceAutomatonMove{source: 0, destination: 0, label: label})
	return a
}

func (a *sequenceAutomaton) appendInto(result *sequenceAutomaton, offset int) {
	for source, moves := range a.moves {
		for _, move := range moves {
			result.moves[source+offset] = append(result.moves[source+offset], sequenceAutomatonMove{
				source: source + offset, destination: move.destination + offset, label: move.label,
			})
		}
	}
	for state := range a.finals {
		result.finals[state+offset] = true
	}
}

func sequenceUnionAutomaton(left, right *sequenceAutomaton) *sequenceAutomaton {
	leftOffset := 1
	rightOffset := leftOffset + len(left.moves)
	result := newSequenceAutomaton(1 + len(left.moves) + len(right.moves))
	left.appendInto(result, leftOffset)
	right.appendInto(result, rightOffset)
	result.moves[0] = append(result.moves[0],
		sequenceAutomatonMove{source: 0, destination: left.initial + leftOffset},
		sequenceAutomatonMove{source: 0, destination: right.initial + rightOffset})
	return result
}

func sequenceConcatAutomaton(left, right *sequenceAutomaton) *sequenceAutomaton {
	rightOffset := len(left.moves)
	result := newSequenceAutomaton(len(left.moves) + len(right.moves))
	left.appendInto(result, 0)
	for state := range left.finals {
		delete(result.finals, state)
		result.moves[state] = append(result.moves[state], sequenceAutomatonMove{source: state, destination: right.initial + rightOffset})
	}
	right.appendInto(result, rightOffset)
	result.initial = left.initial
	return result
}

func sequenceStarAutomaton(body *sequenceAutomaton, includeEmpty bool) *sequenceAutomaton {
	result := newSequenceAutomaton(len(body.moves))
	body.appendInto(result, 0)
	for state := range body.finals {
		result.moves[state] = append(result.moves[state], sequenceAutomatonMove{source: state, destination: body.initial})
	}
	if includeEmpty {
		result.finals[body.initial] = true
	}
	result.initial = body.initial
	return result
}

func sequenceOptionalAutomaton(body *sequenceAutomaton) *sequenceAutomaton {
	result := newSequenceAutomaton(len(body.moves))
	body.appendInto(result, 0)
	result.initial = body.initial
	result.finals[result.initial] = true
	return result
}

func (a *sequenceAutomaton) epsilonClosure(state int) []int {
	seen := map[int]bool{state: true}
	stack := []int{state}
	result := make([]int, 0, 4)
	for len(stack) > 0 {
		state = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		result = append(result, state)
		for _, move := range a.moves[state] {
			if move.label == nil && !seen[move.destination] {
				seen[move.destination] = true
				stack = append(stack, move.destination)
			}
		}
	}
	return result
}

func (a *sequenceAutomaton) closureIsFinal(state int) bool {
	for _, candidate := range a.epsilonClosure(state) {
		if a.finals[candidate] {
			return true
		}
	}
	return false
}

func (a *sequenceAutomaton) movesFrom(state int, closeDestination bool) []sequenceAutomatonMove {
	result := make([]sequenceAutomatonMove, 0, 4)
	for _, source := range a.epsilonClosure(state) {
		for _, move := range a.moves[source] {
			if move.label == nil {
				continue
			}
			if !closeDestination {
				result = append(result, move)
				continue
			}
			for _, destination := range a.epsilonClosure(move.destination) {
				result = append(result, sequenceAutomatonMove{source: source, destination: destination, label: move.label})
			}
		}
	}
	return result
}

type sequenceAutomatonBuilder struct {
	rewriter *SequenceRewriter
	refs     []ast.ASTRef
}

func (b *sequenceAutomatonBuilder) close() (err error) {
	for i := len(b.refs) - 1; i >= 0; i-- {
		err = errors.Join(err, b.rewriter.manager.DelRef(&b.refs[i]))
	}
	return err
}

func (b *sequenceAutomatonBuilder) hold(h ast.ASTHandle) error {
	ref, err := b.rewriter.manager.NewRef(h)
	if err != nil {
		return err
	}
	b.refs = append(b.refs, ref)
	return nil
}

func (b *sequenceAutomatonBuilder) character(value byte) (ast.ASTHandle, error) {
	h, err := b.rewriter.manager.BitVectorNumeralUnsigned(uint32(value), 8)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = b.hold(h); err != nil {
		return ast.ASTHandle{}, err
	}
	return h, nil
}

func (b *sequenceAutomatonBuilder) sequence(expression ast.ASTHandle) (*sequenceAutomaton, error) {
	leaves, err := b.rewriter.manager.SequenceFlattenConcat(expression)
	if err != nil {
		return nil, err
	}
	result := sequenceEpsilonAutomaton()
	for _, leaf := range leaves {
		text, stringValue, err := b.rewriter.manager.SequenceStringValue(leaf)
		if err != nil {
			return nil, err
		}
		if stringValue {
			for i := 0; i < len(text); i++ {
				character, err := b.character(text[i])
				if err != nil {
					return nil, err
				}
				result = sequenceConcatAutomaton(result, sequenceLabelAutomaton(&sequenceAutomatonLabel{kind: sequenceLabelCharacter, character: character}))
			}
			continue
		}
		element, unit, err := b.rewriter.manager.SequenceUnitArgument(leaf)
		if err != nil {
			return nil, err
		}
		if !unit {
			return nil, nil
		}
		result = sequenceConcatAutomaton(result, sequenceLabelAutomaton(&sequenceAutomatonLabel{kind: sequenceLabelCharacter, character: element}))
	}
	return result, nil
}

func (b *sequenceAutomatonBuilder) regex(expression ast.ASTHandle) (*sequenceAutomaton, error) {
	if argument, matched, err := b.rewriter.manager.SequenceUnaryOperationArgument(expression, 11); err != nil {
		return nil, err
	} else if matched {
		return b.sequence(argument)
	}
	for _, row := range []struct {
		kind int32
		make func(*sequenceAutomaton, *sequenceAutomaton) *sequenceAutomaton
	}{{17, sequenceConcatAutomaton}, {18, sequenceUnionAutomaton}} {
		left, right, matched, err := b.rewriter.manager.SequenceBinaryOperationArguments(expression, row.kind)
		if err != nil {
			return nil, err
		}
		if matched {
			leftAutomaton, err := b.regex(left)
			if err != nil || leftAutomaton == nil {
				return nil, err
			}
			rightAutomaton, err := b.regex(right)
			if err != nil || rightAutomaton == nil {
				return nil, err
			}
			return row.make(leftAutomaton, rightAutomaton), nil
		}
	}
	for _, row := range []struct {
		kind int32
		make func(*sequenceAutomaton) *sequenceAutomaton
	}{{14, func(a *sequenceAutomaton) *sequenceAutomaton { return sequenceStarAutomaton(a, true) }},
		{13, func(a *sequenceAutomaton) *sequenceAutomaton { return sequenceStarAutomaton(a, false) }},
		{15, sequenceOptionalAutomaton}} {
		argument, matched, err := b.rewriter.manager.SequenceUnaryOperationArgument(expression, row.kind)
		if err != nil {
			return nil, err
		}
		if matched {
			automaton, err := b.regex(argument)
			if err != nil || automaton == nil {
				return nil, err
			}
			return row.make(automaton), nil
		}
	}
	if lower, upper, matched, err := b.rewriter.manager.SequenceBinaryOperationArguments(expression, 16); err != nil {
		return nil, err
	} else if matched {
		lo, loString, err := b.rewriter.manager.SequenceStringValue(lower)
		if err != nil {
			return nil, err
		}
		hi, hiString, err := b.rewriter.manager.SequenceStringValue(upper)
		if err != nil {
			return nil, err
		}
		if !loString || !hiString || len(lo) != 1 || len(hi) != 1 {
			return nil, nil
		}
		loValue, err := b.character(lo[0])
		if err != nil {
			return nil, err
		}
		hiValue, err := b.character(hi[0])
		if err != nil {
			return nil, err
		}
		return sequenceLabelAutomaton(&sequenceAutomatonLabel{kind: sequenceLabelRange, lowerBound: loValue, upperBound: hiValue}), nil
	}
	body, lower, upper, bounded, loop, err := b.rewriter.manager.RegexLoopParameters(expression)
	if err != nil {
		return nil, err
	}
	if loop {
		if lower < 0 || bounded && upper < lower {
			return nil, nil
		}
		base, err := b.regex(body)
		if err != nil || base == nil {
			return nil, err
		}
		result := sequenceEpsilonAutomaton()
		for i := int32(0); i < lower; i++ {
			result = sequenceConcatAutomaton(result, base)
		}
		if bounded {
			for i := lower; i < upper; i++ {
				result = sequenceConcatAutomaton(result, sequenceOptionalAutomaton(base))
			}
		} else {
			result = sequenceConcatAutomaton(result, sequenceStarAutomaton(base, true))
		}
		return result, nil
	}
	for _, row := range []struct {
		kind int32
		make func() *sequenceAutomaton
	}{{22, sequenceEmptyAutomaton}, {23, func() *sequenceAutomaton {
		return sequenceLoopAutomaton(&sequenceAutomatonLabel{kind: sequenceLabelAny})
	}},
		{24, func() *sequenceAutomaton {
			return sequenceLabelAutomaton(&sequenceAutomatonLabel{kind: sequenceLabelAny})
		}}} {
		matched, err := b.rewriter.sequenceOperation(expression, row.kind)
		if err != nil {
			return nil, err
		}
		if matched {
			return row.make(), nil
		}
	}
	return nil, nil
}

func (b *sequenceAutomatonBuilder) automatonSequence(a *sequenceAutomaton) ([]ast.ASTHandle, bool) {
	state := a.initial
	visited := make(map[int]bool)
	result := make([]ast.ASTHandle, 0, 4)
	for {
		closure := a.epsilonClosure(state)
		final := false
		moves := make([]sequenceAutomatonMove, 0, 2)
		for _, source := range closure {
			final = final || a.finals[source]
			for _, move := range a.moves[source] {
				if move.label != nil {
					moves = append(moves, move)
				}
			}
		}
		if final {
			return result, len(moves) == 0
		}
		if len(moves) != 1 || visited[state] || moves[0].label.kind != sequenceLabelCharacter {
			return nil, false
		}
		visited[state] = true
		result = append(result, moves[0].label.character)
		state = moves[0].destination
	}
}

func (b *sequenceAutomatonBuilder) expressionSequence(expression ast.ASTHandle) ([]ast.ASTHandle, bool, error) {
	leaves, err := b.rewriter.manager.SequenceFlattenConcat(expression)
	if err != nil {
		return nil, false, err
	}
	result := make([]ast.ASTHandle, 0, len(leaves))
	for _, leaf := range leaves {
		text, stringValue, err := b.rewriter.manager.SequenceStringValue(leaf)
		if err != nil {
			return nil, false, err
		}
		if stringValue {
			for i := 0; i < len(text); i++ {
				character, err := b.character(text[i])
				if err != nil {
					return nil, false, err
				}
				result = append(result, character)
			}
			continue
		}
		element, unit, err := b.rewriter.manager.SequenceUnitArgument(leaf)
		if err != nil {
			return nil, false, err
		}
		if !unit {
			return nil, false, nil
		}
		result = append(result, element)
	}
	return result, true, nil
}

func (b *sequenceAutomatonBuilder) booleanAnd(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	if b.rewriter.manager.IsFalse(left) || b.rewriter.manager.IsFalse(right) {
		return b.rewriter.manager.MkFalse()
	}
	if b.rewriter.manager.IsTrue(left) {
		return right, nil
	}
	if b.rewriter.manager.IsTrue(right) || left == right {
		return left, nil
	}
	result, err := b.rewriter.manager.MkAndArray([]ast.ASTHandle{left, right})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = b.hold(result); err != nil {
		return ast.ASTHandle{}, err
	}
	return result, nil
}

func (b *sequenceAutomatonBuilder) booleanOr(left, right ast.ASTHandle) (ast.ASTHandle, error) {
	if b.rewriter.manager.IsTrue(left) || b.rewriter.manager.IsTrue(right) {
		return b.rewriter.manager.MkTrue()
	}
	if b.rewriter.manager.IsFalse(left) {
		return right, nil
	}
	if b.rewriter.manager.IsFalse(right) || left == right {
		return left, nil
	}
	result, err := b.rewriter.manager.MkOrArray([]ast.ASTHandle{left, right})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = b.hold(result); err != nil {
		return ast.ASTHandle{}, err
	}
	return result, nil
}

func (b *sequenceAutomatonBuilder) accept(label *sequenceAutomatonLabel, character ast.ASTHandle) (ast.ASTHandle, error) {
	if label.kind == sequenceLabelAny {
		return b.rewriter.manager.MkTrue()
	}
	if label.kind == sequenceLabelCharacter {
		labelValue, err := b.rewriter.manager.IsValue(label.character)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		characterValue, err := b.rewriter.manager.IsValue(character)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if labelValue && characterValue {
			return b.rewriter.manager.MkBoolVal(label.character == character)
		}
		result, err := b.rewriter.manager.MkEq(character, label.character)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if err = b.hold(result); err != nil {
			return ast.ASTHandle{}, err
		}
		return result, nil
	}
	lo, loNumeral, err := b.rewriter.manager.SequenceUnitByteElementValue(label.lowerBound)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	value, valueNumeral, err := b.rewriter.manager.SequenceUnitByteElementValue(character)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	hi, hiNumeral, err := b.rewriter.manager.SequenceUnitByteElementValue(label.upperBound)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if loNumeral && valueNumeral && hiNumeral {
		return b.rewriter.manager.MkBoolVal(lo <= value && value <= hi)
	}
	lower, err := b.rewriter.manager.MkBitVectorULE(label.lowerBound, character)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = b.hold(lower); err != nil {
		return ast.ASTHandle{}, err
	}
	upper, err := b.rewriter.manager.MkBitVectorULE(character, label.upperBound)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = b.hold(upper); err != nil {
		return ast.ASTHandle{}, err
	}
	return b.booleanAnd(lower, upper)
}

func (r *SequenceRewriter) rewriteStringInRegex(sequence, regex ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err = r.open(); err != nil {
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
	builder := sequenceAutomatonBuilder{rewriter: r}
	defer func() { err = errors.Join(err, builder.close()) }()
	automaton, err := builder.regex(regex)
	if err != nil || automaton == nil {
		return RewriteFailed, err
	}
	if elements, singleton := builder.automatonSequence(automaton); singleton {
		var value ast.ASTHandle
		if len(elements) == 0 {
			sort, e := r.manager.GetSort(sequence)
			if e != nil {
				return RewriteFailed, e
			}
			value, e = r.manager.MkSequenceEmpty(sort)
			if e != nil {
				return RewriteFailed, e
			}
			if e = builder.hold(value); e != nil {
				return RewriteFailed, e
			}
		} else {
			units := make([]ast.ASTHandle, 0, len(elements))
			for _, element := range elements {
				unit, e := r.manager.MkSequenceUnit(element)
				if e != nil {
					return RewriteFailed, e
				}
				if e = builder.hold(unit); e != nil {
					return RewriteFailed, e
				}
				units = append(units, unit)
			}
			valueRef, h, e := r.makeSequencePartsRef(units, sequence)
			if e != nil {
				return RewriteFailed, e
			}
			builder.refs = append(builder.refs, valueRef)
			value = h
		}
		result, e := r.manager.MkEq(sequence, value)
		if e != nil {
			return RewriteFailed, e
		}
		if err = r.manager.SetRef(out, result); err != nil {
			return RewriteFailed, err
		}
		return RewriteFull, nil
	}
	elements, concrete, err := builder.expressionSequence(sequence)
	if err != nil || !concrete {
		return RewriteFailed, err
	}
	truth, err := r.manager.MkTrue()
	if err != nil {
		return RewriteFailed, err
	}
	frontier := map[int]ast.ASTHandle{automaton.initial: truth}
	for _, character := range elements {
		next := make(map[int]ast.ASTHandle)
		for state, accumulated := range frontier {
			for _, move := range automaton.movesFrom(state, false) {
				condition, e := builder.accept(move.label, character)
				if e != nil {
					return RewriteFailed, e
				}
				if r.manager.IsFalse(condition) {
					continue
				}
				condition, e = builder.booleanAnd(condition, accumulated)
				if e != nil {
					return RewriteFailed, e
				}
				if previous, present := next[move.destination]; present && !r.manager.IsTrue(condition) {
					condition, e = builder.booleanOr(condition, previous)
					if e != nil {
						return RewriteFailed, e
					}
				}
				next[move.destination] = condition
			}
		}
		frontier = next
	}
	disjuncts := make([]ast.ASTHandle, 0, len(frontier))
	for state, condition := range frontier {
		if automaton.closureIsFinal(state) {
			disjuncts = append(disjuncts, condition)
		}
	}
	result, err := r.manager.MkOrArray(disjuncts)
	if err != nil {
		return RewriteFailed, err
	}
	if err = r.manager.SetRef(out, result); err != nil {
		return RewriteFailed, err
	}
	return RewriteFull, nil
}
