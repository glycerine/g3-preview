package nlsat

// Source: src/nlsat/nlsat_solver.cpp:167-231; nlsat_solver.h:38-55,64-112,
// 172-205,214.

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

type SolverConfig struct {
	Algebraic    *polynomial.AlgebraicManager
	Polynomial   *polynomial.MultivariatePolynomialManager
	Limit        *control.Reslimit
	MaxMemory    uint64
	Lazy         uint32
	SimplifyCore bool
	Randomize    bool
	Factor       bool
}

// Solver composes the retained concrete NLSAT components. The numerical and
// polynomial managers are borrowed and must outlive Close.
type Solver struct {
	config      SolverConfig
	allocator   *umemory.SmallAllocator
	cache       *polynomial.PolynomialCache
	atoms       *atomFactory
	assumptions *assumptionManager
	clauses     *clauseStore
	assignment  *Assignment
	evaluator   *Evaluator
	search      *searchState
	explanation *explanationAssumptions
	projection  *projectionWorklist
	resolution  *searchResolution
	closed      bool
}

func NewSolver(config SolverConfig) (s *Solver, err error) {
	s = &Solver{config: config}
	if config.Algebraic == nil || config.Polynomial == nil || config.Limit == nil || config.Limit.Closed() {
		return s, arena.ErrConfig
	}
	if config.MaxMemory == 0 {
		s.config.MaxMemory = ^uint64(0)
	}
	q, err := config.Algebraic.RationalManager()
	if err != nil {
		return s, err
	}
	s.allocator, err = umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: diagnostic.DebugBuild}})
	if err != nil {
		return s, err
	}
	s.cache, err = polynomial.NewPolynomialCache(config.Polynomial)
	if err != nil {
		return s, err
	}
	s.atoms, err = newAtomFactory(config.Polynomial, s.cache, s.allocator)
	if err != nil {
		return s, err
	}
	s.assumptions, err = newAssumptionManager(s.allocator)
	if err != nil {
		return s, err
	}
	s.clauses, err = newClauseStore(s.atoms, s.assumptions)
	if err != nil {
		return s, err
	}
	if err = s.clauses.mkTrueBooleanVariable(); err != nil {
		return s, err
	}
	s.assignment, err = NewAssignment(config.Algebraic)
	if err != nil {
		return s, err
	}
	s.evaluator, err = NewEvaluator(s.assignment, config.Polynomial, s.allocator)
	if err != nil {
		return s, err
	}
	s.search, err = newSearchState(s.clauses, s.evaluator, searchConfig{Limit: config.Limit, MaxMemory: s.config.MaxMemory, Lazy: config.Lazy, SimplifyCores: config.SimplifyCore, Randomize: config.Randomize})
	if err != nil {
		return s, err
	}
	s.explanation, err = newExplanationAssumptions(s.search)
	if err != nil {
		return s, err
	}
	s.projection, err = newProjectionWorklist(s.search)
	if err != nil {
		return s, err
	}
	if err = s.explanation.bindProjection(s.projection, config.Factor); err != nil {
		return s, err
	}
	s.resolution, err = newSearchResolution(s.search)
	return s, err
}

func (s *Solver) open() error {
	if s == nil || s.closed || s.search == nil || s.resolution == nil {
		return ErrSearchOwner
	}
	return s.search.open()
}

func (s *Solver) PolynomialManager() (*polynomial.MultivariatePolynomialManager, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	return s.config.Polynomial, nil
}

func (s *Solver) AlgebraicManager() (*polynomial.AlgebraicManager, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	return s.config.Algebraic, nil
}

func (s *Solver) MkVar(integer bool) (uint32, error) {
	if err := s.open(); err != nil {
		return NullVariable, err
	}
	return s.clauses.mkVar(integer)
}

func (s *Solver) MkBoolVar() (sat.BoolVar, error) {
	if err := s.open(); err != nil {
		return sat.NullBoolVar, err
	}
	return s.atoms.mkBoolVar()
}

func (s *Solver) MkInequality(kind AtomKind, factors []polynomial.Polynomial, even []bool, negative bool) (sat.Literal, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	literal, err := s.atoms.mkInequalityLiteral(kind, factors, even)
	if negative {
		literal = literal.Negated()
	}
	return literal, err
}

func (s *Solver) MkRoot(kind AtomKind, variable, index uint32, p polynomial.Polynomial, negative bool) (sat.Literal, error) {
	if err := s.open(); err != nil {
		return 0, err
	}
	b, err := s.atoms.mkRoot(kind, variable, index, p)
	if err != nil {
		return 0, err
	}
	return sat.MakeLiteral(b, negative), nil
}

func (s *Solver) AddClause(literals []sat.Literal, assumption Assumption) error {
	if err := s.open(); err != nil {
		return err
	}
	return s.clauses.mkInputClause(literals, assumption)
}

func (s *Solver) Check() (lbool.LBool, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, err
	}
	return s.search.search(s.resolution, s.explanation)
}

func (s *Solver) Value(variable uint32, out *polynomial.Algebraic) error {
	if err := s.open(); err != nil {
		return err
	}
	if out == nil {
		return polynomial.ErrAlgebraicOwner
	}
	assigned, err := s.assignment.Contains(variable)
	if err != nil {
		return err
	}
	if !assigned {
		return arena.ErrBounds
	}
	value, err := s.assignment.Value(variable)
	if err != nil {
		return err
	}
	return s.config.Algebraic.Copy(out, value)
}

func (s *Solver) BooleanValue(variable sat.BoolVar) (lbool.LBool, error) {
	if err := s.open(); err != nil {
		return lbool.Undef, err
	}
	if uint64(variable) >= uint64(len(s.atoms.bools.values)) {
		return lbool.Undef, arena.ErrBounds
	}
	return s.atoms.bools.values[variable], nil
}

func (s *Solver) Core(out *[]Assumption) error {
	if err := s.open(); err != nil {
		return err
	}
	if out == nil {
		return arena.ErrConfig
	}
	*out = (*out)[:0]
	return s.assumptions.linearize(s.resolution.lemmaAssumptions, out)
}

func (s *Solver) Close() (err error) {
	if s == nil || s.closed {
		return nil
	}
	if s.resolution != nil {
		err = errors.Join(err, s.resolution.close())
		if err == nil {
			s.resolution = nil
		}
	}
	if err == nil && s.search != nil {
		err = s.search.close()
		if err == nil {
			s.search, s.explanation, s.projection = nil, nil, nil
		}
	}
	if err == nil && s.evaluator != nil {
		err = s.evaluator.Close()
		if err == nil {
			s.evaluator = nil
		}
	}
	if err == nil && s.assignment != nil {
		err = s.assignment.Close()
		if err == nil {
			s.assignment = nil
		}
	}
	if err == nil && s.clauses != nil {
		err = s.clauses.close()
		if err == nil {
			s.clauses = nil
		}
	}
	if err == nil && s.assumptions != nil {
		err = s.assumptions.close()
		if err == nil {
			s.assumptions = nil
		}
	}
	if err == nil && s.atoms != nil {
		if cleanup := s.atoms.cleanup(); cleanup != nil {
			err = cleanup
		}
		for _, atom := range s.atoms.bools.atoms {
			if err != nil {
				break
			}
			if atom != nil {
				err = s.atoms.deleteAtom(atom)
			}
		}
		if err == nil {
			err = s.atoms.close()
		}
		if err == nil {
			s.atoms = nil
		}
	}
	if err == nil && s.cache != nil {
		err = s.cache.Close()
		if err == nil {
			s.cache = nil
		}
	}
	if err == nil && s.allocator != nil {
		err = s.allocator.Close()
		if err == nil {
			s.allocator = nil
		}
	}
	if err == nil {
		s.closed = true
	}
	return err
}
