// Source: src/util/memory_manager.cpp:102.
// Licensed under the MIT license; see LICENSE.txt.
// Source author: Nikolaj Bjorner (nbjorner), 2007-07-24.
package memory

// ClearOutOfMemoryForInitialization implements the scalar assignment reached
// after the source first-initialize guard and before generated mem_initialize.
// The containing process binding must supply that control flow. This leaf does
// not initialize children, reset accounting/limits, or publish fully initialized.
func (m *Runtime) ClearOutOfMemoryForInitialization() error {
	if e := m.open(); e != nil {
		return e
	}
	m.accounting.outOfMemory = false
	return nil
}
