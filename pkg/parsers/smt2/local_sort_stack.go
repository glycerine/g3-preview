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

// shrink mirrors ref_vector_core::shrink: release discarded slots in their
// original order. If a release is pending, keep its slot for a later reset.
func (s *localSortStack) shrink(m *ast.ASTManager, size int) error {
	if s == nil || size < 0 || size > len(s.refs) {
		return errLocalSortStack
	}
	var failure error
	for i := size; i < len(s.refs); i++ {
		if s.refs[i] == (ast.ASTRef{}) {
			continue
		}
		failure = errors.Join(failure, m.DelRef(&s.refs[i]))
	}
	if failure == nil {
		s.refs = s.refs[:size]
	}
	return failure
}

func (s *localSortStack) reset(m *ast.ASTManager) error { return s.shrink(m, 0) }
