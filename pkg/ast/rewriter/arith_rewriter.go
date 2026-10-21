// Source: src/ast/rewriter/arith_rewriter.cpp;
// src/ast/rewriter/arith_rewriter.h.
package rewriter

import (
	"errors"
	"math/big"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

const (
	arithOpNum    int32 = 0
	arithOpLE     int32 = 2
	arithOpGE     int32 = 3
	arithOpLT     int32 = 4
	arithOpGT     int32 = 5
	arithOpAdd    int32 = 6
	arithOpSub    int32 = 7
	arithOpUMinus int32 = 8
	arithOpMul    int32 = 9
	arithOpDiv    int32 = 10
	arithOpIDiv   int32 = 11
	arithOpRem    int32 = 12
	arithOpMod    int32 = 13
	arithOpToReal int32 = 14
	arithOpToInt  int32 = 15
	arithOpIsInt  int32 = 16
	arithOpAbs    int32 = 17
	arithOpPower  int32 = 18
)

// ArithRewriter is the concrete arithmetic theory reducer. This first source
// slice owns configuration and the exact rational scalar rewrite paths; the
// expression-polynomial and transcendental paths are added independently.
type ArithRewriter struct {
	self                                      *ArithRewriter
	manager                                   *ast.ASTManager
	family                                    ast.FamilyID
	header                                    uint64
	closed                                    bool
	expandPower, mulToPower, expandTan        bool
	arithLHS, arithIneqLHS, gcdRounding       bool
	elimToReal, pushToReal, algebraicSimplify bool
	elimRem, eq2ineq, sortSums                bool
	flat                                      bool
	maxDegree                                 uint32
}

func NewArithRewriter(m *ast.ASTManager, c *uparams.ParamsContext, g *uparams.GlobalParamsService, local *uparams.ParamsRef) (*ArithRewriter, error) {
	if m == nil || c == nil || g == nil || c.SymbolsField != m.Symbols || g.Context != c {
		return nil, uparams.ErrParamsOwner
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	n := uint64(reflect.TypeOf(ArithRewriter{}).Size())
	if err := m.Budget().Reserve(n); err != nil {
		return nil, err
	}
	r := &ArithRewriter{manager: m, header: n}
	r.self = r
	var err error
	r.family, err = m.RegisterArithmeticPlugin()
	if err == nil {
		err = r.updateParams(c, g, local)
	}
	return r, err
}

func (r *ArithRewriter) open() error {
	if r == nil || r.self != r || r.closed || r.manager == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

func (r *ArithRewriter) updateParams(c *uparams.ParamsContext, g *uparams.GlobalParamsService, local *uparams.ParamsRef) (result error) {
	p, err := NewArithRewriterParams(c, g, local)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, p.Close()) }()
	read := func(dst *bool, f func() (bool, error)) error {
		*dst, err = f()
		return err
	}
	for _, item := range []struct {
		dst *bool
		get func() (bool, error)
	}{
		{&r.arithLHS, p.ArithLHS}, {&r.arithIneqLHS, p.ArithIneqLHS},
		{&r.gcdRounding, p.GCDRounding}, {&r.elimToReal, p.ElimToReal},
		{&r.pushToReal, p.PushToReal}, {&r.algebraicSimplify, p.AlgebraicNumberEvaluator},
		{&r.expandPower, p.ExpandPower}, {&r.mulToPower, p.MulToPower},
		{&r.elimRem, p.ElimRem}, {&r.expandTan, p.ExpandTan},
		{&r.eq2ineq, p.Eq2Ineq}, {&r.sortSums, p.SortSums},
	} {
		if err = read(item.dst, item.get); err != nil {
			return err
		}
	}
	r.maxDegree, err = p.MaxDegree()
	if err != nil {
		return err
	}
	base, err := NewRewriterParams(c, g, local)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, base.Close()) }()
	r.flat, err = base.Flat()
	return err
}

func (r *ArithRewriter) Close() error {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	r.manager.Budget().Release(r.header)
	r.header = 0
	r.closed = true
	return nil
}

func (r *ArithRewriter) numeral(h ast.ASTHandle) (*big.Rat, bool, bool, error) {
	text, integer, matched, err := r.manager.ArithmeticNumeralText(h)
	if err != nil || !matched {
		return nil, integer, matched, err
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, integer, true, ast.ErrASTDomain
	}
	return value, integer, true, nil
}

func (r *ArithRewriter) makeNumeral(value *big.Rat, integer bool) (result ast.ASTHandle, err error) {
	service := r.manager.NumeralService()
	if service == nil || value == nil {
		return result, ast.ErrASTHandle
	}
	n, err := service.NewValue()
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, service.Del(&n)) }()
	if err = service.SetString(&n, value.RatString()); err != nil {
		return result, err
	}
	return r.manager.ArithmeticNumeral(&n, integer)
}

func (r *ArithRewriter) app(kind int32, args ...ast.ASTHandle) (ast.ASTHandle, error) {
	return r.manager.MkFamilyApp(r.family, kind, nil, args)
}

func (r *ArithRewriter) operation(h ast.ASTHandle, kinds ...int32) (int32, []ast.ASTHandle, bool, error) {
	family, err := r.manager.GetFamilyID(h)
	if err != nil || family != r.family {
		return 0, nil, false, err
	}
	kind, err := r.manager.GetDeclKind(h)
	if err != nil {
		return 0, nil, false, err
	}
	matched := len(kinds) == 0
	for _, candidate := range kinds {
		matched = matched || kind == candidate
	}
	if !matched {
		return kind, nil, false, nil
	}
	n, err := r.manager.GetNumArgs(h)
	if err != nil {
		return 0, nil, false, err
	}
	args := make([]ast.ASTHandle, n)
	for i := range args {
		args[i], err = r.manager.GetArg(h, uint32(i))
		if err != nil {
			return 0, nil, false, err
		}
	}
	return kind, args, true, nil
}

func (r *ArithRewriter) arithmeticTerm(h ast.ASTHandle) (bool, error) {
	kind, err := r.manager.GetKind(h)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	family, err := r.manager.GetFamilyID(h)
	return err == nil && family == r.family, err
}

func (r *ArithRewriter) isITE(h ast.ASTHandle) (bool, error) {
	family, err := r.manager.GetFamilyID(h)
	if err != nil || family != r.manager.BasicFamilyID() {
		return false, err
	}
	kind, err := r.manager.GetDeclKind(h)
	return err == nil && kind == ast.ASTOpIte, err
}

func arithFloor(value *big.Rat) *big.Int {
	q, rem := new(big.Int).QuoRem(value.Num(), value.Denom(), new(big.Int))
	if value.Sign() < 0 && rem.Sign() != 0 {
		q.Sub(q, big.NewInt(1))
	}
	return q
}

func arithIntegerDiv(left, right *big.Rat) *big.Int {
	return new(big.Int).Div(left.Num(), right.Num())
}

func arithIntegerMod(left, right *big.Rat) *big.Int {
	return new(big.Int).Mod(left.Num(), right.Num())
}

func (r *ArithRewriter) setBool(out ast.ASTRef, value bool) error {
	h, err := r.manager.MkBoolVal(value)
	if err != nil {
		return err
	}
	return r.manager.SetRef(out, h)
}

func (r *ArithRewriter) rewriteComparison(kind int32, left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if kind == arithOpLT || kind == arithOpGT {
		a, b := right, left
		if kind == arithOpGT {
			a, b = left, right
		}
		le, err := r.app(arithOpLE, a, b)
		if err != nil {
			return RewriteFailed, err
		}
		negated, err := r.manager.MkNot(le)
		if err != nil {
			return RewriteFailed, err
		}
		return Rewrite2, r.manager.SetRef(out, negated)
	}
	newLeft, newRight, changed, err := r.cancelMonomials(left, right, r.arithIneqLHS || r.arithLHS)
	if err != nil {
		return RewriteFailed, err
	}
	if changed {
		left, right = newLeft, newRight
	}
	lv, _, ln, err := r.numeral(left)
	if err != nil {
		return RewriteFailed, err
	}
	rv, _, rn, err := r.numeral(right)
	if err != nil || !ln || !rn {
		if changed && err == nil {
			result, e := r.app(kind, left, right)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, r.manager.SetRef(out, result)
		}
		return RewriteFailed, err
	}
	cmp := lv.Cmp(rv)
	return RewriteDone, r.setBool(out, kind == arithOpLE && cmp <= 0 || kind == arithOpGE && cmp >= 0)
}

func (r *ArithRewriter) rewriteNumeralFold(kind int32, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	identity := int64(0)
	if kind == arithOpMul {
		identity = 1
	}
	value := new(big.Rat).SetInt64(identity)
	integer := true
	for i := uint32(0); i < args.Count(); i++ {
		h, err := args.At(i)
		if err != nil {
			return RewriteFailed, err
		}
		v, isInteger, matched, err := r.numeral(h)
		if err != nil || !matched {
			return RewriteFailed, err
		}
		integer = integer && isInteger
		if kind == arithOpAdd {
			value.Add(value, v)
		} else {
			value.Mul(value, v)
		}
	}
	h, err := r.makeNumeral(value, integer)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, h)
}

func (r *ArithRewriter) rewriteUnaryMinus(arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	v, integer, matched, err := r.numeral(arg)
	if err != nil {
		return RewriteFailed, err
	}
	if matched {
		v.Neg(v)
		h, e := r.makeNumeral(v, integer)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	minusOne, err := r.numeralForTerm(big.NewRat(-1, 1), arg)
	if err != nil {
		return RewriteFailed, err
	}
	h, err := r.app(arithOpMul, minusOne, arg)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite1, r.manager.SetRef(out, h)
}

func (r *ArithRewriter) rewriteDivision(kind int32, left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	lv, li, ln, err := r.numeral(left)
	if err != nil {
		return RewriteFailed, err
	}
	rv, ri, rn, err := r.numeral(right)
	if err != nil || !rn || rv.Sign() == 0 {
		return RewriteFailed, err
	}
	if kind == arithOpDiv {
		if ln {
			h, e := r.makeNumeral(new(big.Rat).Quo(lv, rv), false)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, r.manager.SetRef(out, h)
		}
		coefficient := new(big.Rat).Inv(rv)
		c, e := r.makeNumeral(coefficient, false)
		if e != nil {
			return RewriteFailed, e
		}
		h, e := r.app(arithOpMul, c, left)
		if e != nil {
			return RewriteFailed, e
		}
		return Rewrite1, r.manager.SetRef(out, h)
	}
	if !ln || !li || !ri {
		return RewriteFailed, nil
	}
	var result *big.Int
	switch kind {
	case arithOpIDiv:
		result = arithIntegerDiv(lv, rv)
	case arithOpMod, arithOpRem:
		result = arithIntegerMod(lv, rv)
		if kind == arithOpRem && rv.Sign() < 0 {
			result.Neg(result)
		}
	default:
		return RewriteFailed, nil
	}
	h, err := r.makeNumeral(new(big.Rat).SetInt(result), true)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, h)
}

func (r *ArithRewriter) rewriteMod(left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	lv, li, ln, err := r.numeral(left)
	if err != nil {
		return RewriteFailed, err
	}
	rv, ri, rn, err := r.numeral(right)
	if err != nil {
		return RewriteFailed, err
	}
	if ln && rn && rv.Sign() != 0 && li && ri {
		h, e := r.makeNumeral(new(big.Rat).SetInt(arithIntegerMod(lv, rv)), true)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if rn && ri && rv.Cmp(big.NewRat(1, 1)) == 0 {
		h, e := r.makeNumeral(new(big.Rat), true)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if left == right && !rn {
		zero, e := r.makeNumeral(new(big.Rat), true)
		if e != nil {
			return RewriteFailed, e
		}
		undefined, e := r.app(arithOpMod, zero, zero)
		if e != nil {
			return RewriteFailed, e
		}
		absRef, e := r.manager.NewRef(ast.ASTHandle{})
		if e != nil {
			return RewriteFailed, e
		}
		defer func() { _ = r.manager.DelRef(&absRef) }()
		if _, e = r.rewriteAbs(right, absRef); e != nil {
			return RewriteFailed, e
		}
		absolute, e := r.manager.RefGet(absRef)
		if e != nil {
			return RewriteFailed, e
		}
		equal, e := r.manager.MkEq(right, zero)
		if e != nil {
			return RewriteFailed, e
		}
		result, e := r.manager.MkIte(equal, undefined, absolute)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	if _, inner, yes, e := r.operation(left, arithOpMod); e != nil {
		return RewriteFailed, e
	} else if yes && len(inner) == 2 && inner[1] == right && rn && ri && rv.Sign() != 0 {
		return RewriteDone, r.manager.SetRef(out, left)
	}
	if !rn || !ri || rv.Sign() <= 0 {
		return RewriteFailed, nil
	}
	kind, children, composite, err := r.operation(left, arithOpAdd, arithOpMul)
	if err != nil || !composite {
		return RewriteFailed, err
	}
	target := false
	for _, child := range children {
		if _, _, isMod, e := r.operation(child, arithOpMod); e != nil {
			return RewriteFailed, e
		} else if isMod {
			continue
		}
		if value, _, numeral, e := r.numeral(child); e != nil {
			return RewriteFailed, e
		} else if numeral && new(big.Int).Mod(value.Num(), rv.Num()).Cmp(value.Num()) == 0 {
			continue
		}
		if ite, e := r.isITE(child); e != nil {
			return RewriteFailed, e
		} else if ite {
			continue
		}
		target = true
		break
	}
	if !target {
		return RewriteFailed, nil
	}
	wrapped := make([]ast.ASTHandle, len(children))
	for i, child := range children {
		wrapped[i], err = r.app(arithOpMod, child, right)
		if err != nil {
			return RewriteFailed, err
		}
	}
	inside, err := r.app(kind, wrapped...)
	if err != nil {
		return RewriteFailed, err
	}
	result, err := r.app(arithOpMod, inside, right)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite3, r.manager.SetRef(out, result)
}

func (r *ArithRewriter) rewriteRem(left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	lv, li, ln, err := r.numeral(left)
	if err != nil {
		return RewriteFailed, err
	}
	rv, ri, rn, err := r.numeral(right)
	if err != nil {
		return RewriteFailed, err
	}
	if ln && rn && rv.Sign() != 0 && li && ri {
		value := arithIntegerMod(lv, rv)
		if rv.Sign() < 0 {
			value.Neg(value)
		}
		h, e := r.makeNumeral(new(big.Rat).SetInt(value), true)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if rn && ri && rv.Cmp(big.NewRat(1, 1)) == 0 {
		h, e := r.makeNumeral(new(big.Rat), true)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if rn && ri && rv.Sign() != 0 {
		if _, _, composite, e := r.operation(left, arithOpAdd, arithOpMul); e != nil {
			return RewriteFailed, e
		} else if composite {
			return RewriteFailed, nil
		}
		mod, e := r.app(arithOpMod, left, right)
		if e != nil {
			return RewriteFailed, e
		}
		if rv.Sign() > 0 {
			return Rewrite1, r.manager.SetRef(out, mod)
		}
		negative, e := r.app(arithOpUMinus, mod)
		if e != nil {
			return RewriteFailed, e
		}
		return Rewrite2, r.manager.SetRef(out, negative)
	}
	if !r.elimRem {
		return RewriteFailed, nil
	}
	mod, err := r.app(arithOpMod, left, right)
	if err != nil {
		return RewriteFailed, err
	}
	negative, err := r.app(arithOpUMinus, mod)
	if err != nil {
		return RewriteFailed, err
	}
	zero, err := r.makeNumeral(new(big.Rat), true)
	if err != nil {
		return RewriteFailed, err
	}
	nonnegative, err := r.app(arithOpGE, right, zero)
	if err != nil {
		return RewriteFailed, err
	}
	result, err := r.manager.MkIte(nonnegative, mod, negative)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite3, r.manager.SetRef(out, result)
}

func (r *ArithRewriter) rewritePower(base, exponent ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	x, integer, xn, err := r.numeral(base)
	if err != nil {
		return RewriteFailed, err
	}
	y, _, yn, err := r.numeral(exponent)
	if err != nil || !yn || !y.IsInt() || !y.Num().IsInt64() {
		return RewriteFailed, err
	}
	n := y.Num().Int64()
	if xn && x.Cmp(big.NewRat(1, 1)) == 0 || n == 1 {
		return RewriteDone, r.manager.SetRef(out, base)
	}
	if !xn || n == 0 && x.Sign() == 0 || n < 0 && integer || n > int64(r.maxDegree) || n < -int64(r.maxDegree) {
		return RewriteFailed, nil
	}
	if n == 0 {
		x.SetInt64(1)
	} else {
		negative := n < 0
		if negative {
			n = -n
		}
		result := new(big.Rat).SetInt64(1)
		for i := int64(0); i < n; i++ {
			result.Mul(result, x)
		}
		if negative {
			result.Inv(result)
		}
		x = result
	}
	h, err := r.makeNumeral(x, integer)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, h)
}

func (r *ArithRewriter) rewriteToInt(arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	v, _, matched, err := r.numeral(arg)
	if err != nil {
		return RewriteFailed, err
	}
	if matched {
		h, e := r.makeNumeral(new(big.Rat).SetInt(arithFloor(v)), true)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if yes, child, e := r.unary(arg, arithOpToReal); e != nil {
		return RewriteFailed, e
	} else if yes {
		return RewriteDone, r.manager.SetRef(out, child)
	}
	kind, children, composite, err := r.operation(arg, arithOpAdd, arithOpMul, arithOpPower)
	if err != nil || !composite {
		return RewriteFailed, err
	}
	converted := make([]ast.ASTHandle, len(children))
	for i, child := range children {
		value, _, numeral, e := r.numeral(child)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral && value.IsInt() {
			converted[i], e = r.makeNumeral(value, true)
			if e != nil {
				return RewriteFailed, e
			}
			continue
		}
		if yes, inner, e := r.unary(child, arithOpToReal); e != nil {
			return RewriteFailed, e
		} else if yes {
			converted[i] = inner
			continue
		}
		return RewriteFailed, nil
	}
	result, err := r.app(kind, converted...)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite1, r.manager.SetRef(out, result)
}

func (r *ArithRewriter) rewriteToReal(arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	v, _, matched, err := r.numeral(arg)
	if err != nil {
		return RewriteFailed, err
	}
	if matched {
		h, e := r.makeNumeral(v, false)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	if r.pushToReal {
		kind, children, composite, e := r.operation(arg, arithOpAdd, arithOpMul)
		if e != nil {
			return RewriteFailed, e
		}
		if composite {
			converted := make([]ast.ASTHandle, len(children))
			for i, child := range children {
				converted[i], e = r.app(arithOpToReal, child)
				if e != nil {
					return RewriteFailed, e
				}
			}
			result, e := r.app(kind, converted...)
			if e != nil {
				return RewriteFailed, e
			}
			return Rewrite2, r.manager.SetRef(out, result)
		}
	}
	return RewriteFailed, nil
}

func (r *ArithRewriter) unary(h ast.ASTHandle, kind int32) (bool, ast.ASTHandle, error) {
	family, err := r.manager.GetFamilyID(h)
	if err != nil || family != r.family {
		return false, ast.ASTHandle{}, err
	}
	k, err := r.manager.GetDeclKind(h)
	if err != nil || k != kind {
		return false, ast.ASTHandle{}, err
	}
	n, err := r.manager.GetNumArgs(h)
	if err != nil || n != 1 {
		return false, ast.ASTHandle{}, err
	}
	arg, err := r.manager.GetArg(h, 0)
	return err == nil, arg, err
}

func (r *ArithRewriter) rewriteIsInt(arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	v, _, matched, err := r.numeral(arg)
	if err != nil {
		return RewriteFailed, err
	}
	if matched {
		return RewriteDone, r.setBool(out, v.IsInt())
	}
	if yes, _, e := r.unary(arg, arithOpToReal); e != nil {
		return RewriteFailed, e
	} else if yes {
		return RewriteDone, r.setBool(out, true)
	}
	toInt, err := r.app(arithOpToInt, arg)
	if err != nil {
		return RewriteFailed, err
	}
	toReal, err := r.app(arithOpToReal, toInt)
	if err != nil {
		return RewriteFailed, err
	}
	equality, err := r.manager.MkEq(toReal, arg)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite3, r.manager.SetRef(out, equality)
}

func (r *ArithRewriter) rewriteAbs(arg ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	_, integer, matched, err := r.numeral(arg)
	if err != nil {
		return RewriteFailed, err
	}
	if !matched {
		sort, e := r.manager.GetSort(arg)
		if e != nil {
			return RewriteFailed, e
		}
		sortKind, e := r.manager.GetDeclKind(sort)
		if e != nil {
			return RewriteFailed, e
		}
		integer = sortKind == 1
	}
	zero, err := r.makeNumeral(new(big.Rat), integer)
	if err != nil {
		return RewriteFailed, err
	}
	ge, err := r.app(arithOpGE, arg, zero)
	if err != nil {
		return RewriteFailed, err
	}
	negative, err := r.app(arithOpUMinus, arg)
	if err != nil {
		return RewriteFailed, err
	}
	result, err := r.manager.MkIte(ge, arg, negative)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, r.manager.SetRef(out, result)
}

// MkEqCore performs the arithmetic equality dispatch used by the Boolean and
// combined theory rewriters.
func (r *ArithRewriter) MkEqCore(left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	if r.eq2ineq {
		le, err := r.app(arithOpLE, left, right)
		if err != nil {
			return RewriteFailed, err
		}
		ge, err := r.app(arithOpGE, left, right)
		if err != nil {
			return RewriteFailed, err
		}
		both, err := r.manager.MkAndArgs([]ast.ASTHandle{le, ge})
		if err != nil {
			return RewriteFailed, err
		}
		return Rewrite2, r.manager.SetRef(out, both)
	}
	leftArithmetic, err := r.arithmeticTerm(left)
	if err != nil {
		return RewriteFailed, err
	}
	rightArithmetic, err := r.arithmeticTerm(right)
	if err != nil {
		return RewriteFailed, err
	}
	changed := false
	if r.arithLHS || leftArithmetic || rightArithmetic {
		left, right, changed, err = r.cancelMonomials(left, right, r.arithLHS)
		if err != nil {
			return RewriteFailed, err
		}
	}
	lv, _, ln, err := r.numeral(left)
	if err != nil {
		return RewriteFailed, err
	}
	rv, _, rn, err := r.numeral(right)
	if err != nil || !ln || !rn {
		if changed && err == nil {
			result, e := r.manager.MkEq(left, right)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, r.manager.SetRef(out, result)
		}
		return RewriteFailed, err
	}
	return RewriteDone, r.setBool(out, lv.Cmp(rv) == 0)
}

// MkAppCore applies one arithmetic declaration to already rewritten arguments.
func (r *ArithRewriter) MkAppCore(decl ast.ASTHandle, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	family, err := r.manager.GetFamilyID(decl)
	if err != nil || family != r.family {
		return RewriteFailed, err
	}
	kind, err := r.manager.GetDeclKind(decl)
	if err != nil {
		return RewriteFailed, err
	}
	at := func(i uint32) (ast.ASTHandle, error) { return args.At(i) }
	if kind == arithOpNum {
		return RewriteFailed, nil
	}
	if kind == arithOpAdd {
		return r.rewriteAdd(args, out)
	}
	if kind == arithOpMul {
		return r.rewriteMul(args, out)
	}
	if kind == arithOpSub {
		return r.rewriteSub(args, out)
	}
	if kind == arithOpDiv || kind == arithOpIDiv {
		if args.Count() == 1 {
			h, e := at(0)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, r.manager.SetRef(out, h)
		}
	}
	if kind == arithOpLE || kind == arithOpGE || kind == arithOpLT || kind == arithOpGT || kind == arithOpDiv || kind == arithOpIDiv || kind == arithOpMod || kind == arithOpRem || kind == arithOpPower {
		if args.Count() != 2 {
			return RewriteFailed, ast.ErrASTDomain
		}
		left, e := at(0)
		if e != nil {
			return RewriteFailed, e
		}
		right, e := at(1)
		if e != nil {
			return RewriteFailed, e
		}
		switch kind {
		case arithOpLE, arithOpGE, arithOpLT, arithOpGT:
			return r.rewriteComparison(kind, left, right, out)
		case arithOpPower:
			return r.rewritePower(left, right, out)
		case arithOpMod:
			return r.rewriteMod(left, right, out)
		case arithOpRem:
			return r.rewriteRem(left, right, out)
		default:
			return r.rewriteDivision(kind, left, right, out)
		}
	}
	if args.Count() != 1 {
		return RewriteFailed, nil
	}
	arg, err := at(0)
	if err != nil {
		return RewriteFailed, err
	}
	switch kind {
	case arithOpUMinus:
		return r.rewriteUnaryMinus(arg, out)
	case arithOpToInt:
		return r.rewriteToInt(arg, out)
	case arithOpToReal:
		return r.rewriteToReal(arg, out)
	case arithOpIsInt:
		return r.rewriteIsInt(arg, out)
	case arithOpAbs:
		return r.rewriteAbs(arg, out)
	}
	return RewriteFailed, nil
}
