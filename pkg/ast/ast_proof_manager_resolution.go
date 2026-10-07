package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Source debug svector<bool> found.setx retains repeated source growth and
// diagnoses coverage only after constructing the unmatched-literal buffer.
type astProofFound struct {
	values  []bool
	charged uint64
}

func (f *astProofFound) set(m *ASTManager, j uint32) error {
	if j == math.MaxUint32 {
		return arena.ErrBounds
	}
	need := uint64(j) + 1
	for need > uint64(cap(f.values)) {
		n := uint64(2)
		if cap(f.values) != 0 {
			n = (3*uint64(cap(f.values)) + 1) / 2
		}
		if n > math.MaxUint32-8 || n > uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e := m.BudgetField.Reserve(n); e != nil {
			return e
		}
		next := make([]bool, len(f.values), int(n))
		copy(next, f.values)
		m.BudgetField.Release(f.charged)
		f.charged = n
		f.values = next
	}
	if need > uint64(len(f.values)) {
		old := len(f.values)
		f.values = f.values[:int(need)]
		clear(f.values[old:])
	}
	f.values[j] = true
	return nil
}
func (f *astProofFound) close(m *ASTManager) {
	m.BudgetField.Release(f.charged)
	f.values = nil
	f.charged = 0
}
func (m *ASTManager) proofComplement(a, b ASTHandle) (bool, error) {
	if a == m.external(m.trueApp) && b == m.external(m.falseApp) {
		return true, nil
	}
	if yes, e := m.proofKind(a, ASTOpNot); e != nil {
		return false, e
	} else if yes {
		x, e := m.proofArg(a, 0)
		if e != nil {
			return false, e
		}
		if x == b {
			return true, nil
		}
	}
	if b == m.external(m.trueApp) && a == m.external(m.falseApp) {
		return true, nil
	}
	if yes, e := m.proofKind(b, ASTOpNot); e != nil {
		return false, e
	} else if yes {
		x, e := m.proofArg(b, 0)
		return x == a, e
	}
	return false, nil
}
func (m *ASTManager) MkProofUnitResolution(ps []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if ASTSourceDebug {
		if len(ps) < 2 {
			return ASTHandle{}, ErrASTDomain
		}
		for _, p := range ps {
			yes, e := m.proofHasFact(p)
			if e != nil {
				return ASTHandle{}, e
			}
			if !yes {
				return ASTHandle{}, ErrASTDomain
			}
		}
	}
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	if len(ps) < 2 {
		return ASTHandle{}, arena.ErrBounds
	}
	f, e := m.proofFact(ps[0])
	if e != nil {
		return ASTHandle{}, e
	}
	g, e := m.proofFact(ps[1])
	if e != nil {
		return ASTHandle{}, e
	}
	fact := ASTHandle{}
	complement := false
	if len(ps) == 2 {
		complement, e = m.proofComplement(f, g)
		if e != nil {
			return ASTHandle{}, e
		}
	}
	if len(ps) == 2 && complement {
		fact = m.external(m.falseApp)
	} else {
		if ASTSourceDebug {
			yes, e := m.proofKind(f, ASTOpOr)
			if e != nil {
				return ASTHandle{}, e
			}
			if !yes {
				return ASTHandle{}, ErrASTDomain
			}
		}
		r, e := m.requireKind(f, ASTApp)
		if e != nil {
			return ASTHandle{}, e
		}
		var lits astProofArguments
		defer lits.close(m)
		var found astProofFound
		defer found.close(m)
		for i := uint32(0); i < m.appAt(r.slot).count; i++ {
			lit, e := m.proofArg(f, i)
			if e != nil {
				return ASTHandle{}, e
			}
			matched := false
			for j := 1; j < len(ps); j++ {
				other, e := m.proofFact(ps[j])
				if e != nil {
					return ASTHandle{}, e
				}
				yes, e := m.proofComplement(lit, other)
				if e != nil {
					return ASTHandle{}, e
				}
				if yes {
					matched = true
					if ASTSourceDebug {
						if e = found.set(m, uint32(j)); e != nil {
							return ASTHandle{}, e
						}
						continue
					}
					break
				}
			}
			if !matched {
				if e = lits.push(m, lit); e != nil {
					return ASTHandle{}, e
				}
			}
		}
		if ASTSourceDebug && m.proofMode != ASTProofDisabled {
			for j := 1; j < len(ps); j++ {
				if j >= len(found.values) || !found.values[j] {
					return ASTHandle{}, ErrASTDomain
				}
			}
		}
		switch len(lits.slice()) {
		case 0:
			fact = m.external(m.falseApp)
		case 1:
			fact = lits.slice()[0]
		default:
			fact, e = m.proofMake(ASTOpOr, lits.slice()...)
			if e != nil {
				return ASTHandle{}, e
			}
		}
		// The C++ else block ends here: found then new_lits are destroyed
		// before args growth or final proof candidate allocation. Defers above
		// remain idempotent early-error cleanup.
		found.close(m)
		lits.close(m)
	}
	if e = args.push(m, fact); e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrUnitResolution, args.slice()...)
}
func (m *ASTManager) MkProofUnitResolutionFact(ps []ASTHandle, fact ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	if e := args.push(m, fact); e != nil {
		return ASTHandle{}, e
	}
	if ASTSourceDebug {
		if len(ps) < 2 {
			return ASTHandle{}, arena.ErrBounds
		}
		f, e := m.proofFact(ps[0])
		if e != nil {
			return ASTHandle{}, e
		}
		g, e := m.proofFact(ps[1])
		if e != nil {
			return ASTHandle{}, e
		}
		yes := false
		if len(ps) == 2 {
			yes, e = m.proofComplement(f, g)
			if e != nil {
				return ASTHandle{}, e
			}
		}
		if len(ps) == 2 && yes {
			if fact != m.external(m.falseApp) {
				return ASTHandle{}, ErrASTDomain
			}
		} else {
			yes, e := m.proofKind(f, ASTOpOr)
			if e != nil {
				return ASTHandle{}, e
			}
			if !yes {
				return ASTHandle{}, ErrASTDomain
			}
			r, e := m.requireKind(f, ASTApp)
			if e != nil {
				return ASTHandle{}, e
			}
			n := m.appAt(r.slot).count
			matches := uint32(0)
			for i := uint32(0); i < n; i++ {
				lit, e := m.proofArg(f, i)
				if e != nil {
					return ASTHandle{}, e
				}
				matched := false
				for j := 1; j < len(ps); j++ {
					other, e := m.proofFact(ps[j])
					if e != nil {
						return ASTHandle{}, e
					}
					yes, e := m.proofComplement(lit, other)
					if e != nil {
						return ASTHandle{}, e
					}
					if yes {
						matches++
						matched = true
						break
					}
				}
				if !matched && fact != lit {
					return ASTHandle{}, ErrASTDomain
				}
			}
			if matches != n && matches != n-1 {
				return ASTHandle{}, ErrASTDomain
			}
			if matches == n {
				if fact != m.external(m.falseApp) {
					return ASTHandle{}, ErrASTDomain
				}
			}
		}
	}
	return m.proofMake(ASTPrUnitResolution, args.slice()...)
}
