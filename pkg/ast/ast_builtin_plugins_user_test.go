package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTUserNameDictionaryFailureKeepsNamePrefixAndFreshReplay(t *testing.T) {
	m := astTestManager(t)
	for i := uint32(0); i < 7; i++ {
		k, e := m.registerUserName(util.NumericalSymbol(i))
		astTestMust(t, e)
		if k != int32(i) {
			t.Fatal("registration order", k, i)
		}
	}
	m.BudgetField.FailAfter(0)
	if _, e := m.registerUserName(util.NumericalSymbol(99)); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("source map expansion", e)
	}
	if len(m.builtin.userNames) != 8 || m.builtin.userTable.Size() != 7 {
		t.Fatal("source name push precedes map failure")
	}
	m.BudgetField.FailAfter(-1)
	k, e := m.registerUserName(util.NumericalSymbol(99))
	astTestMust(t, e)
	if k != 8 || len(m.builtin.userNames) != 9 {
		t.Fatal("retry must not roll back/consume failed name prefix", k, len(m.builtin.userNames))
	}
	fresh, e := m.freshUserPlugin()
	astTestMust(t, e)
	defer fresh.close()
	// mk_fresh iterates all source names through register_name; it does not copy
	// the old dictionary's kind8 or duplicate name-vector backing directly.
	k, ok, e := fresh.dictionary.Find(util.NumericalSymbol(99))
	astTestMust(t, e)
	if !ok || k != 7 || len(fresh.names) != 8 {
		t.Fatal("real fresh replay must rebuild kinds", k, ok, len(fresh.names))
	}
}
