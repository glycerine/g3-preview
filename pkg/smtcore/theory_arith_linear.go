// Source: retained linear rational fragment of src/smt/theory_arith.h,
// src/smt/theory_arith_core.h, src/smt/theory_arith_core_def.h, and the
// retained integer-bound fragment of src/smt/theory_arith_int.h.
package smtcore

import (
	"math/big"
	"sort"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type smtLinearForm struct {
	constant big.Rat
	terms    map[uint32]*big.Rat
}

func newSMTLinearForm() smtLinearForm { return smtLinearForm{terms: make(map[uint32]*big.Rat)} }

func (f *smtLinearForm) addTerm(variable uint32, coefficient *big.Rat) {
	if coefficient.Sign() == 0 {
		return
	}
	if old := f.terms[variable]; old != nil {
		old.Add(old, coefficient)
		if old.Sign() == 0 {
			delete(f.terms, variable)
		}
		return
	}
	f.terms[variable] = new(big.Rat).Set(coefficient)
}

func (f *smtLinearForm) add(other *smtLinearForm) {
	f.constant.Add(&f.constant, &other.constant)
	for variable, coefficient := range other.terms {
		f.addTerm(variable, coefficient)
	}
}

func (f *smtLinearForm) scale(coefficient *big.Rat) {
	f.constant.Mul(&f.constant, coefficient)
	for variable, value := range f.terms {
		value.Mul(value, coefficient)
		if value.Sign() == 0 {
			delete(f.terms, variable)
		}
	}
}

type smtLinearArithmeticAtom struct {
	variable  int32
	form      smtLinearForm
	op        int32
	supported bool
	truth     bool
}

type smtLinearArithmeticScope struct {
	atoms, assertions int
	head              int
	unsupported       bool
}

type smtLinearIntegerBranch struct {
	variable uint32
	lower    bool
	value    big.Rat
}

// SMTLinearArithmeticTheory is the rational linear specialization of the
// source arithmetic theory. Each comparison gets a tableau row for its linear
// polynomial; Boolean assignments select its lower or upper infinitesimal
// bound. The tableau is replayed after scope pops because math/simplex owns no
// logical-context trail.
type SMTLinearArithmeticTheory struct {
	*SMTTheoryBase
	solver           *simplex.Solver
	simplexVariables []uint32
	nextSimplex      uint32
	atoms            []smtLinearArithmeticAtom
	atomByBool       map[int32]int
	assertions       []smtIntegerDiffAssertion
	assertionHead    int
	scopes           []smtLinearArithmeticScope
	unsupported      bool
	integer          bool
	delta            big.Rat
	equalityAdapter  *smtArithmeticEqualityAdapter
}

func NewSMTLinearArithmeticTheory(family ast.FamilyID) (*SMTLinearArithmeticTheory, error) {
	return newSMTLinearArithmeticTheory(family, false)
}

func NewSMTLinearIntegerArithmeticTheory(family ast.FamilyID) (*SMTLinearArithmeticTheory, error) {
	return newSMTLinearArithmeticTheory(family, true)
}

func newSMTLinearArithmeticTheory(family ast.FamilyID, integer bool) (*SMTLinearArithmeticTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	t := &SMTLinearArithmeticTheory{
		SMTTheoryBase: base,
		solver:        simplex.NewSolver(nil),
		atomByBool:    make(map[int32]int),
		integer:       integer,
	}
	t.equalityAdapter, err = newSMTArithmeticEqualityAdapter(base)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *SMTLinearArithmeticTheory) ConfigureArithmetic(config SMTArithState) error {
	if t == nil || t.equalityAdapter == nil {
		return ast.ErrASTHandle
	}
	t.equalityAdapter.lazy = config.LazyAdapter
	return nil
}

func (t *SMTLinearArithmeticTheory) createVariable(node *SMTEnode) (SMTTheoryVar, error) {
	if t.IsAttachedToVariable(node) {
		return node.TheoryVar(t.FamilyID()), nil
	}
	v, err := t.SMTTheoryBase.CreateVariable(node)
	if err != nil {
		return SMTNullTheoryVar, err
	}
	for len(t.simplexVariables) <= int(v) {
		t.simplexVariables = append(t.simplexVariables, t.nextSimplex)
		t.nextSimplex++
	}
	return v, nil
}

func (t *SMTLinearArithmeticTheory) ApplySortConstraint(node *SMTEnode, sort ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	family, err := state.manager.GetFamilyID(sort)
	if err != nil || family != t.FamilyID() {
		return err
	}
	kind, err := state.manager.GetDeclKind(sort)
	if err != nil {
		return err
	}
	expected := int32(0)
	if t.integer {
		expected = 1
	}
	if kind != expected {
		t.unsupported = true
		return nil
	}
	_, err = t.createVariable(node)
	return err
}

func (t *SMTLinearArithmeticTheory) numeral(expression ast.ASTHandle) (*big.Rat, bool, error) {
	state, err := t.State()
	if err != nil {
		return nil, false, err
	}
	text, _, matched, err := state.manager.ArithmeticNumeralText(expression)
	if err != nil || !matched {
		return nil, false, err
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, false, ast.ErrASTDomain
	}
	return value, true, nil
}

func (t *SMTLinearArithmeticTheory) operation(expression ast.ASTHandle) (int32, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	kind, err := state.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return 0, false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return 0, false, err
	}
	op, err := state.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func (t *SMTLinearArithmeticTheory) ensureVariable(expression ast.ASTHandle) (uint32, error) {
	state, err := t.State()
	if err != nil {
		return 0, err
	}
	if node := state.context.enodes[expression]; node != nil && t.IsAttachedToVariable(node) {
		return t.simplexVariables[node.TheoryVar(t.FamilyID())], nil
	}
	if err = state.Internalize(expression, false); err != nil {
		return 0, err
	}
	node := state.context.enodes[expression]
	if node == nil {
		node, err = state.MkEnode(expression, false, false, true)
		if err != nil {
			return 0, err
		}
	}
	v, err := t.createVariable(node)
	if err != nil {
		return 0, err
	}
	return t.simplexVariables[v], nil
}

func (t *SMTLinearArithmeticTheory) linearize(expression ast.ASTHandle) (smtLinearForm, bool, error) {
	if value, numeral, err := t.numeral(expression); err != nil {
		return smtLinearForm{}, false, err
	} else if numeral {
		if t.integer && !value.IsInt() {
			return smtLinearForm{}, false, nil
		}
		result := newSMTLinearForm()
		result.constant.Set(value)
		return result, true, nil
	}
	op, arithmetic, err := t.operation(expression)
	if err != nil {
		return smtLinearForm{}, false, err
	}
	state, _ := t.State()
	if arithmetic {
		count, countErr := state.manager.GetNumArgs(expression)
		if countErr != nil {
			return smtLinearForm{}, false, countErr
		}
		switch op {
		case smtArithOpAdd, smtArithOpSub:
			result := newSMTLinearForm()
			for index := uint32(0); index < count; index++ {
				argument, argErr := state.manager.GetArg(expression, index)
				if argErr != nil {
					return smtLinearForm{}, false, argErr
				}
				part, linear, partErr := t.linearize(argument)
				if partErr != nil || !linear {
					return smtLinearForm{}, linear, partErr
				}
				if op == smtArithOpSub && index != 0 {
					part.scale(big.NewRat(-1, 1))
				}
				result.add(&part)
			}
			return result, true, nil
		case smtArithOpUMinus, smtArithOpToReal:
			if t.integer && op == smtArithOpToReal {
				return smtLinearForm{}, false, nil
			}
			if count != 1 {
				return smtLinearForm{}, false, nil
			}
			argument, argErr := state.manager.GetArg(expression, 0)
			if argErr != nil {
				return smtLinearForm{}, false, argErr
			}
			result, linear, partErr := t.linearize(argument)
			if partErr == nil && linear && op == smtArithOpUMinus {
				result.scale(big.NewRat(-1, 1))
			}
			return result, linear, partErr
		case smtArithOpMul:
			result := newSMTLinearForm()
			result.constant.SetInt64(1)
			hasTerms := false
			for index := uint32(0); index < count; index++ {
				argument, argErr := state.manager.GetArg(expression, index)
				if argErr != nil {
					return smtLinearForm{}, false, argErr
				}
				part, linear, partErr := t.linearize(argument)
				if partErr != nil || !linear {
					return smtLinearForm{}, linear, partErr
				}
				if len(part.terms) != 0 {
					if hasTerms {
						return smtLinearForm{}, false, nil
					}
					hasTerms = true
					part.scale(&result.constant)
					result = part
				} else {
					if t.integer && !part.constant.IsInt() {
						return smtLinearForm{}, false, nil
					}
					result.scale(&part.constant)
				}
			}
			return result, true, nil
		case smtArithOpDiv:
			if t.integer {
				return smtLinearForm{}, false, nil
			}
			if count != 2 {
				return smtLinearForm{}, false, nil
			}
			left, argErr := state.manager.GetArg(expression, 0)
			if argErr != nil {
				return smtLinearForm{}, false, argErr
			}
			right, argErr := state.manager.GetArg(expression, 1)
			if argErr != nil {
				return smtLinearForm{}, false, argErr
			}
			denominator, numeral, numeralErr := t.numeral(right)
			if numeralErr != nil || !numeral || denominator.Sign() == 0 {
				return smtLinearForm{}, false, numeralErr
			}
			result, linear, partErr := t.linearize(left)
			if partErr == nil && linear {
				result.scale(new(big.Rat).Inv(denominator))
			}
			return result, linear, partErr
		}
	}
	variable, err := t.ensureVariable(expression)
	if err != nil {
		return smtLinearForm{}, false, err
	}
	result := newSMTLinearForm()
	result.addTerm(variable, big.NewRat(1, 1))
	return result, true, nil
}

func (t *SMTLinearArithmeticTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	_, linear, err := t.linearize(expression)
	if err == nil && !linear {
		t.unsupported = true
		state, stateErr := t.State()
		if stateErr != nil {
			return false, stateErr
		}
		node := state.context.enodes[expression]
		if node == nil {
			node, stateErr = state.MkEnode(expression, false, false, true)
		}
		if stateErr == nil && !t.IsAttachedToVariable(node) {
			_, stateErr = t.createVariable(node)
		}
		return stateErr == nil, stateErr
	}
	return linear, err
}

func (t *SMTLinearArithmeticTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	op, arithmetic, err := t.operation(expression)
	if err != nil {
		return false, err
	}
	if !arithmetic || op < smtArithOpLE || op > smtArithOpGT {
		t.unsupported = true
		return false, nil
	}
	left, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return false, err
	}
	right, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return false, err
	}
	form, linear, err := t.linearize(left)
	if err != nil {
		return false, err
	}
	if !linear {
		t.unsupported = true
		form = newSMTLinearForm()
	}
	rhs, rhsLinear, err := t.linearize(right)
	if err != nil {
		return false, err
	}
	linear = linear && rhsLinear
	if linear {
		rhs.scale(big.NewRat(-1, 1))
		form.add(&rhs)
	} else {
		t.unsupported = true
		form = newSMTLinearForm()
	}
	variable, err := state.MkBoolVar(expression)
	if err != nil {
		return false, err
	}
	if err = state.SetVariableTheory(variable, t.FamilyID()); err != nil {
		return false, err
	}
	t.atomByBool[variable] = len(t.atoms)
	t.atoms = append(t.atoms, smtLinearArithmeticAtom{variable: variable, form: form, op: op, supported: linear})
	return true, nil
}

func (t *SMTLinearArithmeticTheory) Assign(variable int32, truth bool) error {
	index, found := t.atomByBool[variable]
	if !found {
		return ast.ErrASTDomain
	}
	t.atoms[index].truth = truth
	t.assertions = append(t.assertions, smtIntegerDiffAssertion{atom: index, truth: truth})
	return nil
}

func (t *SMTLinearArithmeticTheory) CanPropagate() bool {
	return t != nil && t.assertionHead < len(t.assertions)
}

func smtLinearAtomBound(atom *smtLinearArithmeticAtom, truth, integer bool) (lower bool, value simplex.Infinitesimal) {
	value.Main.Neg(&atom.form.constant)
	op := atom.op
	if !truth {
		switch op {
		case smtArithOpLE:
			op = smtArithOpGT
		case smtArithOpLT:
			op = smtArithOpGE
		case smtArithOpGE:
			op = smtArithOpLT
		case smtArithOpGT:
			op = smtArithOpLE
		}
	}
	lower = op == smtArithOpGE || op == smtArithOpGT
	if integer && op == smtArithOpGT {
		value.Main.Add(&value.Main, big.NewRat(1, 1))
	} else if integer && op == smtArithOpLT {
		value.Main.Sub(&value.Main, big.NewRat(1, 1))
	} else if op == smtArithOpGT {
		value.Epsilon.SetInt64(1)
	} else if op == smtArithOpLT {
		value.Epsilon.SetInt64(-1)
	}
	return lower, value
}

func (t *SMTLinearArithmeticTheory) rebuild() (bool, error) {
	solver, status, err := t.buildSimplex(nil)
	if err == nil && status == lbool.True {
		t.solver = solver
	}
	return status == lbool.True, err
}

func (t *SMTLinearArithmeticTheory) buildSimplex(branches []smtLinearIntegerBranch) (*simplex.Solver, lbool.LBool, error) {
	solver := simplex.NewSolver(nil)
	for variable := uint32(0); variable < t.nextSimplex; variable++ {
		solver.EnsureVariable(variable)
	}
	for index := range t.atoms {
		atom := &t.atoms[index]
		if !atom.supported {
			continue
		}
		base := t.nextSimplex + uint32(index)
		variables := make([]uint32, 0, len(atom.form.terms)+1)
		coefficients := make([]big.Rat, 0, len(atom.form.terms)+1)
		variables = append(variables, base)
		coefficients = append(coefficients, *big.NewRat(1, 1))
		ordered := make([]uint32, 0, len(atom.form.terms))
		for variable := range atom.form.terms {
			ordered = append(ordered, variable)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
		for _, variable := range ordered {
			coefficient := atom.form.terms[variable]
			variables = append(variables, variable)
			coefficients = append(coefficients, *new(big.Rat).Neg(coefficient))
		}
		if _, err := solver.AddRationalRow(base, variables, coefficients); err != nil {
			return nil, lbool.Undef, err
		}
	}
	for _, assertion := range t.assertions {
		atom := &t.atoms[assertion.atom]
		if !atom.supported {
			continue
		}
		lower, bound := smtLinearAtomBound(atom, assertion.truth, t.integer)
		base := t.nextSimplex + uint32(assertion.atom)
		var err error
		if lower {
			err = solver.SetLower(base, bound)
		} else {
			err = solver.SetUpper(base, bound)
		}
		if err != nil {
			return solver, lbool.False, nil
		}
	}
	for index := range branches {
		branch := &branches[index]
		bound := simplex.RationalInfinitesimal(&branch.value)
		var err error
		if branch.lower {
			err = solver.SetLower(branch.variable, bound)
		} else {
			err = solver.SetUpper(branch.variable, bound)
		}
		if err != nil {
			return solver, lbool.False, nil
		}
	}
	status, err := solver.MakeFeasible()
	if err != nil {
		return nil, lbool.Undef, err
	}
	return solver, status, nil
}

func (t *SMTLinearArithmeticTheory) fractionalVariable(solver *simplex.Solver) (uint32, *big.Rat, bool) {
	for _, variable := range t.simplexVariables {
		value := solver.Value(variable)
		if value.Epsilon.Sign() != 0 || !value.Main.IsInt() {
			return variable, new(big.Rat).Set(&value.Main), true
		}
	}
	return 0, nil, false
}

func (t *SMTLinearArithmeticTheory) integerFeasible(branches []smtLinearIntegerBranch, nodes *uint32) (lbool.LBool, error) {
	const maxNodes = uint32(1024)
	if *nodes >= maxNodes {
		return lbool.Undef, nil
	}
	*nodes++
	solver, status, err := t.buildSimplex(branches)
	if err != nil || status != lbool.True {
		return status, err
	}
	variable, value, fractional := t.fractionalVariable(solver)
	if !fractional {
		t.solver = solver
		return lbool.True, nil
	}
	var floor big.Int
	floor.Quo(value.Num(), value.Denom())
	if value.Sign() < 0 && new(big.Int).Rem(value.Num(), value.Denom()).Sign() != 0 {
		floor.Sub(&floor, big.NewInt(1))
	}
	upper := smtLinearIntegerBranch{variable: variable, value: *new(big.Rat).SetInt(&floor)}
	left := append(append([]smtLinearIntegerBranch(nil), branches...), upper)
	leftStatus, err := t.integerFeasible(left, nodes)
	if err != nil || leftStatus == lbool.True {
		return leftStatus, err
	}
	ceil := new(big.Int).Add(&floor, big.NewInt(1))
	lower := smtLinearIntegerBranch{variable: variable, lower: true, value: *new(big.Rat).SetInt(ceil)}
	right := append(append([]smtLinearIntegerBranch(nil), branches...), lower)
	rightStatus, err := t.integerFeasible(right, nodes)
	if err != nil || rightStatus == lbool.True {
		return rightStatus, err
	}
	if leftStatus == lbool.Undef || rightStatus == lbool.Undef {
		return lbool.Undef, nil
	}
	return lbool.False, nil
}

func (t *SMTLinearArithmeticTheory) Propagate() error {
	state, err := t.State()
	if err != nil {
		return err
	}
	t.assertionHead = len(t.assertions)
	feasible, err := t.rebuild()
	if err != nil || feasible || state.inconsistent {
		return err
	}
	literals := make([]SMTLiteral, 0, len(t.assertions))
	for _, assertion := range t.assertions {
		literal := NewSMTLiteral(t.atoms[assertion.atom].variable, false)
		if !assertion.truth {
			literal = literal.Negated()
		}
		literals = append(literals, literal)
	}
	return state.SetTheoryConflict(t.FamilyID(), literals)
}

func (t *SMTLinearArithmeticTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	if err := t.equalityAdapter.PushScope(); err != nil {
		_ = t.SMTTheoryBase.PopScopes(1)
		return err
	}
	t.scopes = append(t.scopes, smtLinearArithmeticScope{len(t.atoms), len(t.assertions), t.assertionHead, t.unsupported})
	return nil
}

func (t *SMTLinearArithmeticTheory) PopScopes(count uint32) error {
	if count == 0 || int(count) > len(t.scopes) {
		return ast.ErrASTDomain
	}
	level := len(t.scopes) - int(count)
	scope := t.scopes[level]
	for index := len(t.atoms); index > scope.atoms; index-- {
		delete(t.atomByBool, t.atoms[index-1].variable)
	}
	t.atoms = t.atoms[:scope.atoms]
	t.assertions = t.assertions[:scope.assertions]
	t.assertionHead = scope.head
	t.unsupported = scope.unsupported
	t.scopes = t.scopes[:level]
	if err := t.equalityAdapter.PopScopes(count); err != nil {
		return err
	}
	if err := t.SMTTheoryBase.PopScopes(count); err != nil {
		return err
	}
	t.simplexVariables = t.simplexVariables[:t.NumVariables()]
	return func() error { _, err := t.rebuild(); return err }()
}

func (t *SMTLinearArithmeticTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	if t.CanPropagate() {
		if err := t.Propagate(); err != nil {
			return SMTFinalCheckGiveUp, err
		}
		return SMTFinalCheckContinue, nil
	}
	if t.unsupported {
		return SMTFinalCheckGiveUp, nil
	}
	if t.integer {
		if _, _, fractional := t.fractionalVariable(t.solver); fractional {
			var nodes uint32
			status, err := t.integerFeasible(nil, &nodes)
			if err != nil {
				return SMTFinalCheckGiveUp, err
			}
			if status == lbool.Undef {
				return SMTFinalCheckGiveUp, nil
			}
			if status == lbool.False {
				state, stateErr := t.State()
				if stateErr != nil {
					return SMTFinalCheckGiveUp, stateErr
				}
				literals := make([]SMTLiteral, 0, len(t.assertions))
				for _, assertion := range t.assertions {
					literal := NewSMTLiteral(t.atoms[assertion.atom].variable, false)
					if !assertion.truth {
						literal = literal.Negated()
					}
					literals = append(literals, literal)
				}
				if len(literals) == 0 {
					return SMTFinalCheckGiveUp, nil
				}
				if stateErr = state.SetTheoryConflict(t.FamilyID(), literals); stateErr != nil {
					return SMTFinalCheckGiveUp, stateErr
				}
				return SMTFinalCheckContinue, nil
			}
		}
	}
	return SMTFinalCheckDone, nil
}

func (t *SMTLinearArithmeticTheory) InitializeSearch() error {
	return t.equalityAdapter.InitializeSearch()
}
func (t *SMTLinearArithmeticTheory) Restart() error { return t.equalityAdapter.Restart() }

func (t *SMTLinearArithmeticTheory) InternalizeEquality(expression ast.ASTHandle, variable int32) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	left, err := state.manager.GetArg(expression, 0)
	if err != nil {
		return err
	}
	right, err := state.manager.GetArg(expression, 1)
	if err != nil {
		return err
	}
	lower, err := state.manager.MkFamilyApp(t.FamilyID(), smtArithOpLE, nil, []ast.ASTHandle{left, right})
	if err != nil {
		return err
	}
	upper, err := state.manager.MkFamilyApp(t.FamilyID(), smtArithOpGE, nil, []ast.ASTHandle{left, right})
	if err != nil {
		return err
	}
	if err = state.Internalize(lower, true); err != nil {
		return err
	}
	if err = state.Internalize(upper, true); err != nil {
		return err
	}
	lowerLiteral, err := state.Literal(lower)
	if err != nil {
		return err
	}
	upperLiteral, err := state.Literal(upper)
	if err != nil {
		return err
	}
	equality := NewSMTLiteral(variable, false)
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality.Negated(), lowerLiteral}); err != nil {
		return err
	}
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality.Negated(), upperLiteral}); err != nil {
		return err
	}
	if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{equality, lowerLiteral.Negated(), upperLiteral.Negated()}); err != nil {
		return err
	}
	return t.equalityAdapter.AddRelevancy(left, right, expression, lower, upper)
}

func (t *SMTLinearArithmeticTheory) NewEquality(left, right SMTTheoryVar) error {
	return t.equalityAdapter.MkAxioms(left, right)
}
func (t *SMTLinearArithmeticTheory) NewDisequality(left, right SMTTheoryVar) error {
	return t.equalityAdapter.MkAxioms(left, right)
}
func (*SMTLinearArithmeticTheory) Name() string { return "arithmetic" }

func (t *SMTLinearArithmeticTheory) InitializeModel(generator *SMTModelGenerator) error {
	state, err := t.State()
	if err != nil || generator == nil || generator.manager != state.manager {
		if err != nil {
			return err
		}
		return ast.ErrASTDomain
	}
	t.delta.SetInt64(1)
	for variable := uint32(0); variable < t.solver.NumVariables(); variable++ {
		value := t.solver.Value(variable)
		for _, item := range []struct {
			bound simplex.Infinitesimal
			valid bool
			lower bool
		}{
			func() struct {
				bound        simplex.Infinitesimal
				valid, lower bool
			} {
				b, ok := t.solver.Lower(variable)
				return struct {
					bound        simplex.Infinitesimal
					valid, lower bool
				}{b, ok, true}
			}(),
			func() struct {
				bound        simplex.Infinitesimal
				valid, lower bool
			} {
				b, ok := t.solver.Upper(variable)
				return struct {
					bound        simplex.Infinitesimal
					valid, lower bool
				}{b, ok, false}
			}(),
		} {
			if !item.valid {
				continue
			}
			var main, epsilon big.Rat
			main.Sub(&value.Main, &item.bound.Main)
			epsilon.Sub(&value.Epsilon, &item.bound.Epsilon)
			if !item.lower {
				main.Neg(&main)
				epsilon.Neg(&epsilon)
			}
			if main.Sign() > 0 && epsilon.Sign() < 0 {
				candidate := new(big.Rat).Quo(&main, new(big.Rat).Neg(&epsilon))
				candidate.Quo(candidate, big.NewRat(2, 1))
				if candidate.Cmp(&t.delta) < 0 {
					t.delta.Set(candidate)
				}
			}
		}
	}
	factory, err := protomodel.NewArithmeticFactory(state.manager)
	if err != nil {
		return err
	}
	return generator.RegisterFactory(factory)
}

func (t *SMTLinearArithmeticTheory) ModelValue(node *SMTEnode, generator *SMTModelGenerator) (SMTModelValueProcedure, error) {
	if node == nil || generator == nil || !t.IsAttachedToVariable(node) {
		return nil, ast.ErrASTDomain
	}
	state, err := t.State()
	if err != nil || generator.manager != state.manager {
		if err != nil {
			return nil, err
		}
		return nil, ast.ErrASTDomain
	}
	theoryVariable := node.TheoryVar(t.FamilyID())
	value := t.solver.Value(t.simplexVariables[theoryVariable])
	var epsilon, rational big.Rat
	epsilon.Mul(&t.delta, &value.Epsilon)
	rational.Add(&value.Main, &epsilon)
	numeral, err := state.manager.ArithmeticNumeralRat(&rational, t.integer)
	if err != nil {
		return nil, err
	}
	return NewSMTExpressionValueProcedure(numeral)
}
