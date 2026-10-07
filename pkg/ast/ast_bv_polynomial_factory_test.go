// Source: src/ast/rewriter/poly_rewriter_def.h:47-53,70-74,78-124,127-138.
package ast

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorPolynomialVariadicFactorySourceCases(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	var args []ASTHandle
	for i := uint32(0); i < 40; i++ {
		h, e := m.MkVar(i, s)
		astTestMust(t, e)
		astTestPin(t, m, h)
		args = append(args, h)
	}
	input, e := m.MkFamilyApp(fid, 4, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, input)
	view, e := m.AppArgumentView(input)
	astTestMust(t, e)
	for _, row := range []struct {
		kind    int32
		unit    uint32
		factory func(ASTHandle, AppArguments) (ASTHandle, error)
	}{{4, 0, m.MkBitVectorPolynomialAdd}, {6, 1, m.MkBitVectorPolynomialMul}} {
		h, e := row.factory(s, SliceArguments(nil))
		astTestMust(t, e)
		astTestPin(t, m, h)
		q := m.numerals
		n, e := q.NewValue()
		astTestMust(t, e)
		var width uint32
		yes, e := m.BitVectorNumeralValue(h, &n, &width)
		astTestMust(t, e)
		got, e := q.String(&n)
		astTestMust(t, e)
		astTestMust(t, q.Del(&n))
		if !yes || width != 8 || got != fmt.Sprint(row.unit) {
			t.Fatal("empty polynomial identity", row.kind, got, width)
		}
		before := m.Stats()
		m.Budget().FailAfter(0)
		one, body := row.factory(ASTHandle{}, SliceArguments(args[:1]))
		m.Budget().FailAfter(-1)
		astTestMust(t, body)
		if one != args[0] || m.Stats() != before {
			t.Fatal("singleton touched sort/factory")
		}
		// The provided current sort is unused for a multi-operand application.
		many, e := row.factory(ASTHandle{}, view)
		astTestMust(t, e)
		astTestPin(t, m, many)
		k, e := m.GetDeclKind(many)
		astTestMust(t, e)
		count, e := m.GetNumArgs(many)
		astTestMust(t, e)
		if k != row.kind || count != 40 {
			t.Fatal("variadic polynomial application", k, count)
		}
		for i, want := range args {
			got, e := m.GetArg(many, uint32(i))
			astTestMust(t, e)
			if got != want {
				t.Fatal("borrowed argument order", i)
			}
		}
	}
	var absent *ASTManager
	got, e := absent.MkBitVectorPolynomialAdd(ASTHandle{}, SliceArguments([]ASTHandle{{}}))
	astTestMust(t, e)
	if !got.IsNull() {
		t.Fatal("singleton changed raw input")
	}
}

func TestASTBitVectorPolynomialCoefficientFactoryOriginalShortcuts(t *testing.T) {
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
	q := m.numerals
	c, e := q.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, q.Del(&c)) }()
	astTestMust(t, q.SetInt64(&c, 1))
	before := m.Stats()
	m.Budget().FailAfter(0)
	got, body := m.MkBitVectorPolynomialCoefficient(&c, ASTHandle{}, ASTHandle{})
	m.Budget().FailAfter(-1)
	astTestMust(t, body)
	if !got.IsNull() || m.Stats() != before {
		t.Fatal("unit coefficient read sort/argument")
	}
	astTestMust(t, q.SetInt64(&c, 7))
	zero, e := m.BitVectorNumeralUnsigned(0, 8)
	astTestMust(t, e)
	astTestPin(t, m, zero)
	var call ASTBitVectorNumeralInvocation
	got, e = m.MkBitVectorPolynomialCoefficientWithInvocation(&c, ASTHandle{}, zero, &call)
	astTestMust(t, e)
	if got != zero {
		t.Fatal("zero argument not returned")
	}
	f := call.receipt.frame
	if f == nil || !f.polynomialZero || !f.retired || !f.consumed[0] {
		t.Fatal("zero test bypassed copied numeral")
	}
	child, _, _ := f.children[2].Causes()
	if child != nil {
		t.Fatal("zero test did not enter original copy", child)
	}
	// Stored value 256 has zero modulo width eight, but the zero predicate
	// tests the copied stored value, so this helper still constructs a MUL.
	raw := bvRawOffsetTestNumeral(t, m, fid, s, "256", 8)
	got, e = m.MkBitVectorPolynomialCoefficient(&c, s, raw)
	astTestMust(t, e)
	astTestPin(t, m, got)
	k, e := m.GetDeclKind(got)
	astTestMust(t, e)
	arg, e := m.GetArg(got, 1)
	astTestMust(t, e)
	if k != 6 || arg != raw {
		t.Fatal("copied zero predicate normalized its operand", k)
	}
	// A coefficient that becomes one during numeral construction is not the
	// original c.is_one() shortcut. Keep the original multiplication node.
	astTestMust(t, q.SetInt64(&c, 257))
	got, e = m.MkBitVectorPolynomialCoefficient(&c, s, x)
	astTestMust(t, e)
	astTestPin(t, m, got)
	k, e = m.GetDeclKind(got)
	astTestMust(t, e)
	literal, e := m.GetArg(got, 0)
	astTestMust(t, e)
	one, e := m.BitVectorNumeralUnsigned(1, 8)
	astTestMust(t, e)
	astTestPin(t, m, one)
	if k != 6 || literal != one {
		t.Fatal("post-normalization one shortcut was added", k)
	}
	astTestMust(t, q.SetInt64(&c, 0))
	got, e = m.MkBitVectorPolynomialCoefficient(&c, s, x)
	astTestMust(t, e)
	astTestPin(t, m, got)
	k, e = m.GetDeclKind(got)
	astTestMust(t, e)
	if k != 6 {
		t.Fatal("coefficient-zero shortcut was added", k)
	}
}

func TestASTBitVectorPolynomialCoefficientEveryActualReservation(t *testing.T) {
	success, failures, zeroCopyFailure, constructorFailure := false, 0, false, false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
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
			p, e := m.NewParameterRational(&n)
			astTestMust(t, e)
			bits, e := m.NewParameterUint(8)
			astTestMust(t, e)
			info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{p, bits}, ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDecl(astTestSymbol(t, m, "polynomial-factory-cold"), nil, s, info)
			astTestMust(t, e)
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestMust(t, m.DelParameter(&bits))
			astTestMust(t, m.DelParameter(&p))
			astTestMust(t, q.Del(&n))
			arg, e := m.MkApp(d, nil)
			astTestMust(t, e)
			astTestPin(t, m, arg)
			c, e := q.NewValue()
			astTestMust(t, e)
			astTestMust(t, q.SetInt64(&c, 7))
			var call ASTBitVectorNumeralInvocation
			m.Budget().FailAfter(ordinal)
			got, body := m.MkBitVectorPolynomialCoefficientWithInvocation(&c, s, arg, &call)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f util.Failure
					if !errors.As(body, &f) || f.Kind != util.FailureMemout {
						t.Fatal("unexpected Runtime refusal", body)
					}
				}
				if !got.IsNull() {
					t.Fatal("failed factory returned an application")
				}
				if f := call.receipt.frame; f != nil && !f.retired {
					zeroCopyFailure = zeroCopyFailure || f.polynomialZero
					constructorFailure = constructorFailure || !f.polynomialZero
				}
				if _, e = m.MkBitVectorPolynomialCoefficientWithInvocation(&c, s, arg, &call); !errors.Is(e, ErrASTPending) {
					t.Fatal("failed factory call retried", e)
				}
			} else {
				success = true
				astTestPin(t, m, got)
				k, e := m.GetDeclKind(got)
				astTestMust(t, e)
				right, e := m.GetArg(got, 1)
				astTestMust(t, e)
				if k != 6 || right != arg {
					t.Fatal("completed coefficient factory", k)
				}
			}
			ready := m.BitVectorNumeralInvocationSourceReady(&call)
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			after := m.BitVectorNumeralInvocationSourceReady(&call)
			if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
				t.Fatal("cleanup changed source history")
			}
			astTestMust(t, q.CleanupValue(&c))
		})
		if success {
			break
		}
	}
	if !success || failures == 0 || !zeroCopyFailure || !constructorFailure {
		t.Fatal("factory refusal horizon", success, failures, zeroCopyFailure, constructorFailure)
	}
}

func TestASTBitVectorPolynomialLiteralConstructedBeforeSortGuard(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	var call ASTBitVectorNumeralInvocation
	got, body := m.MkBitVectorPolynomialAddWithInvocation(ASTHandle{}, SliceArguments(nil), &call)
	if !errors.Is(body, ErrASTHandle) || !got.IsNull() {
		t.Fatal("reached sort guard", got, body)
	}
	f := call.receipt.frame
	if f == nil || !f.polynomialLiteral || !f.complete[5] || !f.consumed[5] || f.placed[0] {
		t.Fatal("literal construction/source DT order", f)
	}
	child, _, _ := f.children[9].Causes()
	if child != nil {
		t.Fatal("integer literal constructor was not entered", child)
	}
	astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
}

func TestASTBitVectorPolynomialEmptyFactoriesEveryActualReservation(t *testing.T) {
	for _, mul := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			t.Run(fmt.Sprintf("%v/%d", mul, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				s, e := m.BitVectorSort(65)
				astTestMust(t, e)
				var call ASTBitVectorNumeralInvocation
				m.Budget().FailAfter(ordinal)
				var got ASTHandle
				var body error
				if mul {
					got, body = m.MkBitVectorPolynomialMulWithInvocation(s, SliceArguments(nil), &call)
				} else {
					got, body = m.MkBitVectorPolynomialAddWithInvocation(s, SliceArguments(nil), &call)
				}
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f util.Failure
						if !errors.As(body, &f) || f.Kind != util.FailureMemout {
							t.Fatal("unexpected identity factory refusal", body)
						}
					}
					if !got.IsNull() {
						t.Fatal("failed identity factory returned app")
					}
					if f := call.receipt.frame; f != nil && (!f.polynomialLiteral || f.retired) {
						t.Fatal("failed literal frame lost")
					}
				} else {
					success = true
					astTestPin(t, m, got)
					if f := call.receipt.frame; f == nil || !f.retired || !f.polynomialLiteral || !f.consumed[5] {
						t.Fatal("literal survived its return full expression")
					}
					q := m.numerals
					value, e := q.NewValue()
					astTestMust(t, e)
					var width uint32
					yes, e := m.BitVectorNumeralValue(got, &value, &width)
					astTestMust(t, e)
					v, e := q.String(&value)
					astTestMust(t, e)
					astTestMust(t, q.Del(&value))
					want := "0"
					if mul {
						want = "1"
					}
					if !yes || width != 65 || v != want {
						t.Fatal("empty factory result", mul, v, width)
					}
				}
				ready := m.BitVectorNumeralInvocationSourceReady(&call)
				astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
				after := m.BitVectorNumeralInvocationSourceReady(&call)
				if (ready == nil) != (after == nil) || ready != nil && !errors.Is(after, ready) {
					t.Fatal("identity cleanup changed source history")
				}
			})
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("empty factory horizon", mul, success, failures)
		}
	}
}
