package cmd_context

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestCommandContextRecursiveDefinitionsPreserveMarkerPatternsAndIdentity(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	b, err := m.MkBoolSort()
	contextMust(t, err)
	name, err := m.Symbols.Intern("recursive-x")
	contextMust(t, err)
	x, err := m.MkVar(0, b)
	contextMust(t, err)
	identity, err := c.DeclareFunction("recursive-id", []ast.ASTHandle{b}, b)
	contextMust(t, err)
	contextMust(t, c.DefineRecursiveFunction(identity, []usymbol.Symbol{name}, x))
	if !c.containsMacro("recursive-id", []ast.ASTHandle{b}) || len(c.assertions) != 0 {
		t.Fatal("variable body must become the source identity macro")
	}
	notX, err := m.MkNot(x)
	contextMust(t, err)
	f, err := c.DeclareFunction("recursive-not", []ast.ASTHandle{b}, b)
	contextMust(t, err)
	contextMust(t, c.DefineRecursiveFunction(f, []usymbol.Symbol{name}, notX))
	if len(c.assertions) != 1 {
		t.Fatal("recursive proper term did not publish one definition")
	}
	q := c.assertions[0].expression
	recursive, err := m.IsRecursiveFunctionDefinition(q)
	contextMust(t, err)
	count, err := m.GetNumPatterns(q)
	contextMust(t, err)
	weight, err := m.GetQuantifierWeight(q)
	contextMust(t, err)
	if !recursive || count != 2 || weight != 0 {
		t.Fatal("source definition metadata", recursive, count, weight)
	}
	left, err := m.GetPattern(q, 0)
	contextMust(t, err)
	call, err := m.GetArg(left, 0)
	contextMust(t, err)
	declaration, err := m.GetDecl(call)
	contextMust(t, err)
	right, err := m.GetPattern(q, 1)
	contextMust(t, err)
	body, err := m.GetArg(right, 0)
	contextMust(t, err)
	if declaration != f || body != notX {
		t.Fatal("definition lost function/body trigger roles")
	}
}

func TestCommandContextRecursiveDefinitionAllocationFrontiers(t *testing.T) {
	for _, identity := range []bool{false, true} {
		t.Run(fmt.Sprint(identity), func(t *testing.T) {
			succeeded := false
			for stop := int64(0); stop < 256 && !succeeded; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					c := contextFixture(t)
					m, err := c.Manager()
					contextMust(t, err)
					b, err := m.MkBoolSort()
					contextMust(t, err)
					x, err := m.MkVar(0, b)
					contextMust(t, err)
					body := x
					if !identity {
						body, err = m.MkNot(x)
						contextMust(t, err)
					}
					ref, err := m.NewRef(body)
					contextMust(t, err)
					t.Cleanup(func() { contextMust(t, m.DelRef(&ref)) })
					name, err := m.Symbols.Intern("recursive-stopped-x")
					contextMust(t, err)
					f, err := c.DeclareFunction("recursive-stopped-f", []ast.ASTHandle{b}, b)
					contextMust(t, err)
					m.Budget().FailAfter(stop)
					err = c.DefineRecursiveFunction(f, []usymbol.Symbol{name}, body)
					m.Budget().FailAfter(-1)
					if err != nil {
						if !errors.Is(err, arena.ErrInjectedFailure) {
							t.Fatal("unexpected stopped recursive definition error", err)
						}
					} else {
						succeeded = true
						if identity != c.containsMacro("recursive-stopped-f", []ast.ASTHandle{b}) {
							t.Fatal("successful definition took the wrong source branch")
						}
					}
				})
				if t.Failed() {
					t.Fatalf("recursive definition frontier %d", stop)
				}
			}
			if !succeeded {
				t.Fatal("no successful recursive definition allocation frontier")
			}
		})
	}
}
