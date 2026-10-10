package lp

import (
	"bytes"
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"
)

// Source: src/test/lp.cpp:2968-2987, test_int_set (--test_int_set).
func TestLPIntSetSource(t *testing.T) {
	s, err := NewIntSet(4)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range []uint32{2, 1, 2} {
		if err = s.Insert(j); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Contains(2) || s.Size() != 2 {
		t.Fatal("source insert assertions")
	}
	s.Erase(2)
	if s.Size() != 1 {
		t.Fatal("source erase assertion")
	}
	s.Erase(2)
	if s.Size() != 1 {
		t.Fatal("source repeated erase assertion")
	}
	if err = s.Insert(3); err != nil {
		t.Fatal(err)
	}
	if err = s.Insert(2); err != nil {
		t.Fatal(err)
	}
	if err = s.Clear(); err != nil || s.Size() != 0 {
		t.Fatal("source clear assertion", err)
	}
}

func TestLPIntSetDenseOrderResizeAndGuards(t *testing.T) {
	var s IntSet
	if !s.Empty() || s.Contains(0) || s.DataSize() != 0 {
		t.Fatal("default constructor")
	}
	if s.Insert(0) != ErrBounds {
		t.Fatal("out of domain")
	}
	s.Erase(math.MaxUint32)
	if err := s.IncreaseSizeByOne(); err != nil {
		t.Fatal(err)
	}
	if err := s.Resize(5); err != nil {
		t.Fatal(err)
	}
	for _, j := range []uint32{4, 0, 2} {
		if err := s.Insert(j); err != nil {
			t.Fatal(err)
		}
	}
	s.Erase(4)
	if !reflect.DeepEqual(s.Indices(), []uint32{2, 0}) {
		t.Fatal("last-member replacement", s.Indices())
	}
	s.Erase(0)
	if !reflect.DeepEqual(s.Indices(), []uint32{2}) {
		t.Fatal("last member erase")
	}
	if err := s.Resize(3); err != nil || !s.Contains(2) {
		t.Fatal("valid shrink", err)
	}
	if err := s.Resize(5); err != nil || s.Contains(3) || s.Contains(4) {
		t.Fatal("new slots must be absent", err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if err := s.Resize(0); err != nil || !s.Empty() {
		t.Fatal(err)
	}
	if s.Resize(math.MaxUint32) != ErrBounds {
		t.Fatal("signed source position domain")
	}
	if p, err := NewIntSet(math.MaxUint32); p != nil || err != ErrBounds {
		t.Fatal(p, err)
	}
}

func TestLPIntSetShrinkRetainsSourceIndexAndReachedClearPrefix(t *testing.T) {
	s, _ := NewIntSet(3)
	_ = s.Insert(0)
	_ = s.Insert(2)
	if err := s.Resize(1); err != nil {
		t.Fatal(err)
	}
	if s.Size() != 2 || !reflect.DeepEqual(s.Indices(), []uint32{0, 2}) {
		t.Fatal("resize must not repair source dense list")
	}
	if err := s.Clear(); err != ErrBounds || s.Contains(0) || s.Size() != 2 {
		t.Fatal("reached undefined-domain guard and completed prefix", err)
	}
}

func TestLPIntSetIndependentFiniteMembership(t *testing.T) {
	s, _ := NewIntSet(32)
	want := map[uint32]bool{}
	r := rand.New(rand.NewPCG(71, 19))
	for step := 0; step < 10000; step++ {
		j := uint32(r.IntN(40))
		switch r.IntN(4) {
		case 0, 1:
			err := s.Insert(j)
			if j < 32 {
				if err != nil {
					t.Fatal(err)
				}
				want[j] = true
			} else if err != ErrBounds {
				t.Fatal(err)
			}
		case 2:
			s.Erase(j)
			delete(want, j)
		case 3:
			if step%53 == 0 {
				if err := s.Clear(); err != nil {
					t.Fatal(err)
				}
				clear(want)
			}
		}
		if int(s.Size()) != len(want) || s.Empty() != (len(want) == 0) {
			t.Fatal(step, "size")
		}
		seen := map[uint32]bool{}
		for _, k := range s.Indices() {
			if seen[k] || !want[k] {
				t.Fatal(step, "dense list", k)
			}
			seen[k] = true
		}
		for k := uint32(0); k < 40; k++ {
			if s.Contains(k) != want[k] {
				t.Fatal(step, k)
			}
		}
	}
}

type lpRefusingWriter struct{ err error }

func (w lpRefusingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestLPIntSetPrintOrderAndWriterFailure(t *testing.T) {
	s, _ := NewIntSet(5)
	_ = s.Insert(3)
	_ = s.Insert(1)
	var out bytes.Buffer
	if err := s.Print(&out); err != nil || out.String() != "3 1 \n" {
		t.Fatal(out.String(), err)
	}
	failure := errors.New("writer refused")
	if err := s.Print(lpRefusingWriter{failure}); err != failure {
		t.Fatal(err)
	}
	_ = s.Clear()
	out.Reset()
	if err := s.Print(&out); err != nil || out.String() != "\n" {
		t.Fatal(out.String(), err)
	}
	if err := s.Print(lpRefusingWriter{failure}); err != failure {
		t.Fatal(err)
	}
}
