package protomodel

import (
	"github.com/glycerine/g3/pkg/ast"
)

// ValueFactory is the concrete theory-value interface used while constructing
// an SMT model.
type ValueFactory interface {
	FamilyID() ast.FamilyID
	SomeValue(ast.ASTHandle) (ast.ASTHandle, error)
	SomeValues(ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error)
	FreshValue(ast.ASTHandle) (ast.ASTHandle, error)
	RegisterValue(ast.ASTHandle) error
	Close() error
}

type basicFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
}

func newBasicFactory(m *ast.ASTManager) *basicFactory {
	return &basicFactory{manager: m, family: m.BasicFamilyID()}
}
func (f *basicFactory) FamilyID() ast.FamilyID { return f.family }
func (f *basicFactory) SomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	family, err := f.manager.GetFamilyID(s)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if family != f.manager.BasicFamilyID() {
		return ast.ASTHandle{}, nil
	}
	k, err := f.manager.GetDeclKind(s)
	if err != nil || k != 0 {
		return ast.ASTHandle{}, err
	}
	return f.manager.MkFalse()
}
func (f *basicFactory) SomeValues(s ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	v1, err := f.SomeValue(s)
	if err != nil || v1.IsNull() {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	v2, err := f.manager.MkTrue()
	return v1, v2, err == nil, err
}
func (f *basicFactory) FreshValue(ast.ASTHandle) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, nil
}
func (f *basicFactory) RegisterValue(ast.ASTHandle) error { return nil }
func (f *basicFactory) Close() error                      { return nil }

// Structured factories borrow the same concrete handle collection shape;
// their enumeration policy is separate from numeric/unsigned simple_factory.
type factoryValueSet struct {
	values []ast.ASTHandle
	seen   map[ast.ASTHandle]struct{}
}
