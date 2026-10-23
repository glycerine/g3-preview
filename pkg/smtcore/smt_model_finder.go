// Source: src/smt/smt_model_finder.h; src/smt/smt_model_finder.cpp model_finder
// registration, scope, search and candidate-model lifecycle.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	"github.com/glycerine/g3/pkg/ast/pattern"
	"github.com/glycerine/g3/pkg/model"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// SMTModelFinder owns the quantified-formula analysis used to extend a ground
// candidate model before MBQI checks it. The macro and AUF algorithms attach
// their per-quantifier state to this lifecycle.
type SMTModelFinder struct {
	manager            *ast.ASTManager
	state              *SMTInternalizerState
	quantifiers        []ast.ASTHandle
	registered         map[ast.ASTHandle]struct{}
	sets               map[ast.ASTHandle][]*smtModelFinderInstantiationSet
	flat               map[ast.ASTHandle]smtModelFinderFlatQuantifier
	macroUtil          *macros.Util
	candidates         map[ast.ASTHandle]*macros.Candidates
	nonGround          map[ast.ASTHandle]map[ast.ASTHandle]struct{}
	auf                map[ast.ASTHandle]bool
	chosen             map[ast.ASTHandle]ast.ASTHandle
	hint               *hintSearchState
	dependencies       *ast.FuncDeclDependencies
	forceTemplate      uint32
	pendingConstraints []ast.ASTRef
	scopes             []int
	closed             bool
}

type smtModelFinderFlatQuantifier struct {
	quantifier ast.ASTHandle
	reference  ast.ASTRef
}

func (f *SMTModelFinder) flatQuantifier(quantifier ast.ASTHandle) ast.ASTHandle {
	if flat, found := f.flat[quantifier]; found {
		return flat.quantifier
	}
	return quantifier
}

func NewSMTModelFinder(state *SMTInternalizerState) (*SMTModelFinder, error) {
	if state == nil || state.manager == nil || state.context == nil {
		return nil, ast.ErrASTHandle
	}
	util, err := macros.NewUtil(state.manager)
	if err != nil {
		return nil, err
	}
	dependencies, err := ast.NewFuncDeclDependencies(state.manager)
	if err != nil {
		return nil, err
	}
	return &SMTModelFinder{
		manager: state.manager, state: state, registered: make(map[ast.ASTHandle]struct{}),
		sets: make(map[ast.ASTHandle][]*smtModelFinderInstantiationSet), macroUtil: util,
		flat:       make(map[ast.ASTHandle]smtModelFinderFlatQuantifier),
		candidates: make(map[ast.ASTHandle]*macros.Candidates), nonGround: make(map[ast.ASTHandle]map[ast.ASTHandle]struct{}),
		auf: make(map[ast.ASTHandle]bool), chosen: make(map[ast.ASTHandle]ast.ASTHandle), hint: newHintSearchState(), dependencies: dependencies,
	}, nil
}

func (f *SMTModelFinder) open() error {
	if f == nil || f.manager == nil || f.state == nil || f.registered == nil || f.sets == nil || f.macroUtil == nil || f.candidates == nil || f.auf == nil || f.hint == nil || f.dependencies == nil || f.closed {
		return ast.ErrASTHandle
	}
	return nil
}

// checkpoint corresponds to model_finder::checkpoint: model-finder work must
// honor the owning manager's cancellation and resource limit.
func (f *SMTModelFinder) checkpoint() error {
	canceled, err := f.manager.Canceled()
	if err != nil {
		return err
	}
	if canceled {
		return errors.New(f.manager.ResourceLimit().CancelMessage())
	}
	return nil
}

func (f *SMTModelFinder) SetForceTemplate(value uint32) {
	if f != nil && !f.closed {
		f.forceTemplate = value
	}
}

func (f *SMTModelFinder) RegisterQuantifier(quantifier ast.ASTHandle) (result error) {
	if err := f.open(); err != nil {
		return err
	}
	if err := f.checkpoint(); err != nil {
		return err
	}
	kind, err := f.manager.GetKind(quantifier)
	if err != nil {
		return err
	}
	if kind != ast.ASTQuantifier {
		return ast.ErrASTDomain
	}
	if _, found := f.registered[quantifier]; found {
		return ast.ErrASTDomain
	}
	f.registered[quantifier] = struct{}{}
	f.quantifiers = append(f.quantifiers, quantifier)
	defer func() {
		if result != nil {
			result = errors.Join(result, f.releaseQuantifier(quantifier))
			f.quantifiers = f.quantifiers[:len(f.quantifiers)-1]
		}
	}()
	count, err := f.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return err
	}
	f.sets[quantifier] = make([]*smtModelFinderInstantiationSet, count)
	candidates, err := macros.NewCandidates(f.manager)
	if err != nil {
		return err
	}
	f.candidates[quantifier] = candidates
	f.nonGround[quantifier] = make(map[ast.ASTHandle]struct{})
	body, err := f.manager.GetQuantifierBody(quantifier)
	if err != nil {
		return err
	}
	nested, err := f.manager.HasQuantifiers(body)
	if err != nil {
		return err
	}
	var flatRef ast.ASTRef
	if nested {
		flatRef, err = pattern.PullQuantifierNoProof(f.manager, quantifier)
	} else {
		flatRef, err = f.manager.NewRef(quantifier)
	}
	if err != nil {
		return err
	}
	flat, err := f.manager.RefGet(flatRef)
	if err != nil {
		return errors.Join(err, f.manager.DelRef(&flatRef))
	}
	f.flat[quantifier] = smtModelFinderFlatQuantifier{quantifier: flat, reference: flatRef}
	body, err = f.manager.GetQuantifierBody(flat)
	if err != nil {
		return err
	}
	nested, err = f.manager.HasQuantifiers(body)
	if err != nil {
		return err
	}
	if nested {
		return ast.ErrASTPending
	}
	bodyKind, err := f.manager.GetKind(body)
	if err != nil {
		return err
	}
	if bodyKind == ast.ASTApp {
		if err = f.macroUtil.CollectQuantifierCandidates(flat, candidates); err != nil {
			return err
		}
	}
	if err = f.collectNonGroundDeclarations(body, f.nonGround[quantifier]); err != nil {
		return err
	}
	f.auf[quantifier], err = f.classifyAUF(quantifier)
	if err != nil {
		return err
	}
	info := &hintQuantifierInfo{nonGround: make([]ast.ASTHandle, 0, len(f.nonGround[quantifier]))}
	for declaration := range f.nonGround[quantifier] {
		info.nonGround = append(info.nonGround, declaration)
	}
	for i := 0; i < candidates.Len(); i++ {
		candidate, err := candidates.At(i)
		if err != nil {
			return err
		}
		info.macros = append(info.macros, hintConditionalMacro{
			function: candidate.Declaration, definition: candidate.Definition, satisfyAtom: candidate.SatisfyAtom,
		})
	}
	f.hint.infos[quantifier] = info
	return nil
}

func (f *SMTModelFinder) collectNonGroundDeclarations(root ast.ASTHandle, result map[ast.ASTHandle]struct{}) error {
	stack := []ast.ASTHandle{root}
	seen := make(map[ast.ASTHandle]struct{})
	for len(stack) != 0 {
		if err := f.checkpoint(); err != nil {
			return err
		}
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, found := seen[n]; found {
			continue
		}
		seen[n] = struct{}{}
		kind, err := f.manager.GetKind(n)
		if err != nil {
			return err
		}
		if kind != ast.ASTApp {
			continue
		}
		ground, err := f.manager.IsGround(n)
		if err != nil {
			return err
		}
		declaration, err := f.manager.GetDecl(n)
		if err != nil {
			return err
		}
		family, err := f.manager.GetFamilyID(declaration)
		if err != nil {
			return err
		}
		if !ground && family != f.manager.BasicFamilyID() {
			result[declaration] = struct{}{}
		}
		count, err := f.manager.GetNumArgs(n)
		if err != nil {
			return err
		}
		for i := uint32(0); i < count; i++ {
			child, err := f.manager.GetArg(n, i)
			if err != nil {
				return err
			}
			stack = append(stack, child)
		}
	}
	return nil
}

func (f *SMTModelFinder) instantiationSet(quantifier ast.ASTHandle, variable uint32, create bool) (*smtModelFinderInstantiationSet, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	sets, found := f.sets[quantifier]
	if !found || variable >= uint32(len(sets)) {
		return nil, ast.ErrASTDomain
	}
	if sets[variable] == nil && create {
		value, err := newSMTModelFinderInstantiationSet(f.manager)
		if err != nil {
			return nil, err
		}
		sets[variable] = value
	}
	return sets[variable], nil
}

func (f *SMTModelFinder) InsertInstantiationTerm(quantifier ast.ASTHandle, variable uint32, term ast.ASTHandle, generation uint32) (bool, error) {
	set, err := f.instantiationSet(quantifier, variable, true)
	if err != nil {
		return false, err
	}
	return set.Insert(term, generation)
}

func (f *SMTModelFinder) Inverse(quantifier ast.ASTHandle, variable uint32, value ast.ASTHandle) (term ast.ASTHandle, generation uint32, found bool, err error) {
	set, err := f.instantiationSet(quantifier, variable, false)
	if err != nil || set == nil {
		return term, 0, false, err
	}
	return set.Inverse(value)
}

func (f *SMTModelFinder) InstantiationValues(quantifier ast.ASTHandle, variable uint32) ([]ast.ASTHandle, error) {
	set, err := f.instantiationSet(quantifier, variable, false)
	if err != nil || set == nil {
		return nil, err
	}
	return set.Values()
}

func (f *SMTModelFinder) releaseQuantifier(quantifier ast.ASTHandle) (result error) {
	for _, set := range f.sets[quantifier] {
		if set != nil {
			result = errors.Join(result, set.Close())
		}
	}
	if candidates := f.candidates[quantifier]; candidates != nil {
		result = errors.Join(result, candidates.Close())
	}
	if flat, found := f.flat[quantifier]; found {
		if err := f.manager.DelRef(&flat.reference); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(f.flat, quantifier)
		}
	}
	if result == nil {
		delete(f.sets, quantifier)
		delete(f.candidates, quantifier)
		delete(f.nonGround, quantifier)
		delete(f.auf, quantifier)
		delete(f.chosen, quantifier)
		delete(f.hint.infos, quantifier)
		delete(f.registered, quantifier)
	}
	return result
}

func (f *SMTModelFinder) macroOccursElsewhere(declaration, except ast.ASTHandle, quantifiers []ast.ASTHandle) bool {
	for _, quantifier := range quantifiers {
		if quantifier == except {
			continue
		}
		if _, found := f.nonGround[quantifier][declaration]; found {
			return true
		}
	}
	return false
}

func (f *SMTModelFinder) publishFunctionDefinition(declaration, definition ast.ASTHandle, proto *protomodel.ProtoModel) error {
	interpretation, err := proto.Core().FunctionInterpretation(declaration)
	if err != nil {
		return err
	}
	if interpretation == nil {
		arity, err := f.manager.GetArity(declaration)
		if err != nil {
			return err
		}
		interpretation, err = model.NewFunctionInterpretation(f.manager, arity)
		if err != nil {
			return err
		}
		if err = proto.Core().RegisterFunction(declaration, interpretation); err != nil {
			return errors.Join(err, interpretation.Close())
		}
	}
	return interpretation.SetElse(definition)
}

func (f *SMTModelFinder) processSimpleMacros(proto *protomodel.ProtoModel, quantifiers []ast.ASTHandle) ([]ast.ASTHandle, error) {
	current := append([]ast.ASTHandle(nil), quantifiers...)
	for {
		removed := false
		next := make([]ast.ASTHandle, 0, len(current))
		for _, quantifier := range current {
			selected := false
			candidates := f.candidates[quantifier]
			for i := 0; candidates != nil && i < candidates.Len(); i++ {
				candidate, err := candidates.At(i)
				if err != nil {
					return nil, err
				}
				if !candidate.SatisfyAtom || f.macroOccursElsewhere(candidate.Declaration, quantifier, current) {
					continue
				}
				if err = f.publishFunctionDefinition(candidate.Declaration, candidate.Definition, proto); err != nil {
					return nil, err
				}
				f.chosen[quantifier] = candidate.Declaration
				selected, removed = true, true
				break
			}
			if !selected {
				next = append(next, quantifier)
			}
		}
		if !removed {
			return next, nil
		}
		current = next
	}
}

func (f *SMTModelFinder) processHintMacros(proto *protomodel.ProtoModel, quantifiers []ast.ASTHandle) ([]ast.ASTHandle, error) {
	for {
		plan, found, err := f.hint.planSubset(f.manager, quantifiers)
		if err != nil || !found {
			return quantifiers, err
		}
		for declaration, definition := range plan.definitions {
			if err = f.publishFunctionDefinition(declaration, definition, proto); err != nil {
				return nil, err
			}
		}
		for _, quantifier := range quantifiers {
			if info := f.hint.infos[quantifier]; info != nil && !info.theOne.IsNull() {
				f.chosen[quantifier] = info.theOne
			}
		}
		if len(plan.remaining) >= len(quantifiers) {
			return nil, ast.ErrASTDomain
		}
		quantifiers = plan.remaining
	}
}

type smtModelFinderFullMacro struct {
	quantifier ast.ASTHandle
	candidate  macros.Candidate
}

func (f *SMTModelFinder) addNonAUFMacro(proto *protomodel.ProtoModel, candidate macros.Candidate) (bool, error) {
	dependencies := make(ast.FuncDeclSet)
	if err := f.dependencies.CollectNonGround(candidate.Definition, dependencies); err != nil {
		return false, err
	}
	inserted, err := f.dependencies.Insert(candidate.Declaration, dependencies)
	if err != nil || !inserted {
		return false, err
	}
	if err = f.publishFunctionDefinition(candidate.Declaration, candidate.Definition, proto); err != nil {
		return false, errors.Join(err, f.dependencies.Erase(candidate.Declaration))
	}
	return true, nil
}

// processNonAUFMacros ports non_auf_macro_solver. Full macros are available to
// non-AUF quantifiers and to every quantifier forced by mbqi.force_template;
// remaining conditional definitions are combined into an ordered ITE else.
func (f *SMTModelFinder) processNonAUFMacros(proto *protomodel.ProtoModel, quantifiers []ast.ASTHandle) ([]ast.ASTHandle, error) {
	current := append([]ast.ASTHandle(nil), quantifiers...)
	for {
		byDeclaration := make(map[ast.ASTHandle]smtModelFinderFullMacro)
		fullOrder := make([]ast.ASTHandle, 0)
		conditional := make(map[ast.ASTHandle]struct{})
		conditionalOrder := make([]ast.ASTHandle, 0)
		for _, quantifier := range current {
			weight, err := f.manager.GetQuantifierWeight(f.flatQuantifier(quantifier))
			if err != nil {
				return nil, err
			}
			forced := weight >= 0 && uint32(weight) >= f.forceTemplate
			candidates := f.candidates[quantifier]
			for i := 0; candidates != nil && i < candidates.Len(); i++ {
				candidate, err := candidates.At(i)
				if err != nil {
					return nil, err
				}
				if candidate.Hint {
					continue
				}
				if f.manager.IsTrue(candidate.Condition) && (!f.auf[quantifier] || forced) {
					if _, found := byDeclaration[candidate.Declaration]; !found {
						fullOrder = append(fullOrder, candidate.Declaration)
					}
					byDeclaration[candidate.Declaration] = smtModelFinderFullMacro{quantifier: quantifier, candidate: candidate}
					delete(conditional, candidate.Declaration)
					continue
				}
				if _, full := byDeclaration[candidate.Declaration]; !full && !f.auf[quantifier] {
					if _, found := conditional[candidate.Declaration]; !found {
						conditionalOrder = append(conditionalOrder, candidate.Declaration)
					}
					conditional[candidate.Declaration] = struct{}{}
				}
			}
		}
		removed := make(map[ast.ASTHandle]struct{})
		for _, declaration := range fullOrder {
			macro := byDeclaration[declaration]
			added, err := f.addNonAUFMacro(proto, macro.candidate)
			if err != nil {
				return nil, err
			}
			if added {
				f.chosen[macro.quantifier] = declaration
				removed[macro.quantifier] = struct{}{}
			}
		}
		for _, declaration := range conditionalOrder {
			if _, stillConditional := conditional[declaration]; !stillConditional {
				continue
			}
			var definition ast.ASTHandle
			selected := make([]ast.ASTHandle, 0)
			ignore := false
			for _, quantifier := range current {
				if _, alreadyRemoved := removed[quantifier]; alreadyRemoved {
					continue
				}
				candidates := f.candidates[quantifier]
				var candidate macros.Candidate
				found := false
				for i := 0; candidates != nil && i < candidates.Len(); i++ {
					row, err := candidates.At(i)
					if err != nil {
						return nil, err
					}
					if row.Declaration == declaration && !row.Hint {
						candidate, found = row, true
					}
				}
				if !found {
					continue
				}
				if f.manager.IsTrue(candidate.Condition) {
					ignore = true
					break
				}
				selected = append(selected, quantifier)
				if definition.IsNull() {
					definition = candidate.Definition
				} else {
					next, err := f.manager.MkIte(candidate.Condition, candidate.Definition, definition)
					if err != nil {
						return nil, err
					}
					definition = next
				}
			}
			if ignore || definition.IsNull() {
				continue
			}
			added, err := f.addNonAUFMacro(proto, macros.Candidate{Declaration: declaration, Definition: definition})
			if err != nil {
				return nil, err
			}
			if added {
				for _, quantifier := range selected {
					f.chosen[quantifier] = declaration
					removed[quantifier] = struct{}{}
				}
			}
		}
		if len(removed) == 0 {
			return current, nil
		}
		next := make([]ast.ASTHandle, 0, len(current)-len(removed))
		for _, quantifier := range current {
			if _, found := removed[quantifier]; !found {
				next = append(next, quantifier)
			}
		}
		current = next
	}
}

func (f *SMTModelFinder) PushScope() error {
	if err := f.open(); err != nil {
		return err
	}
	f.scopes = append(f.scopes, len(f.quantifiers))
	return nil
}

func (f *SMTModelFinder) PopScopes(count uint32) error {
	if err := f.open(); err != nil {
		return err
	}
	if count == 0 || count > uint32(len(f.scopes)) {
		return ast.ErrASTDomain
	}
	level := len(f.scopes) - int(count)
	limit := f.scopes[level]
	for _, quantifier := range f.quantifiers[limit:] {
		if err := f.releaseQuantifier(quantifier); err != nil {
			return err
		}
	}
	f.quantifiers = f.quantifiers[:limit]
	f.scopes = f.scopes[:level]
	return nil
}

func (f *SMTModelFinder) InitSearch() error { return f.open() }

func (f *SMTModelFinder) relevantQuantifiers() ([]ast.ASTHandle, error) {
	if err := f.open(); err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, 0, len(f.quantifiers))
	for _, quantifier := range f.quantifiers {
		if f.state.context.relevant[quantifier] && f.state.context.assignments[quantifier] == lbool.True {
			result = append(result, quantifier)
		}
	}
	return result, nil
}

func (f *SMTModelFinder) collectPatternTerm(quantifier, term ast.ASTHandle, declarations uint32) error {
	kind, err := f.manager.GetKind(term)
	if err != nil {
		return err
	}
	if kind != ast.ASTApp {
		return nil
	}
	declaration, err := f.manager.GetDecl(term)
	if err != nil {
		return err
	}
	count, err := f.manager.GetNumArgs(term)
	if err != nil {
		return err
	}
	ground := f.state.context.byDeclaration[declaration]
	for position := uint32(0); position < count; position++ {
		argument, err := f.manager.GetArg(term, position)
		if err != nil {
			return err
		}
		argumentKind, err := f.manager.GetKind(argument)
		if err != nil {
			return err
		}
		if argumentKind == ast.ASTVar {
			index, err := f.manager.GetVarIndex(argument)
			if err != nil {
				return err
			}
			if index < declarations {
				variable := declarations - index - 1
				for _, application := range ground {
					if application == nil || !f.state.context.relevant[application.owner] || position >= uint32(len(application.arguments)) {
						continue
					}
					binding := application.arguments[position]
					if _, err = f.InsertInstantiationTerm(quantifier, variable, binding.owner, binding.Generation()); err != nil {
						return err
					}
				}
			}
		}
		if err = f.collectPatternTerm(quantifier, argument, declarations); err != nil {
			return err
		}
	}
	return nil
}

func (f *SMTModelFinder) collectPatternInstantiationTerms(quantifier ast.ASTHandle) error {
	declarations, err := f.manager.GetNumDeclarations(quantifier)
	if err != nil {
		return err
	}
	patterns, err := f.manager.GetNumPatterns(quantifier)
	if err != nil {
		return err
	}
	for i := uint32(0); i < patterns; i++ {
		pattern, err := f.manager.GetPattern(quantifier, i)
		if err != nil {
			return err
		}
		if err = f.collectPatternTerm(quantifier, pattern, declarations); err != nil {
			return err
		}
	}
	return nil
}

// FixModel is the production entry used during candidate-model construction.
// Quantifier-specific macro and AUF passes progressively extend this method.
func (f *SMTModelFinder) FixModel(model *protomodel.ProtoModel) error {
	if err := f.open(); err != nil {
		return err
	}
	if model == nil || model.Core() == nil {
		return ast.ErrASTHandle
	}
	if err := f.checkpoint(); err != nil {
		return err
	}
	relevant, err := f.relevantQuantifiers()
	if err != nil {
		return err
	}
	allRelevant := append([]ast.ASTHandle(nil), relevant...)
	// The C++ solver resets AUF nodes and macro-derived sets for each
	// candidate model. Term copies and inverses must not survive that reset.
	for _, sets := range f.sets {
		for i, set := range sets {
			if set != nil {
				if err := set.Close(); err != nil {
					return err
				}
				sets[i] = nil
			}
		}
	}
	for _, quantifier := range relevant {
		delete(f.chosen, quantifier)
		if info := f.hint.infos[quantifier]; info != nil {
			info.theOne = ast.ASTHandle{}
		}
	}
	if err = f.dependencies.Reset(); err != nil {
		return err
	}
	relevant, err = f.processSimpleMacros(model, relevant)
	if err != nil {
		return err
	}
	relevant, err = f.processHintMacros(model, relevant)
	if err != nil {
		return err
	}
	relevant, err = f.processNonAUFMacros(model, relevant)
	if err != nil {
		return err
	}
	if err = f.populateAUFInstantiationSets(relevant, model); err != nil {
		return err
	}
	for _, quantifier := range relevant {
		if err = f.collectPatternInstantiationTerms(quantifier); err != nil {
			return err
		}
	}
	for _, quantifier := range allRelevant {
		if err = f.populateMacroInstantiationSets(quantifier); err != nil {
			return err
		}
		for _, set := range f.sets[quantifier] {
			if set != nil {
				if err = set.MakeInverse(model); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *SMTModelFinder) Close() error {
	if f == nil {
		return ast.ErrASTHandle
	}
	if f.closed {
		return nil
	}
	if f.manager == nil {
		return ast.ErrASTHandle
	}
	var result error
	for _, quantifier := range f.quantifiers {
		result = errors.Join(result, f.releaseQuantifier(quantifier))
	}
	if result != nil {
		return result
	}
	if err := f.dependencies.Close(); err != nil {
		return err
	}
	for i := range f.pendingConstraints {
		if err := f.manager.DelRef(&f.pendingConstraints[i]); err != nil {
			return err
		}
	}
	f.pendingConstraints = nil
	f.closed = true
	f.quantifiers = nil
	f.scopes = nil
	f.registered = nil
	f.sets = nil
	f.flat = nil
	f.macroUtil = nil
	f.candidates = nil
	f.nonGround = nil
	f.auf = nil
	f.chosen = nil
	f.hint = nil
	f.dependencies = nil
	f.state = nil
	f.manager = nil
	return result
}
