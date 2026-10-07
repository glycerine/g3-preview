// Source: src/util/approx_set.h:43-158,162-233; src/util/approx_set.cpp:22-51.
// Source author: Leonardo de Moura (2007-03-02).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestUintApproxSetSourceIteratorRemainingWordEquality(t *testing.T) {
	var empty UintApproxSet
	if !empty.Empty() || empty.Size() != 0 || reflect.TypeOf(empty).Size() != 8 || !empty.Begin().Equal(UintApproxSetEnd()) {
		t.Fatal("source default/traits")
	}
	s := UintApproxSetFrom([]uint32{0, 2, 31, 32, 63, 64, 127, ^uint32(0)})
	want := []uint32{0, 2, 31, 32, 63}
	it := s.Begin()
	for _, v := range want {
		if it.Equal(UintApproxSetEnd()) || it.Current() != v {
			t.Fatal("literal ascending shift traversal", v, it)
		}
		saved := it
		old := it.PostNext()
		if !old.Equal(saved) || old.Current() != saved.Current() {
			t.Fatal("postincrement copy")
		}
	}
	if !it.Equal(UintApproxSetEnd()) || s.Size() != uint32(len(want)) {
		t.Fatal("end/population after collisions", it, s.Size())
	}
	it.Next()
	if !it.Equal(UintApproxSetEnd()) || it.Current() != 65 {
		t.Fatal("source end increment still increments value", it)
	}
	a := NewUintApproxSet(0).Begin()
	b := NewUintApproxSet(1).Begin()
	if a.Current() == b.Current() || !a.Equal(b) {
		t.Fatal("source iterator equality compares remaining word only", a, b)
	}
}

func TestUintApproxSetIndependentFiniteUnsignedAlgebraAndSize(t *testing.T) {
	values := []uint32{0, 2, 9, 31, 32, 63, 64, ^uint32(0)}
	var sets [256]UintApproxSet
	var oracle [256][64]bool
	for mask := 0; mask < 256; mask++ {
		var input []uint32
		for i, v := range values {
			if mask&(1<<i) != 0 {
				input = append(input, v)
				oracle[mask][v%64] = true
			}
		}
		sets[mask] = UintApproxSetFrom(input)
		var size uint32
		for _, v := range oracle[mask] {
			if v {
				size++
			}
		}
		if sets[mask].Size() != size {
			t.Fatal("approximate population is residue count", mask, size)
		}
		copy := sets[mask]
		copy.Reset()
		if !copy.Empty() || sets[mask].Size() != size {
			t.Fatal("source value-copy/reset")
		}
	}
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			u := sets[a].Union(sets[b])
			i := sets[a].Intersection(sets[b])
			d := sets[a]
			d.SubtractAssign(sets[b])
			ua := sets[a]
			ua.UnionAssign(sets[b])
			ia := sets[a]
			ia.IntersectAssign(sets[b])
			if ua != u || ia != i {
				t.Fatal("source assignment operators")
			}
			subset, equal, disjoint := true, true, true
			for r := 0; r < 64; r++ {
				av, bv := oracle[a][r], oracle[b][r]
				v := uint32(r)
				if u.MayContain(v) != (av || bv) || i.MayContain(v) != (av && bv) || d.MayContain(v) != (av && !bv) || sets[a].MustNotContain(v) == av {
					t.Fatal(a, b, r)
				}
				if av && !bv {
					subset = false
				}
				if av != bv {
					equal = false
				}
				if av && bv {
					disjoint = false
				}
			}
			if sets[a].ApproxSubset(sets[b]) != subset || sets[a].MustNotSubset(sets[b]) == subset || sets[a].MustNotSubsume(sets[b]) == subset || sets[a].Equivalent(sets[b]) != equal || sets[a].MayEqual(sets[b]) != equal || sets[a].MustNotEqual(sets[b]) == equal || sets[a].EmptyIntersection(sets[b]) != disjoint {
				t.Fatal(a, b)
			}
		}
	}
}

func TestUintApproxSetSourceDisplayFlagsEveryOutputPrefixAndBorrow(t *testing.T) {
	for _, tc := range []struct {
		values []uint32
		text   string
	}{{nil, "{}"}, {[]uint32{64}, "{0}"}, {[]uint32{127, 1, 65, 32}, "{1, 32, 63}"}} {
		s := UintApproxSetFrom(tc.values)
		copy := s
		f, out, b := numericFileFixture(t)
		before := b.Used()
		b.FailAfter(0)
		paramsMust(t, s.Display(out))
		if s != copy || b.Used() != before {
			t.Fatal("const inline set display")
		}
		b.FailAfter(-1)
		_, e := f.Seek(0, io.SeekStart)
		paramsMust(t, e)
		got, e := io.ReadAll(f)
		paramsMust(t, e)
		if string(got) != tc.text {
			t.Fatal(string(got), tc.text)
		}
		paramsMust(t, out.Close())
		if _, e = f.Write([]byte("caller")); e != nil {
			t.Fatal("borrowed stream file closed", e)
		}
	}
	f, out, _ := numericFileFixture(t)
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, out.SetIntegerBase(16))
	paramsMust(t, out.SetUppercase(true))
	paramsMust(t, out.SetShowBase(true))
	paramsMust(t, out.SetWidth(4))
	paramsMust(t, out.SetFill('_'))
	s := NewUintApproxSet(63)
	paramsMust(t, s.Display(out))
	_, e := f.Seek(0, io.SeekStart)
	paramsMust(t, e)
	got, e := io.ReadAll(f)
	paramsMust(t, e)
	if string(got) != "___{0X3F}" {
		t.Fatal("unsigned numeric insertion and literal width boundary", string(got))
	}
	paramsMust(t, out.Close())
	if e = s.Display(out); e == nil {
		t.Fatal("closed stream")
	}
	if e = s.Display(nil); e != ErrNumericStreamOwner {
		t.Fatal("invalid stream", e)
	}
	s = UintApproxSetFrom([]uint32{0, 2, 31, 32, 63})
	saved := s
	completed := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		b := arena.NewBudget(^uint64(0))
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		b.FailAfter(ordinal)
		cause := s.Display(out)
		stats := out.Stats()
		if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes || s != saved {
			t.Fatal("immediate stream cold/value graph", ordinal, stats, b.Used())
		}
		text, e := out.String()
		paramsMust(t, e)
		if !strings.HasPrefix("{0, 2, 31, 32, 63}", text) {
			t.Fatal("source output prefix", ordinal, text, cause)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(cause)
		}
		b.FailAfter(-1)
		paramsMust(t, out.Close())
		if b.Used() != 0 {
			t.Fatal("cold refund")
		}
		if cause == nil {
			if text != "{0, 2, 31, 32, 63}" {
				t.Fatal(text)
			}
			completed = true
			break
		}
	}
	if !completed {
		t.Fatal("no actual output completion horizon")
	}
}
