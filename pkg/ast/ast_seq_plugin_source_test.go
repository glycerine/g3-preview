// Source: src/ast/seq_decl_plugin.cpp:343-357,500-610;
// src/ast/seq_decl_plugin.h:126-175.
package ast

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTSequenceSourceSetManagerAndRecursiveSignatureInitialization(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterArrayPlugin()
	astTestMust(t, e)
	family, e := m.RegisterSequencePlugin()
	astTestMust(t, e)
	p := m.sequencePlugin
	if !p.initialized || !p.published || p.family != family {
		t.Fatal("literal publication before init", p)
	}
	if p.char.slot == 0 || p.stringSort.slot == 0 || p.regex.slot == 0 {
		t.Fatal("actual set_manager sort fields", p)
	}
	char := m.sortAt(p.char.slot)
	info, e := m.sortInfo(char.info)
	astTestMust(t, e)
	if info.family != m.bitVectorPlugin.family {
		t.Fatal("actual cached BV8 dependency", *info)
	}
	for i, text := range [...]string{"String", "Char"} {
		s, e := m.Symbols.String(p.names[i])
		astTestMust(t, e)
		if string(s) != text {
			t.Fatal("source member symbol order", i, string(s), text)
		}
	}
	if len(p.signatures.words) == 0 || p.signatures.words[1] != 44 {
		t.Fatal("literal LAST_SEQ_OP vector resize", p.signatures)
	}
	for _, tc := range []struct {
		index        uint32
		name         string
		params, size uint32
	}{
		{0, "seq.unit", 1, 1}, {2, "seq.++", 1, 2}, {6, "seq.extract", 1, 3},
		{13, "re.+", 1, 1}, {22, "re.empty", 1, 0}, {27, "int.to.str", 0, 1}, {43, "_re.unroll", 0, 2},
	} {
		sig, e := p.signature(tc.index)
		astTestMust(t, e)
		text, e := m.Symbols.String(sig.name)
		astTestMust(t, e)
		state, e := m.vectorState(sig.domain)
		astTestMust(t, e)
		if string(text) != tc.name || sig.parameters != tc.params || state.size != tc.size || !sig.completed {
			t.Fatal("actual signature constructor fields", tc, string(text), sig, state)
		}
	}
	if p.signatures.words[2+26] != 0 || p.signatures.words[2+42] != 0 {
		t.Fatal("source unused signature slots must remain null", p.signatures.words)
	}
	// mk_sort calls init first, then uses the actual Char identity shortcut.
	param, e := m.NewParameterAST(m.external(p.char))
	astTestMust(t, e)
	h, e := m.MkFamilySort(family, 0, []ASTParameter{param})
	astTestMust(t, e)
	if h != m.external(p.stringSort) {
		t.Fatal("Char sequence is actual String field", h, p.stringSort)
	}
	astTestMust(t, m.DelParameter(&param))
}

func TestASTSequenceEveryActualConstructorAndSignatureReservation(t *testing.T) {
	_, q, _ := astTestOwners(t)
	failed, complete := false, false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		b := arena.NewBudget(1 << 28)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterArrayPlugin()
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterSequencePlugin()
		if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
			t.Fatal("immediate exact constructor graph", ordinal, b.Used(), m.Stats(), sy.Stats())
		}
		p := m.sequencePlugin
		if call != nil {
			failed = true
			if !errors.Is(call, arena.ErrInjectedFailure) {
				t.Fatal("source construction prefix", ordinal, call)
			}
			if p != nil {
				init := p.initialized
				first := p.bodyCause
				slots := p.signatures.sizeSequence()
				// Already-published source plugin causes registration to return without
				// repeating set_manager/init or overwriting a failed source invocation.
				b.FailAfter(-1)
				_, e = m.RegisterSequencePlugin()
				astTestMust(t, e)
				// Family-name symbol construction still runs and may grow the
				// interner before its duplicate lookup; the plugin body is skipped.
				if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
					t.Fatal("actual duplicate family preparation graph", ordinal, b.Used())
				}
				if p.initialized != init || p.bodyCause != first || slots != p.signatures.sizeSequence() {
					t.Fatal("published failed init prefix was replayed", ordinal, p)
				}
			}
		} else {
			complete = true
			if p == nil || !p.initialized || p.signatures.sizeSequence() != 44 {
				t.Fatal("all actual signatures reached", ordinal, p)
			}
		}
		b.FailAfter(-1)
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("own exact constructor roots", ordinal, b.Used())
		}
		if complete {
			t.Log("first full Sequence signature success", ordinal)
			break
		}
	}
	if !failed || !complete {
		t.Fatal("actual Sequence constructor horizon", failed, complete)
	}
}

func TestASTSequenceSourceFinalizeRetainsUnjoinedSignatureUntilContextEnd(t *testing.T) {
	_, q, _ := astTestOwners(t)
	reached := false
	for ordinal := int64(0); ordinal < 110; ordinal++ {
		b := arena.NewBudget(1 << 28)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterArrayPlugin()
		astTestMust(t, e)
		b.FailAfter(ordinal)
		_, call := m.RegisterSequencePlugin()
		if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
			t.Fatal("before lift constructor cold graph", ordinal, b.Used())
		}
		p := m.sequencePlugin
		var stopped *astSequenceSignature
		if p != nil {
			for s := p.owned; s != nil; s = s.next {
				if s.push.acquired && !s.push.published {
					stopped = s
					break
				}
			}
		}
		b.FailAfter(-1)
		if stopped != nil {
			if call == nil || stopped.completed || stopped.bodyCause == nil {
				t.Fatal("actual unfinished append constructor", ordinal, stopped, call)
			}
			invocation, node, body := stopped.push.invocation, stopped.push.node, stopped.bodyCause
			astTestMust(t, m.SourceFinalize())
			if m.sequencePlugin != p || stopped.header == 0 || !stopped.contextEnded || stopped.push.invocation != invocation || stopped.push.node != node || stopped.bodyCause != body {
				t.Fatal("actual source context end lost retained tuple", ordinal, p, stopped)
			}
			if b.Used() != m.Stats().AllocatedBytes+astTestSymbolGraph(sy) {
				t.Fatal("charged source-ended metadata", ordinal, b.Used(), m.Stats())
			}
			astTestMust(t, m.Close())
			if stopped.header != 0 || stopped.push.invocation != invocation || stopped.bodyCause != body {
				t.Fatal("own context-end history", ordinal, stopped)
			}
			if b.Used() != astTestSymbolGraph(sy) {
				t.Fatal("borrowed symbols must survive", ordinal, b.Used(), sy.Stats())
			}
			reached = true
		} else {
			astTestMust(t, m.Close())
		}
		astTestMust(t, sy.Close())
		if b.Used() != 0 {
			t.Fatal("closed owned metadata", ordinal, b.Used())
		}
		if reached {
			break
		}
	}
	if !reached {
		t.Fatal("no real unjoined signature INC frontier")
	}
}
