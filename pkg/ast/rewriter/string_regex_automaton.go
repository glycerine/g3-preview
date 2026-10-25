// Source: src/ast/rewriter/seq_rewriter.cpp mk_seq_automaton retained
// byte-string consumer in src/smt/theory_str.cpp. Reuse the concrete builder.
package rewriter

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/automata"
)

type StringRegexConditionKind uint8

const (
	StringRegexCharacter StringRegexConditionKind = iota
	StringRegexRange
	StringRegexPredicate
)

// StringRegexCondition borrows ASTs from its StringRegexAutomaton owner.
type StringRegexCondition struct {
	Kind                    StringRegexConditionKind
	Character, Lower, Upper ast.ASTHandle
	Predicate               ast.ASTHandle
}

// StringRegexAutomaton owns its graph and roots the regex and all AST labels.
// Graph/Condition views are borrowed and must not outlive this owner.
type StringRegexAutomaton struct {
	builder    sequenceAutomatonBuilder
	graph      *automata.Automaton
	conditions []StringRegexCondition
	closed     bool
}

func NewStringRegexAutomaton(m *ast.ASTManager, regex ast.ASTHandle, solver SequenceExpressionSolver) (owner *StringRegexAutomaton, result error) {
	rw, err := NewSequenceRewriter(m)
	if err != nil {
		return nil, err
	}
	rw.solver = solver
	regexSort, err := m.GetSort(regex)
	if err != nil {
		return nil, err
	}
	sequenceSort, matched, err := m.RegexSequenceSort(regexSort)
	if err != nil || !matched {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return nil, err
	}
	stringSort, err := m.IsStringSort(sequenceSort)
	if err != nil || !stringSort {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return nil, err
	}
	a := &StringRegexAutomaton{builder: sequenceAutomatonBuilder{rewriter: rw}}
	defer func() {
		if result != nil {
			result = errors.Join(result, a.Close())
		}
	}()
	if err := a.builder.hold(regex); err != nil {
		return nil, err
	}
	built, err := a.builder.regex(regex)
	if err != nil {
		return nil, err
	}
	if built == nil {
		return nil, ast.ErrASTDomain
	}
	var finals []uint32
	for state := range built.moves {
		if built.finals[state] {
			finals = append(finals, uint32(state))
		}
	}
	labels := make(map[*sequenceAutomatonLabel]uint32)
	var moves []automata.Move
	for source, outgoing := range built.moves {
		for _, move := range outgoing {
			if move.label == nil {
				moves = append(moves, automata.EpsilonMove(uint32(source), uint32(move.destination)))
				continue
			}
			id, found := labels[move.label]
			if !found {
				condition, err := a.condition(move.label)
				if err != nil {
					return nil, err
				}
				id = uint32(len(a.conditions))
				a.conditions = append(a.conditions, condition)
				labels[move.label] = id
			}
			moves = append(moves, automata.ConditionMove(uint32(source), uint32(move.destination), id))
		}
	}
	a.graph = automata.New(uint32(built.initial), finals, moves)
	return a, nil
}

func (a *StringRegexAutomaton) condition(label *sequenceAutomatonLabel) (StringRegexCondition, error) {
	switch label.kind {
	case sequenceLabelCharacter:
		return StringRegexCondition{Kind: StringRegexCharacter, Character: label.character}, nil
	case sequenceLabelRange:
		return StringRegexCondition{Kind: StringRegexRange, Lower: label.lowerBound, Upper: label.upperBound}, nil
	case sequenceLabelAny:
		predicate, err := a.builder.rewriter.manager.MkTrue()
		if err == nil {
			err = a.builder.hold(predicate)
		}
		return StringRegexCondition{Kind: StringRegexPredicate, Predicate: predicate}, err
	default:
		m := a.builder.rewriter.manager
		sort, err := m.BitVectorSort(8)
		if err != nil {
			return StringRegexCondition{}, err
		}
		variable, err := m.MkVar(0, sort)
		if err != nil {
			return StringRegexCondition{}, err
		}
		if err := a.builder.hold(variable); err != nil {
			return StringRegexCondition{}, err
		}
		predicate, err := a.builder.accept(label, variable)
		if err != nil {
			return StringRegexCondition{}, err
		}
		if err := a.builder.hold(predicate); err != nil {
			return StringRegexCondition{}, err
		}
		return StringRegexCondition{Kind: StringRegexPredicate, Predicate: predicate}, nil
	}
}

func (a *StringRegexAutomaton) Graph() (*automata.Automaton, error) {
	if a == nil || a.closed {
		return nil, ast.ErrASTHandle
	}
	return a.graph, nil
}

func (a *StringRegexAutomaton) Condition(id uint32) (StringRegexCondition, error) {
	if a == nil || a.closed {
		return StringRegexCondition{}, ast.ErrASTHandle
	}
	if uint64(id) >= uint64(len(a.conditions)) {
		return StringRegexCondition{}, ast.ErrASTDomain
	}
	return a.conditions[id], nil
}

func (a *StringRegexAutomaton) Close() error {
	if a == nil || a.closed {
		return nil
	}
	result := a.builder.close()
	a.conditions, a.graph = nil, nil
	a.closed = true
	return result
}
