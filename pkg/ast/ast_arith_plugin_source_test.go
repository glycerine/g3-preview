// Source: src/ast/arith_decl_plugin.cpp:115-332; src/ast/ast.cpp:1635-1638.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTArithmeticSourceRegistrationAndDeclarations(t *testing.T) {
	m := astTestManager(t)
	id, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	p := m.arithmeticPlugin
	if p == nil || !p.published || p.family != id || p.completed != 46 {
		t.Fatal("actual completed source fields", p)
	}
	if p.fields[0] == p.fields[1] {
		t.Fatal("Real and Int are distinct source sorts")
	}
	for i, r := range p.fields {
		astTestMust(t, m.validNode(r))
		if m.nodeAt(r.slot).rc == 0 {
			t.Fatal("actual plugin acquired reference", i, r)
		}
	}
	for _, index := range []int{6, 9, 15, 18} {
		flags, e := m.funcFlags(p.fields[index])
		astTestMust(t, e)
		if !flags.LeftAssociative || !flags.RightAssociative || !flags.FlatAssociative || !flags.Commutative {
			t.Fatal("literal set_associative sets both source bits", index, flags)
		}
	}
	real, e := m.MkFamilySort(id, 0, nil)
	astTestMust(t, e)
	integer, e := m.MkFamilySort(id, 1, nil)
	astTestMust(t, e)
	if real != m.external(p.fields[0]) || integer != m.external(p.fields[1]) {
		t.Fatal("actual arithmetic cached sort dispatcher")
	}
	invalid, e := m.MkFamilySort(id, 2, nil)
	astTestMust(t, e)
	if !invalid.IsNull() {
		t.Fatal("source arithmetic mk_sort default NULL")
	}
	again, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	if again != id || m.arithmeticPlugin != p {
		t.Fatal("source pointer-present registration shortcut")
	}
}

func TestASTArithmeticSourceEveryRegistrationReservation(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	success, published, unfinished := false, false, false
	for ordinal := int64(0); ordinal < 1024; ordinal++ {
		b := arena.NewBudget(1 << 30)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, e = m.RegisterArithmeticPlugin()
		if b.Used() != m.Stats().AllocatedBytes {
			t.Fatal("immediate named cold graph", ordinal, b.Used(), m.Stats())
		}
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(ordinal, e)
			}
			published = published || m.arithmeticPlugin != nil
			unfinished = unfinished || m.arithmeticPluginFailed != nil
			if p := m.arithmeticPlugin; p != nil {
				if !m.pluginAt(p.family).present || !p.published || p.bodyCause != e {
					t.Fatal("published slot retained before failed set_manager", ordinal)
				}
				old := p
				_, retry := m.RegisterArithmeticPlugin()
				astTestMust(t, retry)
				if m.arithmeticPlugin != old {
					t.Fatal("partial plugin must not be reconstructed", ordinal)
				}
			}
		} else {
			success = true
			t.Log("first successful source prefix", ordinal)
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("owning cold retirement", ordinal, b.Used())
		}
		if success {
			break
		}
	}
	if !success || !published || !unfinished {
		t.Fatal(success, published, unfinished)
	}
}

// Independent unified Symbol+AST Budget sweep. The numerical service is an
// explicitly borrowed, separate owner; this constructor never reaches it.
func TestASTArithmeticSourceEverySymbolAndMemberAcquisition(t *testing.T) {
	_, q, _ := astTestOwners(t)
	success := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 30)
		sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		numeralBefore, e := q.Stats()
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterArithmeticPlugin()
		if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
			t.Fatal("actual named symbol and AST graph before fault lift", ordinal, b.Used(), m.Stats(), sy.Stats())
		}
		numeralAfter, e := q.Stats()
		astTestMust(t, e)
		if numeralBefore != numeralAfter {
			t.Fatal("unused borrowed numerical owner", ordinal)
		}
		if call != nil && !errors.Is(call, arena.ErrInjectedFailure) {
			t.Fatal(ordinal, call)
		}
		success = call == nil
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		if b.Used() != astTestSymbolGraph(sy) {
			t.Fatal("foreign borrowed symbols survive AST Close", ordinal, b.Used(), sy.Stats())
		}
		astTestMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("actual namespace end", ordinal, b.Used())
		}
		if success {
			t.Log("first unified source prefix", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("did not cross all actual symbol/member reservations")
	}
}

func TestASTArithmeticSourceFinalizeActualFieldOrderAndSeparateOwnClose(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 30)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	plugin := m.arithmeticPlugin
	var actual [46]astNodeRef
	actual = plugin.fields
	// Source plugin finalize drops exactly its acquired refs; the manager's later
	// table pass owns the unreferenced nodes and may need a real root-list reserve.
	b.FailAfter(0)
	e = m.SourceFinalize()
	if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	for j, r := range plugin.fields {
		if r.slot != 0 {
			t.Fatal("all literal plugin DEC fields consumed before table pass", j, r, actual[j])
		}
	}
	if b.Used() != m.Stats().AllocatedBytes {
		t.Fatal("immediate source stop graph", b.Used(), m.Stats())
	}
	original := m.sourceDTCause
	b.FailAfter(-1)
	astTestMust(t, m.Close())
	if m.sourceDTCause != original {
		t.Fatal("owning close must not rewrite source cause")
	}
	if b.Used() != 0 {
		t.Fatal("exact context end", b.Used())
	}
	// Neither borrowed child was adopted by this manager's teardown.
	_, e = sy.Intern("borrowed survivor")
	astTestMust(t, e)
	_, e = q.Stats()
	astTestMust(t, e)
}

// Same source DEBUG counter-domain fixture as ast_core_refcount_test.go. The
// initial sort and reference are real; only the unreachable max counter is a
// representation seed, not a physical allocation refusal or algorithm RED.
func TestASTArithmeticReusedRealSortRejectedINCOwnAbort(t *testing.T) {
	if !ASTSourceDebug {
		t.Skip("source DEBUG counter-domain guard")
	}
	m := astTestManager(t)
	name, e := m.Symbols.Intern("Real")
	astTestMust(t, e)
	arith, e := m.Symbols.Intern("arith")
	astTestMust(t, e)
	family, e := m.families.MkID(arith)
	astTestMust(t, e)
	info, e := m.NewSortInfo(family, 0, nil, InfiniteSortSize(), false)
	astTestMust(t, e)
	h, e := m.MkSort(name, info)
	astTestMust(t, e)
	astTestMust(t, m.DelSortInfo(&info))
	ref, e := m.NewRef(h)
	astTestMust(t, e)
	r, e := m.internal(h)
	astTestMust(t, e)
	original := m.nodeAt(r.slot).rc
	m.nodeAt(r.slot).rc = ^uint32(0)
	_, call := m.RegisterArithmeticPlugin()
	if call != ErrASTDomain || m.arithmeticPlugin == nil || m.arithmeticPlugin.fields[0] != r {
		t.Fatal("actual published sort before refused INC", call)
	}
	own := m.finalizeArithmeticPlugin()
	count := m.nodeAt(r.slot).rc
	// Restore the seeded source precondition before enclosing ordinary cleanup.
	m.nodeAt(r.slot).rc = original
	astTestMust(t, m.DelRef(&ref))
	if own != nil || count != ^uint32(0) {
		t.Fatal("own abort consumed foreign unacquired Real reference", own, count)
	}
}
