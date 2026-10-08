// Source: src/ast/rewriter/bv_rewriter.h:158-170;
// src/ast/rewriter/bv_rewriter.cpp:26-51,2076-2079,2170-2205;
// src/ast/rewriter/bv_trailing.cpp:26-48,365-388.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// BVRewriter owns the derived BV object's persistent members. The polynomial
// base owns the expression map/configuration, the embedded extraction member
// owns its cached declaration, and trailing owns its original lazy caches.
// AST and parameter services are borrowed. Do not copy this owner.
type BVRewriter struct {
	self                  *BVRewriter
	manager               *ast.ASTManager
	context               *util.ParamsContext
	bvFamily, arithFamily ast.FamilyID
	polynomial            *BVPolynomialRewriter
	extract               MkExtractProc
	trailing              *BVTrailing
	defaultParams         util.ParamsRef
	defaultDecrement      util.ParamsDecrementReceipt
	defaultPlaced         bool
	header                uint64
	running, closed       bool
	cause                 error
}

func NewBVRewriter(m *ast.ASTManager, c *util.ParamsContext, g *util.GlobalParams, local *util.ParamsRef) (*BVRewriter, error) {
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	if c == nil || g == nil || g.ContextField != c {
		return nil, util.ErrParamsOwner
	}
	if local != nil {
		if e := c.Check(local); e != nil {
			return nil, e
		}
	}
	bytes := uint64(reflect.TypeOf(BVRewriter{}).Size())
	if e := m.Budget().Reserve(bytes); e != nil {
		return nil, e
	}
	p := &BVRewriter{manager: m, context: c, header: bytes}
	p.self = p
	if local == nil {
		v, e := c.NewValue()
		if e != nil {
			return p, p.fail(e)
		}
		p.defaultParams = v
		p.defaultPlaced = true
		local = &p.defaultParams
	}
	var e error
	p.bvFamily, e = m.BitVectorUtilityFamilyID()
	if e != nil {
		return p, p.fail(e)
	}
	p.polynomial, e = NewBVPolynomialRewriter(m, c, g, local)
	if e != nil {
		return p, p.fail(e)
	}
	// Source base's two parameter updates precede each derived member.
	p.extract.constructSource(m)
	p.trailing, e = NewBVTrailing(&p.extract)
	if e != nil {
		return p, p.fail(e)
	}
	// arith_util creates its family identity and borrows the manager, with
	// an initially null lazy declaration-plugin pointer.
	p.arithFamily, e = m.ArithmeticFamilyID()
	if e != nil {
		return p, p.fail(e)
	}
	if e = p.polynomial.config.UpdateBitVectorLocal(local); e != nil {
		return p, p.fail(e)
	}
	if e = p.endDefault(c); e != nil {
		return p, p.fail(e)
	}
	return p, nil
}
func (p *BVRewriter) open() error {
	if p == nil || p.self != p || p.closed {
		return ast.ErrASTHandle
	}
	if p.cause != nil {
		return p.cause
	}
	if p.running {
		return ast.ErrASTPending
	}
	return nil
}
func (p *BVRewriter) fail(e error) error {
	if e != nil && p.cause == nil {
		p.cause = e
	}
	return e
}
func (p *BVRewriter) endDefault(c *util.ParamsContext) error {
	if !p.defaultPlaced {
		return nil
	}
	var e error
	if p.defaultDecrement.Invocation() != 0 {
		e = c.CompleteDecrement(&p.defaultParams, p.defaultDecrement)
	} else {
		e = c.DelTracked(&p.defaultParams, &p.defaultDecrement)
	}
	if e == nil {
		p.defaultPlaced = false
		p.defaultDecrement = util.ParamsDecrementReceipt{}
	}
	return e
}
func (p *BVRewriter) State() (BVRewriterConfigurationState, error) {
	if e := p.open(); e != nil {
		return BVRewriterConfigurationState{}, e
	}
	return p.polynomial.config.BitVectorState()
}
func (p *BVRewriter) Update(local *util.ParamsRef) error {
	if e := p.open(); e != nil {
		return e
	}
	if e := p.polynomial.config.UpdateBitVector(local); e != nil {
		return p.fail(e)
	}
	return nil
}
func (p *BVRewriter) SetMkBVToNumeral(v bool) error {
	if e := p.open(); e != nil {
		return e
	}
	return p.polynomial.config.SetMkBVToNumeral(v)
}
func (p *BVRewriter) RewriteAddition(args ast.AppArguments, out ast.ASTRef) (st RewriteStatus, err error) {
	if e := p.open(); e != nil {
		return RewriteFailed, e
	}
	p.running = true
	defer func() { p.running = false; p.fail(err) }()
	return p.polynomial.RewriteBitVectorAddition(args, out)
}
func (p *BVRewriter) RewriteMultiplication(args ast.AppArguments, out ast.ASTRef) (st RewriteStatus, err error) {
	if e := p.open(); e != nil {
		return RewriteFailed, e
	}
	p.running = true
	defer func() { p.running = false; p.fail(err) }()
	return p.polynomial.RewriteBitVectorMultiplication(args, out, p.polynomial.config.bitVector.MultiplyToConcat, &p.extract)
}

func (p *BVRewriter) Close() error {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if p.running {
		return ast.ErrASTPending
	}
	// Stopped invocation frames are younger than persistent derived members.
	if p.polynomial != nil {
		if e := p.polynomial.closeInvocations(); e != nil {
			return e
		}
	}
	if p.trailing != nil {
		if e := p.trailing.Close(); e != nil {
			return e
		}
		p.trailing = nil
	}
	if p.extract.self != nil {
		if e := p.extract.Close(); e != nil {
			return e
		}
	}
	// The caller's default parameter full-expression outlives the constructor.
	// The outer owner retains its context even if base construction failed.
	if p.polynomial != nil {
		if e := p.polynomial.Close(); e != nil {
			return e
		}
		p.polynomial = nil
	}
	if e := p.endDefault(p.context); e != nil {
		return e
	}
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
