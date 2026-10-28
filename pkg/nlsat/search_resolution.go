package nlsat

// Source: src/nlsat/nlsat_solver.cpp:1426-1647.

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// searchResolution is the concrete conflict-resolution scratch state. It owns
// lemma literal references and borrows one live search state.
type searchResolution struct {
	search   *searchState
	marks    []bool
	numMarks uint32
	lemma    scopedLiteralVector
	lazy     scopedLiteralVector
}

func newSearchResolution(r *searchState) (*searchResolution, error) {
	if err := r.open(); err != nil {
		return nil, err
	}
	q := &searchResolution{search: r}
	q.lemma.factory = r.store.atoms
	q.lazy.factory = r.store.atoms
	return q, nil
}

func (q *searchResolution) close() error {
	if q == nil {
		return nil
	}
	if err := q.lemma.close(); err != nil {
		return err
	}
	if err := q.lazy.close(); err != nil {
		return err
	}
	q.search = nil
	q.marks = nil
	q.numMarks = 0
	return nil
}

func (q *searchResolution) buildLazyClause(j *LazyJustification, explanation *explanationAssumptions) (err error) {
	if j == nil || explanation == nil || explanation.search != q.search {
		return ErrSearchOwner
	}
	literals, err := j.Literals()
	if err != nil {
		return err
	}
	if err = q.lazy.reset(); err != nil {
		return err
	}
	if err = explanation.reset(); err != nil {
		return err
	}
	if err = explanation.collectExplanation(literals, false); err != nil {
		return err
	}
	for _, literal := range explanation.output.result {
		if err = q.lazy.push(literal); err != nil {
			return err
		}
	}
	if err = explanation.reset(); err != nil {
		return err
	}
	for _, literal := range literals {
		if err = q.lazy.push(literal.Negated()); err != nil {
			return err
		}
	}
	return nil
}

func (q *searchResolution) marked(b sat.BoolVar) bool {
	return uint64(b) < uint64(len(q.marks)) && q.marks[b]
}

func (q *searchResolution) mark(b sat.BoolVar) {
	if uint64(b) >= uint64(len(q.marks)) {
		q.marks = append(q.marks, make([]bool, int(uint64(b)+1)-len(q.marks))...)
	}
	q.marks[b] = true
}

func (q *searchResolution) resetMark(b sat.BoolVar) error {
	if uint64(b) >= uint64(len(q.marks)) {
		return arena.ErrBounds
	}
	q.marks[b] = false
	return nil
}

func (q *searchResolution) resetMarks() error {
	for _, literal := range q.lemma.result {
		if err := q.resetMark(literal.Variable()); err != nil {
			return err
		}
	}
	return nil
}

func (q *searchResolution) literalMaxVariable(l sat.Literal) (uint32, error) {
	a, err := q.search.store.literalAtom(l)
	if err != nil || a == nil {
		return NullVariable, err
	}
	return arithmeticAtomHeader(a).maxVar, nil
}

func (q *searchResolution) processAntecedent(antecedent sat.Literal) error {
	r := q.search
	b := antecedent.Variable()
	assigned, err := r.assignedValue(antecedent)
	if err != nil {
		return err
	}
	if assigned == lbool.Undef {
		value, err := r.valueCore(antecedent)
		if err != nil {
			return err
		}
		if err = atomAssert("nlsat_solver.cpp", 1458, "value(antecedent) == l_false", value == lbool.False); err != nil {
			return err
		}
		if q.marked(b) {
			return nil
		}
		x, err := q.literalMaxVariable(antecedent)
		if err != nil {
			return err
		}
		valid := x != NullVariable && r.xk != NullVariable && x < r.xk
		if err = atomAssert("nlsat_solver.cpp", 1460, "is_arith_atom(b) && max_var(b) < m_xk", valid); err != nil {
			return err
		}
		if !valid {
			return arena.ErrConfig
		}
		q.mark(b)
		return q.lemma.push(antecedent)
	}
	if uint64(b) >= uint64(len(r.store.atoms.bools.levels)) {
		return arena.ErrBounds
	}
	if q.marked(b) {
		return nil
	}
	q.mark(b)
	x, err := q.literalMaxVariable(antecedent)
	if err != nil {
		return err
	}
	if r.store.atoms.bools.levels[b] == r.level && x == r.xk {
		q.numMarks++
		return nil
	}
	return q.lemma.push(antecedent)
}

func (q *searchResolution) resolveLiterals(skip sat.BoolVar, literals []sat.Literal) error {
	for _, literal := range literals {
		if literal.Variable() != skip {
			if err := q.processAntecedent(literal); err != nil {
				return err
			}
		}
	}
	return nil
}

func (q *searchResolution) onlyPreviousStages(literals []sat.Literal) (bool, error) {
	for _, literal := range literals {
		x, err := q.literalMaxVariable(literal)
		if err != nil {
			return false, err
		}
		if x == q.search.xk {
			return false, nil
		}
	}
	return true, nil
}

func (q *searchResolution) maxScopeLevel(literals []sat.Literal) (uint32, error) {
	var max uint32
	for _, literal := range literals {
		value, err := q.search.valueCore(literal)
		if err != nil {
			return 0, err
		}
		if err = atomAssert("nlsat_solver.cpp", 1565, "value(ls[i]) == l_false", value == lbool.False); err != nil {
			return 0, err
		}
		assigned, err := q.search.assignedValue(literal)
		if err != nil {
			return 0, err
		}
		if assigned == lbool.False {
			b := literal.Variable()
			if uint64(b) >= uint64(len(q.search.store.atoms.bools.levels)) {
				return 0, arena.ErrBounds
			}
			if level := q.search.store.atoms.bools.levels[b]; level > max {
				max = level
			}
			continue
		}
		x, err := q.literalMaxVariable(literal)
		if err != nil {
			return 0, err
		}
		valid := assigned == lbool.Undef && x != NullVariable && q.search.xk != NullVariable && x < q.search.xk
		if !valid {
			return 0, arena.ErrConfig
		}
	}
	return max, nil
}

func (q *searchResolution) removeLiteralsFromLevel(lemma *scopedLiteralVector, level uint32) error {
	if lemma == nil || lemma.factory != q.search.store.atoms {
		return ErrSearchOwner
	}
	size := len(lemma.result)
	j := uint32(0)
	for i := 0; i < size; i++ {
		literal := lemma.result[i]
		b := literal.Variable()
		marked := q.marked(b)
		if err := atomAssert("nlsat_solver.cpp", 1594, "is_marked(b)", marked); err != nil {
			return err
		}
		if !marked {
			return arena.ErrConfig
		}
		value, err := q.search.valueCore(literal)
		if err != nil {
			return err
		}
		if err = atomAssert("nlsat_solver.cpp", 1595, "value(lemma[i]) == l_false", value == lbool.False); err != nil {
			return err
		}
		if value != lbool.False {
			return arena.ErrConfig
		}
		assigned, err := q.search.assignedValue(literal)
		if err != nil {
			return err
		}
		x, err := q.literalMaxVariable(literal)
		if err != nil {
			return err
		}
		if assigned == lbool.False && q.search.store.atoms.bools.levels[b] == level && x == q.search.xk {
			q.numMarks++
			continue
		}
		if err = lemma.set(j, literal); err != nil {
			return err
		}
		j++
	}
	return lemma.shrink(j)
}

func (q *searchResolution) isBooleanLemma(literals []sat.Literal) (bool, error) {
	for _, literal := range literals {
		a, err := q.search.store.literalAtom(literal)
		if err != nil {
			return false, err
		}
		if a != nil {
			return false, nil
		}
	}
	return true, nil
}

func (q *searchResolution) findNewArithmeticLevel(literals []sat.Literal) (uint32, error) {
	boolean, err := q.isBooleanLemma(literals)
	if err != nil {
		return 0, err
	}
	if err = atomAssert("nlsat_solver.cpp", 1624, "!is_bool_lemma(sz, lemma)", !boolean); err != nil {
		return 0, err
	}
	if boolean || len(literals) == 0 {
		return 0, arena.ErrConfig
	}
	var level uint32
	found := false
	for _, literal := range literals[:len(literals)-1] {
		x, err := q.literalMaxVariable(literal)
		if err != nil {
			return 0, err
		}
		if x == q.search.xk {
			b := literal.Variable()
			if uint64(b) >= uint64(len(q.search.store.atoms.bools.levels)) {
				return 0, arena.ErrBounds
			}
			candidate := q.search.store.atoms.bools.levels[b]
			if !found || candidate > level {
				level = candidate
				found = true
			}
		}
	}
	valid := !found || level < q.search.level
	if err = atomAssert("nlsat_solver.cpp", 1641, "!found_lvl || new_lvl < scope_lvl()", valid); err != nil {
		return 0, err
	}
	if !valid {
		return 0, arena.ErrConfig
	}
	if !found {
		if q.search.level == 0 {
			return 0, arena.ErrBounds
		}
		level = q.search.level - 1
	}
	return level, nil
}
