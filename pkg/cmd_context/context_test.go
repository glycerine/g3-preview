package cmd_context

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func contextMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func contextFixture(t *testing.T) *Context {
	t.Helper()
	c, err := NewStandaloneContext()
	contextMust(t, err)
	budget := c.owned.budget
	t.Cleanup(func() {
		contextMust(t, c.Close())
		if n := budget.Used(); n != 0 {
			t.Errorf("command context retained %d bytes", n)
		}
	})
	return c
}

func TestCommandContextDefinitionOwnershipAndScopes(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	b, err := c.LookupSort("Bool")
	contextMust(t, err)
	truth, err := m.MkTrue()
	contextMust(t, err)
	contextMust(t, c.DefineFunction("choose", []ast.ASTHandle{b}, truth))
	out, err := m.NewRef(ast.ASTHandle{})
	contextMust(t, err)
	defer func() { contextMust(t, m.DelRef(&out)) }()
	contextMust(t, c.MakeAppInto("choose", []ast.ASTHandle{truth}, nil, ast.ASTHandle{}, out))
	contextMust(t, c.Push(1))
	u, err := c.DeclareSort("UnusedParameter")
	contextMust(t, err)
	// The body does not mention its parameter's sort.
	contextMust(t, c.DefineFunction("choose", []ast.ASTHandle{u}, truth))
	if err := c.DefineFunction("choose", []ast.ASTHandle{u}, truth); err == nil {
		t.Fatal("duplicate macro accepted")
	}
	if _, err := c.DeclareFunction("choose", []ast.ASTHandle{u}, b); err == nil {
		t.Fatal("function/macro domain collision accepted")
	}
	contextMust(t, c.DefineFunction("temporary", nil, truth))
	contextMust(t, c.MakeAppInto("temporary", nil, nil, ast.ASTHandle{}, out))
	contextMust(t, c.Pop(1))
	value, err := m.RefGet(out)
	contextMust(t, err)
	if value != truth {
		t.Fatal("substitution output lost after scope cleanup")
	}
	if err := c.MakeAppInto("temporary", nil, nil, ast.ASTHandle{}, out); err == nil {
		t.Fatal("scoped macro survived pop")
	}
	contextMust(t, c.MakeAppInto("choose", []ast.ASTHandle{truth}, nil, ast.ASTHandle{}, out))
}

func TestCommandContextScopesDeclarationsAndModel(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_UF"))
	b, err := c.LookupSort("Bool")
	contextMust(t, err)
	_, err = c.DeclareFunction("p", nil, b)
	contextMust(t, err)
	p, err := c.MakeApp("p", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(p, ""))
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	var values bytes.Buffer
	contextMust(t, c.WriteValues(&values, []ast.ASTHandle{p}))
	if values.String() != "((p true))\n" {
		t.Fatal(values.String())
	}
	contextMust(t, c.Push(1))
	_, err = c.DeclareFunction("local", nil, b)
	contextMust(t, err)
	negative, err := c.manager.MkNot(p)
	contextMust(t, err)
	contextMust(t, c.Assert(negative, ""))
	status, err = c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal(status)
	}
	contextMust(t, c.Pop(1))
	if _, err := c.MakeApp("local", nil, nil, ast.ASTHandle{}); err == nil {
		t.Fatal("local declaration survived pop")
	}
	status, err = c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	if err := c.SetLogic("QF_LIA"); err == nil {
		t.Fatal("logic changed after initialization")
	}
}

func TestCommandContextNamedCoreAndProof(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetBoolOption("produce-proofs", true))
	contextMust(t, c.SetBoolOption("produce-unsat-cores", true))
	b, err := c.LookupSort("Bool")
	contextMust(t, err)
	_, err = c.DeclareFunction("p", nil, b)
	contextMust(t, err)
	p, err := c.MakeApp("p", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	contextMust(t, c.Assert(p, "positive"))
	negative, err := c.manager.MkNot(p)
	contextMust(t, err)
	contextMust(t, c.Assert(negative, "negative"))
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.False {
		t.Fatal(status)
	}
	var out bytes.Buffer
	contextMust(t, c.WriteUnsatCore(&out))
	if !bytes.Contains(out.Bytes(), []byte("positive")) || !bytes.Contains(out.Bytes(), []byte("negative")) {
		t.Fatal(out.String())
	}
	out.Reset()
	contextMust(t, c.WriteProof(&out))
	proof, err := c.solver.Proof()
	contextMust(t, err)
	fact, err := c.manager.GetProofFact(proof)
	contextMust(t, err)
	if !c.manager.IsFalse(fact) {
		t.Fatal("printed proof does not derive false")
	}
	if out.Len() == 0 {
		t.Fatal("missing proof")
	}
}
