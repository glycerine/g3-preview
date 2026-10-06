// Source: src/ast/ast.h:1662,2033-2063; src/ast/ast.cpp:2503-2527.
// Licensed under the MIT license; source authors: Leonardo de Moura; Nikolaj Bjorner.
package g3

import (
	"strconv"
	"testing"
)

func basicFactoryKind(t *testing.T, m *ASTManager, h ASTHandle, kind int32, args ...ASTHandle) {
	t.Helper()
	d, e := m.GetDecl(h)
	astTestMust(t, e)
	k, e := m.GetDeclKind(d)
	astTestMust(t, e)
	n, e := m.GetNumArgs(h)
	astTestMust(t, e)
	if k != kind || n != uint32(len(args)) {
		t.Fatal("literal application", k, n, kind, len(args))
	}
	for i, w := range args {
		v, e := m.GetArg(h, uint32(i))
		astTestMust(t, e)
		if v != w {
			t.Fatal("argument order", i, v, w)
		}
	}
}
func TestASTBasicConvenienceFactoriesPreserveLiteralKindsAndInputs(t *testing.T) {
	m := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(1, bs)
	astTestMust(t, e)
	astTestPin(t, m, a)
	b, e := m.MkVar(2, bs)
	astTestMust(t, e)
	astTestPin(t, m, b)
	c, e := m.MkVar(3, bs)
	astTestMust(t, e)
	astTestPin(t, m, c)
	for _, v := range []struct {
		k    int32
		call func() (ASTHandle, error)
		args []ASTHandle
	}{
		{ASTOpIff, func() (ASTHandle, error) { return m.MkEq(a, b) }, []ASTHandle{a, b}},
		{ASTOpIff, func() (ASTHandle, error) { return m.MkIff(a, b) }, []ASTHandle{a, b}},
		{ASTOpOEq, func() (ASTHandle, error) { return m.MkOEq(a, b) }, []ASTHandle{a, b}},
		{ASTOpXor, func() (ASTHandle, error) { return m.MkXor(a, b) }, []ASTHandle{a, b}},
		{ASTOpIte, func() (ASTHandle, error) { return m.MkIte(a, b, c) }, []ASTHandle{a, b, c}},
		{ASTOpImplies, func() (ASTHandle, error) { return m.MkImplies(a, b) }, []ASTHandle{a, b}},
		{ASTOpInterp, func() (ASTHandle, error) { return m.MkInterp(a) }, []ASTHandle{a}},
		{ASTOpAnd, func() (ASTHandle, error) { return m.MkAnd3(a, b, c) }, []ASTHandle{a, b, c}},
		{ASTOpOr, func() (ASTHandle, error) { return m.MkOr3(a, b, c) }, []ASTHandle{a, b, c}},

		{ASTOpAnd, func() (ASTHandle, error) { return m.MkAndArgs([]ASTHandle{a}) }, []ASTHandle{a}},
		{ASTOpOr, func() (ASTHandle, error) { return m.MkOrArgs([]ASTHandle{a}) }, []ASTHandle{a}},
		{ASTOpDistinct, func() (ASTHandle, error) { return m.MkDistinct([]ASTHandle{a, b, c}) }, []ASTHandle{a, b, c}},
	} {
		h, e := v.call()
		astTestMust(t, e)
		astTestPin(t, m, h)
		basicFactoryKind(t, m, h, v.k, v.args...)
	}
	xor, e := m.MkXorArgs([]ASTHandle{a, b, c})
	astTestMust(t, e)
	astTestPin(t, m, xor)
	tail, e := m.GetArg(xor, 1)
	astTestMust(t, e)
	basicFactoryKind(t, m, xor, ASTOpXor, a, tail)
	basicFactoryKind(t, m, tail, ASTOpXor, b, c)
	us, e := m.MkUninterpretedSort(astTestSymbol(t, m, "U"), nil)
	astTestMust(t, e)
	astTestPin(t, m, us)
	x, e := m.MkVar(0, us)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, us)
	astTestMust(t, e)
	astTestPin(t, m, y)
	h, e := m.MkEq(x, y)
	astTestMust(t, e)
	astTestPin(t, m, h)
	basicFactoryKind(t, m, h, ASTOpEq, x, y)
	h, e = m.MkIte(a, x, y)
	astTestMust(t, e)
	astTestPin(t, m, h)
	basicFactoryKind(t, m, h, ASTOpIte, a, x, y)
	sr, e := m.GetSort(h)
	astTestMust(t, e)
	if sr != us {
		t.Fatal("term ITE range")
	}
	for _, v := range []bool{false, true} {
		h, e := m.MkBoolVal(v)
		astTestMust(t, e)
		w, e := m.MkFalse()
		if v {
			w, e = m.MkTrue()
		}
		astTestMust(t, e)
		if h != w {
			t.Fatal("cached Boolean value")
		}
	}
	for _, v := range []struct {
		k    int32
		call func() (ASTHandle, error)
	}{{ASTOpAnd, m.MkAndDecl}, {ASTOpOr, m.MkOrDecl}, {ASTOpNot, m.MkNotDecl}} {
		h, e := v.call()
		astTestMust(t, e)
		k, e := m.GetDeclKind(h)
		astTestMust(t, e)
		if k != v.k {
			t.Fatal("literal declaration")
		}
	}
}
func TestASTDistinctExpandedLiteralNestedPairOrderAndNoEagerInputRead(t *testing.T) {
	m := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	tr, e := m.MkTrue()
	astTestMust(t, e)
	for _, in := range [][]ASTHandle{nil, {{}}} {
		h, e := m.MkDistinctExpanded(in)
		astTestMust(t, e)
		if h != tr {
			t.Fatal("less than two returns true without args read")
		}
	}
	var args [7]ASTHandle
	for i := range args {
		args[i], e = m.MkVar(uint32(i), bs)
		astTestMust(t, e)
		astTestPin(t, m, args[i])
	}
	h, e := m.MkDistinctExpanded(args[:2])
	astTestMust(t, e)
	astTestPin(t, m, h)
	yes, eq, e := m.IsNot(h)
	astTestMust(t, e)
	if !yes {
		t.Fatal("binary distinct NOT")
	}
	basicFactoryKind(t, m, eq, ASTOpIff, args[0], args[1])
	h, e = m.MkDistinctExpanded(args[:])
	astTestMust(t, e)
	astTestPin(t, m, h)
	d, e := m.GetDecl(h)
	astTestMust(t, e)
	k, e := m.GetDeclKind(d)
	astTestMust(t, e)
	n, e := m.GetNumArgs(h)
	astTestMust(t, e)
	if k != ASTOpAnd || n != 21 {
		t.Fatal("all source pairs", k, n)
	}
	idx := uint32(0)
	for i := 0; i < 6; i++ {
		for j := i + 1; j < 7; j++ {
			v, e := m.GetArg(h, idx)
			astTestMust(t, e)
			yes, eq, e := m.IsNot(v)
			astTestMust(t, e)
			if !yes {
				t.Fatal("pair NOT", i, j)
			}
			basicFactoryKind(t, m, eq, ASTOpIff, args[i], args[j])
			idx++
		}
	}
}
func TestASTDistinctExpandedEveryActualReservationKeepsNamedGraph(t *testing.T) {
	reached := false
	for after := int64(0); after < 256; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			m := astTestManager(t)
			bs, e := m.MkBoolSort()
			astTestMust(t, e)
			var args [7]ASTHandle
			for i := range args {
				args[i], e = m.MkVar(uint32(i), bs)
				astTestMust(t, e)
				astTestPin(t, m, args[i])
			}
			other := m.budget.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(m.symbols)
			m.budget.FailAfter(after)
			h, e := m.MkDistinctExpanded(args[:])
			if m.budget.Used() != other+m.Stats().AllocatedBytes+astTestSymbolGraph(m.symbols) {
				t.Fatal("unnamed physical or cold prefix", after, e, m.budget.Used(), m.Stats())
			}
			for _, a := range args {
				rc, ce := m.GetRefCount(a)
				astTestMust(t, ce)
				if rc < 1 {
					t.Fatal("borrowed input lost", after)
				}
			}
			m.budget.FailAfter(-1)
			if e == nil {
				astTestPin(t, m, h)
				reached = true
			} else {
				if !h.IsNull() {
					t.Fatal("failure published result", after)
				}
				h, e = m.MkDistinctExpanded(args[:])
				astTestMust(t, e)
				astTestPin(t, m, h)
			}
		})
		if reached {
			if after == 0 {
				t.Fatal("campaign reached no reservation")
			}
			break
		}
	}
	if !reached {
		t.Fatal("bounded campaign did not complete")
	}
}

func TestASTBasicConvenienceInvalidBorrowAndSortFailure(t *testing.T) {
	m := astTestManager(t)
	other := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(1, bs)
	astTestMust(t, e)
	astTestPin(t, m, a)
	obs, e := other.MkBoolSort()
	astTestMust(t, e)
	foreign, e := other.MkVar(1, obs)
	astTestMust(t, e)
	astTestPin(t, other, foreign)
	before := m.budget.Used()
	h, e := m.MkEq(foreign, a)
	if e == nil || !h.IsNull() || m.budget.Used() != before {
		t.Fatal("foreign lhs read before factory", e, h)
	}
	copied := ASTManager{self: m}
	if _, e = copied.MkEq(a, a); e == nil {
		t.Fatal("foreign self guard")
	}
	us, e := m.MkUninterpretedSort(astTestSymbol(t, m, "term"), nil)
	astTestMust(t, e)
	astTestPin(t, m, us)
	term, e := m.MkVar(0, us)
	astTestMust(t, e)
	astTestPin(t, m, term)
	h, e = m.MkIff(a, term)
	if e == nil || !h.IsNull() {
		t.Fatal("source sort mismatch", e, h)
	}
	rc, e := m.GetRefCount(term)
	astTestMust(t, e)
	if rc != 1 {
		t.Fatal("failed app borrowed term RC", rc)
	}
	h, e = m.MkEq(term, term)
	astTestMust(t, e)
	astTestPin(t, m, h)
	basicFactoryKind(t, m, h, ASTOpEq, term, term)
}
