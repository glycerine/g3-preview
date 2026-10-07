// Source: src/ast/bv_decl_plugin.cpp:214-248,395-413,470-477,523-529,584-589.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestASTBitVectorOriginalConversionCachesAndParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	ints := m.external(m.bitVectorPlugin.fields[4])
	for _, width := range []int32{1, 8, 65} {
		bits, e := m.NewParameterInt(width)
		astTestMust(t, e)
		d, e := m.MkFamilyFuncDecl(fid, 57, []ASTParameter{bits}, []ASTHandle{ints}, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, d)
		rangeSort, e := m.GetRange(d)
		astTestMust(t, e)
		got, ok, e := m.bitVectorSortWidth(fid, rangeSort)
		astTestMust(t, e)
		if !ok || got != width {
			t.Fatal("int2bv original result width", width, ok, got)
		}
		back, e := m.MkFamilyFuncDecl(fid, 58, []ASTParameter{{}}, []ASTHandle{rangeSort}, ASTHandle{})
		astTestMust(t, e)
		astTestPin(t, m, back)
		result, e := m.GetRange(back)
		astTestMust(t, e)
		if result != ints {
			t.Fatal("bv2int range must be pinned Int sort")
		}
		ref, e := m.internal(back)
		astTestMust(t, e)
		info, e := m.funcInfo(m.declAt(ref.slot).info)
		astTestMust(t, e)
		if info.count != 0 || info.flags != (ASTFuncFlags{}) {
			t.Fatal("bv2int discards parameters", info)
		}
		m.Budget().FailAfter(0)
		again, e := m.MkFamilyFuncDecl(fid, 57, []ASTParameter{bits}, []ASTHandle{ints}, ASTHandle{})
		m.Budget().FailAfter(-1)
		astTestMust(t, e)
		if again != d {
			t.Fatal("conversion cache identity")
		}
		x, e := m.MkConst(astTestSymbol(t, m, "conversion-int-input"), ints)
		astTestMust(t, e)
		astTestPin(t, m, x)
		h, e := m.MkFamilyApp(fid, 57, []ASTParameter{bits}, []ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, h)
		h, e = m.MkFamilyApp(fid, 58, nil, []ASTHandle{h})
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, m.DelParameter(&bits))
	}
}

func TestASTBitVectorInt2BVOriginalASTWidthParserAndFirstDomainCache(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	ints := m.external(m.bitVectorPlugin.fields[4])
	s, e := m.BitVectorSort(11)
	astTestMust(t, e)
	x, e := m.MkConst(astTestSymbol(t, m, "width-parameter"), s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	p, e := m.NewParameterAST(x)
	astTestMust(t, e)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFamilyFuncDecl(fid, 57, []ASTParameter{p}, []ASTHandle{bs}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	// Source does not require an Int domain; the first domain is cached by width.
	got, e := m.GetDomain(d, 0)
	astTestMust(t, e)
	if got != bs {
		t.Fatal("int2bv first domain is preserved", got)
	}
	bits, e := m.NewParameterInt(11)
	astTestMust(t, e)
	again, e := m.MkFamilyFuncDecl(fid, 57, []ASTParameter{bits}, []ASTHandle{ints}, ASTHandle{})
	astTestMust(t, e)
	if again != d {
		t.Fatal("cache ignores later signature/parameter identity")
	}
	astTestMust(t, m.DelParameter(&bits))
	astTestMust(t, m.DelParameter(&p))
	// A non-BV expression parameter returns false from get_int2bv_size; the
	// domain overload then obtains the width from the first BV domain instead.
	truth, e := m.MkTrue()
	astTestMust(t, e)
	p, e = m.NewParameterAST(truth)
	astTestMust(t, e)
	s, e = m.BitVectorSort(12)
	astTestMust(t, e)
	d, e = m.MkFamilyFuncDecl(fid, 57, []ASTParameter{p}, []ASTHandle{s}, ASTHandle{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	result, e := m.GetRange(d)
	astTestMust(t, e)
	if result != s {
		t.Fatal("original false-parser fallback width")
	}
	astTestMust(t, m.DelParameter(&p))
}

func TestASTBitVectorConversionResizeBeforeArityValidation(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(65)
	astTestMust(t, e)
	bits, e := m.NewParameterInt(65)
	astTestMust(t, e)
	for _, kind := range []int32{57, 58} {
		ps := []ASTParameter{bits}
		domain := []ASTHandle{s, s}
		if kind == 57 {
			domain = nil
		}
		_, e := m.MkFamilyFuncDecl(fid, kind, ps, domain, ASTHandle{})
		want := "expecting one argument to int2bv"
		if kind == 58 {
			want = "expecting one argument to bv2int"
		}
		if e == nil || e.Error() != want {
			t.Fatal(kind, e)
		}
		cache := &m.bitVectorPlugin.fixed[kind]
		if cache.size() != 66 {
			t.Fatal("resize must precede arity rejection", kind, cache.size())
		}
		r, e := cache.at(65)
		astTestMust(t, e)
		if r.slot != 0 {
			t.Fatal("arity failure must precede declaration publication")
		}
	}
	astTestMust(t, m.DelParameter(&bits))
}

func TestASTBitVectorConversionEveryCacheReservationAndFinalize(t *testing.T) {
	for _, kind := range []int32{57, 58} {
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
				domain := []ASTHandle{s}
				if kind == 57 {
					domain[0] = m.external(m.bitVectorPlugin.fields[4])
				}
				p, e := m.NewParameterInt(65)
				astTestMust(t, e)
				m.Budget().FailAfter(ordinal)
				_, body := m.MkFamilyFuncDecl(fid, kind, []ASTParameter{p}, domain, ASTHandle{})
				m.Budget().FailAfter(-1)
				if body != nil && !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal(kind, ordinal, body)
				}
				if body == nil {
					success = true
				}
				astTestMust(t, m.DelParameter(&p))
				astTestMust(t, m.SourceFinalize())
				if m.bitVectorPlugin != nil {
					t.Fatal("conversion cache source owner retained")
				}
			}()
			if success {
				break
			}
		}
		if !success {
			t.Fatal("conversion reservation horizon", kind)
		}
	}
}

func TestASTBitVectorInt2BVParserTypeGuardsAndOwnedRationalCopy(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	ints := m.external(m.bitVectorPlugin.fields[4])
	for _, ps := range [][]ASTParameter{nil, {{}, {}}} {
		_, e := m.MkFamilyFuncDecl(fid, 57, ps, []ASTHandle{ints}, ASTHandle{})
		if e == nil || e.Error() != "int2bv expects one parameter" {
			t.Fatal("count guard", e)
		}
	}
	p, e := m.NewParameterAST(ints)
	astTestMust(t, e)
	_, e = m.MkFamilyFuncDecl(fid, 57, []ASTParameter{p}, []ASTHandle{ints}, ASTHandle{})
	if e == nil || e.Error() != "int2bv expects one integer parameter" {
		t.Fatal("non-expression AST parameter", e)
	}
	astTestMust(t, m.DelParameter(&p))
	zero, e := m.NewParameterInt(0)
	astTestMust(t, e)
	_, e = m.MkFamilyFuncDecl(fid, 57, []ASTParameter{zero}, []ASTHandle{ints}, ASTHandle{})
	if e == nil || e.Error() != "bit-vector size must be greater than zero" {
		t.Fatal("zero width", e)
	}
	if m.bitVectorPlugin.fixed[57].size() != 0 {
		t.Fatal("zero-width guard must precede resize")
	}
	astTestMust(t, m.DelParameter(&zero))
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&n, "36893488147419103231"))
	rational, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	before := m.bitVectorPlugin.numeralSerial
	_, e = m.MkFamilyFuncDecl(fid, 57, []ASTParameter{rational}, []ASTHandle{ints}, ASTHandle{})
	if e == nil || e.Error() != "int2bv expects one integer parameter" {
		t.Fatal("rational parser kind", e)
	}
	if m.bitVectorPlugin.numeralSerial != before+1 || !m.bitVectorPlugin.numeralPrimary.retired {
		t.Fatal("source rational copy and local destruction were erased")
	}
	astTestMust(t, m.DelParameter(&rational))
	astTestMust(t, m.numerals.Del(&n))
	astTestMust(t, m.SourceFinalize())
}

func TestASTBitVectorInt2BVRejectedRationalStillReachesNumericAllocation(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	n, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&n, "36893488147419103231"))
	rational, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	ints := m.external(m.bitVectorPlugin.fields[4])
	m.Budget().FailAfter(0)
	_, body := m.MkFamilyFuncDecl(fid, 57, []ASTParameter{rational}, []ASTHandle{ints}, ASTHandle{})
	m.Budget().FailAfter(-1)
	var source util.Failure
	if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &source) || source.Kind != util.FailureMemout) {
		t.Fatal("source parameter copy must precede kind rejection", body)
	}
	astTestMust(t, m.DelParameter(&rational))
	astTestMust(t, m.numerals.Del(&n))
}
