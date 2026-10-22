package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtCGTestDeclaration(t *testing.T, manager *ast.ASTManager, name uint32, sort ast.ASTHandle, flags ast.ASTFuncFlags) ast.ASTHandle {
	t.Helper()
	info, err := manager.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, flags, false)
	astTestMust(t, err)
	declaration, err := manager.MkFuncDeclWithInfo(symbol.Numerical(name), []ast.ASTHandle{sort, sort}, sort, info)
	astTestMust(t, err)
	astTestMust(t, manager.DelFuncDeclInfo(&info))
	return declaration
}

func smtCGTestApplication(t *testing.T, manager *ast.ASTManager, declaration ast.ASTHandle, handles []ast.ASTHandle, nodes []*SMTEnode) *SMTEnode {
	t.Helper()
	owner, err := manager.MkApp(declaration, handles)
	astTestMust(t, err)
	byApp := make(map[ast.ASTHandle]*SMTEnode, len(handles))
	for i := range handles {
		byApp[handles[i]] = nodes[i]
	}
	node, err := NewSMTEnode(manager, owner, byApp, SMTEnodeOptions{CongruenceEnabled: true})
	astTestMust(t, err)
	return node
}

func TestSMTCongruenceTableExactCommutativeInsertFindEraseReset(t *testing.T) {
	manager := astTestManager(t)
	sort, err := manager.MkSort(symbol.Numerical(2600), ast.ASTSortInfo{})
	astTestMust(t, err)
	a, an := smtEnodeConstant(t, manager, 2601, sort, 0)
	b, bn := smtEnodeConstant(t, manager, 2602, sort, 0)
	declaration := smtCGTestDeclaration(t, manager, 2603, sort, ast.ASTFuncFlags{Commutative: true})
	ab := smtCGTestApplication(t, manager, declaration, []ast.ASTHandle{a, b}, []*SMTEnode{an, bn})
	ba := smtCGTestApplication(t, manager, declaration, []ast.ASTHandle{b, a}, []*SMTEnode{bn, an})
	table, err := NewSMTCongruenceTable(manager)
	astTestMust(t, err)
	inserted, err := table.Insert(ab)
	astTestMust(t, err)
	if inserted.Node != ab || inserted.Commuted {
		t.Fatal("first insertion", inserted)
	}
	inserted, err = table.Insert(ba)
	astTestMust(t, err)
	if inserted.Node != ab || !inserted.Commuted {
		t.Fatal("commutative match", inserted)
	}
	if found, commuted, err := table.Find(ba); err != nil || found != ab || !commuted {
		t.Fatal("find", found, commuted, err)
	}
	if contains, err := table.ContainsPointer(ba); err != nil || contains {
		t.Fatal("nonrepresentative pointer", contains, err)
	}
	if err := table.CheckInvariant(); err != nil {
		t.Fatal(err)
	}
	if erased, err := table.Erase(ba); err != nil || !erased {
		t.Fatal("erase congruent key", erased, err)
	}
	if found, _, err := table.Find(ab); err != nil || found != nil {
		t.Fatal("find after erase", found, err)
	}
	astTestMust(t, table.Reset())
	astTestMust(t, table.Close())
}

func TestSMTAlmostCongruenceTableGroupsSpecialRoots(t *testing.T) {
	manager := astTestManager(t)
	sort, err := manager.MkSort(symbol.Numerical(2610), ast.ASTSortInfo{})
	astTestMust(t, err)
	a, an := smtEnodeConstant(t, manager, 2611, sort, 0)
	b, bn := smtEnodeConstant(t, manager, 2612, sort, 0)
	c, cn := smtEnodeConstant(t, manager, 2613, sort, 0)
	declaration := smtCGTestDeclaration(t, manager, 2614, sort, ast.ASTFuncFlags{})
	ac := smtCGTestApplication(t, manager, declaration, []ast.ASTHandle{a, c}, []*SMTEnode{an, cn})
	bc := smtCGTestApplication(t, manager, declaration, []ast.ASTHandle{b, c}, []*SMTEnode{bn, cn})
	ca := smtCGTestApplication(t, manager, declaration, []ast.ASTHandle{c, a}, []*SMTEnode{cn, an})
	table := NewSMTAlmostCongruenceTable(an, bn)
	for _, node := range []*SMTEnode{ac, bc, ca} {
		astTestMust(t, table.Insert(node))
	}
	group, err := table.Find(ac)
	astTestMust(t, err)
	if len(group) != 2 || group[0] != bc || group[1] != ac {
		t.Fatal("source prepend group order", group)
	}
	other, err := table.Find(ca)
	astTestMust(t, err)
	if len(other) != 1 || other[0] != ca {
		t.Fatal("independent group", other)
	}
	if erased, err := table.Erase(bc); err != nil || !erased {
		t.Fatal("erase equivalence class", erased, err)
	}
	if group, err = table.Find(ac); err != nil || group != nil {
		t.Fatal("erased group remained", group, err)
	}
	astTestMust(t, table.ResetFor(an, cn))
	if !table.Empty() {
		t.Fatal("reset for roots")
	}
}
