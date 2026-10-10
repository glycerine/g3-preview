// Source: src/model/model.cpp:93-109; src/model/model_core.h:65;
// src/ast/ast.cpp:2561-2582.
package model

import "github.com/glycerine/g3/pkg/ast"

// modelValueProc is the source lexical callback. The AST manager supplies its
// stored sort hash, so lookup does not reenter the manager's locked public API.
type modelValueProc struct{ model *Model }

func (p modelValueProc) SomeValue(s ast.ASTHandle, hash uint32) (ast.ASTHandle, error) {
	md := p.model
	i := md.Universes.find(s, hash)
	if i >= 0 && md.Universes.Cells[i].index != 0 {
		values := md.directory[md.Universes.Cells[i].index-1].values
		if len(values) > 0 {
			return values[0], nil
		}
	}
	return ast.ASTHandle{}, nil
}

// GetSomeValue borrows the first universe value when present, then delegates
// to the original AST plugin/fallback path under a scoped universe callback.
func (md *Model) GetSomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	return md.CoreField.manager.GetSomeValueWithProc(s, modelValueProc{md})
}
func (md *ModelCore) GetSomeValue(s ast.ASTHandle) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	// The source base method is abstract; a bare core has no universe provider.
	if md.Derived == nil {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return md.Derived.GetSomeValue(s)
}
