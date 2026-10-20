// Source: src/ast/rewriter/bit_blaster/bit_blaster_rewriter.h:22-41;
// src/ast/rewriter/bit_blaster/bit_blaster_rewriter.cpp:69-210,625-684.
package bit_blaster

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type RewriterParams struct {
	MaxSteps   uint32
	BlastMul   bool
	BlastAdd   bool
	BlastQuant bool
	BlastFull  bool
}

func DefaultRewriterParams() RewriterParams {
	return RewriterParams{MaxSteps: ^uint32(0), BlastMul: true, BlastAdd: true}
}

type BitBlasterRewriter struct {
	manager *ast.ASTManager
	family  ast.FamilyID
	blaster *Engine
	params  RewriterParams
	cache   map[ast.ASTHandle]ast.ASTRef
	keys    []ast.ASTHandle
	limits  []int
	closed  bool
}

func NewBitBlasterRewriter(m *ast.ASTManager, params *RewriterParams) (*BitBlasterRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	name, err := m.Symbols.Intern("bv")
	if err != nil {
		return nil, err
	}
	family, err := m.FamilyIDForSymbol(name)
	if err != nil {
		return nil, err
	}
	config, err := NewConfig(m, family, SMTBitBlasterParams{})
	if err != nil {
		return nil, err
	}
	engine, err := NewEngine(config)
	if err != nil {
		return nil, err
	}
	actual := DefaultRewriterParams()
	if params != nil {
		actual = *params
	}
	return &BitBlasterRewriter{
		manager: m,
		family:  family,
		blaster: engine,
		params:  actual,
		cache:   make(map[ast.ASTHandle]ast.ASTRef),
	}, nil
}

func (r *BitBlasterRewriter) open() error {
	if r == nil || r.closed || r.manager == nil || r.blaster == nil {
		return ast.ErrASTHandle
	}
	return nil
}

func (r *BitBlasterRewriter) UpdateParams(params RewriterParams) error {
	if err := r.open(); err != nil {
		return err
	}
	r.params = params
	return nil
}

func (r *BitBlasterRewriter) Push() error {
	if err := r.open(); err != nil {
		return err
	}
	r.limits = append(r.limits, len(r.keys))
	return nil
}

func (r *BitBlasterRewriter) NumScopes() (uint32, error) {
	if err := r.open(); err != nil {
		return 0, err
	}
	return uint32(len(r.limits)), nil
}

func (r *BitBlasterRewriter) Pop(scopes uint32) (result error) {
	if err := r.open(); err != nil {
		return err
	}
	if uint64(scopes) > uint64(len(r.limits)) {
		return ast.ErrASTDomain
	}
	if scopes == 0 {
		return nil
	}
	newScopeCount := len(r.limits) - int(scopes)
	limit := r.limits[newScopeCount]
	for i := len(r.keys); i > limit; {
		i--
		key := r.keys[i]
		ref := r.cache[key]
		delete(r.cache, key)
		result = errors.Join(result, r.manager.DelRef(&ref))
	}
	r.keys = r.keys[:limit]
	r.limits = r.limits[:newScopeCount]
	return result
}

func (r *BitBlasterRewriter) Const2Bits() (map[ast.ASTHandle]ast.ASTHandle, error) {
	if err := r.open(); err != nil {
		return nil, err
	}
	result := make(map[ast.ASTHandle]ast.ASTHandle, len(r.cache))
	for key, ref := range r.cache {
		h, err := r.manager.RefGet(ref)
		if err != nil {
			return nil, err
		}
		result[key] = h
	}
	return result, nil
}

func (r *BitBlasterRewriter) getBits(term ast.ASTHandle) (bits []ast.ASTRef, result error) {
	assembled, err := r.manager.IsBitVectorOperation(term, 56)
	if err != nil {
		return nil, err
	}
	defer func() {
		if result != nil {
			result = errors.Join(result, releaseRefs(r.manager, bits))
			bits = nil
		}
	}()
	if assembled {
		count, err := r.manager.GetNumArgs(term)
		if err != nil {
			return nil, err
		}
		for i := uint32(0); i < count; i++ {
			bit, err := r.manager.GetArg(term, i)
			if err != nil {
				return nil, err
			}
			ref, err := r.manager.NewRef(bit)
			if err != nil {
				return nil, err
			}
			bits = append(bits, ref)
		}
		return bits, nil
	}
	width, err := r.manager.BitVectorSize(term)
	if err != nil {
		return nil, err
	}
	for i := uint32(0); i < width; i++ {
		bit, err := r.manager.MkBitVectorBit2Bool(i, term)
		if err != nil {
			return nil, err
		}
		ref, err := r.manager.NewRef(bit)
		if err != nil {
			return nil, err
		}
		bits = append(bits, ref)
	}
	return bits, nil
}

func (r *BitBlasterRewriter) setBitVector(bits []ast.ASTRef, out ast.ASTRef) error {
	handles, err := r.blaster.handles(bits)
	if err != nil {
		return err
	}
	assembled, err := r.manager.MkBitVectorFromBits(handles)
	if err != nil {
		return err
	}
	return r.manager.SetRef(out, assembled)
}

func (r *BitBlasterRewriter) makeConstant(decl ast.ASTHandle, out ast.ASTRef) (result error) {
	if cached, ok := r.cache[decl]; ok {
		return r.manager.AssignRef(out, cached)
	}
	rangeSort, err := r.manager.GetRange(decl)
	if err != nil {
		return err
	}
	width, err := r.manager.BitVectorSize(rangeSort)
	if err != nil {
		return err
	}
	boolSort, err := r.manager.MkBoolSort()
	if err != nil {
		return err
	}
	bits := make([]ast.ASTRef, 0, width)
	defer func() { result = errors.Join(result, releaseRefs(r.manager, bits)) }()
	for i := uint32(0); i < width; i++ {
		bit, err := r.manager.MkFreshConstNullable(nil, boolSort)
		if err != nil {
			return err
		}
		ref, err := r.manager.NewRef(bit)
		if err != nil {
			return err
		}
		bits = append(bits, ref)
	}
	handles, err := r.blaster.handles(bits)
	if err != nil {
		return err
	}
	assembled, err := r.manager.MkBitVectorFromBits(handles)
	if err != nil {
		return err
	}
	owned, err := r.manager.NewRef(assembled)
	if err != nil {
		return err
	}
	if err = r.manager.SetRef(out, assembled); err != nil {
		return errors.Join(err, r.manager.DelRef(&owned))
	}
	r.cache[decl] = owned
	r.keys = append(r.keys, decl)
	return nil
}

func (r *BitBlasterRewriter) Close() (result error) {
	if err := r.open(); err != nil {
		return err
	}
	for _, key := range r.keys {
		ref := r.cache[key]
		result = errors.Join(result, r.manager.DelRef(&ref))
	}
	r.cache = nil
	r.keys = nil
	r.limits = nil
	r.closed = true
	return result
}
