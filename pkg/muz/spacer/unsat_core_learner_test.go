package spacer_test

// Source: src/muz/spacer/spacer_unsat_core_learner.cpp proof marking and
// unsat_core_plugin_lemma lowest B-pure split extraction.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/spacer"
)

func TestUnsatCoreLearnerMarksProofAndExtractsLowestBPureSplit(t *testing.T) {
	command, err := cmd_context.NewStandaloneContext()
	spacerMust(t, err)
	manager, err := command.Manager()
	spacerMust(t, err)
	spacerMust(t, manager.ToggleProofMode(ast.ASTProofEnabled))
	p := spacerBool(t, manager, "new-core-b")
	q := spacerBool(t, manager, "new-core-outside-b")
	notP, err := manager.MkNot(p)
	spacerMust(t, err)
	bProof, err := manager.MkProofAsserted(p)
	spacerMust(t, err)
	aProof, err := manager.MkProofAsserted(notP)
	spacerMust(t, err)
	root, err := manager.MkProofUnitResolution([]ast.ASTHandle{bProof, aProof})
	spacerMust(t, err)
	rootRef, err := manager.NewRef(root)
	spacerMust(t, err)

	learner, err := spacer.NewUnsatCoreLearner(manager)
	spacerMust(t, err)
	plugin, err := spacer.NewUnsatCoreLemmaPlugin(learner)
	spacerMust(t, err)
	spacerMust(t, learner.RegisterPlugin(plugin))
	core, err := learner.ComputeUnsatCore(root, []ast.ASTHandle{p})
	spacerMust(t, err)
	if len(core) != 1 {
		t.Fatal("ordinary lemma plugin core size", len(core))
	}
	coreHandle, err := manager.RefGet(core[0])
	spacerMust(t, err)
	if coreHandle != p {
		t.Fatal("ordinary lemma plugin selected wrong split", coreHandle, p)
	}
	spacerMust(t, manager.DelRef(&core[0]))
	pure, err := learner.OnlyContainsSymbolsB(p)
	spacerMust(t, err)
	if !pure {
		t.Fatal("B assertion was not B-pure")
	}
	pure, err = learner.OnlyContainsSymbolsB(q)
	spacerMust(t, err)
	if pure {
		t.Fatal("foreign declaration was classified as B-pure")
	}
	spacerMust(t, learner.Close())
	if _, err = learner.ComputeUnsatCore(root, nil); !errors.Is(err, spacer.ErrUnsatCoreLearnerOwner) {
		t.Fatal("closed learner accepted proof", err)
	}
	if err = plugin.Finalize(); !errors.Is(err, spacer.ErrUnsatCoreLearnerOwner) {
		t.Fatal("closed owned plugin accepted operation", err)
	}
	spacerMust(t, manager.DelRef(&rootRef))
	spacerMust(t, command.Close())
}
