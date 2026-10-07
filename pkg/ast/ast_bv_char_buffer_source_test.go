// Source: src/util/buffer.h:25-63,261-268; src/ast/ast_smt2_pp.cpp:185-229.
package ast

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorCharacterBufferLiteralInlineAndGrowth(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	var buf astBVCharBuffer16
	b.FailAfter(0)
	for i := byte(0); i < 16; i++ {
		astTestMust(t, buf.push(r, i))
	}
	if buf.capacity() != 16 || buf.pos != 16 || r.Stats().LiveBlocks != 0 {
		t.Fatal("actual inline buffer", buf.pos, buf.capacity(), r.Stats())
	}
	b.FailAfter(-1)
	for i := byte(16); i < 65; i++ {
		astTestMust(t, buf.push(r, i))
	}
	for i := uint32(0); i < 65; i++ {
		v, e := buf.read(r, i)
		astTestMust(t, e)
		if v != byte(i) {
			t.Fatal("ascending growth copy", i, v)
		}
	}
	if buf.capacity() != 128 {
		t.Fatal("literal doubled capacity", buf.capacity())
	}
	astTestMust(t, buf.sourceDestroy(r))
	astTestMust(t, buf.cleanup(r))
	astTestMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual owner end", b.Used())
	}
}

func TestASTBitVectorCharacterBufferEveryGrowthReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		b := arena.NewBudget(1 << 20)
		r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
		astTestMust(t, e)
		foreign, e := r.Allocate(7)
		astTestMust(t, e)
		var buf astBVCharBuffer16
		for i := byte(0); i < 16; i++ {
			astTestMust(t, buf.push(r, i))
		}
		b.FailAfter(ordinal)
		body := buf.push(r, 16)
		astBVCharacterRawGraph(t, r, &buf, foreign)
		expected := uint32(1)
		if buf.raw != (util.Block{}) && !buf.rawConsumed {
			expected++
		}
		if buf.pending != (util.Block{}) {
			expected++
		}
		if r.Stats().LiveBlocks != expected || b.Used() != r.Stats().TotalBytes {
			t.Fatal("immediate actual buffer graph", ordinal, body, r.Stats(), expected, b.Used())
		}
		for i := uint32(0); i < 16; i++ {
			v, e := buf.read(r, i)
			astTestMust(t, e)
			if v != byte(i) {
				t.Fatal("preserved old source bytes", ordinal, i, v)
			}
		}
		if body != nil && buf.pos != 16 {
			t.Fatal("failed push published size", ordinal, buf.pos)
		}
		astTestMust(t, buf.sourceDestroy(r))
		astBVCharacterRawGraph(t, r, &buf, foreign)
		if buf.pending != (util.Block{}) && r.Stats().LiveBlocks != 2 {
			t.Fatal("source DT consumed unjoined NEW", ordinal, r.Stats())
		}
		b.FailAfter(-1)
		astTestMust(t, buf.cleanup(r))
		if r.Stats().LiveBlocks != 1 {
			t.Fatal("foreign owner consumed", ordinal, r.Stats())
		}
		astTestMust(t, r.Deallocate(&foreign))
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("ended graph", ordinal, b.Used())
		}
		if body == nil {
			return
		}
	}
	t.Fatal("actual growth horizon")
}

// Census verifies the explicit returned tokens; it never supplies an owner.
func astBVCharacterRawGraph(t *testing.T, r *util.Runtime, b *astBVCharBuffer16, foreign util.Block) {
	t.Helper()
	root := reflect.ValueOf(r).Elem()
	tokens := map[[3]uint64]bool{}
	add := func(v util.Block) {
		x := reflect.ValueOf(v)
		if testFieldBySourceName(x, "slot").Uint() == 0 {
			return
		}
		tokens[[3]uint64{testFieldBySourceName(x, "origin").Uint(), testFieldBySourceName(x, "slot").Uint(), testFieldBySourceName(x, "generation").Uint()}] = true
	}
	add(foreign)
	if !b.rawConsumed {
		add(b.raw)
	}
	add(b.pending)
	blocks := testFieldBySourceName(root, "blocks")
	live := 0
	for i := 0; i < blocks.Len(); i++ {
		v := blocks.Index(i)
		if !testFieldBySourceName(v, "live").Bool() {
			continue
		}
		key := [3]uint64{testFieldBySourceName(root, "origin").Uint(), uint64(i + 1), testFieldBySourceName(v, "generation").Uint()}
		if !tokens[key] {
			t.Fatal("anonymous reached character allocation", key)
		}
		live++
	}
	if live != len(tokens) {
		t.Fatal("exact character acquisition bijection", live, len(tokens))
	}
}
