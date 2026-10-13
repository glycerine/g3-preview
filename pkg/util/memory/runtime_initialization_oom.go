// Source: src/util/memory_manager.cpp:102.
package memory

import "math"

// BeginSourceInitialize executes memory::initialize's maximum-size update and
// first-initialize guard. A true result requires the caller to initialize the
// generated children and then call CompleteSourceInitialize. Concurrent callers
// are outside the single-goroutine port; a reentry during child construction
// returns ErrProcessLifecyclePending instead of spinning forever.
func (m *Runtime) BeginSourceInitialize(maxSize uint64) (bool, error) {
	if e := m.open(); e != nil {
		return false, e
	}
	p := m.config.Process
	if p == nil || !p.Valid() {
		return false, ErrRuntimeOwner
	}
	if p.finalizing || p.initializing {
		return false, ErrProcessLifecyclePending
	}
	// UINT_MAX is the source sentinel even when size_t is 64 bits.
	if maxSize != math.MaxUint32 {
		if e := m.SetMaxSize(maxSize); e != nil {
			return false, e
		}
	}
	if p.initialized {
		return false, nil
	}
	p.initialized = true
	p.initializing = true
	p.accounting.outOfMemory = false
	return true, nil
}

// CompleteSourceInitialize publishes g_memory_fully_initialized only after
// the generated rational, symbol, and global-parameter stages have succeeded.
func (m *Runtime) CompleteSourceInitialize() error {
	if e := m.open(); e != nil {
		return e
	}
	p := m.config.Process
	if p == nil || !p.Valid() {
		return ErrRuntimeOwner
	}
	if !p.initializing || !p.initialized {
		return ErrProcessLifecycleOrder
	}
	p.fullyInitialized = true
	p.initializing = false
	return nil
}

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
