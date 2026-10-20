package ast

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestLegacySMTRenamingReservedLegalityQuotingAndCollisions(t *testing.T) {
	m := astTestManager(t)
	r, err := NewSMTRenaming(m.Symbols)
	astTestMust(t, err)
	check := func(source, want string) {
		t.Helper()
		s, e := m.Symbols.Intern(source)
		astTestMust(t, e)
		got, e := r.GetSymbol(s)
		astTestMust(t, e)
		text, e := m.Symbols.Display(got)
		astTestMust(t, e)
		if text != want {
			t.Fatalf("legacy SMT rename %q: got %q want %q", source, text, want)
		}
		again, e := r.GetSymbol(s)
		astTestMust(t, e)
		if again != got {
			t.Fatal("legacy SMT rename changed after publication", source)
		}
	}
	check("and", "and")
	check("alpha_1", "alpha_1")
	check("a b", "|a b|")
	check("|a b|", "|a b|1")
	check("", "")
	numerical, err := r.GetSymbol(usymbol.Numerical(7))
	astTestMust(t, err)
	text, err := m.Symbols.Display(numerical)
	astTestMust(t, err)
	if text != "k!70" {
		t.Fatal("legacy numerical symbol suffix", text)
	}
}
