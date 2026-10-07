// Source: src/sat/sat_var_queue.h; src/util/heap.h and vector.h.
package sat

import (
	"github.com/glycerine/g3/pkg/util"
)

// SAT's activity/literal specializations do not bind a utility IntHeap owner.
// Their vector access still runs the actual physical and source assertion guards.
func (q *satVarQueue) assertionsRuntime() (bool, error) { return satStorageSourceDebug, nil }

// The conditional heap namespace has no DebugContext binding in this specialization.
func (q *satVarQueue) cassertRuntimeAt(line int32) error { return nil }
func (q *satVarQueue) assertionFailureRuntime(file string, line int32, condition string) error {
	return util.DebugAssertionFailure(nil, file, line, condition)
}
func (q *satVarQueue) readUintRuntime(c *RuntimeUintColumn, i uint32) (uint32, error) {
	return c.Read(i)
}
func (q *satVarQueue) writeUintRuntime(c *RuntimeUintColumn, i, v uint32) error { return c.Write(i, v) }
func (q *satVarQueue) pushValuesRuntime(v uint32) error                         { return q.runtimeValues.Push(v) }
func (q *satVarQueue) resetIndexRuntime(i uint32) error                         { return q.runtimeIndices.Write(i, 0) }
func (q *satVarQueue) resetValuesRuntime() error                                { return q.runtimeValues.Shrink(0) }
