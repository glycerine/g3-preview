// Source: src/smt/smt_theory.cpp:60-114 display_app/display_flat_app.
package smtcore

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/ast"
)

func smtWriteTheoryText(out io.Writer, text string) error {
	n, err := io.WriteString(out, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}

// DisplayApplication recursively expands only this theory's applications.
// Foreign applications print their AST identity; constants print their head.
func (t *SMTTheoryBase) DisplayApplication(out io.Writer, expression ast.ASTHandle) error {
	return t.displayApplication(out, expression, false)
}

// DisplayFlatApplication preserves the source stack order: matching nested
// associative applications are deferred, then visited in reverse push order.
// Only the outer declaration is flattened; other children use display_app.
func (t *SMTTheoryBase) DisplayFlatApplication(out io.Writer, expression ast.ASTHandle) error {
	return t.displayApplication(out, expression, true)
}

func (t *SMTTheoryBase) displayApplication(out io.Writer, expression ast.ASTHandle, flat bool) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	if out == nil {
		return ast.ErrASTHandle
	}
	m := s.manager
	decl, err := m.GetDecl(expression)
	if err != nil {
		return err
	}
	count, err := m.GetNumArgs(expression)
	if err != nil {
		return err
	}
	if count == 0 {
		return m.DisplayDeclarationHead(out, decl)
	}
	family, err := m.GetFamilyID(expression)
	if err != nil {
		return err
	}
	if family != t.FamilyID() {
		id, err := m.GetASTID(expression)
		if err != nil {
			return err
		}
		return smtWriteTheoryText(out, "#"+strconv.FormatUint(uint64(id), 10))
	}
	if err = smtWriteTheoryText(out, "("); err != nil {
		return err
	}
	if err = m.DisplayDeclarationHead(out, decl); err != nil {
		return err
	}
	associative := false
	if flat {
		associative, err = m.RewriterAssociative(decl)
		if err != nil {
			return err
		}
	}
	todo := []ast.ASTHandle{expression}
	for len(todo) != 0 {
		current := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		count, err = m.GetNumArgs(current)
		if err != nil {
			return err
		}
		for i := uint32(0); i < count; i++ {
			arg, err := m.GetArg(current, i)
			if err != nil {
				return err
			}
			argDecl, err := m.GetDecl(arg)
			if err != nil {
				return err
			}
			if associative && argDecl == decl {
				todo = append(todo, arg)
				continue
			}
			if err = smtWriteTheoryText(out, " "); err != nil {
				return err
			}
			if err = t.DisplayApplication(out, arg); err != nil {
				return err
			}
		}
	}
	return smtWriteTheoryText(out, ")")
}
