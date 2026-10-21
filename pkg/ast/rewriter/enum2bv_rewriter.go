// Source: src/ast/rewriter/enum2bv_rewriter.cpp;
// src/ast/rewriter/enum2bv_rewriter.h.
package rewriter

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type EnumSortPredicate func(ast.ASTHandle) (bool, error)

type enum2BVEntry struct {
	original, bitVector, definition ast.ASTHandle
	owned                           [3]ast.ASTRef
}

type enum2BVConfiguration struct {
	DefaultConfiguration
	owner *Enum2BVRewriter
}

type Enum2BVRewriter struct {
	self        *Enum2BVRewriter
	manager     *ast.ASTManager
	config      enum2BVConfiguration
	traversal   *Traversal
	enum2BV     map[ast.ASTHandle]ast.ASTHandle
	bv2Enum     map[ast.ASTHandle]ast.ASTHandle
	enum2Def    map[ast.ASTHandle]ast.ASTHandle
	entries     []enum2BVEntry
	scopeLimits []int
	bounds      []ast.ASTHandle
	boundRefs   []ast.ASTRef
	flushedRefs []ast.ASTRef
	predicate   EnumSortPredicate
	translated  uint32
	header      uint64
	closed      bool
}

func NewEnum2BVRewriter(m *ast.ASTManager) (*Enum2BVRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	bytes := uint64(reflect.TypeOf(Enum2BVRewriter{}).Size())
	if err := m.Budget().Reserve(bytes); err != nil {
		return nil, err
	}
	r := &Enum2BVRewriter{
		manager: m, header: bytes,
		enum2BV:  make(map[ast.ASTHandle]ast.ASTHandle),
		bv2Enum:  make(map[ast.ASTHandle]ast.ASTHandle),
		enum2Def: make(map[ast.ASTHandle]ast.ASTHandle),
	}
	r.self = r
	r.config.owner = r
	r.config.defaults.manager = m
	traversal, err := NewTraversal(m, &r.config)
	if err != nil {
		if traversal != nil {
			_ = traversal.Close()
		}
		m.Budget().Release(bytes)
		return nil, err
	}
	r.traversal = traversal
	return r, nil
}

func (r *Enum2BVRewriter) open() error {
	if r == nil || r.self != r || r.closed || r.manager == nil || r.traversal == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

func enum2BVWidth(count uint32) uint32 {
	width := uint32(1)
	for width < 32 && uint64(1)<<width < uint64(count) {
		width++
	}
	return width
}

func enum2BVPowerOfTwo(value uint32) bool { return value != 0 && value&(value-1) == 0 }

func (r *Enum2BVRewriter) isFinite(sort ast.ASTHandle) (bool, error) {
	yes, err := r.manager.DatatypeIsEnumSort(sort)
	if err != nil || !yes || r.predicate == nil {
		return yes, err
	}
	return r.predicate(sort)
}

func (r *Enum2BVRewriter) retain(handle ast.ASTHandle) (ast.ASTRef, error) {
	return r.manager.NewRef(handle)
}

func (r *Enum2BVRewriter) reduceArgument(argument ast.ASTHandle) (ast.ASTHandle, bool, error) {
	m := r.manager
	sort, err := m.GetSort(argument)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	finite, err := r.isFinite(sort)
	if err != nil || !finite {
		return ast.ASTHandle{}, false, err
	}
	count, err := m.DatatypeNumConstructors(sort)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	width := enum2BVWidth(count)
	kind, err := m.GetKind(argument)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if kind == ast.ASTVar {
		index, e := m.GetVarIndex(argument)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		bvSort, e := m.BitVectorSort(width)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		value, e := m.MkVar(index, bvSort)
		return value, e == nil, e
	}
	if kind != ast.ASTApp {
		return ast.ASTHandle{}, false, fmt.Errorf("unable to handle nested data-type expression")
	}
	decl, err := m.GetDecl(argument)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if index, matched, e := m.DatatypeConstructorIndex(decl); e != nil {
		return ast.ASTHandle{}, false, e
	} else if matched {
		value, e := m.BitVectorNumeralUnsigned(index, width)
		if e == nil {
			r.translated++
		}
		return value, e == nil, e
	}
	declFamily, err := m.GetFamilyID(decl)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	arity, err := m.GetArity(decl)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if declFamily != ast.NullFamilyID || arity != 0 {
		return ast.ASTHandle{}, false, fmt.Errorf("unable to handle nested data-type expression")
	}
	if fresh, found := r.enum2BV[decl]; found {
		value, e := m.MkApp(fresh, nil)
		return value, e == nil, e
	}
	bvSort, err := m.BitVectorSort(width)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	name, err := m.GetASTName(decl)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	text, err := m.Symbols.Display(name)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	freshApp, err := m.MkFreshConst(text, bvSort)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	freshDecl, err := m.GetDecl(freshApp)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if !enum2BVPowerOfTwo(count) || count == 1 {
		upper, e := m.BitVectorNumeralUnsigned(count-1, width)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		bound, e := m.MkBitVectorULE(freshApp, upper)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		ref, e := r.retain(bound)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		r.bounds = append(r.bounds, bound)
		r.boundRefs = append(r.boundRefs, ref)
	}
	constructors, err := m.DatatypeConstructors(sort)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	definition, err := m.MkApp(constructors[count-1], nil)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	for i := count - 1; i > 0; {
		i--
		numeral, e := m.BitVectorNumeralUnsigned(i, width)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		equal, e := m.MkEq(freshApp, numeral)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		constructor, e := m.MkApp(constructors[i], nil)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
		definition, e = m.MkIte(equal, constructor, definition)
		if e != nil {
			return ast.ASTHandle{}, false, e
		}
	}
	entry := enum2BVEntry{original: decl, bitVector: freshDecl, definition: definition}
	for i, handle := range []ast.ASTHandle{decl, freshDecl, definition} {
		entry.owned[i], err = r.retain(handle)
		if err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = m.DelRef(&entry.owned[j])
			}
			return ast.ASTHandle{}, false, err
		}
	}
	r.entries = append(r.entries, entry)
	r.enum2BV[decl] = freshDecl
	r.bv2Enum[freshDecl] = decl
	r.enum2Def[decl] = definition
	r.translated++
	return freshApp, true, nil
}

func (c *enum2BVConfiguration) ReduceApp(decl ast.ASTHandle, args ast.AppArguments, out, _ ast.ASTRef) (RewriteStatus, error) {
	r := c.owner
	m := r.manager
	family, err := m.GetFamilyID(decl)
	if err != nil {
		return RewriteFailed, err
	}
	kind, err := m.GetDeclKind(decl)
	if err != nil {
		return RewriteFailed, err
	}
	if family == 0 && kind == ast.ASTOpEq && args.Count() == 2 {
		a, e := args.At(0)
		if e != nil {
			return RewriteFailed, e
		}
		b, e := args.At(1)
		if e != nil {
			return RewriteFailed, e
		}
		ra, ma, e := r.reduceArgument(a)
		if e != nil {
			return RewriteFailed, fmt.Errorf("enum2bv equality left: %w", e)
		}
		if ma {
			rb, mb, e := r.reduceArgument(b)
			if e != nil {
				return RewriteFailed, fmt.Errorf("enum2bv equality right: %w", e)
			}
			if mb {
				value, e := m.MkEq(ra, rb)
				if e != nil {
					return RewriteFailed, e
				}
				return RewriteDone, m.SetRef(out, value)
			}
		}
	}
	if family == 0 && kind == ast.ASTOpDistinct {
		converted := make([]ast.ASTHandle, args.Count())
		for i := range converted {
			a, e := args.At(uint32(i))
			if e != nil {
				return RewriteFailed, e
			}
			converted[i], _, e = r.reduceArgument(a)
			if e != nil {
				return RewriteFailed, e
			}
			if converted[i].IsNull() {
				break
			}
		}
		complete := true
		for _, h := range converted {
			complete = complete && !h.IsNull()
		}
		if complete {
			value, e := m.MkDistinct(converted)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, m.SetRef(out, value)
		}
	}
	if _, matched, e := m.DatatypeRecognizerConstructor(decl); e != nil {
		return RewriteFailed, e
	} else if matched && args.Count() == 1 {
		a, e := args.At(0)
		if e != nil {
			return RewriteFailed, e
		}
		ra, yes, e := r.reduceArgument(a)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			index, _, e := m.DatatypeRecognizerConstructorIndex(decl)
			if e != nil {
				return RewriteFailed, e
			}
			sort, e := m.GetSort(ra)
			if e != nil {
				return RewriteFailed, e
			}
			width, e := m.BitVectorSize(sort)
			if e != nil {
				return RewriteFailed, e
			}
			numeral, e := m.BitVectorNumeralUnsigned(index, width)
			if e != nil {
				return RewriteFailed, e
			}
			value, e := m.MkEq(ra, numeral)
			if e != nil {
				return RewriteFailed, e
			}
			return RewriteDone, m.SetRef(out, value)
		}
	}
	for i := uint32(0); i < args.Count(); i++ {
		a, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		s, e := m.GetSort(a)
		if e != nil {
			return RewriteFailed, e
		}
		finite, e := r.isFinite(s)
		if e != nil {
			return RewriteFailed, e
		}
		if finite {
			return RewriteFailed, fmt.Errorf("unable to handle nested data-type expression")
		}
	}
	return RewriteFailed, nil
}

func enum2BVArgumentSlice(arguments ast.AppArguments) ([]ast.ASTHandle, error) {
	result := make([]ast.ASTHandle, arguments.Count())
	for i := range result {
		value, err := arguments.At(uint32(i))
		if err != nil {
			return nil, err
		}
		result[i] = value
	}
	return result, nil
}

func (c *enum2BVConfiguration) ReduceQuantifier(q, body ast.ASTHandle, patterns, noPatterns ast.AppArguments, out, _ ast.ASTRef) (bool, error) {
	r := c.owner
	m := r.manager
	count, err := m.GetNumDeclarations(q)
	if err != nil {
		return false, err
	}
	sorts := make([]ast.ASTHandle, count)
	names := make([]usymbol.Symbol, count)
	var bounds []ast.ASTHandle
	found := false
	for i := uint32(0); i < count; i++ {
		sort, e := m.GetDeclarationSort(q, i)
		if e != nil {
			return false, e
		}
		names[i], e = m.GetDeclarationName(q, i)
		if e != nil {
			return false, e
		}
		finite, e := r.isFinite(sort)
		if e != nil {
			return false, e
		}
		if !finite {
			sorts[i] = sort
			continue
		}
		found = true
		constructors, e := m.DatatypeNumConstructors(sort)
		if e != nil {
			return false, e
		}
		width := enum2BVWidth(constructors)
		sorts[i], e = m.BitVectorSort(width)
		if e != nil {
			return false, e
		}
		if !enum2BVPowerOfTwo(constructors) || constructors == 1 {
			variable, e := m.MkVar(count-i-1, sorts[i])
			if e != nil {
				return false, e
			}
			upper, e := m.BitVectorNumeralUnsigned(constructors-1, width)
			if e != nil {
				return false, e
			}
			bound, e := m.MkBitVectorULE(variable, upper)
			if e != nil {
				return false, e
			}
			bounds = append(bounds, bound)
		}
	}
	if !found {
		return false, nil
	}
	forall, err := m.IsForall(q)
	if err != nil {
		return false, err
	}
	newBody := body
	if len(bounds) != 0 {
		if forall {
			condition, e := m.MkAndArray(bounds)
			if e != nil {
				return false, e
			}
			newBody, e = m.MkImplies(condition, body)
			if e != nil {
				return false, e
			}
		} else {
			bounds = append(bounds, body)
			newBody, err = m.MkAndArray(bounds)
			if err != nil {
				return false, err
			}
		}
	}
	newPatterns, err := enum2BVArgumentSlice(patterns)
	if err != nil {
		return false, err
	}
	newNoPatterns, err := enum2BVArgumentSlice(noPatterns)
	if err != nil {
		return false, err
	}
	weight, err := m.GetQuantifierWeight(q)
	if err != nil {
		return false, err
	}
	qid, skid, err := m.GetQuantifierIDs(q)
	if err != nil {
		return false, err
	}
	value, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: forall, Sorts: sorts, Names: names, Body: newBody,
		Weight: weight, QID: qid, SkolemID: skid,
		Patterns: newPatterns, NoPatterns: newNoPatterns,
	})
	if err != nil {
		return false, err
	}
	return true, m.SetRef(out, value)
}

func (r *Enum2BVRewriter) Rewrite(root ast.ASTHandle, out, proof ast.ASTRef) error {
	if err := r.open(); err != nil {
		return err
	}
	if err := r.traversal.Rewrite(root, out); err != nil {
		return err
	}
	proofs, err := r.manager.ProofsEnabled()
	if err != nil || !proofs {
		return err
	}
	value, err := r.manager.RefGet(out)
	if err != nil {
		return err
	}
	var p ast.ASTHandle
	if value == root {
		p, err = r.manager.MkProofReflexivity(root)
	} else {
		p, err = r.manager.MkProofRewrite(root, value)
	}
	if err != nil {
		return err
	}
	return r.manager.SetRef(proof, p)
}

func (r *Enum2BVRewriter) Push() error {
	if err := r.open(); err != nil {
		return err
	}
	r.scopeLimits = append(r.scopeLimits, len(r.entries))
	return nil
}

func (r *Enum2BVRewriter) Pop(scopes uint32) error {
	if err := r.open(); err != nil {
		return err
	}
	if len(r.bounds) != 0 || int(scopes) > len(r.scopeLimits) {
		return ast.ErrASTDomain
	}
	if scopes != 0 {
		level := len(r.scopeLimits) - int(scopes)
		limit := r.scopeLimits[level]
		for i := len(r.entries) - 1; i >= limit; i-- {
			entry := &r.entries[i]
			delete(r.enum2BV, entry.original)
			delete(r.bv2Enum, entry.bitVector)
			delete(r.enum2Def, entry.original)
			for j := 2; j >= 0; j-- {
				if err := r.manager.DelRef(&entry.owned[j]); err != nil {
					return err
				}
			}
		}
		r.entries = r.entries[:limit]
		r.scopeLimits = r.scopeLimits[:level]
	}
	return r.traversal.Reset()
}

func (r *Enum2BVRewriter) FlushSideConstraints(out *[]ast.ASTHandle) error {
	if err := r.open(); err != nil {
		return err
	}
	if out == nil {
		return ast.ErrASTDomain
	}
	*out = append(*out, r.bounds...)
	r.flushedRefs = append(r.flushedRefs, r.boundRefs...)
	r.bounds = nil
	r.boundRefs = nil
	return nil
}

func (r *Enum2BVRewriter) SetSortPredicate(predicate EnumSortPredicate) error {
	if err := r.open(); err != nil {
		return err
	}
	r.predicate = predicate
	return nil
}
func (r *Enum2BVRewriter) UpdateParams() error { return r.open() }

func (r *Enum2BVRewriter) Cleanup() (err error) {
	if e := r.open(); e != nil {
		return e
	}
	if e := r.traversal.Cleanup(); e != nil {
		return e
	}
	for i := len(r.boundRefs) - 1; i >= 0; i-- {
		err = errors.Join(err, r.manager.DelRef(&r.boundRefs[i]))
	}
	for i := len(r.flushedRefs) - 1; i >= 0; i-- {
		err = errors.Join(err, r.manager.DelRef(&r.flushedRefs[i]))
	}
	for i := len(r.entries) - 1; i >= 0; i-- {
		for j := 2; j >= 0; j-- {
			err = errors.Join(err, r.manager.DelRef(&r.entries[i].owned[j]))
		}
	}
	clear(r.enum2BV)
	clear(r.bv2Enum)
	clear(r.enum2Def)
	r.entries = nil
	r.scopeLimits = nil
	r.bounds = nil
	r.boundRefs = nil
	r.flushedRefs = nil
	r.predicate = nil
	r.translated = 0
	return err
}

func (r *Enum2BVRewriter) NumTranslated() (uint32, error) {
	if e := r.open(); e != nil {
		return 0, e
	}
	return r.translated, nil
}
func (r *Enum2BVRewriter) NumSteps() (uint32, error) {
	if e := r.open(); e != nil {
		return 0, e
	}
	return r.traversal.NumSteps()
}
func (r *Enum2BVRewriter) EnumToBitVector() map[ast.ASTHandle]ast.ASTHandle {
	out := make(map[ast.ASTHandle]ast.ASTHandle, len(r.enum2BV))
	for k, v := range r.enum2BV {
		out[k] = v
	}
	return out
}
func (r *Enum2BVRewriter) BitVectorToEnum() map[ast.ASTHandle]ast.ASTHandle {
	out := make(map[ast.ASTHandle]ast.ASTHandle, len(r.bv2Enum))
	for k, v := range r.bv2Enum {
		out[k] = v
	}
	return out
}
func (r *Enum2BVRewriter) EnumDefinitions() map[ast.ASTHandle]ast.ASTHandle {
	out := make(map[ast.ASTHandle]ast.ASTHandle, len(r.enum2Def))
	for k, v := range r.enum2Def {
		out[k] = v
	}
	return out
}

func (r *Enum2BVRewriter) Close() (err error) {
	if r == nil || r.self != r {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	if r.traversal != nil {
		err = errors.Join(err, r.traversal.Close())
	}
	for i := len(r.boundRefs) - 1; i >= 0; i-- {
		err = errors.Join(err, r.manager.DelRef(&r.boundRefs[i]))
	}
	for i := len(r.flushedRefs) - 1; i >= 0; i-- {
		err = errors.Join(err, r.manager.DelRef(&r.flushedRefs[i]))
	}
	for i := len(r.entries) - 1; i >= 0; i-- {
		for j := 2; j >= 0; j-- {
			err = errors.Join(err, r.manager.DelRef(&r.entries[i].owned[j]))
		}
	}
	r.manager.Budget().Release(r.header)
	r.header = 0
	r.closed = true
	return err
}

var _ Configuration = (*enum2BVConfiguration)(nil)
