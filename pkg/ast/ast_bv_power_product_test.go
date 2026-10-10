// Source: src/ast/rewriter/poly_rewriter_def.h:410-416.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func bvPowerProductRawApp(t *testing.T, m *ASTManager, fid FamilyID, s ASTHandle, k int32, name string, args []ASTHandle) ASTHandle {
	t.Helper()
	domain := make([]ASTHandle, len(args))
	for i := range domain {
		domain[i] = s
	}
	info, e := m.NewFuncDeclInfo(fid, k, nil, ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, name), domain, s, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	h, e := m.MkApp(d, args)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

func TestASTBitVectorPolynomialPowerProductCoefficientAndFallbackOne(t *testing.T) {
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
	y, e := m.MkVar(1, s)
	astTestMust(t, e)
	astTestPin(t, m, y)
	q := m.numerals
	coefficient, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&coefficient)) }()
	raw := bvPowerProductRawApp(t, m, fid, s, 0, "power-product-unread-value", nil)
	symbolic, e := m.MkBitVectorMul(x, y)
	astTestMust(t, e)
	astTestPin(t, m, symbolic)
	nary, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{raw, x, y})
	astTestMust(t, e)
	astTestPin(t, m, nary)
	unary, e := m.MkFamilyApp(fid, 6, nil, []ASTHandle{raw})
	astTestMust(t, e)
	astTestPin(t, m, unary)
	empty := bvPowerProductRawApp(t, m, fid, s, 6, "power-product-empty", nil)
	var call ASTBitVectorNumeralInvocation
	for _, input := range []ASTHandle{x, raw, symbolic, nary, unary, empty} {
		astTestMust(t, q.SetInt64(&coefficient, 99))
		before, e := m.GetRefCount(input)
		astTestMust(t, e)
		got, e := m.ReadBitVectorPolynomialPowerProductWithInvocation(input, &coefficient, &call)
		astTestMust(t, e)
		value, e := q.String(&coefficient)
		astTestMust(t, e)
		after, e := m.GetRefCount(input)
		astTestMust(t, e)
		if got != input || value != "1" || after != before {
			t.Fatal("fallback borrowed product/one", value, after, before)
		}
		f := call.receipt.frame
		if f == nil || !f.polynomialPowerProduct || !f.retired || !f.complete[0] || !f.consumed[0] {
			t.Fatal("one full-expression temporary")
		}
		for _, i := range []int{0, 1} {
			body, _, _ := f.children[i].Causes()
			if body != nil {
				t.Fatal("original constructor/copy missing", i, body)
			}
		}
	}
	for _, literal := range []string{"0", "257", "-257"} {
		num := bvRawOffsetTestNumeral(t, m, fid, s, literal, 8)
		input, e := m.MkBitVectorMul(num, x)
		astTestMust(t, e)
		astTestPin(t, m, input)
		before, e := m.GetRefCount(x)
		astTestMust(t, e)
		got, e := m.ReadBitVectorPolynomialPowerProductWithInvocation(input, &coefficient, &call)
		astTestMust(t, e)
		value, e := q.String(&coefficient)
		astTestMust(t, e)
		after, e := m.GetRefCount(x)
		astTestMust(t, e)
		if got != x || value != literal || after != before {
			t.Fatal("stored coefficient normalized or product retained", value)
		}
		if f := call.receipt.frame; f == nil || !f.retired || f.placed[0] {
			t.Fatal("matched branch constructed fallback one")
		}
	}
	// The original small-one full expression needs no numeric allocation.
	// The default entry shares its actual constructor/copy/DT algorithm.
	astTestMust(t, q.SetInt64(&coefficient, 99))
	before := m.Stats()
	m.Budget().FailAfter(0)
	got, body := m.ReadBitVectorPolynomialPowerProduct(x, &coefficient)
	m.Budget().FailAfter(-1)
	astTestMust(t, body)
	one, e := q.IsOne(&coefficient)
	astTestMust(t, e)
	if got != x || !one || m.Stats() != before {
		t.Fatal("small fallback allocated or changed result", got, one)
	}
}

func TestASTBitVectorPolynomialPowerProductFallbackTemporaryBeforeReceiverGuard(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, s)
	astTestMust(t, e)
	astTestPin(t, m, x)
	var call ASTBitVectorNumeralInvocation
	got, body := m.ReadBitVectorPolynomialPowerProductWithInvocation(x, nil, &call)
	if body == nil || !got.IsNull() {
		t.Fatal("missing receiver accepted", got, body)
	}
	f := call.receipt.frame
	if f == nil || !f.polynomialPowerProduct || !f.complete[0] || !f.consumed[0] {
		t.Fatal("one temporary constructed after receiver check")
	}
	ctor, _, _ := f.children[0].Causes()
	if ctor != nil {
		t.Fatal("one constructor not entered", ctor)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
}

func TestASTBitVectorPolynomialPowerProductEveryActualReadReservation(t *testing.T) {
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
			d, e := m.MkFuncDecl(astTestSymbol(t, m, "power-product-cold"), nil, s, info)
			astTestMust(t, e)
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestMust(t, m.DelParameter(&p))
			astTestMust(t, m.DelParameter(&bits))
			astTestMust(t, q.Del(&n))
			num, e := m.MkApp(d, nil)
			astTestMust(t, e)
			astTestPin(t, m, num)
			x, e := m.MkVar(0, s)
			astTestMust(t, e)
			astTestPin(t, m, x)
			input, e := m.MkBitVectorMul(num, x)
			astTestMust(t, e)
			astTestPin(t, m, input)
			coefficient, e := q.NewValue()
			astTestMust(t, e)
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			got, body := m.ReadBitVectorPolynomialPowerProductWithInvocation(input, &coefficient, &call)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f umemory.Failure
					if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
						t.Fatal("unexpected copy refusal", body)
					}
				}
				if !got.IsNull() {
					t.Fatal("failed read returned product")
				}
				f := call.receipt.frame
				if f == nil || !f.polynomialPowerProduct || f.retired || f.placed[0] {
					t.Fatal("read refusal entered fallback or lost context")
				}
				child, _, _ := f.children[2].Causes()
				if child == nil || errors.Is(child, arena.ErrInvalidSlot) {
					t.Fatal("read failed without exact assignment child", child)
				}
				if _, e = m.ReadBitVectorPolynomialPowerProductWithInvocation(input, &coefficient, &call); !errors.Is(e, ErrASTPending) {
					t.Fatal("failed call retried", e)
				}
			} else {
				success = true
				v, e := q.String(&coefficient)
				astTestMust(t, e)
				want := new(big.Int).Neg(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(3), 4096), big.NewInt(17)))
				if got != x || v != want.String() {
					t.Fatal("large stored coefficient read changed")
				}
			}
			ready := m.BitVectorNumeralInvocationSourceReady(&call)
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			after := m.BitVectorNumeralInvocationSourceReady(&call)
			if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
				t.Fatal("cleanup changed source history")
			}
			astTestMust(t, q.CleanupValue(&coefficient))
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("actual read horizon", success, failures)
	}
}
