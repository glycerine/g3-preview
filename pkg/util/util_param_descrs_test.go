package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func pdstr(s string) *string { return &s }
func pdnew(t *testing.T, m *SymbolManager, b *arena.Budget) *ParamDescrs {
	t.Helper()
	d, e := NewParamDescrs(m, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
func pdinsert(t *testing.T, d *ParamDescrs, n string, k ParamKind, descr, def, mod *string) {
	t.Helper()
	if e := d.InsertName(n, k, descr, def, mod); e != nil {
		t.Fatal(e)
	}
}
func TestParamDescrsSourceHistoryDefaultsAndDisplay(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	pdinsert(t, d, "Z_key", ParamUInt, pdstr("Z text"), pdstr(""), nil)
	pdinsert(t, d, "a-key", ParamBool, pdstr("A text"), pdstr("false"), nil)
	pdinsert(t, d, "Z_key", ParamUInt, pdstr("ignored"), pdstr("99"), nil)
	got, e := d.Display(2, true, true)
	want := "  :z-key (unsigned int) Z text (default: )\n  :a-key (bool) A text (default: false)\n"
	if e != nil || got != want {
		t.Fatalf("display %q %v", got, e)
	}
	got, e = d.Display(0, false, false)
	if e != nil || got != "z_key (unsigned int) (default: )\na_key (bool) (default: false)\n" {
		t.Fatal(got, e)
	}
	if e = d.EraseName("Z_key"); e != nil {
		t.Fatal(e)
	}
	pdinsert(t, d, "Z_key", ParamUInt, pdstr("new"), nil, nil)
	n, e := d.Size()
	if e != nil || n != 3 {
		t.Fatal(n, e)
	}
	a, _ := d.ParamName(0)
	c, _ := d.ParamName(2)
	if a != c {
		t.Fatal("erasure must retain history")
	}
	if _, e = d.ParamName(3); e != arena.ErrBounds {
		t.Fatal(e)
	}
	if _, p, e := d.Default(a); e != nil || p {
		t.Fatal(p, e)
	}
	if _, p, e := d.DescriptionName("missing"); e != nil || p {
		t.Fatal(p, e)
	}
	if e = d.Insert(a, ParamBool, nil, nil, nil); e != ErrParamDescrsPrecondition {
		t.Fatal("kind assertion", e)
	}
	if e = d.Insert(NumericalSymbol(7), ParamBool, nil, nil, nil); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
	if k, e := d.Kind(NumericalSymbol(7)); e != nil || k != ParamInvalid {
		t.Fatal(k, e)
	}
}
func TestParamDescrsSourceModuleAndNullKinds(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	pdinsert(t, d, "x", ParamUInt, pdstr("X"), nil, pdstr("sat"))
	for _, test := range []struct {
		in, out string
		k       ParamKind
	}{{"sat.x", "x", ParamUInt}, {"other.x", "other.x", ParamInvalid}, {"sat.x.tail", "sat.x.tail", ParamInvalid}, {"x", "x", ParamUInt}, {".x", ".x", ParamInvalid}} {
		s := internSymbol(t, m, test.in)
		k, e := d.KindInModule(&s)
		if e != nil || k != test.k || symbolText(t, m, s) != test.out {
			t.Fatal(test, k, e)
		}
	}
	pdinsert(t, d, "sat.x", ParamBool, pdstr("direct"), nil, nil)
	s := internSymbol(t, m, "sat.x")
	k, e := d.KindInModule(&s)
	if e != nil || k != ParamBool || symbolText(t, m, s) != "sat.x" {
		t.Fatal(k, e)
	}
	pdinsert(t, d, "empty", ParamString, pdstr("E"), nil, pdstr(""))
	s = internSymbol(t, m, ".empty")
	k, e = d.KindInModule(&s)
	if e != nil || k != ParamString || symbolText(t, m, s) != "empty" {
		t.Fatal(k, e)
	}
	if e = d.Insert(Symbol{}, ParamInvalid, nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	if p, e := d.Contains(Symbol{}); !p || e != nil {
		t.Fatal(p, e)
	}
	s = Symbol{}
	if _, e = d.KindInModule(&s); e != ErrParamDescrsPrecondition {
		t.Fatal("source null strchr guard", e)
	}
	if _, e = d.Display(0, false, false); e != ErrParamDescrsPrecondition {
		t.Fatal("source display null guard", e)
	}
}
func TestParamDescrsSourceKindLabelsAndHelpers(t *testing.T) {
	labels := []string{"unsigned int", "bool", "double", "rational", "rational", "string", "optional-value", "keyword", "symbol", "symbol-list", "sort", "sort-list", "expression", "expression-list", "declaration", "declaration-list", "sorted-variable", "sorted-variable-list", "s-expression", "unknown"}
	for i, w := range labels {
		if ParamKind(i).String() != w {
			t.Fatal(i)
		}
	}
	if ParamKind(100).String() != "unknown" {
		t.Fatal("unknown")
	}
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	for _, f := range []func(*ParamDescrs) error{InsertMaxMemory, InsertMaxSteps, InsertProduceModels, InsertProduceProofs, InsertTimeout} {
		if e := f(d); e != nil {
			t.Fatal(e)
		}
	}
	for _, x := range []struct {
		n, desc string
		k       ParamKind
	}{{"max_memory", "(default: infty) maximum amount of memory in megabytes.", ParamUInt}, {"max_steps", "(default: infty) maximum number of steps.", ParamUInt}, {"produce_models", "(default: false) model generation.", ParamBool}, {"produce_proofs", "(default: false) proof generation.", ParamBool}, {"timeout", "(default: infty) timeout in milliseconds.", ParamUInt}} {
		v, p, e := d.DescriptionName(x.n)
		if e != nil || !p || v != x.desc {
			t.Fatal(x, v, p, e)
		}
		k, e := d.KindName(x.n)
		if e != nil || k != x.k {
			t.Fatal(k, e)
		}
		if _, p, e = d.DefaultName(x.n); p || e != nil {
			t.Fatal(p, e)
		}
	}
}
func TestParamDescrsCopyInsertionOrderLifetimeAndPrefix(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	b := arena.NewBudget(^uint64(0))
	src := pdnew(t, m, b)
	dst := pdnew(t, m, nil)
	for i := 0; i < 12; i++ {
		pdinsert(t, src, fmt.Sprintf("copy%d", i), ParamUInt, pdstr(fmt.Sprintf("D%d", i)), pdstr("0"), nil)
	}
	pdinsert(t, dst, "copy3", ParamUInt, pdstr("retain"), nil, nil)
	order := []Symbol{}
	for key := range src.entries.All() {
		if key != internSymbol(t, m, "copy3") {
			order = append(order, key)
		}
	}
	if e := dst.Copy(src); e != nil {
		t.Fatal(e)
	}
	for i, s := range order {
		v, e := dst.ParamName(uint32(i + 1))
		if e != nil || v != s {
			t.Fatal("insertion order replay", i, e)
		}
	}
	before := dst.Stats()
	if e := dst.Copy(dst); e != nil || dst.Stats() != before {
		t.Fatal("self copy", e)
	}
	v, _, _ := dst.DescriptionName("copy3")
	if v != "retain" {
		t.Fatal(v)
	}
	if e := src.Close(); e != nil || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
	v, p, e := dst.DescriptionName("copy4")
	if e != nil || !p || v != "D4" {
		t.Fatal(v, p, e)
	}
}
func TestParamDescrsDictionaryBehaviorAndHistory(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	live := map[string]string{}
	history := []Symbol{}
	seed := uint32(91)
	for step := 0; step < 1800; step++ {
		seed = seed*1664525 + 1013904223
		name := fmt.Sprintf("key%d", seed%75)
		key := internSymbol(t, m, name)
		if seed&4 == 0 {
			if e := d.Erase(key); e != nil {
				t.Fatal(e)
			}
			delete(live, name)
		} else {
			value := fmt.Sprint(step)
			_, existed := live[name]
			if e := d.Insert(key, ParamString, &value, nil, nil); e != nil {
				t.Fatal(e)
			}
			if !existed {
				live[name] = value
				history = append(history, key)
			}
		}
		for name, want := range live {
			got, present, e := d.DescriptionName(name)
			if e != nil || !present || got != want {
				t.Fatal(step, name, got, want, present, e)
			}
		}
		if got := d.Stats().Live; got != uint32(len(live)) {
			t.Fatal(step, got, len(live))
		}
		n, e := d.Size()
		if e != nil || int(n) != len(history) {
			t.Fatal(step, n, len(history), e)
		}
		for i, want := range history {
			got, e := d.ParamName(uint32(i))
			if e != nil || got != want {
				t.Fatal(step, i, e)
			}
		}
	}
}
func TestParamDescrsBudgetFailuresAndWarmDuplicate(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	key := internSymbol(t, m, "key")
	fail, success := 0, 0
	for prefix := int64(0); prefix < 22; prefix++ {
		b := arena.NewBudget(^uint64(0))
		d := pdnew(t, m, b)
		base := b.Used()
		b.FailAfter(prefix)
		e := d.Insert(key, ParamUInt, pdstr(strings.Repeat("D", 70)), pdstr("99"), pdstr("mod"))
		b.FailAfter(-1)
		if e != nil {
			fail++
			n, _ := d.Size()
			p, _ := d.Contains(key)
			if n != 0 || (p && d.Texts.Stats().Live != 3) || (!p && d.Texts.Stats().Live != 0) {
				t.Fatal(prefix, n, p, d.Texts.Stats())
			}
			if b.Used() < base {
				t.Fatal("lost owner")
			}
		} else {
			success++
		}
		d.Close()
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
	}
	if fail == 0 || success == 0 {
		t.Fatal(fail, success)
	}
	b := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, b)
	pdinsert(t, d, "key", ParamUInt, pdstr("description"), nil, nil)
	base := b.Used()
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := d.Insert(key, ParamUInt, nil, nil, nil); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	if b.Used() != base {
		t.Fatal("duplicate charged")
	}
	b.FailAfter(-1)
	d.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	for p := int64(0); p < 6; p++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(p)
		d, e := NewParamDescrs(m, ParamDescrsConfig{Budget: b})
		if e == nil {
			d.Close()
		}
		if b.Used() != 0 {
			t.Fatal("constructor leak", p, b.Used())
		}
	}
}

func pdtotal(s ParamDescrsStats) uint64 {
	v := s.Text
	a := v.Descriptors
	return s.OwnerBytes + s.TableBytes + s.NamesBytes + v.OwnerBytes + v.PayloadBytes + v.BlockMetadataBytes + v.DirectoryBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
}
func TestParamDescrsOwnershipNullableMetadataAndGuards(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	other := symbolManagerTest(t, nil, false)
	d := pdnew(t, m, nil)
	raw := "description\x00ignored"
	empty := ""
	pdinsert(t, d, "MiXeD-name", ParamKind(100), &raw, &empty, nil)
	raw = "mutated"
	if v, p, e := d.DescriptionName("MiXeD-name"); e != nil || !p || v != "description" {
		t.Fatal(v, p, e)
	}
	if p, e := d.ContainsName("mixed_name"); p || e != nil {
		t.Fatal("wrappers must not normalize", p, e)
	}
	if v, p, e := d.DefaultName("MiXeD-name"); e != nil || !p || v != "" {
		t.Fatal(v, p, e)
	}
	fk := internSymbol(t, other, "foreign")
	if e := d.Insert(fk, ParamUInt, nil, nil, nil); e != ErrSymbolOwner {
		t.Fatal(e)
	}
	if _, e := d.Contains(fk); e != ErrSymbolOwner {
		t.Fatal(e)
	}
	foreign := pdnew(t, other, nil)
	if e := d.Copy(foreign); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
	copied := ParamDescrs{Self: d, manager: m}
	if _, e := copied.KindName("MiXeD-name"); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
	if e := copied.Close(); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
	if e := d.Insert(DummySymbol(), ParamUInt, nil, nil, nil); e != ErrInvalidSymbol {
		t.Fatal(e)
	}
	pdinsert(t, d, "undescribed", ParamUInt, nil, nil, nil)
	if _, e := d.Display(0, false, false); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
	b := arena.NewBudget(^uint64(0))
	owned := pdnew(t, m, b)
	pdinsert(t, owned, "owned", ParamUInt, pdstr("text"), nil, nil)
	value, _, _ := owned.DescriptionName("owned")
	m.Close()
	if _, e := owned.KindName("owned"); e != arena.ErrClosed {
		t.Fatal(e)
	}
	if e := owned.Close(); e != nil || b.Used() != 0 || value != "text" {
		t.Fatal(e, b.Used(), value)
	}
	if _, e := NewParamDescrs(other, ParamDescrsConfig{Text: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}); e != ErrParamDescrsPrecondition {
		t.Fatal(e)
	}
}
func TestParamDescrsCopyDisplayFaultPrefixesAndPlateau(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	src := pdnew(t, m, nil)
	for i := 0; i < 16; i++ {
		pdinsert(t, src, fmt.Sprintf("merge%d", i), ParamString, pdstr(strings.Repeat("T", 25)), pdstr("0"), nil)
	}
	failures, successes, partial := 0, 0, 0
	for prefix := int64(0); prefix < 90; prefix++ {
		b := arena.NewBudget(^uint64(0))
		dst := pdnew(t, m, b)
		b.FailAfter(prefix)
		e := dst.Copy(src)
		b.FailAfter(-1)
		s := dst.Stats()
		if e != nil {
			failures++
			if s.Live > 0 {
				partial++
			}
		} else {
			successes++
		}
		if (s.Live != s.Names && s.Live != s.Names+1) || s.Text.Live != 2*s.Live || pdtotal(s) != b.Used() {
			t.Fatal(prefix, e, s, b.Used())
		}
		order := []Symbol{}
		for key := range src.entries.All() {
			order = append(order, key)
		}
		for i := uint32(0); i < s.Names; i++ {
			got, _ := dst.ParamName(i)
			if got != order[i] {
				t.Fatal("merge failure source prefix", prefix, i)
			}
		}
		dst.Close()
		if b.Used() != 0 {
			t.Fatal(prefix, b.Used())
		}
	}
	if failures == 0 || successes == 0 || partial == 0 {
		t.Fatal(failures, successes, partial)
	}
	b := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, b)
	for i := 0; i < 16; i++ {
		pdinsert(t, d, fmt.Sprintf("merge%d", i), ParamString, pdstr("D"), nil, nil)
	}
	base := b.Used()
	for prefix := int64(0); prefix < 12; prefix++ {
		b.FailAfter(prefix)
		_, e := d.Display(1, false, true)
		b.FailAfter(-1)
		if b.Used() != base {
			t.Fatal("display scratch leak", prefix, e, b.Used(), base)
		}
	}
	// History intentionally keeps growing after erase/reinsert. Metadata pages
	// plateau independently, while all live descriptor records disappear on erase.
	key := internSymbol(t, m, "plateau")
	textBase := uint64(0)
	for round := 0; round < 60; round++ {
		if e := d.Insert(key, ParamBool, pdstr("payload"), pdstr(""), nil); e != nil {
			t.Fatal(e)
		}
		d.Erase(key)
		st := d.Stats()
		v := st.Text
		owned := v.OwnerBytes + v.PayloadBytes + v.BlockMetadataBytes + v.DirectoryBytes + v.Descriptors.OwnerBytes + v.Descriptors.RecordBytes + v.Descriptors.MetadataBytes + v.Descriptors.DirectoryBytes
		if round == 2 {
			textBase = owned
		}
		if round > 2 && owned != textBase {
			t.Fatal("metadata storage plateau", round, owned, textBase)
		}
	}
	d.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestParamDescrsModuleGoScratchAndMetadataPublication(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	b := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, b)
	pdinsert(t, d, "key", ParamUInt, pdstr("description"), nil, pdstr("mod"))
	original := internSymbol(t, m, "mod.key")
	baseline := b.Used()
	failed, passed := 0, 0
	for prefix := int64(0); prefix < 15; prefix++ {
		q := original
		b.FailAfter(prefix)
		kind, e := d.KindInModule(&q)
		b.FailAfter(-1)
		if e != nil {
			failed++
			if q != original {
				t.Fatal("name published before completed lookup")
			}
		} else {
			passed++
			if kind != ParamUInt || symbolText(t, m, q) != "key" {
				t.Fatal(kind, e)
			}
		}
		if b.Used() != baseline {
			t.Fatal("module temporary graph leak", prefix, b.Used(), baseline)
		}
	}
	if failed != 0 || passed != 15 {
		t.Fatal(failed, passed)
	}
	// Reusing freed arena text requires no reservation; ordinary Go name/map
	// growth does not add an artificial Budget failure point.
	k0 := internSymbol(t, m, "warm0")
	k1 := internSymbol(t, m, "warm1")
	k2 := internSymbol(t, m, "warm2")
	z := pdnew(t, m, b)
	if e := z.Insert(k0, ParamString, pdstr("retained"), nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := z.Insert(k1, ParamString, pdstr("retained"), nil, nil); e != nil {
		t.Fatal(e)
	}
	z.Erase(k1)
	before := b.Used()
	b.FailAfter(0)
	e := z.Insert(k2, ParamString, pdstr("retained"), nil, nil)
	b.FailAfter(-1)
	if e != nil {
		t.Fatal(e)
	}
	text, p, e := z.Description(k2)
	n, _ := z.Size()
	if e != nil || !p || text != "retained" || n != 3 || z.Stats().Live != 2 || b.Used() != before || z.Texts.Stats().Live != 2 {
		t.Fatal(text, p, e, n, z.Stats(), b.Used(), before)
	}
}
func TestParamDescrsSourcePrefixUsesGoTemporaryStorage(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	name := internSymbol(t, m, "four.key")
	b := arena.NewBudget(^uint64(0))
	d := pdnew(t, m, b)
	base := b.Used()
	b.FailAfter(0)
	prefix, suffix, split, e := d.splitName(name)
	b.FailAfter(-1)
	if e != nil || !split || symbolText(t, m, prefix) != "four" || symbolText(t, m, suffix) != "key" || b.Used() != base {
		t.Fatal("Go temporary storage must not consume arena budget", split, e, b.Used(), base)
	}
}
func TestParamDescrsCheckedCapacityBoundaries(t *testing.T) {
	if n, e := pdGrowth(0); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if n, e := pdGrowth(5); e != nil || n != 8 {
		t.Fatal(n, e)
	}
	if uint64(^uint(0)>>1) > uint64(^uint32(0)) {
		n := int(uint64(^uint32(0)))
		if _, e := pdGrowth(n); e != arena.ErrBounds {
			t.Fatal("uint32 capacity overflow", e)
		}
	}
	m := symbolManagerTest(t, nil, false)
	for _, cap := range []uint32{3, 6, 7} {
		d, e := NewParamDescrs(m, ParamDescrsConfig{InitialCapacity: cap})
		if e != nil {
			t.Fatal(cap, e)
		}
		paramsMust(t, d.Close())
	}
	b := arena.NewBudget(^uint64(0))
	if _, e := NewParamDescrs(m, ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 3}}); e != arena.ErrConfig || b.Used() != 0 {
		t.Fatal("nested invalid constructor unwind", e, b.Used())
	}
}
