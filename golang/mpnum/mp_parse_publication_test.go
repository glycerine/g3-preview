// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpq.cpp set(char const*) caller actions and lexical scopes.
package mpnum

import (
	"errors"
	"math/big"
	"strings"
	"testing"
)

func parserPublicationAudit(t *testing.T, m *RationalManager, roots ...*Int) {
	t.Helper()
	qopsAuditLiveOwnership(t, m, roots)
	// Actual numerical bytes live in physical sourceCellBacking, not one Words
	// payload per numeral. Check each named live object's actual extent access.
	refs := []numericCellRef{m.z.tmp[0], m.z.tmp[1], m.z.arg[0], m.z.arg[1], m.z.intMin.cell, m.z.two64.cell}
	for _, n := range roots {
		refs = append(refs, n.cell)
	}
	for _, n := range []*Int{&m.nTmp, &m.addTmp1, &m.addTmp2, &m.addMulTmp.num, &m.addMulTmp.den, &m.ltTmp1.num, &m.ltTmp1.den, &m.ltTmp2.num, &m.ltTmp2.den} {
		refs = append(refs, n.cell)
	}
	for _, ref := range refs {
		if ref == 0 || ref == m.z.failedRef {
			continue
		}
		c := m.z.cell(ref)
		if c.released {
			t.Fatal("named physical cell released", ref)
		}
		_, e := c.backing.readWord(m.z.allocator, 4)
		qopsMust(t, e)
	}
}
func parserPublicationSeed(t *testing.T, m *RationalManager, a *Rational, bigSeed bool) {
	t.Helper()
	a.num, a.den = NewInt(-17), NewInt(13)
	if bigSeed {
		words := make([]uint32, 65)
		words[64] = 1
		words[0] = 13
		qopsMust(t, m.z.SetLimbs(&a.den, words))
		words[0] = 17
		qopsMust(t, m.z.SetLimbs(&a.num, words))
		qopsMust(t, m.z.Neg(&a.num))
	}
}
func TestParserPublicationExceptionComponentPrefixes(t *testing.T) {
	cases := []struct {
		text     string
		reason   ParseErrorReason
		num, den int64
		oldDen   bool
	}{
		{"1/2e0", ParseMixedNotation, 1, 2, false}, {"-1/2E", ParseMixedNotation, 1, 2, false},
		{"1/0e0", ParseMixedNotation, 1, 0, false}, {"1e2/3", ParseMixedNotation, 1, 0, true},
		{"1e/3", ParseMixedNotation, 1, 0, true}, {"1e999999999999/0", ParseMixedNotation, 1, 0, true},
		{"1e4294967296", ParseExponentTooLarge, 1, 0, true}, {"1.25e4294967296", ParseExponentTooLarge, 1, 25, false},
		{"1/", ParseZeroDenominator, 1, 0, false}, {"1/00", ParseZeroDenominator, 1, 0, false}, {"/", ParseZeroDenominator, 0, 0, false},
	}
	for _, sync := range []bool{false, true} {
		for _, bigSeed := range []bool{false, true} {
			for _, c := range cases {
				m, budget := rationalManager(t, sync)
				a := NewRational()
				parserPublicationSeed(t, m, &a, bigSeed)
				oldNum, oldDen := a.num.cell, a.den
				denValue := roundPublicationDecode(t, m.z, &a.den)
				e := m.SetString(&a, c.text)
				parserPublicationAudit(t, m, &a.num, &a.den)
				var parsed *ParseError
				if !errors.As(e, &parsed) || parsed.Reason != c.reason {
					t.Fatal("actual source parser category", sync, bigSeed, c.text, e)
				}
				wantDen := big.NewInt(c.den)
				if c.oldDen {
					wantDen = denValue
				}
				if roundPublicationDecode(t, m.z, &a.num).Cmp(big.NewInt(c.num)) != 0 || roundPublicationDecode(t, m.z, &a.den).Cmp(wantDen) != 0 {
					t.Errorf("source accumulated caller fields hidden: sync=%v big=%v input=%q raw=%+v", sync, bigSeed, c.text, a)
				}
				if c.oldDen && a.den != oldDen {
					t.Error("source unchanged old denominator was replaced")
				}
				if oldNum != 0 && validNumericRef(m.z.cells, oldNum) {
					t.Error("source Reset did not release old BIG numerator")
				}
				if !c.oldDen && c.den == 0 {
					if _, readErr := m.String(&a); !errors.Is(readErr, ErrUninitializedRational) {
						t.Errorf("ordinary zero-den read boundary: %v", readErr)
					}
				}
				qopsMust(t, m.Del(&a))
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("parser exception cleanup", budget.Used())
				}
			}
		}
	}
}
func TestParserPublicationNoOldDenCopyBeforeReset(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, budget := rationalManager(t, sync)
		a := NewRational()
		parserPublicationSeed(t, m, &a, true)
		oldNum, oldDen := a.num.cell, a.den
		bank := roundPublicationFillCells(t, m.z)
		roots := append([]*Int{&a.num, &a.den}, bank...)
		budget.FailAfter(0)
		e := m.SetString(&a, "1e/3")
		parserPublicationAudit(t, m, roots...)
		var parsed *ParseError
		if !errors.As(e, &parsed) || parsed.Reason != ParseMixedNotation {
			t.Errorf("nonsource preparatory denominator allocation took precedence: %v", e)
		}
		if a.num != NewInt(1) || a.den != oldDen || validNumericRef(m.z.cells, oldNum) {
			t.Errorf("Reset/accumulation before exception hidden: %+v", a)
		}
		budget.FailAfter(-1)
		for i := len(roots) - 1; i >= 0; i-- {
			qopsMust(t, m.z.Del(roots[i]))
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal("armed exception cleanup", budget.Used())
		}
	}
}
func TestParserPublicationExplicitInitializerAndReparser(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, seed := range []string{"1/", "/", "raw Go zero"} {
			for _, fixture := range []struct{ text, want string }{{"2", "2"}, {".5", "1/2"}, {"3/4", "3/4"}, {"", "0"}} {
				m, budget := rationalManager(t, sync)
				a := NewRational()
				if seed == "raw Go zero" {
					a = Rational{}
				} else {
					e := m.SetString(&a, seed)
					parserPublicationAudit(t, m, &a.num, &a.den)
					var parsed *ParseError
					if !errors.As(e, &parsed) || parsed.Reason != ParseZeroDenominator {
						t.Fatal("source preparatory exception", e)
					}
					wantNum := int32(0)
					if seed == "1/" {
						wantNum = 1
					}
					if a.num != NewInt(wantNum) || a.den != NewInt(0) {
						t.Errorf("source invalidden publication missing before reparse: seed=%q raw=%+v", seed, a)
					}
				}
				// Explicit parsing initializes/reinitializes fields; ordinary arithmetic
				// remains guarded, without an initialization sidecar or lazy arithmetic.
				if a.den.val == 0 {
					if _, e := m.IsInt(&a); !errors.Is(e, ErrUninitializedRational) {
						t.Fatal("ordinary read accepted zero denominator", e)
					}
				}
				e := m.SetString(&a, fixture.text)
				parserPublicationAudit(t, m, &a.num, &a.den)
				if e != nil {
					t.Errorf("explicit source reparser refused: sync=%v seed=%q text=%q err=%v", sync, seed, fixture.text, e)
				} else if rationalString(t, m, &a) != fixture.want {
					t.Fatal(seed, fixture.text, fixture.want)
				}
				qopsMust(t, m.Del(&a))
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("reparser graph retained", budget.Used())
				}
			}
		}
	}
}

// parserPublicationPair describes one completed caller action, never a Cartesian
// product of independent component prefixes. This finite model is restricted to
// the four existing fault-sweep fixtures (positive digit magnitudes, exponent17).
type parserPublicationPair struct {
	num, den  *big.Int
	failedDen bool
}

func parserPublicationSweepPairs(text string) []parserPublicationPair {
	var pairs []parserPublicationPair
	put := func(n, d *big.Int) {
		pairs = append(pairs, parserPublicationPair{new(big.Int).Set(n), new(big.Int).Set(d), false})
	}
	normalize := func(n, d *big.Int) (*big.Int, *big.Int) {
		g := new(big.Int).GCD(nil, nil, n, d)
		nn := new(big.Int).Quo(n, g)
		put(nn, d)
		dd := new(big.Int).Quo(d, g)
		put(nn, dd)
		return nn, dd
	}
	negative := strings.HasPrefix(text, "-")
	if negative {
		text = text[1:]
	}
	n, d := new(big.Int), big.NewInt(13)
	put(n, d)
	i := 0
	for i < len(text) && text[i] != '/' && text[i] != '.' && text[i] != 'e' {
		n.Mul(n, big.NewInt(10))
		n.Add(n, big.NewInt(int64(text[i]-'0')))
		put(n, d)
		i++
	}
	if i == len(text) {
		d.SetInt64(1)
		put(n, d)
	} else {
		suffix := text[i]
		i++
		if suffix == '/' || suffix == '.' {
			d.SetInt64(0)
			put(n, d)
			digits := int64(0)
			for i < len(text) && text[i] != 'e' {
				d.Mul(d, big.NewInt(10))
				d.Add(d, big.NewInt(int64(text[i]-'0')))
				put(n, d)
				digits++
				i++
			}
			if suffix == '.' {
				scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(digits), nil)
				g := new(big.Int).GCD(nil, nil, d, scale)
				fn := new(big.Int).Quo(d, g)
				fd := new(big.Int).Quo(scale, g)
				d = big.NewInt(1)
				put(n, d)
				// Actual public Add first publishes DEN via Mul(1,fd,DEN).
				// Its BIG-copy allocation refusal owns failedRef with scalar sign +1.
				if fd.Cmp(big.NewInt(2147483647)) > 0 {
					pairs = append(pairs, parserPublicationPair{new(big.Int).Set(n), big.NewInt(1), true})
				}
				d = fd
				put(n, d)
				n = new(big.Int).Add(new(big.Int).Mul(n, fd), fn)
				put(n, d)
				n, d = normalize(n, d)
			}
		}
		if suffix == 'e' || i < len(text) && text[i] == 'e' {
			if suffix != 'e' {
				i++
			}
			expNegative := text[i] == '-'
			if expNegative {
				i++
			}
			exp := new(big.Int)
			exp.SetString(text[i:], 10)
			scale := new(big.Int).Exp(big.NewInt(10), exp, nil)
			if expNegative {
				d = new(big.Int).Mul(d, scale)
			} else {
				n = new(big.Int).Mul(n, scale)
			}
			put(n, d)
			n, d = normalize(n, d)
		}
	}
	if negative {
		n = new(big.Int).Neg(n)
		put(n, d)
	}
	normalize(n, d)
	return pairs
}
func parserPublicationSweepMatches(t *testing.T, m *RationalManager, a *Rational, text string) bool {
	t.Helper()
	if a.num.cell == m.z.failedRef {
		return false
	}
	n := roundPublicationDecode(t, m.z, &a.num)
	for _, p := range parserPublicationSweepPairs(text) {
		if n.Cmp(p.num) != 0 {
			continue
		}
		if p.failedDen {
			if a.den.cell == m.z.failedRef && a.den.val == 1 {
				return true
			}
		} else if a.den.cell != m.z.failedRef && roundPublicationDecode(t, m.z, &a.den).Cmp(p.den) == 0 {
			return true
		}
	}
	return false
}
