package cmd_context

// Source: src/muz/fp/dl_cmds.cpp declare-rel/rule/query command context and
// src/muz/base/dl_context.cpp retained orchestration. The current backend is
// the exact positive zero-arity fragment exposed by muz/fp.

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
	engine, err := muzfp.NewPropositionalEngine(manager)
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
	engine, err := c.fixedpointContext()
	if err != nil {
		return err
	}
	return engine.AddRule(formula, name)
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
	manager, err := c.Manager()
	if err != nil {
		return lbool.Undef, err
	}
	query, err := manager.MkApp(declaration, nil)
	if err != nil {
		return lbool.Undef, err
	}
	return engine.Query(query)
}
