package params

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func pdstr(s string) *string { return &s }
func pdnew(t *testing.T, m *usymbol.Manager, b *arena.Budget) *ParamDescrs {
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
	if paramDescrsStreamSourceDebug {
		bindProcessAssertionRefusal(t)
	}
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
	if e = d.Insert(a, ParamBool, nil, nil, nil); paramDescrsStreamSourceDebug {
		if e != udiag.ErrDebugAssertion {
			t.Fatal("checked kind assertion", e)
		}
	} else if e != nil {
		t.Fatal("release duplicate retains first descriptor", e)
	}
	if k, e := d.Kind(a); e != nil || k != ParamUInt {
		t.Fatal("duplicate changed first kind", k, e)
	}
	other := pdnew(t, m, nil)
	t.Cleanup(func() { paramsMust(t, other.Close()) })
	paramsMust(t, other.Insert(a, ParamBool, pdstr("replacement"), nil, nil))
	if e = d.Copy(other); paramDescrsStreamSourceDebug {
		if e != udiag.ErrDebugAssertion {
			t.Fatal("checked copied-kind assertion", e)
		}
	} else if e != nil {
		t.Fatal("release copied duplicate retains first descriptor", e)
	}
	if k, e := d.Kind(a); e != nil || k != ParamUInt {
		t.Fatal("copied duplicate changed first kind", k, e)
	}
	if n, e := d.Size(); e != nil || n != 3 {
		t.Fatal("duplicate appended name history", n, e)
	}
	if e = d.Insert(usymbol.Numerical(7), ParamBool, nil, nil, nil); paramDescrsStreamSourceDebug {
		if e != udiag.ErrDebugAssertion {
			t.Fatal("checked numerical-name assertion", e)
		}
		if k, e := d.Kind(usymbol.Numerical(7)); e != nil || k != ParamInvalid {
			t.Fatal(k, e)
		}
	} else {
		if e != nil {
			t.Fatal("release numerical-name insert", e)
		}
		if k, e := d.Kind(usymbol.Numerical(7)); e != nil || k != ParamBool {
			t.Fatal(k, e)
		}
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
	if e = d.Insert(usymbol.Symbol{}, ParamInvalid, nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	if p, e := d.Contains(usymbol.Symbol{}); !p || e != nil {
		t.Fatal(p, e)
	}
	s = usymbol.Symbol{}
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
	order := []usymbol.Symbol{}
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
	history := []usymbol.Symbol{}
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
func TestParamDescrsNativeStorageAndWarmDuplicate(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	key := internSymbol(t, m, "key")
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
	return s.OwnerBytes + s.TableBytes + s.NamesBytes
}
func TestParamDescrsOwnershipNullableMetadataAndGuards(t *testing.T) {
	if paramDescrsStreamSourceDebug {
		bindProcessAssertionRefusal(t)
	}
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
	if e := d.Insert(fk, ParamUInt, nil, nil, nil); e != usymbol.ErrOwner {
		t.Fatal(e)
	}
	if _, e := d.Contains(fk); e != usymbol.ErrOwner {
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
	if e := d.Insert(usymbol.Dummy(), ParamUInt, nil, nil, nil); e != usymbol.ErrInvalid {
		t.Fatal(e)
	}
	pdinsert(t, d, "undescribed", ParamUInt, nil, nil, nil)
	if got, e := d.Display(0, false, false); paramDescrsStreamSourceDebug {
		if e != udiag.ErrDebugAssertion {
			t.Fatal("checked description assertion", e)
		}
	} else if e != nil || got != "mixed_name (unknown) (default: )\nundescribed (unsigned int)\n" {
		t.Fatal("release display without requested description", got, e)
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
func TestParamDescrsNativeCopyDisplayAndPlateau(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	src := pdnew(t, m, nil)
	for i := 0; i < 16; i++ {
		pdinsert(t, src, fmt.Sprintf("merge%d", i), ParamString, pdstr(strings.Repeat("T", 25)), pdstr("0"), nil)
	}
	copyBudget := arena.NewBudget(^uint64(0))
	dst := pdnew(t, m, copyBudget)
	copyBudget.FailAfter(0)
	if e := dst.Copy(src); e != nil {
		t.Fatal(e)
	}
	for i, key := range src.Names {
		got, e := dst.ParamName(uint32(i))
		if e != nil || got != key {
			t.Fatal("copy insertion order", i, got, key, e)
		}
	}
	copyBudget.FailAfter(-1)
	dst.Close()
	if copyBudget.Used() != 0 {
		t.Fatal("copy owner leak", copyBudget.Used())
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
	// History grows after erase/reinsert; native metadata does not consume
	// arena budget and all erased dictionary entries disappear.
	key := internSymbol(t, m, "plateau")
	textBase := uint64(0)
	for round := 0; round < 60; round++ {
		if e := d.Insert(key, ParamBool, pdstr("payload"), pdstr(""), nil); e != nil {
			t.Fatal(e)
		}
		d.Erase(key)
		st := d.Stats()
		owned := pdtotal(st)
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
	// Native text and name/map growth require no artificial Budget reservation.
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
	if e != nil || !p || text != "retained" || n != 3 || z.Stats().Live != 2 || b.Used() != before {
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
func TestParamDescrsNativeInitialCapacity(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	for _, cap := range []uint32{3, 6, 7} {
		d, e := NewParamDescrs(m, ParamDescrsConfig{InitialCapacity: cap})
		if e != nil {
			t.Fatal(cap, e)
		}
		paramsMust(t, d.Close())
	}

}
