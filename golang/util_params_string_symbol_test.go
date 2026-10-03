// Copyright (c) 2012 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestParamsBorrowedStringSymbolCopyAndOwnership(t *testing.T) {
	c, s, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	k, e := s.Intern("text")
	paramsMust(t, e)
	spelling, e := s.Intern("real borrowed\x00discarded")
	paramsMust(t, e)
	paramsMust(t, c.SetStringSymbol(&v, k, spelling))
	x := c.entry(&c.blocks[v.slot-1], 0)
	if x.kind != ParamString || x.symbol != spelling || !x.text.borrowed || !x.text.present || x.text.ref != (arena.VectorRef{}) {
		t.Fatal("scalar borrowed record", *x)
	}
	before := c.texts.Stats()
	text, present, e := c.GetString(&v, k, nil)
	paramsMust(t, e)
	if !present || text != "real borrowed" {
		t.Fatal(text, present)
	}
	text, e = c.DisplayKey(&v, k)
	paramsMust(t, e)
	if text != "real borrowed" {
		t.Fatal(text)
	}
	paramsMust(t, c.SetUInt(&dst, k, 9))
	paramsMust(t, c.Copy(&dst, &v))
	y := c.entry(&c.blocks[dst.slot-1], 0)
	if !y.text.borrowed || y.symbol != spelling || y.text.ref != (arena.VectorRef{}) {
		t.Fatal("copy lost borrowed identity", *y)
	}
	if c.texts.Stats() != before {
		t.Fatal("borrowed copy allocated owned text")
	}
	paramsMust(t, c.Del(&v))
	text, present, e = c.GetString(&dst, k, nil)
	paramsMust(t, e)
	if !present || text != "real borrowed" {
		t.Fatal(text)
	}
	paramsMust(t, c.Del(&dst))
	text, present, e = s.BareString(spelling)
	paramsMust(t, e)
	if !present || text != "real borrowed" {
		t.Fatal("Del destroyed borrowed symbol")
	}
}
func TestParamsBorrowedStringSymbolReplacementAndWarmSet(t *testing.T) {
	c, s, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, c.Del(&v)) }()
	k, e := s.Intern("text")
	paramsMust(t, e)
	empty, e := s.Intern("")
	paramsMust(t, e)
	owned := "owned payload"
	paramsMust(t, c.SetString(&v, k, &owned))
	old := c.entry(&c.blocks[v.slot-1], 0).text.ref
	if !c.texts.IsLive(old) {
		t.Fatal("owned precondition")
	}
	paramsMust(t, c.SetStringSymbol(&v, k, empty))
	if c.texts.IsLive(old) {
		t.Fatal("replacement retained owned root")
	}
	text, present, e := c.GetString(&v, k, nil)
	paramsMust(t, e)
	if !present || text != "" {
		t.Fatal("empty vs null", text, present)
	}
	c.budget.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := c.SetStringSymbol(&v, k, empty); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warmed scalar setter", n)
	}
	c.budget.FailAfter(-1)
	paramsMust(t, c.SetString(&v, k, &owned))
	if c.entry(&c.blocks[v.slot-1], 0).text.borrowed {
		t.Fatal("owned route changed")
	}
}
func TestParamsBorrowedStringSymbolProvenanceAndLifetimeGuards(t *testing.T) {
	c, s, _ := paramsTestOwner(t)
	v, e := c.NewValue()
	paramsMust(t, e)
	k, e := s.Intern("text")
	paramsMust(t, e)
	spelling, e := s.Intern("live")
	paramsMust(t, e)
	before := c.Stats()
	for _, bad := range []Symbol{Symbol{}, NumericalSymbol(1), DummySymbol()} {
		if e = c.SetStringSymbol(&v, k, bad); e == nil {
			t.Fatal("invalid bare string accepted")
		}
		if c.Stats() != before {
			t.Fatal("guard mutated params owner")
		}
	}
	_, foreign, _ := paramsTestOwner(t)
	f, e := foreign.Intern("foreign")
	paramsMust(t, e)
	if e = c.SetStringSymbol(&v, k, f); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal(e)
	}
	paramsMust(t, c.SetStringSymbol(&v, k, spelling))
	paramsMust(t, s.Close())
	if _, _, e = c.GetString(&v, k, nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("borrow lifetime", e)
	}
	// Source ownership cleanup does not dereference borrowed spelling.
	paramsMust(t, c.Close())
	v = ParamsRef{owner: c.owner}
}

func paramsBorrowedPhysicalBytes(s ParamsStats) uint64 {
	d := s.Text.Descriptors
	return s.HeaderBytes + s.BlockBytes + s.PageBytes + s.DirectoryBytes + s.FormatBytes + s.Text.OwnerBytes + s.Text.PayloadBytes + s.Text.BlockMetadataBytes + s.Text.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func TestParamsBorrowedStringSymbolSharedCopyFaultPrefixes(t *testing.T) {
	for fail := int64(0); fail < 12; fail++ {
		c, s, _ := paramsTestOwner(t)
		v, e := c.NewValue()
		paramsMust(t, e)
		alias, e := c.NewValue()
		paramsMust(t, e)
		k, e := s.Intern("text")
		paramsMust(t, e)
		old, e := s.Intern("source")
		paramsMust(t, e)
		next, e := s.Intern("destination")
		paramsMust(t, e)
		paramsMust(t, c.SetStringSymbol(&v, k, old))
		paramsMust(t, c.Assign(&alias, &v))
		baseline := c.budget.Used() - paramsBorrowedPhysicalBytes(c.Stats())
		original := *c.entry(&c.blocks[v.slot-1], 0)
		wordsBefore := c.texts.Stats()
		c.budget.FailAfter(fail)
		e = c.SetStringSymbol(&alias, k, next)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(fail, e)
		}
		// These immediate ownership checks happen with the fault controller armed,
		// before any getter can grow format scratch or teardown hides a root.
		if got := *c.entry(&c.blocks[v.slot-1], 0); got != original {
			t.Fatal("shared source changed", fail)
		}
		if c.texts.Stats() != wordsBefore {
			t.Fatal("copy-on-write allocated spelling", fail)
		}
		if got, want := c.budget.Used(), baseline+paramsBorrowedPhysicalBytes(c.Stats()); got != want {
			t.Fatal("actual prefix graph", fail, got, want)
		}
		c.budget.FailAfter(-1)
		if e == nil {
			x := c.entry(&c.blocks[alias.slot-1], 0)
			if !x.text.borrowed || x.symbol != next {
				t.Fatal("successful deep cell copy", fail)
			}
		}
		paramsMust(t, c.Del(&alias))
		paramsMust(t, c.Del(&v))
	}
}
