// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestASTUserNameDictionaryFailureKeepsNamePrefixAndFreshReplay(t *testing.T) {
	m := astTestManager(t)
	for i := uint32(0); i < 7; i++ {
		k, e := m.registerUserName(NumericalSymbol(i))
		astTestMust(t, e)
		if k != int32(i) {
			t.Fatal("registration order", k, i)
		}
	}
	m.budget.FailAfter(0)
	if _, e := m.registerUserName(NumericalSymbol(99)); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("source map expansion", e)
	}
	if len(m.builtin.userNames) != 8 || m.builtin.userTable.size != 7 {
		t.Fatal("source name push precedes map failure")
	}
	m.budget.FailAfter(-1)
	k, e := m.registerUserName(NumericalSymbol(99))
	astTestMust(t, e)
	if k != 8 || len(m.builtin.userNames) != 9 {
		t.Fatal("retry must not roll back/consume failed name prefix", k, len(m.builtin.userNames))
	}
	fresh, e := m.freshUserPlugin()
	astTestMust(t, e)
	defer fresh.close()
	// mk_fresh iterates all source names through register_name; it does not copy
	// the old dictionary's kind8 or duplicate name-vector backing directly.
	k, ok, e := fresh.dictionary.find(NumericalSymbol(99))
	astTestMust(t, e)
	if !ok || k != 7 || len(fresh.names) != 8 {
		t.Fatal("real fresh replay must rebuild kinds", k, ok, len(fresh.names))
	}
}
