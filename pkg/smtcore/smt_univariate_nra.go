// Initial end-to-end QF_NRA conjunction path. Reuses the ported AST polynomial
// translator and NLSAT interval/witness engine; other inputs use ordinary SMT.
package smtcore

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/nlsat"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

type smtNRAAtom struct {
	left, right ast.ASTHandle
	kind        nlsat.AtomKind
	negative    bool
}
type smtNRAInput struct {
	variable  ast.ASTHandle
	atoms     []smtNRAAtom
	nonlinear bool
}

func (in *smtNRAInput) term(m *ast.ASTManager, h ast.ASTHandle, family ast.FamilyID) (bool, bool, error) {
	kind, err := m.GetKind(h)
	if err != nil || kind != ast.ASTApp {
		return false, false, err
	}
	count, err := m.GetNumArgs(h)
	if err != nil {
		return false, false, err
	}
	fid, err := m.GetFamilyID(h)
	if err != nil {
		return false, false, err
	}
	if fid == ast.NullFamilyID && count == 0 {
		sort, err := m.GetSort(h)
		if err != nil {
			return false, false, err
		}
		sf, err := m.GetFamilyID(sort)
		if err != nil {
			return false, false, err
		}
		sk, err := m.GetDeclKind(sort)
		if err != nil {
			return false, false, err
		}
		if sf != family || sk != 0 {
			return false, false, nil
		}
		if !in.variable.IsNull() && in.variable != h {
			return false, false, nil
		}
		in.variable = h
		return true, true, nil
	}
	if fid != family {
		return false, false, nil
	}
	op, err := m.GetDeclKind(h)
	if err != nil {
		return false, false, err
	}
	if op == 0 {
		_, _, matched, err := m.ArithmeticNumeralText(h)
		return matched, false, err
	}
	switch op {
	case 6, 7, 8, 9, 14, 18:
	default:
		return false, false, nil
	}
	varChildren := 0
	limit := count
	if op == 18 {
		if count != 2 {
			return false, false, nil
		}
		exponent, err := m.GetArg(h, 1)
		if err != nil {
			return false, false, err
		}
		text, _, matched, err := m.ArithmeticNumeralText(exponent)
		if err != nil || !matched {
			return false, false, err
		}
		power, err := strconv.ParseUint(text, 10, 32)
		if err != nil {
			return false, false, nil
		}
		if power > 1 {
			in.nonlinear = true
		}
		limit = 1
	}
	for i := uint32(0); i < limit; i++ {
		arg, err := m.GetArg(h, i)
		if err != nil {
			return false, false, err
		}
		ok, variable, err := in.term(m, arg, family)
		if err != nil || !ok {
			return false, false, err
		}
		if variable {
			varChildren++
		}
	}
	if op == 9 && varChildren > 1 {
		in.nonlinear = true
	}
	return true, varChildren > 0, nil
}
func (in *smtNRAInput) formula(k *SMTKernel, h ast.ASTHandle, negative bool, family ast.FamilyID) (bool, error) {
	m := k.manager
	if m.IsTrue(h) || m.IsFalse(h) {
		return false, nil
	} // ordinary backend handles pure constants
	kind, err := m.GetKind(h)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	op, basic, err := k.internalizer.basicOperation(h)
	if err != nil {
		return false, err
	}
	count, err := m.GetNumArgs(h)
	if err != nil {
		return false, err
	}
	if basic && op == ast.ASTOpNot && count == 1 {
		arg, err := m.GetArg(h, 0)
		if err != nil {
			return false, err
		}
		return in.formula(k, arg, !negative, family)
	}
	if basic && op == ast.ASTOpAnd && !negative {
		for i := uint32(0); i < count; i++ {
			arg, err := m.GetArg(h, i)
			if err != nil {
				return false, err
			}
			ok, err := in.formula(k, arg, false, family)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}
	relation := nlsat.AtomEQ
	if !(basic && op == ast.ASTOpEq) {
		fid, err := m.GetFamilyID(h)
		if err != nil || fid != family {
			return false, err
		}
		op, err = m.GetDeclKind(h)
		if err != nil {
			return false, err
		}
		switch op {
		case smtArithOpLT:
			relation = nlsat.AtomLT
		case smtArithOpGT:
			relation = nlsat.AtomGT
		case smtArithOpLE:
			relation = nlsat.AtomGT
			negative = !negative
		case smtArithOpGE:
			relation = nlsat.AtomLT
			negative = !negative
		default:
			return false, nil
		}
	}
	if count != 2 {
		return false, nil
	}
	a, err := m.GetArg(h, 0)
	if err != nil {
		return false, err
	}
	b, err := m.GetArg(h, 1)
	if err != nil {
		return false, err
	}
	for _, h := range []ast.ASTHandle{a, b} {
		ok, _, err := in.term(m, h, family)
		if err != nil || !ok {
			return false, err
		}
	}
	in.atoms = append(in.atoms, smtNRAAtom{a, b, relation, negative})
	return true, nil
}

type smtNRAWorkspace struct {
	kernel      *SMTKernel
	q           *unumeral.RationalManager
	am          *polynomial.AlgebraicManager
	pm          *polynomial.MultivariatePolynomialManager
	convert     *ast.Expr2Polynomial
	conjunction *nlsat.UnivariateConjunction
	cooperation *cooperate.Cooperation
	primes      *primegen.Generator
	params      uparams.ParamsRef
	paramsReady bool
	witness     polynomial.Algebraic
}

func (w *smtNRAWorkspace) Close() error {
	if w.am != nil {
		if err := w.am.CleanupRetained(); err != nil {
			return err
		}
	}
	if w.conjunction != nil {
		if err := w.conjunction.Close(); err != nil {
			return err
		}
		w.conjunction = nil
	}
	if w.convert != nil {
		if err := w.convert.Close(); err != nil {
			return err
		}
		w.convert = nil
	}
	if w.pm != nil {
		if err := w.pm.CloseChecked(); err != nil {
			return err
		}
		w.pm = nil
	}
	if w.am != nil {
		if err := w.am.Del(&w.witness); err != nil {
			return err
		}
		if err := w.am.Close(); err != nil {
			return err
		}
		w.am = nil
	}
	if w.q != nil {
		if err := w.q.CloseChecked(); err != nil {
			return err
		}
		w.q = nil
	}
	if w.primes != nil {
		if err := w.primes.Close(); err != nil {
			return err
		}
		w.primes = nil
	}
	if w.cooperation != nil {
		if err := w.cooperation.Close(); err != nil {
			return err
		}
		w.cooperation = nil
	}
	if w.paramsReady {
		if err := w.kernel.paramGlobals.Context.Del(&w.params); err != nil {
			return err
		}
		w.paramsReady = false
	}
	return nil
}
func (k *SMTKernel) tryUnivariateNRA() (result lbool.LBool, handled bool, err error) {
	if k.setupLogic != "QF_NRA" || !k.nraInputsValid || len(k.nraInputs) == 0 {
		return lbool.Undef, false, nil
	}
	proofs, err := k.manager.ProofsEnabled()
	if err != nil || proofs {
		return lbool.Undef, false, err
	}
	family, err := k.manager.RegisterArithmeticPlugin()
	if err != nil {
		return lbool.Undef, false, err
	}
	in := smtNRAInput{}
	for _, a := range k.nraInputs {
		h, e := k.manager.RefGet(a)
		if e != nil {
			return lbool.Undef, false, e
		}
		ok, e := in.formula(k, h, false, family)
		if e != nil || !ok {
			return lbool.Undef, false, e
		}
	}
	if !in.nonlinear || in.variable.IsNull() || k.paramGlobals == nil {
		return lbool.Undef, false, nil
	}
	handled = true
	result = lbool.Undef
	w := &smtNRAWorkspace{kernel: k}
	k.univariateNRA = w
	defer func() {
		e := w.Close()
		if e == nil {
			k.univariateNRA = nil
		}
		err = errors.Join(err, e)
		if err != nil {
			result = lbool.Undef
		}
	}()
	w.q, err = k.manager.NumeralService().NewPrivateRationalManager()
	if err != nil {
		return
	}
	w.cooperation, err = cooperate.NewCooperation(k.manager.Budget())
	if err != nil {
		return
	}
	w.primes, err = primegen.NewGenerator(k.manager.Budget())
	if err != nil {
		return
	}
	w.params, err = k.paramGlobals.Context.NewValue()
	if err != nil {
		return
	}
	w.paramsReady = true
	w.am, err = polynomial.NewAlgebraicManager(polynomial.AlgebraicManagerConfig{Rationals: k.manager.NumeralService(), Numerals: w.q, Limit: &k.manager.Limit, Cooperation: w.cooperation, Params: &w.params, Globals: k.paramGlobals, Factorization: polynomial.UPolynomialFactorizationConfig{Primes: w.primes}})
	if err != nil {
		return
	}
	w.pm, err = polynomial.NewMultivariatePolynomialManager(polynomial.MultivariatePolynomialManagerConfig{Core: polynomial.MultivariatePolynomialConfig{Integer: w.q.Integers(), Rationals: w.q, Checkpoint: k.manager.Limit.Inc, Cooperation: func() error { return nil }}, Factorization: polynomial.UPolynomialFactorizationConfig{Primes: w.primes}})
	if err != nil {
		return
	}
	w.convert, err = ast.NewExpr2Polynomial(ast.Expr2PolynomialConfig{AST: k.manager, Polynomial: w.pm})
	if err != nil {
		return
	}
	// Pin the one real constant to polynomial variable zero in the real mapper.
	core, e := w.pm.ArithmeticCore()
	if e != nil {
		err = e
		return
	}
	_, err = core.MkVar()
	if err != nil {
		return
	}
	mapping, e := w.convert.Mapping()
	if e != nil {
		err = e
		return
	}
	err = mapping.Insert(in.variable, 0)
	if err != nil {
		return
	}
	w.conjunction, err = nlsat.NewUnivariateConjunction(w.am, w.pm, 0)
	if err != nil {
		return
	}
	for _, a := range in.atoms {
		difference, e := k.manager.MkFamilyApp(family, smtArithOpSub, nil, []ast.ASTHandle{a.left, a.right})
		if e != nil {
			err = e
			return
		}
		ref, e := k.manager.NewRef(difference)
		if e != nil {
			err = e
			return
		}
		p, d, _, e := w.convert.ToPolynomial(difference)
		e = errors.Join(e, k.manager.DelRef(&ref))
		if e != nil {
			err = e
			return
		}
		value, e := p.Get()
		if e == nil {
			denominator, de := d.Get()
			if de == nil {
				var sign int32
				sign, de = w.q.Integers().Sign(denominator)
				if de == nil && sign <= 0 {
					de = ast.ErrASTDomain
				}
			}
			e = de
		}
		if e == nil {
			e = w.conjunction.Add(value, a.kind, a.negative)
		}
		e = errors.Join(e, d.Close(), p.Close())
		if e != nil {
			err = e
			return
		}
	}
	sat, e := w.conjunction.Check(&w.witness)
	if e != nil {
		err = e
		return
	}
	if !sat {
		return lbool.False, true, nil
	}
	result = lbool.True
	if !k.produceModel {
		return
	}
	md, e := model.NewModel(k.manager)
	if e != nil {
		err = e
		return
	}
	published := false
	defer func() {
		if !published {
			err = errors.Join(err, md.Close())
		}
	}()
	rational, e := w.am.IsRational(&w.witness)
	if e != nil {
		err = e
		return
	}
	var numeral ast.ASTHandle
	if rational {
		var r unumeral.Rational
		err = w.am.ToRational(&w.witness, &r)
		if err != nil {
			return
		}
		text, e := w.q.String(&r)
		e = errors.Join(e, w.q.Del(&r))
		if e != nil {
			err = e
			return
		}
		number, ok := new(big.Rat).SetString(text)
		if !ok {
			err = ast.ErrASTDomain
			return
		}
		numeral, err = k.manager.ArithmeticNumeralRat(number, false)
	} else {
		previous, e := uparams.BindProcessGlobalParamsService(k.paramGlobals)
		if e != nil {
			err = e
			return
		}
		target, e := k.manager.ArithmeticAlgebraicManager()
		var value polynomial.Algebraic
		if e == nil {
			e = target.CopyFromCore(&value, w.am.AlgebraicRootCore, &w.witness)
		}
		if e == nil {
			numeral, e = k.manager.ArithmeticAlgebraicNumeral(&value)
		}
		if target != nil {
			e = errors.Join(e, target.Del(&value))
		}
		_, restore := uparams.BindProcessGlobalParamsService(previous)
		err = errors.Join(e, restore)
	}
	if err != nil {
		return
	}
	decl, e := k.manager.GetDecl(in.variable)
	if e != nil {
		err = e
		return
	}
	err = md.Core().RegisterConstant(decl, numeral)
	if err != nil {
		return
	}
	err = md.Core().IncRef()
	if err != nil {
		return
	}
	k.generated = md
	published = true
	return
}

func (k *SMTKernel) restoreNRAInputs(size int) error {
	for len(k.nraInputs) > size {
		last := len(k.nraInputs) - 1
		if err := k.manager.DelRef(&k.nraInputs[last]); err != nil {
			return err
		}
		k.nraInputs = k.nraInputs[:last]
	}
	return nil
}
