// Source: src/util/ref_vector.h:27-359.
package util

// SelfRefCounted is the Go form of an object accepted by ref_unmanaged_wrapper.
type SelfRefCounted interface {
	IncRef()
	DecRef()
}

// RefManagerWrapper adapts a manager and its reference callbacks.
type RefManagerWrapper struct {
	Manager any
	Inc     func(any)
	Dec     func(any)
}

func (wrapper RefManagerWrapper) increment(value any) {
	if wrapper.Inc == nil {
		panic("g3: reference manager requires an increment callback")
	}
	wrapper.Inc(value)
}

func (wrapper RefManagerWrapper) decrement(value any) {
	if wrapper.Dec == nil {
		panic("g3: reference manager requires a decrement callback")
	}
	wrapper.Dec(value)
}

// RefVectorCore owns one reference for each pointer in Nodes.
type RefVectorCore struct {
	self       *RefVectorCore
	manager    RefManagerWrapper
	nodes      []any
	generation uint64
	closed     bool
}

// NewRefVectorCore constructs an empty reference vector core.
func NewRefVectorCore(manager RefManagerWrapper) *RefVectorCore {
	if manager.Inc == nil || manager.Dec == nil {
		panic("g3: reference vector requires increment and decrement callbacks")
	}
	vector := &RefVectorCore{manager: manager}
	vector.self = vector
	return vector
}

// Move transfers the stored references to a new core and leaves vector empty
// and reusable.
func (vector *RefVectorCore) Move() *RefVectorCore {
	vector.requireOpen()
	moved := NewRefVectorCore(vector.manager)
	moved.nodes, vector.nodes = vector.nodes, nil
	vector.changed()
	return moved
}

func (vector *RefVectorCore) requireOpen() {
	if vector == nil || vector.self != vector || vector.closed {
		panic("g3: invalid reference vector owner")
	}
}

func (vector *RefVectorCore) requireIndex(index uint32) {
	if uint64(index) >= uint64(len(vector.nodes)) {
		panic("g3: reference vector index out of bounds")
	}
}

func (vector *RefVectorCore) requireValue(value any) {
	if !validScopedPointer(value) {
		panic("g3: reference vector requires pointer values")
	}
}

func (vector *RefVectorCore) changed() { vector.generation++ }

func (vector *RefVectorCore) decRange(begin, end int) {
	for i := begin; i < end; i++ {
		vector.manager.decrement(vector.nodes[i])
	}
}

// Reset decrements all values in order and retains storage.
func (vector *RefVectorCore) Reset() {
	vector.requireOpen()
	vector.decRange(0, len(vector.nodes))
	clear(vector.nodes)
	vector.nodes = vector.nodes[:0]
	vector.changed()
}

// Finalize decrements all values in order and releases storage.
func (vector *RefVectorCore) Finalize() {
	vector.requireOpen()
	vector.decRange(0, len(vector.nodes))
	clear(vector.nodes)
	vector.nodes = nil
	vector.changed()
}

// Resize shrinks with decrement callbacks or grows with nil slots without
// increment callbacks.
func (vector *RefVectorCore) Resize(size uint32) {
	vector.requireOpen()
	target := int(size)
	if target < len(vector.nodes) {
		vector.decRange(target, len(vector.nodes))
		clear(vector.nodes[target:])
		vector.nodes = vector.nodes[:target]
		vector.changed()
		return
	}
	if target > len(vector.nodes) {
		vector.nodes = append(vector.nodes, make([]any, target-len(vector.nodes))...)
		vector.changed()
	}
}

// ResizeFill shrinks in order or grows by acquiring one reference per new slot.
func (vector *RefVectorCore) ResizeFill(size uint32, value any) {
	vector.requireOpen()
	vector.requireValue(value)
	if int(size) < len(vector.nodes) {
		vector.Shrink(size)
		return
	}
	for uint32(len(vector.nodes)) < size {
		vector.PushBack(value)
	}
}

// Reserve preserves the source method's behavior: it grows the logical vector
// with nil slots rather than only reserving capacity.
func (vector *RefVectorCore) Reserve(size uint32) {
	vector.requireOpen()
	if uint64(size) > uint64(len(vector.nodes)) {
		vector.Resize(size)
	}
}

func (vector *RefVectorCore) Shrink(size uint32) {
	vector.requireOpen()
	if uint64(size) > uint64(len(vector.nodes)) {
		panic("g3: reference vector shrink grows vector")
	}
	target := int(size)
	vector.decRange(target, len(vector.nodes))
	clear(vector.nodes[target:])
	vector.nodes = vector.nodes[:target]
	vector.changed()
}

func (vector *RefVectorCore) PushBack(value any) *RefVectorCore {
	vector.requireOpen()
	vector.requireValue(value)
	vector.manager.increment(value)
	vector.nodes = append(vector.nodes, value)
	vector.changed()
	return vector
}

func (vector *RefVectorCore) PopBack() {
	vector.requireOpen()
	if len(vector.nodes) == 0 {
		panic("g3: reference vector pop from empty vector")
	}
	last := len(vector.nodes) - 1
	value := vector.nodes[last]
	vector.nodes[last] = nil
	vector.nodes = vector.nodes[:last]
	vector.changed()
	vector.manager.decrement(value)
}

func (vector *RefVectorCore) Back() any {
	vector.requireOpen()
	if len(vector.nodes) == 0 {
		panic("g3: reference vector back from empty vector")
	}
	return vector.nodes[len(vector.nodes)-1]
}

func (vector *RefVectorCore) Size() uint32 { vector.requireOpen(); return uint32(len(vector.nodes)) }
func (vector *RefVectorCore) Empty() bool  { return vector.Size() == 0 }

func (vector *RefVectorCore) Get(index uint32) any {
	vector.requireOpen()
	vector.requireIndex(index)
	return vector.nodes[index]
}

func (vector *RefVectorCore) GetDefault(index uint32, fallback any) any {
	vector.requireOpen()
	if uint64(index) >= uint64(len(vector.nodes)) {
		return fallback
	}
	return vector.nodes[index]
}

// CPtr returns a live view that is invalidated by structural edits.
func (vector *RefVectorCore) CPtr() []any { vector.requireOpen(); return vector.nodes }

func (vector *RefVectorCore) Set(index uint32, value any) {
	vector.requireOpen()
	vector.requireIndex(index)
	vector.requireValue(value)
	vector.manager.increment(value)
	vector.manager.decrement(vector.nodes[index])
	vector.nodes[index] = value
}

func (vector *RefVectorCore) EraseIndex(index uint32) {
	vector.requireOpen()
	vector.requireIndex(index)
	value := vector.nodes[index]
	copy(vector.nodes[index:], vector.nodes[index+1:])
	last := len(vector.nodes) - 1
	vector.nodes[last] = nil
	vector.nodes = vector.nodes[:last]
	vector.changed()
	vector.manager.decrement(value)
}

func (vector *RefVectorCore) EraseValue(value any) bool {
	vector.requireOpen()
	vector.requireValue(value)
	for index, candidate := range vector.nodes {
		if sameScopedPointer(candidate, value) {
			vector.EraseIndex(uint32(index))
			return true
		}
	}
	return false
}

func (vector *RefVectorCore) Contains(value any) bool {
	vector.requireOpen()
	vector.requireValue(value)
	for _, candidate := range vector.nodes {
		if sameScopedPointer(candidate, value) {
			return true
		}
	}
	return false
}

func (vector *RefVectorCore) Append(other *RefVectorCore) {
	vector.requireOpen()
	other.requireOpen()
	if vector == other {
		panic("g3: reference vector cannot append itself")
	}
	for _, value := range other.nodes {
		vector.PushBack(value)
	}
}

func (vector *RefVectorCore) AppendValues(values []any) {
	vector.requireOpen()
	for _, value := range values {
		vector.PushBack(value)
	}
}

// Begin returns an iterator at the first slot.
func (vector *RefVectorCore) Begin() RefVectorIterator {
	vector.requireOpen()
	return RefVectorIterator{vector: vector, generation: vector.generation}
}

// End returns the past-the-end iterator.
func (vector *RefVectorCore) End() RefVectorIterator {
	vector.requireOpen()
	return RefVectorIterator{vector: vector, index: uint32(len(vector.nodes)), generation: vector.generation}
}

func (vector *RefVectorCore) SwapIndices(first, second uint32) {
	vector.requireOpen()
	vector.requireIndex(first)
	vector.requireIndex(second)
	vector.nodes[first], vector.nodes[second] = vector.nodes[second], vector.nodes[first]
}

func (vector *RefVectorCore) Reverse() {
	vector.requireOpen()
	for left, right := 0, len(vector.nodes)-1; left < right; left, right = left+1, right-1 {
		vector.nodes[left], vector.nodes[right] = vector.nodes[right], vector.nodes[left]
	}
}

func (vector *RefVectorCore) Close() {
	vector.requireOpen()
	vector.Finalize()
	vector.closed = true
	vector.manager = RefManagerWrapper{}
}

// RefVectorIterator is a checked iterator over a stable vector layout.
type RefVectorIterator struct {
	vector     *RefVectorCore
	index      uint32
	generation uint64
}

func (iterator RefVectorIterator) requireLive(allowEnd bool) {
	iterator.vector.requireOpen()
	if iterator.generation != iterator.vector.generation {
		panic("g3: stale reference vector iterator")
	}
	limit := uint32(len(iterator.vector.nodes))
	if iterator.index > limit || !allowEnd && iterator.index == limit {
		panic("g3: reference vector iterator out of bounds")
	}
}

func (iterator RefVectorIterator) Get() any {
	iterator.requireLive(false)
	return iterator.vector.nodes[iterator.index]
}

func (iterator RefVectorIterator) Next() RefVectorIterator {
	iterator.requireLive(false)
	iterator.index++
	return iterator
}

func (iterator RefVectorIterator) Equal(other RefVectorIterator) bool {
	iterator.requireLive(true)
	other.requireLive(true)
	return iterator.vector == other.vector && iterator.index == other.index
}

// RefVector is a manager-bound reference vector.
type RefVector struct {
	*RefVectorCore
	managerIdentity any
}

func NewRefVector(manager any, inc, dec func(any)) *RefVector {
	if !validScopedPointer(manager) || nilScopedPointer(manager) {
		panic("g3: reference vector manager must be a non-nil pointer")
	}
	wrapper := RefManagerWrapper{Manager: manager, Inc: inc, Dec: dec}
	return &RefVector{RefVectorCore: NewRefVectorCore(wrapper), managerIdentity: manager}
}

func NewRefVectorFrom(manager any, inc, dec func(any), values []any) *RefVector {
	vector := NewRefVector(manager, inc, dec)
	vector.AppendValues(values)
	return vector
}

func (vector *RefVector) Manager() any    { vector.requireOpen(); return vector.managerIdentity }
func (vector *RefVector) GetManager() any { return vector.Manager() }
func (vector *RefVector) M() any          { return vector.Manager() }

func (vector *RefVector) Clone() *RefVector {
	vector.requireOpen()
	clone := NewRefVector(vector.managerIdentity, vector.manager.Inc, vector.manager.Dec)
	clone.Append(vector.RefVectorCore)
	return clone
}

// Move transfers all stored references and leaves vector empty and reusable.
func (vector *RefVector) Move() *RefVector {
	vector.requireOpen()
	return &RefVector{RefVectorCore: vector.RefVectorCore.Move(), managerIdentity: vector.managerIdentity}
}

func (vector *RefVector) Assign(other *RefVector) *RefVector {
	vector.requireOpen()
	other.requireOpen()
	if vector != other {
		vector.Reset()
		vector.Append(other.RefVectorCore)
	}
	return vector
}

func (vector *RefVector) Swap(other *RefVector) {
	vector.requireOpen()
	other.requireOpen()
	if !sameScopedPointer(vector.managerIdentity, other.managerIdentity) {
		panic("g3: reference vector swap requires identical managers")
	}
	vector.nodes, other.nodes = other.nodes, vector.nodes
	vector.changed()
	other.changed()
}

func (vector *RefVector) Element(index uint32) RefVectorElement {
	vector.requireOpen()
	vector.requireIndex(index)
	return RefVectorElement{vector: vector.RefVectorCore, index: index, generation: vector.generation}
}

// RefVectorElement is the checked Go form of ref_vector::element_ref.
type RefVectorElement struct {
	vector     *RefVectorCore
	index      uint32
	generation uint64
}

func (element RefVectorElement) requireLive() {
	element.vector.requireOpen()
	if element.generation != element.vector.generation {
		panic("g3: stale reference vector element")
	}
	element.vector.requireIndex(element.index)
}

func (element RefVectorElement) Get() any {
	element.requireLive()
	return element.vector.nodes[element.index]
}

func (element RefVectorElement) Set(value any) {
	element.requireLive()
	if nilScopedPointer(value) {
		panic("g3: reference vector element assignment requires non-nil pointer")
	}
	element.vector.Set(element.index, value)
}

func (element RefVectorElement) Assign(other RefVectorElement) { element.Set(other.Get()) }

func (element RefVectorElement) Equal(value any) bool {
	element.requireLive()
	element.vector.requireValue(value)
	return sameScopedPointer(element.vector.nodes[element.index], value)
}

// RefUnmanagedWrapper invokes reference operations on the object itself.
type RefUnmanagedWrapper struct{}

func (RefUnmanagedWrapper) IncRef(value any) {
	if !nilScopedPointer(value) {
		value.(SelfRefCounted).IncRef()
	}
}

func (RefUnmanagedWrapper) DecRef(value any) {
	if !nilScopedPointer(value) {
		value.(SelfRefCounted).DecRef()
	}
}

// SRefVector owns objects that perform their own reference counting.
type SRefVector struct{ *RefVectorCore }

func NewSRefVector() *SRefVector {
	wrapper := RefUnmanagedWrapper{}
	inc := wrapper.IncRef
	dec := wrapper.DecRef
	return &SRefVector{RefVectorCore: NewRefVectorCore(RefManagerWrapper{Inc: inc, Dec: dec})}
}

// RefVectorPtrHasher is the Go form of ref_vector_ptr_hash.
type RefVectorPtrHasher struct{ GetID func(any) uint32 }

func (hasher RefVectorPtrHasher) Hash(vector *RefVector) uint32 {
	return RefVectorPtrHash(vector, hasher.GetID)
}

// RefVectorPtrEquality is the Go form of ref_vector_ptr_eq.
type RefVectorPtrEquality struct{}

func (RefVectorPtrEquality) Equal(left, right *RefVector) bool {
	return RefVectorPtrEqual(left, right)
}

// RefVectorPtrHash preserves ref_vector_ptr_hash using object IDs supplied by
// the concrete element type.
func RefVectorPtrHash(vector *RefVector, getID func(any) uint32) uint32 {
	if vector == nil {
		return 0
	}
	vector.requireOpen()
	if len(vector.nodes) == 0 {
		return 0
	}
	if getID == nil {
		panic("g3: reference vector hash requires an ID callback")
	}
	return Composite(17, uint32(len(vector.nodes)), func(index uint32) uint32 {
		return getID(vector.nodes[index])
	})
}

// RefVectorPtrEqual compares vector pointers and element pointer identities.
func RefVectorPtrEqual(left, right *RefVector) bool {
	if left == nil || right == nil {
		return left == right
	}
	left.requireOpen()
	right.requireOpen()
	if len(left.nodes) != len(right.nodes) {
		return false
	}
	for index := range left.nodes {
		if !sameScopedPointer(left.nodes[index], right.nodes[index]) {
			return false
		}
	}
	return true
}
