package spacer

// Source: the state-symbol tuple and formula-conversion paths in
// src/muz/spacer/spacer_sym_mux.h and spacer_sym_mux.cpp.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

var ErrSymMuxOwner = errors.New("muz/spacer: invalid or closed symbol multiplexer")

type SymMux struct {
	manager       *ast.ASTManager
	suffixes      []string
	usedSuffixes  map[string]bool
	nextSuffix    uint32
	primaryToAll  map[ast.ASTHandle][]ast.ASTHandle
	symbolToIndex map[ast.ASTHandle]uint32
	symbolToPrim  map[ast.ASTHandle]ast.ASTHandle
	prefixToPrim  map[ast.ASTHandle]ast.ASTHandle
	primToPrefix  map[ast.ASTHandle]ast.ASTHandle
	primary       []ast.ASTHandle
	nonModel      map[ast.ASTHandle]bool
	references    []ast.ASTRef
	closed        bool
}

func NewSymMux(manager *ast.ASTManager, suffixes []string) (*SymMux, error) {
	if manager == nil {
		return nil, ErrSymMuxOwner
	}
	result := &SymMux{
		manager: manager, suffixes: append([]string(nil), suffixes...), usedSuffixes: make(map[string]bool),
		primaryToAll: make(map[ast.ASTHandle][]ast.ASTHandle), symbolToIndex: make(map[ast.ASTHandle]uint32),
		symbolToPrim: make(map[ast.ASTHandle]ast.ASTHandle), prefixToPrim: make(map[ast.ASTHandle]ast.ASTHandle),
		primToPrefix: make(map[ast.ASTHandle]ast.ASTHandle), nonModel: make(map[ast.ASTHandle]bool),
	}
	for _, suffix := range suffixes {
		result.usedSuffixes[suffix] = true
	}
	return result, nil
}

func (m *SymMux) open() error {
	if m == nil || m.closed || m.manager == nil {
		return ErrSymMuxOwner
	}
	return nil
}

func (m *SymMux) suffix(index uint32) string {
	for uint32(len(m.suffixes)) <= index {
		var value string
		for {
			value = fmt.Sprintf("_%d", m.nextSuffix)
			m.nextSuffix++
			if !m.usedSuffixes[value] {
				break
			}
		}
		m.usedSuffixes[value] = true
		m.suffixes = append(m.suffixes, value)
	}
	return m.suffixes[index]
}

func (m *SymMux) declarationSignature(declaration ast.ASTHandle) ([]ast.ASTHandle, ast.ASTHandle, error) {
	arity, err := m.manager.GetArity(declaration)
	if err != nil {
		return nil, ast.ASTHandle{}, err
	}
	domain := make([]ast.ASTHandle, arity)
	for index := uint32(0); index < arity; index++ {
		domain[index], err = m.manager.GetDomain(declaration, index)
		if err != nil {
			return nil, ast.ASTHandle{}, err
		}
	}
	rangeSort, err := m.manager.GetRange(declaration)
	return domain, rangeSort, err
}

func (m *SymMux) createVariant(prefix ast.ASTHandle, index uint32) (ast.ASTHandle, error) {
	name, err := m.manager.GetASTName(prefix)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	spelling, present, err := m.manager.Symbols.BareString(name)
	if err != nil || !present {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return ast.ASTHandle{}, err
	}
	variantName, err := m.manager.Symbols.Intern(spelling + m.suffix(index))
	if err != nil {
		return ast.ASTHandle{}, err
	}
	domain, rangeSort, err := m.declarationSignature(prefix)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return m.manager.MkFuncDecl(variantName, domain, rangeSort, ast.ASTFuncDeclInfo{})
}

func (m *SymMux) pin(handle ast.ASTHandle) error {
	reference, err := m.manager.NewRef(handle)
	if err == nil {
		m.references = append(m.references, reference)
	}
	return err
}

func (m *SymMux) CreateTuple(prefix ast.ASTHandle, tupleLength uint32) ([]ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return nil, err
	}
	if tupleLength == 0 {
		return nil, ast.ErrASTDomain
	}
	if _, found := m.prefixToPrim[prefix]; found {
		return nil, ast.ErrASTDomain
	}
	tuple := make([]ast.ASTHandle, tupleLength)
	for index := range tuple {
		variant, err := m.createVariant(prefix, uint32(index))
		if err != nil {
			return nil, err
		}
		if err = m.pin(variant); err != nil {
			return nil, err
		}
		tuple[index] = variant
		m.symbolToIndex[variant] = uint32(index)
	}
	primary := tuple[0]
	for _, variant := range tuple {
		m.symbolToPrim[variant] = primary
	}
	if err := m.pin(prefix); err != nil {
		return nil, err
	}
	m.primaryToAll[primary] = tuple
	m.prefixToPrim[prefix] = primary
	m.primToPrefix[primary] = prefix
	m.primary = append(m.primary, primary)
	return append([]ast.ASTHandle(nil), tuple...), nil
}

func (m *SymMux) ensureTupleSize(primary ast.ASTHandle, size uint32) error {
	tuple, found := m.primaryToAll[primary]
	if !found || len(tuple) == 0 || tuple[0] != primary {
		return ast.ErrASTDomain
	}
	prefix := m.primToPrefix[primary]
	for uint32(len(tuple)) < size {
		index := uint32(len(tuple))
		variant, err := m.createVariant(prefix, index)
		if err != nil {
			return err
		}
		if err = m.pin(variant); err != nil {
			return err
		}
		tuple = append(tuple, variant)
		m.symbolToIndex[variant] = index
		m.symbolToPrim[variant] = primary
	}
	m.primaryToAll[primary] = tuple
	return nil
}

func (m *SymMux) IsMuxed(declaration ast.ASTHandle) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	_, found := m.symbolToIndex[declaration]
	return found, nil
}

func (m *SymMux) Index(declaration ast.ASTHandle) (uint32, bool, error) {
	if err := m.open(); err != nil {
		return 0, false, err
	}
	index, found := m.symbolToIndex[declaration]
	return index, found, nil
}

func (m *SymMux) Primary(declaration ast.ASTHandle) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	primary, found := m.symbolToPrim[declaration]
	if !found {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return primary, nil
}

func (m *SymMux) ByPrefix(prefix ast.ASTHandle, index uint32) (ast.ASTHandle, bool, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, false, err
	}
	primary, found := m.prefixToPrim[prefix]
	if !found {
		return ast.ASTHandle{}, false, nil
	}
	value, err := m.ConvertDeclaration(primary, 0, index)
	return value, err == nil, err
}

func (m *SymMux) ConvertDeclaration(declaration ast.ASTHandle, source, target uint32) (ast.ASTHandle, error) {
	if err := m.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if source == target {
		return declaration, nil
	}
	actual, found := m.symbolToIndex[declaration]
	if !found || actual != source {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	primary := declaration
	if source != 0 {
		primary = m.symbolToPrim[declaration]
	}
	if err := m.ensureTupleSize(primary, target+1); err != nil {
		return ast.ASTHandle{}, err
	}
	return m.primaryToAll[primary][target], nil
}

func (m *SymMux) rewriteFormula(root ast.ASTHandle, source, target uint32, homogeneous bool, shift *int) (ast.ASTHandle, error) {
	memo := make(map[ast.ASTHandle]ast.ASTHandle)
	var rewrite func(ast.ASTHandle) (ast.ASTHandle, error)
	rewrite = func(current ast.ASTHandle) (ast.ASTHandle, error) {
		if value, found := memo[current]; found {
			return value, nil
		}
		kind, err := m.manager.GetKind(current)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		result := current
		switch kind {
		case ast.ASTApp:
			declaration, err := m.manager.GetDecl(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			newDeclaration := declaration
			if index, muxed := m.symbolToIndex[declaration]; muxed {
				if shift != nil {
					next := int(index) + *shift
					if next < 0 {
						return ast.ASTHandle{}, ast.ErrASTDomain
					}
					newDeclaration, err = m.ConvertDeclaration(declaration, index, uint32(next))
				} else if index == source {
					newDeclaration, err = m.ConvertDeclaration(declaration, source, target)
				} else if homogeneous {
					return ast.ASTHandle{}, ast.ErrASTDomain
				}
				if err != nil {
					return ast.ASTHandle{}, err
				}
			}
			count, err := m.manager.GetNumArgs(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			arguments := make([]ast.ASTHandle, count)
			changed := newDeclaration != declaration
			for index := uint32(0); index < count; index++ {
				argument, err := m.manager.GetArg(current, index)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				arguments[index], err = rewrite(argument)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				changed = changed || arguments[index] != argument
			}
			if changed {
				result, err = m.manager.MkApp(newDeclaration, arguments)
				if err != nil {
					return ast.ASTHandle{}, err
				}
			}
		case ast.ASTQuantifier:
			body, err := m.manager.GetQuantifierBody(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			newBody, err := rewrite(body)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			patternCount, err := m.manager.GetNumPatterns(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			patterns := make([]ast.ASTHandle, patternCount)
			changed := newBody != body
			for index := uint32(0); index < patternCount; index++ {
				old, err := m.manager.GetPattern(current, index)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				patterns[index], err = rewrite(old)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				changed = changed || patterns[index] != old
			}
			noPatternCount, err := m.manager.GetNumNoPatterns(current)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			noPatterns := make([]ast.ASTHandle, noPatternCount)
			for index := uint32(0); index < noPatternCount; index++ {
				old, err := m.manager.GetNoPattern(current, index)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				noPatterns[index], err = rewrite(old)
				if err != nil {
					return ast.ASTHandle{}, err
				}
				changed = changed || noPatterns[index] != old
			}
			if changed {
				result, err = m.manager.UpdateQuantifierAnnotations(current, patterns, noPatterns, newBody)
				if err != nil {
					return ast.ASTHandle{}, err
				}
			}
		}
		memo[current] = result
		return result, nil
	}
	return rewrite(root)
}

func (m *SymMux) ConvertFormula(formula ast.ASTHandle, source, target uint32, homogeneous bool) (ast.ASTRef, error) {
	if err := m.open(); err != nil {
		return ast.ASTRef{}, err
	}
	converted, err := m.rewriteFormula(formula, source, target, homogeneous, nil)
	if err != nil {
		return ast.ASTRef{}, err
	}
	return m.manager.NewRef(converted)
}

func (m *SymMux) ShiftFormula(formula ast.ASTHandle, distance int) (ast.ASTRef, error) {
	if err := m.open(); err != nil {
		return ast.ASTRef{}, err
	}
	converted, err := m.rewriteFormula(formula, 0, 0, false, &distance)
	if err != nil {
		return ast.ASTRef{}, err
	}
	return m.manager.NewRef(converted)
}

func (m *SymMux) Contains(formula ast.ASTHandle, wanted uint32) (bool, error) {
	if err := m.open(); err != nil {
		return false, err
	}
	seen := make(map[ast.ASTHandle]bool)
	stack := []ast.ASTHandle{formula}
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[current] {
			continue
		}
		seen[current] = true
		kind, err := m.manager.GetKind(current)
		if err != nil {
			return false, err
		}
		if kind == ast.ASTApp {
			declaration, err := m.manager.GetDecl(current)
			if err != nil {
				return false, err
			}
			if index, found := m.symbolToIndex[declaration]; found && index == wanted {
				return true, nil
			}
		}
		if kind == ast.ASTQuantifier {
			body, err := m.manager.GetQuantifierBody(current)
			if err != nil {
				return false, err
			}
			stack = append(stack, body)
			patternCount, err := m.manager.GetNumPatterns(current)
			if err != nil {
				return false, err
			}
			for index := uint32(0); index < patternCount; index++ {
				pattern, err := m.manager.GetPattern(current, index)
				if err != nil {
					return false, err
				}
				stack = append(stack, pattern)
			}
			noPatternCount, err := m.manager.GetNumNoPatterns(current)
			if err != nil {
				return false, err
			}
			for index := uint32(0); index < noPatternCount; index++ {
				pattern, err := m.manager.GetNoPattern(current, index)
				if err != nil {
					return false, err
				}
				stack = append(stack, pattern)
			}
		}
		if kind == ast.ASTApp {
			count, err := m.manager.GetNumArgs(current)
			if err != nil {
				return false, err
			}
			for index := uint32(0); index < count; index++ {
				argument, err := m.manager.GetArg(current, index)
				if err != nil {
					return false, err
				}
				stack = append(stack, argument)
			}
		}
	}
	return false, nil
}

func (m *SymMux) MarkNonModel(declaration ast.ASTHandle) error {
	primary, err := m.Primary(declaration)
	if err == nil {
		m.nonModel[primary] = true
	}
	return err
}

func (m *SymMux) Close() (result error) {
	if m == nil || m.manager == nil {
		return ErrSymMuxOwner
	}
	if m.closed {
		return nil
	}
	for index := len(m.references) - 1; index >= 0; index-- {
		result = errors.Join(result, m.manager.DelRef(&m.references[index]))
	}
	if result == nil {
		m.references = nil
		m.primaryToAll = nil
		m.symbolToIndex = nil
		m.symbolToPrim = nil
		m.prefixToPrim = nil
		m.primToPrefix = nil
		m.nonModel = nil
		m.manager = nil
		m.closed = true
	}
	return result
}
