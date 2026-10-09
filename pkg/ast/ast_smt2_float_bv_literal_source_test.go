// Source: src/ast/ast_smt2_pp.cpp:270-291.
package ast

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestASTSMT2FloatDefaultBitVectorFields(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: c.SymbolsField, Numerals: c.NumeralsField, Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, m.Close()) }()
	_, e = m.RegisterArithmeticPlugin()
	paramsMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	paramsMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	paramsMust(t, e)
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	sig := util.NewInt(4)
	var v util.ArbitraryFloat
	paramsMust(t, m.floatingPointPlugin.numeric.SetComponents(&v, 8, 4, true, 3, &sig))
	node, e := m.floatingPointPlugin.mkNumeral(&v, nil)
	paramsMust(t, e)
	for _, tc := range []struct {
		radix bool
		want  string
	}{
		{false, "(fp (_ bv1 1) (_ bv130 8) (_ bv4 3))"},
		{true, "(fp #b1 #x82 #b100)"},
	} {
		f, e := m.ppFloatBitVectorLiteral(m.external(node), tc.radix, g)
		paramsMust(t, e)
		paramsMust(t, out.ResetText())
		paramsMust(t, m.PrintFormat(out, f, c, p, g))
		got, e := out.String()
		paramsMust(t, e)
		if got != tc.want {
			t.Fatal(tc.radix, got, tc.want)
		}
	}
	paramsMust(t, m.floatingPointPlugin.numeric.Del(&v))
}

func TestASTSMT2FloatDefaultEveryActualReservation(t *testing.T) {
	for _, big := range []bool{false, true} {
		for _, radix := range []bool{false, true} {
			success := false
			for ordinal := int64(0); ordinal < 160; ordinal++ {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				_, e = m.RegisterFloatingPointPlugin()
				astTestMust(t, e)
				p := m.floatingPointPlugin
				sig := util.NewInt(12345)
				bits := uint32(24)
				if big {
					sig = util.NewInt(0)
					owner, e := p.numeric.BorrowScopedInteger(&sig)
					astTestMust(t, e)
					astTestMust(t, owner.SetLimbs(&sig, []uint32{1, 0x80000000}))
					bits = 65
				}
				var v util.ArbitraryFloat
				astTestMust(t, p.numeric.SetComponents(&v, 8, bits, true, 3, &sig))
				node, e := p.mkNumeral(&v, nil)
				astTestMust(t, e)
				astTestMust(t, p.numeric.Del(&v))
				astTestMust(t, p.numeric.DeleteScopedInteger(&sig))
				runtime := m.numerals.RuntimeOwner()
				base := m.BudgetField.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.Symbols) - runtime.Stats().TotalBytes - astFormatterRationalColdBytes(t, m.numerals)
				astBVPrinterRawGraph(t, m, nil)
				m.BudgetField.FailAfter(ordinal)
				out, body := m.ppFloatBitVectorLiteral(m.external(node), radix, nil)
				astBVPrinterRawGraph(t, m, nil)
				if want := base + m.Stats().AllocatedBytes + astTestSymbolGraph(m.Symbols) + runtime.Stats().TotalBytes + astFormatterRationalColdBytes(t, m.numerals); m.BudgetField.Used() != want {
					t.Fatal("before-lift actual cold graph", radix, ordinal, body, m.BudgetField.Used(), want)
				}
				if body == nil {
					success = true
					if out.IsNull() {
						t.Fatal("source format publication")
					}
				}
				m.BudgetField.FailAfter(-1)
				astTestMust(t, p.numeric.Del(&v))
				astTestMust(t, m.Close())
				if success {
					t.Log("source completion horizon", radix, ordinal)
					break
				}
			}
			if !success {
				t.Fatal("no source completion", radix)
			}
		}
	}
}

// Source format.cpp:66-99 constructs a real declaration/application even when
// registration returns a duplicate. Native info-parameter slice growth does
// not add an arena allocation to that warm path.
func TestASTSMT2FloatDefaultWarmUsesNativeInfoStorage(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	sig := util.NewInt(4)
	var v util.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&v, 8, 4, false, 3, &sig))
	node, e := p.mkNumeral(&v, nil)
	astTestMust(t, e)
	first, e := m.ppFloatBitVectorLiteral(m.external(node), true, nil)
	astTestMust(t, e)
	held, e := m.Format.NewRef(first)
	astTestMust(t, e)
	second, e := m.ppFloatBitVectorLiteral(m.external(node), true, nil)
	astTestMust(t, e)
	if second != first {
		t.Fatal("same source format", first, second)
	}
	bias, e := p.numeric.ExponentBias(8)
	astTestMust(t, e)
	if bias != 127 {
		t.Fatal(bias)
	}
	prefix, e := m.Format.FormatString("(fp #b0")
	astTestMust(t, e)
	space, e := m.Format.FormatString(" ")
	astTestMust(t, e)
	if prefix.IsNull() || space.IsNull() {
		t.Fatal("cached source strings")
	}
	// The native profile can repeat the whole numeric formatter without a
	// reservation. Source debug numeral checks have their own allocations, so
	// verify the full float result normally and isolate the format-info path
	// under refusal in that profile.
	if ASTSourceDebug {
		third, e := m.ppFloatBitVectorLiteral(m.external(node), true, nil)
		astTestMust(t, e)
		if third != first {
			t.Fatal("warm float format identity", third, first)
		}
	}
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	defer m.BudgetField.FailAfter(-1)
	var third, want ASTHandle
	var body error
	if ASTSourceDebug {
		third, body = m.Format.FormatString("(fp #b0")
		want = prefix
	} else {
		third, body = m.ppFloatBitVectorLiteral(m.external(node), true, nil)
		want = first
	}
	m.BudgetField.FailAfter(-1)
	if body != nil || third != want || m.BudgetField.Used() != before {
		t.Fatal("warm native info storage", body, third, want, m.BudgetField.Used(), before)
	}
	astTestMust(t, m.Format.DelRef(&held))
	astTestMust(t, p.numeric.Del(&v))
	astTestMust(t, m.Close())
}
