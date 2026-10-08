// Source: src/ast/arith_decl_plugin.cpp:632-639.
package ast

import (
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestASTArithmeticNumeralReadStoredFlagAndOrderedCopy(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	fid, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	q := m.numerals
	v, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&v)) }()
	out, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&out)) }()
	astTestMust(t, q.SetString(&v, "18446744073709551617"))
	for _, flag := range []int32{0, 2, -1} {
		rp, e := m.NewParameterRational(&v)
		astTestMust(t, e)
		ip, e := m.NewParameterInt(flag)
		astTestMust(t, e)
		h, e := m.MkFamilyApp(fid, 0, []ASTParameter{rp, ip}, nil)
		astTestMust(t, e)
		astTestPin(t, m, h)
		astTestMust(t, m.DelParameter(&ip))
		astTestMust(t, m.DelParameter(&rp))
		var call util.RationalServiceInvocation
		integer := flag == 0
		yes, e := m.ArithmeticNumeralValueWithInvocation(fid, h, &out, &integer, &call)
		astTestMust(t, e)
		got, e := q.String(&out)
		astTestMust(t, e)
		if !yes || integer != (flag != 0) || got != "18446744073709551617" {
			t.Fatal("stored metadata", yes, integer, got)
		}
		astTestMust(t, call.Cleanup(false))
		astTestMust(t, call.Cleanup(true))
		astTestMust(t, call.EndContext())
	}
	// Raw metadata deliberately has a rational instead of integer flag. Copy of
	// parameter zero must finish before this reached flag error is reported.
	valid, e := m.ArithmeticNumeral(&v, true)
	astTestMust(t, e)
	astTestPin(t, m, valid)
	sort, e := m.GetSort(valid)
	astTestMust(t, e)
	rp, e := m.NewParameterRational(&v)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{rp, rp}, ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "read-bad-flag"), nil, sort, info)
	astTestMust(t, e)
	astTestPin(t, m, d)
	astTestMust(t, m.DelParameter(&rp))
	h, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, h)
	astTestMust(t, q.SetInt64(&out, 3))
	integer := false
	var call util.RationalServiceInvocation
	yes, e := m.ArithmeticNumeralValueWithInvocation(fid, h, &out, &integer, &call)
	if yes || e != ErrASTDomain || integer {
		t.Fatal("bad flag", yes, e, integer)
	}
	got, e := q.String(&out)
	astTestMust(t, e)
	if got != "18446744073709551617" {
		t.Fatal("flag checked before copy", got)
	}
	astTestMust(t, call.SourceReady())
	astTestMust(t, call.Cleanup(false))
	astTestMust(t, call.Cleanup(true))
	astTestMust(t, call.EndContext())
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	yes, e = m.ArithmeticNumeralValueWithInvocation(fid, x, nil, nil, nil)
	astTestMust(t, e)
	if yes {
		t.Fatal("nonmatch accessed outputs")
	}
}
