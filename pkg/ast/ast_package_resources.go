// Source: src/ast/ast.h; src/util/id_gen.h.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
)

func (m *ASTManager) Budget() *arena.Budget { return m.BudgetField }
func (m *ASTManager) IsOutOfMemory() bool   { return m.ExprIDs.IsOutOfMemory() }

func (m *ASTManager) ResourceLimit() *control.Reslimit { return &m.Limit }
func (m *ASTManager) IsTrue(h ASTHandle) bool          { return h == m.external(m.trueApp) }
func (m *ASTManager) IsFalse(h ASTHandle) bool         { return h == m.external(m.falseApp) }
