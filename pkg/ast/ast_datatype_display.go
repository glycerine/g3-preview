// Source: src/ast/datatype_decl_plugin.cpp:1011-1070.
package ast

import (
	"io"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func (m *ASTManager) DatatypeIsDeclared(sort ASTHandle) (bool, error) {
	if err := m.lock(); err != nil {
		return false, err
	}
	defer m.mu.Unlock()
	r, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return false, err
	}
	if m.datatypePlugin == nil {
		return false, nil
	}
	info := m.sortAt(r.slot).info
	if info == (ASTSortInfo{}) {
		return false, nil
	}
	identity, err := m.sortInfo(info)
	if err != nil {
		return false, err
	}
	if identity.family != m.datatypePlugin.family || identity.kind != ASTDatatypeSort || identity.count == 0 {
		return false, nil
	}
	parameter, err := m.infoParameter(identity, 0)
	if err != nil {
		return false, err
	}
	value, err := m.parameter(parameter)
	if err != nil {
		return false, err
	}
	if value.Kind != ASTParameterSymbol {
		return false, nil
	}
	return m.datatypePlugin.definitionNames[value.symbol] != nil, nil
}

func (m *ASTManager) DisplayDatatype(sort ASTHandle, out io.Writer) error {
	if out == nil {
		return ErrASTDomain
	}
	if err := m.lock(); err != nil {
		return err
	}
	defer m.mu.Unlock()
	root, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return err
	}
	rootDefinition, err := m.datatypeDefinitionForSort(root)
	if err != nil {
		return err
	}
	writeSymbol := func(symbolText string) error {
		_, e := io.WriteString(out, symbolText)
		return e
	}
	symbolText := func(symbol usymbol.Symbol) (string, error) {
		return m.Symbols.Display(symbol)
	}
	rootName, err := symbolText(rootDefinition.name)
	if err != nil {
		return err
	}
	if err = writeSymbol(rootName + " where\n"); err != nil {
		return err
	}
	stack := []astNodeRef{root}
	marked := map[astNodeRef]bool{root: true}
	for len(stack) != 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		definition, e := m.datatypeDefinitionForSort(current)
		if e != nil {
			return e
		}
		name, e := symbolText(definition.name)
		if e != nil {
			return e
		}
		if e = writeSymbol(name + " =\n"); e != nil {
			return e
		}
		for ci := range definition.constructors {
			constructor := &definition.constructors[ci]
			constructorName, e := symbolText(constructor.name)
			if e != nil {
				return e
			}
			if e = writeSymbol("  " + constructorName + " :: "); e != nil {
				return e
			}
			ranges, e := m.datatypeInstantiatedRanges(current, definition, constructor)
			if e != nil {
				return e
			}
			for i, field := range ranges {
				accessorName, e := symbolText(constructor.accessors[i])
				if e != nil {
					return e
				}
				fieldName, e := symbolText(m.sortAt(field.slot).name)
				if e != nil {
					return e
				}
				if e = writeSymbol("(" + accessorName + ": " + fieldName + ") "); e != nil {
					return e
				}
				fieldDefinition, x := m.datatypeDefinitionForSort(field)
				if x == nil && fieldDefinition.classID == rootDefinition.classID && !marked[field] {
					marked[field] = true
					stack = append(stack, field)
				}
			}
			if e = writeSymbol("\n"); e != nil {
				return e
			}
		}
	}
	return nil
}
