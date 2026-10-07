// Source: src/ast/bv_decl_plugin.cpp:360-403,530-576,584-628.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestASTBitVectorOriginalParameterizedDeclarations(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s8, e := m.BitVectorSort(8)
	astTestMust(t, e)
	s5, e := m.BitVectorSort(5)
	astTestMust(t, e)
	s1, e := m.BitVectorSort(1)
	astTestMust(t, e)
	for _, row := range []struct {
		k      int32
		params []int32
		domain []ASTHandle
		width  uint32
		count  uint32
	}{
		{37, nil, []ASTHandle{s8, s5, s1}, 14, 0},
		{37, []int32{99}, []ASTHandle{s8}, 8, 0},
		{38, []int32{4}, []ASTHandle{s8}, 12, 1},
		{39, []int32{0}, []ASTHandle{s8}, 8, 1},
		{40, []int32{7, 3}, []ASTHandle{s8}, 5, 2},
		// The source only checks low <= high and high < width, not low >= 0.
		{40, []int32{0, -1}, []ASTHandle{s8}, 2, 2},
		{41, []int32{3}, []ASTHandle{s8}, 24, 1},
		{41, []int32{-1}, []ASTHandle{s8}, ^uint32(0) - 7, 1},
		{48, nil, []ASTHandle{s8}, 8, 0},
		{49, []int32{-1, 100}, []ASTHandle{s8}, 8, 2},
	} {
		var ps []ASTParameter
		for _, v := range row.params {
			p, e := m.NewParameterInt(v)
			astTestMust(t, e)
			ps = append(ps, p)
		}
		d, e := m.MkFamilyFuncDecl(fid, row.k, ps, row.domain, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		wantRef, e := m.bitVectorPlugin.getSort(row.width)
		want := m.external(wantRef)
		astTestMust(t, e)
		result, e := m.GetRange(d)
		astTestMust(t, e)
		if result != want {
			t.Fatal("original result width", row, result, want)
		}
		arity, e := m.GetArity(d)
		astTestMust(t, e)
		if arity != uint32(len(row.domain)) {
			t.Fatal("original arity", row, arity)
		}
		r, e := m.internal(d)
		astTestMust(t, e)
		info, e := m.funcInfo(m.declAt(r.slot).info)
		astTestMust(t, e)
		if info.count != row.count || info.family != fid || info.kind != row.k || info.flags != (ASTFuncFlags{}) {
			t.Fatal("original metadata", row, info)
		}
		for i, want := range row.domain {
			got, e := m.GetDomain(d, uint32(i))
			astTestMust(t, e)
			if got != want {
				t.Fatal("domain", row, i, got, want)
			}
		}
		for _, p := range ps {
			astTestMust(t, m.DelParameter(&p))
		}
	}
}

func TestASTBitVectorFixedExprOverloadBypassesSortBuffer(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "bv-direct-expr"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	_, e = m.MkFamilyFuncDecl(fid, 4, nil, []ASTHandle{s, s}, ASTHandle{})
	astTestMust(t, e)
	args := make([]ASTHandle, 18)
	for i := range args {
		args[i] = x
	}
	// Cache is already warm. A wrong associative tail is checked directly before
	// the base sort-buffer allocation and before constructing an application.
	b, e := m.MkTrue()
	astTestMust(t, e)
	args[17] = b
	m.Budget().FailAfter(0)
	_, e = m.MkFamilyApp(fid, 4, nil, args)
	m.Budget().FailAfter(-1)
	if e == nil || e.Error() != "declared sorts do not match supplied sorts" {
		t.Fatal("direct expr overload", e)
	}
	// C++'s associative mismatch branch accepts a single matching argument.
	h, e := m.MkFamilyApp(fid, 4, nil, []ASTHandle{x})
	astTestMust(t, e)
	astTestPin(t, m, h)
	n, e := m.GetNumArgs(h)
	astTestMust(t, e)
	if n != 1 {
		t.Fatal("original singleton associative app", n)
	}
}

func TestASTBitVectorParameterizedValidationAndIgnoredConcatParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	for _, row := range []struct {
		k       int32
		params  []int32
		count   int
		message string
	}{
		{38, nil, 1, "invalid sign_extend application"},
		{38, []int32{-1}, 1, "invalid sign_extend application"},
		{39, []int32{2}, 2, "invalid zero_extend application"},
		{40, []int32{3, 4}, 1, "invalid extract application"},
		{40, []int32{8, 0}, 1, "invalid extract application"},
		{41, []int32{2}, 2, "repeat expects one argument"},
		{41, []int32{0}, 1, "repeat expects one nonzero integer parameter"},
		{48, nil, 2, "rotate left expects one argument"},
		{49, nil, 2, "rotate right expects one argument"},
	} {
		var ps []ASTParameter
		for _, v := range row.params {
			p, e := m.NewParameterInt(v)
			astTestMust(t, e)
			ps = append(ps, p)
		}
		domain := make([]ASTHandle, row.count)
		for i := range domain {
			domain[i] = s
		}
		_, e := m.MkFamilyFuncDecl(fid, row.k, ps, domain, ASTHandle{})
		if e == nil || e.Error() != row.message {
			t.Fatal(row, e)
		}
		for _, p := range ps {
			astTestMust(t, m.DelParameter(&p))
		}
	}
	d, e := m.MkFamilyFuncDecl(fid, 37, []ASTParameter{{}}, []ASTHandle{s, s}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	// Unknown operators return null after the original width read.
	unknown, e := m.MkFamilyFuncDecl(fid, 999, nil, []ASTHandle{s}, ASTHandle{})
	astTestMust(t, e)
	if !unknown.IsNull() {
		t.Fatal("source unknown operator must return null")
	}
}

func TestASTBitVectorParameterizedLargeSortIsUncached(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(2048)
	astTestMust(t, e)
	oldSize := m.bitVectorPlugin.sorts.size()
	d, e := m.MkFamilyFuncDecl(fid, 37, nil, []ASTHandle{s, s}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	if m.bitVectorPlugin.sorts.size() != oldSize {
		t.Fatal("large get_bv_sort must bypass sort cache")
	}
	r, e := m.GetRange(d)
	astTestMust(t, e)
	width, ok, e := m.bitVectorSortWidth(fid, r)
	astTestMust(t, e)
	if !ok || width != 4096 {
		t.Fatal("uncached large result width", ok, width)
	}
	node, e := m.internal(r)
	astTestMust(t, e)
	if m.nodeAt(node.slot).rc != 1 {
		t.Fatal("large sort must have declaration ownership only", m.nodeAt(node.slot).rc)
	}
}

func TestASTBitVectorParameterizedEveryFactoryReservation(t *testing.T) {
	for _, kind := range []int32{37, 38, 39, 40, 41, 48, 49} {
		success := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			func() {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				s, e := m.BitVectorSort(65)
				astTestMust(t, e)
				var values []int32
				domain := []ASTHandle{s}
				switch kind {
				case 37:
					domain = append(domain, s)
				case 38, 39:
					values = []int32{2}
				case 40:
					values = []int32{63, 1}
				case 41:
					values = []int32{2}
				case 48, 49:
					values = []int32{3}
				}
				var ps []ASTParameter
				for _, v := range values {
					p, e := m.NewParameterInt(v)
					astTestMust(t, e)
					ps = append(ps, p)
				}
				m.Budget().FailAfter(ordinal)
				h, body := m.MkFamilyFuncDecl(fid, kind, ps, domain, ASTHandle{})
				m.Budget().FailAfter(-1)
				if body != nil && !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal("factory reservation", kind, ordinal, body)
				}
				if body == nil {
					success = true
					astTestPin(t, m, h)
				}
				for _, p := range ps {
					astTestMust(t, m.DelParameter(&p))
				}
			}()
			if success {
				break
			}
		}
		if !success {
			t.Fatal("unreached successful parameterized factory", kind)
		}
	}
}

func TestASTBitVectorParameterizedExpressionFallbackKeepsSortBuffer(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "bv-fallback-expr"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	args := make([]ASTHandle, 18)
	for i := range args {
		args[i] = x
	}
	m.Budget().FailAfter(0)
	_, body := m.MkFamilyApp(fid, 37, nil, args)
	m.Budget().FailAfter(-1)
	if !errors.Is(body, arena.ErrInjectedFailure) {
		t.Fatal("concat must retain source sort-buffer spill", body)
	}
	h, e := m.MkFamilyApp(fid, 37, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, h)
	sort, e := m.GetSort(h)
	astTestMust(t, e)
	width, ok, e := m.bitVectorSortWidth(fid, sort)
	astTestMust(t, e)
	if !ok || width != 144 {
		t.Fatal("actual concat application", width, ok)
	}
}
