package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func cliqueTestInput(t *testing.T, r *Runtime) *RuntimeUintColumn {
	c := &RuntimeUintColumn{}
	c.Bind(r)
	t.Cleanup(func() { memoryRuntimeRequire(t, c.Finalize()) })
	return c
}
func cliqueTestRefusal(t *testing.T, e error) {
	t.Helper()
	if !errors.Is(e, arena.ErrInjectedFailure) {
		var f Failure
		if !errors.As(e, &f) {
			t.Fatal("expected allocation refusal", e)
		}
	}
}

type cliqueTestSpan struct {
	runtime *Runtime
	block   Block
	count   uint32
}

func (s cliqueTestSpan) read(i uint32) (uint32, error) {
	return s.runtime.ReadUint32(s.block, 8+uint64(i)*4)
}
func cliqueTestCapture(o *RuntimeUnsignedRows, i uint32) (cliqueTestSpan, error) {
	c, e := o.Row(i)
	if e != nil {
		return cliqueTestSpan{}, e
	}
	n, e := c.Size()
	return cliqueTestSpan{c.Raw.Runtime, c.Raw.Block, n}, e
}
func cliqueTestAudit(t *testing.T, r *Runtime, g *NegatedLiteralCliques, ps *RuntimeUintColumn, outs ...*RuntimeUnsignedRows) {
	t.Helper()
	roots := g.OwningRoots()
	if ps != nil {
		roots = columnOwningRoots(ps, roots)
	}
	for _, o := range outs {
		roots = append(roots, o.OwningRoots()...)
	}
	var dummy RuntimeVectorColumn
	dummy.Bind(r)
	runtimeVectorAuditBlocks(t, &dummy, roots...)
}

func cliqueTestAppendValues(o *RuntimeUnsignedRows, values []uint32) (err error) {
	c, e := o.BeginRow()
	if e != nil {
		return e
	}
	defer func() { err = o.EndRow(err) }()
	for _, v := range values {
		if err = c.Push(v); err != nil {
			return err
		}
	}
	return nil
}
func cliqueTestRowsAudit(t *testing.T, r *Runtime, o *RuntimeUnsignedRows, cols ...*RuntimeUintColumn) {
	roots := o.OwningRoots()
	for _, c := range cols {
		roots = columnOwningRoots(c, roots)
	}
	var dummy RuntimeVectorColumn
	dummy.Bind(r)
	runtimeVectorAuditBlocks(t, &dummy, roots...)
}
