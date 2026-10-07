// Source: src/model/model.h; src/model/model_core.h.
package model

import (
	"github.com/glycerine/g3/pkg/ast"
)

func (m *Model) Core() *ModelCore             { return &m.CoreField }
func (m *ModelCore) Manager() *ast.ASTManager { return m.manager }
func (m *ModelCore) Check() error             { return m.open() }

func (m *ModelCore) DerivedModel() *Model             { return m.Derived }
func (m *ModelCore) CheckFunctionStorageEmpty() error { return m.Functions.empty() }
func (m *ModelCore) NumFunctionDeclarations() uint32 {
	return uint32(len(m.functionDeclarations.Slice()))
}
