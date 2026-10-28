package ast

// Source: src/ast/seq_decl_plugin.cpp:343-357,500-610;
// src/ast/seq_decl_plugin.h:126-175.

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
	if len(p.signatures) != 44 {
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
		sig := p.signatures[tc.index]
		if sig == nil {
			t.Fatal("missing signature", tc.index)
		}
		text, e := m.Symbols.String(sig.name)
		astTestMust(t, e)
		if string(text) != tc.name || sig.parameters != tc.params || uint32(len(sig.domain)) != tc.size || !sig.completed {
			t.Fatal("actual signature constructor fields", tc, string(text), sig)
		}
	}
	if p.signatures[26] != nil || p.signatures[42] != nil {
		t.Fatal("source unused signature slots must remain null", p.signatures)
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
				slots := len(p.signatures)
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
				if p.initialized != init || p.bodyCause != first || slots != len(p.signatures) {
					t.Fatal("published failed init prefix was replayed", ordinal, p)
				}
			}
		} else {
			complete = true
			if p == nil || !p.initialized || len(p.signatures) != 44 {
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

func TestASTSequenceNativeSignatureOwnershipAndFinalize(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterArrayPlugin()
	astTestMust(t, e)
	_, e = m.RegisterSequencePlugin()
	astTestMust(t, e)
	p := m.sequencePlugin
	// The actual seq.++ signature retains its common sort three times: once
	// for its range and once for each of its two identical domain arguments.
	s := p.signatures[2]
	if len(s.domain) != 2 || s.domain[0] != s.rangeSort || s.domain[1] != s.rangeSort {
		t.Fatal("actual concat signature domains", s)
	}
	before, e := m.GetRefCount(m.external(s.rangeSort))
	astTestMust(t, e)
	astTestMust(t, p.destroySignature(s, false))
	after, e := m.GetRefCount(m.external(s.rangeSort))
	astTestMust(t, e)
	if after != before-3 || len(s.domain) != 0 || s.rangeAcquired || !s.ended {
		t.Fatal("signature cleanup must consume all three actual references", before, after, s)
	}
	astTestMust(t, p.destroySignature(s, true))
	repeated, e := m.GetRefCount(m.external(s.rangeSort))
	astTestMust(t, e)
	if repeated != after {
		t.Fatal("completed signature cleanup replayed a DEC", after, repeated)
	}
	astTestMust(t, m.SourceFinalize())
	for s := p.owned; s != nil; s = s.next {
		if !s.contextEnded || len(s.domain) != 0 || s.rangeAcquired {
			t.Fatal("source finalization retained signature references", s)
		}
	}
	if m.sequencePlugin != p || p.header == 0 {
		t.Fatal("source-ended owner metadata must remain until owning close")
	}
	astTestMust(t, m.Close())
	if p.header != 0 || p.signatures != nil {
		t.Fatal("owning close retained signature storage")
	}
}
