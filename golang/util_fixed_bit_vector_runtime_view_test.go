// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"testing"
)

func TestFixedBitVectorRuntimeBorrowedForeignDestinationAndQueries(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	d, dr, db := fixedRuntimeFixture(t, 65)
	s, sr, sb := fixedRuntimeFixture(t, 65)
	dst, e := d.Allocate0()
	heapMust(t, e)
	src, e := s.Allocate1()
	heapMust(t, e)
	dv, e := d.Borrow(dst)
	heapMust(t, e)
	sv, e := s.Borrow(src)
	heapMust(t, e)
	heapMust(t, m.CopyViews(dv, sv))
	eq, e := m.EqualViews(dv, sv)
	heapMust(t, e)
	if !eq {
		t.Fatal("foreign destination")
	}
	has, e := m.ContainsViews(dv, sv)
	heapMust(t, e)
	if !has {
		t.Fatal("contains")
	}
	heapMust(t, m.UnsetView(dv, 4))
	heapMust(t, m.SetView(dv, 4))
	heapMust(t, m.SetBoolView(dv, 5, false))
	heapMust(t, m.UnionViews(dv, sv))
	heapMust(t, m.IntersectViews(dv, sv))
	heapMust(t, m.SetRangeViews(dv, sv, 40, 3))
	last, e := m.LastWordView(dv)
	heapMust(t, e)
	if last != 1 {
		t.Fatal(last)
	}
	h, e := m.HashView(sv)
	heapMust(t, e)
	if h != utilhash.String(string(bytes.Repeat([]byte{255}, 8)), 65) {
		t.Fatal(h)
	}
	var out bytes.Buffer
	heapMust(t, m.DisplayView(&out, dv))
	if out.String() != string(bytes.Repeat([]byte{'1'}, 65)) {
		t.Fatal(out.String())
	}
	heapMust(t, m.NegView(dv))
	heapMust(t, m.Fill0View(dv))
	heapMust(t, m.Fill1View(dv))
	heapMust(t, m.ResetView(dv))
	fixedRuntimeAudit(t, m)
	fixedRuntimeAudit(t, d, dst)
	fixedRuntimeAudit(t, s, src)
	heapMust(t, d.Deallocate(dst))
	heapMust(t, s.Deallocate(src))
	fixedRuntimeFinish(t, m, r, b)
	fixedRuntimeFinish(t, d, dr, db)
	fixedRuntimeFinish(t, s, sr, sb)
}
func TestFixedBitVectorRuntimeBorrowedReceiverAssertionsAndUnusedViews(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 33)
	d, dr, db := fixedRuntimeFixture(t, 65)
	dst, e := d.Allocate0()
	heapMust(t, e)
	dv, e := d.Borrow(dst)
	heapMust(t, e)
	e = m.SetView(dv, 40)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("receiver bound", e)
		}
	} else {
		heapMust(t, e)
		v, e := d.Get(dst, 40)
		heapMust(t, e)
		if !v {
			t.Fatal("foreign physical write")
		}
	}
	heapMust(t, m.SetRangeViews(dv, dv, 32, 1))
	heapMust(t, d.Deallocate(dst))
	fixedRuntimeFinish(t, m, r, b)
	fixedRuntimeFinish(t, d, dr, db)
	z, zr, zb := fixedRuntimeFixture(t, 0)
	var unused RuntimeFixedBitVectorView
	heapMust(t, z.CopyViews(unused, unused))
	heapMust(t, z.Fill0View(unused))
	heapMust(t, z.Fill1View(unused))
	heapMust(t, z.NegView(unused))
	heapMust(t, z.UnionViews(unused, unused))
	heapMust(t, z.IntersectViews(unused, unused))
	eq, e := z.EqualViews(unused, unused)
	heapMust(t, e)
	if !eq {
		t.Fatal(eq)
	}
	has, e := z.ContainsViews(unused, unused)
	heapMust(t, e)
	if !has {
		t.Fatal(has)
	}
	h, e := z.HashView(unused)
	heapMust(t, e)
	if h != utilhash.String("", 0) {
		t.Fatal(h)
	}
	var text bytes.Buffer
	heapMust(t, z.DisplayView(&text, unused))
	if text.Len() != 0 {
		t.Fatal(text.String())
	}
	fixedRuntimeFinish(t, z, zr, zb)
}
func TestFixedBitVectorRuntimeBorrowedCopyActualExtentPrefix(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 65)
	s, sr, sb := fixedRuntimeFixture(t, 33)
	dst, e := m.Allocate0()
	heapMust(t, e)
	src, e := s.Allocate1()
	heapMust(t, e)
	dv, e := m.Borrow(dst)
	heapMust(t, e)
	sv, e := s.Borrow(src)
	heapMust(t, e)
	e = m.CopyViews(dv, sv)
	if e == nil {
		t.Fatal("actual third word source extent")
	}
	for i := uint32(0); i < 2; i++ {
		w, e := m.Word(dst, i)
		heapMust(t, e)
		if w != ^uint32(0) {
			t.Fatal("copied prefix", i, w)
		}
	}
	fixedRuntimeAudit(t, m, dst)
	fixedRuntimeAudit(t, s, src)
	heapMust(t, m.Deallocate(dst))
	heapMust(t, s.Deallocate(src))
	fixedRuntimeFinish(t, m, r, b)
	fixedRuntimeFinish(t, s, sr, sb)
}
func TestFixedBitVectorRuntimeBorrowedInlineActualColdHash(t *testing.T) {
	large, lr, lb := fixedRuntimeFixture(t, ^uint32(0))
	v, e := large.Allocate()
	heapMust(t, e)
	vv, e := large.Borrow(v)
	heapMust(t, e)
	s, sr, sb := fixedRuntimeFixture(t, 32)
	src, e := s.Allocate1()
	heapMust(t, e)
	sv, e := s.Borrow(src)
	heapMust(t, e)
	heapMust(t, large.SetRangeViews(vv, sv, 31, 0))
	small, r, b := fixedRuntimeFixture(t, 8)
	h, e := small.HashView(vv)
	heapMust(t, e)
	if h != utilhash.String(string([]byte{255}), 8) {
		t.Fatal(h)
	}
	if _, e = large.HashView(vv); e == nil || errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("initialized inlineword then reached second-word extent", e)
	}
	heapMust(t, large.Deallocate(v))
	heapMust(t, s.Deallocate(src))
	fixedRuntimeFinish(t, large, lr, lb)
	fixedRuntimeFinish(t, s, sr, sb)
	fixedRuntimeFinish(t, small, r, b)
}

func TestFixedBitVectorRuntimeForeignInlineFillInitializesActualBytes(t *testing.T) {
	receiver, r, b := fixedRuntimeFixture(t, 32)
	empty, er, eb := fixedRuntimeFixture(t, 0)
	v, e := empty.Allocate()
	heapMust(t, e)
	view, e := empty.Borrow(v)
	heapMust(t, e)
	heapMust(t, receiver.Fill0View(view))
	w, e := empty.Word(v, 0)
	heapMust(t, e)
	if w != 0 || empty.zeroInitialized != 0xF {
		t.Fatal(w, empty.zeroInitialized)
	}
	heapMust(t, receiver.Fill1View(view))
	w, e = empty.Word(v, 0)
	heapMust(t, e)
	if w != ^uint32(0) {
		t.Fatal(w)
	}
	heapMust(t, empty.Reset())
	w, e = empty.Word(v, 0)
	heapMust(t, e)
	if w != ^uint32(0) {
		t.Fatal("allocator reset did not reset inline word", w)
	}
	fixedRuntimeFinish(t, receiver, r, b)
	fixedRuntimeFinish(t, empty, er, eb)
}
func TestFixedBitVectorRuntimeForeignInlinePartialCopyByteInitialization(t *testing.T) {
	receiver, r, b := fixedRuntimeFixture(t, 32)
	empty, er, eb := fixedRuntimeFixture(t, 0)
	src, e := receiver.Allocate()
	heapMust(t, e)
	heapMust(t, receiver.allocator.Write(src.object, 0, 0xAB))
	heapMust(t, receiver.allocator.Write(src.object, 1, 0xCD))
	sv, e := receiver.Borrow(src)
	heapMust(t, e)
	v, e := empty.Allocate()
	heapMust(t, e)
	dv, e := empty.Borrow(v)
	heapMust(t, e)
	e = receiver.CopyViews(dv, sv)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) || empty.zeroInitialized != 3 || empty.zero&0xFFFF != 0xCDAB {
		t.Fatal("two written bytes BEFORE third source uninitialized guard", e, empty.zeroInitialized, empty.zero)
	}
	if _, e = empty.Word(v, 0); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("wholeword requires all four", e)
	}
	one, or, ob := fixedRuntimeFixture(t, 8)
	h, e := one.HashView(dv)
	heapMust(t, e)
	if h != utilhash.String(string([]byte{0xAB}), 8) {
		t.Fatal("only reached initialized byte", h)
	}
	three, tr, tb := fixedRuntimeFixture(t, 24)
	if _, e = three.HashView(dv); !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal("descending byte2 uninitialized before byte1/0", e)
	}
	fixedRuntimeAudit(t, receiver, src)
	fixedRuntimeAudit(t, empty, v)
	heapMust(t, receiver.Deallocate(src))
	fixedRuntimeFinish(t, receiver, r, b)
	fixedRuntimeFinish(t, empty, er, eb)
	fixedRuntimeFinish(t, one, or, ob)
	fixedRuntimeFinish(t, three, tr, tb)
}

func TestFixedBitVectorRuntimeCapturedDirectViewAfterDonorClose(t *testing.T) {
	donor, r, b := fixedRuntimeFixture(t, 32)
	v, e := donor.Allocate1()
	heapMust(t, e)
	span, e := donor.Capture(v)
	heapMust(t, e)
	checked, e := donor.Borrow(v)
	heapMust(t, e)
	receiver, rr, rb := fixedRuntimeFixture(t, 32)
	raw := span.View()
	eq, e := receiver.EqualViews(checked, raw)
	heapMust(t, e)
	if !eq {
		t.Fatal("same actual captured pointer before data read")
	}
	heapMust(t, donor.Close())
	if _, e = checked.get(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("checked handle owner lifetime", e)
	}
	h, e := receiver.HashView(raw)
	heapMust(t, e)
	if h != utilhash.String(string([]byte{255, 255, 255, 255}), 32) {
		t.Fatal(h)
	}
	var text bytes.Buffer
	heapMust(t, receiver.DisplayView(&text, raw))
	if text.String() != string(bytes.Repeat([]byte{'1'}, 32)) {
		t.Fatal(text.String())
	}
	heapMust(t, receiver.UnsetView(raw, 3))
	bit, e := raw.get(3)
	heapMust(t, e)
	if bit {
		t.Fatal("actual raw store after donor Close")
	}
	fixedRuntimeAuditExtra(t, donor, nil, []memory.Block{span.Block})
	heapMust(t, span.ReleaseDirect())
	if _, e = receiver.HashView(raw); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("reached raw lifetime guard", e)
	}
	fixedRuntimeFinish(t, receiver, rr, rb)
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestFixedBitVectorRuntimeRawAddressIdentityBeforePayloadReads(t *testing.T) {
	m, r, b := fixedRuntimeFixture(t, 32)
	a, e := m.Allocate0()
	heapMust(t, e)
	c, e := m.Allocate1()
	heapMust(t, e)
	u, e := m.Allocate()
	heapMust(t, e)
	raw := func(v RuntimeFixedBitVector) RuntimeFixedBitVectorSpan {
		h, off, e := m.allocator.RawLocation(v.object)
		heapMust(t, e)
		return RuntimeFixedBitVectorSpan{Runtime: r, Block: h, Offset: off, Extent: 4, Profile: m.profile, Direct: true}
	}
	sa, sc, su := raw(a), raw(c), raw(u)
	eq, e := m.EqualViews(sa.View(), sc.View())
	heapMust(t, e)
	if eq {
		t.Fatal("distinct actual raw pointers/words falsely self-equal")
	}
	uv, e := m.Borrow(u)
	heapMust(t, e)
	eq, e = m.EqualViews(uv, su.View())
	heapMust(t, e)
	if !eq {
		t.Fatal("mixed same actual address before uninitialized read")
	}
	same := su
	same.Extent = 0
	same.Profile = memory.SmallAllocatorProfile{}
	eq, e = m.EqualViews(su.View(), same.View())
	heapMust(t, e)
	if !eq {
		t.Fatal("source self-address excludes extent/profile")
	}
	for _, v := range []RuntimeFixedBitVector{a, c, u} {
		heapMust(t, m.Deallocate(v))
	}
	fixedRuntimeFinish(t, m, r, b)
	receiver, rr, rb := fixedRuntimeFixture(t, 32)
	owner, or, ob := fixedRuntimeFixture(t, 64)
	v, e := owner.Allocate0()
	heapMust(t, e)
	heapMust(t, owner.putWord(v, 1, 1))
	span, e := owner.Capture(v)
	heapMust(t, e)
	first, second := span, span
	first.Extent = 4
	second.Offset += 4
	second.Extent = 4
	eq, e = receiver.EqualViews(first.View(), second.View())
	heapMust(t, e)
	if eq {
		t.Fatal("same block distinct offsets")
	}
	heapMust(t, owner.Deallocate(v))
	fixedRuntimeFinish(t, receiver, rr, rb)
	fixedRuntimeFinish(t, owner, or, ob)
}
