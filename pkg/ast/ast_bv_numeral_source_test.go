// Source: src/ast/bv_decl_plugin.cpp:416-435,753-789,865-868.
package ast

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTBitVectorNumeralLiteralNormalizationAndRead(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	value, e := q.NewValue()
	astTestMust(t, e)
	out, e := q.NewValue()
	astTestMust(t, e)
	for bits := uint32(1); bits <= 8; bits++ {
		modulus := int64(1) << bits
		for input := int64(-257); input <= 257; input += 17 {
			astTestMust(t, q.SetInt64(&value, input))
			h, e := m.BitVectorNumeral(&value, bits)
			astTestMust(t, e)
			width := uint32(777)
			found, e := m.BitVectorNumeralValue(h, &out, &width)
			astTestMust(t, e)
			want := input % modulus
			if want < 0 {
				want += modulus
			}
			got, e := q.String(&out)
			astTestMust(t, e)
			if !found || width != bits || got != strconv.FormatInt(want, 10) {
				t.Fatal("independent modulo oracle", input, bits, found, width, got, want)
			}
		}
	}
	astTestMust(t, q.Del(&out))
	astTestMust(t, q.Del(&value))
}

func TestASTBitVectorNumeralUnmatchedLeavesArgumentsUnread(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	trueNode, e := m.MkTrue()
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	found, e := m.BitVectorNumeralValue(trueNode, nil, nil)
	if e != nil || found {
		t.Fatal("nonmatched branch must erase output reads", found, e)
	}
	m.BudgetField.FailAfter(-1)
}

// Every constructor reservation is inspected before lifting the fault. The
// explicitly supplied AST owns its placed parameter records and lexical NUMs;
// the original input remains caller-owned throughout.
func TestASTBitVectorBigNumeralEveryConstructorReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 28)
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
		astTestMust(t, e)
		q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		input, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&input, "18446744073709551617"))
		audit := func() {
			values := []*unumeral.OwnedRational{&input}
			for f := m.bitVectorPlugin.numeralFrames; f != nil; f = f.next {
				if f.self != f || f.owner != m.bitVectorPlugin {
					t.Fatal("foreign numeral frame")
				}
				for i := range f.values {
					if f.placed[i] && !f.consumed[i] {
						values = append(values, &f.values[i])
					}
				}
			}
			for _, page := range m.Parameters {
				if page != nil {
					for i := range page {
						v := &page[i]
						if v.Live && v.Kind == ASTParameterRational {
							values = append(values, &v.Rational)
						}
					}
				}
			}
			astFloatingPointConstructorRawGraph(t, q, nil, values...)
		}
		audit()
		oldCold := b.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(sy) - r.Stats().TotalBytes
		b.FailAfter(ordinal)
		h, body := m.BitVectorNumeral(&input, 65)
		audit()
		if want := oldCold + m.Stats().AllocatedBytes + astTestSymbolGraph(sy) + r.Stats().TotalBytes; b.Used() != want {
			t.Fatal("before-lift actual cold graph", ordinal, body, b.Used(), want)
		}
		b.FailAfter(-1)
		if body == nil {
			var out unumeral.OwnedRational
			out, e = q.NewValue()
			astTestMust(t, e)
			width := uint32(0)
			ok, e := m.BitVectorNumeralValue(h, &out, &width)
			if e == nil {
				got, x := q.String(&out)
				astTestMust(t, x)
				if !ok || width != 65 || got != "18446744073709551617" {
					t.Fatal("literal BIG output", ok, width, got)
				}
			}
			b.FailAfter(-1)
			astTestMust(t, e)
			astTestMust(t, q.CleanupValue(&out))
		} else {
			b.FailAfter(-1)
		}
		astTestMust(t, m.Close())
		astTestMust(t, q.CleanupValue(&input))
		astTestMust(t, q.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("ended owner graph", ordinal, b.Used())
		}
		if body == nil {
			t.Log("first completed source constructor", ordinal)
			return
		}
	}
	t.Fatal("constructor reservation horizon")
}

func TestASTBitVectorSignedNormalizationIndependentOracle(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	input, e := q.NewValue()
	astTestMust(t, e)
	for bits := uint32(1); bits <= 8; bits++ {
		modulus := int64(1) << bits
		for n := int64(-257); n <= 257; n += 17 {
			astTestMust(t, q.SetInt64(&input, n))
			out, e := m.normalizeBitVector(&input, bits, true, nil)
			astTestMust(t, e)
			want := n % modulus
			if want < 0 {
				want += modulus
			}
			if want >= modulus/2 {
				want -= modulus
			}
			got, e := q.String(&out)
			astTestMust(t, e)
			if got != strconv.FormatInt(want, 10) {
				t.Fatal("signed normalization", n, bits, got, want)
			}
			astTestMust(t, q.Del(&out))
		}
	}
	astTestMust(t, q.Del(&input))
}

func TestASTBitVectorSignedNormalizationEveryReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 28)
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
		astTestMust(t, e)
		q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
		astTestMust(t, e)
		sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
		astTestMust(t, e)
		m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
		astTestMust(t, e)
		_, e = m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		input, e := q.NewValue()
		astTestMust(t, e)
		astTestMust(t, q.SetString(&input, "18446744073709551617"))
		audit := func() {
			values := []*unumeral.OwnedRational{&input}
			for f := m.bitVectorPlugin.numeralFrames; f != nil; f = f.next {
				if f.self != f || f.owner != m.bitVectorPlugin {
					t.Fatal("foreign numeral frame")
				}
				for i := range f.values {
					if f.placed[i] && !f.consumed[i] {
						values = append(values, &f.values[i])
					}
				}
			}
			for _, page := range m.Parameters {
				if page != nil {
					for i := range page {
						v := &page[i]
						if v.Live && v.Kind == ASTParameterRational {
							values = append(values, &v.Rational)
						}
					}
				}
			}
			astFloatingPointConstructorRawGraph(t, q, nil, values...)
		}
		audit()
		oldCold := b.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(sy) - r.Stats().TotalBytes
		b.FailAfter(ordinal)
		out, body := m.normalizeBitVector(&input, 65, true, nil)
		if body == nil {
			astFloatingPointConstructorRawGraph(t, q, nil, &input, &out)
		} else {
			audit()
		}
		if want := oldCold + m.Stats().AllocatedBytes + astTestSymbolGraph(sy) + r.Stats().TotalBytes; b.Used() != want {
			t.Fatal("before-lift actual cold graph", ordinal, body, b.Used(), want)
		}
		b.FailAfter(-1)
		if body == nil {
			got, e := q.String(&out)
			astTestMust(t, e)
			if got != "-18446744073709551615" {
				t.Fatal("independent signed BIG normalization", got)
			}
			astTestMust(t, q.CleanupValue(&out))
		}
		astTestMust(t, m.Close())
		astTestMust(t, q.CleanupValue(&input))
		astTestMust(t, q.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("ended owner graph", ordinal, b.Used())
		}
		if body == nil {
			t.Log("first completed source constructor", ordinal)
			return
		}
	}
	t.Fatal("constructor reservation horizon")
}
