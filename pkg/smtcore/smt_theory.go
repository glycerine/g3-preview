// Source: src/smt/smt_theory.h and src/smt/smt_theory.cpp internalization and
// theory-variable lifecycle used by the logical context.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

// SMTTheoryInternalizer is the retained concrete theory surface consumed by
// expression internalization. Search, model, and conflict callbacks are added
// with the context stages that consume them.
type SMTTheoryInternalizer interface {
	Initialize(*SMTInternalizerState) error
	FamilyID() ast.FamilyID
	DefaultInternalizer() bool
	InternalizeAtom(ast.ASTHandle, bool) (bool, error)
	InternalizeEquality(ast.ASTHandle, int32) error
	InternalizeTerm(ast.ASTHandle) (bool, error)
	ApplySortConstraint(*SMTEnode, ast.ASTHandle) error
	IsAttachedToVariable(*SMTEnode) bool
	Enode(SMTTheoryVar) (*SMTEnode, error)
	UseDisequalities() bool
	NewEquality(SMTTheoryVar, SMTTheoryVar) error
	NewDisequality(SMTTheoryVar, SMTTheoryVar) error
	Assign(int32, bool) error
	Relevant(ast.ASTHandle) error
	CanPropagate() bool
	Propagate() error
	Setup() error
	InitializeSearch() error
	Restart() error
	FinalCheck() (SMTFinalCheckStatus, error)
	PushScope() error
	PopScopes(uint32) error
}

type SMTTheoryBase struct {
	family    ast.FamilyID
	state     *SMTInternalizerState
	variables []*SMTEnode
	limits    []int
}

func NewSMTTheoryBase(family ast.FamilyID) (*SMTTheoryBase, error) {
	if family == ast.NullFamilyID {
		return nil, ast.ErrASTDomain
	}
	return &SMTTheoryBase{family: family}, nil
}

func (t *SMTTheoryBase) Initialize(state *SMTInternalizerState) error {
	if t == nil || state == nil || t.state != nil || t.family == ast.NullFamilyID {
		return ast.ErrASTDomain
	}
	if err := state.open(); err != nil {
		return err
	}
	t.state = state
	return nil
}

func (t *SMTTheoryBase) FamilyID() ast.FamilyID {
	if t == nil {
		return ast.NullFamilyID
	}
	return t.family
}

func (*SMTTheoryBase) DefaultInternalizer() bool    { return true }
func (*SMTTheoryBase) UseDisequalities() bool       { return true }
func (*SMTTheoryBase) CanPropagate() bool           { return false }
func (*SMTTheoryBase) Propagate() error             { return nil }
func (*SMTTheoryBase) Relevant(ast.ASTHandle) error { return nil }
func (*SMTTheoryBase) Setup() error                 { return nil }
func (*SMTTheoryBase) InitializeSearch() error      { return nil }
func (*SMTTheoryBase) Restart() error               { return nil }
func (*SMTTheoryBase) FinalCheck() (SMTFinalCheckStatus, error) {
	return SMTFinalCheckDone, nil
}

func (t *SMTTheoryBase) State() (*SMTInternalizerState, error) {
	if t == nil || t.state == nil {
		return nil, ast.ErrASTHandle
	}
	if err := t.state.open(); err != nil {
		return nil, err
	}
	return t.state, nil
}

func (t *SMTTheoryBase) CreateVariable(node *SMTEnode) (SMTTheoryVar, error) {
	state, err := t.State()
	if err != nil {
		return SMTNullTheoryVar, err
	}
	if node == nil || node.manager != state.manager || t.IsAttachedToVariable(node) {
		return SMTNullTheoryVar, ast.ErrASTDomain
	}
	variable := SMTTheoryVar(len(t.variables))
	if err = state.attachTheoryVariable(node, t.family, variable); err != nil {
		return SMTNullTheoryVar, err
	}
	t.variables = append(t.variables, node)
	return variable, nil
}

func (t *SMTTheoryBase) IsAttachedToVariable(node *SMTEnode) bool {
	if t == nil || node == nil {
		return false
	}
	variable := node.TheoryVar(t.family)
	return variable >= 0 && int(variable) < len(t.variables) && t.variables[variable] == node
}

func (t *SMTTheoryBase) Enode(variable SMTTheoryVar) (*SMTEnode, error) {
	if t == nil || variable < 0 || int(variable) >= len(t.variables) {
		return nil, ast.ErrASTDomain
	}
	return t.variables[variable], nil
}

func (t *SMTTheoryBase) NumVariables() int {
	if t == nil {
		return 0
	}
	return len(t.variables)
}

func (t *SMTTheoryBase) OldNumVariables(scopes uint32) (int, error) {
	if t == nil || scopes == 0 || scopes > uint32(len(t.limits)) {
		return 0, ast.ErrASTDomain
	}
	return t.limits[len(t.limits)-int(scopes)], nil
}

// MkEqualityLiteral is theory::mk_eq: construct and internalize an equality in
// the requested gate context, then return its Boolean literal.
func (t *SMTTheoryBase) MkEqualityLiteral(left, right ast.ASTHandle, gateContext bool) (SMTLiteral, error) {
	state, err := t.State()
	if err != nil {
		return SMTNullLiteral, err
	}
	equality, err := state.manager.MkEq(left, right)
	if err != nil {
		return SMTNullLiteral, err
	}
	if err = state.Internalize(equality, gateContext); err != nil {
		return SMTNullLiteral, err
	}
	return state.Literal(equality)
}

func (t *SMTTheoryBase) PushScope() error {
	if _, err := t.State(); err != nil {
		return err
	}
	t.limits = append(t.limits, len(t.variables))
	return nil
}

func (t *SMTTheoryBase) PopScopes(scopes uint32) error {
	if _, err := t.State(); err != nil {
		return err
	}
	if scopes == 0 || scopes > uint32(len(t.limits)) {
		return ast.ErrASTDomain
	}
	level := len(t.limits) - int(scopes)
	limit := t.limits[level]
	for index := limit; index < len(t.variables); index++ {
		t.variables[index] = nil
	}
	t.variables = t.variables[:limit]
	t.limits = t.limits[:level]
	return nil
}
