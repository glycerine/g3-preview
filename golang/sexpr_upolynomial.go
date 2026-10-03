// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Source math/polynomial/sexpr2upolynomial.cpp, recursive source parser.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"reflect"
)

type SexprPolynomialError struct {
	Message   string
	Line, Pos int32
}

func (e *SexprPolynomialError) Error() string { return e.Message }

var ErrSexprPolynomialPending = errors.New("g3: polynomial conversion cleanup pending")

type sexprPolynomialFrame struct {
	argument mpnum.UPolynomial
	live     bool
}

// Cold frame storage owns actual source scoped vectors, including stopped
// cleanup. Its real Go bytes are charged separately from numeric payloads.
type SexprPolynomialConverter struct {
	self       *SexprPolynomialConverter
	sexpr      *SexprManager
	polynomial *mpnum.UPolynomialManager
	frames     []sexprPolynomialFrame
	depth      uint32
	header     uint64
	cause      error
	closed     bool
}

func NewSexprPolynomialConverter(s *SexprManager, m *mpnum.UPolynomialManager) (*SexprPolynomialConverter, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	if e := s.config.Rationals.WithUPolynomialRationals(m, func(*mpnum.UPolynomialRationalContext) error { return nil }); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SexprPolynomialConverter{}).Size())
	if e := s.config.Budget.Reserve(n); e != nil {
		return nil, e
	}
	c := &SexprPolynomialConverter{sexpr: s, polynomial: m, header: n}
	c.self = c
	return c, nil
}
func (c *SexprPolynomialConverter) open() error {
	if c == nil || c.self != c {
		return ErrSexprOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}
func (c *SexprPolynomialConverter) parseError(h SexprHandle, msg string) error {
	r, e := c.sexpr.record(h)
	if e != nil {
		return e
	}
	return &SexprPolynomialError{msg, int32(r.line), int32(r.pos)}
}
func (c *SexprPolynomialConverter) frame() (uint32, error) {
	i := c.depth
	if uint64(i) >= uint64(len(c.frames)) {
		n := uint64(len(c.frames)) * 2
		if n == 0 {
			n = 2
		}
		stride := uint64(reflect.TypeOf(sexprPolynomialFrame{}).Size())
		if n > uint64(^uint(0)>>1)/stride {
			return 0, arena.ErrBounds
		}
		if e := c.sexpr.config.Budget.Reserve(n * stride); e != nil {
			return 0, e
		}
		a := make([]sexprPolynomialFrame, int(n))
		copy(a, c.frames)
		c.sexpr.config.Budget.Release(uint64(len(c.frames)) * stride)
		c.frames = a
	}
	c.depth++
	return i, nil
}
func (c *SexprPolynomialConverter) Convert(h SexprHandle, p mpnum.UPolynomial) error {
	return c.convertAtDepth(h, p, 0)
}
func (c *SexprPolynomialConverter) convertAtDepth(h SexprHandle, p mpnum.UPolynomial, depth uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if c.cause != nil {
		return ErrSexprPolynomialPending
	}
	if e := c.sexpr.lock(); e != nil {
		return e
	}
	defer c.sexpr.mu.Unlock()
	return c.sexpr.config.Rationals.WithUPolynomialRationals(c.polynomial, func(ctx *mpnum.UPolynomialRationalContext) error { return c.convert(ctx, h, p, depth) })
}
func (c *SexprPolynomialConverter) child(r *sexprRecord, i uint32) (SexprHandle, error) {
	return c.sexpr.child(r, i)
}
func (c *SexprPolynomialConverter) convert(ctx *mpnum.UPolynomialRationalContext, h SexprHandle, p mpnum.UPolynomial, depth uint32) (result error) {
	if depth > 1<<16 {
		return c.parseError(h, "invalid univariate polynomial, too complex")
	}
	r, e := c.sexpr.record(h)
	if e != nil {
		return e
	}
	switch r.kind {
	case SexprComposite:
		num := r.length
		if num == 0 {
			return c.parseError(h, "invalid univariate polynomial, symbol expected")
		}
		head, e := c.child(r, 0)
		if e != nil {
			return e
		}
		hr, e := c.sexpr.record(head)
		if e != nil {
			return e
		}
		if hr.kind != SexprSymbol {
			return c.parseError(h, "invalid univariate polynomial, symbol expected")
		}
		op, e := c.operator(hr.symbol)
		if e != nil {
			return e
		}
		if op != "+" && op != "-" && op != "*" && op != "^" {
			return c.parseError(h, "invalid univariate polynomial, '+', '-', '^' or '*' expected")
		}
		if op == "^" && num != 3 {
			return c.parseError(h, "invalid univariate polynomial, '^' operator expects two arguments")
		}
		if op != "^" && num <= 1 {
			return c.parseError(h, "invalid univariate polynomial, '"+op+"' operator expects at least one argument")
		}
		first, e := c.child(r, 1)
		if e != nil {
			return e
		}
		if e = c.convert(ctx, first, p, depth+1); e != nil {
			return e
		}
		if op == "^" {
			exponent, e := c.child(r, 2)
			if e != nil {
				return e
			}
			er, e := c.sexpr.record(exponent)
			if e != nil {
				return e
			}
			if er.kind != SexprNumeral {
				return c.parseError(exponent, "invalid univariate polynomial, exponent must be an unsigned integer")
			}
			n, ok, e := ctx.Exponent(&er.numeral)
			if e != nil {
				return e
			}
			if !ok {
				return c.parseError(exponent, "invalid univariate polynomial, exponent must be an unsigned integer")
			}
			return c.polynomial.Power(p, n, p)
		}
		if op == "-" && num == 2 {
			return c.polynomial.NegInPlace(p)
		}
		index, e := c.frame()
		if e != nil {
			return e
		}
		defer func() {
			c.depth--
			// Child numeric cleanup must finish before this older completed vector.
			if c.cause != nil || ctx.Pending() {
				if c.cause == nil {
					c.cause = ErrSexprPolynomialPending
				}
				return
			}
			f := &c.frames[index]
			if f.live {
				if e := c.polynomial.DelPolynomial(&f.argument); e != nil {
					c.cause = e
					ctx.StopVectors(e)
					result = errors.Join(result, e)
				} else {
					f.live = false
				}
			}
		}()
		arg, e := c.polynomial.NewPolynomial()
		if e != nil {
			return e
		}
		c.frames[index] = sexprPolynomialFrame{arg, true}
		for i := uint32(2); i < num; i++ {
			if e = c.polynomial.Reset(arg); e != nil {
				return e
			}
			ch, e := c.child(r, i)
			if e != nil {
				return e
			}
			if e = c.convert(ctx, ch, arg, depth+1); e != nil {
				return e
			}
			switch op {
			case "+":
				e = c.polynomial.Add(arg, p, p)
			case "-":
				e = c.polynomial.Sub(p, arg, p)
			case "*":
				e = c.polynomial.Mul(arg, p, p)
			}
			if e != nil {
				return e
			}
		}
		return nil
	case SexprNumeral:
		return ctx.SetConstant(p, &r.numeral, func() error { return c.parseError(h, "invalid univariate polynomial, integer coefficient expected") })

	case SexprSymbol:
		name, e := c.sexpr.config.Symbols.EqualString(r.symbol, stringPointer("x"))
		if e != nil {
			return e
		}
		if !name {
			return c.parseError(h, "invalid univariate polynomial, variable 'x' expected")
		}
		return ctx.SetIdentity(p)
	default:
		return c.parseError(h, "invalid univariate polynomial, unexpected ")
	}
}
func (c *SexprPolynomialConverter) Cleanup() error {
	if e := c.open(); e != nil {
		return e
	}
	err := c.sexpr.config.Rationals.CleanupUPolynomialRationals(func() error {
		for i := len(c.frames) - 1; i >= 0; i-- {
			f := &c.frames[i]
			if f.live {
				if e := c.polynomial.DelPolynomial(&f.argument); e != nil {
					return e
				}
				f.live = false
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	c.cause = nil
	return nil
}
func (c *SexprPolynomialConverter) Close() error {
	if c != nil && c.self == c && c.closed {
		return nil
	}
	if e := c.open(); e != nil {
		return e
	}
	if e := c.Cleanup(); e != nil {
		return e
	}
	c.sexpr.config.Budget.Release(c.header + uint64(len(c.frames))*uint64(reflect.TypeOf(sexprPolynomialFrame{}).Size()))
	c.frames = nil
	c.header = 0
	c.closed = true
	return nil
}

func stringPointer(s string) *string { return &s }
func (c *SexprPolynomialConverter) operator(sym Symbol) (string, error) {
	if sym.IsNumerical() {
		return "", nil
	}
	for _, op := range []string{"+", "-", "*", "^"} {
		ok, e := c.sexpr.config.Symbols.EqualString(sym, &op)
		if e != nil {
			return "", e
		}
		if ok {
			return op, nil
		}
	}
	return "", nil
}
