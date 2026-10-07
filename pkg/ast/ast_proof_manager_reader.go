package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Concrete lexical ptr_vector<expr/sort>, with no reference callbacks.
// Unlike ptr_buffer it has no inline elements and follows vector growth.
type astProofRawArguments struct {
	values  []ASTHandle
	charged uint64
}

func (v *astProofRawArguments) push(m *ASTManager, h ASTHandle) error {
	if len(v.values) == cap(v.values) {
		n := uint64(2)
		if cap(v.values) != 0 {
			n = (3*uint64(cap(v.values)) + 1) / 2
		}
		width := uint64(reflect.TypeOf(ASTHandle{}).Size())
		if n <= uint64(cap(v.values)) || n > (math.MaxUint32-8)/8 || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := m.BudgetField.Reserve(bytes); e != nil {
			return e
		}
		next := make([]ASTHandle, len(v.values), int(n))
		// Source trivial realloc preserves ended capacity cells too.
		copy(next[:cap(next)], v.values[:cap(v.values)])
		m.BudgetField.Release(v.charged)
		v.charged = bytes
		v.values = next
	}
	v.values = append(v.values, h)
	return nil
}
func (v *astProofRawArguments) close(m *ASTManager) {
	m.BudgetField.Release(v.charged)
	v.values = nil
	v.charged = 0
}

// ASTProofBindings owns only scalar pointer-vector storage, not AST references.
// It supplies the append-only output object required by is_quant_inst.
type ASTProofBindings struct {
	self       *ASTProofBindings
	manager    *ASTManager
	storage    astProofRawArguments
	ownerBytes uint64
	closed     bool
}

func NewASTProofBindings(m *ASTManager) (*ASTProofBindings, error) {
	if e := m.lock(); e != nil {
		return nil, e
	}
	defer m.mu.Unlock()
	n := uint64(reflect.TypeOf(ASTProofBindings{}).Size())
	if e := m.BudgetField.Reserve(n); e != nil {
		return nil, e
	}
	v := &ASTProofBindings{manager: m, ownerBytes: n}
	v.self = v
	return v, nil
}
func (v *ASTProofBindings) check(m *ASTManager) error {
	if v == nil || v.self != v || v.manager != m {
		return ErrASTHandle
	}
	if v.closed {
		return arena.ErrClosed
	}
	return nil
}
func (v *ASTProofBindings) Len() uint32 {
	if v == nil || v.closed {
		return 0
	}
	return uint32(len(v.storage.values))
}
func (v *ASTProofBindings) At(i uint32) (ASTHandle, error) {
	if v == nil {
		return ASTHandle{}, ErrASTHandle
	}
	if e := v.check(v.manager); e != nil {
		return ASTHandle{}, e
	}
	if e := v.manager.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer v.manager.mu.Unlock()
	if uint64(i) >= uint64(len(v.storage.values)) {
		return ASTHandle{}, arena.ErrBounds
	}
	return v.storage.values[i], nil
}
func (v *ASTProofBindings) Reset() error {
	if v == nil {
		return ErrASTHandle
	}
	if e := v.check(v.manager); e != nil {
		return e
	}
	if e := v.manager.lock(); e != nil {
		return e
	}
	defer v.manager.mu.Unlock()
	v.storage.values = v.storage.values[:0]
	return nil
}
func (v *ASTProofBindings) Close() error {
	if v == nil || v.self != v {
		return ErrASTHandle
	}
	if v.closed {
		return nil
	}
	// No AST callbacks: storage can be refunded after the borrowed manager's
	// forced teardown too. The budget remains borrowed from its caller.
	v.storage.close(v.manager)
	v.manager.BudgetField.Release(v.ownerBytes)
	v.ownerBytes = 0
	v.closed = true
	return nil
}
func (m *ASTManager) MatchProofQuantInstantiation(p ASTHandle, out *ASTHandle, binding *ASTProofBindings) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	yes, e := m.proofKind(p, ASTPrQuantInst)
	if e != nil || !yes {
		return false, e
	}
	f, e := m.proofArg(p, 0)
	if e != nil {
		return false, e
	}
	if out == nil {
		return false, ErrASTHandle
	}
	*out = f
	r, e := m.requireKind(p, ASTApp)
	if e != nil {
		return false, e
	}
	d := m.declAt(m.appAt(r.slot).decl.slot)
	if e = binding.check(m); e != nil {
		return false, e
	}
	if ASTSourceDebug && len(binding.storage.values) != 0 {
		return false, ErrASTDomain
	}
	if d.info.slot == 0 {
		return true, nil
	}
	info := m.infoAt(d.info.slot)
	for i := uint32(0); i < info.count; i++ {
		param, e := m.infoParameter(info, i)
		if e != nil {
			return false, e
		}
		record, e := m.parameter(param)
		if e != nil {
			return false, e
		}
		if record.Kind != ASTParameterAST {
			return false, ErrASTDomain
		}
		// get_ast/to_expr checks at the source's parameter extraction site;
		// no global prevalidation hides prior output or append prefixes.
		h := m.external(record.ast)
		if _, e = m.requireExpr(h); e != nil {
			return false, e
		}
		if e = binding.storage.push(m, h); e != nil {
			return false, e
		}
	}
	return true, nil
}
