// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:121-174.
package arith

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// PolynomialBuilder owns the tactic's concrete rational coefficient buffer
// and expression pins. It borrows the AST manager and bound equation store.
type PolynomialBuilder struct {
	self     *PolynomialBuilder
	manager  *ast.ASTManager
	family   ast.FamilyID
	store    *BoundEquationStore
	coeff    *numeral.ScopedRationalVector
	ids      []uint32
	expr2var map[ast.ASTHandle]uint32
	var2expr []ast.ASTHandle
	refs     []ast.ASTRef
	closed   bool
}

func NewPolynomialBuilder(m *ast.ASTManager, store *BoundEquationStore) (*PolynomialBuilder, error) {
	if m == nil || store == nil {
		return nil, arena.ErrInvalidSlot
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	if err := store.open(); err != nil {
		return nil, err
	}
	family, err := m.ArithmeticFamilyID()
	if err != nil {
		return nil, err
	}
	coeff, err := numeral.NewScopedRationalVector(store.manager)
	if err != nil {
		return nil, err
	}
	b := &PolynomialBuilder{manager: m, family: family, store: store, coeff: coeff,
		expr2var: make(map[ast.ASTHandle]uint32)}
	b.self = b
	return b, nil
}

func (b *PolynomialBuilder) open() error {
	if b == nil || b.self != b || b.manager == nil || b.store == nil || b.coeff == nil {
		return arena.ErrInvalidSlot
	}
	if b.closed {
		return arena.ErrClosed
	}
	return b.store.open()
}

func (b *PolynomialBuilder) isArithmetic(h ast.ASTHandle, operation int32) (bool, error) {
	kind, err := b.manager.GetKind(h)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	family, err := b.manager.GetFamilyID(h)
	if err != nil || family != b.family {
		return false, err
	}
	k, err := b.manager.GetDeclKind(h)
	return k == operation, err
}

func (b *PolynomialBuilder) mkVar(term ast.ASTHandle) (uint32, error) {
	toReal, err := b.isArithmetic(term, 14)
	if err != nil {
		return 0, err
	}
	if toReal {
		term, err = b.manager.GetArg(term, 0)
		if err != nil {
			return 0, err
		}
	}
	if x, found := b.expr2var[term]; found {
		return x, nil
	}
	if uint64(len(b.var2expr)) >= uint64(noEquationPosition) {
		return 0, arena.ErrBounds
	}
	sort, err := b.manager.GetSort(term)
	if err != nil {
		return 0, err
	}
	family, err := b.manager.GetFamilyID(sort)
	if err != nil {
		return 0, err
	}
	kind, err := b.manager.GetDeclKind(sort)
	if err != nil {
		return 0, err
	}
	x := uint32(len(b.var2expr))
	if err = b.store.MkVar(x, family == b.family && kind == 1); err != nil {
		return 0, err
	}
	ref, err := b.manager.NewRef(term)
	if err != nil {
		return 0, err
	}
	b.refs = append(b.refs, ref)
	b.expr2var[term] = x
	b.var2expr = append(b.var2expr, term)
	return x, nil
}

func (b *PolynomialBuilder) pushTerm(term ast.ASTHandle) (err error) {
	coefficient := numeral.NewRational()
	defer func() { err = errors.Join(err, b.store.manager.Del(&coefficient)) }()
	xTerm := term
	mul, err := b.isArithmetic(term, 9)
	if err != nil {
		return err
	}
	if mul {
		arity, e := b.manager.GetNumArgs(term)
		if e != nil {
			return e
		}
		if arity == 2 {
			left, e := b.manager.GetArg(term, 0)
			if e != nil {
				return e
			}
			text, _, matched, e := b.manager.ArithmeticNumeralText(left)
			if e != nil {
				return e
			}
			if matched {
				if e = b.store.manager.SetString(&coefficient, text); e != nil {
					return e
				}
				xTerm, e = b.manager.GetArg(term, 1)
				if e != nil {
					return e
				}
			}
		}
	}
	if xTerm == term {
		if err = b.store.manager.SetInt32(&coefficient, 1); err != nil {
			return err
		}
	}
	if err = b.coeff.Push(&coefficient); err != nil {
		return err
	}
	x, err := b.mkVar(xTerm)
	if err != nil {
		return err
	}
	b.ids = append(b.ids, x)
	return nil
}

// MkLinearPolynomial creates the source variable for an expression. An ADD
// also publishes the concrete equation sum(terms)-whole-expression=0.
func (b *PolynomialBuilder) MkLinearPolynomial(term ast.ASTHandle) (uint32, error) {
	if err := b.open(); err != nil {
		return 0, err
	}
	if x, found := b.expr2var[term]; found {
		return x, nil
	}
	x, err := b.mkVar(term)
	if err != nil {
		return 0, err
	}
	add, err := b.isArithmetic(term, 6)
	if err != nil || !add {
		return x, err
	}
	if err = b.coeff.Reset(); err != nil {
		return x, err
	}
	b.ids = b.ids[:0]
	arity, err := b.manager.GetNumArgs(term)
	if err != nil {
		return x, err
	}
	for i := uint32(0); i < arity; i++ {
		arg, e := b.manager.GetArg(term, i)
		if e != nil {
			return x, e
		}
		if e = b.pushTerm(arg); e != nil {
			return x, e
		}
	}
	minusOne := numeral.NewRational()
	if err = b.store.manager.SetInt32(&minusOne, -1); err != nil {
		return x, err
	}
	err = errors.Join(b.coeff.Push(&minusOne), b.store.manager.Del(&minusOne))
	if err != nil {
		return x, err
	}
	b.ids = append(b.ids, x)
	values, err := b.coeff.BorrowedValues()
	if err != nil {
		return x, err
	}
	_, err = b.store.MkRationalEquation(values, b.ids)
	return x, err
}

// ProcessBound is the source process(expr*) branch for equality and ordered
// arithmetic bounds. It consumes neither the expression nor a numeral owner;
// a recognized bound is published before returning true.
func (b *PolynomialBuilder) ProcessBound(term ast.ASTHandle) (processed bool, err error) {
	if err = b.open(); err != nil {
		return false, err
	}
	negated := false
	for {
		isNot, child, e := b.manager.IsNot(term)
		if e != nil {
			return false, e
		}
		if !isNot {
			break
		}
		negated = !negated
		term = child
	}
	kind, err := b.manager.GetKind(term)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	family, err := b.manager.GetFamilyID(term)
	if err != nil {
		return false, err
	}
	op, err := b.manager.GetDeclKind(term)
	if err != nil {
		return false, err
	}
	const (
		boundEqual = iota
		boundLessEqual
		boundGreaterEqual
	)
	bound := boundEqual
	strict := false
	switch {
	case family == b.manager.BasicFamilyID() && op == ast.ASTOpEq:
		if negated {
			return false, nil
		}
	case family == b.family && op == 2:
		bound = boundLessEqual
		if negated {
			bound, strict = boundGreaterEqual, true
		}
	case family == b.family && op == 3:
		bound = boundGreaterEqual
		if negated {
			bound, strict = boundLessEqual, true
		}
	default:
		return false, nil
	}
	lhs, err := b.manager.GetArg(term, 0)
	if err != nil {
		return false, err
	}
	rhs, err := b.manager.GetArg(term, 1)
	if err != nil {
		return false, err
	}
	if _, _, numeralLeft, e := b.manager.ArithmeticNumeralText(lhs); e != nil {
		return false, e
	} else if numeralLeft {
		lhs, rhs = rhs, lhs
		if bound == boundLessEqual {
			bound = boundGreaterEqual
		} else if bound == boundGreaterEqual {
			bound = boundLessEqual
		}
	}
	value, _, numeralRight, err := b.manager.ArithmeticNumeralText(rhs)
	if err != nil || !numeralRight {
		return false, err
	}
	x, err := b.MkLinearPolynomial(lhs)
	if err != nil {
		return false, err
	}
	k := numeral.NewRational()
	defer func() { err = errors.Join(err, b.store.manager.Del(&k)) }()
	if err = b.store.manager.SetString(&k, value); err != nil {
		return false, err
	}
	switch bound {
	case boundEqual:
		if _, err = b.store.AssertLower(x, &k, false); err != nil {
			return false, err
		}
		_, err = b.store.AssertUpper(x, &k, false)
	case boundLessEqual:
		_, err = b.store.AssertUpper(x, &k, strict)
	case boundGreaterEqual:
		_, err = b.store.AssertLower(x, &k, strict)
	}
	return err == nil, err
}

// LowerSubsumed and UpperSubsumed are the concrete rational-buffer callers in
// propagate_ineqs_tactic.cpp:256-277. They reuse the same owned coefficient
// buffer as MkLinearPolynomial and borrow its current contents for one bound
// query; a non-addition has no polynomial subsumption path in the source.
func (b *PolynomialBuilder) LowerSubsumed(term ast.ASTHandle, k *numeral.Rational, strict bool) (bool, error) {
	return b.polynomialSubsumed(term, k, strict, true)
}

func (b *PolynomialBuilder) UpperSubsumed(term ast.ASTHandle, k *numeral.Rational, strict bool) (bool, error) {
	return b.polynomialSubsumed(term, k, strict, false)
}

func (b *PolynomialBuilder) polynomialSubsumed(term ast.ASTHandle, k *numeral.Rational, strict, lower bool) (subsumed bool, err error) {
	if err = b.open(); err != nil {
		return false, err
	}
	add, err := b.isArithmetic(term, 6)
	if err != nil || !add {
		return false, err
	}
	if err = b.coeff.Reset(); err != nil {
		return false, err
	}
	b.ids = b.ids[:0]
	arity, err := b.manager.GetNumArgs(term)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < arity; i++ {
		arg, e := b.manager.GetArg(term, i)
		if e != nil {
			return false, e
		}
		if e = b.pushTerm(arg); e != nil {
			return false, e
		}
	}
	values, err := b.coeff.BorrowedValues()
	if err != nil {
		return false, err
	}
	implied := numeral.NewRational()
	defer func() { err = errors.Join(err, b.store.manager.Del(&implied)) }()
	found, impliedStrict, err := b.store.PolynomialBound(values, b.ids, lower, &implied)
	if err != nil || !found {
		return false, err
	}
	cmp, err := b.store.manager.Compare(&implied, k)
	if err != nil {
		return false, err
	}
	if lower {
		return cmp > 0 || cmp == 0 && (!strict || impliedStrict), nil
	}
	return cmp < 0 || cmp == 0 && (!strict || impliedStrict), nil
}

func (b *PolynomialBuilder) Close() error {
	if b == nil || b.self != b {
		return arena.ErrInvalidSlot
	}
	if b.closed {
		return nil
	}
	if err := b.coeff.Close(); err != nil {
		return err
	}
	for i := len(b.refs) - 1; i >= 0; i-- {
		if err := b.manager.DelRef(&b.refs[i]); err != nil {
			return err
		}
	}
	b.refs, b.ids, b.var2expr, b.expr2var = nil, nil, nil, nil
	b.closed = true
	return nil
}
