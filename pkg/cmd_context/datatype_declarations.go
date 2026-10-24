// Source: src/cmd_context/cmd_context.cpp dt_eh::operator(), concrete datatype
// instantiations. Parametric declaration owners remain a separate parser port.
package cmd_context

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

// DeclareDatatypes publishes the sorts, constructors, recognizers and accessors
// of a concrete mutually recursive block in the current declaration scope.
func (c *Context) DeclareDatatypes(definitions []ast.DatatypeDefinition) (result error) {
	m, err := c.Manager()
	if err != nil {
		return err
	}
	if len(definitions) == 0 {
		return nil
	}
	names := make([]string, len(definitions))
	seen := make(map[string]bool)
	for i, d := range definitions {
		if len(d.Parameters) != 0 {
			return fmt.Errorf("parameterized datatype declarations are not ported yet")
		}
		name, e := m.Symbols.String(d.Name)
		if e != nil {
			return e
		}
		names[i] = name
		if _, found := c.sorts[name]; found || seen[name] || c.BuiltinSortAvailable(name) {
			return fmt.Errorf("sort already defined %s", name)
		}
		seen[name] = true
	}
	sorts, err := m.MkDatatypes(definitions)
	if err != nil {
		return err
	}
	functionsStart, sortsStart := len(c.functionTrail), len(c.sortTrail)
	defer func() {
		if result != nil {
			result = errors.Join(result, c.restoreDeclarations(functionsStart, sortsStart))
		}
	}()
	for i, s := range sorts {
		ref, e := m.NewRef(s)
		if e != nil {
			return e
		}
		c.sorts[names[i]] = s
		c.sortTrail = append(c.sortTrail, declaration{names[i], s, ref})
		constructors, e := m.DatatypeConstructors(s)
		if e != nil {
			return e
		}
		for _, constructor := range constructors {
			if e = c.insertDatatypeFunction(constructor); e != nil {
				return e
			}
			recognizer, e := m.DatatypeConstructorRecognizer(constructor)
			if e != nil {
				return e
			}
			if e = c.insertDatatypeFunction(recognizer); e != nil {
				return e
			}
			accessors, e := m.DatatypeConstructorAccessors(constructor)
			if e != nil {
				return e
			}
			for _, accessor := range accessors {
				if e = c.insertDatatypeFunction(accessor); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func (c *Context) insertDatatypeFunction(d ast.ASTHandle) error {
	m := c.manager
	if err := c.logicChecker.CheckDeclaration(d); err != nil {
		return err
	}
	symbol, err := m.GetASTName(d)
	if err != nil {
		return err
	}
	name, err := m.Symbols.String(symbol)
	if err != nil {
		return err
	}
	arity, err := m.GetArity(d)
	if err != nil {
		return err
	}
	domain := make([]ast.ASTHandle, arity)
	for i := range domain {
		domain[i], err = m.GetDomain(d, uint32(i))
		if err != nil {
			return err
		}
	}
	if c.containsMacro(name, domain) {
		return fmt.Errorf("invalid declaration, named expression already defined with this name %s", name)
	}
	if _, found := c.builtins[name]; found {
		return fmt.Errorf("invalid declaration, builtin symbol %s", name)
	}
	for _, prior := range c.functions[name] {
		if prior == d {
			return fmt.Errorf("invalid declaration, '%s' (with the given signature) already declared", name)
		}
	}
	ref, err := m.NewRef(d)
	if err != nil {
		return err
	}
	c.functions[name] = append(c.functions[name], d)
	c.functionTrail = append(c.functionTrail, declaration{name, d, ref})
	return nil
}
