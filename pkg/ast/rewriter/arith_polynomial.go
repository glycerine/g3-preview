// Source: src/ast/rewriter/poly_rewriter.h;
// src/ast/rewriter/poly_rewriter_def.h.
package rewriter

import (
	"math/big"
	"sort"

	"github.com/glycerine/g3/pkg/ast"
)

func (r *ArithRewriter) integerTerm(h ast.ASTHandle) (bool, error) {
	s, err := r.manager.GetSort(h)
	if err != nil {
		return false, err
	}
	family, err := r.manager.GetFamilyID(s)
	if err != nil || family != r.family {
		return false, err
	}
	kind, err := r.manager.GetDeclKind(s)
	return err == nil && kind == 1, err
}

func (r *ArithRewriter) numeralForTerm(value *big.Rat, term ast.ASTHandle) (ast.ASTHandle, error) {
	integer, err := r.integerTerm(term)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return r.makeNumeral(value, integer)
}

func (r *ArithRewriter) makeAdd(args []ast.ASTHandle, term ast.ASTHandle) (ast.ASTHandle, error) {
	switch len(args) {
	case 0:
		return r.numeralForTerm(new(big.Rat), term)
	case 1:
		return args[0], nil
	default:
		return r.app(arithOpAdd, args...)
	}
}

func (r *ArithRewriter) makeMul(args []ast.ASTHandle, term ast.ASTHandle) (ast.ASTHandle, error) {
	switch len(args) {
	case 0:
		return r.numeralForTerm(big.NewRat(1, 1), term)
	case 1:
		return args[0], nil
	default:
		return r.app(arithOpMul, args...)
	}
}

func (r *ArithRewriter) sameArguments(args ast.AppArguments, values []ast.ASTHandle) (bool, error) {
	if uint32(len(values)) != args.Count() {
		return false, nil
	}
	for i := range values {
		h, err := args.At(uint32(i))
		if err != nil {
			return false, err
		}
		if h != values[i] {
			return false, nil
		}
	}
	return true, nil
}

func (r *ArithRewriter) flattenArguments(args ast.AppArguments, kind int32, recursive bool) ([]ast.ASTHandle, bool, error) {
	values := make([]ast.ASTHandle, 0, args.Count())
	changed := false
	for i := uint32(0); i < args.Count(); i++ {
		h, err := args.At(i)
		if err != nil {
			return nil, false, err
		}
		_, children, matched, err := r.operation(h, kind)
		if err != nil {
			return nil, false, err
		}
		if !r.flat || !matched {
			values = append(values, h)
			continue
		}
		changed = true
		if !recursive {
			values = append(values, children...)
			continue
		}
		stack := append([]ast.ASTHandle(nil), children...)
		for len(stack) != 0 {
			current := stack[0]
			stack = stack[1:]
			_, nested, yes, e := r.operation(current, kind)
			if e != nil {
				return nil, false, e
			}
			if yes {
				stack = append(append([]ast.ASTHandle(nil), nested...), stack...)
			} else {
				values = append(values, current)
			}
		}
	}
	return values, changed, nil
}

func (r *ArithRewriter) monomial(h ast.ASTHandle) (*big.Rat, ast.ASTHandle, error) {
	_, args, matched, err := r.operation(h, arithOpMul)
	if err != nil {
		return nil, ast.ASTHandle{}, err
	}
	if matched && len(args) == 2 {
		coefficient, _, numeral, e := r.numeral(args[0])
		if e != nil {
			return nil, ast.ASTHandle{}, e
		}
		if numeral {
			return coefficient, args[1], nil
		}
	}
	return big.NewRat(1, 1), h, nil
}

func (r *ArithRewriter) coefficientProduct(coefficient *big.Rat, powerProduct, term ast.ASTHandle) (ast.ASTHandle, error) {
	if coefficient.Cmp(big.NewRat(1, 1)) == 0 {
		return powerProduct, nil
	}
	numeral, err := r.numeralForTerm(coefficient, term)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return r.app(arithOpMul, numeral, powerProduct)
}

func (r *ArithRewriter) rewriteAdd(args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	if args.Count() == 0 {
		return RewriteFailed, ast.ErrASTDomain
	}
	if args.Count() == 1 {
		h, err := args.At(0)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	values, flattened, err := r.flattenArguments(args, arithOpAdd, false)
	if err != nil {
		return RewriteFailed, err
	}
	first, err := args.At(0)
	if err != nil {
		return RewriteFailed, err
	}
	constant := new(big.Rat)
	order := make([]ast.ASTHandle, 0, len(values))
	coefficients := make(map[ast.ASTHandle]*big.Rat)
	counts := make(map[ast.ASTHandle]int)
	for _, value := range values {
		v, _, numeral, e := r.numeral(value)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral {
			constant.Add(constant, v)
			continue
		}
		coefficient, product, e := r.monomial(value)
		if e != nil {
			return RewriteFailed, e
		}
		if counts[product] == 0 {
			order = append(order, product)
			coefficients[product] = new(big.Rat)
		}
		counts[product]++
		coefficients[product].Add(coefficients[product], coefficient)
	}
	resultArgs := make([]ast.ASTHandle, 0, len(order)+1)
	if constant.Sign() != 0 {
		h, e := r.numeralForTerm(constant, first)
		if e != nil {
			return RewriteFailed, e
		}
		resultArgs = append(resultArgs, h)
	}
	duplicate := false
	for _, product := range order {
		duplicate = duplicate || counts[product] > 1
		coefficient := coefficients[product]
		if coefficient.Sign() == 0 {
			continue
		}
		h, e := r.coefficientProduct(coefficient, product, first)
		if e != nil {
			return RewriteFailed, e
		}
		resultArgs = append(resultArgs, h)
	}
	if r.sortSums && len(resultArgs) > 1 {
		start := 0
		if constant.Sign() != 0 {
			start = 1
		}
		var cause error
		sort.SliceStable(resultArgs[start:], func(i, j int) bool {
			less, e := r.manager.ASTLess(resultArgs[start+i], resultArgs[start+j])
			if cause == nil {
				cause = e
			}
			return e == nil && less
		})
		if cause != nil {
			return RewriteFailed, cause
		}
	}
	result, err := r.makeAdd(resultArgs, first)
	if err != nil {
		return RewriteFailed, err
	}
	same, err := r.sameArguments(args, resultArgs)
	if err != nil {
		return RewriteFailed, err
	}
	if !flattened && !duplicate && same {
		return RewriteFailed, nil
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *ArithRewriter) rewriteMul(args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	if args.Count() == 0 {
		return RewriteFailed, ast.ErrASTDomain
	}
	if args.Count() == 1 {
		h, err := args.At(0)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	values, flattened, err := r.flattenArguments(args, arithOpMul, true)
	if err != nil {
		return RewriteFailed, err
	}
	first, err := args.At(0)
	if err != nil {
		return RewriteFailed, err
	}
	coefficient := big.NewRat(1, 1)
	factors := make([]ast.ASTHandle, 0, len(values))
	for _, value := range values {
		v, _, numeral, e := r.numeral(value)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral {
			coefficient.Mul(coefficient, v)
		} else {
			factors = append(factors, value)
		}
	}
	if coefficient.Sign() == 0 || len(factors) == 0 {
		h, e := r.numeralForTerm(coefficient, first)
		if e != nil {
			return RewriteFailed, e
		}
		return RewriteDone, r.manager.SetRef(out, h)
	}
	var cause error
	sort.SliceStable(factors, func(i, j int) bool {
		less, e := r.manager.ASTLess(factors[i], factors[j])
		if cause == nil {
			cause = e
		}
		return e == nil && less
	})
	if cause != nil {
		return RewriteFailed, cause
	}
	combined := make([]ast.ASTHandle, 0, len(factors)+1)
	if coefficient.Cmp(big.NewRat(1, 1)) != 0 {
		n, e := r.numeralForTerm(coefficient, first)
		if e != nil {
			return RewriteFailed, e
		}
		combined = append(combined, n)
	}
	for i := 0; i < len(factors); {
		j := i + 1
		for j < len(factors) && factors[j] == factors[i] {
			j++
		}
		if j-i == 1 {
			combined = append(combined, factors[i])
		} else {
			exponent, e := r.numeralForTerm(new(big.Rat).SetInt64(int64(j-i)), first)
			if e != nil {
				return RewriteFailed, e
			}
			power, e := r.app(arithOpPower, factors[i], exponent)
			if e != nil {
				return RewriteFailed, e
			}
			combined = append(combined, power)
		}
		i = j
	}
	result, err := r.makeMul(combined, first)
	if err != nil {
		return RewriteFailed, err
	}
	same, err := r.sameArguments(args, combined)
	if err != nil {
		return RewriteFailed, err
	}
	if !flattened && same {
		return RewriteFailed, nil
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

func (r *ArithRewriter) rewriteSub(args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	if args.Count() == 0 {
		return RewriteFailed, ast.ErrASTDomain
	}
	first, err := args.At(0)
	if err != nil {
		return RewriteFailed, err
	}
	if args.Count() == 1 {
		return RewriteDone, r.manager.SetRef(out, first)
	}
	minusOne, err := r.numeralForTerm(big.NewRat(-1, 1), first)
	if err != nil {
		return RewriteFailed, err
	}
	values := []ast.ASTHandle{first}
	for i := uint32(1); i < args.Count(); i++ {
		h, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		v, _, numeral, e := r.numeral(h)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral && v.Sign() == 0 {
			continue
		}
		negative, e := r.app(arithOpMul, minusOne, h)
		if e != nil {
			return RewriteFailed, e
		}
		values = append(values, negative)
	}
	result, err := r.makeAdd(values, first)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, r.manager.SetRef(out, result)
}
