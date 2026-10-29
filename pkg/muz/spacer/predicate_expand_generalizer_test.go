package spacer

// Source: src/muz/spacer/spacer_context.cpp context::expand_node Farkas and
// false-lemma generalizer guards.

import (
	"testing"

	"github.com/glycerine/g3/pkg/smtcore"
)

func TestLemmaGeneralizerGateRequiresNonFalseFarkasLemma(t *testing.T) {
	manager, globals := newSpacerTestServices(t)
	owner, err := NewManager(manager, nil, 1, smtcore.SMTSolverConfig{Params: globals.Context, Globals: globals})
	if err != nil {
		t.Fatal(err)
	}
	truth, err := manager.MkTrue()
	if err != nil {
		t.Fatal(err)
	}
	falsity, err := manager.MkFalse()
	if err != nil {
		t.Fatal(err)
	}
	transformer, err := NewPredicateTransformer(owner, "generalizer-gate", falsity, truth, PropSolverConfig{})
	if err != nil {
		t.Fatal(err)
	}
	obligation, err := NewProofObligation(nil, transformer, 0, 0, truth, nil)
	if err != nil {
		t.Fatal(err)
	}
	lemma, err := NewLemma(owner, truth, 0)
	if err != nil {
		t.Fatal(err)
	}
	if enabled, err := shouldGeneralizeLemma(obligation, lemma); err != nil || !enabled {
		t.Fatal("Farkas lemma generalizer gate", enabled, err)
	}
	if err = obligation.SetUseFarkas(false); err != nil {
		t.Fatal(err)
	}
	if enabled, err := shouldGeneralizeLemma(obligation, lemma); err != nil || enabled {
		t.Fatal("subset lemma generalizer gate", enabled, err)
	}
	if err = obligation.SetUseFarkas(true); err != nil {
		t.Fatal(err)
	}
	falseLemma, err := NewLemma(owner, falsity, 0)
	if err != nil {
		t.Fatal(err)
	}
	if enabled, err := shouldGeneralizeLemma(obligation, falseLemma); err != nil || enabled {
		t.Fatal("false lemma generalizer gate", enabled, err)
	}
	if err = falseLemma.Close(); err != nil {
		t.Fatal(err)
	}
	if err = lemma.Close(); err != nil {
		t.Fatal(err)
	}
	if err = obligation.Close(); err != nil {
		t.Fatal(err)
	}
	if err = transformer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = owner.Close(); err != nil {
		t.Fatal(err)
	}
}
