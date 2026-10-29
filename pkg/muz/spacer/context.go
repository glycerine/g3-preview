package spacer

// Source: retained construction, init_rules, solve_core, and propagation in
// src/muz/spacer/spacer_context.h and spacer_context.cpp.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/muz/base"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
)

var ErrContextOwner = errors.New("muz/spacer: invalid or closed context")

type Context struct {
	manager      *ast.ASTManager
	owner        *Manager
	config       PropSolverConfig
	transformers map[ast.ASTHandle]*PredicateTransformer
	order        []ast.ASTHandle
	query        *PredicateTransformer
	queryDecl    ast.ASTHandle
	queue        ProofObligationQueue
	lastResult   lbool.LBool
	initialized  bool
	solved       bool
	closed       bool
}

func NewContext(manager *ast.ASTManager, params *uparams.ParamsRef, maximumContexts uint32, config smtcore.SMTSolverConfig) (*Context, error) {
	if manager == nil {
		return nil, ErrContextOwner
	}
	owner, err := NewManager(manager, params, maximumContexts, config)
	if err != nil {
		return nil, err
	}
	propConfig, err := spacerPropSolverConfig(config)
	if err != nil {
		return nil, errors.Join(err, owner.Close())
	}
	return &Context{
		manager: manager, owner: owner, transformers: make(map[ast.ASTHandle]*PredicateTransformer), lastResult: lbool.Undef,
		config: propConfig,
	}, nil
}

func spacerPropSolverConfig(config smtcore.SMTSolverConfig) (PropSolverConfig, error) {
	if config.Params == nil || config.Globals == nil || config.Globals.Context != config.Params {
		return PropSolverConfig{}, ErrContextOwner
	}
	fixedpoint, err := config.Globals.SourceGetModuleName("fixedpoint")
	if err != nil {
		return PropSolverConfig{}, err
	}
	get := func(name string, fallback bool) (bool, error) {
		return config.Params.GetBoolName(&fixedpoint, &name, fallback)
	}
	newCore, err := get("spacer.new_unsat_core", true)
	if err != nil {
		return PropSolverConfig{}, err
	}
	minimize, err := get("spacer.minimize_unsat_core", false)
	if err != nil {
		return PropSolverConfig{}, err
	}
	if minimize {
		return PropSolverConfig{}, fmt.Errorf("muz/spacer: spacer.minimize_unsat_core is not ported")
	}
	optimized, err := get("spacer.farkas_optimized", true)
	if err != nil {
		return PropSolverConfig{}, err
	}
	split, err := get("spacer.split_farkas_literals", false)
	if err != nil {
		return PropSolverConfig{}, err
	}
	farkasAConst, err := get("spacer.farkas_a_const", true)
	if err != nil {
		return PropSolverConfig{}, err
	}
	keepProxy, err := get("spacer.keep_proxy", true)
	if err != nil {
		return PropSolverConfig{}, err
	}
	mode := ItpCoreLegacyFarkas
	if newCore {
		mode = ItpCoreNewFarkas
		if optimized {
			mode = ItpCoreNewOptimized
		}
	}
	return PropSolverConfig{
		Itp: ItpSolverConfig{
			Params: config.Params, Globals: config.Globals, SplitLiterals: split,
			CoreMode: mode, FarkasAConst: farkasAConst,
		},
		KeepProxy: keepProxy,
	}, nil
}

func (c *Context) open() error {
	if c == nil || c.closed || c.manager == nil || c.owner == nil || c.owner.open() != nil {
		return ErrContextOwner
	}
	return nil
}

// SetBackground replaces the formula asserted by subsequently constructed
// predicate solvers. It must be called before rules are initialized.
func (c *Context) SetBackground(background ast.ASTHandle) error {
	if err := c.open(); err != nil {
		return err
	}
	if c.initialized {
		return ast.ErrASTDomain
	}
	return c.owner.SetBackground(background)
}

func closeSpacerRuleBuckets(buckets map[ast.ASTHandle][]*base.Rule) (result error) {
	for _, rules := range buckets {
		for _, rule := range rules {
			result = errors.Join(result, rule.Close())
		}
	}
	return result
}

func appendSpacerDeclaration(order *[]ast.ASTHandle, seen map[ast.ASTHandle]bool, declaration ast.ASTHandle) {
	if !seen[declaration] {
		seen[declaration] = true
		*order = append(*order, declaration)
	}
}

func (c *Context) Initialize(set *base.RuleSet) (resultErr error) {
	return c.InitializePredicates(set, nil)
}

// InitializePredicates constructs transformers for every registered live
// predicate, including output predicates without defining rules.
func (c *Context) InitializePredicates(set *base.RuleSet, declarations []ast.ASTHandle) (resultErr error) {
	if err := c.open(); err != nil {
		return err
	}
	if c.initialized || set == nil {
		return ast.ErrASTDomain
	}
	count, err := set.Len()
	if err != nil {
		return err
	}
	var order []ast.ASTHandle
	seen := make(map[ast.ASTHandle]bool)
	buckets := make(map[ast.ASTHandle][]*base.Rule)
	defer func() { resultErr = errors.Join(resultErr, closeSpacerRuleBuckets(buckets)) }()
	for _, declaration := range declarations {
		appendSpacerDeclaration(&order, seen, declaration)
	}
	for index := 0; index < count; index++ {
		rule, err := set.Rule(index)
		if err != nil {
			return err
		}
		head, err := rule.HeadDeclaration()
		if err != nil {
			return errors.Join(err, rule.Close())
		}
		appendSpacerDeclaration(&order, seen, head)
		buckets[head] = append(buckets[head], rule)
		tails, err := rule.UninterpretedTailSize()
		if err != nil {
			return err
		}
		for tail := 0; tail < tails; tail++ {
			declaration, err := rule.TailDeclaration(tail)
			if err != nil {
				return err
			}
			appendSpacerDeclaration(&order, seen, declaration)
		}
	}
	created := make(map[ast.ASTHandle]*PredicateTransformer, len(order))
	committed := false
	defer func() {
		if committed {
			return
		}
		for index := len(order) - 1; index >= 0; index-- {
			if transformer := created[order[index]]; transformer != nil {
				resultErr = errors.Join(resultErr, transformer.Close())
			}
		}
	}()
	for _, declaration := range order {
		transformer, err := NewRulePredicateTransformer(c.owner, declaration, c.config)
		if err != nil {
			return err
		}
		created[declaration] = transformer
	}
	for _, declaration := range order {
		if err = created[declaration].InitializeRules(created, buckets[declaration]); err != nil {
			return err
		}
	}
	for _, declaration := range order {
		if err = created[declaration].InitReachFacts(); err != nil {
			return err
		}
	}
	c.transformers = created
	c.order = order
	c.initialized = true
	committed = true
	return nil
}

func (c *Context) SetQuery(declaration ast.ASTHandle) error {
	if err := c.open(); err != nil {
		return err
	}
	if !c.initialized || c.solved {
		return ast.ErrASTDomain
	}
	transformer := c.transformers[declaration]
	if transformer == nil {
		return ast.ErrASTDomain
	}
	c.queryDecl = declaration
	c.query = transformer
	return nil
}

func (c *Context) Transformer(declaration ast.ASTHandle) (*PredicateTransformer, bool, error) {
	if err := c.open(); err != nil {
		return nil, false, err
	}
	transformer, found := c.transformers[declaration]
	return transformer, found, nil
}

func (c *Context) NumLevels(declaration ast.ASTHandle) (uint32, error) {
	if err := c.open(); err != nil {
		return 0, err
	}
	transformer := c.transformers[declaration]
	if transformer == nil {
		return 0, ast.ErrASTDomain
	}
	return transformer.NumLevels()
}

func (c *Context) propagate(maximum uint32) (bool, error) {
	for level := uint32(0); level <= maximum; level++ {
		all := true
		for _, declaration := range c.order {
			propagated, err := c.transformers[declaration].PropagateToNextLevel(level)
			if err != nil {
				return false, err
			}
			all = all && propagated
		}
		if all {
			for _, declaration := range c.order {
				if err := c.transformers[declaration].PropagateToInfinity(level); err != nil {
					return false, err
				}
			}
			return true, nil
		}
	}
	return false, nil
}

// Solve runs bounded Spacer search starting at fromLevel. maxLevels is the
// number of search levels to attempt before returning undef.
func (c *Context) Solve(fromLevel, maxLevels uint32) (_ lbool.LBool, resultErr error) {
	if err := c.open(); err != nil {
		return lbool.Undef, err
	}
	if !c.initialized || c.query == nil || c.solved {
		return lbool.Undef, ast.ErrASTDomain
	}
	truth, err := c.manager.MkTrue()
	if err != nil {
		return lbool.Undef, err
	}
	root, err := c.query.NewProofObligation(nil, fromLevel, 0, truth, nil)
	if err != nil {
		return lbool.Undef, err
	}
	if err = c.queue.SetRoot(root); err != nil {
		return lbool.Undef, err
	}
	c.solved = true
	level := fromLevel
	for attempt := uint32(0); attempt < maxLevels; attempt++ {
		reachable, err := c.queue.CheckReachability()
		if err != nil {
			return lbool.Undef, err
		}
		if reachable {
			c.lastResult = lbool.True
			return c.lastResult, nil
		}
		if level > 0 {
			inductive, err := c.propagate(level)
			if err != nil {
				return lbool.Undef, err
			}
			if inductive {
				c.lastResult = lbool.False
				return c.lastResult, nil
			}
		}
		if err = c.queue.IncrementLevel(); err != nil {
			return lbool.Undef, err
		}
		level = c.queue.MaximumLevel()
	}
	c.lastResult = lbool.Undef
	return c.lastResult, nil
}

func (c *Context) LastResult() (lbool.LBool, error) {
	if err := c.open(); err != nil {
		return lbool.Undef, err
	}
	return c.lastResult, nil
}

func (c *Context) CollectStatistics(destination *statistics.Statistics) error {
	if err := c.open(); err != nil {
		return err
	}
	if destination == nil {
		return ast.ErrASTHandle
	}
	if err := c.owner.CollectStatistics(destination); err != nil {
		return err
	}
	for _, declaration := range c.order {
		if err := c.transformers[declaration].CollectStatistics(destination); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) ResetStatistics() error {
	if err := c.open(); err != nil {
		return err
	}
	if err := c.owner.ResetStatistics(); err != nil {
		return err
	}
	for _, declaration := range c.order {
		if err := c.transformers[declaration].ResetStatistics(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Close() (result error) {
	if c == nil || c.manager == nil {
		return ErrContextOwner
	}
	if c.closed {
		return nil
	}
	for index := len(c.order) - 1; index >= 0; index-- {
		declaration := c.order[index]
		if transformer := c.transformers[declaration]; transformer != nil {
			if err := transformer.Close(); err != nil {
				result = errors.Join(result, err)
			} else {
				c.transformers[declaration] = nil
			}
		}
	}
	if result == nil {
		result = c.owner.Close()
	}
	if result == nil {
		c.manager = nil
		c.owner = nil
		c.transformers = nil
		c.order = nil
		c.query = nil
		c.closed = true
	}
	return result
}
