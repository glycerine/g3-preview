// Source: src/test/symbol.cpp, with finite ownership and byte-policy extensions.
package util

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func symbolManagerTest(t *testing.T, b *arena.Budget, direct bool) *SymbolManager {
	t.Helper()
	m, e := NewSymbolManager(SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2, DirectOnly: direct})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Fatal(e)
		}
	})
	return m
}
func internSymbol(t *testing.T, m *SymbolManager, text string) Symbol {
	t.Helper()
	s, e := m.Intern(text)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func symbolText(t *testing.T, m *SymbolManager, s Symbol) string {
	t.Helper()
	v, e := m.String(s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

func TestSymbolSourceAssertions(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	s1, s2, s3 := internSymbol(t, m, "foo"), internSymbol(t, m, "boo"), internSymbol(t, m, "foo")
	if s1 == s2 || s1 != s3 {
		t.Fatal("source symbol identity")
	}
	var output string
	for i, s := range []Symbol{s1, s2, s3} {
		v, e := m.Display(s)
		if e != nil {
			t.Fatal(e)
		}
		if i != 0 {
			output += " "
		}
		output += v
	}
	if output+"\n" != "foo boo foo\n" {
		t.Fatal("source output", output)
	}
	for _, f := range []struct {
		s    Symbol
		text string
		want bool
	}{
		{s1, "foo", true}, {s1, "boo", false}, {s2, "foo", false}, {s3, "foo", true}, {s2, "boo", true},
	} {
		got, e := m.EqualString(f.s, &f.text)
		if e != nil || got != f.want {
			t.Fatal("source text equality", f.text, got, e)
		}
	}
	for _, f := range []struct {
		a, b Symbol
		want bool
	}{
		{s2, s1, true}, {s1, s2, false}, {s1, s3, false},
		{internSymbol(t, m, "abcc"), internSymbol(t, m, "abcd"), true},
		{internSymbol(t, m, "abcd"), internSymbol(t, m, "abcc"), false},
		{internSymbol(t, m, "abc"), internSymbol(t, m, "abcc"), true},
		{internSymbol(t, m, "abcd"), internSymbol(t, m, "abc"), false},
		{NumericalSymbol(10), s1, true}, {s1, NumericalSymbol(10), false},
		{NumericalSymbol(10), NumericalSymbol(20), true}, {NumericalSymbol(20), NumericalSymbol(10), false},
		{NumericalSymbol(10), NumericalSymbol(10), false},
		{internSymbol(t, m, "a"), internSymbol(t, m, "b"), true},
		{internSymbol(t, m, "z"), internSymbol(t, m, "b"), false},
		{internSymbol(t, m, "zzz"), internSymbol(t, m, "b"), false},
		{internSymbol(t, m, "zzz"), internSymbol(t, m, "zzzb"), true},
	} {
		got, e := m.Less(f.a, f.b)
		if e != nil || got != f.want {
			t.Fatal("source order", got, f.want, e)
		}
	}
}

func TestSymbolTagsNullBytesAndCachedHash(t *testing.T) {
	for _, direct := range []bool{false, true} {
		m := symbolManagerTest(t, nil, direct)
		if e := arena.ValidateRecordType(reflect.TypeOf(Symbol{})); e != nil {
			t.Fatal(e)
		}
		for _, f := range []struct {
			value uint32
			text  string
			size  uint32
		}{
			{0, "k!0", 4}, {10, "k!10", 7}, {math.MaxUint32, "k!-1", 35},
		} {
			s := NumericalSymbol(f.value)
			if !s.IsNumerical() || s.IsMarked() {
				t.Fatal("numeric tag")
			}
			n, e := s.Number()
			if e != nil || n != f.value {
				t.Fatal(n, e)
			}
			if symbolText(t, m, s) != f.text {
				t.Fatal("signed rendering")
			}
			h, e := m.Hash(s)
			if e != nil || h != f.value {
				t.Fatal(h, e)
			}
			n, e = m.Size(s)
			if e != nil || n != f.size {
				t.Fatal("source size", n, e)
			}
			contains, e := m.Contains(s, 'k')
			if e != nil || contains {
				t.Fatal("numeric contains", e)
			}
			marked, e := s.Mark()
			if e != nil || !marked.IsMarked() || marked.IsNumerical() {
				t.Fatal("mark tag", e)
			}
			plain, e := marked.unmark()
			if e != nil || plain != s {
				t.Fatal("unmark", e)
			}
			if _, e := m.String(marked); !errors.Is(e, ErrInvalidSymbol) {
				t.Fatal("marked string", e)
			}
		}
		var null Symbol
		if symbolText(t, m, null) != "<null>" {
			t.Fatal("null str")
		}
		if text, present, e := m.BareString(null); e != nil || present || text != "" {
			t.Fatal("source nullable bare string", text, present, e)
		}
		if v, e := m.Display(null); e != nil || v != "null" {
			t.Fatal("null display", v, e)
		}
		if h, e := m.Hash(null); e != nil || h != 0x9e3779d9 {
			t.Fatal("null hash", h, e)
		}
		if ok, e := m.EqualString(null, nil); e != nil || !ok {
			t.Fatal("null equality", e)
		}
		if _, e := m.Size(null); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("null size", e)
		}
		if _, e := m.Contains(null, 0); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("null contains", e)
		}
		if DummySymbol() != (Symbol{Tag: 2}) || !DummySymbol().IsMarked() {
			t.Fatal("dummy tag")
		}
		if _, e := null.Number(); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("null number", e)
		}
		s := internSymbol(t, m, "foo\x00ignored")
		if s != internSymbol(t, m, "foo") || symbolText(t, m, s) != "foo" {
			t.Fatal("NUL truncation")
		}
		if text, present, e := m.BareString(s); e != nil || !present || text != "foo" {
			t.Fatal("source bare spelling", text, present, e)
		}
		if _, _, e := m.BareString(NumericalSymbol(1)); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("bare string numerical precondition", e)
		}
		if h, e := m.Hash(s); e != nil || h != 1484660743 {
			t.Fatal("cached source hash", h, e)
		}
		if n, e := m.Size(s); e != nil || n != 3 {
			t.Fatal(n, e)
		}
		if ok, e := m.Contains(s, 0); e != nil || !ok {
			t.Fatal("strchr terminator", e)
		}
		if ok, e := m.Contains(s, 'o'); e != nil || !ok {
			t.Fatal(e)
		}
		if ok, e := m.Contains(s, 'x'); e != nil || ok {
			t.Fatal(e)
		}
		a, b := internSymbol(t, m, "\x80"), internSymbol(t, m, "a")
		if ok, e := m.Less(a, b); e != nil || !ok {
			t.Fatal("source signed char order", e)
		}
		empty := internSymbol(t, m, "")
		if ok, e := m.Less(a, empty); e != nil || !ok {
			t.Fatal("signed high byte precedes NUL", e)
		}
		if round, e := m.FromToken(s.Token()); e != nil || round != s {
			t.Fatal("opaque identity roundtrip", e)
		}
		if e := m.AssignString(&s, nil); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("assignment null precondition", e)
		}
		text := "boo"
		if e := m.AssignString(&s, &text); e != nil || s != internSymbol(t, m, "boo") {
			t.Fatal("assignment", e)
		}
	}
}

func TestSymbolOwnerFailureLifetimeAndWarmIntern(t *testing.T) {
	for prefix := int64(0); prefix < 20; prefix++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(prefix)
		m, e := NewSymbolManager(SymbolConfig{Budget: b, PageSize: 64, DescriptorPageSize: 2})
		if e == nil {
			m.Close()
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("constructor leak", prefix, b.Used())
		}
	}
	for prefix := int64(0); prefix < 24; prefix++ {
		b := arena.NewBudget(1 << 20)
		m := symbolManagerTest(t, b, false)
		old := internSymbol(t, m, "old")
		destination := old
		text := "a spelling substantially bigger than this small source page payload allocation"
		b.FailAfter(prefix)
		e := m.AssignString(&destination, &text)
		b.FailAfter(-1)
		if e != nil && (!errors.Is(e, arena.ErrInjectedFailure) || destination != old || symbolText(t, m, old) != "old") {
			t.Fatal("publication failure", prefix, e)
		}
		want := internSymbol(t, m, text)
		if e == nil && destination != want {
			t.Fatal("successful publication")
		}
		if err := m.Close(); err != nil || b.Used() != 0 {
			t.Fatal("close leak", prefix, err, b.Used())
		}
		if _, e := m.String(old); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("closed owner", e)
		}
	}
	m, n := symbolManagerTest(t, nil, false), symbolManagerTest(t, nil, false)
	a, b := internSymbol(t, m, "same"), internSymbol(t, n, "same")
	if a == b {
		t.Fatal("foreign owner identity collision")
	}
	if _, e := n.String(a); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal("foreign symbol accepted", e)
	}
	if _, e := n.FromToken(a.Token()); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal("foreign token accepted", e)
	}
	for i := 0; i < 12; i++ {
		internSymbol(t, m, string([]byte{'a', byte('a' + i)}))
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if s, e := m.Intern("same"); e != nil || s != a {
			panic("warm intern")
		}
	}); allocations != 0 {
		t.Fatal("warm allocations", allocations)
	}
}

func TestSymbolConcurrentInternIdentity(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	var wg sync.WaitGroup
	var results [8]Symbol
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				s, e := m.Intern("shared")
				if e != nil {
					t.Error(e)
					return
				}
				results[i] = s
			}
		}(i)
	}
	wg.Wait()
	for _, s := range results {
		if s != results[0] {
			t.Fatal("critical intern identity")
		}
	}
}

// An independent bounded oracle for source signed-char ordering. Once the
// first byte matched, source checks termination before comparing the next
// byte, so a proper nonempty prefix is smaller even before a negative byte.
func sourceSymbolOrderOracle(a, b string) bool {
	if a == b {
		return false
	}
	if len(a) == 0 {
		return int8(0) < int8(b[0])
	}
	if len(b) == 0 {
		return int8(a[0]) < int8(0)
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return int8(a[i]) < int8(b[i])
		}
	}
	return len(a) < len(b)
}

func TestSymbolIndependentBytesAndPrefixOrder(t *testing.T) {
	for _, width := range []uint32{4, 8} {
		b := arena.NewBudget(1 << 20)
		m, e := NewSymbolManager(SymbolConfig{Budget: b, Alignment: width, PageSize: 64, DescriptorPageSize: 2})
		if e != nil {
			t.Fatal(e)
		}
		var texts [100]string
		var symbols [100]Symbol
		rng := rand.New(rand.NewSource(412))
		alphabet := []byte{'a', 'b', 'z', 1, 0x7f, 0x80, 0xff}
		for i := range texts {
			buf := make([]byte, rng.Intn(9))
			for j := range buf {
				buf[j] = alphabet[rng.Intn(len(alphabet))]
			}
			texts[i] = string(buf)
			symbols[i] = internSymbol(t, m, texts[i])
			if symbolText(t, m, symbols[i]) != texts[i] {
				t.Fatal("byte spelling")
			}
			for j := 0; j < i; j++ {
				if (symbols[i] == symbols[j]) != (texts[i] == texts[j]) {
					t.Fatal("independent identity")
				}
			}
			// Source hash prefix is real region backing, not a second per-symbol
			// scalar hash field, and source64's upper size_t bytes are zero.
			ref := arena.BytesRegionRef{Slot: arena.Slot(symbols[i].Value), Generation: symbols[i].Generation}
			n, e := m.region.Size(ref)
			if e != nil || n != width+uint32(len(buf))+1 {
				t.Fatal("source spelling allocation", n, e)
			}
			h, e := m.Hash(symbols[i])
			if e != nil {
				t.Fatal(e)
			}
			for offset := uint32(0); offset < width; offset++ {
				got, e := m.region.Read(ref, offset)
				want := byte(0)
				if offset < 4 {
					want = byte(h >> (offset * 8))
				}
				if e != nil || got != want {
					t.Fatal("hash prefix", got, want, e)
				}
			}
		}
		for i := range texts {
			for j := range texts {
				got, e := m.Less(symbols[i], symbols[j])
				if e != nil || got != sourceSymbolOrderOracle(texts[i], texts[j]) {
					t.Fatal("independent source order", i, j, got, e)
				}
			}
		}
		prefix, negative := internSymbol(t, m, "a"), internSymbol(t, m, "a\x80")
		if got, e := m.Less(prefix, negative); e != nil || !got {
			t.Fatal("source terminator-before-next-comparison", e)
		}
		s := m.Stats()
		d := s.Region.Descriptors
		owned := s.OwnerBytes + s.Table.OwnerBytes + s.Table.TableBytes + s.Region.OwnerBytes + s.Region.PayloadBytes + s.Region.PageHeaderBytes + s.Region.DirectoryBytes + s.Region.LogBytes + s.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
		if owned != b.Used() {
			t.Fatal("complete owner graph accounting", owned, b.Used())
		}
		if e := m.Close(); e != nil || b.Used() != 0 {
			t.Fatal("owner graph close", e, b.Used())
		}
		if s := m.Stats(); s.OwnerBytes != 0 || s.Symbols != 0 || s.Table.OwnerBytes != 0 || s.Region.OwnerBytes != 0 {
			t.Fatal("closed graph stats", s)
		}
	}
}

func TestSymbolDuplicateLookupAndInvalidTokens(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	m := symbolManagerTest(t, b, false)
	var first Symbol
	for _, text := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		s := internSymbol(t, m, text)
		if first.Value == 0 {
			first = s
		}
	}
	b.FailAfter(0)
	if s, e := m.Intern("a"); e != nil || s != first {
		t.Fatal("duplicate lookup should not allocate", s, e)
	}
	b.FailAfter(-1)
	if s, e := m.Intern("a"); e != nil || s != first {
		t.Fatal("stable duplicate identity", e)
	}
	if s, e := m.InternNullable(nil); e != nil || s != (Symbol{}) {
		t.Fatal("null constructor", e)
	}
	for _, invalid := range []Symbol{{Tag: 4}, {owner: 1}, {Tag: 1, owner: 1}, {Tag: 1, Generation: 1}, {owner: m.owner, Value: math.MaxUint32, Generation: 1}} {
		if _, e := m.FromToken(invalid.Token()); e == nil {
			t.Fatal("forged token accepted", invalid)
		}
	}
	marked, e := first.Mark()
	if e != nil {
		t.Fatal(e)
	}
	for _, same := range []Symbol{marked, DummySymbol()} {
		if less, e := m.Less(same, same); e != nil || less {
			t.Fatal("source identity check precedes marked-data dereference", less, e)
		}
	}
	if plain, e := marked.unmark(); e != nil || plain != first {
		t.Fatal("string mark", e)
	}
	if _, e := marked.Mark(); !errors.Is(e, ErrInvalidSymbol) {
		t.Fatal("double mark", e)
	}
	if _, e := first.unmark(); !errors.Is(e, ErrInvalidSymbol) {
		t.Fatal("unmarked unmark", e)
	}
	for _, s := range []Symbol{marked, DummySymbol()} {
		if _, e := m.Hash(s); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal("marked hash", e)
		}
	}
	if ok, e := m.EqualString(NumericalSymbol(math.MaxUint32), func() *string { s := "k!-1\x00tail"; return &s }()); e != nil || !ok {
		t.Fatal("numerical C string equality", e)
	}
}
