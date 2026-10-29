package pdr

// Source: the retained level-frame, safe-assumption, subset-core and model
// paths in src/muz/pdr/pdr_prop_solver.h and pdr_prop_solver.cpp.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

var ErrPropSolverOwner = errors.New("muz/pdr: invalid or closed propositional solver")

type PropCheckResult struct {
	Status       lbool.LBool
	Core         []ast.ASTHandle
	Model        *model.ModelCore
	AssumesLevel bool
}

// PropSolver is the PDR-facing frame and safe-assumption wrapper over one
// guarded SMTContext. Returned cores borrow their input formulas; synthesized
// expansion terms remain valid until the next check or Close. A nonnil model
// is an owning model reference and must be released by the caller.
type PropSolver struct {
	manager        *ast.ASTManager
	contexts       *SMTContextManager
	context        *SMTContext
	name           string
	levelPositive  []ast.ASTHandle
	levelNegative  []ast.ASTHandle
	levelAtoms     map[ast.ASTHandle]bool
	proxies        []ast.ASTHandle
	auxiliary      map[ast.ASTHandle]bool
	references     []ast.ASTRef
	coreReferences []ast.ASTRef
	closed         bool
}

func NewPropSolver(manager *ast.ASTManager, contexts *SMTContextManager, background ast.ASTHandle, name string) (_ *PropSolver, resultErr error) {
	if manager == nil || contexts == nil {
		return nil, ErrPropSolverOwner
	}
	context, err := contexts.Fresh()
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			resultErr = errors.Join(resultErr, context.Close())
		}
	}()
	if err = context.Assert(background); err != nil {
		return nil, err
	}
	p := &PropSolver{
		manager:    manager,
		contexts:   contexts,
		context:    context,
		name:       name,
		levelAtoms: make(map[ast.ASTHandle]bool),
		auxiliary:  make(map[ast.ASTHandle]bool),
	}
	committed = true
	return p, nil
}

func (p *PropSolver) open() error {
	if p == nil || p.closed || p.manager == nil || p.context == nil {
		return ErrPropSolverOwner
	}
	return p.context.open()
}

func (p *PropSolver) pin(handle ast.ASTHandle) error {
	reference, err := p.manager.NewRef(handle)
	if err != nil {
		return err
	}
	p.references = append(p.references, reference)
	return nil
}

func (p *PropSolver) addLevel() error {
	boolean, err := p.manager.MkBoolSort()
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%s#level_%d", p.name, len(p.levelPositive))
	positive, err := p.manager.MkFreshConst(prefix, boolean)
	if err != nil {
		return err
	}
	if err = p.pin(positive); err != nil {
		return err
	}
	negative, err := p.manager.MkNot(positive)
	if err != nil {
		return err
	}
	if err = p.pin(negative); err != nil {
		return err
	}
	declaration, err := p.manager.GetDecl(positive)
	if err != nil {
		return err
	}
	p.levelPositive = append(p.levelPositive, positive)
	p.levelNegative = append(p.levelNegative, negative)
	p.levelAtoms[positive] = true
	p.levelAtoms[negative] = true
	p.auxiliary[declaration] = true
	return nil
}

func (p *PropSolver) ensureLevel(level uint32) error {
	for uint32(len(p.levelPositive)) <= level {
		if err := p.addLevel(); err != nil {
			return err
		}
	}
	return nil
}

func (p *PropSolver) LevelCount() (uint32, error) {
	if err := p.open(); err != nil {
		return 0, err
	}
	return uint32(len(p.levelPositive)), nil
}

// EnsureLevel creates every frame guard through level. It exposes the source
// add_level operation to predicate transformers without asserting a dummy
// formula into a newly created frame.
func (p *PropSolver) EnsureLevel(level uint32) error {
	if err := p.open(); err != nil {
		return err
	}
	return p.ensureLevel(level)
}

func (p *PropSolver) IsAuxSymbol(declaration ast.ASTHandle) (bool, error) {
	if err := p.open(); err != nil {
		return false, err
	}
	if p.auxiliary[declaration] {
		return true, nil
	}
	return p.context.IsAuxPredicate(declaration)
}

func (p *PropSolver) AddFormula(formula ast.ASTHandle) error {
	if err := p.open(); err != nil {
		return err
	}
	return p.context.Assert(formula)
}

func (p *PropSolver) AddLevelFormula(formula ast.ASTHandle, level uint32) error {
	if err := p.open(); err != nil {
		return err
	}
	if err := p.ensureLevel(level); err != nil {
		return err
	}
	guarded, err := p.manager.MkOr(formula, p.levelPositive[level])
	if err != nil {
		return err
	}
	return p.context.Assert(guarded)
}

func (p *PropSolver) isDirectAssumption(expression ast.ASTHandle) (bool, error) {
	if negated, argument, err := p.manager.IsNot(expression); err != nil {
		return false, err
	} else if negated {
		expression = argument
	}
	kind, err := p.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return false, err
	}
	declaration, err := p.manager.GetDecl(expression)
	if err != nil {
		return false, err
	}
	family, err := p.manager.GetFamilyID(declaration)
	if err != nil || family != ast.NullFamilyID {
		return false, err
	}
	arity, err := p.manager.GetArity(declaration)
	return arity == 0, err
}

func (p *PropSolver) proxy(index uint32) (ast.ASTHandle, error) {
	if index < uint32(len(p.proxies)) {
		return p.proxies[index], nil
	}
	boolean, err := p.manager.MkBoolSort()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	proxy, err := p.manager.MkFreshConst(fmt.Sprintf("pdr_proxy_%d", len(p.proxies)), boolean)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = p.pin(proxy); err != nil {
		return ast.ASTHandle{}, err
	}
	declaration, err := p.manager.GetDecl(proxy)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	p.proxies = append(p.proxies, proxy)
	p.auxiliary[declaration] = true
	return proxy, nil
}

type safeAssumptions struct {
	atoms          []ast.ASTHandle
	proxyToFormula map[ast.ASTHandle]ast.ASTHandle
	formulaToProxy map[ast.ASTHandle]ast.ASTHandle
	references     []ast.ASTRef
	generated      map[ast.ASTHandle]bool
}

func (p *PropSolver) arithmeticEquality(expression ast.ASTHandle) (left, right ast.ASTHandle, matched bool, result error) {
	kind, err := p.manager.GetKind(expression)
	if err != nil || kind != ast.ASTApp {
		return left, right, false, err
	}
	family, err := p.manager.GetFamilyID(expression)
	if err != nil || family != p.manager.BasicFamilyID() {
		return left, right, false, err
	}
	operation, err := p.manager.GetDeclKind(expression)
	if err != nil || operation != ast.ASTOpEq {
		return left, right, false, err
	}
	left, err = p.manager.GetArg(expression, 0)
	if err != nil {
		return left, right, false, err
	}
	right, err = p.manager.GetArg(expression, 1)
	if err != nil {
		return left, right, false, err
	}
	sort, err := p.manager.GetSort(left)
	if err != nil {
		return left, right, false, err
	}
	arithmetic, err := p.manager.ArithmeticFamilyID()
	if err != nil {
		return left, right, false, err
	}
	sortFamily, err := p.manager.GetFamilyID(sort)
	if err != nil || sortFamily != arithmetic {
		return left, right, false, err
	}
	sortKind, err := p.manager.GetDeclKind(sort)
	if err != nil {
		return left, right, false, err
	}
	return left, right, sortKind == 0 || sortKind == 1, nil
}

func (p *PropSolver) expandArithmeticEqualities(atoms []ast.ASTHandle) (result []ast.ASTHandle, references []ast.ASTRef, generated map[ast.ASTHandle]bool, resultErr error) {
	generated = make(map[ast.ASTHandle]bool)
	committed := false
	defer func() {
		if committed {
			return
		}
		for index := len(references) - 1; index >= 0; index-- {
			resultErr = errors.Join(resultErr, p.manager.DelRef(&references[index]))
		}
	}()
	for index := 0; index < len(atoms); index++ {
		left, right, matched, err := p.arithmeticEquality(atoms[index])
		if err != nil {
			return nil, references, generated, err
		}
		if !matched {
			continue
		}
		lessEqual, err := p.manager.MkArithmeticLE(left, right)
		if err != nil {
			return nil, references, generated, err
		}
		lessEqualRef, err := p.manager.NewRef(lessEqual)
		if err != nil {
			return nil, references, generated, err
		}
		references = append(references, lessEqualRef)
		generated[lessEqual] = true
		greaterEqual, err := p.manager.MkArithmeticGE(left, right)
		if err != nil {
			return nil, references, generated, err
		}
		greaterEqualRef, err := p.manager.NewRef(greaterEqual)
		if err != nil {
			return nil, references, generated, err
		}
		references = append(references, greaterEqualRef)
		generated[greaterEqual] = true
		atoms[index] = lessEqual
		if index+1 == len(atoms) {
			atoms = append(atoms, greaterEqual)
		} else {
			next := atoms[index+1]
			atoms = append(atoms, next)
			atoms[index+1] = greaterEqual
		}
		index++
	}
	committed = true
	return atoms, references, generated, nil
}

func (s *safeAssumptions) close(manager *ast.ASTManager) (result error) {
	for index := len(s.references) - 1; index >= 0; index-- {
		result = errors.Join(result, manager.DelRef(&s.references[index]))
	}
	s.references = nil
	return result
}

func (p *PropSolver) makeSafeAssumptions(input []ast.ASTHandle) (_ *safeAssumptions, resultErr error) {
	atoms, err := p.manager.FlattenAnd(append([]ast.ASTHandle(nil), input...))
	if err != nil {
		return nil, err
	}
	atoms, references, generated, err := p.expandArithmeticEqualities(atoms)
	if err != nil {
		return nil, err
	}
	safe := &safeAssumptions{
		atoms:          atoms,
		proxyToFormula: make(map[ast.ASTHandle]ast.ASTHandle),
		formulaToProxy: make(map[ast.ASTHandle]ast.ASTHandle),
		references:     references,
		generated:      generated,
	}
	committed := false
	defer func() {
		if !committed {
			resultErr = errors.Join(resultErr, safe.close(p.manager))
		}
	}()
	nextProxy := uint32(0)
	for index, atom := range safe.atoms {
		direct, x := p.isDirectAssumption(atom)
		if x != nil {
			return nil, x
		}
		if direct {
			continue
		}
		if proxy, found := safe.formulaToProxy[atom]; found {
			safe.atoms[index] = proxy
			continue
		}
		proxy, x := p.proxy(nextProxy)
		if x != nil {
			return nil, x
		}
		nextProxy++
		implication, x := p.manager.MkImplies(proxy, atom)
		if x != nil {
			return nil, x
		}
		if x = p.context.Assert(implication); x != nil {
			return nil, x
		}
		safe.proxyToFormula[proxy] = atom
		safe.formulaToProxy[atom] = proxy
		safe.atoms[index] = proxy
	}
	committed = true
	return safe, nil
}

func (p *PropSolver) levelAssumptions(level uint32) []ast.ASTHandle {
	result := make([]ast.ASTHandle, len(p.levelPositive))
	for index := range result {
		if uint32(index) >= level {
			result[index] = p.levelNegative[index]
		} else {
			result[index] = p.levelPositive[index]
		}
	}
	return result
}

func (p *PropSolver) clearCoreReferences() (result error) {
	for index := len(p.coreReferences) - 1; index >= 0; index-- {
		result = errors.Join(result, p.manager.DelRef(&p.coreReferences[index]))
	}
	p.coreReferences = nil
	return result
}

func (p *PropSolver) check(atoms []ast.ASTHandle, formula ast.ASTHandle, level *uint32, wantCore, wantModel bool) (result PropCheckResult, resultErr error) {
	if err := p.open(); err != nil {
		return result, err
	}
	if err := p.clearCoreReferences(); err != nil {
		return result, err
	}
	if level != nil {
		if err := p.ensureLevel(*level); err != nil {
			return result, err
		}
	}
	scope, err := p.context.BeginDelay()
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, scope.Close()) }()
	safe, err := p.makeSafeAssumptions(atoms)
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, safe.close(p.manager)) }()
	truth, err := p.manager.MkTrue()
	if err != nil {
		return result, err
	}
	if formula != truth {
		if err = p.context.Assert(formula); err != nil {
			return result, err
		}
	}
	assumptions := append([]ast.ASTHandle(nil), safe.atoms...)
	if level != nil {
		assumptions = append(assumptions, p.levelAssumptions(*level)...)
	}
	result.Status, err = p.context.Check(assumptions)
	if err != nil {
		return result, err
	}
	if result.Status == lbool.False {
		core, x := p.context.UnsatCore()
		if x != nil {
			return result, x
		}
		for _, item := range core {
			if p.levelAtoms[item] {
				result.AssumesLevel = true
				continue
			}
			if original, proxied := safe.proxyToFormula[item]; proxied {
				if wantCore {
					if safe.generated[original] {
						reference, x := p.manager.NewRef(original)
						if x != nil {
							return result, x
						}
						p.coreReferences = append(p.coreReferences, reference)
					}
					result.Core = append(result.Core, original)
				}
				continue
			}
			declaration, x := p.manager.GetDecl(item)
			if x != nil {
				return result, x
			}
			auxiliary, x := p.IsAuxSymbol(declaration)
			if x != nil {
				return result, x
			}
			if wantCore && !auxiliary {
				result.Core = append(result.Core, item)
			}
		}
	} else if result.Status == lbool.True && wantModel {
		result.Model, err = p.context.Model()
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (p *PropSolver) CheckAssumptions(atoms []ast.ASTHandle, wantCore, wantModel bool) (PropCheckResult, error) {
	if err := p.open(); err != nil {
		return PropCheckResult{}, err
	}
	truth, err := p.manager.MkTrue()
	if err != nil {
		return PropCheckResult{}, err
	}
	return p.check(atoms, truth, nil, wantCore, wantModel)
}

func (p *PropSolver) CheckAtLevel(level uint32, atoms []ast.ASTHandle, wantCore, wantModel bool) (PropCheckResult, error) {
	if err := p.open(); err != nil {
		return PropCheckResult{}, err
	}
	truth, err := p.manager.MkTrue()
	if err != nil {
		return PropCheckResult{}, err
	}
	return p.check(atoms, truth, &level, wantCore, wantModel)
}

func (p *PropSolver) CheckAssumptionsAndFormula(atoms []ast.ASTHandle, formula ast.ASTHandle, wantCore, wantModel bool) (PropCheckResult, error) {
	return p.check(atoms, formula, nil, wantCore, wantModel)
}

func (p *PropSolver) Close() (result error) {
	if p == nil || p.manager == nil {
		return ErrPropSolverOwner
	}
	if p.closed {
		return nil
	}
	if p.context != nil {
		result = errors.Join(result, p.context.Close())
	}
	result = errors.Join(result, p.clearCoreReferences())
	for index := len(p.references) - 1; index >= 0; index-- {
		result = errors.Join(result, p.manager.DelRef(&p.references[index]))
	}
	if result == nil {
		p.context = nil
		p.manager = nil
		p.contexts = nil
		p.references = nil
		p.closed = true
	}
	return result
}
