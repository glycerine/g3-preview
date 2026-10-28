package rewriter

// Source: src/util/util.h:131-139; src/ast/rewriter/bv_rewriter.cpp:684,814,863,909-921.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterSourceSaturatingUint64ShiftBoundaries(t *testing.T) {
	m, q, fid := bvOrRuntimeFixture(t)
	p := bvNotTestOwner(t, m, false, false)
	literal := func(value string, width uint32) ast.ASTHandle {
		t.Helper()
		v, ok := new(big.Int).SetString(value, 10)
		if !ok {
			t.Fatal(value)
		}
		return bvOrRawNumeral(t, m, q, fid, width, v)
	}
	large := literal("9223372036854775809", 65) // 2^63 + 1
	narrow := literal("9223372036854775809", 64)
	one := literal("1", 64)
	sixtyThree := literal("63", 64)
	sixtyFour := literal("64", 64)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, tc := range []struct {
		name, want string
		width      uint32
		call       func() (RewriteStatus, error)
	}{
		{"extract-64-bit-mask", "9223372036854775809", 64, func() (RewriteStatus, error) { return p.RewriteExtract(63, 0, large, out) }},
		{"extract-shift-by-64", "0", 1, func() (RewriteStatus, error) { return p.RewriteExtract(64, 64, large, out) }},
		{"logical-right-63", "1", 64, func() (RewriteStatus, error) { return p.RewriteLogicalShiftRight(narrow, sixtyThree, out) }},
		{"logical-left-63", "9223372036854775808", 64, func() (RewriteStatus, error) { return p.RewriteShiftLeft(one, sixtyThree, out) }},
		{"arithmetic-right-1", "13835058055282163712", 64, func() (RewriteStatus, error) { return p.RewriteArithmeticShiftRight(narrow, one, out) }},
		{"arithmetic-right-64", "18446744073709551615", 64, func() (RewriteStatus, error) { return p.RewriteArithmeticShiftRight(narrow, sixtyFour, out) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, e := tc.call()
			astTestMust(t, e)
			if st != RewriteDone {
				t.Fatal("source numeric rewrite status", st)
			}
			h, e := m.RefGet(out)
			astTestMust(t, e)
			v, e := q.NewValue()
			astTestMust(t, e)
			var width uint32
			yes, e := m.BitVectorNumeralValue(h, &v, &width)
			astTestMust(t, e)
			got, e := q.String(&v)
			astTestMust(t, e)
			astTestMust(t, q.Del(&v))
			if !yes || width != tc.width || got != tc.want {
				t.Fatal("source saturating shift", yes, width, got, tc.width, tc.want)
			}
		})
	}
}
