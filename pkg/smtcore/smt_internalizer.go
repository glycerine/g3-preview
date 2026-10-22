// Source: src/smt/smt_internalizer.cpp assertion, formula, gate, and default
// uninterpreted-term internalization.
package smtcore

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

const smtDeepExpressionThreshold uint32 = 1024

func (s *SMTInternalizerState) SetRelevancy(propagator *SMTRelevancyPropagator) error {
	if err := s.open(); err != nil {
		return err
	}
	if propagator != nil && propagator.context != s.context {
		return ast.ErrASTDomain
	}
	s.relevancy = propagator
	return nil
}

func (s *SMTInternalizerState) isBoolean(expression ast.ASTHandle) (bool, error) {
	sort, err := s.manager.GetSort(expression)
	if err != nil {
		return false, err
	}
	boolean, err := s.manager.MkBoolSort()
	return sort == boolean, err
}

func (s *SMTInternalizerState) basicOperation(expression ast.ASTHandle) (int32, bool, error) {
	kind, err := s.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return 0, false, err
	}
	family, err := s.manager.GetFamilyID(expression)
	if err != nil {
		return 0, false, err
	}
	if family != s.manager.BasicFamilyID() {
		return 0, false, nil
	}
	op, err := s.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func (s *SMTInternalizerState) expressionTheory(expression ast.ASTHandle) (SMTTheoryInternalizer, error) {
	family, err := s.manager.GetFamilyID(expression)
	if err != nil {
		return nil, err
	}
	return s.theories[family], nil
}

func (s *SMTInternalizerState) applySortConstraint(expression ast.ASTHandle, node *SMTEnode) error {
	sort, err := s.manager.GetSort(expression)
	if err != nil {
		return err
	}
	family, err := s.manager.GetFamilyID(sort)
	if err != nil {
		return err
	}
	if theory := s.theories[family]; theory != nil {
		return theory.ApplySortConstraint(node, sort)
	}
	return nil
}

func smtInternalizerGateOperation(operation int32) bool {
	return operation == ast.ASTOpAnd || operation == ast.ASTOpOr || operation == ast.ASTOpIff || operation == ast.ASTOpIte
}

func (s *SMTInternalizerState) Clauses() []SMTInternalizedClause {
	if s == nil {
		return nil
	}
	result := make([]SMTInternalizedClause, len(s.clauses))
	for index := range s.clauses {
		result[index] = s.clauses[index]
		literals := s.clauses[index].Literals
		if s.clauses[index].clause != nil {
			literals = s.clauses[index].clause.Literals()
		}
		result[index].Literals = append([]SMTLiteral(nil), literals...)
		result[index].clause = nil
		result[index].owner = nil
	}
	return result
}

func (s *SMTInternalizerState) Inconsistent() bool { return s != nil && s.inconsistent }

func (s *SMTInternalizerState) assignLiteral(literal SMTLiteral) error {
	return s.AssignLiteral(literal, NewSMTBooleanAxiom())
}

func (s *SMTInternalizerState) addClause(literals []SMTLiteral, root bool, proof ast.ASTHandle) error {
	return s.addClauseKind(literals, root, SMTClauseAux, proof)
}

func (s *SMTInternalizerState) addClauseKind(literals []SMTLiteral, root bool, kind SMTClauseKind, proof ast.ASTHandle) error {
	if s.inconsistent {
		return nil
	}
	seen := make(map[SMTLiteral]struct{}, len(literals))
	kept := make([]SMTLiteral, 0, len(literals))
	for _, literal := range literals {
		if literal == SMTTrueLiteral {
			return nil
		}
		if literal == SMTFalseLiteral {
			continue
		}
		if _, found := seen[literal.Negated()]; found {
			return nil
		}
		if _, found := seen[literal]; found {
			continue
		}
		value, err := s.Assignment(literal)
		if err != nil {
			return err
		}
		if value == lbool.True {
			return nil
		}
		if value == lbool.False {
			continue
		}
		seen[literal] = struct{}{}
		kept = append(kept, literal)
	}
	s.statistics.NumMkLits += uint32(len(kept))
	if len(kept) == 0 {
		s.inconsistent = true
		s.conflict = NewSMTBooleanAxiom()
		s.statistics.NumConflicts++
		return nil
	}
	if len(kept) == 1 {
		return s.assignLiteral(kept[0])
	}
	clause, owner, err := s.makeClause(kept, kind, proof)
	if err != nil {
		return err
	}
	s.clauses = append(s.clauses, SMTInternalizedClause{Literals: kept, Root: root, Proof: proof, clause: clause, owner: owner})
	s.statistics.NumMkClause++
	if len(kept) == 2 {
		s.statistics.NumMkBinClause++
	}
	return nil
}

// AddLearnedClause installs a conflict lemma into the current logical scope.
func (s *SMTInternalizerState) AddLearnedClause(literals []SMTLiteral) error {
	if err := s.open(); err != nil {
		return err
	}
	if len(literals) == 0 {
		return ast.ErrASTDomain
	}
	return s.addClauseKind(literals, false, SMTClauseLearned, ast.ASTHandle{})
}

func (s *SMTInternalizerState) markRelevant(expression ast.ASTHandle) error {
	if s.relevancy == nil {
		return s.context.SetRelevant(expression, true)
	}
	return s.relevancy.MarkRelevant(expression)
}

func (s *SMTInternalizerState) ensureBooleanEnode(expression ast.ASTHandle, variable int32, suppressArguments, mergeTruthValue, congruenceEnabled bool) error {
	node := s.context.enodes[expression]
	if node == nil {
		var err error
		node, err = s.MkEnode(expression, suppressArguments, mergeTruthValue, congruenceEnabled)
		if err != nil {
			return err
		}
	} else if mergeTruthValue {
		node.mergeTruthValue = true
	}
	data, err := s.BooleanData(variable)
	if err != nil {
		return err
	}
	data.SetENode()
	return nil
}

func (s *SMTInternalizerState) Internalize(expression ast.ASTHandle, gateContext bool) error {
	if err := s.open(); err != nil {
		return err
	}
	kind, err := s.manager.GetKind(expression)
	if err != nil {
		return err
	}
	if kind == ast.ASTVar {
		return ast.ErrASTDomain
	}
	boolean, err := s.isBoolean(expression)
	if err != nil {
		return err
	}
	if boolean {
		return s.internalizeFormula(expression, gateContext)
	}
	if kind != ast.ASTApp || gateContext {
		return ast.ErrASTDomain
	}
	return s.internalizeTerm(expression)
}

func (s *SMTInternalizerState) internalizeFormula(expression ast.ASTHandle, gateContext bool) error {
	if s.manager.IsTrue(expression) || s.manager.IsFalse(expression) {
		return nil
	}
	op, basic, err := s.basicOperation(expression)
	if err != nil {
		return err
	}
	if basic && op == ast.ASTOpNot && gateContext {
		child, err := s.manager.GetArg(expression, 0)
		if err != nil {
			return err
		}
		return s.Internalize(child, true)
	}
	if variable, found := s.boolVar[expression]; found {
		if !gateContext {
			return s.ensureBooleanEnode(expression, variable, true, true, false)
		}
		return nil
	}
	if kind, err := s.manager.GetKind(expression); err != nil {
		return err
	} else if kind == ast.ASTQuantifier {
		return s.internalizeQuantifier(expression, gateContext)
	}
	if basic && op == ast.ASTOpDistinct {
		return s.internalizeDistinct(expression, gateContext)
	}
	if !basic {
		theory, err := s.expressionTheory(expression)
		if err != nil {
			return err
		}
		if theory != nil {
			handled, err := theory.InternalizeAtom(expression, gateContext)
			if err != nil {
				return err
			}
			if handled {
				variable, found := s.boolVar[expression]
				if !found {
					return ast.ErrASTDomain
				}
				if !gateContext {
					if err = s.ensureBooleanEnode(expression, variable, true, true, false); err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
	if err = s.internalizeFormulaCore(expression, gateContext); err != nil {
		return err
	}
	if basic && op == ast.ASTOpEq {
		variable, _ := s.boolVar[expression]
		data, err := s.BooleanData(variable)
		if err != nil {
			return err
		}
		data.SetEquality()
		left, err := s.manager.GetArg(expression, 0)
		if err != nil {
			return err
		}
		sort, err := s.manager.GetSort(left)
		if err != nil {
			return err
		}
		family, err := s.manager.GetFamilyID(sort)
		if err != nil {
			return err
		}
		if theory := s.theories[family]; theory != nil {
			if err = theory.InternalizeEquality(expression, variable); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SMTInternalizerState) internalizeQuantifier(expression ast.ASTHandle, gateContext bool) error {
	if !gateContext || s.quantifiers == nil {
		return ast.ErrASTDomain
	}
	forall, err := s.manager.IsForall(expression)
	if err != nil || !forall {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	variable, err := s.MkBoolVar(expression)
	if err != nil {
		return err
	}
	data, err := s.BooleanData(variable)
	if err != nil {
		return err
	}
	data.SetQuantifier()
	generation := s.generation
	if cached, found := s.cachedGeneration[expression]; found {
		generation = cached
	}
	return s.quantifiers.Add(expression, generation)
}

func (s *SMTInternalizerState) internalizeDistinct(expression ast.ASTHandle, gateContext bool) error {
	count, err := s.manager.GetNumArgs(expression)
	if err != nil {
		return err
	}
	arguments := make([]ast.ASTHandle, count)
	for index := range arguments {
		arguments[index], err = s.manager.GetArg(expression, uint32(index))
		if err != nil {
			return err
		}
	}
	expanded, err := s.manager.MkDistinctExpanded(arguments)
	if err != nil {
		return err
	}
	if err = s.Internalize(expanded, true); err != nil {
		return err
	}
	variable, err := s.MkBoolVar(expression)
	if err != nil {
		return err
	}
	literal := NewSMTLiteral(variable, false)
	definition, err := s.Literal(expanded)
	if err != nil {
		return err
	}
	if err = s.addClause([]SMTLiteral{literal.Negated(), definition}, false, ast.ASTHandle{}); err != nil {
		return err
	}
	if err = s.addClause([]SMTLiteral{literal, definition.Negated()}, false, ast.ASTHandle{}); err != nil {
		return err
	}
	if s.relevancy != nil {
		if err = s.relevancy.AddDependency(expression, expanded); err != nil {
			return err
		}
	}
	if !gateContext {
		return s.ensureBooleanEnode(expression, variable, true, true, false)
	}
	return nil
}

func (s *SMTInternalizerState) internalizeFormulaCore(expression ast.ASTHandle, gateContext bool) error {
	op, basic, err := s.basicOperation(expression)
	if err != nil {
		return err
	}
	isGate := basic && (smtInternalizerGateOperation(op) || op == ast.ASTOpNot)
	count, err := s.manager.GetNumArgs(expression)
	if err != nil {
		return err
	}
	for index := uint32(0); index < count; index++ {
		argument, err := s.manager.GetArg(expression, index)
		if err != nil {
			return err
		}
		if err = s.Internalize(argument, isGate); err != nil {
			return fmt.Errorf("internalize formula child %d: %w", index, err)
		}
	}
	variable, isNew := s.boolVar[expression]
	if !isNew {
		variable, err = s.MkBoolVar(expression)
		if err != nil {
			return fmt.Errorf("internalize formula Boolean variable: %w", err)
		}
		isNew = true
	} else {
		isNew = false
	}
	if s.context.enodes[expression] == nil && (!gateContext || !isGate && count > 0) {
		if err = s.ensureBooleanEnode(expression, variable, isGate, !gateContext, true); err != nil {
			return fmt.Errorf("internalize formula enode: %w", err)
		}
	}
	if isNew && basic && (smtInternalizerGateOperation(op) || op == ast.ASTOpNot) {
		if err = s.addGateConstraints(expression, op); err != nil {
			return fmt.Errorf("internalize formula gate constraints: %w", err)
		}
	}
	return nil
}

func (s *SMTInternalizerState) argumentLiterals(expression ast.ASTHandle) ([]SMTLiteral, error) {
	count, err := s.manager.GetNumArgs(expression)
	if err != nil {
		return nil, err
	}
	result := make([]SMTLiteral, count)
	for index := range result {
		argument, err := s.manager.GetArg(expression, uint32(index))
		if err != nil {
			return nil, err
		}
		result[index], err = s.Literal(argument)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *SMTInternalizerState) addGateConstraints(expression ast.ASTHandle, operation int32) error {
	variable := s.boolVar[expression]
	literal := NewSMTLiteral(variable, false)
	arguments, err := s.argumentLiterals(expression)
	if err != nil {
		return err
	}
	add := func(values ...SMTLiteral) error { return s.addClause(values, false, ast.ASTHandle{}) }
	switch operation {
	case ast.ASTOpNot:
		if len(arguments) != 1 {
			return ast.ErrASTDomain
		}
		if err = add(literal.Negated(), arguments[0].Negated()); err != nil {
			return err
		}
		return add(literal, arguments[0])
	case ast.ASTOpAnd:
		for _, argument := range arguments {
			if err = add(literal.Negated(), argument); err != nil {
				return err
			}
		}
		last := []SMTLiteral{literal}
		for _, argument := range arguments {
			last = append(last, argument.Negated())
		}
		if err = s.addClause(last, false, ast.ASTHandle{}); err != nil {
			return err
		}
		if s.relevancy != nil {
			return s.relevancy.AddAndWatches(expression)
		}
	case ast.ASTOpOr:
		for _, argument := range arguments {
			if err = add(literal, argument.Negated()); err != nil {
				return err
			}
		}
		last := []SMTLiteral{literal.Negated()}
		last = append(last, arguments...)
		if err = s.addClause(last, false, ast.ASTHandle{}); err != nil {
			return err
		}
		if s.relevancy != nil {
			return s.relevancy.AddOrWatches(expression)
		}
	case ast.ASTOpIff:
		if len(arguments) != 2 {
			return ast.ErrASTDomain
		}
		for _, clause := range [][]SMTLiteral{
			{literal.Negated(), arguments[0], arguments[1].Negated()}, {literal.Negated(), arguments[0].Negated(), arguments[1]},
			{literal, arguments[0], arguments[1]}, {literal, arguments[0].Negated(), arguments[1].Negated()},
		} {
			if err = s.addClause(clause, false, ast.ASTHandle{}); err != nil {
				return err
			}
		}
	case ast.ASTOpIte:
		if len(arguments) != 3 {
			return ast.ErrASTDomain
		}
		for _, clause := range [][]SMTLiteral{
			{literal.Negated(), arguments[0].Negated(), arguments[1]}, {literal.Negated(), arguments[0], arguments[2]},
			{literal, arguments[0].Negated(), arguments[1].Negated()}, {literal, arguments[0], arguments[2].Negated()},
		} {
			if err = s.addClause(clause, false, ast.ASTHandle{}); err != nil {
				return err
			}
		}
		if s.relevancy != nil {
			return s.relevancy.AddITEWatches(expression)
		}
	}
	return nil
}

func (s *SMTInternalizerState) internalizeTerm(expression ast.ASTHandle) error {
	if node := s.context.enodes[expression]; node != nil {
		theory, err := s.expressionTheory(expression)
		if err != nil {
			return err
		}
		if theory != nil && !theory.IsAttachedToVariable(node) {
			handled, err := theory.InternalizeTerm(expression)
			if err != nil {
				return err
			}
			if handled && !theory.IsAttachedToVariable(node) {
				return ast.ErrASTDomain
			}
		}
		return nil
	}
	op, basic, err := s.basicOperation(expression)
	if err != nil {
		return err
	}
	if basic && op == ast.ASTOpIte {
		return s.internalizeTermITE(expression)
	}
	theory, err := s.expressionTheory(expression)
	if err != nil {
		return err
	}
	if theory != nil {
		handled, err := theory.InternalizeTerm(expression)
		if err != nil {
			return err
		}
		if handled {
			node := s.context.enodes[expression]
			if node == nil || !theory.IsAttachedToVariable(node) {
				return ast.ErrASTDomain
			}
			return s.applySortConstraint(expression, node)
		}
	}
	count, err := s.manager.GetNumArgs(expression)
	if err != nil {
		return err
	}
	for index := uint32(0); index < count; index++ {
		argument, err := s.manager.GetArg(expression, index)
		if err != nil {
			return err
		}
		if err = s.Internalize(argument, false); err != nil {
			return err
		}
	}
	node, err := s.MkEnode(expression, false, false, true)
	if err != nil {
		return err
	}
	return s.applySortConstraint(expression, node)
}

func (s *SMTInternalizerState) internalizeTermITE(expression ast.ASTHandle) error {
	condition, err := s.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	thenTerm, err := s.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	elseTerm, err := s.manager.GetArg(expression, 2)
	if err != nil {
		return err
	}
	thenEquality, err := s.manager.MkEq(expression, thenTerm)
	if err != nil {
		return err
	}
	elseEquality, err := s.manager.MkEq(expression, elseTerm)
	if err != nil {
		return err
	}
	if _, err = s.MkEnode(expression, true, false, false); err != nil {
		return fmt.Errorf("internalize term ITE enode: %w", err)
	}
	for _, item := range []struct {
		expression ast.ASTHandle
		gate       bool
	}{{condition, true}, {thenTerm, false}, {elseTerm, false}, {thenEquality, true}, {elseEquality, true}} {
		if err = s.Internalize(item.expression, item.gate); err != nil {
			return fmt.Errorf("internalize term ITE child %d: %w", item.expression.Slot(), err)
		}
	}
	conditionLiteral, err := s.Literal(condition)
	if err != nil {
		return err
	}
	thenLiteral, err := s.Literal(thenEquality)
	if err != nil {
		return err
	}
	elseLiteral, err := s.Literal(elseEquality)
	if err != nil {
		return err
	}
	if err = s.addClause([]SMTLiteral{conditionLiteral.Negated(), thenLiteral}, false, ast.ASTHandle{}); err != nil {
		return fmt.Errorf("internalize term ITE then clause: %w", err)
	}
	if err = s.addClause([]SMTLiteral{conditionLiteral, elseLiteral}, false, ast.ASTHandle{}); err != nil {
		return fmt.Errorf("internalize term ITE else clause: %w", err)
	}
	if s.relevancy != nil {
		return s.relevancy.AddTermITEWatches(expression, thenEquality, elseEquality)
	}
	return nil
}

func (s *SMTInternalizerState) preInternalizeDeep(expression ast.ASTHandle) error {
	depth, err := s.manager.GetDepth(expression)
	if err != nil || depth <= smtDeepExpressionThreshold {
		return err
	}
	order, err := NewSMTInternalizationOrder(s.manager, s.context)
	if err != nil {
		return err
	}
	for existing := range s.boolVar {
		if err = order.SetBooleanInternalized(existing, true); err != nil {
			return err
		}
	}
	for _, theory := range s.theoryOrder {
		if !theory.DefaultInternalizer() {
			if err = order.SetNonDefaultFamily(theory.FamilyID(), true); err != nil {
				return err
			}
		}
	}
	items, err := order.Sort(expression)
	if err != nil {
		return err
	}
	for _, item := range items {
		kind, err := s.manager.GetKind(item.Expression)
		if err != nil {
			return err
		}
		if kind == ast.ASTApp {
			family, err := s.manager.GetFamilyID(item.Expression)
			if err != nil {
				return err
			}
			// Concrete theory internalizers retain the source top-down walk.
			if family != ast.NullFamilyID && family != s.manager.BasicFamilyID() {
				continue
			}
		}
		if err = s.Internalize(item.Expression, item.GateContext); err != nil {
			return fmt.Errorf("pre-internalize deep expression: %w", err)
		}
	}
	return nil
}

func (s *SMTInternalizerState) Assert(expression, proof ast.ASTHandle, generation uint32) error {
	previousGeneration := s.generation
	if err := s.SetGeneration(generation); err != nil {
		return err
	}
	defer func() { s.generation = previousGeneration }()
	boolean, err := s.isBoolean(expression)
	if err != nil || !boolean {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	if err = s.preInternalizeDeep(expression); err != nil {
		return err
	}
	op, basic, err := s.basicOperation(expression)
	if err != nil {
		return err
	}
	if basic && smtInternalizerGateOperation(op) {
		arguments, err := s.argumentHandles(expression)
		if err != nil {
			return err
		}
		switch op {
		case ast.ASTOpAnd:
			return ast.ErrASTDomain
		case ast.ASTOpOr:
			literals := make([]SMTLiteral, len(arguments))
			for index, argument := range arguments {
				if err = s.Internalize(argument, true); err != nil {
					return err
				}
				literals[index], err = s.Literal(argument)
				if err != nil {
					return err
				}
			}
			if err = s.addClause(literals, true, proof); err != nil {
				return err
			}
			if s.relevancy != nil {
				if err = s.relevancy.AddOrWatches(expression); err != nil {
					return err
				}
			}
		case ast.ASTOpIff:
			for _, argument := range arguments {
				if err = s.Internalize(argument, true); err != nil {
					return err
				}
			}
			left, err := s.Literal(arguments[0])
			if err != nil {
				return err
			}
			right, err := s.Literal(arguments[1])
			if err != nil {
				return err
			}
			if err = s.addClause([]SMTLiteral{left, right.Negated()}, true, proof); err != nil {
				return err
			}
			if err = s.addClause([]SMTLiteral{left.Negated(), right}, true, proof); err != nil {
				return err
			}
		case ast.ASTOpIte:
			for _, argument := range arguments {
				if err = s.Internalize(argument, true); err != nil {
					return err
				}
			}
			condition, err := s.Literal(arguments[0])
			if err != nil {
				return err
			}
			thenLiteral, err := s.Literal(arguments[1])
			if err != nil {
				return err
			}
			elseLiteral, err := s.Literal(arguments[2])
			if err != nil {
				return err
			}
			if err = s.addClause([]SMTLiteral{condition.Negated(), thenLiteral}, true, proof); err != nil {
				return err
			}
			if err = s.addClause([]SMTLiteral{condition, elseLiteral}, true, proof); err != nil {
				return err
			}
			if s.relevancy != nil {
				if err = s.relevancy.AddITEWatches(expression); err != nil {
					return err
				}
			}
		}
		return s.markRelevant(expression)
	}
	if err = s.Internalize(expression, true); err != nil {
		return err
	}
	literal, err := s.Literal(expression)
	if err != nil {
		return err
	}
	if err = s.assignLiteral(literal); err != nil {
		return err
	}
	return s.markRelevant(expression)
}

func (s *SMTInternalizerState) argumentHandles(expression ast.ASTHandle) ([]ast.ASTHandle, error) {
	count, err := s.manager.GetNumArgs(expression)
	if err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, count)
	for index := range result {
		result[index], err = s.manager.GetArg(expression, uint32(index))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *SMTInternalizerState) InternalizeInstance(instance *SMTQIInstance) error {
	if instance == nil || instance.manager != s.manager {
		return ast.ErrASTDomain
	}
	if instance.ReducedTrue() {
		return nil
	}
	lemma, err := instance.Lemma()
	if err != nil {
		return err
	}
	proof, err := instance.Proof()
	if err != nil {
		return err
	}
	return s.Assert(lemma, proof, instance.Generation())
}
