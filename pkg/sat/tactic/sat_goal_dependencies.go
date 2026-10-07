// Source: src/sat/tactic/goal2sat.cpp:425-485;
// src/sat/tactic/atom2bool_var.cpp:88-111; src/sat/tactic/sat_tactic.cpp:133-138.
package tactic

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util"
)

func (tr *GoalTranslator) retainDependency(h ast.ASTHandle) error {
	if len(tr.dependencyTrail) == cap(tr.dependencyTrail) {
		stride := uint64(reflect.TypeOf(h).Size())
		n, e := util.SourceVectorCapacity(cap(tr.dependencyTrail), stride)
		if e != nil {
			return e
		}
		if e = tr.solver.Budget().Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ast.ASTHandle, len(tr.dependencyTrail), n)
		copy(next, tr.dependencyTrail)
		tr.solver.Budget().Release(uint64(cap(tr.dependencyTrail)) * stride)
		tr.dependencyTrail = next
	}
	if e := tr.manager.IncRef(h); e != nil {
		return e
	}
	tr.dependencyTrail = append(tr.dependencyTrail, h)
	return nil
}
func (tr *GoalTranslator) processOwning(h ast.ASTHandle) (err error) {
	if err = tr.manager.IncRef(h); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, tr.manager.DecRef(h)) }()
	return tr.process(h)
}
func (tr *GoalTranslator) insertGoalDependency(original, dep ast.ASTHandle, sign bool) (err error) {
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
		if e = tr.interfaces.prepare(tr.solver.Budget()); e != nil {
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
		l = l.Negated()
	}
	hash, e := m.GetHash(original)
	if e != nil {
		return e
	}
	if e = tr.dependencies.prepare(tr.solver.Budget()); e != nil {
		return e
	}
	tr.dependencies.insertPrepared(original, hash, l.Index())
	tr.results = tr.results[:len(tr.results)-1]
	return nil
}

// Dependencies are borrowed map keys, as in source dep2asm. The caller keeps
// its labels alive while consuming the assumption/core mapping.
func (tr *GoalTranslator) AppendDependencyAssumptions(out []sat.Literal) ([]sat.Literal, error) {
	if e := tr.open(); e != nil {
		return out, e
	}
	for _, entry := range tr.dependencies.Cells {
		if entry.Used {
			out = append(out, sat.Literal(entry.value))
		}
	}
	return out, nil
}
func (tr *GoalTranslator) CompileGoal(g *tactic.Goal) (err error) {
	if err = tr.open(); err != nil {
		return err
	}
	if err = g.CheckOwner(); err != nil {
		return err
	}
	if g.Manager() != tr.manager {
		return ast.ErrASTHandle
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
	var forms, allDeps util.PointerVector[ast.ASTHandle]
	defer forms.Close(m.Budget())
	defer allDeps.Close(m.Budget())
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
				if e = allDeps.Push(m.Budget(), h); e != nil {
					return e
				}
			}
		}
		f, e := g.Form(i)
		if e != nil {
			return e
		}
		if e = forms.Push(m.Budget(), f); e != nil {
			return e
		}
	}
	tr.interfaces.clear()
	if e = tr.collectInterfaceWithDependencies(forms.Slice(), allDeps.Slice()); e != nil {
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
