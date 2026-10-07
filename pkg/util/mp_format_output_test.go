// Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerSourceCharBufferInline1024AndActualLexicalOverflow(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var c sourceCharBuffer
	// mpz.cpp display constructs buffer(11*size(a),0), not a retained text page.
	if e = c.resize(r, 11*93); e != nil {
		t.Fatal(e)
	}
	if c.size() != 1023 || c.capacity() != 1024 || r.Stats().LiveBlocks != 0 {
		t.Fatal("source inline1024 char constructor allocated")
	}
	for i := uint32(0); i < c.size(); i++ {
		if v, e := c.read(r, i); e != nil || v != 0 {
			t.Fatal("source char constructor zero initialization", v, e)
		}
	}
	if e = c.finalize(r); e != nil {
		t.Fatal(e)
	}
	if e = c.resize(r, 11*94); e != nil {
		t.Fatal(e)
	}
	if c.size() != 1034 || c.capacity() != 2048 || r.Stats().LiveBlocks != 1 || r.Stats().PhysicalBytes != 2056 {
		t.Fatal("actual source char overflow topology")
	}
	if e = c.write(r, 0, '7'); e != nil {
		t.Fatal(e)
	}
	if e = c.write(r, 1, 0); e != nil {
		t.Fatal(e)
	}
	if v, e := c.read(r, 0); e != nil || v != '7' {
		t.Fatal("actual char backing", v, e)
	}
	if e = c.finalize(r); e != nil {
		t.Fatal(e)
	}
	if r.Stats().LiveBlocks != 0 || c.capacity() != 1024 || c.size() != 0 {
		t.Fatal("source lexical char destructor retained raw buffer")
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("source char scope root leak")
	}
}

func TestIntegerSourceCharGrowthRefusalRetainsActualOldAndCounterPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	var c sourceCharBuffer
	if e = c.resize(r, 2048); e != nil {
		t.Fatal(e)
	}
	if e = c.write(r, 0, 'x'); e != nil {
		t.Fatal(e)
	}
	old := r.Stats()
	b.FailAfter(0)
	if e = c.push(r, 'y'); !errors.Is(e, Failure{Kind: FailureMemout}) {
		t.Fatal("physical char allocation refusal", e)
	}
	if c.capacity() != 2048 || c.size() != 2048 || r.Stats() != old {
		t.Fatal("char refusal lost old physical root")
	}
	if v, e := c.read(r, 0); e != nil || v != 'x' {
		t.Fatal("char refusal lost prior payload", v, e)
	}
	if n, e := r.AllocationCount(); e != nil || n != 2 {
		t.Fatal("actual source char failed allocation count", n, e)
	}
	if n, e := r.AllocationSize(); e != nil || n != 6160 {
		t.Fatal("actual source char failed allocation extent", n, e)
	}
	b.FailAfter(-1)
	if e = c.finalize(r); e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("char refusal cleanup leak")
	}
}
