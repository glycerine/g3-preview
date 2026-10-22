// Source: src/smt/smt_conflict_resolution.cpp initialize_resolve(),
// process_antecedent(), resolve(), and the non-minimizing part of
// finalize_resolve().
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type SMTConflictResolution struct {
	Resolved           bool
	ConflictLevel      uint32
	NewScopeLevel      uint32
	InternalScopeLevel uint32
	Lemma              []SMTLiteral
}

type SMTConflictResolver struct {
	state      *SMTInternalizerState
	expander   *SMTAntecedentExpander
	lemmaProof ast.ASTRef
	closePhase uint8
	closed     bool
}

func (r *SMTConflictResolver) minimizeLemma(lemma []SMTLiteral, marks []bool, levels []uint32, baseLevel uint32) ([]SMTLiteral, error) {
	levelSet := make(map[uint32]struct{}, len(lemma))
	for _, literal := range lemma {
		levelSet[levels[literal.Var()]] = struct{}{}
	}
	temporary := make([]int32, 0)
	var implied func(SMTLiteral) (bool, error)
	implied = func(literal SMTLiteral) (bool, error) {
		oldTemporary := len(temporary)
		stack := []int32{literal.Var()}
		processAntecedent := func(antecedent SMTLiteral) bool {
			variable := antecedent.Var()
			level := levels[variable]
			if marks[variable] || level <= baseLevel {
				return true
			}
			if _, found := levelSet[level]; !found {
				return false
			}
			marks[variable] = true
			temporary = append(temporary, variable)
			stack = append(stack, variable)
			return true
		}
		fail := func() (bool, error) {
			for len(temporary) > oldTemporary {
				index := len(temporary) - 1
				marks[temporary[index]] = false
				temporary = temporary[:index]
			}
			return false, nil
		}
		for len(stack) != 0 {
			index := len(stack) - 1
			variable := stack[index]
			stack = stack[:index]
			justification := r.state.booleanData[variable].Justification()
			switch justification.Kind() {
			case SMTBooleanJustificationClause:
				clause, ok := justification.Clause()
				if !ok || clause == nil {
					return false, ast.ErrASTDomain
				}
				position := -1
				for i := 0; i < len(clause.literals) && i < 2; i++ {
					if clause.literals[i].Var() == variable {
						position = i
						break
					}
				}
				if position < 0 {
					return false, ast.ErrASTDomain
				}
				for i, item := range clause.literals {
					if i != position && !processAntecedent(item.Negated()) {
						return fail()
					}
				}
				if clause.justification != nil {
					antecedents, err := r.expander.ExpandJustification(clause.justification)
					if err != nil {
						return false, err
					}
					for _, antecedent := range antecedents {
						if !processAntecedent(antecedent) {
							return fail()
						}
					}
				}
			case SMTBooleanJustificationBinaryClause:
				antecedent, ok := justification.Literal()
				if !ok {
					return false, ast.ErrASTDomain
				}
				if !processAntecedent(antecedent) {
					return fail()
				}
			case SMTBooleanJustificationAxiom:
				if levels[variable] > baseLevel {
					return fail()
				}
			case SMTBooleanJustificationObject:
				if r.state.booleanData[variable].Assumption {
					return fail()
				}
				object, ok := justification.Justification()
				if !ok || object == nil {
					return false, ast.ErrASTDomain
				}
				antecedents, err := r.expander.ExpandJustification(object)
				if err != nil {
					return false, err
				}
				for _, antecedent := range antecedents {
					if !processAntecedent(antecedent) {
						return fail()
					}
				}
			default:
				return false, ast.ErrASTDomain
			}
		}
		return true, nil
	}
	kept := lemma[:1]
	for _, literal := range lemma[1:] {
		remove, err := implied(literal)
		if err != nil {
			return nil, err
		}
		if remove {
			marks[literal.Var()] = true
			temporary = append(temporary, literal.Var())
			r.state.statistics.NumMinimizedLits++
		} else {
			kept = append(kept, literal)
		}
	}
	for _, variable := range temporary {
		marks[variable] = false
	}
	return kept, nil
}

func NewSMTConflictResolver(state *SMTInternalizerState) (*SMTConflictResolver, error) {
	if state == nil || state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	proof, err := state.manager.NewRef(ast.ASTHandle{})
	if err != nil {
		return nil, err
	}
	expander := NewSMTAntecedentExpander()
	for variable := int32(SMTFirstBoolVar); int(variable) < len(state.booleanData); variable++ {
		if !state.booleanData[variable].IsENode() {
			continue
		}
		node := state.context.enodes[state.boolExpr[variable]]
		if node == nil {
			_ = state.manager.DelRef(&proof)
			return nil, ast.ErrASTDomain
		}
		if err = expander.SetENodeLiteral(node, NewSMTLiteral(variable, false)); err != nil {
			_ = state.manager.DelRef(&proof)
			return nil, err
		}
	}
	return &SMTConflictResolver{state: state, expander: expander, lemmaProof: proof}, nil
}

// LemmaProof returns the resolver-owned proof produced by the most recent
// Resolve call. It is null when proof generation is disabled.
func (r *SMTConflictResolver) LemmaProof() (ast.ASTHandle, error) {
	if r == nil || r.state == nil || r.closed {
		return ast.ASTHandle{}, ast.ErrASTHandle
	}
	return r.state.manager.RefGet(r.lemmaProof)
}

func (r *SMTConflictResolver) Close() error {
	if r == nil {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	if r.state == nil {
		return ast.ErrASTHandle
	}
	if r.closePhase == 0 {
		if err := r.state.manager.DelRef(&r.lemmaProof); err != nil {
			return err
		}
		r.closePhase = 1
	}
	r.expander = nil
	r.state = nil
	r.closed = true
	return nil
}

// UnsatCore traces a base conflict back through Boolean and theory
// justifications and returns only assumption literals.
func (r *SMTConflictResolver) UnsatCore(conflict SMTBooleanJustification, notLiteral SMTLiteral) ([]SMTLiteral, error) {
	if r == nil || r.state == nil || r.expander == nil || r.state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	seen := make([]bool, len(r.state.booleanData))
	queue := make([]SMTLiteral, 0)
	core := make([]SMTLiteral, 0)
	add := func(literal SMTLiteral) error {
		variable := literal.Var()
		if variable < SMTFirstBoolVar || int(variable) >= len(seen) {
			return ast.ErrASTDomain
		}
		if !seen[variable] {
			seen[variable] = true
			queue = append(queue, literal)
		}
		return nil
	}
	var expand func(SMTLiteral, SMTBooleanJustification) error
	expand = func(consequent SMTLiteral, justification SMTBooleanJustification) error {
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
					if err := add(clause.literals[0].Negated()); err != nil {
						return err
					}
					start = 2
				} else {
					return ast.ErrASTDomain
				}
			}
			for _, literal := range clause.literals[start:] {
				if err := add(literal.Negated()); err != nil {
					return err
				}
			}
			if clause.justification != nil {
				antecedents, err := r.expander.ExpandJustification(clause.justification)
				if err != nil {
					return err
				}
				for _, literal := range antecedents {
					if err = add(literal); err != nil {
						return err
					}
				}
			}
		case SMTBooleanJustificationBinaryClause:
			literal, ok := justification.Literal()
			if !ok {
				return ast.ErrASTDomain
			}
			return add(literal)
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
				if err = add(literal); err != nil {
					return err
				}
			}
		default:
			return ast.ErrASTDomain
		}
		return nil
	}
	if notLiteral != SMTNullLiteral {
		if err := add(notLiteral); err != nil {
			return nil, err
		}
	}
	if err := expand(SMTFalseLiteral, conflict); err != nil {
		return nil, err
	}
	for head := 0; head < len(queue); head++ {
		literal := queue[head]
		variable := literal.Var()
		if r.state.booleanData[variable].Assumption {
			core = append(core, literal)
			continue
		}
		assigned := NewSMTLiteral(variable, r.state.assignment[NewSMTLiteral(variable, false).Index()] != lbool.True)
		if err := expand(assigned, r.state.booleanData[variable].Justification()); err != nil {
			return nil, err
		}
	}
	return core, nil
}

func (r *SMTConflictResolver) levels() []uint32 {
	levels := make([]uint32, len(r.state.booleanData))
	for variable := int32(SMTFirstBoolVar); int(variable) < len(levels); variable++ {
		levels[variable] = r.state.booleanData[variable].ScopeLevel
	}
	return levels
}

func (r *SMTConflictResolver) Resolve(conflict SMTBooleanJustification, notLiteral SMTLiteral, baseLevel uint32) (SMTConflictResolution, error) {
	if r == nil || r.state == nil || r.expander == nil || r.closed || r.state.open() != nil {
		return SMTConflictResolution{}, ast.ErrASTHandle
	}
	if err := r.state.manager.ResetRef(r.lemmaProof); err != nil {
		return SMTConflictResolution{}, err
	}
	originalConflict := conflict
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
		proof, proofErr := r.conflictProof(originalConflict, notLiteral, nil)
		if proofErr != nil {
			return SMTConflictResolution{}, proofErr
		}
		if !proof.IsNull() {
			if err = r.state.manager.SetRef(r.lemmaProof, proof); err != nil {
				return SMTConflictResolution{}, err
			}
		}
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
	lemma, err = r.minimizeLemma(lemma, marks, levels, baseLevel)
	if err != nil {
		return SMTConflictResolution{}, err
	}
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
	proof, proofErr := r.conflictProof(originalConflict, notLiteral, lemma)
	if proofErr != nil {
		return SMTConflictResolution{}, proofErr
	}
	if !proof.IsNull() {
		if err = r.state.manager.SetRef(r.lemmaProof, proof); err != nil {
			return SMTConflictResolution{}, errors.Join(err, r.state.manager.ResetRef(r.lemmaProof))
		}
	}
	return result, nil
}
