package memory

// ProcessAccounting is the shared source memory_manager.cpp counter and limit
// state for a group of Runtime owners. The caller confines the process and its
// runtimes to one goroutine, as with the individual Runtime owners. Each runtime
// keeps its own allocation lane, including the source TLS synchronization
// threshold; only the global fields are shared.
type ProcessAccounting struct {
	self       *ProcessAccounting
	accounting utilMemoryAccounting
}

func NewProcessAccounting() *ProcessAccounting {
	p := &ProcessAccounting{}
	p.self = p
	return p
}

// Valid reports whether this is the original, initialized process owner.
func (p *ProcessAccounting) Valid() bool { return p != nil && p.self == p }

// AllocationSize is the process-wide source memory::get_allocation_size value.
// It excludes unsynchronized thread-local lane deltas, as the source getter does.
func (p *ProcessAccounting) AllocationSize() uint64 {
	if !p.Valid() {
		return 0
	}
	return p.accounting.allocationSize()
}
