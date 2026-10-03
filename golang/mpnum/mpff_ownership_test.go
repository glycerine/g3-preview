// Copyright (c) 2012 Microsoft Corporation. Owned-storage regressions for MPFF.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math/big"
	"reflect"
	"testing"
)

// Inspect actual roots before rendering or teardown; Close alone cannot detect
// a lost temporary because it also releases the entire significand backing.
func fastAudit(t *testing.T, m *FastFloatManager, values ...*FastFloat) {
	t.Helper()
	roots := map[uint32]bool{m.one.index(): true}
	for _, a := range values {
		if a.index() != 0 {
			if roots[a.index()] {
				t.Fatal("duplicate owning root", a.index())
			}
			roots[a.index()] = true
			serviceMust(t, m.check(a))
		}
	}
	active := uint32(0)
	for i := uint32(1); i < m.capacity; i++ {
		if m.read(&m.metadata, 2*i) == 1 {
			active++
			if !roots[i] {
				t.Fatal("unrooted block", i)
			}
		} else if !ffIsZero(ffLimbView{m: m, v: &m.significands, offset: i * m.precision, size: m.precision}) {
			t.Fatal("dead nonzero block", i)
		}
	}
	if active != m.live || active != uint32(len(roots)) {
		t.Fatal("live mismatch", active, m.live, len(roots))
	}
	refs := map[arena.VectorRef]bool{}
	add := func(r arena.VectorRef) {
		if r != (arena.VectorRef{}) {
			if refs[r] {
				t.Fatal("duplicate workspace")
			}
			refs[r] = true
		}
	}
	for _, v := range []ffVector{m.significands, m.metadata, m.freeIDs, m.setBuffer} {
		add(v.ref)
	}
	for _, v := range m.buffers {
		add(v.ref)
	}
	for _, v := range m.decimal {
		add(v.ref)
	}
	for _, r := range m.natural.divScratch {
		add(r)
	}
	for _, r := range m.natural.formatScratch {
		add(r)
	}
	if uint64(len(refs)) != uint64(m.words.Stats().Live) {
		t.Fatal("unrooted Words", len(refs), m.words.Stats())
	}
	if m.natural.words != m.words || m.natural.ownsWords {
		t.Fatal("invalid nested Words ownership")
	}
	if m.natural.text != nil {
		live := uint32(0)
		if m.natural.textRef != (arena.SmallBytesRef{}) {
			live = 1
			if !m.natural.text.IsLive(m.natural.textRef) {
				t.Fatal("invalid text root")
			}
		}
		if m.natural.text.Stats().Live != live {
			t.Fatal("unrooted text workspace")
		}
	}
}
func fastExact(m *FastFloatManager, a *FastFloat) *big.Rat {
	if a.index() == 0 {
		return new(big.Rat)
	}
	n := new(big.Int)
	for i := m.precision; i > 0; i-- {
		n.Lsh(n, 32)
		n.Add(n, new(big.Int).SetUint64(uint64(m.sig(a).read(i-1))))
	}
	if a.negative() {
		n.Neg(n)
	}
	if a.exponent >= 0 {
		return new(big.Rat).SetInt(n.Lsh(n, uint(a.exponent)))
	}
	return new(big.Rat).SetFrac(n, new(big.Int).Lsh(big.NewInt(1), uint(-int64(a.exponent))))
}
func TestFastFloatIndependentDirectedAliases(t *testing.T) {
	for _, p := range []uint32{2, 3, 5, 8} {
		for _, up := range []bool{false, true} {
			m := fastManager(t, p)
			serviceMust(t, m.SetRounding(up))
			for n := int64(-7); n <= 7; n++ {
				for d := int64(1); d <= 5; d++ {
					for op := 0; op < 4; op++ {
						for alias := 0; alias < 3; alias++ {
							var a, b, c FastFloat
							serviceMust(t, m.SetFraction64(&a, n, uint64(d)))
							serviceMust(t, m.SetFraction64(&b, 5, 7))
							av, bv := fastExact(m, &a), fastExact(m, &b)
							exact := new(big.Rat)
							out := &c
							if alias == 1 {
								out = &a
							}
							if alias == 2 {
								out = &b
							}
							var e error
							switch op {
							case 0:
								exact.Add(av, bv)
								e = m.Add(&a, &b, out)
							case 1:
								exact.Sub(av, bv)
								e = m.Sub(&a, &b, out)
							case 2:
								exact.Mul(av, bv)
								e = m.Mul(&a, &b, out)
							case 3:
								exact.Quo(av, bv)
								e = m.Div(&a, &b, out)
							}
							serviceMust(t, e)
							got := fastExact(m, out)
							cmp := got.Cmp(exact)
							if up && cmp < 0 || !up && cmp > 0 {
								t.Fatal("enclosure", p, up, n, d, op, alias, got, exact)
							}
							fastAudit(t, m, &a, &b, &c)
							serviceMust(t, m.Del(&c))
							serviceMust(t, m.Del(&b))
							serviceMust(t, m.Del(&a))
						}
					}
				}
			}
			fastAudit(t, m)
		}
	}
}
func TestFastFloatConstructorReservationPrefixes(t *testing.T) {
	completed := false
	for fail := int64(0); fail < 256; fail++ {
		b := arena.NewBudget(1 << 28)
		c := DefaultFastFloatConfig(serviceConfig(b).Integers.Words)
		c.InitialCapacity = 1
		c.PrecisionWords = 5
		b.FailAfter(fail)
		m, e := NewFastFloatManager(c)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(fail, e)
			}
		} else {
			fastAudit(t, m)
			m.Close()
			completed = true
		}
		if b.Used() != 0 {
			t.Fatal("constructor leak", fail, b.Used())
		}
		if completed {
			break
		}
	}
	if !completed {
		t.Fatal("sweep incomplete")
	}
}
func TestFastFloatArithmeticFailureRoots(t *testing.T) {
	for op := 0; op < 8; op++ {
		complete := false
		for fail := int64(0); fail < 512; fail++ {
			budget := arena.NewBudget(1 << 28)
			c := DefaultFastFloatConfig(serviceConfig(budget).Integers.Words)
			c.InitialCapacity = 2
			c.PrecisionWords = 33
			m, e := NewFastFloatManager(c)
			serviceMust(t, e)
			var a, b, out FastFloat
			serviceMust(t, m.SetInt32(&a, 7))
			serviceMust(t, m.SetInt32(&b, 3))
			budget.FailAfter(fail)
			switch op {
			case 0:
				e = m.Add(&a, &b, &out)
			case 1:
				e = m.Sub(&a, &b, &out)
			case 2:
				e = m.Mul(&a, &b, &out)
			case 3:
				e = m.Div(&a, &b, &out)
			case 4:
				e = m.SetFraction32(&out, -7, 3)
			case 5:
				e = m.SetFraction64(&out, -7, 3)
			case 6:
				e = m.Power(&a, 13, &out)
			case 7:
				e = m.Copy(&out, &a)
			}
			fastAudit(t, m, &a, &b, &out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(op, fail, e)
			}
			// Del remains allocation-free with the same exhausted injector armed.
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.Del(&b))
			serviceMust(t, m.Del(&a))
			fastAudit(t, m)
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("leak", op, fail, budget.Used())
			}
			if e == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("sweep incomplete", op)
		}
	}
}
func TestFastFloatPowerWrapPrefixes(t *testing.T) {
	m := fastManager(t, 2)
	var a, out FastFloat
	defer func() { m.Del(&out); m.Del(&a) }()
	serviceMust(t, m.SetInt32(&a, 3))
	serviceMust(t, m.RoundToPlusInf())
	e := m.Power(&a, 1<<31, &out)
	if !errors.Is(e, ErrFastFloatOverflow) {
		t.Fatal("finite overflow must precede wrap", e)
	}
	if fastExact(m, &out).Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatal("completed result prefix")
	}
	fastAudit(t, m, &a, &out)
	serviceMust(t, m.SetFraction32(&a, 1, 3))
	serviceMust(t, m.RoundToMinusInf())
	e = m.Power(&a, 1<<31, &out)
	if !errors.Is(e, ErrFastFloatPowerDomain) || out.index() != 0 {
		t.Fatal("actual wrap after underflow", e, out)
	}
	fastAudit(t, m, &a, &out)
	serviceMust(t, m.Reset(&a))
	serviceMust(t, m.SetInt32(&out, 7))
	e = m.Power(&a, 0, &out)
	if sourceExtendedGCDDebug {
		if !errors.Is(e, ErrFastFloatInvariant) || fastExact(m, &out).Cmp(big.NewRat(7, 1)) != 0 {
			t.Fatal(e, out)
		}
	} else if e != nil || out.index() != 0 {
		t.Fatal(e, out)
	}
	fastAudit(t, m, &a, &out)
}

func TestFastFloatFormattingFailureAndWarmStorage(t *testing.T) {
	for op := 0; op < 4; op++ {
		completed := false
		for fail := int64(0); fail < 256; fail++ {
			budget := arena.NewBudget(1 << 28)
			c := DefaultFastFloatConfig(serviceConfig(budget).Integers.Words)
			c.InitialCapacity = 2
			m, e := NewFastFloatManager(c)
			serviceMust(t, e)
			var a FastFloat
			serviceMust(t, m.SetFraction32(&a, -1, 3))
			dst := make([]byte, 512)
			for i := range dst {
				dst[i] = '!'
			}
			budget.FailAfter(fail)
			var n int
			switch op {
			case 0:
				n, e = m.FormatInto(&a, dst)
			case 1:
				n, e = m.FormatRawInto(&a, dst)
			case 2:
				n, e = m.FormatSMT2Into(&a, true, dst)
			case 3:
				n, e = m.FormatDecimalInto(&a, 64, 128, dst)
			}
			fastAudit(t, m, &a)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(op, fail, e)
				}
				for _, v := range dst {
					if v != '!' {
						t.Fatal("published on failure")
					}
				}
			} else {
				if n == 0 {
					t.Fatal("empty output")
				}
				completed = true
			}
			serviceMust(t, m.Del(&a))
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("formatting leak", op, fail)
			}
			if completed {
				break
			}
		}
		if !completed {
			t.Fatal("incomplete format sweep", op)
		}
	}
	m := fastManager(t, 3)
	var a, b, out FastFloat
	defer func() { m.Del(&out); m.Del(&b); m.Del(&a) }()
	serviceMust(t, m.SetFraction32(&a, 1, 3))
	serviceMust(t, m.SetFraction32(&b, 5, 7))
	dst := make([]byte, 512)
	run := func() {
		serviceMust(t, m.Mul(&a, &b, &out))
		serviceMust(t, m.Div(&a, &b, &out))
		_, e := m.FormatDecimalInto(&out, 32, 128, dst)
		serviceMust(t, e)
		_, e = m.FormatInto(&out, dst)
		serviceMust(t, e)
		_, e = m.FormatRawInto(&out, dst)
		serviceMust(t, e)
		_, e = m.FormatSMT2Into(&out, true, dst)
		serviceMust(t, e)
	}
	run()
	used := m.config.Storage.Descriptors.Budget.Used()
	m.config.Storage.Descriptors.Budget.FailAfter(0)
	if n := testing.AllocsPerRun(100, run); n != 0 {
		t.Fatal("warm allocations", n)
	}
	if m.config.Storage.Descriptors.Budget.Used() != used {
		t.Fatal("warm budget growth")
	}
	fastAudit(t, m, &a, &b, &out)
	short := []byte("!!")
	n, e := m.FormatDecimalInto(&a, 32, 128, short)
	if !errors.Is(e, ErrShortBuffer) || n <= 2 || string(short) != "!!" {
		t.Fatal(n, e, short)
	}
}
func TestFastFloatIDRecyclingAndRetirement(t *testing.T) {
	m := fastManager(t, 2)
	var a, b, c FastFloat
	serviceMust(t, m.SetInt32(&a, -7))
	serviceMust(t, m.SetInt32(&b, 3))
	aid, bid := a.index(), b.index()
	dead := a
	m.config.Storage.Descriptors.Budget.FailAfter(0)
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
	if a != dead {
		t.Fatal("source dead scalar bits changed")
	}
	if !errors.Is(m.check(&a), ErrFastFloatOwner) {
		t.Fatal("dead handle accepted")
	}
	serviceMust(t, m.SetInt32(&c, 1))
	if c.index() != bid {
		t.Fatal("not LIFO", aid, bid, c)
	}
	if ffGeneration(c.bits) != 0 && ffGeneration(c.bits) == ffGeneration(b.bits) {
		t.Fatal("generation not advanced")
	}
	m.write(&m.metadata, 2*c.index()+1, ^uint32(0))
	c.bits = ffPack(c.index(), ^uint32(0), false)
	retired := c.index()
	serviceMust(t, m.Del(&c))
	if m.read(&m.metadata, 2*retired) != 2 {
		t.Fatal("not retired")
	}
	c = FastFloat{}
	serviceMust(t, m.SetInt32(&c, 2))
	if c.index() != aid {
		t.Fatal("retired reused")
	}
	fastAudit(t, m, &c)
	serviceMust(t, m.Del(&c))
	fastAudit(t, m)
}
func TestFastFloatScopedFailureLifetimes(t *testing.T) {
	for op := 0; op < 5; op++ {
		complete := false
		for fail := int64(0); fail < 256; fail++ {
			budget := arena.NewBudget(1 << 28)
			cfg := DefaultFastFloatConfig(serviceConfig(budget).Integers.Words)
			cfg.InitialCapacity = 2
			m, e := NewFastFloatManager(cfg)
			serviceMust(t, e)
			a, e := NewScopedFastFloat(m)
			serviceMust(t, e)
			b, e := NewScopedFastFloat(m)
			serviceMust(t, e)
			serviceMust(t, a.SetInt32(7))
			serviceMust(t, b.SetInt32(3))
			v, e := NewScopedFastFloatVector(m)
			serviceMust(t, e)
			budget.FailAfter(fail)
			var out *ScopedFastFloat
			switch op {
			case 0:
				out, e = a.Clone()
			case 1:
				out, e = a.Add(b)
			case 2:
				out, e = a.Mul(b)
			case 3:
				e = v.Push(&a.num)
			case 4:
				e = v.Resize(17)
			}
			roots := []*FastFloat{&a.num, &b.num}
			if out != nil {
				roots = append(roots, &out.num)
			}
			for i := range v.values {
				roots = append(roots, &v.values[i])
			}
			fastAudit(t, m, roots...)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(op, fail, e)
			}
			if out != nil {
				serviceMust(t, out.Close())
				serviceMust(t, out.Close())
			}
			serviceMust(t, v.Close())
			serviceMust(t, a.Close())
			serviceMust(t, b.Close())
			fastAudit(t, m)
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("scope leak", op, fail, budget.Used())
			}
			if e == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("incomplete scope sweep", op)
		}
	}
}

func TestFastFloatDivisionSourceDiagnosticPrefix(t *testing.T) {
	m := fastManager(t, 33)
	var a, b, c FastFloat
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&c)
	serviceMust(t, m.SetInt32(&a, 3))
	serviceMust(t, m.SetInt32(&b, 7))
	serviceMust(t, m.SetInt32(&c, 1))
	if sourceExtendedGCDDebug {
		ffZero(m.sig(&a))
		m.config.Storage.Descriptors.Budget.FailAfter(0)
		e := m.Div(&a, &b, &c)
		if !errors.Is(e, ErrFastFloatInvariant) || c.index() == 0 {
			t.Fatal("source asserts after allocation before division", e, c)
		}
		fastAudit(t, m, &a, &b, &c)
	} else {
		serviceMust(t, m.Div(&a, &b, &c))
		fastAudit(t, m, &a, &b, &c)
	}
}

func fastOwnedBytes(m *FastFloatManager) uint64 {
	s := m.words.Stats()
	d := s.Descriptors
	n := m.natural.RuntimeOwner().Stats().TotalBytes + m.headerBytes + m.natural.headerBytes + s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if m.natural.text != nil {
		b := m.natural.text.Stats()
		n += b.OwnerBytes + b.PayloadBytes + b.MetadataBytes + b.DirectoryBytes
	}
	return n
}
func TestFastFloatActualWholeGraphCharges(t *testing.T) {
	m := fastManager(t, 5)
	budget := m.config.Storage.Descriptors.Budget
	if m.words.Stats().Live != 7 || m.capacity != 1024 || m.live != 1 {
		t.Fatal("cold graph", m.words.Stats(), m.capacity, m.live)
	}
	if budget.Used() != fastOwnedBytes(m) {
		t.Fatal("cold charge", budget.Used(), fastOwnedBytes(m))
	}
	var a FastFloat
	serviceMust(t, m.SetFraction32(&a, 1, 3))
	dst := make([]byte, 512)
	_, e := m.FormatDecimalInto(&a, 32, 128, dst)
	serviceMust(t, e)
	if budget.Used() != fastOwnedBytes(m) {
		t.Fatal("formatter charge", budget.Used(), fastOwnedBytes(m))
	}
	if m.decimal[0].capacity != 1024 || m.decimal[1].capacity != 1024 || m.decimal[3].capacity != 1024 || m.decimal[2].capacity != 16 {
		t.Fatal("source inline workspace histories", m.decimal)
	}
	fastAudit(t, m, &a)
	serviceMust(t, m.Del(&a))
	fastAudit(t, m)
	if e = arena.ValidateRecordType(reflect.TypeOf(FastFloat{})); e != nil {
		t.Fatal(e)
	}
}
