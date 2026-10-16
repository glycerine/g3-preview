// Source: src/util/scoped_numeral.h, src/util/inf_eps_rational.h.
package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// This is an explicitly malformed Go physical layout, not a valid source
// allocation failure. The 32-bit allocator refuses the actual destructor read.
func TestLexicalInfDestructorRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	z := s.Manager.Z
	v := s.infEmpty()
	if e = s.Manager.SetString(&v.First, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.Manager.SetString(&v.Second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	ref := v.Second.Num.Cell
	cell := z.Cell(ref)
	zcell := cell
	zcell.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, zcell)
	defer func() {
		if ValidNumericRef(z.Cells, ref) {
			c := z.Cell(ref)
			c.Backing.Requested = cell.Backing.Requested
			z.Store(ref, c)
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
	if v.ownerID != s.ownerID || v.Second.Num.Cell != ref || v.First.Num.Cell == 0 {
		t.Fatal("refused caller destructor lost ownership")
	}
	serviceAuditRationalRoots(t, s, &v.First, &v.Second)
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
			r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
			if e != nil {
				t.Fatal(e)
			}
			cfg := serviceConfig(b)
			cfg.Integers.Memory = r
			cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
			s, e := NewRationalService(cfg)
			if e != nil {
				t.Fatal(e)
			}
			z := s.Manager.Z
			v := s.infEmpty()
			if e = s.Manager.SetString(&v.First, "18446744073709551617"); e != nil {
				t.Fatal(e)
			}
			if e = s.Manager.SetString(&v.Second, "18446744073709551619"); e != nil {
				t.Fatal(e)
			}
			firstRef, secondRef := v.First.Num.Cell, v.Second.Num.Cell
			ref := secondRef
			if first {
				ref = firstRef
			}
			original := z.Cell(ref)
			bad := original
			bad.Backing.Requested = uint64(math.MaxUint32) + 1
			z.Store(ref, bad)
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
			if first && ValidNumericRef(z.Cells, secondRef) {
				t.Fatal("completed SECOND destruction replayed or retained")
			}
			if !first && !ValidNumericRef(z.Cells, firstRef) {
				t.Fatal("FIRST destroyed past refused SECOND")
			}
			named := []*Rational{}
			for i := s.Lexical.cursor; i < s.Lexical.Count; i++ {
				named = append(named, &s.Lexical.Roots[i])
			}
			serviceAuditRationalRoots(t, s, named...)
			// A pending owning expression must not stop independent scalar queries.
			if got, e := s.IsZero(&OwnedRational{Raw: NewRational(), ownerID: s.ownerID}); e != nil || !got {
				t.Fatal(got, e)
			}
			if _, e = s.One(); !errors.Is(e, ErrRationalLexicalPending) {
				t.Fatal("new owning local admitted", e)
			}
			if e = s.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) || s.sourceFinalizing {
				t.Fatal("pending lexical scope changed finalization prefix", e)
			}
			c := z.Cell(ref)
			c.Backing.Requested = original.Backing.Requested
			z.Store(ref, c)
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
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	v := s.infEmpty()
	z := s.Manager.Z
	if e = s.Manager.SetString(&v.First, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.Manager.SetString(&v.Second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	first, partial := v.First.Num.Cell, v.Second.Num.Cell
	original := z.Cell(first)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(first, bad)
	// FIRST completed its copy constructor; SECOND allocated a partial NUM and
	// then failed. Source destroys FIRST; partial SECOND is a later owning abort.
	s.abortInfClone(&v, 1)
	st, _ := s.LexicalState()
	if st.Roots != 2 || st.Completed != 1 || st.Aborts != 1 {
		t.Fatal(st)
	}
	if s.Lexical.Roots[0].Num.Cell != first || s.Lexical.Roots[1].Num.Cell != partial {
		t.Fatal("failed member destructor incorrectly ran before completed FIRST unwind")
	}
	serviceAuditRationalRoots(t, s, &s.Lexical.Roots[0], &s.Lexical.Roots[1])
	c := z.Cell(first)
	c.Backing.Requested = original.Backing.Requested
	z.Store(first, c)
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

// This bounded fixture exercises lexical cleanup of four scalar locals, two
// members of a named value and two returned-copy members. It does not claim
// a retained source path for the removed approximation division functions.
// It deliberately corrupts a physical layout after construction; no allocator
// callback or budget refusal is presented as a source destructor exception.
func lexicalEightRootFixture(s *RationalService, corrupt bool) (out OwnedInfRational, err error, broken NumericCellRef, requested uint64) {
	defer s.finishLexicalInf(&out, &err)
	var locals [4]Rational
	for i := range locals {
		locals[i] = NewRational()
		if err = s.Manager.SetString(&locals[i], "18446744073709551617"); err != nil {
			return
		}
		defer s.destroyLexicalRational(&locals[i])
	}
	value := s.infEmpty()
	if err = s.Manager.SetString(&value.First, "18446744073709551619"); err != nil {
		return
	}
	if err = s.Manager.SetString(&value.Second, "18446744073709551621"); err != nil {
		return
	}
	defer s.infDestroy(&value)
	if out, err = s.infClone(&value); err != nil {
		return
	}
	if corrupt {
		broken = value.Second.Num.Cell
		c := s.Manager.Z.Cell(broken)
		requested = c.Backing.Requested
		c.Backing.Requested = uint64(math.MaxUint32) + 1
		s.Manager.Z.Store(broken, c)
	}
	return
}
func TestLexicalUnpublishedResultAndEightChargedRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
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
		roots[i] = &s.Lexical.Roots[i]
	}
	serviceAuditRationalRoots(t, s, roots...)
	c := s.Manager.Z.Cell(ref)
	c.Backing.Requested = requested
	s.Manager.Z.Store(ref, c)
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
	// A subsequent complete expression returns precisely one owning token.
	out, e, _, _ = lexicalEightRootFixture(s, false)
	if e != nil || out.ownerID != s.ownerID {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s, &out.First, &out.Second)
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
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
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
	z := s.Manager.Z
	src := s.infEmpty()
	if e := s.Manager.SetString(&src.First, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.Manager.SetString(&src.Second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	failed := NewRational()
	b.FailAfter(0)
	e = z.Copy(&failed.Num, &src.First.Num)
	if !numericPhysicalReservationRefusal(e) || failed.Num.Cell != z.FailedNullRef {
		t.Fatal("real failed-copy token not published", e)
	}
	serviceAuditRationalRoots(t, s, &src.First, &src.Second, &failed)
	b.FailAfter(-1)
	if e = z.CleanupPendingAllocation(); e != nil {
		t.Fatal(e)
	}
	src.Second.Den = failed.Num
	failed.Num = Int{} // actual failed-copy ownership transfer
	out, e := s.InfClone(&src)
	if !errors.Is(e, arena.ErrInvalidSlot) || out.ownerID != 0 {
		t.Fatal(e, out.ownerID)
	}
	// The real failed-copy token is nonlive, but still requires explicit owning
	// abort. It is never treated as a completed source numeral destructor.
	if e = z.Del(&src.Second.Den); e != nil {
		t.Fatal(e)
	}
	src.Second.Den = NewInt(1)
	state, _ := s.LexicalState()
	if state.Pending || state.Roots != 0 {
		t.Fatal("normal failed constructor abort leaked", state)
	}
	serviceAuditRationalRoots(t, s, &src.First, &src.Second)
	if e = s.InfDel(&src); e != nil {
		t.Fatal(e)
	}
}

func TestLexicalCallerDestructorLateMemberRead(t *testing.T) {
	s, _ := serviceNew(t)
	z := s.Manager.Z
	v := s.infEmpty()
	if e := s.Manager.SetString(&v.First, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e := s.Manager.SetString(&v.Second, "18446744073709551619"); e != nil {
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
	v.Second.Den = stale
	e := s.InfDel(&v)
	if !errors.Is(e, arena.ErrInvalidSlot) || v.Second.Num.Cell != 0 || v.First.Num.Cell == 0 || v.ownerID != s.ownerID {
		t.Fatal("source NUM destruction must precede endangered DEN read", e, v.Second.Num.Cell)
	}
	// The intentionally stale test field has no owning root. Only that scalar is
	// reset; source-completed NUM destruction must not be replayed by retry.
	v.Second.Den = NewInt(1)
	serviceAuditRationalRoots(t, s, &v.First, &v.Second)
	if e = s.InfDel(&v); e != nil {
		t.Fatal(e)
	}
}

func TestLexicalOriginalFailureAndCleanupCause(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	v := s.infEmpty()
	z := s.Manager.Z
	if e = s.Manager.SetString(&v.First, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	if e = s.Manager.SetString(&v.Second, "18446744073709551619"); e != nil {
		t.Fatal(e)
	}
	ref := v.First.Num.Cell
	c := z.Cell(ref)
	requested := c.Backing.Requested
	c.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, c)
	// An actual guarded read causes construction failure; the intentionally
	// malformed completed FIRST destructor independently refuses its release.
	original := z.Copy(&v.Second.Den, nil)
	if !errors.Is(original, arena.ErrInvalidSlot) {
		t.Fatal(original)
	}
	s.abortInfClone(&v, 1)
	s.finishLexicalError(&original)
	state, _ := s.LexicalState()
	if !errors.Is(original, arena.ErrInvalidSlot) || !errors.Is(state.OriginalCause, arena.ErrInvalidSlot) || !errors.Is(state.Cause, arena.ErrBounds) {
		t.Fatal(original, state)
	}
	serviceAuditRationalRoots(t, s, &s.Lexical.Roots[0], &s.Lexical.Roots[1])
	c = z.Cell(ref)
	c.Backing.Requested = requested
	z.Store(ref, c)
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
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(b).Integers
	cfg.Memory = r
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	m, e := NewRationalManager(RationalConfig{Integers: cfg, Synchronized: true})
	if e != nil {
		t.Fatal(e)
	}
	work := NewRational()
	if e = m.SetString(&work, "18446744073709551617"); e != nil {
		t.Fatal(e)
	}
	ref := work.Num.Cell
	c := m.Z.Cell(ref)
	requested := c.Backing.Requested
	c.Backing.Requested = uint64(math.MaxUint32) + 1
	m.Z.Store(ref, c)
	original := m.Z.Copy(&work.Den, nil)
	m.finishLexicalCopy(&work, &original)
	state, e := m.LexicalState()
	if e != nil || state.Roots != 1 || !state.Pending || !errors.Is(state.OriginalCause, arena.ErrInvalidSlot) || !errors.Is(state.Cause, arena.ErrBounds) {
		t.Fatal(state, e)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&m.lexical.roots[0].Num, &m.lexical.roots[0].Den})
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) || m.SourceFinalizing {
		t.Fatal(e)
	}
	c = m.Z.Cell(ref)
	c.Backing.Requested = requested
	m.Z.Store(ref, c)
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
