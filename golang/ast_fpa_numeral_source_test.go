// Source: src/ast/fpa_decl_plugin.cpp:64-95; src/ast/ast.cpp:76-85.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/mpnum"
	"testing"
)

func TestASTFloatingPointNumeralLiteralDeclarationsAndExternalDeletion(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: c.symbols, Numerals: c.numerals, Budget: b})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.Close()) }()
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, tc := range []struct {
		sig  int32
		exp  int64
		sign bool
		kind int32
		name string
	}{
		{1, 128, false, 8, "NaN"}, {0, 128, false, 6, "+oo"}, {0, 128, true, 7, "-oo"},
		{0, -127, false, 9, "+zero"}, {0, -127, true, 10, "-zero"}, {19, 3, false, 5, "fp.numeral"},
	} {
		sig := mpnum.NewInt(tc.sig)
		var value mpnum.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&value, 8, 24, tc.sign, tc.exp, &sig))
		app, e := p.mkNumeral(&value, g)
		astTestMust(t, e)
		decl := m.appAt(app.slot).decl
		astTestMust(t, e)
		d := m.declAt(decl.slot)
		name, e := m.symbols.String(d.name)
		astTestMust(t, e)
		info, e := m.info(d.info.slot, d.info.generation)
		astTestMust(t, e)
		if name != tc.name || info.family != p.family || info.kind != tc.kind {
			t.Fatal("literal numeral declaration", tc, name, *info)
		}
		if tc.kind == 5 {
			if info.count != 1 {
				t.Fatal("actual external parameter")
			}
			h, e := m.infoParameter(info, 0)
			astTestMust(t, e)
			param, e := m.parameter(h)
			astTestMust(t, e)
			if param.kind != ASTParameterExternal {
				t.Fatal("source external kind")
			}
			id := param.external
			astTestMust(t, m.incNode(app))
			_, e = m.decNode(app)
			astTestMust(t, e)
			if len(p.ids.free) != 1 || p.ids.free[0] != id {
				t.Fatal("actual attached external DEL did not recycle", id, p.ids.free)
			}
		}
		astTestMust(t, p.numeric.Del(&value))
	}
}

func TestASTFloatingPointNumeralDeclarationEveryReservationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := astTestManager(t)
		b := m.budget
		var e error
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterFloatingPointPlugin()
		astTestMust(t, e)
		p := m.floatingPointPlugin
		sig := mpnum.NewInt(19)
		var value mpnum.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&value, 8, 24, false, 3, &sig))
		base := b.Used() - m.Stats().AllocatedBytes - m.numerals.RuntimeOwner().Stats().TotalBytes
		b.FailAfter(ordinal)
		decl, call := p.mkNumeralDecl(&value, nil)
		astFloatingPointConstructorRawGraph(t, m.numerals, p)
		if b.Used() != base+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes {
			t.Fatal("actual before-lift declaration cold graph", ordinal, call, b.Used(), base+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes, m.Stats(), p.numeric.OwnedStorageBytes())
		}
		if call == nil {
			completed = true
			if decl.slot == 0 || p.ids.next != 1 {
				t.Fatal("actual source declaration publication", decl, p.ids.next)
			}
		}
		b.FailAfter(-1)
		astTestMust(t, p.numeric.Del(&value))
		astTestMust(t, m.Close())
		if completed {
			t.Log("first actual completed declaration prefix", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("declaration prefix horizon")
	}
}

func TestASTFloatingPointSortLiteralMinimumMessagesAndCheckOrder(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	for _, tc := range []struct {
		e, s    uint32
		message string
	}{
		{1, 1, "minimum number of significand bits is 1"},
		{1, 2, "minimum number of exponent bits is 2"},
	} {
		used := m.budget.Used()
		result, err := m.mkFloatSort(tc.e, tc.s)
		if err == nil || err.Error() != tc.message || result.slot != 0 || m.budget.Used() != used {
			t.Fatal("literal exception prefix", tc, result, err)
		}
	}
}

func TestASTFloatingPointNumeralReadUsesActualExternalRowAndSpecialSort(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, tc := range []struct {
		sig         int32
		exp         int64
		sign        bool
		expectedSig int32
	}{
		{19, 3, false, 19}, {0, 128, true, 0}, {0, 128, false, 0}, {1, 128, false, (1 << 23) - 1}, {0, -127, false, 0}, {0, -127, true, 0},
	} {
		sig := mpnum.NewInt(tc.sig)
		var source, out mpnum.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&source, 8, 24, tc.sign, tc.exp, &sig))
		app, e := p.mkNumeral(&source, nil)
		astTestMust(t, e)
		var receipt mpnum.ArbitraryFloatValueCopyReceipt
		found, e := p.isNumeral(app, &out, &receipt)
		astTestMust(t, e)
		if !found || out.EBits() != 8 || out.SBits() != 24 || out.Sign() != tc.sign || out.Exponent() != tc.exp {
			t.Fatal("source numeral branch", tc, out)
		}
		words, e := p.numeric.ValueWords(&out)
		astTestMust(t, e)
		if words[3] != uint32(tc.expectedSig) || words[4] != 0 || words[5] != 0 {
			t.Fatal("source numeral payload", tc, words)
		}
		astTestMust(t, p.numeric.CleanupValueCopy(&receipt, false))
		astTestMust(t, p.numeric.Del(&out))
		astTestMust(t, p.numeric.CleanupValueCopy(&receipt, true))
		astTestMust(t, p.numeric.Del(&source))
	}
	astTestMust(t, m.Close())
}

func TestASTFloatingPointNumeralReadUnmatchedErasesOutput(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	h, e := m.MkTrue()
	astTestMust(t, e)
	n, e := m.internal(h)
	astTestMust(t, e)
	var receipt mpnum.ArbitraryFloatValueCopyReceipt
	used := m.budget.Used()
	m.budget.FailAfter(0)
	found, e := p.isNumeral(n, nil, &receipt)
	if e != nil || found || receipt != (mpnum.ArbitraryFloatValueCopyReceipt{}) || m.budget.Used() != used {
		t.Fatal("unmatched source branch read output/receipt", found, e)
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	m.budget.FailAfter(-1)
	astTestMust(t, m.Close())
}
