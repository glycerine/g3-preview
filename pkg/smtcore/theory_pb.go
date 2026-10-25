// Source: src/smt/theory_pb.cpp and src/smt/theory_pb.h retained exact
// pseudo-Boolean atom semantics.
package smtcore

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/backtrack"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
)

type smtPseudoBooleanStatistics struct {
	predicates                         uint64
	clauses                            uint64
	compiled                           uint64
	compiledVariables, compiledClauses uint64
	propagations, conflicts            uint64
}

// SMTPseudoBooleanTheory keeps concrete normalized constraints and native
// variable/literal watches, weighted conflict resolution and restart-triggered
// sorting-network compilation, with the optional shared-row simplex relaxation.
type SMTPseudoBooleanTheory struct {
	*SMTTheoryBase
	statistics     smtPseudoBooleanStatistics
	watches        map[int32]*smtPBVariableWatches
	constraints    []*smtPBConstraint
	toCompile      []*smtPBConstraint
	limits         []int
	trail          *backtrack.TrailStack
	controls       SMTPBState
	configured     bool
	parameterOwner *SMTPBParams // borrowed only until the source setup boundary
	simplex        *smtPBSimplex
}

func NewSMTPseudoBooleanTheory(family ast.FamilyID) (*SMTPseudoBooleanTheory, error) {
	base, err := NewSMTTheoryBase(family)
	if err != nil {
		return nil, err
	}
	return &SMTPseudoBooleanTheory{SMTTheoryBase: base, trail: backtrack.NewTrailStack(), controls: SMTPBState{ConflictFrequency: 1000, LearnComplements: true, EnableCompilation: true}}, nil
}

func (t *SMTPseudoBooleanTheory) operation(expression ast.ASTHandle) (int32, bool, error) {
	state, err := t.State()
	if err != nil {
		return 0, false, err
	}
	family, err := state.manager.GetFamilyID(expression)
	if err != nil || family != t.FamilyID() {
		return 0, false, err
	}
	op, err := state.manager.GetDeclKind(expression)
	return op, err == nil, err
}

func smtPseudoBooleanRational(manager *ast.ASTManager, value unumeral.OwnedRational) (*big.Rat, error) {
	service := manager.NumeralService()
	text, textErr := service.String(&value)
	err := errors.Join(textErr, service.Del(&value))
	if err != nil {
		return nil, err
	}
	result, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, ast.ErrASTDomain
	}
	return result, nil
}

func (t *SMTPseudoBooleanTheory) InternalizeAtom(expression ast.ASTHandle, _ bool) (bool, error) {
	return t.internalizeAtom(expression, false)
}

func (t *SMTPseudoBooleanTheory) internalizeAtom(expression ast.ASTHandle, retainAssignments bool) (bool, error) {
	if err := t.Setup(); err != nil {
		return false, err
	}
	op, own, err := t.operation(expression)
	if err != nil || !own || op < ast.ASTPBOpAtMost || op > ast.ASTPBOpAuxBool {
		return false, err
	}
	state, _ := t.State()
	if _, found := state.boolVar[expression]; found {
		return false, nil
	}
	variable, err := state.MkBoolVar(expression)
	if err != nil {
		return false, err
	}
	if err = state.SetVariableTheory(variable, t.FamilyID()); err != nil {
		return false, err
	}
	t.statistics.predicates++
	if op == ast.ASTPBOpAuxBool {
		return true, nil
	}
	c := &smtPBConstraint{literal: NewSMTLiteral(variable, false), equality: op == ast.ASTPBOpEQ}
	bound, err := state.manager.PseudoBooleanBound(expression)
	if err != nil {
		return false, err
	}
	k, err := smtPseudoBooleanRational(state.manager, bound)
	if err != nil {
		return false, err
	}
	c.args[0].bound.Set(k)
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < count; i++ {
		h, err := state.manager.GetArg(expression, i)
		if err != nil {
			return false, err
		}
		literal, err := t.compileArgument(h)
		if err != nil {
			return false, err
		}
		coefficient, err := state.manager.PseudoBooleanCoefficient(expression, i)
		if err != nil {
			return false, err
		}
		q, err := smtPseudoBooleanRational(state.manager, coefficient)
		if err != nil {
			return false, err
		}
		value, err := state.Assignment(literal)
		if err != nil {
			return false, err
		}
		// A strengthened consequence must keep its scoped false inputs in
		// the definition. Folding them into an axiom can erase assumption
		// premises and produce an invalid learned Boolean unit.
		if retainAssignments {
			value = lbool.Undef
		}
		switch value {
		case lbool.True:
			c.args[0].bound.Sub(&c.args[0].bound, q)
		case lbool.Undef:
			c.args[0].terms = append(c.args[0].terms, smtPBTerm{literal, *q})
		}
	}
	a := &c.args[0]
	if op == ast.ASTPBOpAtMost || op == ast.ASTPBOpLE {
		a.bound.Neg(&a.bound)
		for i := range a.terms {
			a.terms[i].coeff.Neg(&a.terms[i].coeff)
		}
	}
	a.unique()
	truth := a.normalize(c.equality)
	a.prune(c.equality)
	if truth != lbool.Undef {
		lit := c.literal
		if truth == lbool.False {
			lit = lit.Negated()
		}
		return true, state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{lit})
	}
	if !c.equality {
		a.negate()
		a.negate()
		c.args[1] = a.copy()
		c.args[1].negate()
		if a.bound.Cmp(big.NewRat(1, 1)) == 0 && !t.controls.EnableSimplex {
			literals := []SMTLiteral{c.literal.Negated()}
			for _, term := range a.terms {
				literals = append(literals, term.literal)
				if err = state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{c.literal, term.literal.Negated()}); err != nil {
					return false, err
				}
				t.statistics.clauses++
			}
			t.statistics.clauses++
			return true, state.addTheoryAxiom(t.FamilyID(), literals)
		}
	}
	t.initCompilation(c)
	t.initVariableWatch(c)
	t.info(variable).constraint = c
	t.constraints = append(t.constraints, c)
	if err = t.simplexInternalize(c); err != nil {
		return false, err
	}
	return true, nil
}

func (*SMTPseudoBooleanTheory) InternalizeTerm(ast.ASTHandle) (bool, error)        { return false, nil }
func (*SMTPseudoBooleanTheory) InternalizeEquality(ast.ASTHandle, int32) error     { return nil }
func (*SMTPseudoBooleanTheory) ApplySortConstraint(*SMTEnode, ast.ASTHandle) error { return nil }
func (*SMTPseudoBooleanTheory) UseDisequalities() bool                             { return false }
func (*SMTPseudoBooleanTheory) NewEquality(SMTTheoryVar, SMTTheoryVar) error       { return ast.ErrASTDomain }
func (*SMTPseudoBooleanTheory) NewDisequality(SMTTheoryVar, SMTTheoryVar) error {
	return ast.ErrASTDomain
}
func (t *SMTPseudoBooleanTheory) Assign(v int32, value bool) error { return t.assignNative(v, value) }
func (*SMTPseudoBooleanTheory) FinalCheck() (SMTFinalCheckStatus, error) {
	return SMTFinalCheckDone, nil
}
func (*SMTPseudoBooleanTheory) Name() string { return "pb" }

func (t *SMTPseudoBooleanTheory) PushScope() error {
	if err := t.SMTTheoryBase.PushScope(); err != nil {
		return err
	}
	t.trail.PushScope()
	t.limits = append(t.limits, len(t.constraints))
	return nil
}

func (t *SMTPseudoBooleanTheory) PopScopes(count uint32) error {
	if count == 0 || count > uint32(len(t.limits)) {
		return ast.ErrASTDomain
	}
	if err := t.trail.PopScope(count); err != nil {
		return err
	}
	level := len(t.limits) - int(count)
	limit := t.limits[level]
	for index := len(t.constraints) - 1; index >= limit; index-- {
		c := t.constraints[index]
		t.clearWatch(c)
		t.info(c.literal.Var()).constraint = nil
		if err := t.simplexRemove(c); err != nil {
			return err
		}
		t.toCompile = smtPBRemove(t.toCompile, c)
	}
	clear(t.constraints[limit:])
	t.constraints = t.constraints[:limit]
	t.limits = t.limits[:level]
	return t.SMTTheoryBase.PopScopes(count)
}

func (t *SMTPseudoBooleanTheory) Close() error {
	if t == nil {
		return ast.ErrASTHandle
	}
	if err := t.trail.Reset(); err != nil {
		return err
	}
	t.watches = nil
	t.constraints = nil
	t.toCompile = nil
	t.limits = nil
	t.variables = nil
	t.parameterOwner = nil
	t.simplex = nil
	return nil
}

func (t *SMTPseudoBooleanTheory) CollectStatistics(target *statistics.Statistics) error {
	if t == nil || target == nil {
		return ast.ErrASTHandle
	}
	target.UpdateUint64("pb predicates", t.statistics.predicates)
	target.UpdateUint64("pb clauses", t.statistics.clauses)
	target.UpdateUint64("pb compilations", t.statistics.compiled)
	target.UpdateUint64("pb compiled vars", t.statistics.compiledVariables)
	target.UpdateUint64("pb compiled clauses", t.statistics.compiledClauses)
	target.UpdateUint64("pb propagations", t.statistics.propagations)
	target.UpdateUint64("pb conflicts", t.statistics.conflicts)
	if t.simplex != nil {
		s := t.simplex.solver
		target.UpdateUint64("simplex add rows", s.Matrix().AddRowCount())
		target.UpdateUint64("simplex num pivots", s.Stats.NumPivots)
		target.UpdateUint64("simplex num infeasible", s.Stats.NumInfeasible)
		target.UpdateUint64("simplex num checks", s.Stats.NumChecks)
	}
	return nil
}

func (t *SMTPseudoBooleanTheory) ResetStatistics() {
	if t != nil {
		t.statistics = smtPseudoBooleanStatistics{}
	}
}
