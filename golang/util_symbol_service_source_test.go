// Source: src/util/symbol.cpp:70-83.
// Licensed under the MIT license. See LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestSymbolServiceSourcePublicationAndReinitialize(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := NewSymbolService(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	header := b.Used()
	if _, e = s.Manager(); !errors.Is(e, ErrSymbolServiceUninitialized) {
		t.Fatal("static null pointer", e)
	}
	if e = s.FinalizeSource(); e != nil || b.Used() != header {
		t.Fatal("dealloc null", e, b.Used())
	}
	if e = s.InitializeSource(); e != nil {
		t.Fatal(e)
	}
	m, e := s.Manager()
	if e != nil {
		t.Fatal(e)
	}
	old, e := m.Intern("owned")
	if e != nil {
		t.Fatal(e)
	}
	used := b.Used()
	b.FailAfter(0)
	if e = s.InitializeSource(); e != nil || b.Used() != used {
		t.Fatal("pointer-present shortcut", e)
	}
	same, _ := s.Manager()
	if same != m {
		t.Fatal("replaced live manager")
	}
	if e = s.FinalizeSource(); e != nil || b.Used() != header {
		t.Fatal("finalize", e, b.Used())
	}
	if _, e = s.Manager(); !errors.Is(e, ErrSymbolServiceUninitialized) {
		t.Fatal("pointer cleared", e)
	}
	if _, e = m.Intern("dead"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("old owner live", e)
	}
	b.FailAfter(-1)
	if e = s.InitializeSource(); e != nil {
		t.Fatal(e)
	}
	next, _ := s.Manager()
	fresh, e := next.Intern("owned")
	if e != nil || fresh == old {
		t.Fatal("fresh owner", e)
	}
	if _, e = next.String(old); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal("stale identity accepted", e)
	}
	if e = s.Close(); e != nil || b.Used() != 0 {
		t.Fatal("close", e, b.Used())
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = s.InitializeSource(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner", e)
	}
}

func TestSymbolServiceSourceEveryConstructorReservation(t *testing.T) {
	for _, direct := range []bool{false, true} {
		success := false
		for prefix := int64(0); prefix < 40; prefix++ {
			b := arena.NewBudget(1 << 24)
			s, e := NewSymbolService(SymbolConfig{Budget: b, DirectOnly: direct})
			if e != nil {
				t.Fatal(e)
			}
			header := b.Used()
			b.FailAfter(prefix)
			e = s.InitializeSource()
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(prefix, e)
				}
				if s.manager != nil || b.Used() != header {
					t.Fatal("unpublished construction graph", prefix, b.Used(), header)
				}
				if x := s.FinalizeSource(); x != nil || b.Used() != header {
					t.Fatal("null finalization", prefix, x)
				}
			} else {
				if s.manager == nil || b.Used() <= header {
					t.Fatal("completed manager publication", prefix)
				}
				st := s.manager.Stats()
				d := st.Region.Descriptors
				owned := header + st.OwnerBytes + st.Table.OwnerBytes + st.Table.TableBytes + st.Table.DirectoryBytes + st.Region.OwnerBytes + st.Region.PayloadBytes + st.Region.PageHeaderBytes + st.Region.DirectoryBytes + st.Region.LogBytes + st.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
				if owned != b.Used() {
					t.Fatal("immediate complete constructor graph", prefix, owned, b.Used())
				}
				success = true
			}
			b.FailAfter(-1)
			if !success {
				if e = s.InitializeSource(); e != nil {
					t.Fatal("retry", prefix, e)
				}
			}
			if e = s.Close(); e != nil || b.Used() != 0 {
				t.Fatal("owner cleanup", prefix, e, b.Used())
			}
			if success {
				t.Log("first success", direct, prefix)
				break
			}
		}
		if !success {
			t.Fatal("constructor horizon", direct)
		}
	}
}

func TestSymbolServiceSourceCopiedOwnerAndPrepareFailure(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	b.FailAfter(0)
	if s, e := NewSymbolService(SymbolConfig{Budget: b}); s != nil || !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != 0 {
		t.Fatal("cold owner refusal", s, e, b.Used())
	}
	b.FailAfter(-1)
	s, e := NewSymbolService(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	copy := new(SymbolService)
	reflect.ValueOf(copy).Elem().Set(reflect.ValueOf(s).Elem())
	used := b.Used()
	b.FailAfter(0)
	if e = copy.InitializeSource(); !errors.Is(e, ErrSymbolServiceOwner) {
		t.Fatal("copied init", e)
	}
	if e = copy.FinalizeSource(); !errors.Is(e, ErrSymbolServiceOwner) {
		t.Fatal("copied finalize", e)
	}
	if e = copy.Close(); !errors.Is(e, ErrSymbolServiceOwner) || b.Used() != used {
		t.Fatal("copied cleanup", e)
	}
	b.FailAfter(-1)
	if e = s.Close(); e != nil || b.Used() != 0 {
		t.Fatal("original cleanup", e, b.Used())
	}
}
