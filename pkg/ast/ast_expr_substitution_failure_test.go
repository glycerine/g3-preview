// Licensed under the MIT license.
package ast

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTExprSubstitutionConsumedEraseFinishesSameEntry(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 10200)
	value := astDependencyExpr(t, m, 10201)
	dep, e := m.DependencyLeaf(astDependencyExpr(t, m, 10202))
	astTestMust(t, e)
	s, e := m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, dep))
	m.BudgetField.FailAfter(0)
	e = m.ExprSubstitutionErase(s, key)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	found, e := m.ExprSubstitutionContains(s, key)
	astTestMust(t, e)
	if found {
		t.Fatal("consumed deletion error must finish same entry")
	}
	if m.dependencies.live != 0 {
		t.Fatal("dependency roots before Close", m.dependencies.live)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionDestructorExecutesSourceSparseReset(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	var keys [100]ASTHandle
	value := astDependencyExpr(t, m, 10300)
	for i := range keys {
		keys[i] = astDependencyExpr(t, m, uint32(10400+i))
		astTestMust(t, m.ExprSubstitutionInsert(s, keys[i], value, ASTHandle{}, ASTDependency{}))
	}
	for i := 0; i < 95; i++ {
		astTestMust(t, m.ExprSubstitutionErase(s, keys[i]))
	}
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	if table.capacity != 256 {
		t.Fatal(table)
	}
	m.BudgetField.FailAfter(0)
	e = m.DelExprSubstitution(&s)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("source sparse reset allocation must run before member destruction", e)
	}
	if s != (ASTExprSubstitution{}) {
		t.Fatal("fully consumed lexical owner must be reclaimed", s)
	}
}
func TestASTExprSubstitutionResetRefusalKeepsConsumedKeyProgress(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	key, e := m.MkVar(11500, sort)
	astTestMust(t, e)
	value, e := m.MkVar(11501, sort)
	astTestMust(t, e)
	base := m.allocator.Stats().Live
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, ASTDependency{}))
	m.nodeAt(value.slot).rc = 0 // Actual refused source counter guard, no fake callback.
	e = m.ExprSubstitutionReset(s)
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal(e)
	}
	if _, e = m.GetRefCount(key); !errors.Is(e, ErrASTHandle) {
		t.Fatal("consumed key must be dead", e)
	}
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	entry, e := m.substEntry(table.payload, table.index)
	astTestMust(t, e)
	if entry.owned != 2 {
		t.Fatal("retain only unconsumed value", entry)
	}
	m.nodeAt(value.slot).rc = 1
	astTestMust(t, m.ExprSubstitutionReset(s))
	astTestMust(t, m.DelExprSubstitution(&s))
	if m.allocator.Stats().Live != base {
		t.Fatal("cold records before Close")
	}
}
func TestASTExprSubstitutionConstructorEveryReservationPrefix(t *testing.T) {
	success := false
	for fail := int64(0); fail < 80; fail++ {
		m := astTestManager(t)
		astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
		cells, payload := m.allocator.Stats().Live, m.vectorAllocator.Stats().Live
		m.BudgetField.FailAfter(fail)
		s, e := m.NewExprSubstitutionPolicy(true, true)
		m.BudgetField.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(fail, e)
			}
			if s != (ASTExprSubstitution{}) {
				t.Fatal(s)
			}
		} else {
			success = true
			astTestMust(t, m.DelExprSubstitution(&s))
		}
		if m.allocator.Stats().Live != cells || m.vectorAllocator.Stats().Live != payload {
			t.Fatal("constructor immediate owned graph", fail, m.allocator.Stats().Live, cells, m.vectorAllocator.Stats().Live, payload)
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("reservation sweep never reached success")
	}
}
func TestASTExprSubstitutionPhysicalLayoutBoundsAndBorrowedProvenance(t *testing.T) {
	for _, ty := range []reflect.Type{reflect.TypeOf(ASTExprSubstitution{}), reflect.TypeOf(ASTScopedExprSubstitution{}), reflect.TypeOf(astExprSubstTable{}), reflect.TypeOf(astExprSubstEntry{})} {
		astTestMust(t, arena.ValidateRecordType(ty))
	}
	if n, e := astExprSubstTableBytes(8, 192); e != nil || n != 192 {
		t.Fatal(n, e)
	}
	if _, e := astExprSubstTableBytes(8, 191); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	limit := uint32(uint64(^uint32(0)) / 24)
	if _, e := astExprSubstTableBytes(limit+1, uint64(^uint32(0))); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if _, e := astScopedExprSubstCapacity(0, 19); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if n, e := astScopedExprSubstCapacity(2, 24); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	m := astTestManager(t)
	s, e := m.NewExprSubstitutionPolicy(true, false)
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	if n, e := m.allocator.Size(s.record); e != nil || n != 128 {
		t.Fatal(n, e)
	}
	if n, e := m.allocator.Size(scope.record); e != nil || n != 48 {
		t.Fatal(n, e)
	}
	primary, e := m.substTable(s, 0)
	astTestMust(t, e)
	dep, e := m.substTable(s, 2)
	astTestMust(t, e)
	proof, e := m.substTable(s, 1)
	astTestMust(t, e)
	if primary.payload == dep.payload || proof.capacity != 0 {
		t.Fatal("actual distinct policy backing", primary, dep, proof)
	}
	foreign := astTestManager(t)
	if _, e := foreign.ExprSubstitutionEmpty(s); !errors.Is(e, ErrASTHandle) {
		t.Fatal(e)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionExplicitProofGuardRunsAfterPrimaryConstruction(t *testing.T) {
	m := astTestManager(t)
	base, payload := m.allocator.Stats().Live, m.vectorAllocator.Stats().Live
	s, e := m.NewExprSubstitutionPolicy(false, true)
	if ASTSourceDebug {
		if !errors.Is(e, ErrASTDomain) || s != (ASTExprSubstitution{}) {
			t.Fatal(s, e)
		}
	} else {
		astTestMust(t, e)
		astTestMust(t, m.DelExprSubstitution(&s))
	}
	if m.allocator.Stats().Live != base || m.vectorAllocator.Stats().Live != payload {
		t.Fatal("explicit ctor cleanup")
	}
	m.BudgetField.FailAfter(0)
	_, e = m.NewExprSubstitutionPolicy(false, true)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("storage preparation precedes assertion", e)
	}
}
func TestASTExprSubstitutionResetFailureKeepsLaterDependencyTableRoots(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 12000)
	dep, e := m.DependencyLeaf(value)
	astTestMust(t, e)
	astTestMust(t, m.DependencyInc(dep))
	var keys [100]ASTHandle
	for i := range keys {
		keys[i] = astDependencyExpr(t, m, uint32(12100+i))
		astTestMust(t, m.ExprSubstitutionInsert(s, keys[i], value, ASTHandle{}, dep))
	}
	for i := 0; i < 95; i++ {
		astTestMust(t, m.ExprSubstitutionErase(s, keys[i]))
	}
	live := m.vectorAllocator.Stats().Live
	m.BudgetField.FailAfter(0)
	e = m.ExprSubstitutionReset(s)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	rc, e := m.DependencyRefCount(dep)
	astTestMust(t, e)
	if rc != 6 {
		t.Fatal("later dependency table must survive primary reset failure", rc)
	}
	if m.vectorAllocator.Stats().Live != live-1 {
		t.Fatal("source delete-before-reallocate prefix")
	}
	primary, e := m.substTable(s, 0)
	astTestMust(t, e)
	if primary.payload != (arena.SmallBytesRef{}) || primary.capacity != 128 {
		t.Fatal(primary)
	}
	astTestMust(t, m.ExprSubstitutionReset(s))
	astTestMust(t, m.DelExprSubstitution(&s))
	astTestMust(t, m.DependencyDec(dep))
}
func TestASTScopedExprSubstitutionFailedTrailPushRetainsInstalledBinding(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 12200)
	value := astDependencyExpr(t, m, 12201)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	e = m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, ASTDependency{})
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	rc, e := m.GetRefCount(key)
	astTestMust(t, e)
	if rc != 3 {
		t.Fatal("map and failed unstored trail INC", rc)
	}
	a, e := m.scopedSubstState(scope)
	astTestMust(t, e)
	trail, e := m.vectorState(a.trail)
	astTestMust(t, e)
	if trail.size != 0 {
		t.Fatal(trail)
	}
	found, e := m.ExprSubstitutionContains(s, key)
	astTestMust(t, e)
	if !found {
		t.Fatal("installed untrailed binding")
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
	astTestMust(t, m.DecRef(key))
}
func TestASTScopedExprSubstitutionConsumedPopErrorResumesSuffix(t *testing.T) {
	m := astTestManager(t)
	key := astDependencyExpr(t, m, 12300)
	value := astDependencyExpr(t, m, 12301)
	dep, e := m.DependencyLeaf(value)
	astTestMust(t, e)
	s, e := m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	scope, e := m.NewScopedExprSubstitution(s)
	astTestMust(t, e)
	astTestMust(t, m.ScopedExprSubstitutionPush(scope))
	astTestMust(t, m.ScopedExprSubstitutionInsert(scope, key, value, ASTHandle{}, dep))
	m.BudgetField.FailAfter(0)
	e = m.ScopedExprSubstitutionPop(scope, 1)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	a, e := m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation>>30 != 1 || a.index != 1 || a.size != 1 {
		t.Fatal("completed erase cursor before trail resize", a)
	}
	astTestMust(t, m.ScopedExprSubstitutionPop(scope, 1))
	a, e = m.scopedSubstState(scope)
	astTestMust(t, e)
	if a.operation != 0 || a.size != 0 {
		t.Fatal(a)
	}
	if m.dependencies.live != 0 {
		t.Fatal(m.dependencies.live)
	}
	astTestMust(t, m.DelScopedExprSubstitution(&scope))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionLateEraseHashGuardReturnsRawConsumedPrefix(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	key, e := m.MkVar(12400, sort)
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 12401)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, ASTDependency{}))
	e = m.ExprSubstitutionErase(s, key)
	if !errors.Is(e, ErrASTHandle) {
		t.Fatal("source final hash would dereference freed key", e)
	}
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	if table.size != 1 {
		t.Fatal("erase was not reached", table)
	}
	for i := uint32(0); i < table.capacity; i++ {
		a, e := m.substEntry(table.payload, i)
		astTestMust(t, e)
		if a.state == 1 && a.owned != 0 {
			t.Fatal("callbacks already consumed", a)
		}
	}
	astTestMust(t, m.ExprSubstitutionReset(s))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionFinalizeRefusalPreservesOldCapacity(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 13000)
	for i := uint32(0); i < 100; i++ {
		key := astDependencyExpr(t, m, 13100+i)
		astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, ASTDependency{}))
	}
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	if table.capacity != 256 {
		t.Fatal(table)
	}
	m.BudgetField.FailAfter(0)
	e = m.ExprSubstitutionCleanup(s)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	table, e = m.substTable(s, 0)
	astTestMust(t, e)
	if table.capacity != 256 || table.payload != (arena.SmallBytesRef{}) {
		t.Fatal("source alloc64 BEFORE publishing capacity64", table)
	}
	astTestMust(t, m.ExprSubstitutionCleanup(s))
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionCollisionsWrapAndLastTombstone(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 13200)
	var keys [5]ASTHandle
	found := 0
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	for i := uint32(0); found < len(keys) && i < 1000; i++ {
		h, e := m.MkVar(13300+i, sort)
		astTestMust(t, e)
		if m.nodeAt(h.slot).hash&7 == 7 {
			keys[found] = h
			astTestPin(t, m, h)
			found++
		}
	}
	if found != len(keys) {
		t.Fatal("finite real AST hash collision search", found)
	}
	for i := 0; i < 4; i++ {
		astTestMust(t, m.ExprSubstitutionInsert(s, keys[i], value, ASTHandle{}, ASTDependency{}))
	}
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	for i := 0; i < 4; i++ {
		a, e := m.substEntry(table.payload, uint32(i+7)&7)
		astTestMust(t, e)
		if a.key != (astNodeRef{keys[i].slot, keys[i].generation}) {
			t.Fatal("source used bucket must continue, wrap", i, a)
		}
	}
	astTestMust(t, m.ExprSubstitutionErase(s, keys[1]))
	astTestMust(t, m.ExprSubstitutionErase(s, keys[2]))
	astTestMust(t, m.ExprSubstitutionInsert(s, keys[4], value, ASTHandle{}, ASTDependency{}))
	table, e = m.substTable(s, 0)
	astTestMust(t, e)
	a, e := m.substEntry(table.payload, 1)
	astTestMust(t, e)
	if a.key != (astNodeRef{keys[4].slot, keys[4].generation}) || table.deleted != 1 || table.size != 3 {
		t.Fatal("last tombstone before free wins", a, table)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
func TestASTExprSubstitutionResetConsumedErrorPreservesLaterEntries(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitutionCores(true)
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 14000)
	var deps [2]ASTDependency
	for i := range deps {
		key := astDependencyExpr(t, m, uint32(14001+i))
		deps[i], e = m.DependencyLeaf(value)
		astTestMust(t, e)
		astTestMust(t, m.ExprSubstitutionInsert(s, key, value, ASTHandle{}, deps[i]))
	}
	m.BudgetField.FailAfter(0)
	e = m.ExprSubstitutionReset(s)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if m.dependencies.live != 1 {
		t.Fatal("only completed entry cleanup before source error", m.dependencies.live)
	}
	astTestMust(t, m.ExprSubstitutionReset(s))
	astTestMust(t, m.DelExprSubstitution(&s))
	if m.dependencies.live != 0 {
		t.Fatal(m.dependencies.live)
	}
}
func TestASTExprSubstitutionDisabledPolicyDoesNotInspectUnusedForeignArguments(t *testing.T) {
	m := astTestManager(t)
	foreign := astTestManager(t)
	astTestMust(t, foreign.ToggleProofMode(ASTProofEnabled))
	proof, e := foreign.MkProofTrue()
	astTestMust(t, e)
	astTestPin(t, foreign, proof)
	dep, e := foreign.DependencyLeaf(astDependencyExpr(t, foreign, 15000))
	astTestMust(t, e)
	astTestMust(t, foreign.DependencyInc(dep))
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	key := astDependencyExpr(t, m, 15001)
	value := astDependencyExpr(t, m, 15002)
	astTestMust(t, m.ExprSubstitutionInsert(s, key, value, proof, dep))
	astTestMust(t, m.DelExprSubstitution(&s))
	astTestMust(t, foreign.DependencyDec(dep))
}
func TestASTScopedExprSubstitutionConstructorEveryReservationPrefix(t *testing.T) {
	success := false
	for fail := int64(0); fail < 80; fail++ {
		m := astTestManager(t)
		s, e := m.NewExprSubstitution()
		astTestMust(t, e)
		cells, payload := m.allocator.Stats().Live, m.vectorAllocator.Stats().Live
		m.BudgetField.FailAfter(fail)
		scope, e := m.NewScopedExprSubstitution(s)
		m.BudgetField.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || scope != (ASTScopedExprSubstitution{}) {
				t.Fatal(fail, scope, e)
			}
		} else {
			success = true
			astTestMust(t, m.DelScopedExprSubstitution(&scope))
		}
		if m.allocator.Stats().Live != cells || m.vectorAllocator.Stats().Live != payload {
			t.Fatal("scope actual constructor unwind before Close", fail)
		}
		astTestMust(t, m.DelExprSubstitution(&s))
		if success {
			break
		}
	}
	if !success {
		t.Fatal("scope sweep did not reach completion")
	}
}
func TestASTExprSubstitutionGrowthFailurePreservesEachMapPrefix(t *testing.T) {
	for _, which := range []uint32{0, 1, 2} {
		m := astTestManager(t)
		astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
		s, e := m.NewExprSubstitutionPolicy(true, true)
		astTestMust(t, e)
		value := astDependencyExpr(t, m, 16000)
		proof, e := m.MkProofAsserted(value)
		astTestMust(t, e)
		astTestPin(t, m, proof)
		dep, e := m.DependencyLeaf(value)
		astTestMust(t, e)
		astTestMust(t, m.DependencyInc(dep))
		var keys [8]ASTHandle
		for i := range keys {
			keys[i] = astDependencyExpr(t, m, uint32(16001+i))
		}
		for i := 0; i < 7; i++ {
			astTestMust(t, m.ExprSubstitutionInsert(s, keys[i], value, proof, dep))
		}
		// Expand earlier maps ahead of the tested source action. Each still uses
		// its own real table, probe history and source rehash; no algorithm stub.
		for i := uint32(0); i < which; i++ {
			table, e := m.substTable(s, i)
			astTestMust(t, e)
			astTestMust(t, m.substRehash(s, i, &table, 16))
		}
		beforeRC, e := m.GetRefCount(keys[7])
		astTestMust(t, e)
		beforeDep, e := m.DependencyRefCount(dep)
		astTestMust(t, e)
		live := m.vectorAllocator.Stats().Live
		m.BudgetField.FailAfter(0)
		e = m.ExprSubstitutionInsert(s, keys[7], value, proof, dep)
		m.BudgetField.FailAfter(-1)
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(which, e)
		}
		rc, e := m.GetRefCount(keys[7])
		astTestMust(t, e)
		want := beforeRC
		if which > 0 {
			want++
		}
		if rc != want {
			t.Fatal("primary publication before side growth", which, rc, want)
		}
		depRC, e := m.DependencyRefCount(dep)
		astTestMust(t, e)
		if depRC != beforeDep {
			t.Fatal("failed optional table insertion precedes INC", which, depRC, beforeDep)
		}
		table, e := m.substTable(s, which)
		astTestMust(t, e)
		if table.capacity != 8 || table.size != 7 {
			t.Fatal("unpublished growth failure", which, table)
		}
		if m.vectorAllocator.Stats().Live != live {
			t.Fatal("no lost temporary table before Close")
		}
		astTestMust(t, m.DelExprSubstitution(&s))
		astTestMust(t, m.DependencyDec(dep))
	}
}
func TestASTExprSubstitutionPrimaryGrowthRefusalAllowsSourceRetry(t *testing.T) {
	m := astTestManager(t)
	s, e := m.NewExprSubstitution()
	astTestMust(t, e)
	value := astDependencyExpr(t, m, 17200)
	var keys [8]ASTHandle
	for i := range keys {
		keys[i] = astDependencyExpr(t, m, uint32(17201+i))
		if i < 7 {
			astTestMust(t, m.ExprSubstitutionInsert(s, keys[i], value, ASTHandle{}, ASTDependency{}))
		}
	}
	m.BudgetField.FailAfter(0)
	e = m.ExprSubstitutionInsert(s, keys[7], value, ASTHandle{}, ASTDependency{})
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	astTestMust(t, m.ExprSubstitutionInsert(s, keys[7], value, ASTHandle{}, ASTDependency{}))
	table, e := m.substTable(s, 0)
	astTestMust(t, e)
	if table.size != 8 {
		t.Fatal("retry must retain previous bindings", table)
	}
	astTestMust(t, m.DelExprSubstitution(&s))
}
