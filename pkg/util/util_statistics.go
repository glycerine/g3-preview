// Concrete translation of util/statistics.h and statistics.cpp.
package util

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
)

var (
	ErrStatisticsOwner          = errors.New("g3: invalid statistics owner")
	ErrStatisticsKey            = errors.New("g3: null, foreign or invalid statistics key")
	ErrStatisticsSelfCopy       = errors.New("g3: source statistics self append does not terminate")
	ErrStatisticsNonterminating = errors.New("g3: source statistics internal nonempty key loop does not terminate")
)

// Keys borrow their byte storage; updates deliberately do not dereference them.
type StatisticsKey struct {
	origin uint64
	ref    arena.BytesRegionRef
	offset uint32
}
type StatisticsConfig struct {
	Keys   *arena.BytesRegion
	Budget *arena.Budget
}
type statisticsUintContribution struct {
	Key   StatisticsKey
	Value uint32
}
type statisticsDoubleContribution struct {
	key   StatisticsKey
	value float64
}
type statisticsKeyBinding struct {
	origin uint64
	region *arena.BytesRegion
}
type StatisticsStats struct {
	OwnerBytes, UintBytes, DoubleBytes, BindingBytes                    uint64
	UintSize, DoubleSize, UintCapacity, DoubleCapacity, BindingCapacity uint32
}

// Never shallow-copy Statistics. Regions and Budget are borrowed and externally
// confined; this mutex does not synchronize a Budget shared with other owners.
type Statistics struct {
	mu         sync.Mutex
	self       *Statistics
	keys       *arena.BytesRegion
	budget     *arena.Budget
	U          []statisticsUintContribution
	d          []statisticsDoubleContribution
	bindings   []statisticsKeyBinding
	ownerBytes uint64
	closed     bool
}

var statisticsTransferMutex sync.Mutex

func NewStatistics(c StatisticsConfig) (*Statistics, error) {
	if c.Keys == nil || c.Keys.Stats().OwnerBytes == 0 {
		return nil, ErrStatisticsKey
	}
	return newRetainedStatistics(c)
}

// Copy/move construction borrows a retained identity, even if its bytes were
// closed. Like source char* copying, this action never dereferences key text.
func newRetainedStatistics(c StatisticsConfig) (*Statistics, error) {
	if c.Keys == nil || c.Keys.OwnerID() == 0 {
		return nil, ErrStatisticsKey
	}
	n := uint64(reflect.TypeOf(Statistics{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	s := &Statistics{keys: c.Keys, budget: c.Budget, ownerBytes: n}
	s.self = s
	return s, nil
}

func (s *Statistics) lock() error {
	if s == nil || s.self != s {
		return ErrStatisticsOwner
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return arena.ErrClosed
	}
	return nil
}
func NewStatisticsKey(r *arena.BytesRegion, ref arena.BytesRegionRef, offset uint32) (StatisticsKey, error) {
	if r == nil {
		return StatisticsKey{}, ErrStatisticsKey
	}
	n, e := r.Size(ref)
	if e != nil {
		return StatisticsKey{}, e
	}
	if offset >= n {
		return StatisticsKey{}, arena.ErrBounds
	}
	return StatisticsKey{r.OwnerID(), ref, offset}, nil
}
func (s *Statistics) keyRegion(k StatisticsKey) *arena.BytesRegion {
	if k.origin == 0 {
		return nil
	}
	if k.origin == s.keys.OwnerID() {
		return s.keys
	}
	lo, hi := 0, len(s.bindings)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s.bindings[mid].origin < k.origin {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(s.bindings) && s.bindings[lo].origin == k.origin {
		return s.bindings[lo].region
	}
	return nil
}
func (s *Statistics) bind(r *arena.BytesRegion) error {
	if r == nil {
		return ErrStatisticsKey
	}
	id := r.OwnerID()
	if id == s.keys.OwnerID() {
		return nil
	}
	i := 0
	for i < len(s.bindings) && s.bindings[i].origin < id {
		i++
	}
	if i < len(s.bindings) && s.bindings[i].origin == id {
		return nil
	}
	if len(s.bindings) == cap(s.bindings) {
		n, e := statisticsNextCapacity(cap(s.bindings))
		if e != nil {
			return e
		}
		bytes, e := statisticsBackingBytes(uint64(n), uint64(reflect.TypeOf(statisticsKeyBinding{}).Size()), uint64(^uint(0)>>1))
		if e != nil {
			return e
		}
		if e = s.budget.Reserve(bytes); e != nil {
			return e
		}
		v := make([]statisticsKeyBinding, len(s.bindings), n)
		copy(v, s.bindings)
		s.budget.Release(uint64(cap(s.bindings)) * uint64(reflect.TypeOf(statisticsKeyBinding{}).Size()))
		s.bindings = v
	}
	s.bindings = s.bindings[:len(s.bindings)+1]
	copy(s.bindings[i+1:], s.bindings[i:])
	s.bindings[i] = statisticsKeyBinding{id, r}
	return nil
}
func (s *Statistics) BindKeyRegion(r *arena.BytesRegion) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if r == nil || r.Stats().OwnerBytes == 0 {
		return ErrStatisticsKey
	}
	return s.bind(r)
}

// maxBytes is the actual host-int bound in production; an explicit scalar
// parameter permits a hermetic 32bit boundary test without huge allocation.
func statisticsBackingBytes(n, stride, maxBytes uint64) (uint64, error) {
	if n > uint64(^uint32(0)) || stride == 0 || n > maxBytes/stride {
		return 0, arena.ErrBounds
	}
	return n * stride, nil
}
func statisticsNextCapacity(old int) (int, error) {
	n := uint64(2)
	if old != 0 {
		n = (3*uint64(old) + 1) / 2
	}
	if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
func (s *Statistics) growUint(n int) error {
	if n == 0 {
		return nil
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(statisticsUintContribution{}).Size())
	if bytes > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := s.budget.Reserve(bytes); e != nil {
		return e
	}
	v := make([]statisticsUintContribution, len(s.U), n)
	copy(v, s.U)
	s.budget.Release(uint64(cap(s.U)) * uint64(reflect.TypeOf(statisticsUintContribution{}).Size()))
	s.U = v
	return nil
}
func (s *Statistics) growDouble(n int) error {
	if n == 0 {
		return nil
	}
	bytes := uint64(n) * uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size())
	if bytes > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e := s.budget.Reserve(bytes); e != nil {
		return e
	}
	v := make([]statisticsDoubleContribution, len(s.d), n)
	copy(v, s.d)
	s.budget.Release(uint64(cap(s.d)) * uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size()))
	s.d = v
	return nil
}
func (s *Statistics) appendUint(k StatisticsKey, v uint32) error {
	if uint64(len(s.U)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if len(s.U) == cap(s.U) {
		n, e := statisticsNextCapacity(cap(s.U))
		if e != nil {
			return e
		}
		if e = s.growUint(n); e != nil {
			return e
		}
	}
	s.U = append(s.U, statisticsUintContribution{k, v})
	return nil
}
func (s *Statistics) appendDouble(k StatisticsKey, v float64) error {
	if uint64(len(s.d)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	if len(s.d) == cap(s.d) {
		n, e := statisticsNextCapacity(cap(s.d))
		if e != nil {
			return e
		}
		if e = s.growDouble(n); e != nil {
			return e
		}
	}
	s.d = append(s.d, statisticsDoubleContribution{k, v})
	return nil
}
func (s *Statistics) UpdateUint(k StatisticsKey, v uint32) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if v == 0 {
		return nil
	}
	return s.appendUint(k, v)
}
func (s *Statistics) UpdateDouble(k StatisticsKey, v float64) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	if v == 0 {
		return nil
	}
	return s.appendDouble(k, v)
}
func (s *Statistics) Reset() error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	s.U = s.U[:0]
	s.d = s.d[:0]
	return nil
}
func (s *Statistics) Size() uint32 {
	if e := s.lock(); e != nil {
		return 0
	}
	defer s.mu.Unlock()
	return uint32(len(s.U) + len(s.d))
}
func (s *Statistics) IsUint(i uint32) bool {
	if e := s.lock(); e != nil {
		return false
	}
	defer s.mu.Unlock()
	return uint64(i) < uint64(len(s.U))
}
func (s *Statistics) Key(i uint32) (StatisticsKey, error) {
	if e := s.lock(); e != nil {
		return StatisticsKey{}, e
	}
	defer s.mu.Unlock()
	if uint64(i) < uint64(len(s.U)) {
		return s.U[i].Key, nil
	}
	j := uint64(i) - uint64(len(s.U))
	if j >= uint64(len(s.d)) {
		return StatisticsKey{}, arena.ErrBounds
	}
	return s.d[j].key, nil
}
func (s *Statistics) UintValue(i uint32) (uint32, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	if uint64(i) >= uint64(len(s.U)) {
		return 0, arena.ErrBounds
	}
	return s.U[i].Value, nil
}
func (s *Statistics) DoubleValue(i uint32) (float64, error) {
	if e := s.lock(); e != nil {
		return 0, e
	}
	defer s.mu.Unlock()
	j := uint64(i) - uint64(len(s.U))
	if i < uint32(len(s.U)) || j >= uint64(len(s.d)) {
		return 0, arena.ErrBounds
	}
	return s.d[j].value, nil
}
func (s *Statistics) Stats() StatisticsStats {
	if s == nil || s.self != s {
		return StatisticsStats{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return StatisticsStats{s.ownerBytes, uint64(cap(s.U)) * uint64(reflect.TypeOf(statisticsUintContribution{}).Size()), uint64(cap(s.d)) * uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size()), uint64(cap(s.bindings)) * uint64(reflect.TypeOf(statisticsKeyBinding{}).Size()), uint32(len(s.U)), uint32(len(s.d)), uint32(cap(s.U)), uint32(cap(s.d)), uint32(cap(s.bindings))}
}
func (s *Statistics) clearUint() {
	s.budget.Release(uint64(cap(s.U)) * uint64(reflect.TypeOf(statisticsUintContribution{}).Size()))
	s.U = nil
}
func (s *Statistics) clearDouble() {
	s.budget.Release(uint64(cap(s.d)) * uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size()))
	s.d = nil
}
func (s *Statistics) Close() error {
	if s == nil || s.self != s {
		return ErrStatisticsOwner
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.clearDouble()
	s.clearUint()
	s.budget.Release(uint64(cap(s.bindings))*uint64(reflect.TypeOf(statisticsKeyBinding{}).Size()) + s.ownerBytes)
	s.bindings = nil
	s.ownerBytes = 0
	s.closed = true
	return nil
}
func (s *Statistics) adoptKey(src *Statistics, k StatisticsKey) error {
	r := src.keyRegion(k)
	if r == nil {
		return nil
	}
	return s.bind(r)
}

// Cross-owner operations share this cold lock, avoiding Go pointer ordering.
func statisticsLockPair(s, src *Statistics) error {
	if e := s.lock(); e != nil {
		return e
	}
	if src != s {
		if e := src.lock(); e != nil {
			s.mu.Unlock()
			return e
		}
	}
	return nil
}
func statisticsUnlockPair(s, src *Statistics) {
	if src != s {
		src.mu.Unlock()
	}
	s.mu.Unlock()
}
func (s *Statistics) Copy(src *Statistics) error {
	statisticsTransferMutex.Lock()
	defer statisticsTransferMutex.Unlock()
	if e := statisticsLockPair(s, src); e != nil {
		return e
	}
	defer statisticsUnlockPair(s, src)
	if s == src {
		if len(s.U)+len(s.d) != 0 {
			return ErrStatisticsSelfCopy
		}
		return nil
	}
	for _, v := range src.U {
		if e := s.adoptKey(src, v.Key); e != nil {
			return e
		}
		if e := s.appendUint(v.Key, v.Value); e != nil {
			return e
		}
	}
	for _, v := range src.d {
		if e := s.adoptKey(src, v.key); e != nil {
			return e
		}
		if e := s.appendDouble(v.key, v.value); e != nil {
			return e
		}
	}
	return nil
}
func (s *Statistics) assign(src *Statistics) error {
	if s == src {
		return nil
	}
	s.clearUint()
	for _, v := range src.U {
		if e := s.adoptKey(src, v.Key); e != nil {
			return e
		}
	}
	if e := s.growUint(cap(src.U)); e != nil {
		return e
	}
	s.U = s.U[:len(src.U)]
	copy(s.U, src.U)
	s.clearDouble()
	for _, v := range src.d {
		if e := s.adoptKey(src, v.key); e != nil {
			return e
		}
	}
	if e := s.growDouble(cap(src.d)); e != nil {
		return e
	}
	s.d = s.d[:len(src.d)]
	copy(s.d, src.d)
	return nil
}
func (s *Statistics) Assign(src *Statistics) error {
	statisticsTransferMutex.Lock()
	defer statisticsTransferMutex.Unlock()
	if e := statisticsLockPair(s, src); e != nil {
		return e
	}
	defer statisticsUnlockPair(s, src)
	return s.assign(src)
}

// svector has no move assignment: an rvalue binds its const-copy overload.
func (s *Statistics) MoveAssign(src *Statistics) error { return s.Assign(src) }
func (s *Statistics) Clone(b *arena.Budget) (*Statistics, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	v, e := newRetainedStatistics(StatisticsConfig{s.keys, b})
	if e != nil {
		return nil, e
	}
	if e = v.assign(s); e != nil {
		v.Close()
		return nil, e
	}
	return v, nil
}
func (s *Statistics) MoveClone(b *arena.Budget) (*Statistics, error) {
	if e := s.lock(); e != nil {
		return nil, e
	}
	defer s.mu.Unlock()
	v, e := newRetainedStatistics(StatisticsConfig{s.keys, b})
	if e != nil {
		return nil, e
	}
	n := uint64(cap(s.U))*uint64(reflect.TypeOf(statisticsUintContribution{}).Size()) + uint64(cap(s.d))*uint64(reflect.TypeOf(statisticsDoubleContribution{}).Size()) + uint64(cap(s.bindings))*uint64(reflect.TypeOf(statisticsKeyBinding{}).Size())
	if b != s.budget && n != 0 {
		if e = b.Reserve(n); e != nil {
			v.Close()
			return nil, e
		}
		s.budget.Release(n)
	}
	v.U, v.d, v.bindings = s.U, s.d, s.bindings
	s.U = nil
	s.d = nil
	s.bindings = nil
	return v, nil
}
func (s *Statistics) keyLength(k StatisticsKey) (uint32, error) {
	r := s.keyRegion(k)
	if r == nil {
		return 0, ErrStatisticsKey
	}
	n, e := r.Size(k.ref)
	if e != nil {
		return 0, e
	}
	if k.offset >= n {
		return 0, arena.ErrBounds
	}
	for i := uint32(0); i < n-k.offset; i++ {
		v, e := r.Read(k.ref, k.offset+i)
		if e != nil {
			return 0, e
		}
		if v == 0 {
			return i, nil
		}
	}
	return 0, arena.ErrBounds
}
func (s *Statistics) keyByte(k StatisticsKey, i uint32) (byte, error) {
	r := s.keyRegion(k)
	if r == nil {
		return 0, ErrStatisticsKey
	}
	if uint64(k.offset)+uint64(i) > uint64(^uint32(0)) {
		return 0, arena.ErrBounds
	}
	return r.Read(k.ref, k.offset+i)
}
func (s *Statistics) keyHash(k StatisticsKey) (uint32, error) {
	n, e := s.keyLength(k)
	if e != nil {
		return 0, e
	}
	var readErr error
	v := WordReader(n, 17, func(word uint32) uint32 {
		var w uint32
		for j := uint32(0); j < 4 && word*4+j < n; j++ {
			c, e := s.keyByte(k, word*4+j)
			if e != nil {
				readErr = e
				return 0
			}
			w |= uint32(c) << (8 * j)
		}
		return w
	})
	return v, readErr
}
func (s *Statistics) keyCompare(a, b StatisticsKey) (int, error) {
	for i := uint32(0); ; i++ {
		x, e := s.keyByte(a, i)
		if e != nil {
			return 0, e
		}
		y, e := s.keyByte(b, i)
		if e != nil {
			return 0, e
		}
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
		if x == 0 {
			return 0, nil
		}
		if i == ^uint32(0) {
			return 0, arena.ErrBounds
		}
	}
}
func (s *Statistics) KeyText(k StatisticsKey) (string, error) {
	if e := s.lock(); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	n, e := s.keyLength(k)
	if e != nil {
		return "", e
	}
	if uint64(n) > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	var out strings.Builder
	out.Grow(int(n))
	for i := uint32(0); i < n; i++ {
		v, e := s.keyByte(k, i)
		if e != nil {
			return "", e
		}
		out.WriteByte(v)
	}
	return out.String(), nil
}

func (s *Statistics) literalKey(name string) (StatisticsKey, error) {
	ref, e := s.keys.Allocate(uint32(len(name) + 1))
	if e != nil {
		return StatisticsKey{}, e
	}
	for i := range name {
		if e = s.keys.Write(ref, uint32(i), name[i]); e != nil {
			return StatisticsKey{}, e
		}
	}
	if e = s.keys.Write(ref, uint32(len(name)), 0); e != nil {
		return StatisticsKey{}, e
	}
	return StatisticsKey{s.keys.OwnerID(), ref, 0}, nil
}

func (s *Statistics) appendLiteralUint64(name string, value uint64) error {
	if value == 0 {
		return nil
	}
	k, e := s.literalKey(name)
	if e != nil {
		return e
	}
	if value <= uint64(^uint32(0)) {
		return s.appendUint(k, uint32(value))
	}
	return s.appendDouble(k, float64(value))
}

func (s *Statistics) appendLiteralDouble(name string, value float64) error {
	if value == 0 {
		return nil
	}
	k, e := s.literalKey(name)
	if e != nil {
		return e
	}
	return s.appendDouble(k, value)
}

// GetRlimitStatistics reads the actual source reslimit count. The literal is
// allocated explicitly in the receiver's borrowed key domain; no hidden cache.
// That caller-owned region, not Statistics.Close, owns this spelling lifetime.
func GetRlimitStatistics(l *Reslimit, s *Statistics) error {
	if l == nil {
		return ErrStatisticsOwner
	}
	if l.ClosedField {
		return arena.ErrClosed
	}
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.appendLiteralUint64("rlimit count", l.Count())
}

type goMemoryStatistics struct {
	heapAlloc uint64
	heapSys   uint64
	mallocs   uint64
}

func memoryMegabytes(bytes uint64) float64 {
	const mebibyte = uint64(1024 * 1024)
	centi := (bytes/mebibyte)*100 + ((bytes%mebibyte)*100)/mebibyte
	return float64(centi) / 100
}

func (s *Statistics) appendMemoryStatistics(memory goMemoryStatistics) error {
	if e := s.appendLiteralDouble("max memory", memoryMegabytes(memory.heapSys)); e != nil {
		return e
	}
	if e := s.appendLiteralDouble("memory", memoryMegabytes(memory.heapAlloc)); e != nil {
		return e
	}
	return s.appendLiteralUint64("num allocs", memory.mallocs)
}

// GetMemoryStatistics uses the Go runtime allocator counters corresponding to
// statistics.cpp:239-246. HeapAlloc is current heap object memory. HeapSys is
// the standard library's estimate of the largest heap size reached, and
// Mallocs is the cumulative heap-object allocation count.
func GetMemoryStatistics(s *Statistics) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return s.appendMemoryStatistics(goMemoryStatistics{
		heapAlloc: memory.HeapAlloc,
		heapSys:   memory.HeapSys,
		mallocs:   memory.Mallocs,
	})
}
