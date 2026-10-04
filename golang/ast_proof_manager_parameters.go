// Licensed under the MIT license.
package g3

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

// Lexical vector<parameter>. Entries own concrete parameter records; AST
// values are borrowed until declaration registration acquires its references.
type astProofParameters struct {
	values  []ASTParameter
	charged uint64
}

func (v *astProofParameters) pushOwned(m *ASTManager, p ASTParameter) error {
	if len(v.values) == cap(v.values) {
		n := uint64(2)
		if cap(v.values) != 0 {
			n = (3*uint64(cap(v.values)) + 1) / 2
		}
		width := uint64(reflect.TypeOf(ASTParameter{}).Size())
		if n <= uint64(cap(v.values)) || n > (math.MaxUint32-8)/16 || n > uint64(^uint(0)>>1)/width {
			m.dropParameter(p)
			return arena.ErrBounds
		}
		bytes := n * width
		if e := m.budget.Reserve(bytes); e != nil {
			m.dropParameter(p)
			return e
		}
		next := make([]ASTParameter, len(v.values), int(n))
		copy(next, v.values)
		m.budget.Release(v.charged)
		v.charged = bytes
		v.values = next
	}
	v.values = append(v.values, p)
	return nil
}
func (v *astProofParameters) close(m *ASTManager) {
	for _, p := range v.values {
		m.dropParameter(p)
	}
	m.budget.Release(v.charged)
	v.values = nil
	v.charged = 0
}
func (m *ASTManager) MkProofQuantInstantiation(f ASTHandle, bindings []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	var parameters astProofParameters
	defer parameters.close(m)
	for _, h := range bindings {
		p, e := m.newParameterAST(h)
		if e != nil {
			return ASTHandle{}, e
		}
		if e = parameters.pushOwned(m, p); e != nil {
			return ASTHandle{}, e
		}
		// The source assertion is after push, and parameter(AST) always has
		// this tag even when its borrowed AST pointer is null.
		if astSourceDebug && m.parameterAt(p.slot).kind != ASTParameterAST {
			return ASTHandle{}, ErrASTDomain
		}
	}
	args := [1]ASTHandle{f}
	r, e := m.mkFamilyApp(0, ASTPrQuantInst, parameters.values, args[:])
	return m.external(r), e
}
func (m *ASTManager) MkProofTheoryLemma(tid FamilyID, f ASTHandle, ps []ASTHandle, params []ASTParameter) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	var args astProofArguments
	defer args.close(m)
	var parameters astProofParameters
	defer parameters.close(m)
	s, e := m.families.GetName(tid)
	if e != nil {
		return ASTHandle{}, e
	}
	p, e := m.newParameterSymbol(s)
	if e != nil {
		return ASTHandle{}, e
	}
	if e = parameters.pushOwned(m, p); e != nil {
		return ASTHandle{}, e
	}
	for _, param := range params {
		p, e = m.copyParameter(param)
		if e != nil {
			return ASTHandle{}, e
		}
		if e = parameters.pushOwned(m, p); e != nil {
			return ASTHandle{}, e
		}
	}
	if e = args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	if e = args.push(m, f); e != nil {
		return ASTHandle{}, e
	}
	r, e := m.mkFamilyApp(0, ASTPrTheoryLemma, parameters.values, args.slice())
	return m.external(r), e
}
