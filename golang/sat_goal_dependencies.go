// Source: src/sat/tactic/goal2sat.cpp:425-485;
// src/sat/tactic/atom2bool_var.cpp:88-111; src/sat/tactic/sat_tactic.cpp:133-138.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"
)

func (tr *satGoalTranslator) retainDependency(h ASTHandle) error {
	if len(tr.dependencyTrail) == cap(tr.dependencyTrail) {
		stride := uint64(reflect.TypeOf(h).Size())
		n, e := satGoalCapacity(cap(tr.dependencyTrail), stride)
		if e != nil {
			return e
		}
		if e = tr.solver.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ASTHandle, len(tr.dependencyTrail), n)
		copy(next, tr.dependencyTrail)
		tr.solver.budget.Release(uint64(cap(tr.dependencyTrail)) * stride)
		tr.dependencyTrail = next
	}
	if e := tr.manager.IncRef(h); e != nil {
		return e
	}
	tr.dependencyTrail = append(tr.dependencyTrail, h)
	return nil
}
func (tr *satGoalTranslator) processOwning(h ASTHandle) (err error) {
	if err = tr.manager.IncRef(h); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, tr.manager.DecRef(h)) }()
	return tr.process(h)
}
func (tr *satGoalTranslator) insertGoalDependency(original, dep ASTHandle, sign bool) (err error) {
	m := tr.manager
	simple, e := tr.isUninterpretedConstant(dep)
	if e != nil {
		return e
	}
	fresh := dep
	if !simple {
		fresh, e = m.MkFreshConst("dep", tr.boolSort)
		if e != nil {
			return e
		}
	}
	if e = m.IncRef(fresh); e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DecRef(fresh)) }()
	if !simple {
		if e = tr.retainDependency(fresh); e != nil {
			return e
		}
		hash, e := m.GetHash(fresh)
		if e != nil {
			return e
		}
		if e = tr.interfaces.prepare(tr.solver.budget); e != nil {
			return e
		}
		tr.interfaces.insertPrepared(fresh, hash, 0)
		iff, e := m.MkIff(fresh, dep)
		if e != nil {
			return e
		}
		if e = tr.processOwning(iff); e != nil {
			return e
		}
	}
	if e = tr.convertAtom(fresh, false, false); e != nil {
		return e
	}
	l := tr.results[len(tr.results)-1]
	if sign {
		l = l.negated()
	}
	hash, e := m.GetHash(original)
	if e != nil {
		return e
	}
	if e = tr.dependencies.prepare(tr.solver.budget); e != nil {
		return e
	}
	tr.dependencies.insertPrepared(original, hash, l.index())
	tr.results = tr.results[:len(tr.results)-1]
	return nil
}

// Dependencies are borrowed map keys, as in source dep2asm. The caller keeps
// its labels alive while consuming the assumption/core mapping.
func (tr *satGoalTranslator) AppendDependencyAssumptions(out []satLiteral) ([]satLiteral, error) {
	if e := tr.open(); e != nil {
		return out, e
	}
	for _, entry := range tr.dependencies.cells {
		if entry.used {
			out = append(out, satLiteral(entry.value))
		}
	}
	return out, nil
}
func (tr *satGoalTranslator) CompileGoal(g *tacticGoal) (err error) {
	if err = tr.open(); err != nil {
		return err
	}
	if err = g.open(); err != nil {
		return err
	}
	if g.manager != tr.manager {
		return ErrASTHandle
	}
	defer func() {
		if err != nil {
			tr.failed = true
		}
	}()
	m := tr.manager
	deps, e := m.NewExprVector()
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelExprVector(&deps)) }()
	var forms, allDeps modelDeclarationVector
	defer forms.close(m)
	defer allDeps.close(m)
	size, e := g.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		d, e := g.Dependency(i)
		if e != nil {
			return e
		}
		if !d.IsNull() {
			if e = m.ExprVectorReset(deps); e != nil {
				return e
			}
			if e = m.DependencyLinearize(d, deps); e != nil {
				return e
			}
			n, e := m.ExprVectorSize(deps)
			if e != nil {
				return e
			}
			for j := uint32(0); j < n; j++ {
				h, e := m.ExprVectorGet(deps, j)
				if e != nil {
					return e
				}
				if e = allDeps.push(m, h); e != nil {
					return e
				}
			}
		}
		f, e := g.Form(i)
		if e != nil {
			return e
		}
		if e = forms.push(m, f); e != nil {
			return e
		}
	}
	tr.interfaces.clear()
	if e = tr.collectInterfaceWithDependencies(forms.values, allDeps.values); e != nil {
		return e
	}
	var fmls satGoalLiteralExpressions
	defer func() { err = errors.Join(err, fmls.close(m)) }()
	for i := uint32(0); i < size; i++ {
		f, e := g.Form(i)
		if e != nil {
			return e
		}
		d, e := g.Dependency(i)
		if e != nil {
			return e
		}
		skip := false
		if !d.IsNull() {
			if e = m.ExprVectorReset(deps); e != nil {
				return e
			}
			if e = m.DependencyLinearize(d, deps); e != nil {
				return e
			}
			if e = fmls.resize(m, 0); e != nil {
				return e
			}
			if e = fmls.resize(m, 1); e != nil {
				return e
			}
			if e = fmls.set(m, 0, f); e != nil {
				return e
			}
			n, e := m.ExprVectorSize(deps)
			if e != nil {
				return e
			}
			for j := uint32(0); j < n; j++ {
				original, e := m.ExprVectorGet(deps, j)
				if e != nil {
					return e
				}
				sign, inner, e := m.IsNot(original)
				if e != nil {
					return e
				}
				if !sign {
					inner = original
				}
				if e = tr.insertGoalDependency(original, inner, sign); e != nil {
					return e
				}
				if original == f {
					skip = true
					break
				}
				guard := inner
				if !sign {
					guard, e = m.MkNot(original)
					if e != nil {
						return e
					}
				}
				count := uint32(len(fmls.values))
				if e = fmls.resize(m, count+1); e != nil {
					return e
				}
				if e = fmls.set(m, count, guard); e != nil {
					return e
				}
			}
			if !skip {
				f, e = m.MkOrArgs(fmls.values)
				if e != nil {
					return e
				}
			}
		}
		if !skip {
			if e = tr.processOwning(f); e != nil {
				return e
			}
		}
	}
	return nil
}
