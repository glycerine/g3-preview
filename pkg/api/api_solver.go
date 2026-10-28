package api

// Source: retained public lifecycle of src/api/api_context.cpp and
// src/api/api_solver.cpp. Go exposes the concrete owning solver context rather
// than reproducing opaque C handles or foreign-language bindings.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine" // generated parameter collectors
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
)

var ErrSMTContextOwner = errors.New("api: invalid or closed SMT context")

// SMTContext is an owning, single-goroutine Go solver context. Copies share
// the same state; closing any copy closes the context. Manager returns the
// borrowed AST owner used to construct formulas. Keep caller-owned ASTRefs and
// model references within their documented lifetimes and release them before
// Close.
type SMTContext struct{ *smtContextState }

type smtContextState struct {
	command *cmd_context.Context
	closed  bool
}

// NewSMTContext constructs an independent solver context with source defaults.
// Logic and proof/core options must be selected before Manager initializes the
// AST manager, matching the native context lifecycle.
func NewSMTContext() (*SMTContext, error) {
	command, err := cmd_context.NewStandaloneContext()
	if err != nil {
		return nil, err
	}
	return &SMTContext{smtContextState: &smtContextState{command: command}}, nil
}

func (c *SMTContext) open() error {
	if c == nil || c.smtContextState == nil || c.closed || c.command == nil {
		return ErrSMTContextOwner
	}
	return nil
}

// Manager returns the borrowed AST manager for term construction and
// inspection.
func (c *SMTContext) Manager() (*ast.ASTManager, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.command.Manager()
}

func (c *SMTContext) SetLogic(logic string) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.SetLogic(logic)
}

func (c *SMTContext) SetBoolOption(name string, value bool) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.SetBoolOption(name, value)
}

func (c *SMTContext) SetUIntOption(name string, value uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.SetUIntOption(name, value)
}

func (c *SMTContext) SetSymbolOption(name, value string) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.SetSymbolOption(name, value)
}

// Assert adds an untracked Boolean formula.
func (c *SMTContext) Assert(formula ast.ASTHandle) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.Assert(formula, "")
}

// AssertTracked adds a Boolean formula with a named unsat-core label.
func (c *SMTContext) AssertTracked(formula ast.ASTHandle, name string) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.Assert(formula, name)
}

func (c *SMTContext) Push() error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.Push(1)
}

func (c *SMTContext) Pop(count uint32) error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.Pop(count)
}

// CheckSat checks the current assertions under temporary assumptions.
func (c *SMTContext) CheckSat(assumptions ...ast.ASTHandle) (lbool.LBool, error) {
	if err := c.open(); err != nil {
		return lbool.Undef, err
	}
	return c.command.CheckSat(assumptions)
}

// Model returns one owning reference. The caller must call DecRef.
func (c *SMTContext) Model() (*model.ModelCore, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.command.Model()
}

// UnsatCore returns borrowed tracked labels from the most recent result.
func (c *SMTContext) UnsatCore() ([]ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.command.UnsatCore()
}

// Proof returns a borrowed proof from the most recent result.
func (c *SMTContext) Proof() (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return c.command.Proof()
}

func (c *SMTContext) ReasonUnknown() (string, error) {
	if err := c.open(); err != nil {
		return "", err
	}
	return c.command.ReasonUnknown()
}

func (c *SMTContext) ResetAssertions() error {
	if err := c.open(); err != nil {
		return err
	}
	return c.command.ResetAssertions()
}

// Close releases the solver, AST manager, and all private runtime services.
func (c *SMTContext) Close() error {
	if c == nil || c.smtContextState == nil || c.closed {
		return nil
	}
	if err := c.command.Close(); err != nil {
		return err
	}
	c.command = nil
	c.closed = true
	return nil
}
