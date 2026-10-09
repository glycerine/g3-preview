// Source: src/util/symbol.cpp:56-79; src/ast/format.cpp:147-152.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorRuntimeCStringInternActualBorrow(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	var buf astBVCharBuffer16
	text := "#b010101010101010101010101010101010101"
	for i := range text {
		astTestMust(t, buf.push(r, text[i]))
	}
	astTestMust(t, buf.push(r, 0))
	got, e := util.InternSymbolRuntimeCString(sy, r, buf.raw, 0, uint64(buf.pos))
	astTestMust(t, e)
	want, e := sy.Intern(text)
	astTestMust(t, e)
	if got != want {
		t.Fatal("same interning source engine", got, want)
	}
	b.FailAfter(0)
	again, e := util.InternSymbolRuntimeCString(sy, r, buf.raw, 0, uint64(buf.pos))
	astTestMust(t, e)
	if again != got {
		t.Fatal("warm actual borrowed identity")
	}
	b.FailAfter(-1)
	astTestMust(t, buf.sourceDestroy(r))
	astTestMust(t, buf.cleanup(r))
	astTestMust(t, sy.Close())
	astTestMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual context end", b.Used())
	}
}

func TestASTBitVectorRuntimeCharactersReachActualFormatString(t *testing.T) {
	m := astTestManager(t)
	r := m.numerals.RuntimeOwner()
	var buf astBVCharBuffer16
	text := "#b010101010101010101010101010101010101"
	for i := range text {
		astTestMust(t, buf.push(r, text[i]))
	}
	astTestMust(t, buf.push(r, 0))
	got, e := m.formatBVCharacters(&buf)
	astTestMust(t, e)
	want, e := m.FormatString(text)
	astTestMust(t, e)
	if got != want {
		t.Fatal("literal mk_string interning/registration", got, want)
	}
	astTestMust(t, buf.sourceDestroy(r))
	astTestMust(t, buf.cleanup(r))
	before := m.Stats()
	if _, e = m.formatBVCharacters(&buf); e == nil || m.Stats() != before {
		t.Fatal("ended source bytes reached formatter", e)
	}
}
