// Source: src/smt/theory_str.cpp retained theory ownership, internalization,
// mk_var, string cache and scoped union-find lifecycle.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/backtrack"
)

// SMTStringTheory is the distinct theory_str owner. Its remaining constraint
// callbacks/final check must be ported before setup selects it for QF_S.
type SMTStringTheory struct {
	*SMTTheoryBase
	parameterOwner     *SMTStrParams
	find               *backtrack.UnionFind
	findTrail          *backtrack.TrailStack
	level              uint32
	charset            [256]byte
	charsetLookup      map[byte]int
	strings            map[string]ast.ASTHandle
	refs               []ast.ASTRef
	cacheAccesses      uint64
	unusedID           uint32
	basicTodo          []*SMTEnode
	concatTodo         []*SMTEnode
	concatEvalTodo     []*SMTEnode
	libraryTodo        []*SMTEnode
	delayedSetup       []ast.ASTHandle
	conversions        []ast.ASTHandle
	searchStarted      bool
	freshID            uint32
	concatCache        map[[2]ast.ASTHandle]ast.ASTHandle
	variableSet        map[ast.ASTHandle]struct{}
	internalVariables  map[ast.ASTHandle]struct{}
	variableScopes     map[uint32]map[ast.ASTHandle]struct{}
	axiomatized        map[ast.ASTHandle]struct{}
	contains           []ast.ASTHandle
	containsRefs       []ast.ASTRef
	containsByPair     map[[2]ast.ASTHandle]ast.ASTHandle
	containsIndex      map[ast.ASTHandle]map[[2]ast.ASTHandle]struct{}
	regexVariables     map[ast.ASTHandle]struct{}
	regexMemberships   map[ast.ASTHandle]map[string]ast.ASTHandle
	regexTerms         map[ast.ASTHandle]struct{}
	regexTermsByString map[ast.ASTHandle][]ast.ASTHandle
	regexNFACache      map[ast.ASTHandle]*smtStringNFA
	concatBreakFlags   map[[2]ast.ASTHandle]ast.ASTHandle
	closed             bool
}

func NewSMTStringTheory(family ast.FamilyID, parameters *SMTStrParams) (*SMTStringTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	if parameters != nil {
		if _, err = parameters.State(); err != nil {
			return nil, err
		}
	}
	context := backtrack.NewUnionFindDefaultContext()
	find, err := backtrack.NewUnionFind(context)
	if err != nil {
		return nil, err
	}
	t := &SMTStringTheory{SMTTheoryBase: base, parameterOwner: parameters, find: find, findTrail: context.GetTrailStack(), strings: make(map[string]ast.ASTHandle), charsetLookup: make(map[byte]int)}
	t.concatCache = make(map[[2]ast.ASTHandle]ast.ASTHandle)
	t.variableSet = make(map[ast.ASTHandle]struct{})
	t.internalVariables = make(map[ast.ASTHandle]struct{})
	t.variableScopes = make(map[uint32]map[ast.ASTHandle]struct{})
	t.axiomatized = make(map[ast.ASTHandle]struct{})
	t.containsByPair = make(map[[2]ast.ASTHandle]ast.ASTHandle)
	t.containsIndex = make(map[ast.ASTHandle]map[[2]ast.ASTHandle]struct{})
	t.regexVariables = make(map[ast.ASTHandle]struct{})
	t.regexMemberships = make(map[ast.ASTHandle]map[string]ast.ASTHandle)
	t.regexTerms = make(map[ast.ASTHandle]struct{})
	t.regexTermsByString = make(map[ast.ASTHandle][]ast.ASTHandle)
	t.regexNFACache = make(map[ast.ASTHandle]*smtStringNFA)
	t.concatBreakFlags = make(map[[2]ast.ASTHandle]ast.ASTHandle)
	// Source initialize_charset's active default alphabet. The unused final
	// slot remains zero; NUL is excluded from the 255-character search order.
	index := 0
	for _, bounds := range [][2]int{{97, 123}, {65, 91}, {48, 58}, {32, 48}, {58, 65}, {91, 97}, {123, 127}, {1, 32}, {127, 256}} {
		for value := bounds[0]; value < bounds[1]; value++ {
			t.charset[index] = byte(value)
			t.charsetLookup[byte(value)] = index
			index++
		}
	}
	return t, nil
}

func (t *SMTStringTheory) retain(expression ast.ASTHandle) error {
	ref, err := t.state.manager.NewRef(expression)
	if err != nil {
		return err
	}
	t.refs = append(t.refs, ref)
	return nil
}

func (t *SMTStringTheory) stringLiteral(text string) (ast.ASTHandle, error) {
	state, err := t.State()
	if err != nil || t.closed {
		if err == nil {
			err = ast.ErrASTHandle
		}
		return ast.ASTHandle{}, err
	}
	cache := true
	if t.parameterOwner != nil {
		parameters, err := t.parameterOwner.State()
		if err != nil {
			return ast.ASTHandle{}, err
		}
		cache = parameters.StringConstantCache
	}
	if cache {
		t.cacheAccesses++
		if value, found := t.strings[text]; found {
			return value, nil
		}
	}
	value, err := state.manager.MkString(text)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if cache {
		if err = t.retain(value); err != nil {
			return ast.ASTHandle{}, err
		}
		t.strings[text] = value
	}
	return value, nil
}

func (t *SMTStringTheory) mkVar(node *SMTEnode) (SMTTheoryVar, error) {
	state, err := t.State()
	if err != nil {
		return SMTNullTheoryVar, err
	}
	if node == nil || node.manager != state.manager {
		return SMTNullTheoryVar, ast.ErrASTDomain
	}
	sort, err := state.manager.GetSort(node.Owner())
	if err != nil {
		return SMTNullTheoryVar, err
	}
	isString, err := state.manager.IsStringSort(sort)
	if err != nil || !isString {
		return SMTNullTheoryVar, err
	}
	if t.IsAttachedToVariable(node) {
		return node.TheoryVar(t.FamilyID()), nil
	}
	variable, err := t.CreateVariable(node)
	if err != nil {
		return SMTNullTheoryVar, err
	}
	if t.find.MkVar() != uint32(variable) {
		return SMTNullTheoryVar, ast.ErrASTDomain
	}
	return variable, state.markRelevant(node.Owner())
}

func (t *SMTStringTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	return t.InternalizeTerm(expression)
}

func (t *SMTStringTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return false, err
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < count; i++ {
		argument, err := state.manager.GetArg(expression, i)
		if err != nil {
			return false, err
		}
		if err = state.Internalize(argument, false); err != nil {
			return false, err
		}
	}
	if node := state.context.enodes[expression]; node != nil {
		_, err := t.mkVar(node)
		return err == nil, err
	}
	sort, err := state.manager.GetSort(expression)
	if err != nil {
		return false, err
	}
	boolean, err := state.manager.MkBoolSort()
	if err != nil {
		return false, err
	}
	isBoolean := sort == boolean
	if isBoolean {
		variable, found := state.BoolVar(expression)
		if !found {
			variable, err = state.MkBoolVar(expression)
			if err != nil {
				return false, err
			}
		}
		if err = state.SetVariableTheory(variable, t.FamilyID()); err != nil {
			return false, err
		}
		if err = state.ensureBooleanEnode(expression, variable, false, true, true); err != nil {
			return false, err
		}
	}
	node := state.context.enodes[expression]
	if node == nil {
		node, err = state.MkEnode(expression, false, isBoolean, true)
		if err != nil {
			return false, err
		}
	}
	for _, argument := range node.arguments {
		if _, err = t.mkVar(argument); err != nil {
			return false, err
		}
	}
	if _, err = t.mkVar(node); err != nil {
		return false, err
	}
	if _, constant, err := state.manager.SequenceStringValue(expression); err != nil {
		return false, err
	} else if constant {
		t.basicTodo = append(t.basicTodo, node)
	}
	return true, nil
}

func (t *SMTStringTheory) ApplySortConstraint(node *SMTEnode, _ ast.ASTHandle) error {
	_, err := t.mkVar(node)
	return err
}

func (*SMTStringTheory) InternalizeEquality(ast.ASTHandle, int32) error  { return nil }
func (*SMTStringTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error { return nil }
func (*SMTStringTheory) Assign(int32, bool) error                        { return nil }
func (*SMTStringTheory) Name() string                                    { return "seq" }

func (t *SMTStringTheory) NewEquality(left, right SMTTheoryVar) error {
	if left < 0 || right < 0 || uint32(left) >= t.find.GetNumVars() || uint32(right) >= t.find.GetNumVars() {
		return ast.ErrASTDomain
	}
	// Native handle_equality runs before merging. Initial concat shortcuts,
	// consistency, coherence, grouping, constant and parent stages are
	// connected; concat/concat arrangements and unroll/testers remain open.
	lhs, rhs := t.variables[left].Owner(), t.variables[right].Owner()
	shortcut, err := t.stringConcatEqualityShortcut(lhs, rhs)
	if err != nil {
		return err
	}
	if !shortcut {
		consistent, err := t.checkNewEqualityValuesAndLengths(lhs, rhs)
		if err != nil {
			return err
		}
		if consistent {
			if err := t.handleConstantEqualityStages(lhs, rhs); err != nil {
				return err
			}
		}
	}
	t.find.Merge(uint32(left), uint32(right))
	return nil
}

func (t *SMTStringTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.findTrail.PushScope()
	t.level++
	return nil
}

func (t *SMTStringTheory) PopScopes(count uint32) error {
	if count == 0 || count > t.level {
		return ast.ErrASTDomain
	}
	t.level -= count
	kept := t.basicTodo[:0]
	for _, node := range t.basicTodo {
		if node.InternalScopeLevel() <= t.level {
			kept = append(kept, node)
		}
	}
	for i := len(kept); i < len(t.basicTodo); i++ {
		t.basicTodo[i] = nil
	}
	t.basicTodo = kept
	if err := t.findTrail.PopScope(count); err != nil {
		return err
	}
	return t.SMTTheoryBase.PopScopes(count)
}

func (t *SMTStringTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	_, err := t.State()
	// The native final_check_eh/handle_equality/library pipeline is not yet
	// connected. This owner must not turn an incomplete port into SAT.
	return SMTFinalCheckGiveUp, err
}

func (t *SMTStringTheory) Close() (result error) {
	if t == nil || t.closed {
		return nil
	}
	result = t.findTrail.Reset()
	// Native contains_map is a later-declared expr_ref_vector and closes
	// before m_trail, also in forward element order.
	for i := range t.containsRefs {
		result = errors.Join(result, t.state.manager.DelRef(&t.containsRefs[i]))
	}
	// Native m_trail is an expr_ref_vector and releases its range forward.
	for i := range t.refs {
		result = errors.Join(result, t.state.manager.DelRef(&t.refs[i]))
	}
	t.refs, t.basicTodo, t.strings = nil, nil, nil
	t.containsRefs, t.contains = nil, nil
	t.concatTodo, t.concatEvalTodo, t.libraryTodo = nil, nil, nil
	t.delayedSetup, t.conversions = nil, nil
	t.concatCache, t.variableSet, t.internalVariables = nil, nil, nil
	t.variableScopes, t.axiomatized = nil, nil
	t.containsByPair, t.containsIndex = nil, nil
	t.regexVariables, t.regexTerms, t.regexNFACache = nil, nil, nil
	t.regexMemberships, t.regexTermsByString = nil, nil
	t.concatBreakFlags = nil
	t.closed = true
	return result
}

func (t *SMTStringTheory) Fresh() (*SMTStringTheory, error) {
	if t == nil || t.closed {
		return nil, ast.ErrASTHandle
	}
	return NewSMTStringTheory(t.FamilyID(), t.parameterOwner)
}

var _ SMTTheoryInternalizer = (*SMTStringTheory)(nil)
