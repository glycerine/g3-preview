package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTUserNameDictionaryRegistrationAndFreshReplay(t *testing.T) {
	m := astTestManager(t)
	for i := uint32(0); i < 7; i++ {
		k, e := m.registerUserName(usymbol.Numerical(i))
		astTestMust(t, e)
		if k != int32(i) {
			t.Fatal("registration order", k, i)
		}
	}
	k, e := m.registerUserName(usymbol.Numerical(99))
	astTestMust(t, e)
	if k != 7 || len(m.builtin.userNames) != 8 || m.builtin.userTable.Size() != 8 {
		t.Fatal("registration order", k, len(m.builtin.userNames), m.builtin.userTable.Size())
	}
	duplicate, e := m.registerUserName(usymbol.Numerical(99))
	astTestMust(t, e)
	if duplicate != 7 || len(m.builtin.userNames) != 8 {
		t.Fatal("duplicate registration", duplicate, len(m.builtin.userNames))
	}
	fresh, e := m.freshUserPlugin()
	astTestMust(t, e)
	defer fresh.close()
	// mk_fresh iterates all source names through register_name; it does not copy
	// the old dictionary's kind8 or duplicate name-vector backing directly.
	k, ok, e := fresh.dictionary.Find(usymbol.Numerical(99))
	astTestMust(t, e)
	if !ok || k != 7 || len(fresh.names) != 8 {
		t.Fatal("real fresh replay must rebuild kinds", k, ok, len(fresh.names))
	}
}

func TestASTUserNameNativeGrowthAndFreshReplay(t *testing.T) {
	m := astTestManager(t)
	budget := m.BudgetField
	before := budget.Used()
	budget.FailAfter(0)
	defer budget.FailAfter(-1)
	for i := uint32(0); i < 129; i++ {
		kind, e := m.registerUserName(usymbol.Numerical(i))
		astTestMust(t, e)
		if kind != int32(i) {
			t.Fatal("registration order", i, kind)
		}
	}
	budget.FailAfter(-1)
	if budget.Used() != before || len(m.builtin.userNames) != 129 {
		t.Fatal("native name growth charged arena or lost order", budget.Used(), before, len(m.builtin.userNames))
	}
	fresh, e := m.freshUserPlugin()
	astTestMust(t, e)
	defer fresh.close()
	for i := uint32(0); i < 129; i++ {
		kind, found, e := fresh.dictionary.Find(usymbol.Numerical(i))
		astTestMust(t, e)
		if !found || kind != int32(i) || fresh.names[i] != usymbol.Numerical(i) {
			t.Fatal("fresh plugin replay", i, kind, found)
		}
	}
}
