// Source: src/ast/rewriter/bit_blaster/bit_blaster_rewriter.h:22-41;
// src/ast/rewriter/bit_blaster/bit_blaster_rewriter.cpp:69-210,625-684.
package bit_blaster

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	parent "github.com/glycerine/g3/pkg/ast/rewriter"
)

type RewriterParams struct {
	MaxSteps    uint32
	MaxMemoryMB uint32
	BlastMul    bool
	BlastAdd    bool
	BlastQuant  bool
	BlastFull   bool
}

func DefaultRewriterParams() RewriterParams {
	return RewriterParams{MaxSteps: ^uint32(0), MaxMemoryMB: ^uint32(0), BlastMul: true, BlastAdd: true}
}

type BitBlasterRewriter struct {
	self      *BitBlasterRewriter
	manager   *ast.ASTManager
	family    ast.FamilyID
	blaster   *Engine
	params    RewriterParams
	config    bitBlasterTraversalConfiguration
	traversal *parent.Traversal
	cache     map[ast.ASTHandle]ast.ASTRef
	keys      []ast.ASTHandle
	limits    []int
	bindings  []ast.ASTRef
	closed    bool
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
	config, err := newRewriterConfig(m, family)
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
	if err = engine.SetMaxMemoryMB(actual.MaxMemoryMB); err != nil {
		return nil, err
	}
	r := &BitBlasterRewriter{
		manager: m,
		family:  family,
		blaster: engine,
		params:  actual,
		cache:   make(map[ast.ASTHandle]ast.ASTRef),
	}
	r.self, r.config.owner = r, r
	r.traversal, err = parent.NewTraversal(m, &r.config)
	if err != nil {
		if r.traversal != nil {
			err = errors.Join(err, r.traversal.Close())
		}
		return r, err
	}
	return r, nil
}

func (r *BitBlasterRewriter) open() error {
	if r == nil || r.self != r || r.closed || r.manager == nil || r.blaster == nil || r.traversal == nil {
		return ast.ErrASTHandle
	}
	return nil
}

func (r *BitBlasterRewriter) UpdateParams(params RewriterParams) error {
	if err := r.open(); err != nil {
		return err
	}
	r.params = params
	return r.blaster.SetMaxMemoryMB(params.MaxMemoryMB)
}

func (r *BitBlasterRewriter) Manager() (*ast.ASTManager, error) {
	if err := r.open(); err != nil {
		return nil, err
	}
	return r.manager, nil
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
	if r == nil || r.self != r || r.manager == nil {
		return ast.ErrASTHandle
	}
	if r.closed {
		return nil
	}
	if r.traversal != nil {
		result = errors.Join(result, r.traversal.Close())
	}
	result = errors.Join(result, r.releaseBindings(0))
	for _, key := range r.keys {
		ref := r.cache[key]
		result = errors.Join(result, r.manager.DelRef(&ref))
	}
	r.cache = nil
	r.keys = nil
	r.limits = nil
	r.bindings = nil
	r.closed = true
	return result
}
