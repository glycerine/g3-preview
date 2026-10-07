// Source: src/tactic/goal.h.
package tactic

import (
	"github.com/glycerine/g3/pkg/ast"
)

func (g *Goal) Manager() *ast.ASTManager { return g.manager }
func (g *Goal) CheckOwner() error        { return g.open() }
