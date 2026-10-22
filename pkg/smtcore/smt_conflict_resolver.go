// Source: src/smt/smt_conflict_resolution.cpp initialize_resolve(),
// process_antecedent(), resolve(), and the non-minimizing part of
// finalize_resolve().
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type SMTConflictResolution struct {
	Resolved           bool
	ConflictLevel      uint32
	NewScopeLevel      uint32
	InternalScopeLevel uint32
	Lemma              []SMTLiteral
}

type SMTConflictResolver struct {
	state    *SMTInternalizerState
	expander *SMTAntecedentExpander
}

func NewSMTConflictResolver(state *SMTInternalizerState) (*SMTConflictResolver, error) {
	if state == nil || state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTConflictResolver{state: state, expander: NewSMTAntecedentExpander()}, nil
}

func (r *SMTConflictResolver) levels() []uint32 {
	levels := make([]uint32, len(r.state.booleanData))
	for variable := int32(SMTFirstBoolVar); int(variable) < len(levels); variable++ {
		levels[variable] = r.state.booleanData[variable].ScopeLevel
	}
	return levels
}

func (r *SMTConflictResolver) Resolve(conflict SMTBooleanJustification, notLiteral SMTLiteral, baseLevel uint32) (SMTConflictResolution, error) {
	if r == nil || r.state == nil || r.expander == nil || r.state.open() != nil {
		return SMTConflictResolution{}, ast.ErrASTHandle
	}
	consequent := SMTFalseLiteral
	if notLiteral != SMTNullLiteral {
		consequent = notLiteral.Negated()
	}
	levels := r.levels()
	conflictLevel, err := SMTBooleanJustificationMaxLevel(r.expander, consequent, conflict, levels)
	if err != nil {
		return SMTConflictResolution{}, err
	}
	result := SMTConflictResolution{ConflictLevel: conflictLevel}
	if conflictLevel <= baseLevel {
		return result, nil
	}
	marks := make([]bool, len(r.state.booleanData))
	lemma := []SMTLiteral{SMTNullLiteral}
	numMarks := 0
	processAntecedent := func(literal SMTLiteral) error {
		variable := literal.Var()
		if variable < SMTFirstBoolVar || int(variable) >= len(marks) {
			return ast.ErrASTDomain
		}
		level := levels[variable]
		if marks[variable] || level <= baseLevel {
			return nil
		}
		marks[variable] = true
		r.state.activity[variable]++
		if level == conflictLevel {
			numMarks++
		} else {
			lemma = append(lemma, literal.Negated())
		}
		return nil
	}
	var process func(SMTLiteral, SMTBooleanJustification) error
	process = func(consequent SMTLiteral, justification SMTBooleanJustification) error {
		switch justification.Kind() {
		case SMTBooleanJustificationClause:
			clause, ok := justification.Clause()
			if !ok || clause == nil {
				return ast.ErrASTDomain
			}
			start := 0
			if consequent != SMTFalseLiteral {
				if clause.literals[0] == consequent {
					start = 1
				} else if len(clause.literals) > 1 && clause.literals[1] == consequent {
					if err := processAntecedent(clause.literals[0].Negated()); err != nil {
						return err
					}
					start = 2
				} else {
					return ast.ErrASTDomain
				}
			}
			for _, literal := range clause.literals[start:] {
				if err := processAntecedent(literal.Negated()); err != nil {
					return err
				}
			}
			if clause.justification != nil {
				antecedents, err := r.expander.ExpandJustification(clause.justification)
				if err != nil {
					return err
				}
				for _, literal := range antecedents {
					if err = processAntecedent(literal); err != nil {
						return err
					}
				}
			}
		case SMTBooleanJustificationBinaryClause:
			literal, ok := justification.Literal()
			if !ok {
				return ast.ErrASTDomain
			}
			return processAntecedent(literal)
		case SMTBooleanJustificationAxiom:
		case SMTBooleanJustificationObject:
			object, ok := justification.Justification()
			if !ok || object == nil {
				return ast.ErrASTDomain
			}
			antecedents, err := r.expander.ExpandJustification(object)
			if err != nil {
				return err
			}
			for _, literal := range antecedents {
				if err = processAntecedent(literal); err != nil {
					return err
				}
			}
		default:
			return ast.ErrASTDomain
		}
		return nil
	}
	if notLiteral != SMTNullLiteral {
		if err = processAntecedent(notLiteral); err != nil {
			return SMTConflictResolution{}, err
		}
	}
	index := len(r.state.assignedLiterals) - 1
	for {
		if err = process(consequent, conflict); err != nil {
			return SMTConflictResolution{}, err
		}
		for index >= 0 && !marks[r.state.assignedLiterals[index].Var()] {
			index--
		}
		if index < 0 {
			return SMTConflictResolution{}, ast.ErrASTDomain
		}
		consequent = r.state.assignedLiterals[index]
		variable := consequent.Var()
		conflict = r.state.booleanData[variable].Justification()
		marks[variable] = false
		index--
		numMarks--
		if numMarks == 0 {
			break
		}
	}
	lemma[0] = consequent.Negated()
	result.Resolved, result.Lemma = true, lemma
	result.InternalScopeLevel = r.state.booleanData[lemma[0].Var()].InternalLevel()
	result.NewScopeLevel = baseLevel
	for _, literal := range lemma[1:] {
		variable := literal.Var()
		marks[variable] = false
		if levels[variable] > result.NewScopeLevel {
			result.NewScopeLevel = levels[variable]
		}
		if level := r.state.booleanData[variable].InternalLevel(); level > result.InternalScopeLevel {
			result.InternalScopeLevel = level
		}
	}
	return result, nil
}
