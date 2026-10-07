// Source: src/ast/bv_decl_plugin.cpp:67-142,181-212,249-357,470-519.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/util"
)

// The order after bcomp follows finalize(), rather than enum order.
var astBVFixedOrder = [...]int32{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 42, 43, 44, 52, 53, 54, 45, 46, 47, 50, 51}

func bvFixedName(k int32) string {
	switch k {
	case 3:
		return "bvneg"
	case 4:
		return "bvadd"
	case 5:
		return "bvsub"
	case 6:
		return "bvmul"
	case 7:
		return "bvsdiv"
	case 8:
		return "bvudiv"
	case 9:
		return "bvsrem"
	case 10:
		return "bvurem"
	case 11:
		return "bvsmod"
	case 12:
		return "bvsdiv0"
	case 13:
		return "bvudiv0"
	case 14:
		return "bvsrem0"
	case 15:
		return "bvurem0"
	case 16:
		return "bvsmod0"
	case 17:
		return "bvsdiv_i"
	case 18:
		return "bvudiv_i"
	case 19:
		return "bvsrem_i"
	case 20:
		return "bvurem_i"
	case 21:
		return "bvsmod_i"
	case 22:
		return "bvule"
	case 23:
		return "bvsle"
	case 24:
		return "bvuge"
	case 25:
		return "bvsge"
	case 26:
		return "bvult"
	case 27:
		return "bvslt"
	case 28:
		return "bvugt"
	case 29:
		return "bvsgt"
	case 30:
		return "bvand"
	case 31:
		return "bvor"
	case 32:
		return "bvnot"
	case 33:
		return "bvxor"
	case 34:
		return "bvnand"
	case 35:
		return "bvnor"
	case 36:
		return "bvxnor"
	case 42:
		return "bvredor"
	case 43:
		return "bvredand"
	case 44:
		return "bvcomp"
	case 45:
		return "bvshl"
	case 46:
		return "bvlshr"
	case 47:
		return "bvashr"
	case 50:
		return "ext_rotate_left"
	case 51:
		return "ext_rotate_right"
	case 52:
		return "bvumul_noovfl"
	case 53:
		return "bvsmul_noovfl"
	case 54:
		return "bvsmul_noudfl"
	default:
		return ""
	}
}

func (p *astBitVectorPluginOwner) fixedDecl(k int32, width uint32) (astNodeRef, error) {
	name := bvFixedName(k)
	if name == "" {
		return astNodeRef{}, nil
	}
	if width == math.MaxUint32 {
		return astNodeRef{}, ErrASTDomain
	}
	if p.fixedPending.slot != 0 {
		return astNodeRef{}, ErrASTPending
	}
	v := &p.fixed[k]
	if e := v.resizeNull(p.owner.BudgetField, width+1); e != nil {
		return astNodeRef{}, e
	}
	r, e := v.at(width)
	if e != nil || r.slot != 0 {
		return r, e
	}
	s, e := p.getSort(width)
	if e != nil {
		return astNodeRef{}, e
	}
	arity := 2
	if k == 3 || k >= 12 && k <= 16 || k == 32 || k == 42 || k == 43 {
		arity = 1
	}
	result := s
	if k >= 22 && k <= 29 || k >= 52 && k <= 54 {
		result = p.owner.boolSort
	}
	if k == 42 || k == 43 || k == 44 {
		result, e = p.getSort(1)
		if e != nil {
			return astNodeRef{}, e
		}
	}
	ac := k == 4 || k == 6 || k == 30 || k == 31 || k == 33
	flags := ASTFuncFlags{LeftAssociative: ac, RightAssociative: ac, FlatAssociative: ac, Commutative: ac || k == 44, Idempotent: k == 30 || k == 31}
	domain := [2]astNodeRef{s, s}
	r, e = p.owner.makePluginDeclaration(p.family, name, k, domain[:arity], result, flags)
	if e != nil {
		return astNodeRef{}, e
	}
	// Source assigns before INC. Remember an interrupted INC explicitly, never
	// infer acquisition from a non-null slot or the node's current reference count.
	v.words[2+width*2], v.words[3+width*2] = r.slot, r.generation
	p.fixedPending, p.fixedPendingKind, p.fixedPendingWidth = r, k, width
	if e = p.owner.incNode(r); e != nil {
		return r, e
	}
	p.fixedPending = astNodeRef{}
	return r, nil
}

func (m *ASTManager) bitVectorDecl(k int32, ps []ASTParameter, domain []ASTHandle) (astNodeRef, error) {
	p := m.bitVectorPlugin
	if p == nil || !p.published {
		return astNodeRef{}, ErrASTHandle
	}
	switch k {
	case 0:
		return m.bitVectorNumeralDecl(ps, len(domain))
	case 1:
		return p.fields[1], nil
	case 2:
		return p.fields[0], nil
	case 59:
		return p.fields[2], nil
	case 60:
		return p.fields[3], nil
	}
	// Numerals and parameterized operators require their own source factories.
	if k == 56 || k == 57 {
		return astNodeRef{}, ErrASTDomain
	}
	fail := func(message string) (astNodeRef, error) {
		return astNodeRef{}, ASTException{util.NewDefaultException(message)}
	}
	if len(domain) == 0 {
		return fail("no arguments supplied to bit-vector operator")
	}
	sort, e := m.requireKind(domain[0], ASTSort)
	if e != nil {
		return astNodeRef{}, e
	}
	info, e := m.sortInfo(m.sortAt(sort.slot).info)
	if e != nil {
		return astNodeRef{}, e
	}
	if info.family != p.family || info.kind != 0 {
		return fail("could not extract bit-vector size")
	}
	param, e := m.infoParameter(info, 0)
	if e != nil {
		return astNodeRef{}, e
	}
	value, e := m.parameter(param)
	if e != nil {
		return astNodeRef{}, e
	}
	if value.Kind != ASTParameterInt {
		return astNodeRef{}, ErrASTDomain
	}
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	r, e := p.fixedDecl(k, uint32(value.integer))
	if e != nil {
		return astNodeRef{}, e
	}
	if r.slot == 0 {
		return m.bitVectorParameterizedDecl(k, ps, domain, value.integer)
	}
	arity := int(m.declAt(r.slot).arity)
	count := len(domain)
	if count != arity {
		flags, e := m.funcFlags(r)
		if e != nil {
			return astNodeRef{}, e
		}
		if flags.LeftAssociative && flags.RightAssociative {
			count = arity
		} else {
			return fail("declared arity mismatches supplied arity")
		}
	}
	// C++ assumes an associative declaration has at least two supplied sorts.
	// Reject that undefined input rather than indexing outside the Go slice.
	if count > len(domain) {
		return astNodeRef{}, ErrASTDomain
	}
	for i := 0; i < count; i++ {
		expected, e := m.readInternal(m.declAt(r.slot).domain, uint32(i))
		if e != nil {
			return astNodeRef{}, e
		}
		if domain[i] != m.external(expected) {
			return fail("declared sorts do not match supplied sorts")
		}
	}
	return r, nil
}

func (p *astBitVectorPluginOwner) fixedBytes() uint64 {
	var n uint64
	for _, k := range astBVFixedOrder {
		n += p.fixed[k].bytes()
	}
	return n
}
func (p *astBitVectorPluginOwner) releaseFixed() {
	for _, k := range astBVFixedOrder {
		p.fixed[k].close(p.owner.BudgetField)
	}
}
func (p *astBitVectorPluginOwner) finalizeFixedPending(own bool) error {
	if p.fixedPending.slot == 0 {
		return nil
	}
	if !own {
		return ErrASTPending
	}
	v := &p.fixed[p.fixedPendingKind]
	i := p.fixedPendingWidth
	v.words[2+i*2], v.words[3+i*2] = 0, 0
	p.fixedPending = astNodeRef{}
	return nil
}
func (p *astBitVectorPluginOwner) finalizeFixed() error {
	for _, k := range astBVFixedOrder {
		v := &p.fixed[k]
		for i := uint32(0); i < v.size(); i++ {
			r, e := v.at(i)
			if e != nil {
				return e
			}
			if r.slot == 0 {
				continue
			}
			consumed, e := p.owner.decNode(r)
			if consumed {
				v.words[2+i*2], v.words[3+i*2] = 0, 0
			}
			if e != nil {
				if p.destructorCause == nil {
					p.destructorCause = e
				}
				return e
			}
		}
	}
	return nil
}
