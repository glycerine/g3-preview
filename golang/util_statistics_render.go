// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// statistics.cpp map construction and rendering; external std::sort is adapted
// to an allocation-free concrete heap sort on borrowed scalar key references.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type statisticsMapEntry struct {
	key            StatisticsKey
	hash, state, u uint32
	d              float64
}
type statisticsMap struct {
	owner      *Statistics
	entries    []statisticsMapEntry
	size       uint32
	double     bool
	ownerBytes uint64
}

func newStatisticsMap(s *Statistics, d bool) (*statisticsMap, error) {
	n := uint64(reflect.TypeOf(statisticsMap{}).Size())
	if e := s.budget.Reserve(n); e != nil {
		return nil, e
	}
	m := &statisticsMap{owner: s, double: d, ownerBytes: n}
	if e := m.grow(8); e != nil {
		s.budget.Release(n)
		return nil, e
	}
	return m, nil
}
func (m *statisticsMap) close() {
	m.owner.budget.Release(m.ownerBytes + uint64(len(m.entries))*uint64(reflect.TypeOf(statisticsMapEntry{}).Size()))
	m.entries = nil
	m.ownerBytes = 0
}
func (m *statisticsMap) grow(n uint64) error {
	stride := uint64(reflect.TypeOf(statisticsMapEntry{}).Size())
	if n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := m.owner.budget.Reserve(n * stride); e != nil {
		return e
	}
	v := make([]statisticsMapEntry, int(n))
	for _, entry := range m.entries {
		if entry.state == 2 {
			i := uint64(entry.hash) & (n - 1)
			for v[i].state == 2 {
				i = (i + 1) & (n - 1)
			}
			v[i] = entry
		}
	}
	m.owner.budget.Release(uint64(len(m.entries)) * stride)
	m.entries = v
	return nil
}
func (m *statisticsMap) find(k StatisticsKey) (int, bool, error) {
	h, e := m.owner.keyHash(k)
	if e != nil {
		return 0, false, e
	}
	mask := uint32(len(m.entries) - 1)
	i := h & mask
	for steps := 0; steps < len(m.entries); steps++ {
		v := m.entries[i]
		if v.state == 0 {
			return int(i), false, nil
		}
		if v.state == 2 && v.hash == h {
			cmp, e := m.owner.keyCompare(v.key, k)
			if e != nil {
				return 0, false, e
			}
			if cmp == 0 {
				return int(i), true, nil
			}
		}
		i = (i + 1) & mask
	}
	return 0, false, arena.ErrBounds
}
func (m *statisticsMap) insert(k StatisticsKey, u uint32, d float64) error {
	if m.size<<2 > uint32(len(m.entries))*3 {
		if e := m.grow(2 * uint64(len(m.entries))); e != nil {
			return e
		}
	}
	h, e := m.owner.keyHash(k)
	if e != nil {
		return e
	}
	i, found, e := m.find(k)
	if e != nil {
		return e
	}
	if !found {
		m.size++
	}
	m.entries[i] = statisticsMapEntry{k, h, 2, u, d}
	return nil
}
func (s *Statistics) maps() (u, d *statisticsMap, e error) {
	u, e = newStatisticsMap(s, false)
	if e != nil {
		return
	}
	d, e = newStatisticsMap(s, true)
	if e != nil {
		u.close()
		u = nil
		return
	}
	for _, v := range s.u {
		idx, found, err := u.find(v.key)
		if err != nil {
			e = err
			break
		}
		n := v.value
		if found {
			n += u.entries[idx].u
		}
		if e = u.insert(v.key, n, 0); e != nil {
			break
		}
	}
	if e == nil {
		for _, v := range s.d {
			idx, found, err := d.find(v.key)
			if err != nil {
				e = err
				break
			}
			n := v.value
			if found {
				n = n + d.entries[idx].d
			}
			if e = d.insert(v.key, 0, n); e != nil {
				break
			}
		}
	}
	if e != nil {
		d.close()
		u.close()
		u = nil
		d = nil
	}
	return
}

type statisticsKeyBuffer struct {
	owner      *Statistics
	inline     [16]StatisticsKey
	heap       []StatisticsKey
	length     int
	ownerBytes uint64
}

func newStatisticsKeyBuffer(s *Statistics) (*statisticsKeyBuffer, error) {
	n := uint64(reflect.TypeOf(statisticsKeyBuffer{}).Size())
	if e := s.budget.Reserve(n); e != nil {
		return nil, e
	}
	return &statisticsKeyBuffer{owner: s, ownerBytes: n}, nil
}
func (b *statisticsKeyBuffer) data() []StatisticsKey {
	if b.heap != nil {
		return b.heap
	}
	return b.inline[:]
}
func (b *statisticsKeyBuffer) push(k StatisticsKey) error {
	old := b.data()
	if b.length == len(old) {
		n := uint64(len(old)) * 2
		stride := uint64(reflect.TypeOf(StatisticsKey{}).Size())
		if n > uint64(^uint(0)>>1)/stride {
			return arena.ErrBounds
		}
		if e := b.owner.budget.Reserve(n * stride); e != nil {
			return e
		}
		v := make([]StatisticsKey, int(n))
		copy(v, old)
		b.owner.budget.Release(uint64(len(b.heap)) * stride)
		b.heap = v
	}
	b.data()[b.length] = k
	b.length++
	return nil
}
func (b *statisticsKeyBuffer) close() {
	b.owner.budget.Release(b.ownerBytes + uint64(len(b.heap))*uint64(reflect.TypeOf(StatisticsKey{}).Size()))
	b.heap = nil
	b.ownerBytes = 0
}
func (b *statisticsKeyBuffer) sift(start, end int) error {
	v := b.data()
	root := start
	for root*2+1 < end {
		child := root*2 + 1
		if child+1 < end {
			cmp, e := b.owner.keyCompare(v[child], v[child+1])
			if e != nil {
				return e
			}
			if cmp < 0 {
				child++
			}
		}
		cmp, e := b.owner.keyCompare(v[root], v[child])
		if e != nil {
			return e
		}
		if cmp >= 0 {
			return nil
		}
		v[root], v[child] = v[child], v[root]
		root = child
	}
	return nil
}
func (b *statisticsKeyBuffer) sort() error {
	for i := b.length/2 - 1; i >= 0; i-- {
		if e := b.sift(i, b.length); e != nil {
			return e
		}
	}
	v := b.data()
	for end := b.length - 1; end > 0; end-- {
		v[0], v[end] = v[end], v[0]
		if e := b.sift(0, end); e != nil {
			return e
		}
	}
	return nil
}

// CString stream insertion preserves one width consumption for the entire key.
// Direct region reads fill the stream's charged reusable scratch; no uncharged
// intermediate string or byte allocation is used for borrowed spellings.
func (s *Statistics) appendKey(out *NumericTextStream, k StatisticsKey, skip uint32) error {
	n, e := s.keyLength(k)
	if e != nil {
		return e
	}
	if skip > n {
		return arena.ErrBounds
	}
	if e = out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	n -= skip
	if e = out.ensureScratch(uint64(n)); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		v, e := s.keyByte(k, skip+i)
		if e != nil {
			return e
		}
		out.scratch[i] = v
	}
	return out.emitBytes(out.scratch[:n], false)
}
func (s *Statistics) display(out *NumericTextStream, mode uint32) error {
	if e := out.lock(); e != nil {
		return e
	}
	out.mu.Unlock()
	u, d, e := s.maps()
	if e != nil {
		return e
	}
	defer u.close()
	defer d.close()
	if mode == 2 {
		return s.internalMap(out, u, d)
	}
	keys, e := newStatisticsKeyBuffer(s)
	if e != nil {
		return e
	}
	defer keys.close()
	for _, m := range []*statisticsMap{u, d} {
		for _, entry := range m.entries {
			if entry.state == 2 {
				if e = keys.push(entry.key); e != nil {
					return e
				}
			}
		}
	}
	if e = keys.sort(); e != nil {
		return e
	}
	max := uint32(0)
	for _, k := range keys.data()[:keys.length] {
		n, e := s.keyLength(k)
		if e != nil {
			return e
		}
		c, e := s.keyByte(k, 0)
		if e != nil {
			return e
		}
		if c == ':' {
			n--
		}
		if n > max {
			max = n
		}
	}
	if mode == 1 {
		if e = out.AppendCString("("); e != nil {
			return e
		}
	}
	for i, k := range keys.data()[:keys.length] {
		idx, found, e := u.find(k)
		if e != nil {
			return e
		}
		var dv float64
		if !found {
			j, ok, e := d.find(k)
			if e != nil {
				return e
			}
			if ok {
				dv = d.entries[j].d
			}
		}
		n, e := s.keyLength(k)
		if e != nil {
			return e
		}
		first, e := s.keyByte(k, 0)
		if e != nil {
			return e
		}
		skip := uint32(0)
		if first == ':' {
			skip = 1
		}
		paddingLen := n
		if mode == 1 {
			if i > 0 {
				if e = out.AppendCString("\n "); e != nil {
					return e
				}
			}
			if e = out.AppendCString(":"); e != nil {
				return e
			}
			for j := skip; j < n; j++ {
				c, e := s.keyByte(k, j)
				if e != nil {
					return e
				}
				if !IsSMT2SimpleSymbolChar(c) {
					c = '-'
				}
				if e = out.AppendByte(c); e != nil {
					return e
				}
			}
		} else {
			if e = s.appendKey(out, k, skip); e != nil {
				return e
			}
			if e = out.AppendCString(":"); e != nil {
				return e
			}
			paddingLen -= skip
		}
		for j := paddingLen; j < max; j++ {
			if e = out.AppendCString(" "); e != nil {
				return e
			}
		}
		if e = out.AppendCString(" "); e != nil {
			return e
		}
		if found {
			if e = out.AppendUint32(u.entries[idx].u); e != nil {
				return e
			}
		} else {
			if e = out.SetFloatMode(NumericFixedFloat); e != nil {
				return e
			}
			if e = out.SetPrecision(2); e != nil {
				return e
			}
			if e = out.AppendFloat64(dv); e != nil {
				return e
			}
		}
		if mode == 0 {
			if e = out.AppendCString("\n"); e != nil {
				return e
			}
		}
	}
	if mode == 1 {
		return out.AppendCString(")\n")
	}
	return nil
}
func (s *Statistics) internalMap(out *NumericTextStream, u, d *statisticsMap) error {
	for _, m := range []*statisticsMap{u, d} {
		for _, entry := range m.entries {
			if entry.state != 2 {
				continue
			}
			c, e := s.keyByte(entry.key, 0)
			if e != nil {
				return e
			}
			if c == ':' {
				c, e = s.keyByte(entry.key, 1)
				if e != nil {
					return e
				}
			}
			if c != 0 {
				return ErrStatisticsNonterminating
			}
			if e = out.AppendCString(" "); e != nil {
				return e
			}
			if m.double {
				e = out.AppendFloat64(entry.d)
			} else {
				e = out.AppendUint32(entry.u)
			}
			if e != nil {
				return e
			}
			if e = out.AppendCString("\n"); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Statistics) DisplayInto(out *NumericTextStream) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.display(out, 0)
}
func (s *Statistics) DisplaySMT2Into(out *NumericTextStream) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.display(out, 1)
}
func (s *Statistics) DisplayInternalInto(out *NumericTextStream) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return s.display(out, 2)
}
func (s *Statistics) displayText(mode uint32) (string, error) {
	if e := s.lock(); e != nil {
		return "", e
	}
	defer s.mu.Unlock()
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: s.budget})
	if e != nil {
		return "", e
	}
	defer out.Close()
	if e = s.display(out, mode); e != nil {
		return "", e
	}
	return out.String()
}
func (s *Statistics) Display() (string, error)         { return s.displayText(0) }
func (s *Statistics) DisplaySMT2() (string, error)     { return s.displayText(1) }
func (s *Statistics) DisplayInternal() (string, error) { return s.displayText(2) }
