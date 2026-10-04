// Source: src/math/polynomial/upolynomial.cpp:138-157,1259-1265; src/util/mpzzp.h:97-105. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

// Census uses named member/caller fields for numeral ownership and physical
// backing extents for charged typed storage. It never acquires caller ownership
// by walking descriptor locations.
func upolynomialSourcePhysicalAudit(t *testing.T, q *RationalManager, u *UPolynomialManager, callers ...UPolynomial) {
	t.Helper()
	roots := []*Int{&q.nTmp, &q.addTmp1, &q.addTmp2, &q.addMulTmp.num, &q.addMulTmp.den, &q.ltTmp1.num, &q.ltTmp1.den, &q.ltTmp2.num, &q.ltTmp2.den, &u.modular.p, &u.modular.lower, &u.modular.upper, &u.modular.invTmp1, &u.modular.invTmp2, &u.modular.invTmp3, &u.modular.divTmp}
	states := []*upolyVectorState{&u.db, &u.dbab1, &u.dbab2, &u.tr, &u.push, &u.basic, &u.div1, &u.div2, &u.exactDiv, &u.gcd1, &u.gcd2, &u.cra, &u.mgcd[0], &u.mgcd[1], &u.mgcd[2], &u.mgcd[3], &u.mgcd[4], &u.mgcd[5], &u.sqf1, &u.sqf2, &u.pw}
	for _, p := range callers {
		if p.origin != u.origin || p.slot == 0 {
			t.Fatal("caller provenance")
		}
		d := &u.pages[(p.slot-1)/32].rows[(p.slot-1)%32]
		if !d.live || d.generation != p.generation {
			t.Fatal("caller liveness")
		}
		states = append(states, &d.state)
	}
	for _, st := range states {
		if st.backing == 0 {
			continue
		}
		a := u.values(st)
		for i := uint32(0); i < st.initialized; i++ {
			roots = append(roots, &a[i])
		}
	}
	integerPhysicalAudit(t, q.z, roots...)
	n := q.headerBytes + integerActualOwnedBytes(q.z) + u.header + u.modular.header + u.store.header
	n += uint64(cap(u.pages))*uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()) + uint64(len(u.pages))*uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()) + uint64(cap(u.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size()) + uint64(cap(u.store.backings))*uint64(reflect.TypeOf(upolyBacking{}).Size())
	for _, b := range u.store.backings {
		if b.live {
			n += uint64(cap(b.values)) * uint64(reflect.TypeOf(Int{}).Size())
		}
	}
	if n != q.budget.Used() {
		t.Fatal("actual cold and physical graph", n, q.budget.Used())
	}
}

// Independent lifecycle oracle: source destruction releases private vectors,
// while a separately owned polynomial and charged adapter metadata survive.
func TestUPolynomialSourceFinalizePrivateMembersAndCaller(t *testing.T) {
	q, b := rationalManager(t, false)
	visits := uint32(0)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { visits++; return visits < 1000 }, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	p, e := u.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, u.SetInts(p, []Int{NewInt(5), NewInt(2)}))
	members := [21]*upolyVectorState{&u.db, &u.dbab1, &u.dbab2, &u.tr, &u.push, &u.basic, &u.div1, &u.div2, &u.exactDiv, &u.gcd1, &u.gcd2, &u.cra, &u.mgcd[0], &u.mgcd[1], &u.mgcd[2], &u.mgcd[3], &u.mgcd[4], &u.mgcd[5], &u.sqf1, &u.sqf2, &u.pw}
	for i, s := range members {
		serviceMust(t, u.reserve(s, 1))
		serviceMust(t, q.z.SetUint64(&u.values(s)[0], uint64(1)<<40|uint64(i+1)))
	}
	upolynomialSourcePhysicalAudit(t, q, u, p)
	if e = u.Close(); !errors.Is(e, ErrUPolynomialLive) || u.closing {
		t.Fatal("owning caller preguard", e)
	}
	before := b.Used()
	serviceMust(t, u.SourceFinalize())
	upolynomialSourcePhysicalAudit(t, q, u, p)
	for _, s := range members {
		if *s != (upolyVectorState{}) {
			t.Fatal("source private backing retained", *s)
		}
	}
	if !u.sourceFinalized || u.closed || u.header == 0 || b.Used() >= before {
		t.Fatal("source/owning lifetime conflated")
	}
	if _, e = u.Size(p); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary access after source destruction", e)
	}
	// This is explicit owning cleanup of the caller field, not another source
	// destructor invocation or discovery of roots through the descriptor table.
	serviceMust(t, u.CleanupPolynomial(&p))
	serviceMust(t, u.Close())
}

func TestUPolynomialSourceFinalizeStoppedPrivateGuard(t *testing.T) {
	q, _ := rationalManager(t, false)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	serviceMust(t, u.reserve(&u.db, 2))
	serviceMust(t, q.z.SetUint64(&u.values(&u.db)[0], uint64(1)<<40))
	u.values(&u.db)[1] = Int{cell: q.z.failedRef}
	e = u.SourceFinalize()
	if !errors.Is(e, arena.ErrInvalidSlot) || !u.sourceGuard || u.sourceIndex != 1 || u.values(&u.db)[0].cell != 0 || u.db.backing == 0 {
		t.Fatal("actual ascending completed prefix", e, u.sourceIndex, u.db)
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	used := q.z.budget.Used()
	if e2 := u.SourceFinalize(); !errors.Is(e2, arena.ErrInvalidSlot) || q.z.budget.Used() != used {
		t.Fatal("guard retry changed source prefix", e2)
	}
	serviceMust(t, u.Close())
}

func TestUPolynomialSourceFinalizeDerivedBackingBeforeCoreBody(t *testing.T) {
	q, _ := rationalManager(t, false)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	derived := [5]*upolyVectorState{&u.db, &u.dbab1, &u.dbab2, &u.tr, &u.push}
	for _, s := range derived {
		serviceMust(t, u.reserve(s, 1))
		serviceMust(t, q.z.SetUint64(&u.values(s)[0], 1<<40))
	}
	serviceMust(t, u.reserve(&u.basic, 1))
	u.values(&u.basic)[0] = Int{cell: q.z.failedRef}
	serviceMust(t, q.z.SetUint64(&u.modular.p, 1<<45))
	modulus := u.modular.p.cell
	upolynomialSourcePhysicalAudit(t, q, u)
	if e = u.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) || u.sourceStage != 2 || u.sourceIndex != 0 {
		t.Fatal("reached base destructor body", e, u.sourceStage, u.sourceIndex)
	}
	for _, s := range derived {
		if *s != (upolyVectorState{}) {
			t.Fatal("derived backing must die before base body", *s)
		}
	}
	if u.basic.backing == 0 || u.modular.p.cell != modulus || !validNumericRef(q.z.cells, modulus) {
		t.Fatal("unreached member destruction")
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	serviceMust(t, u.Close())
	if u.sourceFinalized || !u.sourceGuard || !errors.Is(u.sourceCause, arena.ErrInvalidSlot) {
		t.Fatal("owning abort rewrote stopped source history")
	}
}

func TestUPolynomialSourceFinalizeOwnerGuards(t *testing.T) {
	var nilOwner *UPolynomialManager
	if e := nilOwner.SourceFinalize(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	q, _ := rationalManager(t, false)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	copyOwner := *u
	if e = copyOwner.SourceFinalize(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	serviceMust(t, u.Close())
	if e = u.SourceFinalize(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("owning-closed source entry", e)
	}
}

// The public realloc recipe deliberately reaches source signed-overflow UB.
// It exercises the guarded Go postphysical boundary, not valid exhaustion or
// an invented deallocation failure from an allocation limit.
func TestUPolynomialSourceFinalizeDoesNotOwnReleasedLeafRetry(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, r, u.Close)
		contextTLSFinish(t, r, q.CloseChecked)
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("owning physical graph", b.Used())
		}
	}()
	serviceMust(t, u.reserve(&u.db, 1))
	digits := make([]uint32, 40000)
	digits[0] = 17
	digits[len(digits)-1] = 1
	serviceMust(t, q.z.SetLimbs(&u.values(&u.db)[0], digits))
	old := u.values(&u.db)[0].cell
	contextTLSNearMinimum(t, r)
	before := r.Stats()
	e = u.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || u.sourceIndex != 0 || u.sourceStage != 0 || u.sourceCause != e {
		t.Fatal("actual stopped leaf", e, u.sourceStage, u.sourceIndex)
	}
	c := q.z.cell(old)
	if !c.released || !c.cleanupPending || u.values(&u.db)[0].cell != old || r.Stats().LiveBlocks != before.LiveBlocks-1 {
		t.Fatal("completed physical prefix", c, r.Stats(), before)
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	used, blocks := b.Used(), r.Stats()
	if again := u.SourceFinalize(); again != e {
		t.Errorf("literal source retry must retain exact stopped cause, got %v want %v", again, e)
	}
	if b.Used() != used || r.Stats() != blocks || u.values(&u.db)[0].cell != old {
		t.Error("source retry acknowledged owning release")
	}
}

// A physically zero Go tail is not a constructed source mpz. The guard is
// reached only after deleting the genuinely constructed earlier element.
func TestUPolynomialSourceFinalizeUnconstructedTailPrefix(t *testing.T) {
	q, _ := rationalManager(t, false)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	serviceMust(t, u.reserve(&u.db, 2))
	serviceMust(t, q.z.SetUint64(&u.values(&u.db)[0], 1<<40))
	u.db.initialized = 1
	e = u.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || u.sourceIndex != 1 || u.values(&u.db)[0].cell != 0 || u.db.size != 2 || u.db.backing == 0 {
		t.Fatal("actual constructed deletion before endangered tail", e, u.sourceIndex, u.db, u.values(&u.db)[0])
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	serviceMust(t, u.Close())
	if u.sourceFinalized || !errors.Is(u.sourceCause, arena.ErrBounds) || u.sourceIndex != 1 {
		t.Fatal("owning abort changed source prefix")
	}
}

func TestUPolynomialSourceFinalizeFirstEntryConsumedLeafGuard(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	defer func() {
		contextTLSFinish(t, r, u.Close)
		contextTLSFinish(t, r, q.CloseChecked)
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("owning physical graph", b.Used())
		}
	}()
	serviceMust(t, u.reserve(&u.db, 1))
	digits := make([]uint32, 40000)
	digits[0] = 17
	digits[len(digits)-1] = 1
	serviceMust(t, q.z.SetLimbs(&u.values(&u.db)[0], digits))
	old := u.values(&u.db)[0].cell
	contextTLSNearMinimum(t, r)
	before := r.Stats()
	e = q.z.Del(&u.values(&u.db)[0])
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual owning leaf stop", e)
	}
	c := q.z.cell(old)
	if !c.released || !c.cleanupPending || r.Stats().LiveBlocks != before.LiveBlocks-1 {
		t.Fatal("actual completed physical release")
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	used, blocks := b.Used(), r.Stats()
	e = u.SourceFinalize()
	if !errors.Is(e, arena.ErrInvalidSlot) || !u.sourceGuard || u.sourceStage != 0 || u.sourceIndex != 0 || u.values(&u.db)[0].cell != old {
		t.Errorf("source consumed pointer guard: %v", e)
	}
	if b.Used() != used || r.Stats() != blocks {
		t.Error("first source entry acknowledged owning release")
	}
}

// The setter records a genuine source nullptr after fresh Copy allocation
// refusal. This endpoint validates the shared paired-state prerequisite; it is
// not a manufactured failed-token guard or a new source allocation event.
func TestUPolynomialSourceFinalizeFreshNullMember(t *testing.T) {
	q, b := rationalPublicationDirectOwner(t, false)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.z, Checkpoint: func() bool { panic("unreached") }, Cooperation: func() error { panic("unreached") }})
	serviceMust(t, e)
	serviceMust(t, u.reserve(&u.db, 1))
	a := &u.values(&u.db)[0]
	b.FailAfter(0)
	e = q.z.Copy(a, &q.z.two64)
	if !numericPhysicalReservationRefusal(e) || a.cell != q.z.failedNullRef {
		t.Fatal("actual fresh Copy source-null prefix", e, *a)
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	before := q.z.memory.Stats()
	beforeCount, e := q.z.memory.AllocationCount()
	serviceMust(t, e)
	b.FailAfter(-1)
	serviceMust(t, u.SourceFinalize())
	after := q.z.memory.Stats()
	afterCount, e := q.z.memory.AllocationCount()
	serviceMust(t, e)
	if !u.sourceFinalized || u.sourceGuard || u.sourceCause != nil || beforeCount != afterCount || before.LiveBlocks != after.LiveBlocks {
		t.Fatal("NULL source Del fabricated raw events", before, after)
	}
	upolynomialSourcePhysicalAudit(t, q, u)
	serviceMust(t, u.Close())
}
