// Selected LAZY_USE_LIST specialization of sat_simplifier.h/.cpp and vector.h.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// One inline nontrivial source list. The actual 32-byte record contains no Go
// pointer. Word descriptors own pooled primitive clause-reference payloads.
// alive distinguishes a destroyed inline object from a constructed empty one.
type satClauseUseListRecord struct {
	payload                                 arena.VectorRef
	length, capacity, liveSize, initialized uint32
	alive                                   uint64
}
type satClauseUseLists struct {
	self                  *satClauseUseLists
	allocator             *ClauseAllocator // borrowed; clauses are never owned by a list
	budget                *arena.Budget
	words                 *arena.Words
	records               []satClauseUseListRecord
	backingID, ownerBytes uint64
	closed                bool
}

func newSATClauseUseLists(a *ClauseAllocator, b *arena.Budget, c arena.VectorConfig) (*satClauseUseLists, error) {
	if e := a.open(); e != nil {
		return nil, e
	}
	if c.Descriptors.Budget != b {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(satClauseUseLists{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	w, e := arena.NewWords(c)
	if e != nil {
		b.Release(n)
		return nil, e
	}
	u := &satClauseUseLists{allocator: a, budget: b, words: w, ownerBytes: n}
	u.self = u
	return u, nil
}
func (u *satClauseUseLists) open() error {
	if u == nil || u.self != u {
		return ErrSATOwner
	}
	if u.closed {
		return arena.ErrClosed
	}
	return nil
}

func (u *satClauseUseLists) destroy(i uint32) error {
	r := &u.records[i]
	if e := u.words.Free(r.payload); e != nil {
		return e
	}
	*r = satClauseUseListRecord{}
	return nil
}
func (u *satClauseUseLists) Init(numVars uint32) error {
	if e := u.open(); e != nil {
		return e
	}
	for i := range u.records {
		if e := u.destroy(uint32(i)); e != nil {
			return e
		}
	}
	u.records = u.records[:0]
	n := 2 * numVars // The source multiplication wraps as unsigned32.
	if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(satClauseUseListRecord{}).Size()) {
		return arena.ErrBounds
	}
	if int(n) > cap(u.records) {
		if u.backingID == ^uint64(0) {
			return arena.ErrSlotLimit
		}
		u.records = make([]satClauseUseListRecord, int(n))
		u.backingID++
	} else {
		u.records = u.records[:n]
	}
	for i := range u.records {
		u.records[i] = satClauseUseListRecord{alive: 1}
	}
	return nil
}
func (u *satClauseUseLists) Get(l Literal) (satClauseUseListView, error) {
	if e := u.open(); e != nil {
		return satClauseUseListView{}, e
	}
	i := l.index()
	if satStorageSourceDebug && uint64(i) >= uint64(len(u.records)) {
		return satClauseUseListView{}, ErrSATAssertion
	}
	if uint64(i) >= uint64(len(u.records)) {
		return satClauseUseListView{}, arena.ErrBounds
	}
	if u.records[i].alive == 0 {
		return satClauseUseListView{}, ErrSATOwner
	}
	return satClauseUseListView{owner: u, index: i, backingID: u.backingID}, nil
}
func (u *satClauseUseLists) route(c StoredClauseRef, mode uint32, except Literal) error {
	if e := u.open(); e != nil {
		return e
	}
	n, e := u.allocator.Size(c)
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		l, e := u.allocator.Literal(c, i)
		if e != nil {
			return e
		}
		if mode == 2 && l == except {
			continue
		}
		v, e := u.Get(l)
		if e != nil {
			return e
		}
		if mode == 0 {
			e = v.Insert(c)
		} else {
			e = v.Erase(c)
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (u *satClauseUseLists) Insert(c StoredClauseRef) error { return u.route(c, 0, 0) }
func (u *satClauseUseLists) Erase(c StoredClauseRef) error  { return u.route(c, 1, 0) }
func (u *satClauseUseLists) EraseExcept(c StoredClauseRef, l Literal) error {
	return u.route(c, 2, l)
}
func (u *satClauseUseLists) Finalize() error {
	if e := u.open(); e != nil {
		return e
	}
	if u.backingID == ^uint64(0) {
		return arena.ErrSlotLimit
	}
	for i := range u.records {
		if e := u.destroy(uint32(i)); e != nil {
			return e
		}
	}
	u.records = nil
	u.backingID++
	return nil
}
func (u *satClauseUseLists) Close() error {
	if u == nil || u.self != u {
		return ErrSATOwner
	}
	if u.closed {
		return nil
	}
	if e := u.Finalize(); e != nil {
		return e
	}
	u.words.Close()
	u.budget.Release(u.ownerBytes)
	u.ownerBytes = 0
	u.closed = true
	return nil
}

// A borrowed inline-object address represented by outer backing identity and
// index. Same-address transparent reconstruction remains visible; relocation
// or deallocation invalidates it. Inner payload growth does not invalidate it.
type satClauseUseListView struct {
	owner     *satClauseUseLists
	index     uint32
	backingID uint64
}

func (v satClauseUseListView) record() (*satClauseUseListRecord, error) {
	if e := v.owner.open(); e != nil {
		return nil, e
	}
	u := v.owner
	if v.backingID != u.backingID || uint64(v.index) >= uint64(len(u.records)) || u.records[v.index].alive == 0 {
		return nil, ErrSATOwner
	}
	return &u.records[v.index], nil
}
func (v satClauseUseListView) Size() (uint32, error) {
	r, e := v.record()
	if e != nil {
		return 0, e
	}
	return r.liveSize, nil
}
func (v satClauseUseListView) Empty() (bool, error) { n, e := v.Size(); return n == 0, e }
func (v satClauseUseListView) read(i uint32) (StoredClauseRef, error) {
	r, e := v.record()
	if e != nil {
		return StoredClauseRef{}, e
	}
	if satStorageSourceDebug && i >= r.length {
		return StoredClauseRef{}, ErrSATAssertion
	}
	if i >= r.capacity || i >= r.initialized {
		return StoredClauseRef{}, arena.ErrBounds
	}
	var a [5]uint32
	for k := uint32(0); k < 5; k++ {
		a[k], e = v.owner.words.Read(r.payload, i*5+k)
		if e != nil {
			return StoredClauseRef{}, e
		}
	}
	return StoredClauseRef{owner: uint64(a[0]) | uint64(a[1])<<32, slot: a[2], generation: a[3], storageKind: a[4]}, nil
}
func (v satClauseUseListView) write(i uint32, c StoredClauseRef) error {
	r, e := v.record()
	if e != nil {
		return e
	}
	if satStorageSourceDebug && i >= r.length {
		return ErrSATAssertion
	}
	return v.construct(r, i, c)
}

// Primitive placement construction has no source operator[] logical assertion.
func (v satClauseUseListView) construct(r *satClauseUseListRecord, i uint32, c StoredClauseRef) error {
	if i >= r.capacity {
		return arena.ErrBounds
	}
	a := [5]uint32{uint32(c.owner), uint32(c.owner >> 32), c.slot, c.generation, c.storageKind}
	for k := uint32(0); k < 5; k++ {
		if e := v.owner.words.Write(r.payload, i*5+k, a[k]); e != nil {
			return e
		}
	}
	if i >= r.initialized {
		r.initialized = i + 1
	}
	return nil
}
func (v satClauseUseListView) contains(c StoredClauseRef) (bool, error) {
	r, e := v.record()
	if e != nil {
		return false, e
	}
	for i := uint32(0); i < r.length; i++ {
		x, e := v.read(i)
		if e != nil {
			return false, e
		}
		if x == c {
			return true, nil
		}
	}
	return false, nil
}
func (v satClauseUseListView) removed(c StoredClauseRef) (bool, error) {
	if _, e := v.record(); e != nil {
		return false, e
	}
	return v.owner.allocator.Removed(c)
}
func (v satClauseUseListView) assertClause(c StoredClauseRef, contains, removed bool) error {
	found, e := v.contains(c)
	if e != nil {
		return e
	}
	if found != contains {
		return ErrSATAssertion
	}
	dead, e := v.removed(c)
	if e != nil {
		return e
	}
	if dead != removed {
		return ErrSATAssertion
	}
	return nil
}
func (v satClauseUseListView) Insert(c StoredClauseRef) error {
	r, e := v.record()
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		if e = v.assertClause(c, false, false); e != nil {
			return e
		}
	}
	if r.length == r.capacity {
		cap := uint32(2)
		if r.capacity != 0 {
			cap = (3*r.capacity + 1) >> 1
		}
		// Source pointer backing uses an eight-byte selected architecture stride.
		if cap <= r.capacity || cap*8+8 <= r.capacity*8+8 || uint64(cap)*5 > uint64(^uint32(0)) {
			return arena.ErrBounds
		}
		p, e := v.owner.words.Resize(r.payload, cap*5)
		if e != nil {
			return e
		}
		r.payload = p
		r.capacity = cap
	}
	// Source push constructs at the old end before publishing vector size.
	i := r.length
	if e = v.construct(r, i, c); e != nil {
		return e
	}
	r.length++
	r.liveSize++
	return nil
}
func (v satClauseUseListView) Erase(c StoredClauseRef) error {
	r, e := v.record()
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		if e = v.assertClause(c, true, true); e != nil {
			return e
		}
	}
	r.liveSize--
	return nil
}
func (v satClauseUseListView) EraseNotRemoved(c StoredClauseRef) error {
	r, e := v.record()
	if e != nil {
		return e
	}
	if satStorageSourceDebug {
		if e = v.assertClause(c, true, false); e != nil {
			return e
		}
	}
	for i := uint32(0); i < r.length; i++ {
		x, e := v.read(i)
		if e != nil {
			return e
		}
		if x != c {
			continue
		}
		for j := i + 1; j < r.length; j++ {
			x, e = v.read(j)
			if e != nil {
				return e
			}
			if e = v.write(j-1, x); e != nil {
				return e
			}
		}
		r.length--
		break
	}
	r.liveSize--
	return nil
}
func (v satClauseUseListView) Reset() error {
	r, e := v.record()
	if e != nil {
		return e
	}
	if e = v.owner.words.Free(r.payload); e != nil {
		return e
	}
	*r = satClauseUseListRecord{alive: 1}
	return nil
}
func (v satClauseUseListView) CheckInvariant() (bool, error) {
	r, e := v.record()
	if e != nil {
		return false, e
	}
	var count uint32
	for i := uint32(0); i < r.length; i++ {
		c, e := v.read(i)
		if e != nil {
			return false, e
		}
		dead, e := v.removed(c)
		if e != nil {
			return false, e
		}
		if !dead {
			count++
		}
	}
	if satStorageSourceDebug && count != r.liveSize {
		return false, ErrSATAssertion
	}
	return true, nil
}

// By-value lexical owning iterator. Do not shallow-copy an active iterator.
type satClauseUseListsIterator struct {
	view       satClauseUseListView
	size, i, j uint32
	closed     bool
}

func (v satClauseUseListView) Begin() (satClauseUseListsIterator, error) {
	r, e := v.record()
	if e != nil {
		return satClauseUseListsIterator{}, e
	}
	it := satClauseUseListsIterator{view: v, size: r.length}
	e = it.consume()
	return it, e
}
func (it *satClauseUseListsIterator) AtEnd() bool { return it.i == it.size }
func (it *satClauseUseListsIterator) consume() error {
	for {
		if it.AtEnd() {
			return nil
		}
		c, e := it.view.read(it.i)
		if e != nil {
			return e
		}
		dead, e := it.view.removed(c)
		if e != nil {
			return e
		}
		if !dead {
			return it.view.write(it.j, c)
		}
		it.i++
	}
}
func (it *satClauseUseListsIterator) Current() (StoredClauseRef, error) {
	if it.closed {
		return StoredClauseRef{}, arena.ErrClosed
	}
	if satStorageSourceDebug && it.AtEnd() {
		return StoredClauseRef{}, ErrSATAssertion
	}
	return it.view.read(it.i)
}
func (it *satClauseUseListsIterator) Next() error {
	if it.closed {
		return arena.ErrClosed
	}
	if satStorageSourceDebug {
		if it.AtEnd() {
			return ErrSATAssertion
		}
		c, e := it.view.read(it.i)
		if e != nil {
			return e
		}
		dead, e := it.view.removed(c)
		if e != nil {
			return e
		}
		if dead {
			return ErrSATAssertion
		}
	}
	it.i++
	it.j++
	return it.consume()
}
func (it *satClauseUseListsIterator) Close() error {
	if it.closed {
		return nil
	}
	for it.i < it.size {
		if e := it.Next(); e != nil {
			return e
		}
	}
	r, e := it.view.record()
	if e != nil {
		return e
	}
	if satStorageSourceDebug && it.j > r.length {
		return ErrSATAssertion
	}
	if it.j > r.capacity || it.j > r.initialized {
		return arena.ErrBounds
	}
	r.length = it.j
	it.closed = true
	return nil
}
