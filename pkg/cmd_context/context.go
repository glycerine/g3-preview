// Source: src/cmd_context/cmd_context.cpp retained solver, assertion,
// declaration and scope operations; cmd_context.h.
package cmd_context

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type declaration struct {
	name   string
	handle ast.ASTHandle
	ref    ast.ASTRef
}
type assertion struct {
	expression, tag ast.ASTHandle
	ref, tagRef     ast.ASTRef
}
type scope struct{ functions, sorts, assertions, macros int }
type builtin struct {
	family ast.FamilyID
	kind   int32
}

// Context owns its AST manager, solver, declarations and original assertions.
// Parameter and model-evaluator services are borrowed unless constructed by
// NewStandaloneContext. The AST manager and solver initialize on first use,
// so set-logic and proof options take effect before their construction.
type Context struct {
	config                           smtcore.SMTSolverConfig
	parameters                       *smtcore.ContextParams
	params                           uparams.ParamsRef
	paramsReady                      bool
	manager                          *ast.ASTManager
	solver                           solver.Solver
	tacticResult                     *solver.SimpleCheckSatResult
	logic                            string
	functions                        map[string][]ast.ASTHandle
	sorts                            map[string]ast.ASTHandle
	builtins                         map[string][]builtin
	functionTrail, sortTrail         []declaration
	macros                           map[string][]macroDeclaration
	macroTrail                       []string
	assertions                       []assertion
	scopes                           []scope
	PrintSuccess, GlobalDeclarations bool
	checked                          bool
	status                           lbool.LBool
	owned                            *contextServices
	closed                           bool
	initializationError              error
}

func NewContext(config smtcore.SMTSolverConfig) (*Context, error) {
	if config.Params == nil || config.Globals == nil {
		return nil, uparams.ErrParamsOwner
	}
	c := &Context{config: config, functions: make(map[string][]ast.ASTHandle), sorts: make(map[string]ast.ASTHandle), builtins: make(map[string][]builtin), macros: make(map[string][]macroDeclaration), status: lbool.Undef}
	var err error
	c.params, err = config.Params.NewValue()
	if err != nil {
		return c, err
	}
	c.paramsReady = true
	if config.ModelEvaluator.Params != nil {
		if err = config.Params.Append(&c.params, config.ModelEvaluator.Params); err != nil {
			return c, err
		}
	}
	c.config.ModelEvaluator.Params = &c.params
	c.parameters, err = smtcore.NewContextParams(config.Params, config.Globals, config.Params.BudgetField)
	if err != nil {
		return c, err
	}
	state, err := c.parameters.State()
	if err != nil {
		return c, err
	}
	c.PrintSuccess = state.SMTLib2Compliant
	return c, nil
}

func (c *Context) valid() error {
	if c == nil || c.closed || c.parameters == nil {
		return fmt.Errorf("closed command context")
	}
	return nil
}

func (c *Context) Compliant() (bool, error) {
	if err := c.valid(); err != nil {
		return false, err
	}
	state, err := c.parameters.State()
	return state.SMTLib2Compliant, err
}

// GlobalParameters is borrowed by the source shell's command-line scanner.
func (c *Context) GlobalParameters() (*uparams.GlobalParamsService, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	return c.config.Globals, nil
}

// RefreshGlobalParameters is reached after source command-line assignments
// and before constructing the AST manager and solver.
func (c *Context) RefreshGlobalParameters() error {
	if err := c.valid(); err != nil {
		return err
	}
	if c.manager != nil {
		return fmt.Errorf("command context is already initialized")
	}
	if err := c.parameters.UpdateGlobal(); err != nil {
		return err
	}
	state, err := c.parameters.State()
	if err == nil {
		c.PrintSuccess = state.SMTLib2Compliant
	}
	return err
}

func (c *Context) SetLogic(logic string) error {
	if err := c.valid(); err != nil {
		return err
	}
	if c.logic != "" {
		return fmt.Errorf("the logic has already been set")
	}
	if c.manager != nil {
		return fmt.Errorf("logic must be set before initialization")
	}
	if !solver.SupportedLogic(logic) {
		return fmt.Errorf("unsupported logic %s", logic)
	}
	c.logic = logic
	return nil
}

func (c *Context) NumeralsAsReal() bool { return solver.LogicHasRealsOnly(c.logic) }

func (c *Context) SetBoolOption(name string, value bool) error {
	if err := c.valid(); err != nil {
		return err
	}
	name = strings.ReplaceAll(name, "-", "_")
	switch name {
	case "print_success":
		c.PrintSuccess = value
		return nil
	case "global_declarations", "global_decls":
		if c.manager != nil {
			return fmt.Errorf("option value cannot be modified after initialization")
		}
		c.GlobalDeclarations = value
		return nil
	case "produce_models":
		name = "model"
	case "produce_proofs":
		name = "proof"
	case "produce_unsat_cores":
		name = "unsat_core"
	}
	if name == "proof" || name == "model" || name == "unsat_core" || name == "smtlib2_compliant" {
		if c.manager != nil && (name == "proof" || name == "unsat_core" || name == "smtlib2_compliant") {
			return fmt.Errorf("option value cannot be modified after initialization")
		}
		text := strconv.FormatBool(value)
		if err := c.parameters.Set(&name, &text); err != nil {
			return err
		}
	} else {
		if err := c.config.Globals.SourceSetName(name, strconv.FormatBool(value)); err != nil {
			return err
		}
		name = strings.TrimPrefix(name, "smt.")
	}
	if err := c.config.Params.SetBoolName(&c.params, &name, value); err != nil {
		return err
	}
	if c.solver != nil {
		if err := c.solver.UpdateParams(&c.params); err != nil {
			return err
		}
		if name == "model" {
			return c.solver.SetProduceModels(value)
		}
	}
	return nil
}

// SetUIntOption follows the retained unsigned front-end and module option
// paths. The source parameter catalog validates module names and types.
func (c *Context) SetUIntOption(name string, value uint32) error {
	if err := c.valid(); err != nil {
		return err
	}
	name = strings.ReplaceAll(name, "-", "_")
	text := strconv.FormatUint(uint64(value), 10)
	if name == "timeout" || name == "rlimit" {
		if err := c.parameters.Set(&name, &text); err != nil {
			return err
		}
		if err := c.config.Params.SetUIntName(&c.params, &name, value); err != nil {
			return err
		}
	} else {
		if err := c.config.Globals.SourceSetName(name, text); err != nil {
			return err
		}
		// Preserve the catalog's type. An integer token can also supply a
		// Double parameter; a local UInt shadow would hide that typed value.
	}
	if c.solver != nil {
		return c.solver.UpdateParams(&c.params)
	}
	return nil
}

func (c *Context) Manager() (manager *ast.ASTManager, result error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	if c.initializationError != nil {
		return nil, c.initializationError
	}
	defer func() {
		if result != nil {
			c.initializationError = result
		}
	}()
	if c.manager != nil {
		return c.manager, nil
	}
	p := c.config.Params
	m, err := c.parameters.MkASTManager(ast.ASTManagerConfig{Symbols: p.SymbolsField, Numerals: p.NumeralsField, Budget: p.BudgetField})
	c.manager = m
	if err != nil {
		return nil, err
	}
	if err = m.RegisterDeclarationPlugins(); err != nil {
		return nil, err
	}
	if err = c.initBuiltins(); err != nil {
		return nil, err
	}
	var proofs, models, cores bool
	if err = c.parameters.GetSolverParams(m, &c.params, &proofs, &models, &cores); err != nil {
		return nil, err
	}
	if c.logic == "QF_AUFLIA" {
		c.solver, err = portfolio.NewQFAUFLIAStrategicSolver(m, &c.params, proofs, models, cores, c.config)
	} else {
		var direct *smtcore.SMTSolver
		direct, err = smtcore.NewSMTSolver(m, &c.params, c.logic, c.config)
		if err == nil {
			c.solver = direct
		}
	}
	if err != nil {
		return nil, err
	}
	if err = c.solver.SetProduceModels(models); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Context) DeclareSort(name string) (ast.ASTHandle, error) {
	m, err := c.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if _, found := c.sorts[name]; found || name == "Bool" || name == "Int" || name == "Real" || name == "Array" || name == "BitVec" || name == "String" {
		return ast.ASTHandle{}, fmt.Errorf("sort already defined %s", name)
	}
	symbol, err := m.Symbols.Intern(name)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	sort, err := m.MkUninterpretedSort(symbol, nil)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	ref, err := m.NewRef(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	c.sorts[name] = sort
	c.sortTrail = append(c.sortTrail, declaration{name, sort, ref})
	return sort, nil
}

func (c *Context) LookupSort(name string) (ast.ASTHandle, error) {
	m, err := c.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if s, found := c.sorts[name]; found {
		return s, nil
	}
	if name == "Bool" {
		return m.MkBoolSort()
	}
	if name == "String" || name == "StringSequence" {
		family, err := m.SequenceFamilyID()
		if err != nil {
			return ast.ASTHandle{}, err
		}
		return m.MkFamilySort(family, 2, nil)
	}
	if name == "Int" || name == "Real" {
		family, err := m.ArithmeticFamilyID()
		if err != nil {
			return ast.ASTHandle{}, err
		}
		kind := int32(0)
		if name == "Int" {
			kind = 1
		}
		return m.MkFamilySort(family, kind, nil)
	}
	return ast.ASTHandle{}, fmt.Errorf("unknown sort %s", name)
}

func (c *Context) DeclareFunction(name string, domain []ast.ASTHandle, rangeSort ast.ASTHandle) (ast.ASTHandle, error) {
	m, err := c.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if c.containsMacro(name, domain) {
		return ast.ASTHandle{}, fmt.Errorf("invalid declaration, named expression already defined with this name %s", name)
	}
	if _, found := c.builtins[name]; found {
		return ast.ASTHandle{}, fmt.Errorf("invalid declaration, builtin symbol %s", name)
	}
	symbol, err := m.Symbols.Intern(name)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	d, err := m.MkFuncDecl(symbol, domain, rangeSort, ast.ASTFuncDeclInfo{})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	for _, prior := range c.functions[name] {
		if prior == d {
			return ast.ASTHandle{}, fmt.Errorf("invalid declaration, '%s' (with the given signature) already declared", name)
		}
	}
	ref, err := m.NewRef(d)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	c.functions[name] = append(c.functions[name], d)
	c.functionTrail = append(c.functionTrail, declaration{name, d, ref})
	return d, nil
}

func (c *Context) Assert(formula ast.ASTHandle, name string) (result error) {
	m, err := c.Manager()
	if err != nil {
		return err
	}
	ref, err := m.NewRef(formula)
	if err != nil {
		return err
	}
	row := assertion{expression: formula, ref: ref}
	state, err := c.parameters.State()
	if err == nil && name != "" && state.UnsatCore {
		symbol, e := m.Symbols.Intern(name)
		err = e
		if err == nil {
			b, e := m.MkBoolSort()
			err = e
			if err == nil {
				row.tag, err = m.MkConst(symbol, b)
			}
		}
		if err == nil {
			row.tagRef, err = m.NewRef(row.tag)
		}
	}
	if err == nil {
		if row.tag.IsNull() {
			err = c.solver.Assert(formula)
		} else {
			err = c.solver.AssertTracked(formula, row.tag)
		}
	}
	if err != nil {
		if row.tagRef != (ast.ASTRef{}) {
			err = errors.Join(err, m.DelRef(&row.tagRef))
		}
		return errors.Join(err, m.DelRef(&row.ref))
	}
	c.assertions = append(c.assertions, row)
	c.checked = false
	return nil
}

func (c *Context) Push(count uint32) error {
	if count == 0 {
		return c.valid()
	}
	if _, err := c.Manager(); err != nil {
		return err
	}
	for ; count > 0; count-- {
		if err := c.solver.Push(); err != nil {
			return err
		}
		c.scopes = append(c.scopes, scope{len(c.functionTrail), len(c.sortTrail), len(c.assertions), len(c.macroTrail)})
	}
	c.checked = false
	return nil
}

func (c *Context) Pop(count uint32) error {
	if err := c.valid(); err != nil {
		return err
	}
	if uint64(count) > uint64(len(c.scopes)) {
		return fmt.Errorf("invalid pop command, argument is greater than the current stack depth")
	}
	c.checked = false
	if count == 0 {
		return nil
	}
	if err := c.solver.Pop(count); err != nil {
		return err
	}
	level := len(c.scopes) - int(count)
	s := c.scopes[level]
	if !c.GlobalDeclarations {
		if err := c.restoreMacros(s.macros); err != nil {
			return err
		}
		if err := c.restoreDeclarations(s.functions, s.sorts); err != nil {
			return err
		}
	}
	if err := c.restoreAssertions(s.assertions); err != nil {
		return err
	}
	c.scopes = c.scopes[:level]
	return nil
}

func (c *Context) restoreDeclarations(functions, sorts int) error {
	for len(c.functionTrail) > functions {
		i := len(c.functionTrail) - 1
		row := &c.functionTrail[i]
		values := c.functions[row.name]
		if len(values) == 1 {
			delete(c.functions, row.name)
		} else {
			c.functions[row.name] = values[:len(values)-1]
		}
		if err := c.manager.DelRef(&row.ref); err != nil {
			return err
		}
		c.functionTrail = c.functionTrail[:i]
	}
	for len(c.sortTrail) > sorts {
		i := len(c.sortTrail) - 1
		row := &c.sortTrail[i]
		delete(c.sorts, row.name)
		if err := c.manager.DelRef(&row.ref); err != nil {
			return err
		}
		c.sortTrail = c.sortTrail[:i]
	}
	return nil
}

func (c *Context) restoreAssertions(size int) error {
	for len(c.assertions) > size {
		i := len(c.assertions) - 1
		row := &c.assertions[i]
		if row.tagRef != (ast.ASTRef{}) {
			if err := c.manager.DelRef(&row.tagRef); err != nil {
				return err
			}
		}
		if err := c.manager.DelRef(&row.ref); err != nil {
			return err
		}
		c.assertions = c.assertions[:i]
	}
	return nil
}

func (c *Context) CheckSat(assumptions []ast.ASTHandle) (lbool.LBool, error) {
	if _, err := c.Manager(); err != nil {
		return lbool.Undef, err
	}
	c.checked = false
	if err := c.clearTacticResult(); err != nil {
		return lbool.Undef, err
	}
	status, err := c.solver.CheckSat(assumptions)
	if err == nil {
		c.status, c.checked = status, true
	}
	return status, err
}

func (c *Context) Model() (*model.ModelCore, error) {
	state, err := c.parameters.State()
	if err != nil {
		return nil, err
	}
	if !c.checked || c.status != lbool.True || !state.Model {
		return nil, fmt.Errorf("model is not available")
	}
	return c.checkSatResult().AcquireModel()
}

func (c *Context) WriteModel(out io.Writer) (result error) {
	m, err := c.Model()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, m.DecRef()) }()
	if _, err = io.WriteString(out, "(model \n"); err != nil {
		return err
	}
	if err = m.WriteSMT2(out, c.config.Params, c.config.Globals, 2); err != nil {
		return err
	}
	_, err = io.WriteString(out, ")\n")
	return err
}

func (c *Context) Close() error {
	if c == nil {
		return nil
	}
	if c.closed {
		return nil
	}
	if err := c.clearTacticResult(); err != nil {
		return err
	}
	if c.solver != nil {
		if err := c.solver.Close(); err != nil {
			return err
		}
		c.solver = nil
	}
	if c.manager != nil {
		if err := c.restoreMacros(0); err != nil {
			return err
		}
		if err := c.restoreAssertions(0); err != nil {
			return err
		}
		if err := c.restoreDeclarations(0, 0); err != nil {
			return err
		}
		if err := c.manager.Close(); err != nil {
			return err
		}
		c.manager = nil
	}
	if c.parameters != nil {
		if err := c.parameters.Close(); err != nil {
			return err
		}
		c.parameters = nil
	}
	if c.paramsReady {
		if err := c.config.Params.Del(&c.params); err != nil {
			return err
		}
		c.paramsReady = false
	}
	if c.owned != nil {
		if err := c.owned.Close(); err != nil {
			return err
		}
		c.owned = nil
	}
	c.closed = true
	return nil
}
