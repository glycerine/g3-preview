package ast

// Source: src/ast/array_decl_plugin.cpp:329-431;
// src/ast/array_decl_plugin.h:176-185.

import "testing"

func TestASTArraySetOriginalDeclarationsParametersFlagsAndAssociativeApps(t *testing.T) {
	m := astTestManager(t)
	fid, s, sorts := arrayDeclFixture(t, m, 2)
	for _, op := range []int32{ASTArrayOpSetUnion, ASTArrayOpSetIntersect, ASTArrayOpSetDifference, ASTArrayOpSetComplement, ASTArrayOpSetSubset} {
		domain := []ASTHandle{s, s}
		if op == ASTArrayOpSetComplement {
			domain = domain[:1]
		}
		d, e := m.MkFamilyFuncDecl(fid, op, nil, domain, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		r := m.declAt(mustInternal(t, m, d).slot)
		info, e := m.funcInfo(r.info)
		astTestMust(t, e)
		if op == ASTArrayOpSetUnion || op == ASTArrayOpSetIntersect {
			flags := ASTFuncFlags{LeftAssociative: true, RightAssociative: true, Commutative: true, Idempotent: true}
			if info.flags != flags || r.arity != 2 {
				t.Fatal("set flags/arity", op, info.flags, r.arity)
			}
			count := uint32(0)
			if op == ASTArrayOpSetUnion {
				count = 1
			}
			if info.count != count {
				t.Fatal("union-only sort parameter", op, info.count)
			}
			if count == 1 {
				p, e := m.infoParameter(info, 0)
				astTestMust(t, e)
				h, e := m.ParameterAST(p)
				astTestMust(t, e)
				if h != s {
					t.Fatal("retained union sort")
				}
			}
			for _, n := range []int{1, 3, 18} {
				domains := make([]ASTHandle, n)
				for i := range domains {
					domains[i] = s
				}
				again, e := m.MkFamilyFuncDecl(fid, op, nil, domains, ASTHandle{})
				astTestMust(t, e)
				if again != d {
					t.Fatal("canonical binary set declaration", op, n)
				}
			}
		} else if info.flags != (ASTFuncFlags{}) || info.count != 0 || r.arity != uint32(len(domain)) {
			t.Fatal("fixed set declaration", op)
		}
		want := s
		if op == ASTArrayOpSetSubset {
			want = sorts[2]
		}
		if m.external(r.rangeSort) != want {
			t.Fatal("set result sort", op)
		}
	}
	atomDecl, e := m.MkFuncDecl(astTestSymbol(t, m, "set-array"), nil, s, ASTFuncDeclInfo{})
	astTestMust(t, e)
	a, e := m.MkApp(atomDecl, nil)
	astTestMust(t, e)
	astTestPin(t, m, a)
	for _, op := range []int32{ASTArrayOpSetUnion, ASTArrayOpSetIntersect} {
		one, e := m.MkFamilyApp(fid, op, nil, []ASTHandle{a})
		astTestMust(t, e)
		astTestPin(t, m, one)
		count, e := m.GetNumArgs(one)
		astTestMust(t, e)
		if count != 1 {
			t.Fatal("unary set application")
		}
		three, e := m.MkFamilyApp(fid, op, nil, []ASTHandle{a, a, a})
		astTestMust(t, e)
		astTestPin(t, m, three)
		count, e = m.GetNumArgs(three)
		astTestMust(t, e)
		nested, e := m.GetArg(three, 1)
		astTestMust(t, e)
		nestedCount, e := m.GetNumArgs(nested)
		astTestMust(t, e)
		if count != 2 || nestedCount != 2 {
			t.Fatal("source right-associated set tree")
		}
	}
	empty, e := m.MkEmptySet(s)
	astTestMust(t, e)
	astTestPin(t, m, empty)
	full, e := m.MkFullSet(s)
	astTestMust(t, e)
	astTestPin(t, m, full)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	for _, pair := range [][2]ASTHandle{{empty, no}, {full, yes}} {
		v, e := m.GetArg(pair[0], 0)
		astTestMust(t, e)
		if v != pair[1] {
			t.Fatal("empty/full set value")
		}
		again, e := m.MkConstArray(s, pair[1])
		astTestMust(t, e)
		if again != pair[0] {
			t.Fatal("set constant identity")
		}
	}
}

func TestASTArraySetOriginalValidation(t *testing.T) {
	m := astTestManager(t)
	fid, s, sorts := arrayDeclFixture(t, m, 1)
	_, multi, _ := arrayDeclFixture(t, m, 2)
	other, e := m.MkUninterpretedSort(astTestSymbol(t, m, "set-non-bool-range"), nil)
	astTestMust(t, e)
	astTestPin(t, m, other)
	bp, e := m.NewParameterAST(sorts[0])
	astTestMust(t, e)
	op, e := m.NewParameterAST(other)
	astTestMust(t, e)
	badRange, e := m.MkFamilySort(fid, 0, []ASTParameter{bp, op})
	astTestMust(t, e)
	astTestPin(t, m, badRange)
	astTestMust(t, m.DelParameter(&bp))
	astTestMust(t, m.DelParameter(&op))
	for _, kind := range []int32{ASTArrayOpSetUnion, ASTArrayOpSetIntersect, ASTArrayOpSetDifference, ASTArrayOpSetComplement, ASTArrayOpSetSubset} {
		for _, domain := range [][]ASTHandle{nil, {sorts[0]}, {badRange}, {s, multi}, {s, badRange}} {
			if _, e = m.MkFamilyFuncDecl(fid, kind, nil, domain, ASTHandle{}); e != ErrASTDomain {
				t.Fatal("bad set domain", kind, domain, e)
			}
		}
	}
	for _, kind := range []int32{ASTArrayOpSetDifference, ASTArrayOpSetSubset} {
		for _, n := range []int{1, 3} {
			domain := make([]ASTHandle, n)
			for i := range domain {
				domain[i] = s
			}
			if _, e = m.MkFamilyFuncDecl(fid, kind, nil, domain, ASTHandle{}); e != ErrASTDomain {
				t.Fatal("binary set arity")
			}
		}
	}
	if _, e = m.MkFamilyFuncDecl(fid, ASTArrayOpSetComplement, nil, []ASTHandle{s, s}, ASTHandle{}); e != ErrASTDomain {
		t.Fatal("complement arity")
	}
}
