// Source: src/util/memory_manager.cpp:102.
package memory

// ClearOutOfMemoryForInitialization implements the scalar assignment reached
// after the source first-initialize guard and before generated mem_initialize.
// The containing process binding must supply that control flow. This leaf does
// not initialize children, reset accounting/limits, or publish fully initialized.
func (m *Runtime) ClearOutOfMemoryForInitialization() error {
	if e := m.open(); e != nil {
		return e
	}
	m.accountingOwner().outOfMemory = false
	return nil
}
