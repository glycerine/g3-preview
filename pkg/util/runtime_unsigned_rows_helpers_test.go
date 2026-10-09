package util

import "testing"

func unsignedRowsTestInput(t *testing.T, r *Runtime) *RuntimeUintColumn {
	c := &RuntimeUintColumn{}
	c.Bind(r)
	t.Cleanup(func() { memoryRuntimeRequire(t, c.Finalize()) })
	return c
}

func unsignedRowsTestAppend(o *RuntimeUnsignedRows, values []uint32) (err error) {
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

func unsignedRowsTestAudit(t *testing.T, r *Runtime, o *RuntimeUnsignedRows, cols ...*RuntimeUintColumn) {
	t.Helper()
	roots := o.OwningRoots()
	for _, c := range cols {
		roots = columnOwningRoots(c, roots)
	}
	var dummy RuntimeVectorColumn
	dummy.Bind(r)
	runtimeVectorAuditBlocks(t, &dummy, roots...)
}
