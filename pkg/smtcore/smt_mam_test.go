package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func smtMAMFixture(t *testing.T) (*ast.ASTManager, *SMTInternalizerState, *SMTQuantifierRegistry, *SMTMAM) {
	t.Helper()
	m := astTestManager(t)
	costs, err := NewSMTQICostModel(m, "0", "0", nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, costs.Close()) })
	registry, err := NewSMTQuantifierRegistry(m, NewSMTQIQueueSchedule(m, 10, 20, false), costs, 100)
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, registry)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	mam, err := NewSMTMAM(state, registry)
	astTestMust(t, err)
	return m, state, registry, mam
}

func smtMAMQuantifier(t *testing.T, m *ast.ASTManager, name string, sort ast.ASTHandle, terms []ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	pattern, err := m.MkPattern(terms)
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{astTestSymbol(t, m, name+"-x")}, Body: truth, Patterns: []ast.ASTHandle{pattern},
	})
	astTestMust(t, err)
	return quantifier, pattern
}

func TestSMTMAMMatchesUnaryAndRepeatedVariablesModuloEquality(t *testing.T) {
	m, state, registry, mam := smtMAMFixture(t)
	sort, err := m.MkSort(astTestSymbol(t, m, "mam-unary-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	predicateInfo, err := m.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	predicate, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "mam-repeated"), []ast.ASTHandle{sort, sort}, boolean, predicateInfo)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&predicateInfo))
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	trigger, err := m.MkApp(predicate, []ast.ASTHandle{variable, variable})
	astTestMust(t, err)
	quantifier, pattern := smtMAMQuantifier(t, m, "mam-repeated", sort, []ast.ASTHandle{trigger})
	astTestMust(t, registry.Add(quantifier, 0))
	_, a := smtEnodeConstant(t, m, 2600, sort, 2)
	_, b := smtEnodeConstant(t, m, 2601, sort, 3)
	astTestMust(t, state.context.RegisterEnode(a, true))
	astTestMust(t, state.context.RegisterEnode(b, true))
	application, err := m.MkApp(predicate, []ast.ASTHandle{a.owner, b.owner})
	astTestMust(t, err)
	node, err := NewSMTEnode(m, application, state.context.enodes, SMTEnodeOptions{Generation: 4, CongruenceEnabled: true})
	astTestMust(t, err)
	node.congruence = node
	astTestMust(t, state.context.RegisterEnode(node, true))
	astTestMust(t, mam.AddPattern(quantifier, pattern))
	astTestMust(t, mam.Match())
	if len(registry.schedule.newEntries) != 0 {
		t.Fatal("repeated variable matched distinct equality classes")
	}
	astTestMust(t, state.context.SetRelevant(application, false))
	application, err = m.MkApp(predicate, []ast.ASTHandle{a.owner, a.owner})
	astTestMust(t, err)
	node, err = NewSMTEnode(m, application, state.context.enodes, SMTEnodeOptions{Generation: 5, CongruenceEnabled: true})
	astTestMust(t, err)
	node.congruence = node
	astTestMust(t, state.context.RegisterEnode(node, true))
	mam.Relevant(node)
	astTestMust(t, mam.Match())
	if len(registry.schedule.newEntries) != 1 || registry.schedule.newEntries[0].Fingerprint.args[0].root != a.root {
		t.Fatal("repeated-variable trigger binding", registry.schedule.newEntries)
	}
}

func TestSMTMAMJoinsMultiPatternsAndRollsBackPatterns(t *testing.T) {
	m, state, registry, mam := smtMAMFixture(t)
	sort, err := m.MkSort(astTestSymbol(t, m, "mam-multi-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	p := smtQuickPredicate(t, m, "mam-multi-p", sort, boolean)
	q := smtQuickPredicate(t, m, "mam-multi-q", sort, boolean)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	pv, err := m.MkApp(p, []ast.ASTHandle{variable})
	astTestMust(t, err)
	qv, err := m.MkApp(q, []ast.ASTHandle{variable})
	astTestMust(t, err)
	quantifier, pattern := smtMAMQuantifier(t, m, "mam-multi", sort, []ast.ASTHandle{pv, qv})
	astTestMust(t, registry.Add(quantifier, 0))
	_, a := smtEnodeConstant(t, m, 2610, sort, 1)
	_, b := smtEnodeConstant(t, m, 2611, sort, 2)
	astTestMust(t, state.context.RegisterEnode(a, true))
	astTestMust(t, state.context.RegisterEnode(b, true))
	smtQuickApplicationNode(t, m, state.context, p, a, true)
	smtQuickApplicationNode(t, m, state.context, p, b, true)
	smtQuickApplicationNode(t, m, state.context, q, b, true)
	astTestMust(t, mam.PushScope())
	astTestMust(t, mam.AddPattern(quantifier, pattern))
	astTestMust(t, mam.Match())
	if len(registry.schedule.newEntries) != 1 || registry.schedule.newEntries[0].Fingerprint.args[0].root != b.root {
		t.Fatal("multi-pattern join", registry.schedule.newEntries)
	}
	astTestMust(t, mam.PopScopes(1))
	registry.schedule.newEntries = nil
	mam.Relevant(b)
	astTestMust(t, mam.Match())
	if len(registry.schedule.newEntries) != 0 {
		t.Fatal("popped pattern remained active")
	}
}

func TestSMTMAMTracksNestedGroundPatternTermsAsShared(t *testing.T) {
	m, state, registry, mam := smtMAMFixture(t)
	sort, err := m.MkSort(astTestSymbol(t, m, "mam-shared-sort"), ast.ASTSortInfo{})
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	info, err := m.NewFuncDeclInfo(ast.NullFamilyID, -1, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	predicate, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "mam-shared-p"), []ast.ASTHandle{sort, sort}, boolean, info)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	constant, err := m.MkConst(astTestSymbol(t, m, "mam-shared-c"), sort)
	astTestMust(t, err)
	variable, err := m.MkVar(0, sort)
	astTestMust(t, err)
	trigger, err := m.MkApp(predicate, []ast.ASTHandle{variable, constant})
	astTestMust(t, err)
	quantifier, pattern := smtMAMQuantifier(t, m, "mam-shared", sort, []ast.ASTHandle{trigger})
	astTestMust(t, registry.Add(quantifier, 0))
	astTestMust(t, mam.PushScope())
	astTestMust(t, mam.AddPattern(quantifier, pattern))
	node := state.context.enodes[constant]
	if node == nil || !mam.IsShared(node) {
		t.Fatal("nested ground pattern term was not internalized and shared")
	}
	astTestMust(t, mam.PopScopes(1))
	if mam.IsShared(node) {
		t.Fatal("shared pattern term survived pattern rollback")
	}
}
