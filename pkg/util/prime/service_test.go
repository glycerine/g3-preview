// Source: src/util/prime_generator.cpp:23-27,99-130; src/util/prime_generator.h:29-52.
package prime

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestDefaultPrimeEagerLifetimeAndIteratorSourcePrefixes(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewDefaultService(b)
	primeMust(t, e)
	defer s.Close()
	st := s.Stats()
	if st.Generator.Count < 2 || st.OwnerBytes != uint64(reflect.TypeOf(DefaultService{}).Size()) || b.Used() != st.TotalBytes {
		t.Fatal("actual eager graph", st, b.Used())
	}
	g := s.generator
	it, e := NewServiceIterator(s, nil)
	primeMust(t, e)
	p, e := it.Next()
	primeMust(t, e)
	if p != 2 || it.index != 1 || !it.global || it.generator != g {
		t.Fatal(p, it)
	}
	primeMust(t, s.Finalize())
	if s.generator != g || s.Stats().Generator.Count != 0 || s.Stats().Generator.Capacity != 0 {
		t.Fatal("no object recreation")
	}
	before := b.Used()
	_, e = it.Next()
	if !errors.Is(e, arena.ErrBounds) || it.index != 2 || b.Used() != before {
		t.Fatal("increment before empty back read", e, it.index, b.Used(), before)
	}
	again, e := NewServiceIterator(s, nil)
	primeMust(t, e)
	_, e = again.Next()
	if !errors.Is(e, arena.ErrBounds) || again.index != 1 || s.generator != g {
		t.Fatal(e, again.index)
	}
	primeMust(t, s.Close())
	_, e = again.Next()
	if !errors.Is(e, arena.ErrClosed) || again.index != 2 {
		t.Fatal(e, again.index)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestDefaultPrimeRegisteredSequential10000IndependentSieve(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := NewDefaultService(b)
	primeMust(t, e)
	defer s.Close()
	it, e := NewServiceIterator(s, nil)
	primeMust(t, e)
	var composite [104730]bool
	for p := 2; p*p < len(composite); p++ {
		if !composite[p] {
			for n := p * p; n < len(composite); n += p {
				composite[n] = true
			}
		}
	}
	idx := 0
	for n := 2; n < len(composite); n++ {
		if !composite[n] {
			p, e := it.Next()
			primeMust(t, e)
			if p != uint64(n) {
				t.Fatal(idx, p, n)
			}
			idx++
		}
	}
	if idx != 10000 || it.index != 10000 || b.Used() != s.Stats().TotalBytes {
		t.Fatal(idx, it.index, b.Used(), s.Stats())
	}
}
func TestDefaultPrimeEveryEagerConstructionRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(ordinal)
		s, e := NewDefaultService(b)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != 0 {
				t.Fatal(ordinal, s, e, b.Used())
			}
			failures++
		} else {
			success++
			if b.Used() != s.Stats().TotalBytes {
				t.Fatal(ordinal, b.Used(), s.Stats())
			}
		}
		b.FailAfter(-1)
		if s != nil {
			primeMust(t, s.Close())
		}
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success != 0 {
			break
		}
	}
	if failures == 0 || success != 1 {
		t.Fatal(failures, success)
	}
	t.Log("actual refusal horizon", failures)
}
func TestDefaultPrimeLocalAndCachedWarmSourceRoutes(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewDefaultService(b)
	primeMust(t, e)
	defer s.Close()
	local, e := NewGenerator(b)
	primeMust(t, e)
	defer local.Close()
	it, e := NewServiceIterator(nil, local)
	primeMust(t, e)
	if it.global {
		t.Fatal("explicit local route")
	}
	before := b.Used()
	b.FailAfter(0)
	for _, p := range []uint64{2, 3, 5, 7, 11} {
		got, e := it.Next()
		primeMust(t, e)
		if got != p {
			t.Fatal(got, p)
		}
	}
	if b.Used() != before {
		t.Fatal("cached path allocated")
	}
	b.FailAfter(-1)
	copy := *s
	if e = copy.Finalize(); !errors.Is(e, ErrServiceOwner) {
		t.Fatal(e)
	}
}
func TestDefaultPrimeIndexWrapAndActualGrowthRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := NewDefaultService(b)
	primeMust(t, e)
	defer s.Close()
	it, e := NewServiceIterator(s, nil)
	primeMust(t, e)
	it.index = ^uint32(0)
	_, e = it.Next()
	if !errors.Is(e, ErrCapacity) || it.index != 0 {
		t.Fatal(e, it.index)
	}
	p, e := it.Next()
	primeMust(t, e)
	if p != 2 {
		t.Fatal(p)
	}
	it.index = s.Stats().Generator.Count
	before := s.Stats()
	b.FailAfter(0)
	_, e = it.Next()
	after := s.Stats()
	if !errors.Is(e, arena.ErrInjectedFailure) || it.index != before.Generator.Count+1 ||
		after.Generator.Count <= before.Generator.Count || after.Generator.Count > after.Generator.Capacity ||
		after.TotalBytes != before.TotalBytes || b.Used() != before.TotalBytes {
		t.Fatal(e, it.index, after, before, b.Used())
	}
	for i := before.Generator.Count; i < after.Generator.Count; i++ {
		p, e := s.generator.At(i)
		primeMust(t, e)
		for d := uint64(2); d <= p/d; d++ {
			if p%d == 0 {
				t.Fatal("published composite", i, p)
			}
		}
	}
	b.FailAfter(-1)
}

func primeMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
