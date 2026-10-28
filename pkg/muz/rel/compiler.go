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
	finiteSingleTailRule
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

type totalRegisterKey struct {
	sort      ast.ASTHandle
	predicate ast.ASTHandle
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
	totals        map[totalRegisterKey]uint32
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
		totals:        make(map[totalRegisterKey]uint32),
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
	validArguments := func(application ast.ASTHandle) error {
		count, x := c.manager.GetNumArgs(application)
		if x != nil {
			return x
		}
		for index := uint32(0); index < count; index++ {
			argument, y := c.manager.GetArg(application, index)
			if y != nil {
				return y
			}
			kind, y := c.manager.GetKind(argument)
			if y != nil {
				return y
			}
			if kind == ast.ASTVar {
				continue
			}
			value, y := c.manager.IsValue(argument)
			if y != nil {
				return y
			}
			if !value {
				return ErrCompilerUnsupported
			}
		}
		return nil
	}
	if positive == 0 && uninterpreted == 0 && tails == 0 {
		if err = validArguments(result.head); err != nil {
			return result, err
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
	if err = validArguments(result.head); err != nil {
		return result, err
	}
	if err = validArguments(result.tail); err != nil {
		return result, err
	}
	result.kind = finiteSingleTailRule
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
		if rule.kind == finiteSingleTailRule {
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

func (c *finiteCompiler) totalRegister(rule *base.Rule, predicate, sort ast.ASTHandle, domain uint64) (uint32, error) {
	key := totalRegisterKey{sort: sort, predicate: predicate}
	if register, found := c.totals[key]; found {
		return register, nil
	}
	signature := Signature{domains: []uint64{domain}}
	register := c.fresh(signature)
	if err := c.append(c.execution, NewTotalInstruction(signature, register), rule); err != nil {
		return 0, err
	}
	c.totals[key] = register
	return register, nil
}

func (c *finiteCompiler) emptyTotalRegister(rule *base.Rule, predicate ast.ASTHandle) (uint32, error) {
	if register, found := c.emptyTotals[predicate]; found {
		return register, nil
	}
	signature := Signature{}
	register := c.fresh(signature)
	if err := c.append(c.execution, NewTotalInstruction(signature, register), rule); err != nil {
		return 0, err
	}
	c.emptyTotals[predicate] = register
	return register, nil
}

func (c *finiteCompiler) project(register uint32, signature Signature, removed []int, owned bool, rule *base.Rule) (uint32, Signature, bool, error) {
	projected, err := projectedSignature(signature, removed)
	if err != nil {
		return 0, Signature{}, false, err
	}
	result := register
	if owned {
		c.signatures[result] = projected
	} else {
		result = c.fresh(projected)
	}
	if err = c.append(c.execution, NewProjectionInstruction(register, removed, result), rule); err != nil {
		return 0, Signature{}, false, err
	}
	return result, projected, true, nil
}

func (c *finiteCompiler) appendColumn(current uint32, signature Signature, owned, present bool, columnRegister uint32, domain uint64, rule *base.Rule) (uint32, Signature, bool, bool, error) {
	if !present {
		return columnRegister, Signature{domains: []uint64{domain}}, false, true, nil
	}
	joined, err := NewSignature(append(signature.Domains(), domain))
	if err != nil {
		return 0, Signature{}, false, false, err
	}
	result := current
	if owned {
		c.signatures[result] = joined
	} else {
		result = c.fresh(joined)
	}
	if err = c.append(c.execution, NewJoinInstruction(current, columnRegister, nil, nil, result), rule); err != nil {
		return 0, Signature{}, false, false, err
	}
	return result, joined, true, true, nil
}

func (c *finiteCompiler) duplicateColumn(current uint32, signature Signature, column int, owned bool, rule *base.Rule) (uint32, Signature, bool, error) {
	if column < 0 || column >= signature.Len() {
		return 0, Signature{}, false, ErrCompilerUnsupported
	}
	single := current
	temporary := false
	if signature.Len() != 1 {
		removed := make([]int, 0, signature.Len()-1)
		for index := 0; index < signature.Len(); index++ {
			if index != column {
				removed = append(removed, index)
			}
		}
		one, err := projectedSignature(signature, removed)
		if err != nil {
			return 0, Signature{}, false, err
		}
		single = c.fresh(one)
		if err = c.append(c.execution, NewProjectionInstruction(current, removed, single), rule); err != nil {
			return 0, Signature{}, false, err
		}
		temporary = true
	}
	domain := signature.domains[column]
	joined, err := NewSignature(append(signature.Domains(), domain))
	if err != nil {
		return 0, Signature{}, false, err
	}
	result := current
	if owned {
		c.signatures[result] = joined
	} else {
		result = c.fresh(joined)
	}
	if err = c.append(c.execution, NewJoinInstruction(current, single, []int{column}, []int{0}, result), rule); err != nil {
		return 0, Signature{}, false, err
	}
	if temporary {
		if err = c.append(c.execution, NewDeallocateInstruction(single), rule); err != nil {
			return 0, Signature{}, false, err
		}
	}
	return result, joined, true, nil
}

func renamedSignature(signature Signature, cycle []int) (Signature, error) {
	if len(cycle) < 2 {
		return signature, nil
	}
	domains := signature.Domains()
	original := signature.Domains()
	seen := make(map[int]bool)
	for _, column := range cycle {
		if column < 0 || column >= len(domains) || seen[column] {
			return Signature{}, ErrCompilerUnsupported
		}
		seen[column] = true
	}
	for index := 1; index < len(cycle); index++ {
		domains[cycle[index-1]] = original[cycle[index]]
	}
	domains[cycle[len(cycle)-1]] = original[cycle[0]]
	return NewSignature(domains)
}

func (c *finiteCompiler) reorder(current uint32, signature Signature, order []int, owned bool, rule *base.Rule) (uint32, Signature, bool, error) {
	if len(order) != signature.Len() {
		return 0, Signature{}, false, ErrCompilerUnsupported
	}
	seen := make([]bool, len(order))
	for _, source := range order {
		if source < 0 || source >= len(order) || seen[source] {
			return 0, Signature{}, false, ErrCompilerUnsupported
		}
		seen[source] = true
	}
	visited := make([]bool, len(order))
	for start := range order {
		if visited[start] || order[start] == start {
			visited[start] = true
			continue
		}
		cycle := make([]int, 0, 2)
		for column := start; !visited[column]; column = order[column] {
			visited[column] = true
			cycle = append(cycle, column)
		}
		renamed, err := renamedSignature(signature, cycle)
		if err != nil {
			return 0, Signature{}, false, err
		}
		result := current
		if owned {
			c.signatures[result] = renamed
		} else {
			result = c.fresh(renamed)
		}
		if err = c.append(c.execution, NewRenameInstruction(current, cycle, result), rule); err != nil {
			return 0, Signature{}, false, err
		}
		current, signature, owned = result, renamed, true
	}
	return current, signature, owned, nil
}

func copyVariableColumns(source map[uint32][]int) map[uint32][]int {
	result := make(map[uint32][]int, len(source))
	for variable, columns := range source {
		result[variable] = append([]int(nil), columns...)
	}
	return result
}

func (c *finiteCompiler) assembleHead(rule finiteCompiledRule, predicate ast.ASTHandle, current uint32, signature Signature, owned, present bool, variableColumns map[uint32][]int) (uint32, bool, error) {
	sorts, target, err := c.relations.PredicateSignature(predicate)
	if err != nil {
		return 0, false, err
	}
	count := target.Len()
	if count == 0 && !present {
		register, x := c.emptyTotalRegister(rule.rule, predicate)
		return register, false, x
	}
	columns := make([]int, count)
	for index := range columns {
		columns[index] = -1
	}
	boundVariables := make([]uint32, count)
	isBound := make([]bool, count)
	available := copyVariableColumns(variableColumns)
	referenced := make([]bool, signature.Len())
	for index := 0; index < count; index++ {
		argument, x := c.manager.GetArg(rule.head, uint32(index))
		if x != nil {
			return 0, false, x
		}
		kind, x := c.manager.GetKind(argument)
		if x != nil {
			return 0, false, x
		}
		if kind != ast.ASTVar {
			continue
		}
		variable, x := c.manager.GetVarIndex(argument)
		if x != nil {
			return 0, false, x
		}
		if source := available[variable]; len(source) != 0 {
			columns[index] = source[0]
			available[variable] = source[1:]
			referenced[source[0]] = true
			boundVariables[index], isBound[index] = variable, true
		} else if len(variableColumns[variable]) != 0 {
			boundVariables[index], isBound[index] = variable, true
		}
	}
	if present {
		removed := make([]int, 0)
		offset := make([]int, signature.Len())
		for column := 0; column < signature.Len(); column++ {
			offset[column] = len(removed)
			if !referenced[column] {
				removed = append(removed, column)
			}
		}
		if len(removed) != 0 {
			current, signature, owned, err = c.project(current, signature, removed, owned, rule.rule)
			if err != nil {
				return 0, false, err
			}
			for index, column := range columns {
				if column >= 0 {
					columns[index] = column - offset[column]
				}
			}
		}
	}
	primary := make(map[uint32]int)
	for index, variable := range boundVariables {
		if isBound[index] && columns[index] >= 0 {
			if _, found := primary[variable]; !found {
				primary[variable] = columns[index]
			}
		}
	}
	for index := 0; index < count; index++ {
		if !isBound[index] || columns[index] >= 0 {
			continue
		}
		column, found := primary[boundVariables[index]]
		if !found {
			return 0, false, ErrCompilerUnsupported
		}
		current, signature, owned, err = c.duplicateColumn(current, signature, column, owned, rule.rule)
		if err != nil {
			return 0, false, err
		}
		columns[index] = signature.Len() - 1
	}
	unbound := make(map[uint32]int)
	for index := 0; index < count; index++ {
		if columns[index] >= 0 {
			continue
		}
		argument, x := c.manager.GetArg(rule.head, uint32(index))
		if x != nil {
			return 0, false, x
		}
		kind, x := c.manager.GetKind(argument)
		if x != nil {
			return 0, false, x
		}
		if kind == ast.ASTVar {
			variable, y := c.manager.GetVarIndex(argument)
			if y != nil {
				return 0, false, y
			}
			if column, found := unbound[variable]; found {
				current, signature, owned, err = c.duplicateColumn(current, signature, column, owned, rule.rule)
				if err != nil {
					return 0, false, err
				}
				columns[index] = signature.Len() - 1
				continue
			}
			domain, y := target.Domain(index)
			if y != nil {
				return 0, false, y
			}
			total, y := c.totalRegister(rule.rule, predicate, sorts[index], domain)
			if y != nil {
				return 0, false, y
			}
			column := signature.Len()
			current, signature, owned, present, err = c.appendColumn(current, signature, owned, present, total, domain, rule.rule)
			if err != nil {
				return 0, false, err
			}
			columns[index], unbound[variable] = column, column
			continue
		}
		value, x := c.relations.EncodeValue(sorts[index], argument)
		if x != nil {
			return 0, false, x
		}
		domain, x := target.Domain(index)
		if x != nil {
			return 0, false, x
		}
		singleton, x := c.constantRegister(rule.rule, sorts[index], domain, value)
		if x != nil {
			return 0, false, x
		}
		column := signature.Len()
		current, signature, owned, present, err = c.appendColumn(current, signature, owned, present, singleton, domain, rule.rule)
		if err != nil {
			return 0, false, err
		}
		columns[index] = column
	}
	current, signature, owned, err = c.reorder(current, signature, columns, owned, rule.rule)
	if err != nil {
		return 0, false, err
	}
	if !signature.equal(target) {
		return 0, false, ErrCompilerUnsupported
	}
	return current, owned, nil
}

func (c *finiteCompiler) groundHead(rule finiteCompiledRule, predicate ast.ASTHandle) (uint32, bool, error) {
	return c.assembleHead(rule, predicate, 0, Signature{}, false, false, nil)
}

func (c *finiteCompiler) singleTailHead(rule finiteCompiledRule, predicate ast.ASTHandle) (uint32, bool, error) {
	tailPredicate, err := c.manager.GetDecl(rule.tail)
	if err != nil {
		return 0, false, err
	}
	sorts, signature, err := c.relations.PredicateSignature(tailPredicate)
	if err != nil {
		return 0, false, err
	}
	current := c.predicateRegs[tailPredicate]
	owned := false
	variableColumns := make(map[uint32][]int)
	tailCount, err := c.manager.GetNumArgs(rule.tail)
	if err != nil {
		return 0, false, err
	}
	for index := uint32(0); index < tailCount; index++ {
		argument, x := c.manager.GetArg(rule.tail, index)
		if x != nil {
			return 0, false, x
		}
		kind, x := c.manager.GetKind(argument)
		if x != nil {
			return 0, false, x
		}
		if kind == ast.ASTVar {
			variable, y := c.manager.GetVarIndex(argument)
			if y != nil {
				return 0, false, y
			}
			column := 0
			for _, columns := range variableColumns {
				column += len(columns)
			}
			variableColumns[variable] = append(variableColumns[variable], column)
			continue
		}
		value, x := c.relations.EncodeValue(sorts[index], argument)
		if x != nil {
			return 0, false, x
		}
		column := 0
		for _, columns := range variableColumns {
			column += len(columns)
		}
		projected, x := projectedSignature(signature, []int{column})
		if x != nil {
			return 0, false, x
		}
		result := current
		if owned {
			c.signatures[result] = projected
		} else {
			result = c.fresh(projected)
		}
		if x = c.append(c.execution, NewSelectEqualProjectInstruction(current, value, column, result), rule.rule); x != nil {
			return 0, false, x
		}
		current, signature, owned = result, projected, true
	}
	for _, columns := range variableColumns {
		if len(columns) < 2 {
			continue
		}
		if !owned {
			result := c.fresh(signature)
			if err = c.append(c.execution, NewCloneInstruction(current, result), rule.rule); err != nil {
				return 0, false, err
			}
			current, owned = result, true
		}
		if err = c.append(c.execution, NewFilterIdenticalInstruction(current, columns), rule.rule); err != nil {
			return 0, false, err
		}
	}
	return c.assembleHead(rule, predicate, current, signature, owned, true, variableColumns)
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
	case finiteSingleTailRule:
		source, owned, err = c.singleTailHead(rule, predicate)
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
		if rule.kind == finiteSingleTailRule {
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
// no-tail and single-tail rule slice into execution and termination bytecode.
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
