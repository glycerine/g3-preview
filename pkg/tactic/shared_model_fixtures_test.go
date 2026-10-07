package tactic

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
)

func modelTestDecl(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	p := goalTestAtom(t, m, name)
	d, e := m.GetDecl(p)
	astTestMust(t, e)
	return d
}
func modelTestCore(t *testing.T, m *ast.ASTManager) *model.ModelCore {
	t.Helper()
	md, e := model.NewModelCore(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, md.Close()) })
	return md
}
