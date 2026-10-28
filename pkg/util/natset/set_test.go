package natset

// Source: src/util/nat_set.h:25-85.

import "testing"

func TestMembershipResetAndRemoval(t *testing.T) {
	s := New(4)
	if s.Domain() != 4 || !s.Empty() {
		t.Fatal(s.Domain(), s.Empty())
	}
	s.Insert(1)
	s.Insert(3)
	if !s.Contains(1) || !s.Contains(3) || s.Contains(0) || s.Empty() {
		t.Fatal("insert membership")
	}
	s.Remove(1)
	if s.Contains(1) || !s.Contains(3) {
		t.Fatal("remove membership")
	}
	s.Reset()
	if !s.Empty() || s.Contains(3) {
		t.Fatal("reset membership")
	}
}

func TestDomainGrowthShrinkAndRegrowth(t *testing.T) {
	s := New(2)
	s.Insert(1)
	s.AssureDomain(5)
	if s.Domain() != 6 || !s.Contains(1) || s.Contains(5) {
		t.Fatal("growth")
	}
	s.Insert(5)
	s.SetDomain(2)
	if s.Domain() != 2 || !s.Contains(1) {
		t.Fatal("shrink")
	}
	s.SetDomain(6)
	if s.Contains(5) || !s.Contains(1) {
		t.Fatal("regrowth")
	}
}

func TestTimestampWrapClearsBacking(t *testing.T) {
	s := New(3)
	s.current = ^uint32(0) - 1
	s.timestamps[0] = ^uint32(0)
	s.timestamps[1] = 17
	s.Reset()
	if s.current != 0 || !s.Empty() {
		t.Fatal(s.current, s.timestamps)
	}
	s.Insert(2)
	if !s.Contains(2) {
		t.Fatal("insert after wrap")
	}
}

func TestOutOfDomainAccessPanics(t *testing.T) {
	for name, call := range map[string]func(*Set){
		"contains": func(s *Set) { s.Contains(2) },
		"insert":   func(s *Set) { s.Insert(2) },
		"remove":   func(s *Set) { s.Remove(2) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("out-of-domain access did not panic")
				}
			}()
			call(New(2))
		})
	}
}
