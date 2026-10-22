// Source: src/smt/smt_internalizer.cpp Boolean-variable, enode, and literal
// registration methods.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtInternalizerScope struct {
	booleanLimit int
	enodeLimit   int
	pendingLimit int
}

type SMTInternalizerCongruence struct {
	Node     *SMTEnode
	Existing *SMTEnode
	Commuted bool
}

type SMTInternalizerState struct {
	manager          *ast.ASTManager
	context          *SMTCheckerContext
	congruence       *SMTCongruenceTable
	quantifiers      *SMTQuantifierRegistry
	boolVar          map[ast.ASTHandle]int32
	boolExpr         []ast.ASTHandle
	booleanData      []SMTBoolVarData
	activity         []float64
	assignment       []lbool.LBool
	watches          []SMTWatchList
	booleanStack     []ast.ASTHandle
	enodeStack       []*SMTEnode
	scopes           []smtInternalizerScope
	cachedGeneration map[ast.ASTHandle]uint32
	pending          []SMTInternalizerCongruence
	generation       uint32
	scopeLevel       uint32
	statistics       SMTContextStatistics
	closed           bool
}

func NewSMTInternalizerState(manager *ast.ASTManager, context *SMTCheckerContext, quantifiers *SMTQuantifierRegistry) (*SMTInternalizerState, error) {
	if manager == nil || context == nil || context.manager != manager || quantifiers != nil && quantifiers.manager != manager {
		return nil, ast.ErrASTHandle
	}
	table, err := NewSMTCongruenceTable(manager)
	if err != nil {
		return nil, err
	}
	state := &SMTInternalizerState{
		manager: manager, context: context, congruence: table, quantifiers: quantifiers,
		boolVar: make(map[ast.ASTHandle]int32), boolExpr: make([]ast.ASTHandle, 1), booleanData: make([]SMTBoolVarData, 1),
		activity: make([]float64, 1), assignment: []lbool.LBool{lbool.True, lbool.False}, watches: make([]SMTWatchList, 2),
		cachedGeneration: make(map[ast.ASTHandle]uint32),
	}
	state.booleanData[0].Initialize(0)
	return state, nil
}

func (s *SMTInternalizerState) open() error {
	if s == nil || s.manager == nil || s.context == nil || s.congruence == nil || s.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (s *SMTInternalizerState) SetGeneration(generation uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	s.generation = generation
	if generation > s.statistics.MaxGeneration {
		s.statistics.MaxGeneration = generation
	}
	return nil
}

func (s *SMTInternalizerState) SetCachedGeneration(expression ast.ASTHandle, generation uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	if _, err := s.manager.GetKind(expression); err != nil {
		return err
	}
	s.cachedGeneration[expression] = generation
	return nil
}

func (s *SMTInternalizerState) BoolVar(expression ast.ASTHandle) (int32, bool) {
	if s == nil {
		return SMTNullBoolVar, false
	}
	variable, found := s.boolVar[expression]
	return variable, found
}

func (s *SMTInternalizerState) BoolExpression(variable int32) (ast.ASTHandle, error) {
	if err := s.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if variable < SMTFirstBoolVar || int(variable) >= len(s.boolExpr) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return s.boolExpr[variable], nil
}

func (s *SMTInternalizerState) BooleanData(variable int32) (*SMTBoolVarData, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if variable < SMTFirstBoolVar || int(variable) >= len(s.booleanData) {
		return nil, ast.ErrASTDomain
	}
	return &s.booleanData[variable], nil
}

func (s *SMTInternalizerState) MkBoolVar(expression ast.ASTHandle) (int32, error) {
	if err := s.open(); err != nil {
		return SMTNullBoolVar, err
	}
	if _, found := s.boolVar[expression]; found {
		return SMTNullBoolVar, ast.ErrASTDomain
	}
	sort, err := s.manager.GetSort(expression)
	if err != nil {
		return SMTNullBoolVar, err
	}
	boolean, err := s.manager.MkBoolSort()
	if err != nil {
		return SMTNullBoolVar, err
	}
	if sort != boolean || s.manager.IsTrue(expression) || s.manager.IsFalse(expression) {
		return SMTNullBoolVar, ast.ErrASTDomain
	}
	if err = s.manager.IncRef(expression); err != nil {
		return SMTNullBoolVar, err
	}
	variable := int32(len(s.boolExpr))
	s.boolVar[expression] = variable
	s.boolExpr = append(s.boolExpr, expression)
	data := SMTBoolVarData{}
	data.Initialize(s.scopeLevel)
	s.booleanData = append(s.booleanData, data)
	s.activity = append(s.activity, 0)
	s.assignment = append(s.assignment, lbool.Undef, lbool.Undef)
	s.watches = append(s.watches, SMTWatchList{}, SMTWatchList{})
	s.booleanStack = append(s.booleanStack, expression)
	s.statistics.NumMkBoolVar++
	return variable, nil
}

func (s *SMTInternalizerState) Literal(expression ast.ASTHandle) (SMTLiteral, error) {
	if err := s.open(); err != nil {
		return SMTNullLiteral, err
	}
	if s.manager.IsTrue(expression) {
		return SMTTrueLiteral, nil
	}
	if s.manager.IsFalse(expression) {
		return SMTFalseLiteral, nil
	}
	kind, err := s.manager.GetKind(expression)
	if err != nil {
		return SMTNullLiteral, err
	}
	if kind == ast.ASTApp {
		family, err := s.manager.GetFamilyID(expression)
		if err != nil {
			return SMTNullLiteral, err
		}
		op, err := s.manager.GetDeclKind(expression)
		if err != nil {
			return SMTNullLiteral, err
		}
		if family == s.manager.BasicFamilyID() && op == ast.ASTOpNot {
			child, err := s.manager.GetArg(expression, 0)
			if err != nil {
				return SMTNullLiteral, err
			}
			variable, found := s.boolVar[child]
			if !found {
				return SMTNullLiteral, ast.ErrASTDomain
			}
			return NewSMTLiteral(variable, true), nil
		}
	}
	variable, found := s.boolVar[expression]
	if !found {
		return SMTNullLiteral, ast.ErrASTDomain
	}
	return NewSMTLiteral(variable, false), nil
}

func (s *SMTInternalizerState) SetAssignment(variable int32, value lbool.LBool) error {
	if err := s.open(); err != nil {
		return err
	}
	if variable < SMTFirstBoolVar || int(variable) >= len(s.boolExpr) || value < lbool.False || value > lbool.True {
		return ast.ErrASTDomain
	}
	positive := NewSMTLiteral(variable, false)
	s.assignment[positive.Index()] = value
	s.assignment[positive.Negated().Index()] = value.Negated()
	return s.context.SetAssignment(s.boolExpr[variable], value)
}

func (s *SMTInternalizerState) Assignment(literal SMTLiteral) (lbool.LBool, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, err
	}
	if literal.Index() < 0 || int(literal.Index()) >= len(s.assignment) {
		return lbool.Undef, ast.ErrASTDomain
	}
	return s.assignment[literal.Index()], nil
}

func (s *SMTInternalizerState) WatchList(literal SMTLiteral) (*SMTWatchList, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if literal.Index() < 0 || int(literal.Index()) >= len(s.watches) {
		return nil, ast.ErrASTDomain
	}
	return &s.watches[literal.Index()], nil
}

func (s *SMTInternalizerState) MkEnode(expression ast.ASTHandle, suppressArguments, mergeTruthValue, congruenceEnabled bool) (*SMTEnode, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if s.context.enodes[expression] != nil {
		return nil, ast.ErrASTDomain
	}
	generation := s.generation
	if cached, found := s.cachedGeneration[expression]; found {
		generation = cached
	}
	if err := s.manager.IncRef(expression); err != nil {
		return nil, err
	}
	node, err := NewSMTEnode(s.manager, expression, s.context.enodes, SMTEnodeOptions{
		Generation: generation, InternalScopeLevel: s.scopeLevel, SuppressArguments: suppressArguments,
		MergeTruthValue: mergeTruthValue, CongruenceEnabled: congruenceEnabled, UpdateChildrenParent: true,
	})
	if err != nil {
		_ = s.manager.DecRef(expression)
		return nil, err
	}
	count := node.NumArgs()
	if count == 0 {
		unique, err := s.manager.IsUniqueValue(expression)
		if err != nil {
			_ = node.Delete(true)
			_ = s.manager.DecRef(expression)
			return nil, err
		}
		if unique {
			if err = node.MarkAsInterpreted(); err != nil {
				_ = node.Delete(true)
				_ = s.manager.DecRef(expression)
				return nil, err
			}
		}
	} else if node.IsTrueEquality() {
		node.congruence = node
	} else if congruenceEnabled {
		inserted, err := s.congruence.Insert(node)
		if err != nil {
			_ = node.Delete(true)
			_ = s.manager.DecRef(expression)
			return nil, err
		}
		node.congruence = inserted.Node
		if inserted.Node != node {
			s.pending = append(s.pending, SMTInternalizerCongruence{Node: node, Existing: inserted.Node, Commuted: inserted.Commuted})
		}
	} else {
		node.congruence = node
	}
	if err = s.context.RegisterEnode(node, false); err != nil {
		if node.congruence == node && count != 0 && congruenceEnabled && !node.IsTrueEquality() {
			_, _ = s.congruence.Erase(node)
		}
		_ = node.Delete(true)
		_ = s.manager.DecRef(expression)
		return nil, err
	}
	s.enodeStack = append(s.enodeStack, node)
	s.statistics.NumMkEnode++
	return node, nil
}

func (s *SMTInternalizerState) PendingCongruences() []SMTInternalizerCongruence {
	if s == nil {
		return nil
	}
	return append([]SMTInternalizerCongruence(nil), s.pending...)
}

func (s *SMTInternalizerState) PushScope() error {
	if err := s.open(); err != nil {
		return err
	}
	s.scopes = append(s.scopes, smtInternalizerScope{booleanLimit: len(s.booleanStack), enodeLimit: len(s.enodeStack), pendingLimit: len(s.pending)})
	s.scopeLevel++
	return nil
}

func (s *SMTInternalizerState) unregisterEnode(node *SMTEnode) error {
	if s.context.enodes[node.owner] != node {
		return ast.ErrASTDomain
	}
	bucket := s.context.byDeclaration[node.declaration]
	if len(bucket) == 0 || bucket[len(bucket)-1] != node {
		return ast.ErrASTDomain
	}
	if node.IsCongruenceRoot() && node.NumArgs() != 0 && node.congruenceEnabled && !node.IsTrueEquality() {
		erased, err := s.congruence.Erase(node)
		if err != nil || !erased {
			if err != nil {
				return err
			}
			return ast.ErrASTDomain
		}
	}
	delete(s.context.enodes, node.owner)
	s.context.byDeclaration[node.declaration] = bucket[:len(bucket)-1]
	delete(s.context.relevant, node.owner)
	if err := node.Delete(true); err != nil {
		return err
	}
	if err := s.manager.DecRef(node.owner); err != nil {
		return err
	}
	s.statistics.NumDelEnode++
	return nil
}

func (s *SMTInternalizerState) popTo(scope smtInternalizerScope) error {
	for len(s.enodeStack) > scope.enodeLimit {
		node := s.enodeStack[len(s.enodeStack)-1]
		if err := s.unregisterEnode(node); err != nil {
			return err
		}
		s.enodeStack = s.enodeStack[:len(s.enodeStack)-1]
	}
	for len(s.booleanStack) > scope.booleanLimit {
		expression := s.booleanStack[len(s.booleanStack)-1]
		variable := s.boolVar[expression]
		if s.quantifiers != nil {
			kind, err := s.manager.GetKind(expression)
			if err != nil {
				return err
			}
			if kind == ast.ASTQuantifier {
				if _, statErr := s.quantifiers.Stat(expression); statErr == nil {
					if err = s.quantifiers.Delete(expression); err != nil {
						return err
					}
				} else if !errors.Is(statErr, ast.ErrASTDomain) {
					return statErr
				}
			}
		}
		delete(s.boolVar, expression)
		delete(s.context.assignments, expression)
		s.boolExpr[variable] = ast.ASTHandle{}
		s.boolExpr = s.boolExpr[:variable]
		s.booleanData = s.booleanData[:variable]
		s.activity = s.activity[:variable]
		s.assignment = s.assignment[:2*variable]
		s.watches = s.watches[:2*variable]
		s.booleanStack = s.booleanStack[:len(s.booleanStack)-1]
		if err := s.manager.DecRef(expression); err != nil {
			return err
		}
		s.statistics.NumDelBoolVar++
	}
	if scope.pendingLimit < 0 || scope.pendingLimit > len(s.pending) {
		return ast.ErrASTDomain
	}
	s.pending = s.pending[:scope.pendingLimit]
	return nil
}

func (s *SMTInternalizerState) PopScopes(count uint32) error {
	if err := s.open(); err != nil {
		return err
	}
	if count == 0 || count > uint32(len(s.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(s.scopes) - int(count)
	if err := s.popTo(s.scopes[level]); err != nil {
		return err
	}
	s.scopes = s.scopes[:level]
	s.scopeLevel = uint32(level)
	return nil
}

func (s *SMTInternalizerState) Statistics() SMTContextStatistics {
	if s == nil {
		return SMTContextStatistics{}
	}
	return s.statistics
}

func (s *SMTInternalizerState) Close() error {
	if s == nil || s.manager == nil {
		return ast.ErrASTHandle
	}
	if s.closed {
		return nil
	}
	var result error
	result = errors.Join(result, s.popTo(smtInternalizerScope{}))
	result = errors.Join(result, s.congruence.Close())
	if result == nil {
		s.closed = true
		s.context = nil
		s.manager = nil
	}
	return result
}
