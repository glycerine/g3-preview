package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestFPABVWrapSourceExtractIdentityAndRefusals(t *testing.T) {
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	rw, err := NewFPARewriter(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, rw.Close()) }()
	wide, err := m.BitVectorSort(32)
	astTestMust(t, err)
	base, err := m.MkVar(0, wide)
	astTestMust(t, err)
	astTestPin(t, m, base)
	other, err := m.MkVar(1, wide)
	astTestMust(t, err)
	astTestPin(t, m, other)
	sigSort, err := m.BitVectorSort(23)
	astTestMust(t, err)
	plainSig, err := m.MkVar(2, sigSort)
	astTestMust(t, err)
	astTestPin(t, m, plainSig)
	part := func(high, low uint32, source ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkBitVectorExtract(high, low, source)
		astTestMust(t, e)
		return h
	}
	makeFP := func(sign, exponent, significand ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(rw.family, ast.FPAOpFP, nil, []ast.ASTHandle{sign, exponent, significand})
		astTestMust(t, e)
		return h
	}
	sign, exponent, significand := part(31, 31, base), part(30, 23, base), part(22, 0, base)
	fp := makeFP(sign, exponent, significand)
	astTestPin(t, m, fp)
	fpSort, err := m.GetSort(fp)
	astTestMust(t, err)
	decl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBVWrap, nil, []ast.ASTHandle{fpSort}, ast.ASTHandle{})
	astTestMust(t, err)
	astTestPin(t, m, decl)
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, invalid := range [][]ast.ASTHandle{nil, {fpSort, fpSort}, {boolSort}} {
		if _, e := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBVWrap, nil, invalid, ast.ASTHandle{}); e == nil {
			t.Fatalf("invalid bv_wrap declaration accepted: %v", invalid)
		}
	}
	wrapped, err := m.MkFamilyApp(rw.family, ast.FPAOpBVWrap, nil, []ast.ASTHandle{fp})
	astTestMust(t, err)
	width, err := m.BitVectorSize(wrapped)
	astTestMust(t, err)
	if width != 32 {
		t.Fatalf("BV wrap result width: %d", width)
	}
	out := astTestPin(t, m, base)
	status, err := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{fp}), out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || result != base {
		t.Fatalf("same-base extracts must return original BV: status=%v result=%v", status, result)
	}
	// Source checks the total width, not bit-range order or overlap.
	overlapping := makeFP(part(0, 0, base), part(7, 0, base), part(22, 0, base))
	status, err = rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{overlapping}), out)
	astTestMust(t, err)
	result, err = m.RefGet(out)
	astTestMust(t, err)
	if status != RewriteDone || result != base {
		t.Fatalf("source accepts overlapping extracts: status=%v result=%v", status, result)
	}
	widerSort, err := m.BitVectorSort(64)
	astTestMust(t, err)
	wider, err := m.MkVar(3, widerSort)
	astTestMust(t, err)
	astTestPin(t, m, wider)
	for _, tc := range []struct {
		name string
		fp   ast.ASTHandle
	}{
		{"different-base", makeFP(sign, exponent, part(22, 0, other))},
		{"not-extract", makeFP(sign, exponent, plainSig)},
		{"base-width", makeFP(part(31, 31, wider), part(30, 23, wider), part(22, 0, wider))},
	} {
		status, e := rw.MkAppCore(decl, ast.SliceArguments([]ast.ASTHandle{tc.fp}), out)
		astTestMust(t, e)
		if status != RewriteFailed {
			t.Fatalf("%s should refuse: %v", tc.name, status)
		}
	}
	rmSort, err := m.MkFamilySort(rw.family, 1, nil)
	astTestMust(t, err)
	rmDecl, err := m.MkFamilyFuncDecl(rw.family, ast.FPAOpBVWrap, nil, []ast.ASTHandle{rmSort}, ast.ASTHandle{})
	astTestMust(t, err)
	rm, err := m.MkFamilyApp(rw.family, 0, nil, nil)
	astTestMust(t, err)
	rmWrap, err := m.MkApp(rmDecl, []ast.ASTHandle{rm})
	astTestMust(t, err)
	width, err = m.BitVectorSize(rmWrap)
	astTestMust(t, err)
	if width != 3 {
		t.Fatalf("RM wrapper width: %d", width)
	}
	status, err = rw.MkAppCore(rmDecl, ast.SliceArguments([]ast.ASTHandle{rm}), out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatalf("RM wrapper has no mk_bvwrap reduction: %v", status)
	}
}
