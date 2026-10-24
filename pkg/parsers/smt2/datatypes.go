// Source: src/parsers/smt2/smt2parser.cpp parse_declare_datatype(s),
// parse_datatype_dec and parse_constructor_decls: concrete declaration grammar.
package smt2

import (
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

type datatypePendingField struct {
	datatype, constructor, accessor int
	name                            string
}

func (p *commandParser) declareDatatypes(single bool) error {
	m, err := p.context.Manager()
	if err != nil {
		return err
	}
	var definitions []ast.DatatypeDefinition
	var pending []datatypePendingField
	var headers []string
	modern := false
	if single {
		name, e := p.symbol()
		if e != nil {
			return e
		}
		headers = append(headers, name)
	} else {
		if err = p.expect(LeftParenToken); err != nil {
			return err
		}
		modern = p.token == LeftParenToken
		if modern {
			for p.token != RightParenToken {
				if err = p.expect(LeftParenToken); err != nil {
					return err
				}
				name, e := p.symbol()
				if e != nil {
					return e
				}
				arity, e := p.unsigned()
				if e != nil {
					return e
				}
				if arity != 0 {
					return fmt.Errorf("parameterized datatype declarations are not ported yet")
				}
				headers = append(headers, name)
				if err = p.expect(RightParenToken); err != nil {
					return err
				}
			}
		} else if p.token != RightParenToken {
			return fmt.Errorf("parameterized datatype declarations are not ported yet")
		}
		if err = p.expect(RightParenToken); err != nil {
			return err
		}
	}
	parseBody := func(name string, old bool) error {
		if e := p.expect(LeftParenToken); e != nil {
			return e
		}
		if old {
			var e error
			name, e = p.symbol()
			if e != nil {
				return e
			}
		}
		if p.token == SymbolToken && p.scanner.ID() == "par" {
			return fmt.Errorf("parameterized datatype declarations are not ported yet")
		}
		symbol, e := m.Symbols.Intern(name)
		if e != nil {
			return e
		}
		definition := ast.DatatypeDefinition{Name: symbol}
		for p.token != RightParenToken {
			parenthesized := p.token == LeftParenToken
			if parenthesized {
				if e = p.advance(); e != nil {
					return e
				}
			}
			constructorName, e := p.symbol()
			if e != nil {
				return e
			}
			constructorSymbol, e := m.Symbols.Intern(constructorName)
			if e != nil {
				return e
			}
			recognizer, e := m.Symbols.Intern("is-" + constructorName)
			if e != nil {
				return e
			}
			constructor := ast.DatatypeConstructorDefinition{Name: constructorSymbol, Recognizer: recognizer}
			if parenthesized {
				for p.token != RightParenToken {
					if e = p.expect(LeftParenToken); e != nil {
						return e
					}
					fieldName, e := p.symbol()
					if e != nil {
						return e
					}
					fieldSymbol, e := m.Symbols.Intern(fieldName)
					if e != nil {
						return e
					}
					field := ast.DatatypeAccessorDefinition{Name: fieldSymbol}
					if p.token == SymbolToken {
						sortName := p.scanner.ID()
						pending = append(pending, datatypePendingField{len(definitions), len(definition.Constructors), len(constructor.Accessors), sortName})
						if e = p.advance(); e != nil {
							return e
						}
					} else {
						field.Range, e = p.sort()
						if e != nil {
							return e
						}
					}
					constructor.Accessors = append(constructor.Accessors, field)
					if e = p.expect(RightParenToken); e != nil {
						return e
					}
				}
				if e = p.expect(RightParenToken); e != nil {
					return e
				}
			}
			definition.Constructors = append(definition.Constructors, constructor)
		}
		if len(definition.Constructors) == 0 {
			return fmt.Errorf("invalid datatype declaration, datatype must have at least one constructor")
		}
		definitions = append(definitions, definition)
		return p.expect(RightParenToken)
	}
	if single {
		if err = parseBody(headers[0], false); err != nil {
			return err
		}
	} else {
		if err = p.expect(LeftParenToken); err != nil {
			return err
		}
		for p.token != RightParenToken {
			name := ""
			if modern {
				if len(definitions) >= len(headers) {
					return fmt.Errorf("invalid datatype declaration, too many data-type bodies defined")
				}
				name = headers[len(definitions)]
			}
			if err = parseBody(name, !modern); err != nil {
				return err
			}
		}
		if modern && len(definitions) != len(headers) {
			return fmt.Errorf("invalid datatype declaration, too few datatype definitions compared to declared sorts")
		}
		if err = p.expect(RightParenToken); err != nil {
			return err
		}
	}
	if err = p.requireEnd(); err != nil {
		return err
	}
	indices := make(map[string]uint32)
	for i, d := range definitions {
		name, e := m.Symbols.String(d.Name)
		if e != nil {
			return e
		}
		if _, found := indices[name]; found {
			return fmt.Errorf("datatype sort already declared %s", name)
		}
		indices[name] = uint32(i)
	}
	for _, row := range pending {
		field := &definitions[row.datatype].Constructors[row.constructor].Accessors[row.accessor]
		if index, found := indices[row.name]; found {
			field.Recursive = true
			field.RecursiveIndex = index
		} else {
			field.Range, err = p.context.LookupSort(row.name)
			if err != nil {
				return fmt.Errorf("invalid datatype declaration, unknown sort '%s'", row.name)
			}
		}
	}
	// Missing references precede duplicate-accessor checks in both retained
	// declaration forms, including mutually recursive blocks.
	for _, definition := range definitions {
		accessorNames := make(map[string]bool)
		for _, constructor := range definition.Constructors {
			for _, accessor := range constructor.Accessors {
				name, e := m.Symbols.String(accessor.Name)
				if e != nil {
					return e
				}
				if accessorNames[name] {
					return fmt.Errorf("invalid datatype declaration, repeated accessor identifier '%s'", name)
				}
				accessorNames[name] = true
			}
		}
	}
	return p.context.DeclareDatatypes(definitions)
}
