package transforms_test

// Source: src/muz/transforms/dl_mk_unfold.cpp one-round tail expansion.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/muz/transforms"
)

func TestRuleUnfoldExpandsOriginalTailOnce(t *testing.T) {
	f := newConeFixture(t, "P", "Q", "Goal")
	f.add(t, f.applications["P"], "fact-P")
	f.edge(t, "P", "Q")
	f.edge(t, "Q", "Goal")
	coneMust(t, f.set.SetOutputPredicate(f.declarations["Goal"]))
	valid, err := f.set.Finalize()
	coneMust(t, err)
	if !valid {
		t.Fatal("unfold fixture rejected")
	}
	transformer := base.NewRuleTransformer()
	t.Cleanup(func() { coneMust(t, transformer.Close()) })
	coneMust(t, transformer.Register(transforms.NewRuleUnfoldPlugin(f.manager, f.rules, f.globals.Context, f.globals, 0)))
	modified, err := transformer.Apply(f.set)
	coneMust(t, err)
	if !modified {
		t.Fatal("unfold plugin returned no replacement")
	}
	count, err := f.set.Len()
	coneMust(t, err)
	if count != 3 {
		t.Fatalf("unfolded rules %d, want 3", count)
	}
	wantTails := map[ast.ASTHandle]int{
		f.declarations["P"]:    0,
		f.declarations["Q"]:    0,
		f.declarations["Goal"]: 1,
	}
	for index := 0; index < count; index++ {
		rule, getErr := f.set.Rule(index)
		coneMust(t, getErr)
		head, getErr := rule.HeadDeclaration()
		tails, tailErr := rule.UninterpretedTailSize()
		getErr = errors.Join(getErr, tailErr, rule.Close())
		coneMust(t, getErr)
		if tails != wantTails[head] {
			t.Fatal("unfolded tail count", head, tails, wantTails[head])
		}
	}
}
