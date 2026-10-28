package pdr_test

// Source: src/muz/pdr/pdr_context.cpp context::init_rules allocation,
// transformer initialization, and level-zero query setup.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/pdr"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestContextInitializesHornProgramAndAnswersInitialQueries(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	pdrMust(t, err)
	manager, err := command.Manager()
	pdrMust(t, err)
	globals, err := command.GlobalParameters()
	pdrMust(t, err)
	context, err := pdr.NewContext(manager, nil, 4, pdr.ReachableConstraintCache, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	pdrMust(t, err)
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	pDeclaration := pdrFunction(t, manager, "context_p", []ast.ASTHandle{boolean}, boolean)
	qDeclaration := pdrFunction(t, manager, "context_q", []ast.ASTHandle{boolean}, boolean)
	ruleManager, err := base.NewRuleManager(manager)
	pdrMust(t, err)
	pdrMust(t, ruleManager.RegisterPredicate(pDeclaration))
	pdrMust(t, ruleManager.RegisterPredicate(qDeclaration))
	set, err := base.NewRuleSet(ruleManager)
	pdrMust(t, err)
	variable, err := manager.MkVar(0, boolean)
	pdrMust(t, err)
	truth, err := manager.MkTrue()
	pdrMust(t, err)
	falsity, err := manager.MkFalse()
	pdrMust(t, err)
	pHead, err := manager.MkApp(pDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	constraint, err := manager.MkEq(variable, truth)
	pdrMust(t, err)
	pRule, err := ruleManager.NewRule(pHead, []ast.ASTHandle{constraint}, nil, "p-initial")
	pdrMust(t, err)
	pdrMust(t, set.Add(pRule))
	pdrMust(t, pRule.Close())
	qHead, err := manager.MkApp(qDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	pTail, err := manager.MkApp(pDeclaration, []ast.ASTHandle{variable})
	pdrMust(t, err)
	qRule, err := ruleManager.NewRule(qHead, []ast.ASTHandle{pTail}, nil, "q-step")
	pdrMust(t, err)
	pdrMust(t, set.Add(qRule))
	pdrMust(t, qRule.Close())
	pdrMust(t, context.Initialize(set))

	for _, check := range []struct {
		declaration ast.ASTHandle
		argument    ast.ASTHandle
		want        lbool.LBool
	}{
		{pDeclaration, truth, lbool.True},
		{pDeclaration, falsity, lbool.False},
		{qDeclaration, truth, lbool.False},
	} {
		query, x := manager.MkApp(check.declaration, []ast.ASTHandle{check.argument})
		pdrMust(t, x)
		result, x := context.CheckInitial(query, check.want == lbool.True)
		pdrMust(t, x)
		if result.Status != check.want {
			t.Fatal("initial query", check.declaration, check.argument, result.Status, check.want)
		}
		if result.Model != nil {
			pdrMust(t, result.Model.DecRef())
		}
	}
	if transformer, found, x := context.Transformer(qDeclaration); x != nil || !found || transformer == nil {
		t.Fatal("query transformer lookup", transformer, found, x)
	}

	pdrMust(t, context.Close())
	pdrMust(t, set.Close())
	pdrMust(t, ruleManager.Close())
	pdrMust(t, command.Close())
}
