package cmd_context

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestCommandContextPseudoBooleanBuiltinVisibilityAndDeclarationReferences(t *testing.T) {
	for _, logic := range []string{"", "ALL", "QF_FD", "QF_BV", "QF_UF", "QF_LIA"} {
		t.Run(logic, func(t *testing.T) {
			c := contextFixture(t)
			if logic != "" {
				contextMust(t, c.SetLogic(logic))
			}
			m, err := c.Manager()
			contextMust(t, err)
			bs, err := m.MkBoolSort()
			contextMust(t, err)
			visible := logic == "" || logic == "ALL" || logic == "QF_FD"
			for _, name := range []string{"at-most", "at-least", "pble", "pbge", "pbeq"} {
				indices := []uint32{1}
				if name != "at-most" && name != "at-least" {
					indices = []uint32{2, 1, 3}
				}
				d, err := c.FindFunctionDeclaration(name, indices, []ast.ASTHandle{bs, bs}, bs)
				if !visible {
					if err == nil {
						t.Fatal("pseudo-Boolean name escaped its source logic gate", logic, name)
					}
					continue
				}
				contextMust(t, err)
				params := make([]ast.ASTParameter, len(indices))
				for i, index := range indices {
					params[i], err = m.NewParameterUint(index)
					contextMust(t, err)
				}
				yes, err := m.MkTrue()
				contextMust(t, err)
				no, err := m.MkFalse()
				contextMust(t, err)
				app, err := c.MakeApp(name, []ast.ASTHandle{yes, no}, params, ast.ASTHandle{})
				for i := range params {
					contextMust(t, m.DelParameter(&params[i]))
				}
				contextMust(t, err)
				actual, err := m.GetDecl(app)
				contextMust(t, err)
				if actual != d {
					t.Fatal("indexed application and declaration reference disagree", name)
				}
			}
		})
	}
}
