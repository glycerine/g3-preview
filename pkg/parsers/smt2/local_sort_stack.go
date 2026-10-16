// Source: src/parsers/smt2/smt2parser.cpp:230-234,1723-1729.
package smt2

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

var errLocalSortStack = errors.New("smt2 parser: qualified sort stack is empty")

// localSortStack is the parser's concrete sort_ref_vector owner. Native slice
// slots retain AST references until the parser consumes or clears them.
type localSortStack struct{ refs []ast.ASTRef }

func (s *localSortStack) push(m *ast.ASTManager, sort ast.ASTHandle) error {
	ref, err := m.NewRef(sort)
	if err != nil {
		return err
	}
	s.refs = append(s.refs, ref)
	return nil
}

func (s *localSortStack) top(m *ast.ASTManager) (ast.ASTHandle, error) {
	if s == nil || len(s.refs) == 0 {
		return ast.ASTHandle{}, errLocalSortStack
	}
	return m.RefGet(s.refs[len(s.refs)-1])
}

func (s *localSortStack) pop(m *ast.ASTManager) error {
	if s == nil || len(s.refs) == 0 {
		return errLocalSortStack
	}
	last := len(s.refs) - 1
	err := m.DelRef(&s.refs[last])
	if s.refs[last] == (ast.ASTRef{}) {
		s.refs = s.refs[:last]
	}
	return err
}
