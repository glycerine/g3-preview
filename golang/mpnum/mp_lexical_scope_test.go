// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

// This is an explicitly malformed Go physical layout, not a valid source
// allocation failure. The 32-bit allocator refuses the actual destructor read.
func TestLexicalInfDestructorRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	z := s.manager.z
	v := s.infEmpty()
	if e = s.manager.SetString(&v.first, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.manager.SetString(&v.second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	ref := v.second.num.cell
	cell := z.cell(ref)
	zcell := cell
	zcell.backing.requested = uint64(math.MaxUint32) + 1
	z.store(ref, zcell)
	defer func() {
		if validNumericRef(z.cells, ref) {
			c := z.cell(ref)
			c.backing.requested = cell.backing.requested
			z.store(ref, c)
		}
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if e := r.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("retained bytes", b.Used())
		}
	}()
	e = s.InfDel(&v)
	if v.ownerID != s.ownerID || v.second.num.cell != ref || v.first.num.cell == 0 {
		t.Fatal("refused caller destructor lost ownership")
	}
	serviceAuditRationalRoots(t, s, &v.first, &v.second)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("actual refused destructor must be observed, got %v", e)
	}
}

func TestLexicalPendingOwnerAPI(t *testing.T) {
	s, _ := serviceNew(t)
	state, e := s.LexicalState()
	if e != nil || state.Roots != 0 || state.Pending {
		t.Fatal(state, e)
	}
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
}

// Only the deliberately corrupted requested-byte metadata is repaired. No
// arithmetic fields, allocator registry or callback counters are manufactured.
func TestLexicalReverseMemberProgressAndExplicitRecovery(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(map[bool]string{false: "second", true: "first"}[first], func(t *testing.T) {
			b := arena.NewBudget(1 << 24)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
			if e != nil {
				t.Fatal(e)
			}
			cfg := serviceConfig(b)
			cfg.Integers.Memory = r
			cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
			s, e := NewRationalService(cfg)
			if e != nil {
				t.Fatal(e)
			}
			z := s.manager.z
			v := s.infEmpty()
			if e = s.manager.SetString(&v.first, "18446744073709551617"); e != nil {
				t.Fatal(e)
			}
			if e = s.manager.SetString(&v.second, "18446744073709551619"); e != nil {
				t.Fatal(e)
			}
			firstRef, secondRef := v.first.num.cell, v.second.num.cell
			ref := secondRef
			if first {
				ref = firstRef
			}
			original := z.cell(ref)
			bad := original
			bad.backing.requested = uint64(math.MaxUint32) + 1
			z.store(ref, bad)
			if e = s.infDestroy(&v); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(e)
			}
			if v.ownerID != 0 {
				t.Fatal("completed lexical token not transferred")
			}
			st, _ := s.LexicalState()
			want := uint8(2)
			if first {
				want = 1
			}
			if !st.Pending || st.Roots != want || st.Completed != want || st.Aborts != 0 {
				t.Fatal(st)
			}
			if first && validNumericRef(z.cells, secondRef) {
				t.Fatal("completed SECOND destruction replayed or retained")
			}
			if !first && !validNumericRef(z.cells, firstRef) {
				t.Fatal("FIRST destroyed past refused SECOND")
			}
			named := []*Rational{}
			for i := s.lexical.cursor; i < s.lexical.count; i++ {
				named = append(named, &s.lexical.roots[i])
			}
			serviceAuditRationalRoots(t, s, named...)
			// A pending owning expression must not stop independent scalar queries.
			if got, e := s.IsZero(&OwnedRational{raw: NewRational(), ownerID: s.ownerID}); e != nil || !got {
				t.Fatal(got, e)
			}
			if _, e = s.One(); !errors.Is(e, ErrRationalLexicalPending) {
				t.Fatal("new owning local admitted", e)
			}
			if e = s.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) || s.sourceFinalizing {
				t.Fatal("pending lexical scope changed finalization prefix", e)
			}
			c := z.cell(ref)
			c.backing.requested = original.backing.requested
			z.store(ref, c)
			if e = s.CleanupLexical(); e != nil {
				t.Fatal(e)
			}
			st, _ = s.LexicalState()
			if st.Pending || st.Roots != 0 {
				t.Fatal(st)
			}
			serviceAuditRationalRoots(t, s)
			if e = s.CleanupLexical(); e != nil {
				t.Fatal("idempotent cleanup", e)
			}
			if e = s.CloseChecked(); e != nil {
				t.Fatal(e)
			}
			if e = r.Close(); e != nil {
				t.Fatal(e)
			}
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		})
	}
}

func TestLexicalFailedConstructorCompletedMemberOrder(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	v := s.infEmpty()
	z := s.manager.z
	if e = s.manager.SetString(&v.first, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.manager.SetString(&v.second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	first, partial := v.first.num.cell, v.second.num.cell
	original := z.cell(first)
	bad := original
	bad.backing.requested = uint64(math.MaxUint32) + 1
	z.store(first, bad)
	// FIRST completed its copy constructor; SECOND allocated a partial NUM and
	// then failed. Source destroys FIRST; partial SECOND is a later owning abort.
	s.abortInfClone(&v, 1)
	st, _ := s.LexicalState()
	if st.Roots != 2 || st.Completed != 1 || st.Aborts != 1 {
		t.Fatal(st)
	}
	if s.lexical.roots[0].num.cell != first || s.lexical.roots[1].num.cell != partial {
		t.Fatal("failed member destructor incorrectly ran before completed FIRST unwind")
	}
	serviceAuditRationalRoots(t, s, &s.lexical.roots[0], &s.lexical.roots[1])
	c := z.cell(first)
	c.backing.requested = original.backing.requested
	z.store(first, c)
	// CloseChecked itself drains genuine named roots, before child finalization.
	if e = s.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// This bounded fixture executes the exact cleanup shape of infDiv: four
// scalar locals, two members of a named value and two returned-copy members.
// It deliberately corrupts a physical layout after construction; no allocator
// callback or budget refusal is presented as a source destructor exception.
func lexicalEightRootFixture(s *RationalService, corrupt bool) (out OwnedInfRational, err error, broken numericCellRef, requested uint64) {
	defer s.finishLexicalInf(&out, &err)
	var locals [4]Rational
	for i := range locals {
		locals[i] = NewRational()
		if err = s.manager.SetString(&locals[i], "18446744073709551617"); err != nil {
			return
		}
		defer s.destroyLexicalRational(&locals[i])
	}
	value := s.infEmpty()
	if err = s.manager.SetString(&value.first, "18446744073709551619"); err != nil {
		return
	}
	if err = s.manager.SetString(&value.second, "18446744073709551621"); err != nil {
		return
	}
	defer s.infDestroy(&value)
	if out, err = s.infClone(&value); err != nil {
		return
	}
	if corrupt {
		broken = value.second.num.cell
		c := s.manager.z.cell(broken)
		requested = c.backing.requested
		c.backing.requested = uint64(math.MaxUint32) + 1
		s.manager.z.store(broken, c)
	}
	return
}
func TestLexicalUnpublishedResultAndEightChargedRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	out, e, ref, requested := lexicalEightRootFixture(s, true)
	if !errors.Is(e, arena.ErrBounds) || out.ownerID != 0 {
		t.Fatal("success escaped refused lexical destruction", e, out.ownerID)
	}
	state, _ := s.LexicalState()
	if state.Roots != 8 || state.Completed != 8 || state.Aborts != 0 || !state.Pending {
		t.Fatal(state)
	}
	roots := make([]*Rational, 8)
	for i := range roots {
		roots[i] = &s.lexical.roots[i]
	}
	serviceAuditRationalRoots(t, s, roots...)
	c := s.manager.z.cell(ref)
	c.backing.requested = requested
	s.manager.z.store(ref, c)
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
	// A subsequent complete expression returns precisely one owning token.
	out, e, _, _ = lexicalEightRootFixture(s, false)
	if e != nil || out.ownerID != s.ownerID {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s, &out.first, &out.second)
	if e = s.InfDel(&out); e != nil {
		t.Fatal(e)
	}
	if e = s.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestLexicalCloneFailedMemberReadUnwind(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error(b.Used())
		}
	}()
	z := s.manager.z
	src := s.infEmpty()
	if e := s.manager.SetString(&src.first, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.manager.SetString(&src.second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	failed := NewRational()
	b.FailAfter(0)
	e = z.Copy(&failed.num, &src.first.num)
	if !numericPhysicalReservationRefusal(e) || failed.num.cell != z.failedRef {
		t.Fatal("real failed-copy token not published", e)
	}
	serviceAuditRationalRoots(t, s, &src.first, &src.second, &failed)
	b.FailAfter(-1)
	if e = z.CleanupPendingAllocation(); e != nil {
		t.Fatal(e)
	}
	src.second.den = failed.num
	failed.num = Int{} // actual failed-copy ownership transfer
	out, e := s.InfClone(&src)
	if !errors.Is(e, arena.ErrInvalidSlot) || out.ownerID != 0 {
		t.Fatal(e, out.ownerID)
	}
	// The real failed-copy token is nonlive, but still requires explicit owning
	// abort. It is never treated as a completed source numeral destructor.
	if e = z.Del(&src.second.den); e != nil {
		t.Fatal(e)
	}
	src.second.den = NewInt(1)
	state, _ := s.LexicalState()
	if state.Pending || state.Roots != 0 {
		t.Fatal("normal failed constructor abort leaked", state)
	}
	serviceAuditRationalRoots(t, s, &src.first, &src.second)
	if e = s.InfDel(&src); e != nil {
		t.Fatal(e)
	}
}

func TestLexicalCallerDestructorLateMemberRead(t *testing.T) {
	s, _ := serviceNew(t)
	z := s.manager.z
	v := s.infEmpty()
	if e := s.manager.SetString(&v.first, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.manager.SetString(&v.second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	dead := Int{}
	if e := z.SetString(&dead, "18446744073709551621"); e != nil {
		t.Fatal(e)
	}
	stale := dead
	if e := z.Del(&dead); e != nil {
		t.Fatal(e)
	}
	v.second.den = stale
	e := s.InfDel(&v)
	if !errors.Is(e, arena.ErrInvalidSlot) || v.second.num.cell != 0 || v.first.num.cell == 0 || v.ownerID != s.ownerID {
		t.Fatal("source NUM destruction must precede endangered DEN read", e, v.second.num.cell)
	}
	// The intentionally stale test field has no owning root. Only that scalar is
	// reset; source-completed NUM destruction must not be replayed by retry.
	v.second.den = NewInt(1)
	serviceAuditRationalRoots(t, s, &v.first, &v.second)
	if e = s.InfDel(&v); e != nil {
		t.Fatal(e)
	}
}

func TestLexicalOriginalFailureAndCleanupCause(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	v := s.infEmpty()
	z := s.manager.z
	if e = s.manager.SetString(&v.first, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.manager.SetString(&v.second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	ref := v.first.num.cell
	c := z.cell(ref)
	requested := c.backing.requested
	c.backing.requested = uint64(math.MaxUint32) + 1
	z.store(ref, c)
	// An actual guarded read causes construction failure; the intentionally
	// malformed completed FIRST destructor independently refuses its release.
	original := z.Copy(&v.second.den, nil)
	if !errors.Is(original, arena.ErrInvalidSlot) {
		t.Fatal(original)
	}
	s.abortInfClone(&v, 1)
	s.finishLexicalError(&original)
	state, _ := s.LexicalState()
	if !errors.Is(original, arena.ErrInvalidSlot) || !errors.Is(state.OriginalCause, arena.ErrInvalidSlot) || !errors.Is(state.Cause, arena.ErrBounds) {
		t.Fatal(original, state)
	}
	serviceAuditRationalRoots(t, s, &s.lexical.roots[0], &s.lexical.roots[1])
	c = z.cell(ref)
	c.backing.requested = requested
	z.store(ref, c)
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	if e = s.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestLexicalRationalManagerOneRootContinuation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b).Integers
	cfg.Memory = r
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 32}
	m, e := NewRationalManager(RationalConfig{Integers: cfg, Synchronized: true})
	if e != nil {
		t.Fatal(e)
	}
	work := NewRational()
	if e = m.SetString(&work, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	ref := work.num.cell
	c := m.z.cell(ref)
	requested := c.backing.requested
	c.backing.requested = uint64(math.MaxUint32) + 1
	m.z.store(ref, c)
	original := m.z.Copy(&work.den, nil)
	m.finishLexicalCopy(&work, &original)
	state, e := m.LexicalState()
	if e != nil || state.Roots != 1 || !state.Pending || !errors.Is(state.OriginalCause, arena.ErrInvalidSlot) || !errors.Is(state.Cause, arena.ErrBounds) {
		t.Fatal(state, e)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&m.lexical.roots[0].num, &m.lexical.roots[0].den})
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) || m.sourceFinalizing {
		t.Fatal(e)
	}
	c = m.z.cell(ref)
	c.backing.requested = requested
	m.z.store(ref, c)
	if e = m.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
