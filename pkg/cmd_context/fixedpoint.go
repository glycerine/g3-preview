package cmd_context

// Source: src/muz/fp/dl_cmds.cpp declare-rel/rule/query command context and
// src/muz/base/dl_context.cpp retained orchestration. The current backend is
// the positive finite-domain fragment exposed by muz/fp.

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	muzfp "github.com/glycerine/g3/pkg/muz/fp"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (c *Context) fixedpointContext() (*muzfp.PropositionalEngine, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	if c.fixedpoint != nil {
		return c.fixedpoint, nil
	}
	manager, err := c.Manager()
	if err != nil {
		return nil, err
	}
	engine, err := muzfp.NewPropositionalEngineWithServices(manager, c.config.Params, c.config.Globals)
	if err != nil {
		return nil, err
	}
	c.fixedpoint = engine
	return engine, nil
}

func (c *Context) DeclareRelation(name string, domain []ast.ASTHandle) (ast.ASTHandle, error) {
	manager, err := c.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	boolean, err := manager.MkBoolSort()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	declaration, err := c.DeclareFunction(name, domain, boolean)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	engine, err := c.fixedpointContext()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if err = engine.RegisterPredicate(declaration); err != nil {
		return ast.ASTHandle{}, err
	}
	return declaration, nil
}

func (c *Context) AddFixedpointRule(formula ast.ASTHandle, name string) error {
	return c.AddFixedpointRuleWithBound(formula, name, ^uint32(0))
}

// AddFixedpointRuleWithBound accepts the rule command's retained recursion
// bound. The concrete Datalog engine ignores it, as does the source Datalog
// flush path; the bound is consumed only by the unretained duality engine.
func (c *Context) AddFixedpointRuleWithBound(formula ast.ASTHandle, name string, _ uint32) error {
	engine, err := c.fixedpointContext()
	if err != nil {
		return err
	}
	return engine.AddRule(formula, name)
}

// DeclareFixedpointVariable declares and registers an implicit Datalog
// variable, matching the declare-var command's constant-to-bound-variable path.
func (c *Context) DeclareFixedpointVariable(name string, sort ast.ASTHandle) error {
	declaration, err := c.DeclareFunction(name, nil, sort)
	if err != nil {
		return err
	}
	manager, err := c.Manager()
	if err != nil {
		return err
	}
	variable, err := manager.MkApp(declaration, nil)
	if err != nil {
		return err
	}
	engine, err := c.fixedpointContext()
	if err != nil {
		return err
	}
	return engine.RegisterVariable(variable)
}

func (c *Context) PushFixedpoint() error {
	engine, err := c.fixedpointContext()
	if err != nil {
		return err
	}
	return engine.Push()
}

func (c *Context) PopFixedpoint() error {
	engine, err := c.fixedpointContext()
	if err != nil {
		return err
	}
	return engine.Pop()
}

func (c *Context) QueryFixedpoint(declaration ast.ASTHandle) (lbool.LBool, error) {
	engine, err := c.fixedpointContext()
	if err != nil {
		return lbool.Undef, err
	}
	selection, supported, err := c.GetOption(":fixedpoint.engine")
	if err != nil {
		return lbool.Undef, err
	}
	if !supported {
		return lbool.Undef, fmt.Errorf("cmd_context: fixedpoint.engine option is unavailable")
	}
	if err = engine.SelectEngine(selection); err != nil {
		return lbool.Undef, err
	}
	return engine.QueryPredicate(declaration)
}

// FixedpointAnswer returns an owning formula for the most recently saturated
// finite relation. The caller releases it through Manager.DelRef.
func (c *Context) FixedpointAnswer(declaration ast.ASTHandle) (ast.ASTRef, error) {
	engine, err := c.fixedpointContext()
	if err != nil {
		return ast.ASTRef{}, err
	}
	return engine.Answer(declaration)
}
