// Source: src/smt/smt_case_split_queue.cpp activity-based case selection used
// by src/smt/smt_context.cpp decide().
package smtcore

import (
	"sort"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

// SMTActivityCaseSplitQueue selects the retained search strategy. Activity
// and relevancy strategies scan Go slices; strategy 6 uses the context's
// persistent concrete priority heap and its variable lifecycle callbacks.
type SMTActivityCaseSplitQueue struct {
	state                                 *SMTInternalizerState
	phaseSelection, cachingOn, cachingOff uint32
	randomVariableFrequency               float64
	strategy                              uint32
	activeCreationLimit                   uint64
	relevantInitialCount                  int
	relCaseSplitOrder                     uint32
	lookaheadDiseq                        bool
	phaseCounter                          uint32
	cacheEnabled, phaseDefault            bool
}

func NewSMTActivityCaseSplitQueue(state *SMTInternalizerState) (*SMTActivityCaseSplitQueue, error) {
	if state == nil || state.open() != nil {
		return nil, ast.ErrASTHandle
	}
	return &SMTActivityCaseSplitQueue{state: state, phaseSelection: 3, cachingOn: 400, cachingOff: 100, cacheEnabled: true}, nil
}

func (q *SMTActivityCaseSplitQueue) ConfigurePhases(state SMTParamsState) error {
	if q == nil || q.state == nil || state.PhaseSelection > 6 {
		return ast.ErrASTDomain
	}
	q.phaseSelection = state.PhaseSelection
	q.cachingOn = state.PhaseCachingOn
	q.cachingOff = state.PhaseCachingOff
	q.randomVariableFrequency = state.RandomVarFreq
	q.strategy = state.CaseSplitStrategy
	if q.strategy >= 3 && q.strategy <= 5 && (state.RelevancyLvl < 2 || state.AutoConfig) {
		q.strategy = 0
	}
	q.activeCreationLimit = q.state.nextBooleanCreation
	q.relCaseSplitOrder = state.RelCaseSplitOrder
	q.lookaheadDiseq = state.LookaheadDiseq
	q.relevantInitialCount = 0
	if q.state.relevancy != nil {
		q.relevantInitialCount = len(q.state.relevancy.RelevantExpressions())
	}
	q.phaseCounter = 0
	q.cacheEnabled = true
	q.phaseDefault = false
	if q.strategy == 6 {
		return q.state.ensureTheoryAwareBranching()
	}
	return nil
}

func (q *SMTActivityCaseSplitQueue) RecordConflict() {
	q.phaseCounter++
	limit := q.cachingOff
	if q.cacheEnabled {
		limit = q.cachingOn
	}
	if q.phaseCounter < limit {
		return
	}
	q.phaseCounter = 0
	q.cacheEnabled = !q.cacheEnabled
	if q.phaseSelection == 4 {
		q.phaseDefault = !q.phaseDefault
	}
}

func (q *SMTActivityCaseSplitQueue) PrepareConflict() error {
	if q == nil || q.state == nil {
		return ast.ErrASTHandle
	}
	if q.phaseSelection != 3 && q.phaseSelection != 4 {
		return nil
	}
	start := 0
	if q.state.scopeLevel != 0 {
		if len(q.state.scopes) == 0 {
			return ast.ErrASTDomain
		}
		start = q.state.scopes[len(q.state.scopes)-1].assignedLimit
	}
	for _, literal := range q.state.assignedLiterals[start:] {
		q.state.booleanData[literal.Var()].PhaseAvailable = false
	}
	return nil
}

func (q *SMTActivityCaseSplitQueue) occurrencePhase(variable int32) bool {
	positive, negative := 0, 0
	for i := range q.state.clauses {
		record := &q.state.clauses[i]
		literals := record.Literals
		if record.clause != nil {
			literals = record.clause.literals
		}
		for _, literal := range literals {
			if literal.Var() != variable {
				continue
			}
			if literal.Sign() {
				negative++
			} else {
				positive++
			}
		}
	}
	return positive > negative
}

// NextCaseSplit returns the highest-activity unassigned Boolean variable and
// the source phase preference. Ties are deterministic in Go variable order.
func (q *SMTActivityCaseSplitQueue) NextCaseSplit() (int32, lbool.LBool, error) {
	if q == nil || q.state == nil {
		return SMTNullBoolVar, lbool.Undef, ast.ErrASTHandle
	}
	if err := q.state.open(); err != nil {
		return SMTNullBoolVar, lbool.Undef, err
	}
	next, preferred, err := q.selectVariable()
	if err != nil {
		return SMTNullBoolVar, lbool.Undef, err
	}
	if next == SMTNullBoolVar {
		return next, lbool.Undef, nil
	}
	if preferred != lbool.Undef {
		return next, preferred, nil
	}
	data := &q.state.booleanData[next]
	if data.IsQuantifier() {
		return next, lbool.False, nil
	}
	if data.TryTrueFirst() {
		return next, lbool.True, nil
	}
	phase := false
	switch q.phaseSelection {
	case 0:
	case 1:
		phase = true
	case 2, 3, 4:
		phase = q.phaseDefault
		if q.cacheEnabled && data.PhaseAvailable {
			phase = data.Phase
		}
	case 5:
		phase = q.state.random.Next()%2 == 0
	case 6:
		phase = q.occurrencePhase(next)
	default:
		return SMTNullBoolVar, lbool.Undef, ast.ErrASTDomain
	}
	if phase {
		return next, lbool.True, nil
	}
	return next, lbool.False, nil
}

func (q *SMTActivityCaseSplitQueue) selectVariable() (int32, lbool.LBool, error) {
	switch q.strategy {
	case 6:
		variable, err := q.state.theoryAwareBranching.next(q.randomVariableFrequency)
		return variable, lbool.Undef, err
	case 3:
		next, phase, err := q.nextRelevanceFIFO()
		if err == nil && next != SMTNullBoolVar && phase != lbool.False && q.lookaheadDiseq && q.selectedEqualityDisequal(next) {
			phase = lbool.False
		}
		return next, phase, err
	case 4:
		if next, err := q.randomVariable(0.02); err != nil || next != SMTNullBoolVar {
			return next, lbool.Undef, err
		}
		if next, phase, err := q.relevantVariable(true, false); err != nil || next != SMTNullBoolVar {
			return next, phase, err
		}
		next, err := q.highestActivityRange(q.activeCreationLimit, ^uint64(0))
		return next, lbool.Undef, err
	case 5:
		return q.relevantVariable(false, true)
	default:
		next, err := q.randomVariable(q.randomVariableFrequency)
		if err != nil || next != SMTNullBoolVar {
			return next, lbool.Undef, err
		}
		creationLimit := ^uint64(0)
		if q.strategy == 1 {
			creationLimit = q.activeCreationLimit
		}
		next, err = q.highestActivity(creationLimit)
		if err != nil {
			return SMTNullBoolVar, lbool.Undef, err
		}
		if next == SMTNullBoolVar && q.strategy == 1 && q.activeCreationLimit < q.state.nextBooleanCreation {
			q.activeCreationLimit = q.state.nextBooleanCreation
			next, err = q.highestActivity(q.activeCreationLimit)
		}
		return next, lbool.Undef, err
	}
}

// The Go no-progress check needs availability, not a consumed decision.
// Strategy 6's native next_case_split removes a selected heap entry.
func (q *SMTActivityCaseSplitQueue) hasCaseSplit() (bool, error) {
	if q == nil || q.state == nil {
		return false, ast.ErrASTHandle
	}
	if err := q.state.open(); err != nil {
		return false, err
	}
	if q.strategy != 6 {
		variable, _, err := q.NextCaseSplit()
		return variable != SMTNullBoolVar, err
	}
	values, err := q.state.theoryAwareBranching.heap.AppendValues(nil)
	if err != nil {
		return false, err
	}
	for _, variable := range values {
		assignment, err := q.state.Assignment(NewSMTLiteral(variable, false))
		if err != nil {
			return false, err
		}
		if assignment == lbool.Undef {
			return true, nil
		}
	}
	return false, nil
}

func (q *SMTActivityCaseSplitQueue) selectedEqualityDisequal(variable int32) bool {
	if variable < SMTFirstBoolVar || int(variable) >= len(q.state.boolExpr) {
		return false
	}
	node := q.state.context.enodes[q.state.boolExpr[variable]]
	if node == nil || !node.IsEquality() || node.NumArgs() != 2 {
		return false
	}
	left, leftErr := node.Arg(0)
	right, rightErr := node.Arg(1)
	return leftErr == nil && rightErr == nil && q.extendedDisequal(left, right, 2)
}

func (q *SMTActivityCaseSplitQueue) extendedDisequal(first, second *SMTEnode, depth uint32) bool {
	if first == nil || second == nil || first.Root() == second.Root() {
		return false
	}
	firstRoot, secondRoot := first.Root(), second.Root()
	if firstRoot.IsInterpreted() && secondRoot.IsInterpreted() || q.state.context.isDisequal(firstRoot, secondRoot) {
		return true
	}
	if depth == 0 {
		return false
	}
	if firstRoot.NumParents() > secondRoot.NumParents() {
		firstRoot, secondRoot = secondRoot, firstRoot
	}
	for _, firstParent := range firstRoot.Parents() {
		if firstParent == nil || firstParent.IsEquality() || !firstParent.IsCongruenceRoot() ||
			q.state.relevancy != nil && !q.state.relevancy.IsRelevant(firstParent.Owner()) {
			continue
		}
		for _, secondParent := range secondRoot.Parents() {
			if secondParent == nil || secondParent.IsEquality() || !secondParent.IsCongruenceRoot() ||
				q.state.relevancy != nil && !q.state.relevancy.IsRelevant(secondParent.Owner()) ||
				firstParent.Root() == secondParent.Root() || firstParent.Declaration() != secondParent.Declaration() ||
				firstParent.NumArgs() != secondParent.NumArgs() {
				continue
			}
			almostCongruent := true
			for index := uint32(0); index < firstParent.NumArgs(); index++ {
				firstArg, firstErr := firstParent.Arg(index)
				secondArg, secondErr := secondParent.Arg(index)
				if firstErr != nil || secondErr != nil {
					almostCongruent = false
					break
				}
				firstArg, secondArg = firstArg.Root(), secondArg.Root()
				if firstArg == secondArg {
					continue
				}
				firstOriginal := firstArg == firstRoot || firstArg == secondRoot
				secondOriginal := secondArg == firstRoot || secondArg == secondRoot
				if !firstOriginal || !secondOriginal {
					almostCongruent = false
					break
				}
			}
			if almostCongruent && q.extendedDisequal(firstParent, secondParent, depth-1) {
				return true
			}
		}
	}
	return false
}

func (q *SMTActivityCaseSplitQueue) randomVariable(frequency float64) (int32, error) {
	threshold := int(frequency * float64(z3rand.Max))
	if int(q.state.random.Next()) >= threshold || len(q.state.boolExpr) == 0 {
		return SMTNullBoolVar, nil
	}
	candidate := int32(q.state.random.Next() % uint32(len(q.state.boolExpr)))
	if candidate < SMTFirstBoolVar {
		return SMTNullBoolVar, nil
	}
	value, err := q.state.Assignment(NewSMTLiteral(candidate, false))
	if err != nil || value != lbool.Undef {
		return SMTNullBoolVar, err
	}
	return candidate, nil
}

type smtRelevantCaseSplit struct {
	expression ast.ASTHandle
	index      int
	generation uint32
	search     bool
}

func (q *SMTActivityCaseSplitQueue) relevantVariable(baseOnly, generationOrder bool) (int32, lbool.LBool, error) {
	if q.state.relevancy == nil {
		return SMTNullBoolVar, lbool.Undef, nil
	}
	expressions := q.state.relevancy.RelevantExpressions()
	entries := make([]smtRelevantCaseSplit, 0, len(expressions))
	for index, expression := range expressions {
		search := index >= q.relevantInitialCount
		if variable, found := q.state.BoolVar(expression); found {
			search = q.state.booleanCreation[variable] > q.activeCreationLimit
		}
		if baseOnly && search {
			continue
		}
		entry := smtRelevantCaseSplit{expression: expression, index: index, search: search}
		if generationOrder && search {
			generation, err := q.expressionGeneration(expression)
			if err != nil {
				return SMTNullBoolVar, lbool.Undef, err
			}
			entry.generation = generation
		}
		entries = append(entries, entry)
	}
	if generationOrder {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].search != entries[j].search {
				return !entries[i].search
			}
			if entries[i].search && entries[i].generation != entries[j].generation {
				return entries[i].generation < entries[j].generation
			}
			return entries[i].index < entries[j].index
		})
	}
	for _, entry := range entries {
		next, phase, usable, err := q.relevantCandidate(entry.expression)
		if err != nil {
			return SMTNullBoolVar, lbool.Undef, err
		}
		if usable {
			return next, phase, nil
		}
	}
	return SMTNullBoolVar, lbool.Undef, nil
}

func (q *SMTActivityCaseSplitQueue) relevantCandidate(expression ast.ASTHandle) (int32, lbool.LBool, bool, error) {
	boolean, err := q.state.isBoolean(expression)
	if err != nil || !boolean {
		return SMTNullBoolVar, lbool.Undef, false, err
	}
	op, basic, err := q.state.basicOperation(expression)
	if err != nil {
		return SMTNullBoolVar, lbool.Undef, false, err
	}
	isOr, isAnd := basic && op == ast.ASTOpOr, basic && op == ast.ASTOpAnd
	variable, internalized := q.state.BoolVar(expression)
	if !internalized && !isOr {
		return SMTNullBoolVar, lbool.Undef, false, nil
	}
	value := lbool.True
	if internalized {
		value, err = q.state.Assignment(NewSMTLiteral(variable, false))
		if err != nil {
			return SMTNullBoolVar, lbool.Undef, false, err
		}
	}
	if isOr && value == lbool.True || isAnd && value == lbool.False {
		child, found, err := q.relevantChild(expression, value)
		if err != nil || !found {
			return SMTNullBoolVar, lbool.Undef, false, err
		}
		literal, err := q.state.Literal(child)
		if err != nil {
			if err = q.state.Internalize(child, false); err != nil {
				return SMTNullBoolVar, lbool.Undef, false, err
			}
			literal, err = q.state.Literal(child)
		}
		if err != nil {
			return SMTNullBoolVar, lbool.Undef, false, err
		}
		phase := lbool.True
		if literal.Sign() {
			phase = lbool.False
		}
		return literal.Var(), phase, true, nil
	}
	if internalized && value == lbool.Undef {
		return variable, lbool.Undef, true, nil
	}
	return SMTNullBoolVar, lbool.Undef, false, nil
}

func (q *SMTActivityCaseSplitQueue) relevantChild(parent ast.ASTHandle, parentValue lbool.LBool) (ast.ASTHandle, bool, error) {
	count, err := q.state.manager.GetNumArgs(parent)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	undefined := make([]ast.ASTHandle, 0, count)
	for index := uint32(0); index < count; index++ {
		child, err := q.state.manager.GetArg(parent, index)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
		literal, err := q.state.Literal(child)
		if err != nil {
			continue
		}
		value, err := q.state.Assignment(literal)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
		if value == parentValue {
			return ast.ASTHandle{}, false, nil
		}
		if value == lbool.Undef {
			if q.relCaseSplitOrder != 0 || len(undefined) == 0 {
				undefined = append(undefined, child)
			}
		}
	}
	if len(undefined) == 0 {
		return ast.ASTHandle{}, false, nil
	}
	if q.relCaseSplitOrder == 1 && len(undefined) > 1 {
		return undefined[q.state.random.Next()%uint32(len(undefined))], true, nil
	}
	return undefined[len(undefined)-1], true, nil
}

func (q *SMTActivityCaseSplitQueue) expressionGeneration(expression ast.ASTHandle) (uint32, error) {
	stack := []ast.ASTHandle{expression}
	seen := make(map[ast.ASTHandle]bool)
	var maximum uint32
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[current] {
			continue
		}
		seen[current] = true
		if node := q.state.context.enodes[current]; node != nil {
			if node.Generation() > maximum {
				maximum = node.Generation()
			}
			continue
		}
		kind, err := q.state.manager.GetKind(current)
		if err != nil {
			return 0, err
		}
		if kind != ast.ASTApp {
			continue
		}
		count, err := q.state.manager.GetNumArgs(current)
		if err != nil {
			return 0, err
		}
		for index := uint32(0); index < count; index++ {
			child, err := q.state.manager.GetArg(current, index)
			if err != nil {
				return 0, err
			}
			stack = append(stack, child)
		}
	}
	return maximum, nil
}

func (q *SMTActivityCaseSplitQueue) highestActivity(creationLimit uint64) (int32, error) {
	return q.highestActivityRange(0, creationLimit)
}

func (q *SMTActivityCaseSplitQueue) highestActivityRange(creationAfter, creationLimit uint64) (int32, error) {
	next := SMTNullBoolVar
	best := 0.0
	for variable := int32(SMTFirstBoolVar); int(variable) < len(q.state.boolExpr); variable++ {
		creation := q.state.booleanCreation[variable]
		if creation <= creationAfter || creation > creationLimit {
			continue
		}
		value, err := q.state.Assignment(NewSMTLiteral(variable, false))
		if err != nil {
			return SMTNullBoolVar, err
		}
		if value != lbool.Undef {
			continue
		}
		activity := q.state.activity[variable]
		if next == SMTNullBoolVar || activity > best {
			next, best = variable, activity
		}
	}
	return next, nil
}
