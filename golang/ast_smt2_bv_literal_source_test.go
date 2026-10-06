// Source: src/ast/ast_smt2_pp.cpp:165-233.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Independent bounded modular arithmetic and radix spelling; the production
// method must build the actual source format graph, then the actual pp engine
// transports it through the borrowed stream.
func TestASTSMT2BitVectorLiteralRadixAndSignedOracle(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	m, e := NewASTManager(ASTManagerConfig{Symbols: c.symbols, Numerals: c.numerals, Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, m.Close()) }()
	_, e = m.RegisterArithmeticPlugin()
	paramsMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	paramsMust(t, e)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	value, e := c.numerals.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, c.numerals.CleanupValue(&value)) }()
	for width := uint32(1); width <= 9; width++ {
		modulus := int64(1) << width
		for input := int64(-19); input <= 19; input += 7 {
			paramsMust(t, c.numerals.SetInt64(&value, input))
			h, e := m.BitVectorNumeral(&value, width)
			paramsMust(t, e)
			for _, signed := range []bool{false, true} {
				for _, radix := range []bool{false, true} {
					n := input % modulus
					if n < 0 {
						n += modulus
					}
					if signed && n >= modulus/2 {
						n -= modulus
					}
					negative := n < 0
					if negative {
						n = -n
					}
					var want string
					if !radix {
						want = "(_ bv" + strconv.FormatInt(n, 10) + " " + strconv.FormatUint(uint64(width), 10) + ")"
					} else {
						base, digits, prefix := 2, int(width), "#b"
						if width%4 == 0 {
							base, digits, prefix = 16, int(width/4), "#x"
						}
						body := strconv.FormatInt(n, base)
						want = prefix + strings.Repeat("0", digits-len(body)) + body
					}
					if negative {
						want = "(bvneg " + want + ")"
					}
					format, e := m.ppBitVectorLiteral(h, signed, radix, g)
					paramsMust(t, e)
					paramsMust(t, out.ResetText())
					paramsMust(t, m.PrintFormat(out, format, c, p, g))
					got, e := out.String()
					paramsMust(t, e)
					if got != want {
						t.Fatal("literal radix/sign policy", width, input, signed, radix, got, want)
					}
				}
			}
		}
	}
}

func astBVPrinterRawGraph(t *testing.T, m *ASTManager, input *mpnum.OwnedRational) {
	t.Helper()
	var values []*mpnum.OwnedRational
	if input != nil {
		values = append(values, input)
	}
	var blocks []memory.Block
	p := m.bitVectorPlugin
	for f := p.numeralFrames; f != nil; f = f.next {
		if f.self != f || f.owner != p {
			t.Fatal("foreign numeral caller")
		}
		for i := range f.values {
			if f.placed[i] && !f.consumed[i] {
				values = append(values, &f.values[i])
			}
		}
	}
	for f := p.literalFrames; f != nil; f = f.next {
		if f.self != f || f.owner != p {
			t.Fatal("foreign literal caller")
		}
		for i := range f.values {
			if f.placed[i] && !f.consumed[i] {
				values = append(values, &f.values[i])
			}
		}
		if !f.characters.rawConsumed {
			blocks = append(blocks, f.characters.raw)
		}
		blocks = append(blocks, f.characters.pending)
	}
	if fp := m.floatingPointPlugin; fp != nil {
		for f := fp.literalFrames; f != nil; f = f.next {
			if f.self != f || f.owner != fp {
				t.Fatal("foreign float caller")
			}
			for i := range f.bvValues {
				if f.bvPlaced[i] && !f.bvConsumed[i] {
					values = append(values, &f.bvValues[i])
				}
			}
		}
	}
	for _, page := range m.parameters {
		for i := range page {
			v := &page[i]
			if v.live && v.kind == ASTParameterRational {
				values = append(values, &v.rational)
			}
		}
	}
	astFloatingPointRawGraphWithBlocks(t, m.numerals, m.floatingPointPlugin, blocks, values...)
}

func TestASTSMT2BitVectorLiteralEveryReachedReservation(t *testing.T) {
	for _, signed := range []bool{false, true} {
		for _, radix := range []bool{false, true} {
			complete := false
			for ordinal := int64(0); ordinal < 128; ordinal++ {
				c, sy, _ := paramsTestOwner(t)
				b := c.budget
				catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
				paramsMust(t, e)
				cfg := gparamsTestConfig(b)
				cfg.Catalog = catalog
				g, e := NewGlobalParamsService(sy, c, cfg)
				paramsMust(t, e)
				paramsMust(t, g.InitializeSource())
				local, e := c.NewValue()
				paramsMust(t, e)
				p := &local
				t.Cleanup(func() { paramsMust(t, c.Del(p)); paramsMust(t, g.Close()); paramsMust(t, catalog.Close()) })
				m, e := NewASTManager(ASTManagerConfig{Symbols: c.symbols, Numerals: c.numerals, Budget: b})
				paramsMust(t, e)
				_, e = m.RegisterArithmeticPlugin()
				paramsMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				paramsMust(t, e)
				input, e := c.numerals.NewValue()
				paramsMust(t, e)
				paramsMust(t, c.numerals.SetString(&input, "18446744073709551617"))
				h, e := m.BitVectorNumeral(&input, 65)
				paramsMust(t, e)
				runtime := c.numerals.RuntimeOwner()
				before := b.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(c.symbols) - runtime.Stats().TotalBytes
				astBVPrinterRawGraph(t, m, &input)
				b.FailAfter(ordinal)
				format, body := m.ppBitVectorLiteral(h, signed, radix, g)
				astBVPrinterRawGraph(t, m, &input)
				if want := before + m.Stats().AllocatedBytes + astTestSymbolGraph(c.symbols) + runtime.Stats().TotalBytes; b.Used() != want {
					for f := m.bitVectorPlugin.literalFrames; f != nil; f = f.next {
						t.Log("literal diagnostic", f.header, len(f.buffer.state.heap), f.text.storageBytes(), f.text.streamEnded, f.text.returned, f.sourceDone, f.ownDone)
					}
					t.Log("components", before, m.Stats().AllocatedBytes, astTestSymbolGraph(c.symbols), runtime.Stats().TotalBytes)
					t.Fatal("before-lift actual literal cold graph", signed, radix, ordinal, body, b.Used(), want)
				}
				b.FailAfter(-1)
				if body == nil {
					out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
					paramsMust(t, e)
					paramsMust(t, m.PrintFormat(out, format, c, p, g))
					got, e := out.String()
					paramsMust(t, e)
					want := "(_ bv18446744073709551617 65)"
					if radix {
						want = "#b1" + strings.Repeat("0", 63) + "1"
					}
					if signed {
						want = "(bvneg (_ bv18446744073709551615 65))"
						if radix {
							want = "(bvneg #b0" + strings.Repeat("1", 64) + ")"
						}
					}
					if got != want {
						t.Fatal("independent BIG source numeral spelling", got, want)
					}
					paramsMust(t, out.Close())
					complete = true
					t.Log("first complete literal", signed, radix, ordinal)
				}
				paramsMust(t, m.Close())
				paramsMust(t, c.numerals.CleanupValue(&input))
				if complete {
					break
				}
			}
			if !complete {
				t.Fatal("literal actual reservation horizon", signed, radix)
			}
		}
	}
}

func TestASTSMT2BitVectorLiteralWarmInlinePrimaryDoesNotArchive(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	q := m.numerals
	value, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetInt64(&value, 19))
	h, e := m.BitVectorNumeral(&value, 8)
	astTestMust(t, e)
	first, e := m.ppBitVectorLiteral(h, false, true, nil)
	astTestMust(t, e)
	retained, e := m.format.NewRef(first)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.format.DelRef(&retained)); astTestMust(t, q.CleanupValue(&value)) }()
	// The first duplicate reaches actual node/info ID recycling. Warm that
	// source vector publication too, rather than assuming first construction
	// exercised the duplicate destructor's free-ID path.
	duplicate, e := m.ppBitVectorLiteral(h, false, true, nil)
	astTestMust(t, e)
	if duplicate != first {
		t.Fatal("actual duplicate format identity", duplicate, first)
	}
	before := m.budget.Used()
	m.budget.FailAfter(0)
	later, e := m.ppBitVectorLiteral(h, false, true, nil)
	after := m.budget.Used()
	if e != nil {
		for f := m.bitVectorPlugin.literalFrames; f != nil; f = f.next {
			t.Log("reached lexical buffers", f.header, f.characters.pos, f.characters.sourceDone, f.sourceDone, f.normalization.sourceDone)
			for i := range f.children {
				a, b, c := f.children[i].Causes()
				t.Log("actual child", i, a, b, c)
			}
		}
		t.Log("symbol table", m.symbols.Stats().Table)
	}
	m.budget.FailAfter(-1)
	if e != nil || later != first || after != before {
		t.Fatal("warm literal allocation-frontier audit", later, e, after, before)
	}
}

// The source first SASSERT can fail on a real live non-BV app. C continues
// through the actual default val/width path; release erases the provider read.
func TestASTSMT2BitVectorFalseNumeralActualActionAndErasure(t *testing.T) {
	for _, command := range []string{"C", "T"} {
		t.Run(command, func(t *testing.T) {
			_, d, _, input, streams := permutationActionFixture(t, command)
			b := d.budget
			sy, e := NewSymbolManager(SymbolConfig{Budget: b})
			paramsMust(t, e)
			q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
			paramsMust(t, e)
			c, e := NewParamsContext(sy, q, ParamsConfig{Budget: b, Text: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
			paramsMust(t, e)
			catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(b))
			paramsMust(t, e)
			cfg := gparamsTestConfig(b)
			cfg.Catalog = catalog
			g, e := NewGlobalParamsService(sy, c, cfg)
			paramsMust(t, e)
			paramsMust(t, g.BindNamespaceAssertions(d))
			m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
			paramsMust(t, e)
			defer func() {
				paramsMust(t, m.Close())
				paramsMust(t, g.Close())
				paramsMust(t, catalog.Close())
				paramsMust(t, c.Close())
				paramsMust(t, q.Close())
				paramsMust(t, sy.Close())
			}()
			_, e = m.RegisterArithmeticPlugin()
			paramsMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			paramsMust(t, e)
			h, e := m.MkTrue()
			paramsMust(t, e)
			got, body := m.ppBitVectorLiteral(h, false, false, g)
			state, e := input.State()
			paramsMust(t, e)
			data, e := os.ReadFile(streams.stderr.file.Name())
			paramsMust(t, e)
			if astSourceDebug {
				want := "ASSERTION VIOLATION\nFile: src/ast/ast_smt2_pp.cpp\nLine: 167\nu.is_numeral(t)\n" + debugSourcePrompt
				if string(data) != want || state.Position != 1 {
					t.Fatal("actual false site/action", string(data), state)
				}
				if command == "T" {
					var ex DefaultException
					if !errors.As(body, &ex) || !got.IsNull() {
						t.Fatal("actual thrown action", got, body)
					}
					return
				}
			} else if len(data) != 0 || state.Position != 0 {
				t.Fatal("compiled-erased assertion read provider", string(data), state)
			}
			paramsMust(t, body)
			want, e := m.format.FormatString("(_ bv0 1)")
			paramsMust(t, e)
			if got != want {
				t.Fatal("continued source default val and width", got, want)
			}
		})
	}
}
