package spacer_test

// Source: src/muz/spacer/spacer_context.cpp pob and pob_queue.

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/spacer"
)

func TestSpacerProofObligationBindingsDirtyStateAndQueue(t *testing.T) {
	command, manager, owner := lemmaOwnerFixture(t)
	truth, err := manager.MkTrue()
	spacerMust(t, err)
	falsehood, err := manager.MkFalse()
	spacerMust(t, err)
	transformer, err := spacer.NewPredicateTransformer(owner, "proof-obligation", falsehood, truth, spacer.PropSolverConfig{})
	spacerMust(t, err)
	x := spacerBool(t, manager, "proof-obligation-x")
	root, err := spacer.NewProofObligation(nil, transformer, 2, 0, x, []ast.ASTHandle{x})
	spacerMust(t, err)
	post, err := root.Post()
	spacerMust(t, err)
	skolem, err := root.Skolems()
	spacerMust(t, err)
	if len(skolem) != 1 || post != skolem[0] || post == x {
		t.Fatal("proof-obligation skolem replacement", post, skolem, x)
	}
	notPost, err := manager.MkNot(post)
	spacerMust(t, err)
	added, err := transformer.AddLemma(notPost, spacer.InfinityLevel)
	spacerMust(t, err)
	if !added {
		t.Fatal("blocking lemma not added")
	}
	blocked, usesLevel, err := transformer.IsBlocked(root)
	spacerMust(t, err)
	if !blocked || usesLevel != spacer.InfinityLevel {
		t.Fatal("ground proof-obligation blocking", blocked, usesLevel)
	}
	qblocked, err := transformer.IsQuantifierBlocked(root)
	spacerMust(t, err)
	if !qblocked {
		t.Fatal("quantifier proof-obligation blocking")
	}
	lemma, err := spacer.NewLemmaFromProofObligation(root, nil, 2)
	spacerMust(t, err)
	lemmaBody, err := lemma.Body()
	spacerMust(t, err)
	if kind, err := manager.GetKind(lemmaBody); err != nil || kind != ast.ASTQuantifier {
		t.Fatal("proof-obligation quantified lemma", kind, err)
	}
	instances, err := lemma.Instances()
	spacerMust(t, err)
	if len(instances) != 1 {
		t.Fatal("proof-obligation lemma instances", len(instances))
	}
	instance, err := manager.RefGet(instances[0])
	spacerMust(t, err)
	notX, err := manager.MkNot(x)
	spacerMust(t, err)
	if instance != notX {
		t.Fatal("proof-obligation lemma instance", instance, notX)
	}
	spacerMust(t, manager.DelRef(&instances[0]))
	spacerMust(t, lemma.Close())
	spacerMust(t, root.NewPost(truth))
	if dirty, err := root.Dirty(); err != nil || !dirty {
		t.Fatal("dirty proof obligation", dirty, err)
	}
	spacerMust(t, root.Clean())
	if post, err = root.Post(); err != nil || post != truth {
		t.Fatal("clean proof obligation", post, truth, err)
	}
	child, err := spacer.NewProofObligation(root, transformer, 1, 1, truth, nil)
	spacerMust(t, err)
	deep, err := spacer.NewProofObligation(root, transformer, 2, 1, truth, nil)
	spacerMust(t, err)
	var queue spacer.ProofObligationQueue
	spacerMust(t, queue.SetRoot(root))
	spacerMust(t, queue.Push(deep))
	spacerMust(t, queue.Push(child))
	top, err := queue.Top()
	spacerMust(t, err)
	if top != child {
		t.Fatal("proof-obligation queue priority", top, child)
	}
	popped, err := queue.Pop()
	spacerMust(t, err)
	if popped != child {
		t.Fatal("proof-obligation queue pop", popped, child)
	}
	top, err = queue.Top()
	spacerMust(t, err)
	if top != root {
		t.Fatal("proof-obligation root eligibility", top, root)
	}
	spacerMust(t, root.Close())
	if closed, err := child.IsClosed(); err != nil || !closed {
		t.Fatal("proof-obligation close cascade", closed, err)
	}
	spacerMust(t, child.Release())
	spacerMust(t, deep.Release())
	spacerMust(t, root.Release())
	spacerMust(t, transformer.Close())
	spacerMust(t, owner.Close())
	spacerMust(t, command.Close())
}
