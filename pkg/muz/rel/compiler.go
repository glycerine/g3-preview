package rel

// Source: src/muz/rel/dl_compiler.cpp retained load, register allocation,
// ground-head assembly, nonrecursive ordering, union, and store paths.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
)

var ErrCompilerUnsupported = errors.New("muz/rel: rule is outside the finite compiler fragment")

func onlyCompilerUnsupported(err error) bool {
	if err == nil {
		return false
	}
	if many, ok := err.(interface{ Unwrap() []error }); ok {
		children := many.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !onlyCompilerUnsupported(child) {
				return false
			}
		}
		return true
	}
	if one, ok := err.(interface{ Unwrap() error }); ok {
		return onlyCompilerUnsupported(one.Unwrap())
	}
	return err == ErrCompilerUnsupported
}

type finiteRuleKind uint8

const (
	finiteGroundRule finiteRuleKind = iota
	finiteIdentityRule
)

type finiteCompiledRule struct {
	rule *base.Rule
	kind finiteRuleKind
	head ast.ASTHandle
	tail ast.ASTHandle
}

type constantRegisterKey struct {
	sort  ast.ASTHandle
	value Element
}

type finiteCompiler struct {
	manager       *ast.ASTManager
	relations     *RelationManager
	execution     *InstructionBlock
	termination   *InstructionBlock
	predicateRegs map[ast.ASTHandle]uint32
	signatures    []Signature
	loaded        []ast.ASTHandle
	constants     map[constantRegisterKey]uint32
	emptyTotals   map[ast.ASTHandle]uint32
}

func newFiniteCompiler(relations *RelationManager) (*finiteCompiler, error) {
	if err := relations.open(); err != nil {
		return nil, err
	}
	return &finiteCompiler{
		manager: relations.manager, relations: relations,
		execution: NewInstructionBlock(), termination: NewInstructionBlock(),
		predicateRegs: make(map[ast.ASTHandle]uint32),
		constants:     make(map[constantRegisterKey]uint32),
		emptyTotals:   make(map[ast.ASTHandle]uint32),
	}, nil
}

func (c *finiteCompiler) closeBlocks() error {
	return errors.Join(c.execution.Close(), c.termination.Close())
}

func (c *finiteCompiler) fresh(signature Signature) uint32 {
	register := uint32(len(c.signatures))
	c.signatures = append(c.signatures, Signature{domains: signature.Domains()})
	return register
}

func (c *finiteCompiler) append(block *InstructionBlock, instruction Instruction, rule *base.Rule) error {
	if instruction == nil {
		return ErrInstruction
	}
	if rule != nil {
		if err := instruction.Accounting().SetAccountingParentRule(rule); err != nil {
			return errors.Join(err, instruction.Close())
		}
	}
	if err := block.Append(instruction); err != nil {
		return errors.Join(err, instruction.Close())
	}
	return nil
}

func (c *finiteCompiler) ensureLoaded(predicate ast.ASTHandle) error {
	if _, found := c.predicateRegs[predicate]; found {
		return nil
	}
	_, signature, err := c.relations.PredicateSignature(predicate)
	if err != nil {
		return err
	}
	register := c.fresh(signature)
	instruction, err := NewLoadInstruction(c.relations, predicate, register)
	if err != nil {
		return err
	}
	if err = c.append(c.execution, instruction, nil); err != nil {
		return err
	}
	c.predicateRegs[predicate] = register
	c.loaded = append(c.loaded, predicate)
	return nil
}

func (c *finiteCompiler) classify(rule *base.Rule) (finiteCompiledRule, error) {
	result := finiteCompiledRule{rule: rule}
	var err error
	result.head, err = rule.Head()
	if err != nil {
		return result, err
	}
	positive, err := rule.PositiveTailSize()
	if err != nil {
		return result, err
	}
	uninterpreted, err := rule.UninterpretedTailSize()
	if err != nil {
		return result, err
	}
	tails, err := rule.TailSize()
	if err != nil {
		return result, err
	}
	if positive == 0 && uninterpreted == 0 && tails == 0 {
		count, x := c.manager.GetNumArgs(result.head)
		if x != nil {
			return result, x
		}
		for index := uint32(0); index < count; index++ {
			argument, y := c.manager.GetArg(result.head, index)
			if y != nil {
				return result, y
			}
			value, y := c.manager.IsValue(argument)
			if y != nil {
				return result, y
			}
			if !value {
				return result, ErrCompilerUnsupported
			}
		}
		result.kind = finiteGroundRule
		return result, nil
	}
	if positive != 1 || uninterpreted != 1 || tails != 1 {
		return result, ErrCompilerUnsupported
	}
	result.tail, _, err = rule.Tail(0)
	if err != nil {
		return result, err
	}
	headCount, err := c.manager.GetNumArgs(result.head)
	if err != nil {
		return result, err
	}
	tailCount, err := c.manager.GetNumArgs(result.tail)
	if err != nil || headCount != tailCount {
		return result, errors.Join(ErrCompilerUnsupported, err)
	}
	for index := uint32(0); index < headCount; index++ {
		headArgument, x := c.manager.GetArg(result.head, index)
		if x != nil {
			return result, x
		}
		tailArgument, x := c.manager.GetArg(result.tail, index)
		if x != nil {
			return result, x
		}
		headKind, x := c.manager.GetKind(headArgument)
		if x != nil {
			return result, x
		}
		tailKind, x := c.manager.GetKind(tailArgument)
		if x != nil {
			return result, x
		}
		if headKind != ast.ASTVar || tailKind != ast.ASTVar {
			return result, ErrCompilerUnsupported
		}
		headVariable, x := c.manager.GetVarIndex(headArgument)
		if x != nil {
			return result, x
		}
		tailVariable, x := c.manager.GetVarIndex(tailArgument)
		if x != nil {
			return result, x
		}
		if headVariable != tailVariable {
			return result, ErrCompilerUnsupported
		}
	}
	result.kind = finiteIdentityRule
	return result, nil
}

func closeCompiledRules(rules []finiteCompiledRule) (result error) {
	for _, rule := range rules {
		result = errors.Join(result, rule.rule.Close())
	}
	return result
}

func (c *finiteCompiler) readRules(set *base.RuleSet) ([]finiteCompiledRule, error) {
	count, err := set.Len()
	if err != nil {
		return nil, err
	}
	rules := make([]finiteCompiledRule, 0, count)
	for index := 0; index < count; index++ {
		rule, x := set.Rule(index)
		if x != nil {
			return nil, errors.Join(x, closeCompiledRules(rules))
		}
		compiled, x := c.classify(rule)
		if x != nil {
			return nil, errors.Join(x, rule.Close(), closeCompiledRules(rules))
		}
		rules = append(rules, compiled)
	}
	return rules, nil
}

func (c *finiteCompiler) orderRules(rules []finiteCompiledRule) ([]finiteCompiledRule, error) {
	byHead := make(map[ast.ASTHandle][]finiteCompiledRule)
	headOrder := make([]ast.ASTHandle, 0)
	seenHead := make(map[ast.ASTHandle]bool)
	dependencies := make(map[ast.ASTHandle][]ast.ASTHandle)
	for _, rule := range rules {
		head, err := c.manager.GetDecl(rule.head)
		if err != nil {
			return nil, err
		}
		if !seenHead[head] {
			seenHead[head] = true
			headOrder = append(headOrder, head)
		}
		byHead[head] = append(byHead[head], rule)
		if rule.kind == finiteIdentityRule {
			tail, x := c.manager.GetDecl(rule.tail)
			if x != nil {
				return nil, x
			}
			dependencies[head] = append(dependencies[head], tail)
		}
	}
	state := make(map[ast.ASTHandle]uint8)
	orderedHeads := make([]ast.ASTHandle, 0, len(headOrder))
	var visit func(ast.ASTHandle) error
	visit = func(head ast.ASTHandle) error {
		if state[head] == 1 {
			return ErrCompilerUnsupported
		}
		if state[head] == 2 {
			return nil
		}
		state[head] = 1
		for _, dependency := range dependencies[head] {
			if seenHead[dependency] {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}
		state[head] = 2
		orderedHeads = append(orderedHeads, head)
		return nil
	}
	for _, head := range headOrder {
		if err := visit(head); err != nil {
			return nil, err
		}
	}
	ordered := make([]finiteCompiledRule, 0, len(rules))
	for _, head := range orderedHeads {
		ordered = append(ordered, byHead[head]...)
	}
	return ordered, nil
}

func (c *finiteCompiler) constantRegister(rule *base.Rule, sort ast.ASTHandle, domain uint64, value Element) (uint32, error) {
	key := constantRegisterKey{sort: sort, value: value}
	if register, found := c.constants[key]; found {
		return register, nil
	}
	register := c.fresh(Signature{domains: []uint64{domain}})
	instruction, err := NewUnarySingletonInstruction(domain, value, register)
	if err != nil {
		return 0, err
	}
	if err = c.append(c.execution, instruction, rule); err != nil {
		return 0, err
	}
	c.constants[key] = register
	return register, nil
}

func (c *finiteCompiler) groundHead(rule finiteCompiledRule, predicate ast.ASTHandle) (uint32, bool, error) {
	sorts, signature, err := c.relations.PredicateSignature(predicate)
	if err != nil {
		return 0, false, err
	}
	count := signature.Len()
	if count == 0 {
		if register, found := c.emptyTotals[predicate]; found {
			return register, false, nil
		}
		register := c.fresh(signature)
		if err = c.append(c.execution, NewTotalInstruction(signature, register), rule.rule); err != nil {
			return 0, false, err
		}
		c.emptyTotals[predicate] = register
		return register, false, nil
	}
	var current uint32
	owned := false
	currentSignature, _ := NewSignature(nil)
	for index := 0; index < count; index++ {
		argument, x := c.manager.GetArg(rule.head, uint32(index))
		if x != nil {
			return 0, false, x
		}
		value, x := c.relations.EncodeValue(sorts[index], argument)
		if x != nil {
			return 0, false, x
		}
		domain, x := signature.Domain(index)
		if x != nil {
			return 0, false, x
		}
		singleton, x := c.constantRegister(rule.rule, sorts[index], domain, value)
		if x != nil {
			return 0, false, x
		}
		if index == 0 {
			current = singleton
			currentSignature = Signature{domains: []uint64{domain}}
			continue
		}
		domains := append(currentSignature.Domains(), domain)
		joined, x := NewSignature(domains)
		if x != nil {
			return 0, false, x
		}
		result := current
		if owned {
			c.signatures[result] = joined
		} else {
			result = c.fresh(joined)
		}
		if x = c.append(c.execution, NewJoinInstruction(current, singleton, nil, nil, result), rule.rule); x != nil {
			return 0, false, x
		}
		current, currentSignature, owned = result, joined, true
	}
	return current, owned, nil
}

func (c *finiteCompiler) compileRule(rule finiteCompiledRule) error {
	predicate, err := c.manager.GetDecl(rule.head)
	if err != nil {
		return err
	}
	target := c.predicateRegs[predicate]
	var source uint32
	owned := false
	switch rule.kind {
	case finiteGroundRule:
		source, owned, err = c.groundHead(rule, predicate)
	case finiteIdentityRule:
		var tail ast.ASTHandle
		tail, err = c.manager.GetDecl(rule.tail)
		source = c.predicateRegs[tail]
	default:
		err = ErrCompilerUnsupported
	}
	if err != nil {
		return err
	}
	if !c.signatures[source].equal(c.signatures[target]) {
		return fmt.Errorf("%w: compiled rule signatures differ", ErrCompilerUnsupported)
	}
	if err = c.append(c.execution, NewUnionInstruction(source, target, VoidRegister), rule.rule); err != nil {
		return err
	}
	if owned {
		return c.append(c.execution, NewDeallocateInstruction(source), rule.rule)
	}
	return nil
}

func (c *finiteCompiler) compile(set *base.RuleSet) (resultErr error) {
	rules, err := c.readRules(set)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, closeCompiledRules(rules)) }()
	ordered, err := c.orderRules(rules)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		head, x := c.manager.GetDecl(rule.head)
		if x != nil {
			return x
		}
		if x = c.ensureLoaded(head); x != nil {
			return x
		}
		if rule.kind == finiteIdentityRule {
			tail, y := c.manager.GetDecl(rule.tail)
			if y != nil {
				return y
			}
			if y = c.ensureLoaded(tail); y != nil {
				return y
			}
		}
	}
	for _, rule := range ordered {
		if err = c.compileRule(rule); err != nil {
			return err
		}
	}
	for _, predicate := range c.loaded {
		instruction, x := NewStoreInstruction(c.relations, predicate, c.predicateRegs[predicate])
		if x != nil {
			return x
		}
		if x = c.append(c.termination, instruction, nil); x != nil {
			return x
		}
	}
	return nil
}

// CompileFiniteRules compiles the currently retained nonrecursive finite
// ground and identity-rule slice into execution and termination bytecode.
// Ownership of both returned blocks transfers to the caller.
func CompileFiniteRules(relations *RelationManager, set *base.RuleSet) (*InstructionBlock, *InstructionBlock, error) {
	compiler, err := newFiniteCompiler(relations)
	if err != nil {
		return nil, nil, err
	}
	if set == nil {
		if closeErr := compiler.closeBlocks(); closeErr != nil {
			return nil, nil, closeErr
		}
		return nil, nil, ErrCompilerUnsupported
	}
	if err = compiler.compile(set); err != nil {
		closeErr := compiler.closeBlocks()
		if onlyCompilerUnsupported(err) && closeErr != nil {
			return nil, nil, closeErr
		}
		return nil, nil, errors.Join(err, closeErr)
	}
	return compiler.execution, compiler.termination, nil
}
