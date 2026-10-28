package nlsat

// Source: src/nlsat/nlsat_explain.cpp:704-725,864-949.

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
)

type explanationCellScope struct {
	roots                    *polynomial.AlgebraicVector
	lower, upper             polynomial.Algebraic
	lowerActive, upperActive bool
	active, executing        bool
}

func (n *explanationAssumptions) finishCell() error {
	s := &n.cell
	if s.executing {
		return ErrSearchPending
	}
	am := n.search.evaluator.am
	if s.roots != nil {
		if err := s.roots.Reset(); err != nil {
			return err
		}
	}
	if s.upperActive {
		if err := am.Del(&s.upper); err != nil {
			return err
		}
		s.upperActive = false
	}
	if s.lowerActive {
		if err := am.Del(&s.lower); err != nil {
			return err
		}
		s.lowerActive = false
	}
	s.active = false
	return nil
}

// addRootLiteral currently uses the source fallback representation. Linear and
// quadratic rewrites are equivalent optimizations and remain separate work.
func (n *explanationAssumptions) addRootLiteral(k AtomKind, y, index uint32, p polynomial.Polynomial) error {
	b, err := n.search.store.atoms.mkRoot(k, y, index, p)
	if err != nil {
		return err
	}
	return n.addLiteral(sat.MakeLiteral(b, true))
}

func (n *explanationAssumptions) addCellLiterals(ps *polynomial.PolynomialRefVector, y uint32) (err error) {
	s := &n.cell
	if s.active {
		return ErrSearchPending
	}
	assigned, err := n.search.assignment.Contains(y)
	if err != nil {
		return err
	}
	if err = atomAssert("nlsat_explain.cpp", 881, "m_assignment.is_assigned(y)", assigned); err != nil {
		return err
	}
	if !assigned {
		return arena.ErrConfig
	}
	s.active, s.executing = true, true
	defer func() {
		s.executing = false
		finish := n.finishCell()
		err = errors.Join(err, finish)
	}()
	am := n.search.evaluator.am
	if s.roots == nil {
		s.roots, err = am.NewVector()
		if err != nil {
			return err
		}
	}
	yValue, err := n.search.assignment.Value(y)
	if err != nil {
		return err
	}
	var lowerPolynomial, upperPolynomial polynomial.Polynomial
	var lowerIndex, upperIndex uint32
	count, err := ps.Size()
	if err != nil {
		return err
	}
	core := n.search.store.atoms.ineq.core
	for k := uint32(0); k < count; k++ {
		p, e := ps.Get(k)
		if e != nil {
			return e
		}
		max, e := core.MaxVariable(p)
		if e != nil {
			return e
		}
		if max != y {
			continue
		}
		if e = s.roots.Reset(); e != nil {
			return e
		}
		if e = am.IsolateMultivariateRoots(n.search.store.atoms.ineq.pm, p, UndefVarAssignment{Assignment: n.search.assignment, Variable: y}, s.roots); e != nil {
			return e
		}
		for i := uint32(0); i < s.roots.Len(); i++ {
			root, e := s.roots.Borrow(i)
			if e != nil {
				return e
			}
			comparison, e := am.Compare(yValue, root)
			if e != nil {
				return e
			}
			if comparison == 0 {
				if e = n.addRootLiteral(AtomRootEQ, y, i+1, p); e != nil {
					return e
				}
				return nil
			}
			if comparison < 0 {
				better := !s.upperActive
				if !better {
					better, e = am.Less(root, &s.upper)
					if e != nil {
						return e
					}
				}
				if better {
					s.upperActive = true
					if e = am.Copy(&s.upper, root); e != nil {
						return e
					}
					upperPolynomial, upperIndex = p, i+1
				}
			} else {
				better := !s.lowerActive
				if !better {
					better, e = am.Less(&s.lower, root)
					if e != nil {
						return e
					}
				}
				if better {
					s.lowerActive = true
					if e = am.Copy(&s.lower, root); e != nil {
						return e
					}
					lowerPolynomial, lowerIndex = p, i+1
				}
			}
		}
	}
	if s.lowerActive {
		kind := AtomRootGT
		if n.fullDimensional {
			kind = AtomRootGE
		}
		if err = n.addRootLiteral(kind, y, lowerIndex, lowerPolynomial); err != nil {
			return err
		}
	}
	if s.upperActive {
		kind := AtomRootLT
		if n.fullDimensional {
			kind = AtomRootLE
		}
		if err = n.addRootLiteral(kind, y, upperIndex, upperPolynomial); err != nil {
			return err
		}
	}
	return nil
}

func (n *explanationAssumptions) collectCellLiterals(ps *polynomial.PolynomialRefVector, y uint32, fullDimensional bool) (err error) {
	if ps == nil {
		return arena.ErrConfig
	}
	if err = n.start(); err != nil {
		return err
	}
	n.fullDimensional = fullDimensional
	defer func() { err = n.finish(err) }()
	return n.addCellLiterals(ps, y)
}
