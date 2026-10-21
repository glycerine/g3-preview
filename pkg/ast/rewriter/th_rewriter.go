// Source: src/ast/rewriter/th_rewriter.cpp;
// src/ast/rewriter/th_rewriter.h.
package rewriter

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type theoryRewriterConfiguration struct {
	DefaultConfiguration
	owner                *TheoryRewriter
	boolean              *BoolRewriter
	arithmetic           *ArithRewriter
	bitVector            *BVRewriter
	array                *ArrayRewriter
	datatype             *DatatypeRewriter
	floating             *FPARewriter
	datalog              *DLRewriter
	pseudoBoolean        *PseudoBooleanRewriter
	sequence             *SequenceRewriter
	maxMemory            uint64
	maxSteps             uint32
	pullCheapITE         bool
	flat                 bool
	cacheAll             bool
	pushITEArithmetic    bool
	pushITEBitVector     bool
	ignoreGroundPatterns bool
	rewritePatternTerms  bool
}

func (c *theoryRewriterConfiguration) CacheAllResults() bool { return c.cacheAll }
func (c *theoryRewriterConfiguration) RewritePatterns() bool { return c.rewritePatternTerms }
func (c *theoryRewriterConfiguration) MaxStepsExceeded(steps uint32) (bool, error) {
	if c.owner.manager.Budget().Used() > c.maxMemory {
		return false, newRewriterException("max. memory exceeded")
	}
	return steps > c.maxSteps, nil
}
func (c *theoryRewriterConfiguration) FlatAssoc(decl ast.ASTHandle) (bool, error) {
	if !c.flat {
		return false, nil
	}
	family, err := c.owner.manager.GetFamilyID(decl)
	if err != nil || family == ast.NullFamilyID {
		return false, err
	}
	kind, err := c.owner.manager.GetDeclKind(decl)
	if err != nil {
		return false, err
	}
	if family == c.owner.manager.BasicFamilyID() {
		return kind == ast.ASTOpAnd || kind == ast.ASTOpOr, nil
	}
	if family == c.arithmetic.family {
		return kind == arithOpAdd, nil
	}
	if family == c.bitVector.bvFamily {
		return kind == 4 || kind == 31 || kind == 30 || kind == 33, nil
	}
	return false, nil
}

func (c *theoryRewriterConfiguration) bitNumeral(h ast.ASTHandle) (uint32, bool, error) {
	service := c.owner.manager.NumeralService()
	value, err := service.NewValue()
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = service.Del(&value) }()
	var width uint32
	matched, err := c.owner.manager.BitVectorNumeralValue(h, &value, &width)
	if err != nil || !matched {
		return 0, matched, err
	}
	if width != 1 {
		return 0, false, nil
	}
	valid, err := service.IsUint64(&value)
	if err != nil || !valid {
		return 0, false, err
	}
	v, err := service.Uint64(&value)
	if err != nil || v > 1 {
		return 0, false, err
	}
	return uint32(v), true, nil
}

func (c *theoryRewriterConfiguration) equalityBit(term ast.ASTHandle) (x ast.ASTHandle, value uint32, matched bool, err error) {
	kind, err := c.owner.manager.GetKind(term)
	if err != nil || kind != ast.ASTApp {
		return x, 0, false, err
	}
	family, err := c.owner.manager.GetFamilyID(term)
	if err != nil || family != c.owner.manager.BasicFamilyID() {
		return x, 0, false, err
	}
	op, err := c.owner.manager.GetDeclKind(term)
	if err != nil || op != ast.ASTOpEq {
		return x, 0, false, err
	}
	if count, e := c.owner.manager.GetNumArgs(term); e != nil || count != 2 {
		return x, 0, false, e
	}
	left, err := c.owner.manager.GetArg(term, 0)
	if err != nil {
		return x, 0, false, err
	}
	isBV, err := c.owner.manager.IsBitVector(left)
	if err != nil || !isBV {
		return x, 0, false, err
	}
	width, err := c.owner.manager.BitVectorSize(left)
	if err != nil || width != 1 {
		return x, 0, false, err
	}
	right, err := c.owner.manager.GetArg(term, 1)
	if err != nil {
		return x, 0, false, err
	}
	if v, yes, e := c.bitNumeral(left); e != nil || yes {
		return right, v, yes, e
	}
	if v, yes, e := c.bitNumeral(right); e != nil || yes {
		return left, v, yes, e
	}
	return x, 0, false, nil
}

func (c *theoryRewriterConfiguration) tamagotchi(left, right ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	x, value, matched, err := c.equalityBit(left)
	condition := right
	if err != nil {
		return RewriteFailed, err
	}
	if !matched {
		x, value, matched, err = c.equalityBit(right)
		condition = left
		if err != nil || !matched {
			return RewriteFailed, err
		}
	}
	thenValue, err := c.owner.manager.BitVectorNumeralUnsigned(value, 1)
	if err != nil {
		return RewriteFailed, err
	}
	elseValue, err := c.owner.manager.BitVectorNumeralUnsigned(1-value, 1)
	if err != nil {
		return RewriteFailed, err
	}
	ite, err := c.owner.manager.MkIte(condition, thenValue, elseValue)
	if err != nil {
		return RewriteFailed, err
	}
	equality, err := c.owner.manager.MkEq(x, ite)
	if err != nil {
		return RewriteFailed, err
	}
	return Rewrite2, c.owner.manager.SetRef(out, equality)
}

func (c *theoryRewriterConfiguration) reduceAppCore(decl ast.ASTHandle, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	m := c.owner.manager
	family, err := m.GetFamilyID(decl)
	if err != nil || family == ast.NullFamilyID {
		return RewriteFailed, err
	}
	if family == m.BasicFamilyID() {
		kind, err := m.GetDeclKind(decl)
		if err != nil {
			return RewriteFailed, err
		}
		if kind == ast.ASTOpEq {
			if args.Count() != 2 {
				return RewriteFailed, ast.ErrASTDomain
			}
			left, err := args.At(0)
			if err != nil {
				return RewriteFailed, err
			}
			right, err := args.At(1)
			if err != nil {
				return RewriteFailed, err
			}
			sort, err := m.GetSort(left)
			if err != nil {
				return RewriteFailed, err
			}
			sortFamily, err := m.GetFamilyID(sort)
			if err != nil {
				return RewriteFailed, err
			}
			status := RewriteFailed
			switch sortFamily {
			case c.arithmetic.family:
				status, err = c.arithmetic.MkEqCore(left, right, out)
			case c.bitVector.bvFamily:
				status, err = c.bitVector.RewriteEquality(left, right, out)
			case c.datatype.family:
				status, err = c.datatype.RewriteEquality(left, right, out)
			case c.floating.family:
				status, err = c.floating.MkEqCore(left, right, out)
			case c.sequence.family:
				status, err = c.sequence.RewriteEquality(left, right, out)
			}
			if err != nil || status != RewriteFailed {
				return status, err
			}
		}
		if kind == ast.ASTOpEq || kind == ast.ASTOpIff {
			if args.Count() != 2 {
				return RewriteFailed, ast.ErrASTDomain
			}
			left, err := args.At(0)
			if err != nil {
				return RewriteFailed, err
			}
			right, err := args.At(1)
			if err != nil {
				return RewriteFailed, err
			}
			if status, err := c.tamagotchi(left, right, out); err != nil || status != RewriteFailed {
				return status, err
			}
		}
		if kind == ast.ASTOpIte {
			if args.Count() != 3 {
				return RewriteFailed, ast.ErrASTDomain
			}
			condition, err := args.At(0)
			if err != nil {
				return RewriteFailed, err
			}
			thenValue, err := args.At(1)
			if err != nil {
				return RewriteFailed, err
			}
			elseValue, err := args.At(2)
			if err != nil {
				return RewriteFailed, err
			}
			sort, err := m.GetSort(thenValue)
			if err != nil {
				return RewriteFailed, err
			}
			sortFamily, err := m.GetFamilyID(sort)
			if err != nil {
				return RewriteFailed, err
			}
			if sortFamily == c.bitVector.bvFamily {
				if status, err := c.bitVector.RewriteIte(condition, thenValue, elseValue, out); err != nil || status != RewriteFailed {
					return status, err
				}
			}
		}
		return c.boolean.MkAppCore(decl, args, out)
	}
	switch family {
	case c.arithmetic.family:
		return c.arithmetic.MkAppCore(decl, args, out)
	case c.bitVector.bvFamily:
		return c.bitVector.RewriteApp(decl, args, out)
	case c.array.family:
		return c.array.MkAppCore(decl, args, out)
	case c.datatype.family:
		return c.datatype.RewriteApp(decl, args, out)
	case c.floating.family:
		return c.floating.MkAppCore(decl, args, out)
	case c.datalog.family:
		return c.datalog.RewriteApp(decl, args, out)
	case c.pseudoBoolean.family:
		return c.pseudoBoolean.RewriteApp(decl, args, out)
	case c.sequence.family:
		return c.sequence.RewriteApp(decl, args, out)
	default:
		return RewriteFailed, nil
	}
}

func (c *theoryRewriterConfiguration) ReduceApp(decl ast.ASTHandle, args ast.AppArguments, out, proof ast.ASTRef) (RewriteStatus, error) {
	if err := c.owner.manager.SetRef(proof, ast.ASTHandle{}); err != nil {
		return RewriteFailed, err
	}
	return c.reduceAppCore(decl, args, out)
}

// TheoryRewriter applies all retained cheap theory reducers through one
// traversal. This first source slice supplies construction and core dispatch;
// the source pull/push-ITE and substitution extensions are added separately.
type TheoryRewriter struct {
	self          *TheoryRewriter
	manager       *ast.ASTManager
	context       *uparams.ParamsContext
	globals       *uparams.GlobalParamsService
	local         *uparams.ParamsRef
	configuration theoryRewriterConfiguration
	traversal     *Traversal
	header        uint64
	closed        bool
}

func NewTheoryRewriter(m *ast.ASTManager, context *uparams.ParamsContext, globals *uparams.GlobalParamsService, local *uparams.ParamsRef) (*TheoryRewriter, error) {
	if m == nil || context == nil || globals == nil || context.SymbolsField != m.Symbols || globals.Context != context {
		return nil, uparams.ErrParamsOwner
	}
	if err := m.RegisterDeclarationPlugins(); err != nil {
		return nil, err
	}
	n := uint64(reflect.TypeOf(TheoryRewriter{}).Size())
	if err := m.Budget().Reserve(n); err != nil {
		return nil, err
	}
	r := &TheoryRewriter{manager: m, context: context, globals: globals, local: local, header: n}
	r.self = r
	r.configuration.owner = r
	if err := r.constructConfiguration(); err != nil {
		return r, err
	}
	var err error
	if r.traversal, err = NewTraversal(m, &r.configuration); err != nil {
		return r, err
	}
	return r, nil
}

func (r *TheoryRewriter) constructConfiguration() (err error) {
	c := &r.configuration
	legacy, err := r.globals.Manager()
	if err != nil {
		return err
	}
	if c.boolean, err = NewBoolRewriter(r.manager, r.context, r.globals, r.local); err != nil {
		return err
	}
	if c.arithmetic, err = NewArithRewriter(r.manager, r.context, r.globals, r.local); err != nil {
		return err
	}
	if c.bitVector, err = NewBVRewriter(r.manager, r.context, legacy, r.local); err != nil {
		return err
	}
	if c.array, err = NewArrayRewriter(r.manager, r.context, r.globals, r.local); err != nil {
		return err
	}
	if c.datatype, err = NewDatatypeRewriter(r.manager); err != nil {
		return err
	}
	if c.floating, err = NewFPARewriter(r.manager); err != nil {
		return err
	}
	if c.datalog, err = NewDLRewriter(r.manager); err != nil {
		return err
	}
	if c.pseudoBoolean, err = NewPseudoBooleanRewriter(r.manager); err != nil {
		return err
	}
	if c.sequence, err = NewSequenceRewriter(r.manager); err != nil {
		return err
	}
	return r.readLocalParams()
}

func (r *TheoryRewriter) readLocalParams() (result error) {
	p, err := NewRewriterParams(r.context, r.globals, r.local)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, p.Close()) }()
	c := &r.configuration
	megabytes, err := p.MaxMemory()
	if err != nil {
		return err
	}
	c.maxMemory = uint64(megabytes) * 1024 * 1024
	if c.maxSteps, err = p.MaxSteps(); err != nil {
		return err
	}
	if c.pullCheapITE, err = p.PullCheapITE(); err != nil {
		return err
	}
	if c.flat, err = p.Flat(); err != nil {
		return err
	}
	if c.cacheAll, err = p.CacheAll(); err != nil {
		return err
	}
	if c.pushITEArithmetic, err = p.PushITEArith(); err != nil {
		return err
	}
	if c.pushITEBitVector, err = p.PushITEBV(); err != nil {
		return err
	}
	if c.ignoreGroundPatterns, err = p.IgnorePatternsOnGroundQBody(); err != nil {
		return err
	}
	c.rewritePatternTerms, err = p.RewritePatterns()
	return err
}

func (r *TheoryRewriter) open() error {
	if r == nil || r.self != r || r.closed || r.traversal == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

func (r *TheoryRewriter) Rewrite(term ast.ASTHandle, out ast.ASTRef) error {
	if err := r.open(); err != nil {
		return err
	}
	return r.traversal.Rewrite(term, out)
}

func (r *TheoryRewriter) MakeApp(decl ast.ASTHandle, args ast.AppArguments, out ast.ASTRef) error {
	if err := r.open(); err != nil {
		return err
	}
	if err := r.manager.SetRef(out, ast.ASTHandle{}); err != nil {
		return err
	}
	status, err := r.configuration.reduceAppCore(decl, args, out)
	if err != nil {
		return err
	}
	if status != RewriteFailed {
		return nil
	}
	h, err := r.manager.RewriterMakeApp(decl, args)
	if err != nil {
		return err
	}
	return r.manager.SetRef(out, h)
}

func (r *TheoryRewriter) NumSteps() (uint32, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return r.traversal.NumSteps()
}

func (r *TheoryRewriter) Reset() error {
	if err := r.open(); err != nil {
		return err
	}
	return r.traversal.Reset()
}

func (r *TheoryRewriter) closeConfiguration() error {
	c := &r.configuration
	c.sequence = nil
	if c.pseudoBoolean != nil {
		if err := c.pseudoBoolean.Close(); err != nil {
			return err
		}
		c.pseudoBoolean = nil
	}
	if c.datalog != nil {
		if err := c.datalog.Close(); err != nil {
			return err
		}
		c.datalog = nil
	}
	if c.floating != nil {
		if err := c.floating.Close(); err != nil {
			return err
		}
		c.floating = nil
	}
	c.datatype = nil
	if c.array != nil {
		if err := c.array.Close(); err != nil {
			return err
		}
		c.array = nil
	}
	if c.bitVector != nil {
		if err := c.bitVector.Close(); err != nil {
			return err
		}
		c.bitVector = nil
	}
	if c.arithmetic != nil {
		if err := c.arithmetic.Close(); err != nil {
			return err
		}
		c.arithmetic = nil
	}
	if c.boolean != nil {
		if err := c.boolean.Close(); err != nil {
			return err
		}
		c.boolean = nil
	}
	return nil
}

func (r *TheoryRewriter) Close() error {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	if r.traversal != nil {
		if err := r.traversal.Close(); err != nil {
			return err
		}
		r.traversal = nil
	}
	if err := r.closeConfiguration(); err != nil {
		return err
	}
	r.manager.Budget().Release(r.header)
	r.header = 0
	r.closed = true
	return nil
}
