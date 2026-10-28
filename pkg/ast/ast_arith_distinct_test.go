package ast

// Source: src/ast/arith_decl_plugin.cpp:588-594,602-626.

import "testing"

func TestASTArithmeticOriginalNonzeroOffsetDistinctness(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&n)) }()
	for _, integer := range []bool{true, false} {
		index := 0
		if integer {
			index = 1
		}
		s := m.external(m.arithmeticPlugin.fields[index])
		x, e := m.MkConst(astTestSymbol(t, m, "offset-base"), s)
		astTestMust(t, e)
		astTestPin(t, m, x)
		y, e := m.MkConst(astTestSymbol(t, m, "offset-other"), s)
		astTestMust(t, e)
		astTestPin(t, m, y)
		values := []string{"0", "1", "-1", "18446744073709551617"}
		if !integer {
			values = append(values, "1/3", "-1/3")
		}
		for _, literal := range values {
			astTestMust(t, q.SetString(&n, literal))
			value, e := m.ArithmeticNumeral(&n, integer)
			astTestMust(t, e)
			astTestPin(t, m, value)
			for _, args := range [][]ASTHandle{{x, value}, {value, x}} {
				sum, e := m.MkFamilyApp(fid, 6, nil, args)
				astTestMust(t, e)
				astTestPin(t, m, sum)
				for _, pair := range [][2]ASTHandle{{sum, x}, {x, sum}} {
					m.Budget().FailAfter(0)
					got, e := m.AreDistinct(pair[0], pair[1])
					m.Budget().FailAfter(-1)
					astTestMust(t, e)
					if got != (literal != "0") {
						t.Fatal("source nonzero offset lemma", integer, literal, pair, got)
					}
				}
				got, e := m.AreDistinct(sum, y)
				astTestMust(t, e)
				if got {
					t.Fatal("distinctness requires exact base identity")
				}
			}
			three, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, value, value})
			astTestMust(t, e)
			astTestPin(t, m, three)
			got, e := m.AreDistinct(three, x)
			astTestMust(t, e)
			if got {
				t.Fatal("original lemma requires exactly two ADD arguments")
			}
		}
		sum, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, y})
		astTestMust(t, e)
		got, e := m.AreDistinct(sum, x)
		astTestMust(t, e)
		if got {
			t.Fatal("symbolic offset must not be assumed nonzero")
		}
	}
	// The original cheap predicate does not look through an inserted cast.
	// Mixed Real + Int therefore remains unknown to this particular check.
	x, e := m.MkConst(astTestSymbol(t, m, "mixed-offset-base"), m.external(m.arithmeticPlugin.fields[0]))
	astTestMust(t, e)
	astTestPin(t, m, x)
	astTestMust(t, q.SetInt64(&n, 1))
	one, e := m.ArithmeticNumeral(&n, true)
	astTestMust(t, e)
	sum, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{x, one})
	astTestMust(t, e)
	for _, pair := range [][2]ASTHandle{{sum, x}, {x, sum}} {
		got, e := m.AreDistinct(pair[0], pair[1])
		astTestMust(t, e)
		if got {
			t.Fatal("source offset predicate must not invent cast elimination")
		}
	}
}
