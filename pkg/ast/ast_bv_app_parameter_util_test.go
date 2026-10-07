// Source: src/ast/bv_decl_plugin.h:399-402,416-423;
// src/ast/bv_decl_plugin.cpp:880-885.
package ast

import (
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestASTBitVectorOriginalParameterizedUtilities(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	for _, test := range []struct {
		kind   int32
		values []int32
		width  uint32
		build  func() (ASTHandle, error)
	}{
		{40, []int32{6, 2}, 5, func() (ASTHandle, error) { return m.MkBitVectorExtract(6, 2, x) }},
		{39, []int32{3}, 11, func() (ASTHandle, error) { return m.MkBitVectorZeroExtend(3, x) }},
		{38, []int32{3}, 11, func() (ASTHandle, error) { return m.MkBitVectorSignExtend(3, x) }},
	} {
		var params []ASTParameter
		for _, v := range test.values {
			p, e := m.NewParameterInt(v)
			astTestMust(t, e)
			params = append(params, p)
		}
		want, e := m.MkFamilyApp(fid, test.kind, params, []ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, want)
		for _, p := range params {
			astTestMust(t, m.DelParameter(&p))
		}
		before := m.liveParameters
		got, e := test.build()
		astTestMust(t, e)
		width, e := m.BitVectorSize(got)
		astTestMust(t, e)
		if got != want || width != test.width || m.liveParameters != before {
			t.Fatal("parameter utility", test.kind, width, got, want)
		}
		if test.kind == 40 {
			var low, high uint32
			var base ASTHandle
			yes, e := m.BitVectorExtractValue(got, &low, &high, &base)
			astTestMust(t, e)
			if !yes || low != 2 || high != 6 || base != x {
				t.Fatal("extract parameter order", low, high, base)
			}
		}
	}
	// unsigned parameter construction preserves the source int32 conversion.
	// Low -1 is accepted by the existing extract factory, yielding width 8.
	got, e := m.MkBitVectorExtract(6, ^uint32(0), x)
	astTestMust(t, e)
	width, e := m.BitVectorSize(got)
	astTestMust(t, e)
	if width != 8 {
		t.Fatal("source unsigned-to-signed parameter", width)
	}
}

func TestASTBitVectorOriginalToIntUtility(t *testing.T) {
	m := astTestManager(t)
	arith, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(65)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	got, e := m.MkBitVectorToInt(x)
	astTestMust(t, e)
	astTestPin(t, m, got)
	want, e := m.MkFamilyApp(fid, 58, nil, []ASTHandle{x})
	astTestMust(t, e)
	intSort, e := m.MkFamilySort(arith, 1, nil)
	astTestMust(t, e)
	result, e := m.GetSort(got)
	astTestMust(t, e)
	if got != want || result != intSort {
		t.Fatal("bv2int utility identity/range")
	}
	before := m.Stats()
	params := m.liveParameters
	m.Budget().FailAfter(0)
	again, e := m.MkBitVectorToInt(x)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if again != got || m.Stats() != before || m.liveParameters != params {
		t.Fatal("warm scalar parameter allocated/acquired")
	}
}

func TestASTBitVectorUtilityEveryColdReservation(t *testing.T) {
	for _, kind := range []int32{38, 39, 40, 58} {
		success := false
		failures := 0
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			t.Run(strconv.Itoa(int(kind))+"/"+strconv.FormatInt(ordinal, 10), func(t *testing.T) {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				x, e := m.MkVar(0, sort)
				astTestMust(t, e)
				m.Budget().FailAfter(ordinal)
				var got ASTHandle
				var body error
				switch kind {
				case 38:
					got, body = m.MkBitVectorSignExtend(2, x)
				case 39:
					got, body = m.MkBitVectorZeroExtend(2, x)
				case 40:
					got, body = m.MkBitVectorExtract(63, 1, x)
				case 58:
					got, body = m.MkBitVectorToInt(x)
				}
				m.Budget().FailAfter(-1)
				f := &m.bitVectorUtilityParameters
				if f.active || f.count != 0 || f.values != ([2]astParameterRecord{}) {
					t.Fatal("scalar parameter scope retained")
				}
				if f.epoch != 0 {
					_, e = m.parameter(ASTParameter{m.origin, math.MaxUint32 - 2, f.epoch * 2})
					if e != ErrASTHandle {
						t.Fatal("expired stack token readable", e)
					}
				}
				if body != nil {
					if !errors.Is(body, arena.ErrInjectedFailure) {
						t.Fatal("unexpected refusal", body)
					}
					failures++
				} else {
					success = true
					if got.IsNull() {
						t.Fatal("successful constructor is null")
					}
				}
				// Fixture closure releases reached factory/cache/info/application roots
				// and checks the actual shared budget returns to zero.
			})
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("cold reservation horizon", kind, success, failures)
		}
	}
}

func TestASTBitVectorUtilityRejectsUnsignedNegativeExtension(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	for _, build := range []func(uint32, ASTHandle) (ASTHandle, error){m.MkBitVectorSignExtend, m.MkBitVectorZeroExtend} {
		_, e = build(^uint32(0), x)
		var source ASTException
		if !errors.As(e, &source) {
			t.Fatal("source negative extension guard", e)
		}
		if m.bitVectorUtilityParameters.active {
			t.Fatal("exception retained scalar parameters")
		}
	}
}
