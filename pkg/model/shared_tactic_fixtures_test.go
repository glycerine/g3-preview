package model_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func goalTestAtom(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	h := proofTestBool(t, m, name)
	astTestPin(t, m, h)
	return h
}

func goalTestNew(t *testing.T, m *ast.ASTManager, proofs, cores bool) *tactic.Goal {
	t.Helper()
	g, e := tactic.NewGoal(m, proofs, true, cores)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return g
}
func goalTestApp(t *testing.T, m *ast.ASTManager, op int32, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(0, op, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

type modelConverterTestLeaf struct {
	name   string
	calls  *[]string
	refs   uint32
	closed bool
	label  usymbol.Symbol
	run    func(*tactic.ModelRef, uint32) error
	fail   error
}

func (c *modelConverterTestLeaf) IncRef() error {
	if c.closed {
		return tactic.ErrModelConverterOwner
	}
	c.refs++
	return nil
}
func (c *modelConverterTestLeaf) DecRef() error {
	_, e := c.DecRefConsumed()
	return e
}
func (c *modelConverterTestLeaf) DecRefConsumed() (bool, error) {
	if c.closed || c.refs == 0 {
		return false, tactic.ErrModelConverterOwner
	}
	c.refs--
	if c.refs == 0 {
		return true, c.Close()
	}
	return true, nil
}
func (c *modelConverterTestLeaf) Close() error {
	if !c.closed {
		c.closed = true
		*c.calls = append(*c.calls, "destroy:"+c.name)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyModel(m *tactic.ModelRef, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("model:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, i)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyUnindexed(m *tactic.ModelRef) error {
	*c.calls = append(*c.calls, "unindexed:"+c.name)
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, 0)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyLabels(l *tactic.ModelLabels, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("labels:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	return l.Append(c.label)
}
func (c *modelConverterTestLeaf) Cancel() error {
	*c.calls = append(*c.calls, "cancel:"+c.name)
	return nil
}
func (c *modelConverterTestLeaf) Display(out io.Writer) error {
	if c.closed || out == nil {
		return tactic.ErrModelConverterOwner
	}
	_, err := io.WriteString(out, "("+c.name+")\n")
	return err
}
func (c *modelConverterTestLeaf) Translate(translator *ast.ASTTranslation) (tactic.ModelConverter, error) {
	if c.closed || translator == nil {
		return nil, tactic.ErrModelConverterOwner
	}
	return &modelConverterTestLeaf{name: c.name, calls: c.calls, label: c.label, run: c.run, fail: c.fail}, nil
}
func proofTestBool(t *testing.T, m *ast.ASTManager, name string) ast.ASTHandle {
	t.Helper()
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkConst(astTestSymbol(t, m, name), s)
	astTestMust(t, e)
	return h
}
