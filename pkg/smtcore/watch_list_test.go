package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTWatchListTypedRegionsAndRemoval(t *testing.T) {
	var watches SMTWatchList
	first, second, third := &SMTClause{}, &SMTClause{}, &SMTClause{}
	for _, clause := range []*SMTClause{first, second, third} {
		if err := watches.InsertClause(clause); err != nil {
			t.Fatal(err)
		}
	}
	one := NewSMTLiteral(1, false)
	two := NewSMTLiteral(2, true)
	three := NewSMTLiteral(3, false)
	for _, literal := range []SMTLiteral{one, two, three} {
		if err := watches.InsertLiteral(literal); err != nil {
			t.Fatal(err)
		}
	}
	if watches.Size() != 6 || watches.FindClause(second) != 1 || watches.FindLiteral(two) != 1 {
		t.Fatal("watch lookup", watches.Size(), watches.FindClause(second), watches.FindLiteral(two))
	}
	if got := watches.Literals(); len(got) != 3 || got[0] != three || got[1] != two || got[2] != one {
		t.Fatal("source reverse literal insertion order", got)
	}
	if !watches.RemoveClause(second) || watches.RemoveClause(second) {
		t.Fatal("clause removal")
	}
	if got := watches.Clauses(); len(got) != 2 || got[0] != first || got[1] != third {
		t.Fatal("clause order after removal", got)
	}
	if !watches.RemoveLiteral(two) || watches.RemoveLiteral(two) {
		t.Fatal("literal removal")
	}
	if got := watches.Literals(); len(got) != 2 || got[0] != three || got[1] != one {
		t.Fatal("literal order after removal", got)
	}
}

func TestSMTWatchListCompactionResetAndGuards(t *testing.T) {
	var watches SMTWatchList
	a, b, c := &SMTClause{}, &SMTClause{}, &SMTClause{}
	for _, clause := range []*SMTClause{a, b, c} {
		if err := watches.InsertClause(clause); err != nil {
			t.Fatal(err)
		}
	}
	clauses := watches.Clauses()
	clauses[1] = clauses[2]
	if err := watches.SetClauseCount(2); err != nil {
		t.Fatal(err)
	}
	if got := watches.Clauses(); len(got) != 2 || got[0] != a || got[1] != c {
		t.Fatal("in-place compaction", got)
	}
	if err := watches.SetClauseCount(3); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("wide clause end", err)
	}
	if err := (*SMTWatchList)(nil).InsertClause(a); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("nil insertion", err)
	}
	if err := watches.InsertLiteral(SMTFalseLiteral); err != nil {
		t.Fatal(err)
	}
	watches.Reset()
	if watches.Size() != 0 || watches.Clauses() == nil || watches.Literals() == nil {
		t.Fatal("reset should retain allocated slices")
	}
	if err := watches.InsertLiteral(SMTTrueLiteral); err != nil {
		t.Fatal(err)
	}
	watches.ResetAndReleaseMemory()
	if watches.Size() != 0 || watches.Clauses() != nil || watches.Literals() != nil {
		t.Fatal("release reset")
	}
}
