// Source: src/smt/theory_pb.cpp pb_factory and init_model. The source PB
// build_models hook is false; Boolean roots use assignment-value wrappers
// before family dispatch in smt_model_generator.cpp, not pb_model_value_proc.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type smtPBValueFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
}

func (f *smtPBValueFactory) FamilyID() ast.FamilyID { return f.family }

// The source ignores the requested sort and supplies canonical Booleans.
func (f *smtPBValueFactory) SomeValue(ast.ASTHandle) (ast.ASTHandle, error) {
	if f == nil || f.manager == nil {
		return ast.ASTHandle{}, ast.ErrASTHandle
	}
	return f.manager.MkTrue()
}

func (f *smtPBValueFactory) SomeValues(ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	if f == nil || f.manager == nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, ast.ErrASTHandle
	}
	a, err := f.manager.MkTrue()
	if err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	}
	b, err := f.manager.MkFalse()
	return a, b, err == nil, err
}

func (*smtPBValueFactory) FreshValue(ast.ASTHandle) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, nil
}
func (*smtPBValueFactory) RegisterValue(ast.ASTHandle) error { return nil }
func (f *smtPBValueFactory) Close() error {
	if f != nil {
		f.manager = nil
	}
	return nil
}

func (t *SMTPseudoBooleanTheory) InitializeModel(generator *SMTModelGenerator) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	if generator == nil || generator.manager != s.manager {
		return ast.ErrASTDomain
	}
	if err = generator.open(); err != nil {
		return err
	}
	return generator.RegisterFactory(&smtPBValueFactory{manager: s.manager, family: t.FamilyID()})
}
