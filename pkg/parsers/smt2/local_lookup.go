// Source: src/parsers/smt2/smt2parser.cpp:1699-1721.
package smt2

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

var errLocalBindingLevel = errors.New("smt2 parser: local binding level exceeds current bindings")
var errLocalShifter = errors.New("smt2 parser: variable shifter is unavailable")

// resolveLocal is the local-identifier branch of parse_expr_name. A found
// result is an owning expression-stack reference; the parser releases it when
// its frame pops. A missing name is resolved by the command context.
func (e *smt2LocalEnv) resolveLocal(m *ast.ASTManager, shifter *rewriter.VarShifter, name usymbol.Symbol, numBindings uint32) (ast.ASTRef, bool, error) {
	l, found := e.find(name)
	if !found {
		return ast.ASTRef{}, false, nil
	}
	ground, err := m.IsGround(l.term)
	if err != nil {
		return ast.ASTRef{}, true, err
	}
	if ground || l.level == numBindings {
		ref, err := m.NewRef(l.term)
		return ref, true, err
	}
	if l.level > numBindings {
		return ast.ASTRef{}, true, errLocalBindingLevel
	}
	if shifter == nil {
		return ast.ASTRef{}, true, errLocalShifter
	}
	ref, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		return ast.ASTRef{}, true, err
	}
	if err = shifter.ShiftBy(l.term, numBindings-l.level, ref); err != nil {
		return ast.ASTRef{}, true, errors.Join(err, m.DelRef(&ref))
	}
	return ref, true, nil
}
