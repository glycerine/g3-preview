// Concrete constructed<true> scalar lifecycle adaptation of object_allocator.h.
package arena

import (
	"errors"
	"math"
	"reflect"
	"sync"
)

var (
	ErrPoolMode   = errors.New("arena: pool operation in wrong worker mode")
	ErrPoolWorker = errors.New("arena: invalid pool worker")
)

// SlotLimit limits physical slots per worker region. Zero uses MaxUint32.
// A pool budget accounts backing pages, ledgers, free arrays and directories;
// constructors account the cold owner/policy header; callback work stays external.
type Pool32Config struct {
	PageSize, Workers, SlotLimit uint32
	Budget                       *Budget
	DisableDestructors           bool
}

// Pool32Policy is one cold dispatcher per owner. Callbacks MUST NOT call any
// method on this pool: the owner mutex is held. They must not panic. External
// callback effects cannot be rolled back. No callback/closure is stored per slot.
// Nil Construct constructs zero; nil Reset preserves the entire scalar value.
// Recycle is a reset-only policy, unlike Slab32's zero/destruction contract.
type Pool32Policy struct {
	Construct func() uint32
	Reset     func(uint32) uint32
	Destroy   func(uint32)
}

// Pool32Ref's RegionWorker remains the physical allocation owner even when
// recycled onto another worker's free list. Refs are implicit-owner storage
// tokens; public domain wrappers must additionally validate their context owner.
type Pool32Ref struct {
	RegionWorker uint32
	Slot         Slot
	Generation   Generation
}
type pool32Meta struct {
	generation                 Generation
	live, initialized, retired bool
}
type pool32Page struct {
	data []uint32
	meta []pool32Meta
}
type pool32Worker struct {
	pages       []pool32Page
	free        []Pool32Ref
	nextSlot    uint64
	activePages uint32
}
type Pool32Stats struct {
	// OwnerBytes charges only this owner header; nested stats account their owners.
	OwnerBytes                                                uint64
	Workers, Live, Constructed, Retired, Pages                uint32
	RecordBytes, MetadataBytes, DirectoryBytes, FreeListBytes uint64
}

// Pool32 preserves worker partitions and reset-only constructed state. Its cold
// mutex serializes metadata/budget operations (a concurrency layout adaptation).
// A Budget shared with other owners still requires caller synchronization.
// Only the source construct=true policy is implemented; no raw unconstructed
// record path, rational-cell counterpart or complete source template is claimed.
type Pool32 struct {
	mu                 sync.Mutex
	workers            []pool32Worker
	config             Pool32Config
	policy             Pool32Policy
	concurrent, closed bool
}
type Pool32Worker struct {
	pool   *Pool32
	worker uint32
}

func (w Pool32Worker) Allocate() (Pool32Ref, error) { return w.pool.AllocateWorker(w.worker) }
func (w Pool32Worker) Recycle(ref Pool32Ref) error  { return w.pool.RecycleWorker(w.worker, ref) }

func NewPool32(config Pool32Config, policy Pool32Policy) (*Pool32, error) {
	if config.PageSize == 0 {
		config.PageSize = 1024
	}
	if config.Workers == 0 {
		config.Workers = 8
	}
	if config.SlotLimit == 0 {
		config.SlotLimit = math.MaxUint32
	}
	maxInt := uint64(^uint(0) >> 1)
	if config.PageSize&(config.PageSize-1) != 0 || uint64(config.PageSize) > maxInt/uint64(reflect.TypeOf(pool32Meta{}).Size()) {
		return nil, ErrConfig
	}
	header := uint64(reflect.TypeOf(Pool32{}).Size())
	if e := config.Budget.Reserve(header); e != nil {
		return nil, e
	}
	p := &Pool32{config: config, policy: policy}
	if e := p.reserveWorkers(config.Workers); e != nil {
		config.Budget.Release(header)
		return nil, e
	}
	return p, nil
}
func pool32Sum(a, b uint64) (uint64, error) {
	if a > math.MaxUint64-b {
		return 0, ErrBudget
	}
	return a + b, nil
}
func pool32Product(a, b uint64) (uint64, error) {
	if b != 0 && a > math.MaxUint64/b {
		return 0, ErrBudget
	}
	return a * b, nil
}

func (p *Pool32) reserveWorkers(count uint32) error {
	if uint64(count) <= uint64(len(p.workers)) {
		return nil
	}
	workerSize := uint64(reflect.TypeOf(pool32Worker{}).Size())
	if uint64(count) > uint64(^uint(0)>>1)/workerSize {
		return ErrPoolWorker
	}
	pageDir := uint64(reflect.TypeOf(pool32Page{}).Size())
	metaSize := uint64(reflect.TypeOf(pool32Meta{}).Size())
	perWorker := uint64(p.config.PageSize)*(4+metaSize) + pageDir
	extra, e := pool32Product(uint64(count)-uint64(len(p.workers)), perWorker)
	if e != nil {
		return e
	}
	directory := uint64(count) * workerSize
	amount, e := pool32Sum(directory, extra)
	if e != nil {
		return e
	}
	if e = p.config.Budget.Reserve(amount); e != nil {
		return e
	}
	workers := make([]pool32Worker, count)
	copy(workers, p.workers)
	for i := len(p.workers); i < len(workers); i++ {
		workers[i] = pool32Worker{pages: []pool32Page{{data: make([]uint32, p.config.PageSize), meta: make([]pool32Meta, p.config.PageSize)}}, nextSlot: 1, activePages: 1}
	}
	p.config.Budget.Release(uint64(cap(p.workers)) * workerSize)
	p.workers = workers
	return nil
}
func (p *Pool32) ReserveWorkers(count uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if p.concurrent {
		return ErrPoolMode
	}
	return p.reserveWorkers(count)
}
func (p *Pool32) EnableConcurrent(enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	p.concurrent = enabled
	return nil
}
func (p *Pool32) Worker(worker uint32) (Pool32Worker, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return Pool32Worker{}, ErrClosed
	}
	if uint64(worker) >= uint64(len(p.workers)) {
		return Pool32Worker{}, ErrPoolWorker
	}
	return Pool32Worker{p, worker}, nil
}

func (p *Pool32) ensurePage(worker uint32, index uint32) error {
	if index == math.MaxUint32 {
		return ErrSlotLimit
	} // active page count must not wrap
	w := &p.workers[worker]
	if uint64(index) < uint64(len(w.pages)) {
		if w.pages[index].data == nil {
			if e := p.config.Budget.Reserve(uint64(p.config.PageSize) * 4); e != nil {
				return e
			}
			w.pages[index].data = make([]uint32, p.config.PageSize)
		}
		if index >= w.activePages {
			w.activePages = index + 1
		}
		return nil
	}
	// Allocation advances at most one page at a time; skipped retired slots
	// can retain older tombstone pages, whose payload is restored above.
	if uint64(index) != uint64(len(w.pages)) {
		return ErrSlotLimit
	}
	newCap := cap(w.pages)
	newDir := uint64(0)
	entrySize := uint64(reflect.TypeOf(pool32Page{}).Size())
	if len(w.pages) == newCap {
		if newCap > int(^uint(0)>>1)/2 {
			return ErrSlotLimit
		}
		newCap *= 2
		if newCap == 0 {
			newCap = 1
		}
		if uint64(newCap) > uint64(^uint(0)>>1)/entrySize {
			return ErrSlotLimit
		}
		newDir = uint64(newCap) * entrySize
	}
	amount := uint64(p.config.PageSize)*(4+uint64(reflect.TypeOf(pool32Meta{}).Size())) + newDir
	if e := p.config.Budget.Reserve(amount); e != nil {
		return e
	}
	if newDir != 0 {
		pages := make([]pool32Page, len(w.pages), newCap)
		copy(pages, w.pages)
		p.config.Budget.Release(uint64(cap(w.pages)) * entrySize)
		w.pages = pages
	}
	w.pages = append(w.pages, pool32Page{data: make([]uint32, p.config.PageSize), meta: make([]pool32Meta, p.config.PageSize)})
	w.activePages = index + 1
	return nil
}
func (p *Pool32) record(ref Pool32Ref) (*pool32Page, uint32, *pool32Meta) {
	if ref.Slot == 0 || ref.Generation == 0 || uint64(ref.RegionWorker) >= uint64(len(p.workers)) {
		return nil, 0, nil
	}
	n := uint32(ref.Slot) - 1
	index := n / p.config.PageSize
	offset := n & (p.config.PageSize - 1)
	w := &p.workers[ref.RegionWorker]
	if uint64(index) >= uint64(len(w.pages)) {
		return nil, 0, nil
	}
	page := &w.pages[index]
	m := &page.meta[offset]
	if !m.live || m.generation != ref.Generation {
		return nil, 0, nil
	}
	return page, offset, m
}
func (p *Pool32) Allocate() (Pool32Ref, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return Pool32Ref{}, ErrClosed
	}
	if p.concurrent {
		return Pool32Ref{}, ErrPoolMode
	}
	return p.allocate(0)
}
func (p *Pool32) AllocateWorker(worker uint32) (Pool32Ref, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return Pool32Ref{}, ErrClosed
	}
	if !p.concurrent {
		return Pool32Ref{}, ErrPoolMode
	}
	if uint64(worker) >= uint64(len(p.workers)) {
		return Pool32Ref{}, ErrPoolWorker
	}
	return p.allocate(worker)
}
func (p *Pool32) allocate(worker uint32) (Pool32Ref, error) {
	w := &p.workers[worker]
	if len(w.free) > 0 {
		ref := w.free[len(w.free)-1]
		owner := &p.workers[ref.RegionWorker]
		n := uint32(ref.Slot) - 1
		m := &owner.pages[n/p.config.PageSize].meta[n&(p.config.PageSize-1)]
		w.free = w.free[:len(w.free)-1]
		m.generation++
		m.live = true
		ref.Generation = m.generation
		return ref, nil
	}
	slot := w.nextSlot
	for slot <= uint64(p.config.SlotLimit) {
		index := uint32((slot - 1) / uint64(p.config.PageSize))
		if e := p.ensurePage(worker, index); e != nil {
			return Pool32Ref{}, e
		}
		offset := uint32((slot - 1) % uint64(p.config.PageSize))
		m := &w.pages[index].meta[offset]
		if m.retired {
			slot++
			continue
		}
		// Source eagerly adds an empty page immediately when the current page is
		// filled. Preflight that predictable allocation before Construct/publishing.
		if slot%uint64(p.config.PageSize) == 0 {
			if e := p.ensurePage(worker, index+1); e != nil {
				return Pool32Ref{}, e
			}
			m = &w.pages[index].meta[offset]
		}
		value := uint32(0)
		if p.policy.Construct != nil {
			value = p.policy.Construct()
		}
		w.pages[index].data[offset] = value
		m.initialized = true
		m.live = true
		m.generation++
		w.nextSlot = slot + 1
		return Pool32Ref{worker, Slot(slot), m.generation}, nil
	}
	return Pool32Ref{}, ErrSlotLimit
}
func (p *Pool32) Read(ref Pool32Ref) (uint32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, ErrClosed
	}
	page, offset, m := p.record(ref)
	if m == nil {
		return 0, ErrInvalidSlot
	}
	return page.data[offset], nil
}
func (p *Pool32) Write(ref Pool32Ref, value uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	page, offset, m := p.record(ref)
	if m == nil {
		return ErrInvalidSlot
	}
	page.data[offset] = value
	return nil
}
func (p *Pool32) IsLive(ref Pool32Ref) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return false
	}
	_, _, m := p.record(ref)
	return m != nil
}
func (p *Pool32) Recycle(ref Pool32Ref) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if p.concurrent {
		return ErrPoolMode
	}
	return p.recycle(0, ref)
}
func (p *Pool32) RecycleWorker(worker uint32, ref Pool32Ref) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if !p.concurrent {
		return ErrPoolMode
	}
	if uint64(worker) >= uint64(len(p.workers)) {
		return ErrPoolWorker
	}
	return p.recycle(worker, ref)
}
func (p *Pool32) recycle(worker uint32, ref Pool32Ref) error {
	page, offset, m := p.record(ref)
	if m == nil {
		return ErrInvalidSlot
	}
	w := &p.workers[worker]
	if m.generation != math.MaxUint32 && len(w.free) == cap(w.free) {
		newCap := cap(w.free)
		if newCap == 0 {
			newCap = 1
		} else {
			if newCap > int(^uint(0)>>1)/2 {
				return ErrSlotLimit
			}
			newCap *= 2
		}
		entrySize := uint64(reflect.TypeOf(Pool32Ref{}).Size())
		if uint64(newCap) > uint64(^uint(0)>>1)/entrySize {
			return ErrSlotLimit
		}
		if e := p.config.Budget.Reserve(uint64(newCap) * entrySize); e != nil {
			return e
		}
		free := make([]Pool32Ref, len(w.free), newCap)
		copy(free, w.free)
		p.config.Budget.Release(uint64(cap(w.free)) * entrySize)
		w.free = free
	}
	if p.policy.Reset != nil {
		page.data[offset] = p.policy.Reset(page.data[offset])
	}
	m.live = false
	if m.generation == math.MaxUint32 {
		m.retired = true
	} else {
		w.free = append(w.free, ref)
	}
	return nil
}

func (p *Pool32) destroyWorker(w *pool32Worker) {
	if p.config.DisableDestructors || p.policy.Destroy == nil {
		return
	}
	visit := func(index uint32) {
		page := &w.pages[index]
		for i := range page.meta {
			if page.meta[i].initialized {
				p.policy.Destroy(page.data[i])
			}
		}
	}
	if w.activePages > 0 {
		visit(w.activePages - 1)
		for i := uint32(0); i+1 < w.activePages; i++ {
			visit(i)
		}
	}
}
func (p *Pool32) Reset() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}
	if p.concurrent {
		return ErrPoolMode
	}
	// Destination first-page payload and scratch directory coexist with the old
	// pages until every destructor succeeds. Fail before destructor side effects.
	scratch := uint64(len(p.workers)) * uint64(reflect.TypeOf([]uint32(nil)).Size())
	payload, e := pool32Product(uint64(len(p.workers)), uint64(p.config.PageSize)*4)
	if e != nil {
		return e
	}
	amount, e := pool32Sum(scratch, payload)
	if e != nil {
		return e
	}
	if e = p.config.Budget.Reserve(amount); e != nil {
		return e
	}
	replacement := make([][]uint32, len(p.workers))
	for i := range replacement {
		replacement[i] = make([]uint32, p.config.PageSize)
	}
	for i := range p.workers {
		w := &p.workers[i]
		p.destroyWorker(w)
		for j := range w.pages {
			page := &w.pages[j]
			p.config.Budget.Release(uint64(len(page.data)) * 4)
			page.data = nil
			for k := range page.meta {
				page.meta[k].live = false
				page.meta[k].initialized = false
				if page.meta[k].generation == math.MaxUint32 {
					page.meta[k].retired = true
				}
			}
		}
		w.pages[0].data = replacement[i]
		clear(w.free)
		w.free = w.free[:0]
		w.nextSlot = 1
		w.activePages = 1
	}
	p.config.Budget.Release(scratch)
	return nil
}
func (p *Pool32) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	for i := range p.workers {
		p.destroyWorker(&p.workers[i])
	}
	s := p.stats()
	p.config.Budget.Release(s.RecordBytes + s.MetadataBytes + s.DirectoryBytes + s.FreeListBytes + s.OwnerBytes)
	p.workers = nil
	p.closed = true
	return nil
}
func (p *Pool32) Stats() Pool32Stats { p.mu.Lock(); defer p.mu.Unlock(); return p.stats() }
func (p *Pool32) stats() Pool32Stats {
	s := Pool32Stats{Workers: uint32(len(p.workers)), DirectoryBytes: uint64(cap(p.workers)) * uint64(reflect.TypeOf(pool32Worker{}).Size())}
	if !p.closed {
		s.OwnerBytes = uint64(reflect.TypeOf(Pool32{}).Size())
	}
	for i := range p.workers {
		w := &p.workers[i]
		s.DirectoryBytes += uint64(cap(w.pages)) * uint64(reflect.TypeOf(pool32Page{}).Size())
		s.FreeListBytes += uint64(cap(w.free)) * uint64(reflect.TypeOf(Pool32Ref{}).Size())
		for j := range w.pages {
			page := &w.pages[j]
			if page.data != nil {
				s.Pages++
			}
			s.RecordBytes += uint64(len(page.data)) * 4
			s.MetadataBytes += uint64(len(page.meta)) * uint64(reflect.TypeOf(pool32Meta{}).Size())
			for k := range page.meta {
				m := &page.meta[k]
				if m.live {
					s.Live++
				}
				if m.initialized {
					s.Constructed++
				}
				if m.retired {
					s.Retired++
				}
			}
		}
	}
	return s
}
