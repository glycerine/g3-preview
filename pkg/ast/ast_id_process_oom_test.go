package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTIDRecycleReadsSharedProcessOutOfMemory(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	p := memory.NewProcessAccounting()
	r, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	other, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: p, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	integers := numeral.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: integers})
	if err != nil {
		t.Fatal(err)
	}
	sy, err := symbol.NewManager(symbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	sort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	first, err := m.MkVar(17, sort)
	if err != nil {
		t.Fatal(err)
	}
	firstID, err := m.GetASTID(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IncRef(first); err != nil {
		t.Fatal(err)
	}
	if err := other.SetMaxSize(1); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Allocate(1); err == nil || !r.State().OutOfMemory || !m.IsOutOfMemory() {
		t.Fatal("second runtime must publish shared source OOM", err, r.State())
	}
	if err := m.DecRef(first); err != nil || m.ExprIDs.FreeLen() != 0 {
		t.Fatal("source ID recycle must skip during process OOM", err, m.ExprIDs.FreeIDs())
	}
	second, err := m.MkVar(18, sort)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := m.GetASTID(second)
	if err != nil || secondID == firstID {
		t.Fatal("deleted ID was recycled after process OOM", secondID, firstID, err)
	}
	for _, err := range []error{m.Close(), q.Close(), sy.Close(), other.Close(), r.Close()} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.Used() != 0 {
		t.Fatal("retained process OOM owners", b.Used())
	}
}
