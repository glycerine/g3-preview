package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTUserNameDictionaryRegistrationAndFreshReplay(t *testing.T) {
	m := astTestManager(t)
	for i := uint32(0); i < 7; i++ {
		k, e := m.registerUserName(util.NumericalSymbol(i))
		astTestMust(t, e)
		if k != int32(i) {
			t.Fatal("registration order", k, i)
		}
	}
	k, e := m.registerUserName(util.NumericalSymbol(99))
	astTestMust(t, e)
	if k != 7 || len(m.builtin.userNames) != 8 || m.builtin.userTable.Size() != 8 {
		t.Fatal("registration order", k, len(m.builtin.userNames), m.builtin.userTable.Size())
	}
	duplicate, e := m.registerUserName(util.NumericalSymbol(99))
	astTestMust(t, e)
	if duplicate != 7 || len(m.builtin.userNames) != 8 {
		t.Fatal("duplicate registration", duplicate, len(m.builtin.userNames))
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
