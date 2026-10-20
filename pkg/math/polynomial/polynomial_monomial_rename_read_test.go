package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialRenameLeafRawSortIgnoresAssertionOwner(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 3; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	a, e := m.MkMonomial([]MonomialPower{{0, 2}, {1, 3}})
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	// Source rename has changed the key before its caller rebuilds the table.
	// Abort this known caller root through its exact token, not source hash erase.
	t.Cleanup(func() { serviceMust(t, m.CleanupValue(&a)) })
	before := m.config.Budget.Used()
	polynomialClosedCellAssertionContext(t)
	serviceMust(t, m.renameObject(a, []uint32{2, 0, 1}))
	expected := []MonomialPower{{0, 3}, {2, 2}}
	for i, want := range expected {
		v, e := m.powerFieldCore(a, uint32(i), 0)
		serviceMust(t, e)
		d, e := m.powerFieldCore(a, uint32(i), 4)
		serviceMust(t, e)
		if v != want.Var || d != want.Degree {
			t.Fatal(i, v, d, want)
		}
	}
	if m.config.Budget.Used() != before {
		t.Fatal("raw Go sort acquired arena ownership")
	}
}

// An actual many-to-one map reaches the permutation assertion first, then
// the new-table uniqueness assertion. Continue is defined through insertion;
// this is diagnostic continuation, not a valid variable permutation.
func TestMonomialRenameUniqueAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := "C" + action
			if action == "disabled" {
				choices = ""
			}
			output := polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			m := monomialFixture(t)
			for i := 0; i < 2; i++ {
				_, e := m.MkVar()
				serviceMust(t, e)
			}
			a, e := m.MkVariable(0, 1)
			serviceMust(t, e)
			b, e := m.MkVariable(1, 1)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(a))
			serviceMust(t, m.IncRef(b))
			// Caller roots remain real even when the invalid mapping creates equal keys.
			t.Cleanup(func() { serviceMust(t, m.CleanupValue(&b)); serviceMust(t, m.CleanupValue(&a)) })
			body := m.Rename([]uint32{0, 0})
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(body, ErrMonomialAssertion) || m.table.entries.Len() != 3 {
					t.Fatal(body, m.table.entries.Len())
				}
				_, e = output.Seek(0, 0)
				serviceMust(t, e)
				text, e := io.ReadAll(output)
				serviceMust(t, e)
				if !strings.Contains(string(text), "Line: 1279") || !strings.Contains(string(text), "Line: 1289\n!new_table.contains(m)") {
					t.Fatal(string(text))
				}
			} else {
				serviceMust(t, body)
				if m.table.entries.Len() != 2 {
					t.Fatal(m.table.entries.Len())
				}
				canonical, e := m.MkVariable(0, 1)
				serviceMust(t, e)
				if canonical != b {
					t.Fatal("rename insertion did not replace the prior equal key", canonical, b)
				}
			}
			if m.renameFound != nil || m.renameTable.entries.Len() != 0 {
				t.Fatal("source local table did not retire")
			}
		})
	}
}
