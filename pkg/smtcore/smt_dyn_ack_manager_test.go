package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTDynamicAckManagerBuildsPairAndTripleClauses(t *testing.T) {
	manager := astTestManager(t)
	sort, values := smtUFConstants(t, manager, "dynamic-ack-manager-", 3)
	function, err := manager.MkFuncDecl(astTestSymbol(t, manager, "dynamic-ack-manager-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	applications := make([]ast.ASTHandle, len(values))
	for index, value := range values {
		applications[index], err = manager.MkApp(function, []ast.ASTHandle{value})
		astTestMust(t, err)
	}
	_, _, state := smtInternalizerFixture(t, manager, nil)
	config := defaultSMTDynAckState()
	config.Threshold = 1
	config.Factor = 1
	config.Equalities = true
	astTestMust(t, state.ConfigureDynamicAck(config))
	astTestMust(t, state.dynamicAck.schedule.conflictCongruence(applications[0], applications[1], dynAckRoot))
	astTestMust(t, state.dynamicAck.schedule.usedEquality(applications[0], applications[1], applications[2], true))
	state.statistics.NumConflicts = 2
	progress, err := state.dynamicAck.propagate()
	astTestMust(t, err)
	if !progress || state.statistics.NumDynAck != 2 || len(state.clauses) != 2 {
		t.Fatal("dynamic Ackermann propagation", progress, state.statistics.NumDynAck, len(state.clauses))
	}
	if len(state.clauses[0].Literals) != 2 || len(state.clauses[1].Literals) != 3 {
		t.Fatal("dynamic Ackermann clause shapes", state.clauses[0].Literals, state.clauses[1].Literals)
	}
	pair := dynAckPair{applications[0], applications[1]}
	triple := dynAckTriple{applications[0], applications[1], applications[2]}
	if _, found := state.dynamicAck.pairs.instantiated[pair]; !found {
		t.Fatal("pair instantiation was not retained")
	}
	if _, found := state.dynamicAck.triples.instantiated[triple]; !found {
		t.Fatal("triple instantiation was not retained")
	}
}
