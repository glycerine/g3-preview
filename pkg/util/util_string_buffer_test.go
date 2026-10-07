package util

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func sb64(t *testing.T, b *arena.Budget) *StringBuffer64 {
	t.Helper()
	s, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func sbtext(t *testing.T, s *StringBuffer64) string {
	t.Helper()
	v, e := s.CStr()
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestStringBufferSourceThreeAssertions(t *testing.T) {
	s := sb64(t, nil)
	s.AppendCString("Testing")
	s.AppendInt32(10)
	s.AppendBool(true)
	if sbtext(t, s) != "Testing10true" {
		t.Fatal("source tst1")
	}
	s.Reset()
	seed := uint32(117)
	for i := 0; i < 10000; i++ {
		seed = seed*1664525 + 1013904223
		if e := s.AppendInt32(int32(seed % 10)); e != nil {
			t.Fatal(e)
		}
	}
	if len(sbtext(t, s)) != 10000 {
		t.Fatal("source tst2")
	}
	a, e := NewStringBuffer32(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := NewStringBuffer128(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.AppendCString("World")
	a.AppendCString("Hello")
	a.AppendByte(' ')
	if e = a.AppendBuffer128(b); e != nil {
		t.Fatal(e)
	}
	text, e := a.CStr()
	if e != nil || text != "Hello World" {
		t.Fatal("source tst3", text, e)
	}
}
func TestStringBufferInlineTerminatorResetAndBytes(t *testing.T) {
	a, e := NewStringBuffer32(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	a.AppendCString(strings.Repeat("a", 32))
	if a.Stats().Capacity != 32 || a.Stats().OverflowBytes != 0 || a.Size() != 32 {
		t.Fatal(a.Stats())
	}
	text, e := a.CStr()
	if e != nil || len(text) != 32 || a.Stats().Capacity != 64 || a.Size() != 32 {
		t.Fatal(text, e, a.Stats())
	}
	a.Reset()
	if a.Stats().Capacity != 64 || !a.Empty() {
		t.Fatal(a.Stats())
	}
	s := sb64(t, nil)
	s.AppendCString("A\x00ignored")
	s.AppendByte(0)
	s.AppendByte(255)
	if s.Size() != 3 || sbtext(t, s) != "A" {
		t.Fatal(s.Size(), sbtext(t, s))
	}
	v, e := s.Read(2)
	if e != nil || v != 255 {
		t.Fatal(v, e)
	}
	if _, e = s.Read(3); e != arena.ErrBounds {
		t.Fatal(e)
	}
	s.Reset()
	if e = s.AppendCString("\xff\xfe"); e != nil || sbtext(t, s) != "\xff\xfe" {
		t.Fatal(e, sbtext(t, s))
	}
	b, e := NewStringBuffer128(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	b.AppendCString(strings.Repeat("x", 128))
	if b.Stats().OverflowBytes != 0 {
		t.Fatal(b.Stats())
	}
	if _, e = b.CStr(); e != nil || b.Stats().Capacity != 256 {
		t.Fatal(e, b.Stats())
	}
}
func TestStringBufferNumericUnsignedAdapterAndLongDomains(t *testing.T) {
	s := sb64(t, nil)
	for _, v := range []uint32{0, 1, 2147483647, 2147483648, 4294967295} {
		s.Reset()
		if e := s.AppendUInt32(v); e != nil || sbtext(t, s) != strconv.FormatInt(int64(int32(v)), 10) {
			t.Fatal(v, e, sbtext(t, s))
		}
	}
	for _, v := range []int64{-9223372036854775808, -2147483648, -1, 0, 2147483647, 9223372036854775807} {
		s.Reset()
		if e := s.AppendLong(v); e != nil || sbtext(t, s) != strconv.FormatInt(v, 10) {
			t.Fatal(v, e, sbtext(t, s))
		}
	}
	s.Reset()
	s.AppendBool(false)
	if sbtext(t, s) != "false" {
		t.Fatal(sbtext(t, s))
	}
	short, e := NewStringBuffer32(StringBufferConfig{LongBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	defer short.Close()
	if e = short.AppendLong(2147483648); e != ErrStringBufferLongRange || short.Size() != 0 {
		t.Fatal(e, short.Size())
	}
	if e = short.AppendLong(-2147483648); e != nil {
		t.Fatal(e)
	}
}
func TestStringBufferFailureCapacityPrefixesAndImmediateBudgets(t *testing.T) {
	for prefix := int64(0); prefix < 4; prefix++ {
		b := arena.NewBudget(^uint64(0))
		s := sb64(t, b)
		s.AppendCString("old")
		base := b.Used()
		b.FailAfter(prefix)
		e := s.AppendCString(strings.Repeat("n", 300))
		b.FailAfter(-1)
		stats := s.Stats()
		if prefix < 3 {
			if e != arena.ErrInjectedFailure || s.Size() != 3 || sbtext(t, s) != "old" {
				t.Fatal(prefix, e, s.Size())
			}
			want := uint64(64) << uint(prefix)
			if stats.Capacity != want {
				t.Fatal(prefix, stats)
			}
		} else if e != nil || s.Size() != 303 {
			t.Fatal(prefix, e, s.Size())
		}
		if b.Used() != base+stats.OverflowBytes || stats.OwnerBytes != base {
			t.Fatal(prefix, b.Used(), base, stats)
		}
		s.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	b := arena.NewBudget(^uint64(0))
	s := sb64(t, b)
	s.AppendCString(strings.Repeat("x", 64))
	b.FailAfter(0)
	if _, e := s.CStr(); e != arena.ErrInjectedFailure || s.Size() != 64 || s.Stats().OverflowBytes != 0 {
		t.Fatal(e, s.Stats())
	}
	b.FailAfter(-1)
}
func TestStringBufferNineRoutesSourceCStrFirstAndSelfCases(t *testing.T) {
	a, _ := NewStringBuffer32(StringBufferConfig{})
	defer a.Close()
	b, _ := NewStringBuffer64(StringBufferConfig{})
	defer b.Close()
	c, _ := NewStringBuffer128(StringBufferConfig{})
	defer c.Close()
	a.AppendCString("a")
	b.AppendCString("b")
	c.AppendCString("c")
	for _, f := range []func() error{func() error { return a.AppendBuffer32(a) }, func() error { return a.AppendBuffer64(b) }, func() error { return a.AppendBuffer128(c) }, func() error { return b.AppendBuffer32(a) }, func() error { return b.AppendBuffer64(b) }, func() error { return b.AppendBuffer128(c) }, func() error { return c.AppendBuffer32(a) }, func() error { return c.AppendBuffer64(b) }, func() error { return c.AppendBuffer128(c) }} {
		if e := f(); e != nil {
			t.Fatal(e)
		}
	}
	s := sb64(t, nil)
	s.AppendCString(strings.Repeat("x", 40))
	if e := s.AppendBuffer64(s); e != nil || s.Size() != 80 || s.Stats().Capacity != 128 {
		t.Fatal("valid inline-grow self", e, s.Stats())
	}
	before := s.Stats()
	if e := s.AppendBuffer64(s); e != ErrStringBufferAlias || s.Stats() != before {
		t.Fatal("source heap dangling guard", e, s.Stats())
	}
	s.Reset()
	s.AppendCString(strings.Repeat("f", 128))
	if e := s.AppendBuffer64(s); e != nil || s.Size() != 256 || s.Stats().Capacity != 256 {
		t.Fatal("full heap sourceCStr grows first", e, s.Stats())
	}
	srcBudget := arena.NewBudget(^uint64(0))
	dstBudget := arena.NewBudget(^uint64(0))
	src := sb64(t, srcBudget)
	dst := sb64(t, dstBudget)
	src.AppendCString(strings.Repeat("s", 64))
	dst.AppendCString("D")
	dstBudget.FailAfter(0)
	e := dst.AppendBuffer64(src)
	dstBudget.FailAfter(-1)
	if e != arena.ErrInjectedFailure || src.Stats().Capacity != 128 || src.Size() != 64 || dst.Size() != 1 {
		t.Fatal(e, src.Stats(), dst.Stats())
	}
	if srcBudget.Used() != src.Stats().OwnerBytes+128 || dstBudget.Used() != dst.Stats().OwnerBytes {
		t.Fatal(srcBudget.Used(), dstBudget.Used())
	}
}
func TestStringBufferSymbolBulkAndNumericPrefixFailure(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	s := sb64(t, nil)
	for _, sym := range []Symbol{Symbol{}, NumericalSymbol(0), NumericalSymbol(2147483648), internSymbol(t, m, "named")} {
		s.Reset()
		if e := s.AppendSymbol(m, sym); e != nil {
			t.Fatal(e)
		}
		want, _ := m.Display(sym)
		if sbtext(t, s) != want {
			t.Fatal(sbtext(t, s), want)
		}
	}
	b := arena.NewBudget(^uint64(0))
	d := sb64(t, b)
	d.AppendCString(strings.Repeat("a", 62))
	b.FailAfter(0)
	e := d.AppendSymbol(m, NumericalSymbol(100))
	b.FailAfter(-1)
	if e != arena.ErrInjectedFailure || d.Size() != 64 || sbtext(t, d) != strings.Repeat("a", 62)+"k!" {
		t.Fatal(e, d.Size())
	}
	d.Reset()
	d.AppendCString("old")
	long := internSymbol(t, m, strings.Repeat("z", 300))
	b.FailAfter(1)
	e = d.AppendSymbol(m, long)
	b.FailAfter(-1)
	if e != arena.ErrInjectedFailure || d.Size() != 3 || sbtext(t, d) != "old" || d.Stats().Capacity != 256 {
		t.Fatal(e, d.Stats())
	}
}
func TestStringBufferIndependentByteModelAndWarmAllocations(t *testing.T) {
	s := sb64(t, nil)
	model := []byte{}
	seed := uint32(91)
	for i := 0; i < 1800; i++ {
		seed = seed*1664525 + 1013904223
		switch seed % 4 {
		case 0:
			s.Reset()
			model = model[:0]
		case 1:
			v := byte(seed >> 24)
			s.AppendByte(v)
			model = append(model, v)
		case 2:
			input := string([]byte{byte(seed >> 24), 'N', 0, 'X'})
			s.AppendCString(input)
			model = append(model, []byte(input[:cSymbolLength(input)])...)
		case 3:
			v := int32(seed)
			s.AppendInt32(v)
			model = append(model, []byte(strconv.FormatInt(int64(v), 10))...)
		}
		if uint64(len(model)) != s.Position() {
			t.Fatal(i, s.Position(), len(model))
		}
		for j, v := range model {
			got, e := s.Read(uint64(j))
			if e != nil || got != v {
				t.Fatal(i, j, got, v, e)
			}
		}
		n := len(model)
		for j, v := range model {
			if v == 0 {
				n = j
				break
			}
		}
		if sbtext(t, s) != string(model[:n]) {
			t.Fatal(i)
		}
	}
	s.Reset()
	s.AppendCString(strings.Repeat("warm", 100))
	s.Reset()
	if n := testing.AllocsPerRun(100, func() {
		s.Reset()
		s.AppendInt32(-123)
		s.AppendUInt32(^uint32(0))
		s.AppendLong(-9223372036854775808)
		s.AppendBool(true)
		s.AppendByte('x')
		s.AppendCString("bytes")
	}); n != 0 {
		t.Fatal(n)
	}
}
func TestStringBufferPhysicalInlineOwnerChargesAndExactPeak(t *testing.T) {
	a, e := NewStringBuffer32(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := NewStringBuffer64(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	c, e := NewStringBuffer128(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	for _, x := range []struct {
		stats StringBufferStats
		bytes uint64
	}{{a.Stats(), uint64(reflect.TypeOf(StringBuffer32{}).Size())}, {b.Stats(), uint64(reflect.TypeOf(StringBuffer64{}).Size())}, {c.Stats(), uint64(reflect.TypeOf(StringBuffer128{}).Size())}} {
		if x.stats.OwnerBytes != x.bytes || x.stats.OverflowBytes != 0 || x.stats.Capacity != x.stats.InlineBytes {
			t.Fatal(x)
		}
	}
	for _, r := range []reflect.Type{reflect.TypeOf([32]byte{}), reflect.TypeOf([64]byte{}), reflect.TypeOf([128]byte{})} {
		if e := arena.ValidateRecordType(r); e != nil {
			t.Fatal(e)
		}
	}
	owner := b.Stats().OwnerBytes
	for _, extra := range []uint64{383, 384} {
		budget := arena.NewBudget(owner + extra)
		s := sb64(t, budget)
		if e := s.AppendCString(strings.Repeat("x", 65)); e != nil {
			t.Fatal(e)
		}
		e := s.AppendCString(strings.Repeat("y", 100))
		if extra == 383 {
			if e != arena.ErrBudget || s.Position() != 65 || s.Stats().Capacity != 128 {
				t.Fatal(extra, e, s.Stats())
			}
		} else if e != nil || s.Position() != 165 || s.Stats().Capacity != 256 {
			t.Fatal(extra, e, s.Stats())
		}
		if budget.Used() != owner+s.Stats().OverflowBytes {
			t.Fatal(budget.Used(), s.Stats())
		}
		s.Close()
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}
}
func TestStringBufferSourceFailureBeforeDestAndReturnedTextLifetime(t *testing.T) {
	srcBudget := arena.NewBudget(^uint64(0))
	dstBudget := arena.NewBudget(^uint64(0))
	src := sb64(t, srcBudget)
	dst := sb64(t, dstBudget)
	src.AppendCString(strings.Repeat("s", 64))
	dst.AppendCString("D")
	before := dst.Stats()
	srcBudget.FailAfter(0)
	e := dst.AppendBuffer64(src)
	srcBudget.FailAfter(-1)
	if e != arena.ErrInjectedFailure || dst.Stats() != before || src.Stats().Capacity != 64 {
		t.Fatal(e, src.Stats(), dst.Stats())
	}
	src.Reset()
	src.AppendCString("A")
	src.AppendByte(0)
	src.AppendCString("hidden")
	if e := dst.AppendBuffer64(src); e != nil || sbtext(t, dst) != "DA" {
		t.Fatal(e, sbtext(t, dst))
	}
	text := sbtext(t, dst)
	dst.Reset()
	dst.AppendCString("changed")
	dst.Close()
	if text != "DA" {
		t.Fatal("returned copy lifetime", text)
	}
	src.Reset()
	if src.Inline[0] != 'A' {
		t.Fatal("reset changed retained bytes")
	}
	if got, e := src.CStr(); e != nil || got != "" || src.Inline[0] != 0 {
		t.Fatal("empty CStr terminator write", got, e)
	}
}
func TestStringBufferOwnerConfigurationAndSymbolGuards(t *testing.T) {
	budget := arena.NewBudget(0)
	if _, e := NewStringBuffer64(StringBufferConfig{Budget: budget}); e != arena.ErrBudget || budget.Used() != 0 {
		t.Fatal(e, budget.Used())
	}
	if _, e := NewStringBuffer32(StringBufferConfig{Budget: budget, LongBits: 16}); e != ErrStringBufferConfig || budget.Used() != 0 {
		t.Fatal(e, budget.Used())
	}
	m := symbolManagerTest(t, nil, false)
	other := symbolManagerTest(t, nil, false)
	s := sb64(t, nil)
	s.AppendCString("before")
	before := s.Stats()
	if e := s.AppendSymbol(m, internSymbol(t, other, "foreign")); e != ErrSymbolOwner || s.Stats() != before {
		t.Fatal(e, s.Stats())
	}
	if e := s.AppendSymbol(m, DummySymbol()); e != ErrInvalidSymbol {
		t.Fatal(e)
	}
	if e := s.AppendSymbol(nil, Symbol{}); e != ErrInvalidSymbol {
		t.Fatal(e)
	}
	copied := StringBuffer64{Self: s}
	if e := copied.AppendCString("x"); e != ErrStringBufferOwner {
		t.Fatal(e)
	}
	if e := copied.Close(); e != ErrStringBufferOwner {
		t.Fatal(e)
	}
	if e := s.AppendBuffer64(nil); e != ErrStringBufferOwner {
		t.Fatal(e)
	}
	m.Close()
	if e := s.AppendSymbol(m, Symbol{}); e != arena.ErrClosed || s.Stats() != before {
		t.Fatal(e, s.Stats())
	}
	s.Close()
	if e := s.AppendByte('x'); e != arena.ErrClosed {
		t.Fatal(e)
	}
	if _, e := s.CStr(); e != arena.ErrClosed {
		t.Fatal(e)
	}
	if e := s.Reset(); e != arena.ErrClosed {
		t.Fatal(e)
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	alias, e := NewStringBuffer(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer alias.Close()
	if alias.Stats().InlineBytes != 64 {
		t.Fatal(alias.Stats())
	}
}
