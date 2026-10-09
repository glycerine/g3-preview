// Source: src/util/memory_manager.cpp:33-72.
package util

// RecordAllocationFailure applies the memory manager's sticky out-of-memory
// state and configured failure transport to storage allocated by a typed Go
// leaf on this Runtime's budget.
func (m *Runtime) RecordAllocationFailure() error {
	if e := m.open(); e != nil {
		return e
	}
	m.accounting.outOfMemory = true
	return m.emitFailure(FailureMemout)
}
