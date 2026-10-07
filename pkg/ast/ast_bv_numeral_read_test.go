// Source: src/ast/bv_decl_plugin.cpp:778-790; src/ast/rewriter/bv_rewriter.h:36.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func bvFinishReadInvocation(t *testing.T, c *util.RationalServiceInvocation) {
	t.Helper()
	before := c.SourceReady()
	astTestMust(t, c.Cleanup(false))
	astTestMust(t, c.Cleanup(true))
	after := c.SourceReady()
	if (before == nil) != (after == nil) || before != nil && !errors.Is(after, before) {
		t.Fatal("read cleanup changed source history", before, after)
	}
	astTestMust(t, c.EndContext())
}

func TestASTBitVectorNumeralReadInvocationStoredValueAndUnmatchedOutputs(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	m.Budget().FailAfter(0)
	matched, body := m.BitVectorNumeralValueWithInvocation(x, nil, nil, nil)
	m.Budget().FailAfter(-1)
	if matched || body != nil {
		t.Fatal("unmatched read touched outputs/invocation", matched, body)
	}
	q := m.numerals
	out, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&out)) }()
	for _, literal := range []string{"0", "257", "-1", "340282366920938463463374607431768211457"} {
		h := bvRawOffsetTestNumeral(t, m, fid, s, literal, 8)
		var call util.RationalServiceInvocation
		width := uint32(777)
		matched, e = m.BitVectorNumeralValueWithInvocation(h, &out, &width, &call)
		astTestMust(t, e)
		got, e := q.String(&out)
		astTestMust(t, e)
		if !matched || got != literal || width != 8 {
			t.Fatal("stored numeral read normalized or changed width", got, width)
		}
		bvFinishReadInvocation(t, &call)
	}
}

func TestASTBitVectorNumeralReadCopiesBeforeReachedWidthError(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	astTestMust(t, q.SetInt64(&n, 257))
	p, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{p}, ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "read-missing-width"), nil, s, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&p))
	astTestMust(t, q.Del(&n))
	h, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, h)
	out, e := q.NewValue()
	astTestMust(t, e)
	var call util.RationalServiceInvocation
	width := uint32(77)
	matched, body := m.BitVectorNumeralValueWithInvocation(h, &out, &width, &call)
	if body == nil || matched || width != 77 {
		t.Fatal("missing width domain", matched, body, width)
	}
	got, e := q.String(&out)
	astTestMust(t, e)
	if got != "257" {
		t.Fatal("width guard moved before copy", got)
	}
	astTestMust(t, call.SourceReady())
	bvFinishReadInvocation(t, &call)
	astTestMust(t, q.Del(&out))
}

func TestASTBitVectorNumeralReadEveryActualCopyReservation(t *testing.T) {
	success, failures := false, 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := bvOffsetRuntimeTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			fid, e := m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			s, e := m.BitVectorSort(8)
			astTestMust(t, e)
			q := m.numerals
			n, e := q.NewValue()
			astTestMust(t, e)
			var words [129]uint32
			words[0], words[128] = 17, 3
			astTestMust(t, q.Manager.Z.SetLimbs(&n.Raw.Num, words[:]))
			astTestMust(t, q.Manager.Z.Neg(&n.Raw.Num))
			p, e := m.NewParameterRational(&n)
			astTestMust(t, e)
			bits, e := m.NewParameterUint(8)
			astTestMust(t, e)
			info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{p, bits}, ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDecl(astTestSymbol(t, m, "read-cold"), nil, s, info)
			astTestMust(t, e)
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestMust(t, m.DelParameter(&p))
			astTestMust(t, m.DelParameter(&bits))
			astTestMust(t, q.Del(&n))
			h, e := m.MkApp(d, nil)
			astTestMust(t, e)
			astTestPin(t, m, h)
			out, e := q.NewValue()
			astTestMust(t, e)
			var call util.RationalServiceInvocation
			width := uint32(777)
			before := m.bitVectorPlugin.numeralSerial
			m.Budget().FailAfter(ordinal)
			matched, body := m.BitVectorNumeralValueWithInvocation(h, &out, &width, &call)
			m.Budget().FailAfter(-1)
			if m.bitVectorPlugin.numeralSerial != before {
				t.Fatal("direct read introduced an AST numerical wrapper")
			}
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f util.Failure
					if !errors.As(body, &f) || f.Kind != util.FailureMemout {
						t.Fatal("unexpected copy refusal", body)
					}
				}
				if matched || width != 777 {
					t.Fatal("copy failure published width", matched, width)
				}
				child, _, _ := call.Causes()
				if child == nil || errors.Is(child, arena.ErrInvalidSlot) {
					t.Fatal("failed read lost actual copy child", child)
				}
			} else {
				success = true
				got, e := q.String(&out)
				astTestMust(t, e)
				want := new(big.Int).Neg(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(3), 4096), big.NewInt(17)))
				if !matched || width != 8 || got != want.String() {
					t.Fatal("completed stored read", matched, width)
				}
			}
			bvFinishReadInvocation(t, &call)
			astTestMust(t, q.CleanupValue(&out))
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("actual copy horizon", success, failures)
	}
}
