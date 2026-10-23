// Source: src/smt/smt_model_finder.cpp auf_solver node indexes and f_var.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	protomodel "github.com/glycerine/g3/pkg/smtcore/proto_model"
)

type smtModelFinderAUFKey struct {
	owner ast.ASTHandle
	index uint32
}

type smtModelFinderFVar struct {
	quantifier, function ast.ASTHandle
	position, variable   uint32
	offset               ast.ASTHandle
}

type smtModelFinderAUFGraph struct {
	finder    *SMTModelFinder
	nodes     []*smtModelFinderAUFNode
	uvars     map[smtModelFinderAUFKey]*smtModelFinderAUFNode
	arguments map[smtModelFinderAUFKey]*smtModelFinderAUFNode
	fvars     []smtModelFinderFVar
	literals  []smtModelFinderAUFLiteral
	refs      []ast.ASTRef
	model     *protomodel.ProtoModel
	selects   []smtModelFinderSelectVar
	witnesses map[ast.ASTHandle]ast.ASTHandle
}

func newSMTModelFinderAUFGraph(finder *SMTModelFinder) *smtModelFinderAUFGraph {
	return &smtModelFinderAUFGraph{
		finder: finder, uvars: make(map[smtModelFinderAUFKey]*smtModelFinderAUFNode),
		arguments: make(map[smtModelFinderAUFKey]*smtModelFinderAUFNode),
	}
}

func (g *smtModelFinderAUFGraph) node(index map[smtModelFinderAUFKey]*smtModelFinderAUFNode, key smtModelFinderAUFKey, sort ast.ASTHandle) (*smtModelFinderAUFNode, error) {
	if node := index[key]; node != nil {
		if node.sort != sort {
			return nil, ast.ErrASTDomain
		}
		return node, nil
	}
	node, err := newSMTModelFinderAUFNode(uint32(len(g.nodes)), sort)
	if err != nil {
		return nil, err
	}
	index[key] = node
	g.nodes = append(g.nodes, node)
	return node, nil
}

// Variable indexes are de Bruijn indexes; the manager stores binder sorts in
// declaration order, as does the public model-finder instantiation-set API.
func (g *smtModelFinderAUFGraph) uvar(quantifier ast.ASTHandle, variable uint32) (*smtModelFinderAUFNode, error) {
	flat := g.finder.flatQuantifier(quantifier)
	count, err := g.finder.manager.GetNumDeclarations(flat)
	if err != nil || variable >= count {
		return nil, errors.Join(err, ast.ErrASTDomain)
	}
	sort, err := g.finder.manager.GetDeclarationSort(flat, count-variable-1)
	if err != nil {
		return nil, err
	}
	return g.node(g.uvars, smtModelFinderAUFKey{quantifier, variable}, sort)
}

func (g *smtModelFinderAUFGraph) argument(function ast.ASTHandle, position uint32) (*smtModelFinderAUFNode, error) {
	sort, err := g.finder.manager.GetDomain(function, position)
	if err != nil {
		return nil, err
	}
	return g.node(g.arguments, smtModelFinderAUFKey{function, position}, sort)
}

func (g *smtModelFinderAUFGraph) collectFVars(quantifier ast.ASTHandle) error {
	body, err := g.finder.manager.GetQuantifierBody(g.finder.flatQuantifier(quantifier))
	if err != nil {
		return err
	}
	stack := []ast.ASTHandle{body}
	seen := make(map[ast.ASTHandle]bool)
	for len(stack) != 0 {
		term := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[term] {
			continue
		}
		seen[term] = true
		kind, err := g.finder.manager.GetKind(term)
		if err != nil {
			return err
		}
		// Nested binders require the quantifier-flattening stage before they
		// can contribute qinfo for this quantifier's variables.
		if kind != ast.ASTApp {
			continue
		}
		family, err := g.finder.manager.GetFamilyID(term)
		if err != nil {
			return err
		}
		function, err := g.finder.manager.GetDecl(term)
		if err != nil {
			return err
		}
		if err := g.collectSelectVars(quantifier, term); err != nil {
			return err
		}
		count, err := g.finder.manager.GetNumArgs(term)
		if err != nil {
			return err
		}
		for position := uint32(0); position < count; position++ {
			arg, err := g.finder.manager.GetArg(term, position)
			if err != nil {
				return err
			}
			argKind, err := g.finder.manager.GetKind(arg)
			if err != nil {
				return err
			}
			if family == ast.NullFamilyID && argKind == ast.ASTVar {
				variable, err := g.finder.manager.GetVarIndex(arg)
				if err != nil {
					return err
				}
				g.fvars = append(g.fvars, smtModelFinderFVar{quantifier: quantifier, function: function, position: position, variable: variable})
			} else if family == ast.NullFamilyID {
				if err := g.collectOffset(quantifier, function, position, arg); err != nil {
					return err
				}
			}
			stack = append(stack, arg)
		}
	}
	return nil
}

func (g *smtModelFinderAUFGraph) build(quantifiers []ast.ASTHandle) error {
	for _, quantifier := range quantifiers {
		count, err := g.finder.manager.GetNumDeclarations(g.finder.flatQuantifier(quantifier))
		if err != nil {
			return err
		}
		for variable := uint32(0); variable < count; variable++ {
			if _, err := g.uvar(quantifier, variable); err != nil {
				return err
			}
		}
		if err := g.collectFVars(quantifier); err != nil {
			return err
		}
		body, err := g.finder.manager.GetQuantifierBody(g.finder.flatQuantifier(quantifier))
		if err != nil {
			return err
		}
		if err := g.collectLiterals(quantifier, body, false); err != nil {
			return err
		}
	}
	// All equalities must be processed before allocating root sets.
	for _, info := range g.fvars {
		argument, err := g.argument(info.function, info.position)
		if err != nil {
			return err
		}
		variable, err := g.uvar(info.quantifier, info.variable)
		if err != nil {
			return err
		}
		if info.offset.IsNull() {
			if err = argument.merge(variable); err != nil {
				return err
			}
		}
	}
	if err := g.processLiterals(); err != nil {
		return err
	}
	if err := g.processSelectVars(); err != nil {
		return err
	}
	for _, node := range g.nodes {
		if node.parent == nil {
			if err := node.makeInstantiationSet(g.finder.manager); err != nil {
				return err
			}
		}
	}
	context := g.finder.state.context
	for _, info := range g.fvars {
		node := g.arguments[smtModelFinderAUFKey{info.function, info.position}]
		for _, application := range context.byDeclaration[info.function] {
			if application == nil || !context.relevant[application.owner] {
				continue
			}
			if info.position >= uint32(len(application.arguments)) {
				return ast.ErrASTDomain
			}
			// Source f_var explicitly requires the original argument enode.
			// Its class representative can hide necessary MBQI instances.
			argument := application.arguments[info.position]
			if !info.offset.IsNull() && node.root() == g.uvars[smtModelFinderAUFKey{info.quantifier, info.variable}].root() {
				if err := g.insertShifted(node, argument.owner, info.offset, argument.Generation(), false); err != nil {
					return err
				}
				continue
			}
			if _, err := node.insert(argument.owner, argument.Generation()); err != nil {
				return err
			}
		}
	}
	if err := g.populateLiterals(); err != nil {
		return err
	}
	if err := g.populateSelectVars(); err != nil {
		return err
	}
	return g.populateOffsets()
}

func (g *smtModelFinderAUFGraph) publishInstantiationSets() error {
	for key, node := range g.uvars {
		count, err := g.finder.manager.GetNumDeclarations(g.finder.flatQuantifier(key.owner))
		if err != nil {
			return err
		}
		originalCount, err := g.finder.manager.GetNumDeclarations(key.owner)
		if err != nil {
			return err
		}
		if count-key.index-1 >= originalCount {
			continue
		}
		for term, generation := range node.root().set.elements {
			if _, err := g.finder.InsertInstantiationTerm(key.owner, count-key.index-1, term, generation); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *smtModelFinderAUFGraph) close() (result error) {
	for _, node := range g.nodes {
		result = errors.Join(result, node.close())
	}
	for i := range g.refs {
		result = errors.Join(result, g.finder.manager.DelRef(&g.refs[i]))
	}
	g.refs = nil
	return result
}

func (f *SMTModelFinder) populateAUFInstantiationSets(quantifiers []ast.ASTHandle, model *protomodel.ProtoModel) (result error) {
	graph := newSMTModelFinderAUFGraph(f)
	graph.model = model
	defer func() { result = errors.Join(result, graph.close()) }()
	if err := graph.build(quantifiers); err != nil {
		return err
	}
	if err := graph.extendModel(); err != nil {
		return err
	}
	return graph.publishInstantiationSets()
}
