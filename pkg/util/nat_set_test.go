// Source: src/util/nat_set.h:25-85.
package util

import "testing"

func TestNatSetDomainMembershipResetAndRemoval(t *testing.T) {
	s := NewNatSet(4)
	if s.GetDomain() != 4 || !s.Empty() {
		t.Fatal("initial state", s.GetDomain(), s.Empty())
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
	s.Insert(0)
	if !s.Contains(0) || s.Empty() {
		t.Fatal("insert after reset")
	}
}

func TestNatSetDomainGrowthShrinkAndRegrowth(t *testing.T) {
	s := NewNatSet(2)
	s.Insert(1)
	s.AssureDomain(5)
	if s.GetDomain() != 6 || !s.Contains(1) || s.Contains(5) {
		t.Fatal("growth", s.GetDomain(), s.Contains(1), s.Contains(5))
	}
	s.Insert(5)
	s.SetDomain(2)
	if s.GetDomain() != 2 || !s.Contains(1) {
		t.Fatal("shrink")
	}
	s.SetDomain(6)
	if s.Contains(5) || !s.Contains(1) {
		t.Fatal("regrowth must zero removed suffix")
	}
	s.AssureDomain(1)
	if s.GetDomain() != 6 {
		t.Fatal("assure inside domain changed size", s.GetDomain())
	}
}

func TestNatSetTimestampWrapClearsBacking(t *testing.T) {
	s := NewNatSet(3)
	s.current = ^uint32(0) - 1
	s.timestamps[0] = ^uint32(0)
	s.timestamps[1] = 17
	s.Reset()
	if s.current != 0 || !s.Empty() {
		t.Fatal("wrap reset", s.current, s.timestamps)
	}
	for i, timestamp := range s.timestamps {
		if timestamp != 0 {
			t.Fatal("uncleared timestamp", i, timestamp)
		}
	}
	s.Insert(2)
	if !s.Contains(2) {
		t.Fatal("insert after wrap")
	}
}

func TestNatSetOutOfDomainAccessPanics(t *testing.T) {
	tests := []struct {
		name string
		call func(*NatSet)
	}{
		{"contains", func(s *NatSet) { s.Contains(2) }},
		{"insert", func(s *NatSet) { s.Insert(2) }},
		{"remove", func(s *NatSet) { s.Remove(2) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("out-of-domain access did not panic")
				}
			}()
			test.call(NewNatSet(2))
		})
	}
}
