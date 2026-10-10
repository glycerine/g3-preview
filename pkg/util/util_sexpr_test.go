// sexpr has no direct registered source tests; these freeze defined source
// branches and explicit undefined-behavior adaptations with independent oracles.
package util

import (
	"errors"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func sxMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func sxOwner(t *testing.T, b *arena.Budget) (*SexprManager, *usymbol.Manager, *unumeral.RationalService) {
	t.Helper()
	s, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	sxMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 2}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{Budget: b, PageSize: 2}}}})
	sxMust(t, e)
	m, e := NewSexprManager(SexprConfig{Budget: b, PageSize: 2, Symbols: s, Rationals: q})
	sxMust(t, e)
	t.Cleanup(func() {
		sxMust(t, m.Close())
		sxMust(t, q.Close())
		sxMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("owner bytes", b.Used())
		}
	})
	return m, s, q
}
func sxNumber(t *testing.T, q *unumeral.RationalService, text string) unumeral.OwnedRational {
	v, e := q.NewValue()
	sxMust(t, e)
	sxMust(t, q.SetString(&v, text))
	return v
}
func sxRender(t *testing.T, m *SexprManager, h SexprHandle, want string, err error) {
	t.Helper()
	got, e := m.Display(h)
	if got != want || !errors.Is(e, err) {
		t.Fatalf("display %q %v want %q %v", got, e, want, err)
	}
}
func TestSexprKindsFactoriesAndDefinedDisplay(t *testing.T) {
	m, s, q := sxOwner(t, arena.NewBudget(1<<24))
	v := sxNumber(t, q, "-7/5")
	defer q.Del(&v)
	n, e := m.MkNumeral(&v, 17, 23)
	sxMust(t, e)
	text, e := m.MkString("quote\" back\\\n\x00hidden", 1, 2)
	sxMust(t, e)
	sym := internSymbol(t, s, "alpha")
	a, e := m.MkSymbol(sym, 3, 4)
	sxMust(t, e)
	key, e := m.MkKeyword(sym, 5, 6)
	sxMust(t, e)
	empty, e := m.MkComposite(nil, math.MaxUint32, math.MaxUint32)
	sxMust(t, e)
	root, e := m.MkComposite([]SexprHandle{n, text, a, key, empty}, 7, 8)
	sxMust(t, e)
	sxRender(t, m, empty, ")", nil)
	sxRender(t, m, root, "(-7/5 \"quote\\\" back\\\n\" alpha alpha ))", nil)
	sxRender(t, m, n, "-7/5", ErrSexprDisplayContinuation)
	atom, e := m.DisplayAtom(text)
	sxMust(t, e)
	if atom != "\"quote\\\" back\\\n\"" {
		t.Fatal(atom)
	}
	for i, h := range []SexprHandle{root, n, text, key, a} {
		k, e := m.Kind(h)
		sxMust(t, e)
		if k != []SexprKind{SexprComposite, SexprNumeral, SexprString, SexprKeyword, SexprSymbol}[i] {
			t.Fatal(k)
		}
	}
	if rc, e := m.RefCount(root); e != nil || rc != 0 {
		t.Fatal(rc, e)
	}
	if rc, e := m.RefCount(n); e != nil || rc != 1 {
		t.Fatal(rc, e)
	}
	if line, e := m.Line(n); e != nil || line != 17 {
		t.Fatal(line, e)
	}
	if pos, e := m.Pos(n); e != nil || pos != 23 {
		t.Fatal(pos, e)
	}
	copy, e := m.Numeral(n)
	sxMust(t, e)
	sxMust(t, q.SetInt64(&copy, 99))
	sxMust(t, q.Del(&copy))
	sxRender(t, m, n, "-7/5", ErrSexprDisplayContinuation)
	if _, e = m.Child(root, 5); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if _, e = m.Numeral(text); !errors.Is(e, ErrSexprKind) {
		t.Fatal(e)
	}
	got, e := m.String(text)
	sxMust(t, e)
	if got != "quote\" back\\\n\x00hidden" {
		t.Fatal(got)
	}
	sxMust(t, m.IncRef(root))
	sxMust(t, m.DecRef(root))
	if _, e = m.Kind(n); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
}
func TestSexprBVSourceWidthQuirks(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	for _, tc := range []struct {
		v     string
		width uint32
		want  string
	}{{"0", 0, "#x"}, {"0", 8, "#x00"}, {"-1", 8, "#x00"}, {"256", 4, "#x100"}, {"31", 3, "#b11111"}, {"5", 8, "#x05"}, {"5", 7, "#b0000101"}} {
		v := sxNumber(t, q, tc.v)
		h, e := m.MkBVNumeral(&v, tc.width, 0, 0)
		sxMust(t, e)
		sxMust(t, q.Del(&v))
		sxRender(t, m, h, tc.want, ErrSexprDisplayContinuation)
		sxMust(t, m.IncRef(h))
		sxMust(t, m.DecRef(h))
	}
	v := sxNumber(t, q, "1/2")
	h, e := m.MkBVNumeral(&v, 4, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&v))
	sxFractionalBVSourceProfile(t, m, h)
}
func TestSexprRCWrapAndDeletionLIFO(t *testing.T) {
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	a, e := m.MkString("a", 0, 0)
	sxMust(t, e)
	b, e := m.MkString("b", 0, 0)
	sxMust(t, e)
	root, e := m.MkComposite([]SexprHandle{a, b, a}, 0, 0)
	sxMust(t, e)
	sxMust(t, m.IncRef(root))
	sxMust(t, m.DecRef(root))
	// Composite is recycled first, then a, then b (ascending pushes b,a; LIFO pops a,b).
	x, e := m.MkString("x", 0, 0)
	sxMust(t, e)
	if x.slot != b.slot {
		t.Fatal("deletion order", x, a)
	}
	r, _ := m.record(x)
	r.refs = math.MaxUint32
	if n := testing.AllocsPerRun(20, func() {
		r.refs = math.MaxUint32
		if e := m.IncRef(x); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	if count, e := m.RefCount(x); e != nil || count != 0 {
		t.Fatal(count, e)
	}
	if e := m.DecRef(x); !errors.Is(e, ErrSexprRefCount) {
		t.Fatal(e)
	}
	sxMust(t, m.IncRef(x))
	sxMust(t, m.DecRef(x))
}
func TestSexprConcreteRefAndVectorActions(t *testing.T) {
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	a, e := m.MkString("a", 0, 0)
	sxMust(t, e)
	b, e := m.MkString("b", 0, 0)
	sxMust(t, e)
	r, e := m.NewRef(a)
	sxMust(t, e)
	copy, e := m.CloneRef(&r)
	sxMust(t, e)
	if n, _ := m.RefCount(a); n != 2 {
		t.Fatal(n)
	}
	sxMust(t, m.AssignRef(&r, a))
	sxMust(t, m.ResetRef(&copy))
	stolen, e := m.StealRef(&r)
	sxMust(t, e)
	if stolen != a || !r.IsNull() {
		t.Fatal(stolen, r)
	}
	vec, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, vec.Reserve(3))
	if vec.Size() != 3 {
		t.Fatal("source reserve resizes", vec.Size())
	}
	sxMust(t, vec.Set(0, a))
	sxMust(t, vec.Set(1, b))
	sxMust(t, vec.Set(2, a))
	sxMust(t, m.DecRef(stolen))
	sxMust(t, vec.Reverse())
	cl, e := vec.Clone()
	sxMust(t, e)
	sxMust(t, cl.EraseHandle(a))
	if cl.Size() != 2 {
		t.Fatal(cl.Size())
	}
	sxMust(t, cl.Close())
	sxMust(t, vec.Shrink(1))
	sxMust(t, vec.Reset())
	sxMust(t, vec.Close())
	if _, e = m.Kind(a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
}
func TestSexprHandlesAndBoundedDAG(t *testing.T) {
	sxMust(t, arena.ValidateRecordType(reflect.TypeOf(SexprHandle{})))
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	var handles []SexprHandle
	var expected []string
	for i := 0; i < 25; i++ {
		var h SexprHandle
		var e error
		var want string
		if i < 5 {
			h, e = m.MkString(string(rune('a'+i)), 0, 0)
			want = "\"" + string(rune('a'+i)) + "\""
		} else {
			ids := []int{(i * 3) % i, (i*7 + 1) % i}
			h, e = m.MkComposite([]SexprHandle{handles[ids[0]], handles[ids[1]]}, 0, 0)
			want = "(" + expected[ids[0]] + " " + expected[ids[1]] + ")"
		}
		sxMust(t, e)
		sxMust(t, m.IncRef(h))
		handles = append(handles, h)
		expected = append(expected, want)
		if i >= 5 {
			sxRender(t, m, h, want, nil)
		}
	}
	for _, h := range handles {
		sxMust(t, m.DecRef(h))
	}
	if m.Stats().Live != 0 {
		t.Fatal(m.Stats())
	}
}

func TestSexprStringBytesAndWrapperRemainingBranches(t *testing.T) {
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	text := "λ☃\"\x00tail"
	h, e := m.MkString(text, 0, 0)
	sxMust(t, e)
	got, e := m.String(h)
	sxMust(t, e)
	if got != text {
		t.Fatal("byte-preserving string", got)
	}
	sxRender(t, m, h, "\"λ☃\\\"\"", ErrSexprDisplayContinuation)
	c, e := m.MkCString(text, 0, 0)
	sxMust(t, e)
	got, e = m.String(c)
	sxMust(t, e)
	if got != "λ☃\"" {
		t.Fatal(got)
	}
	r, e := m.NewRef(h)
	sxMust(t, e)
	moved, e := m.MoveRef(&r)
	sxMust(t, e)
	if !r.IsNull() {
		t.Fatal(r)
	}
	equal, e := m.EqualRef(&r, &moved)
	sxMust(t, e)
	if equal {
		t.Fatal("ref identity")
	}
	sxMust(t, m.SwapRef(&r, &moved))
	sxMust(t, m.AssignRefFrom(&moved, &r))
	sxMust(t, m.CloseRef(&r))
	sxMust(t, m.CloseRef(&moved))
	v, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, v.ResizeFill(4, c))
	if v.Size() != 4 || v.Capacity() < v.Size() {
		t.Fatal("native vector size/capacity", v.Size(), v.Capacity())
	}
	element, e := v.Element(0)
	sxMust(t, e)
	sxMust(t, v.SetElement(element, c))
	sxMust(t, v.SwapIndices(0, 3))
	sxMust(t, v.PopBack())
	if _, e = v.GetElement(element); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	def, e := v.GetDefault(100, c)
	sxMust(t, e)
	if def != c {
		t.Fatal(def)
	}
	found, e := v.Contains(c)
	sxMust(t, e)
	if !found {
		t.Fatal(found)
	}
	dst := make([]SexprHandle, 3)
	n, e := v.CopyInto(dst)
	sxMust(t, e)
	if n != 3 || dst[0] != c {
		t.Fatal(dst, n)
	}
	out, e := v.Move()
	sxMust(t, e)
	if !v.Empty() || out.Size() != 3 {
		t.Fatal(out.Size())
	}
	sxMust(t, v.Assign(out))
	sxMust(t, out.Finalize())
	sxMust(t, out.Close())
	sxMust(t, v.Finalize())
	sxMust(t, v.Close())
	sxMust(t, arena.ValidateRecordType(reflect.TypeOf(SexprVectorElement{})))
}

func TestSexprFailurePrefixesImmediateOwnership(t *testing.T) {
	for op := 0; op < 7; op++ {
		success := false
		failures := 0
		for prefix := int64(0); prefix < 256; prefix++ {
			b := arena.NewBudget(1 << 25)
			m, _, q := sxOwner(t, b)
			v := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579/47890485652059026823698344598447161988085597568237573")
			if _, e := q.FormatInto(&v, nil); !errors.Is(e, unumeral.ErrShortBuffer) {
				t.Fatal(e)
			} // Prime only legitimate numeric formatting scratch.
			a, e := m.MkNumeral(&v, 0, 0)
			sxMust(t, e)
			sxMust(t, m.IncRef(a))
			atom, e := m.MkString("hello", 0, 0)
			sxMust(t, e)
			sxMust(t, m.IncRef(atom))
			var display SexprHandle
			if op == 3 {
				integer := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579")
				bv, e := m.MkBVNumeral(&integer, 8, 0, 0)
				sxMust(t, e)
				sxMust(t, q.Del(&integer))
				display, e = m.MkComposite([]SexprHandle{a, bv, atom}, 0, 0)
				sxMust(t, e)
				sxMust(t, m.IncRef(display))
			}
			vec, e := m.NewRefVector()
			sxMust(t, e)
			sxMust(t, vec.Append([]SexprHandle{a, atom}))
			if op == 6 {
				sxMust(t, vec.Finalize())
			}
			before, e := q.Stats()
			sxMust(t, e)
			cachedBefore, _ := numericServiceScratchRoots(t, q)
			records := m.Stats().Live
			rca, _ := m.RefCount(a)
			rcatom, _ := m.RefCount(atom)
			b.FailAfter(prefix)
			var h SexprHandle
			var clone *SexprRefVector
			var rendered string
			switch op {
			case 0:
				h, e = m.MkNumeral(&v, 0, 0)
			case 1:
				h, e = m.MkString("a long copied UTF8 string λ☃", 0, 0)
			case 2:
				h, e = m.MkComposite([]SexprHandle{a, atom, a}, 0, 0)
			case 3:
				rendered, e = m.Display(display)
			case 4:
				e = vec.PushBack(a)
			case 5:
				clone, e = vec.Clone()
			case 6:
				e = vec.ResizeFill(7, a)
			}
			if op == 3 {
				integer, _ := new(big.Int).SetString("1684996666696914987166688442938726917102321526408785780068975640579", 10)
				want := "(1684996666696914987166688442938726917102321526408785780068975640579/47890485652059026823698344598447161988085597568237573 #x" + integer.Text(16) + " \"hello\")"
				if !strings.HasPrefix(want, rendered) || (e == nil && rendered != want) {
					t.Fatal("rendered failure prefix", prefix, rendered, want, e)
				}
			}
			// Inspect the complete public numeric live counts before String/Del/Close.
			after, err := q.Stats()
			sxMust(t, err)
			added := uint32(0)
			if e == nil && op == 0 {
				added = 2
			}
			cachedAfter, naturalAfter := numericServiceScratchRoots(t, q)
			expectedCells := before.LiveCells - cachedBefore + cachedAfter + added
			if after.LiveCells != expectedCells || after.LiveNumericObjects != expectedCells || after.RetainedCellDescriptors != 0 || after.NaturalPayloads != naturalAfter || after.LivePayloads != expectedCells+naturalAfter+after.PendingCellObjects {
				t.Fatal("temporary numeric leak", op, prefix, before, after, added)
			}
			if e != nil && !numericStorageReservationRefusal(e) {
				t.Fatal(op, prefix, e)
			}
			if e != nil {
				failures++
			} else {
				success = true
			}
			wantA, wantAtom := rca, rcatom
			if op == 2 && e == nil {
				wantA += 2
				wantAtom++
			}
			if op == 4 && e == nil {
				wantA++
			}
			if op == 5 && e == nil {
				wantA++
				wantAtom++
			}
			if op == 6 {
				wantA += vec.Size()
			}
			if got, _ := m.RefCount(a); got != wantA {
				t.Fatal("source prefix refs", op, prefix, got, wantA)
			}
			if got, _ := m.RefCount(atom); got != wantAtom {
				t.Fatal(op, prefix, got, wantAtom)
			}
			expectedLive := records
			if !h.IsNull() {
				expectedLive++
			}
			if m.Stats().Live != expectedLive {
				t.Fatal("failed factory published record", op, prefix, m.Stats())
			}
			b.FailAfter(-1)
			if clone != nil {
				sxMust(t, clone.Close())
			}
			sxMust(t, vec.Close())
			sxMust(t, m.Close())
			sxMust(t, q.Del(&v))
			sxMust(t, q.Close())
			sxMust(t, m.Config.Symbols.Close())
			if b.Used() != 0 {
				t.Fatal("owned storage leak", op, prefix, b.Used())
			}
			if success {
				break
			}
		}
		if !success || op < 3 && failures == 0 {
			t.Fatal(op, success, failures)
		}
	}
}
func TestSexprWarmPlateauAndGenerationGuards(t *testing.T) {
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	run := func() {
		h, e := m.MkString("abc", 0, 0)
		if e != nil {
			panic(e)
		}
		if e = m.IncRef(h); e != nil {
			panic(e)
		}
		if e = m.DecRef(h); e != nil {
			panic(e)
		}
	}
	for i := 0; i < 20; i++ {
		run()
	}
	before := m.Config.Budget.Used()
	if n := testing.AllocsPerRun(100, run); n != 0 {
		t.Fatal("warm record path", n)
	}
	if m.Config.Budget.Used() != before {
		t.Fatal("plateau")
	}
	h, e := m.MkString("old", 0, 0)
	sxMust(t, e)
	r, _ := m.record(h)
	r.generation = math.MaxUint32
	h.generation = math.MaxUint32
	sxMust(t, m.IncRef(h))
	sxMust(t, m.DecRef(h))
	next, e := m.MkString("new", 0, 0)
	sxMust(t, e)
	if next.slot == h.slot || m.Stats().Retired != 1 {
		t.Fatal("generation wrap", next, h)
	}
	if _, e = m.Kind(h); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if e = m.IncRef(SexprHandle{}); !errors.Is(e, ErrSexprOwner) {
		t.Fatal(e)
	}
	other, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	if _, e = other.Kind(next); !errors.Is(e, ErrSexprOwner) {
		t.Fatal(e)
	}
	vec, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, vec.PushBack(next))
	vec.epoch = math.MaxUint64
	if e = vec.PushBack(next); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if vec.Size() != 1 {
		t.Fatal(vec.Size())
	}
	sxMust(t, vec.Close())
	sxMust(t, m.Close())
	if _, e = m.Kind(next); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestSexprFactoryRecordBeforeValueCopy(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	m.Config.SlotLimit = 1
	occupied, e := m.MkString("occupied", 0, 0)
	sxMust(t, e)
	v := sxNumber(t, q, "1684996666696914987166688442938726917102321526408785780068975640579/47890485652059026823698344598447161988085597568237573")
	before, e := q.Stats()
	sxMust(t, e)
	m.Config.Budget.FailAfter(0)
	if _, e = m.MkNumeral(&v, 0, 0); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("source record allocation must precede numeric constructor copy", e)
	}
	if _, e = m.MkString(strings.Repeat("x", 40000), 0, 0); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("record allocation before string constructor", e)
	}
	if _, e = m.MkComposite([]SexprHandle{occupied}, 0, 0); !errors.Is(e, arena.ErrSlotLimit) {
		t.Fatal("record allocation before child payload", e)
	}
	after, e := q.Stats()
	sxMust(t, e)
	if after.LiveCells != before.LiveCells || after.LivePayloads != before.LivePayloads {
		t.Fatal(before, after)
	}
	m.Config.Budget.FailAfter(-1)
	sxMust(t, q.Del(&v))
}

func TestSexprFinalizationEpochAndZeroWidthBuffer(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	zero := sxNumber(t, q, "0")
	h, e := m.MkBVNumeral(&zero, 0, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&zero))
	sxRender(t, m, h, "#x", ErrSexprDisplayContinuation)
	vec, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, vec.PushBack(h))
	vec.epoch = math.MaxUint64 - 1
	sxMust(t, vec.Finalize())
	if vec.Size() != 0 || vec.Capacity() != 0 {
		t.Fatal("finalize retains source backing")
	}
	sxMust(t, vec.Close())
	m.nextVector = math.MaxUint64
	before := m.Config.Budget.Used()
	if _, e = m.NewRefVector(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if m.Config.Budget.Used() != before {
		t.Fatal("exhausted identity reserved header")
	}
}

func TestSexprConstructorAndWholePhysicalGraph(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	m, s, q := sxOwner(t, b)
	base := b.Used()
	seen := false
	for prefix := int64(0); prefix < 64; prefix++ {
		b.FailAfter(prefix)
		other, e := NewSexprManager(SexprConfig{Budget: b, PageSize: 2, Symbols: s, Rationals: q})
		b.FailAfter(-1)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if b.Used() != base {
				t.Fatal("constructor leaked owner headers", prefix, b.Used(), base)
			}
		} else {
			sxMust(t, other.Close())
			if b.Used() != base {
				t.Fatal("constructor refund")
			}
			seen = true
			break
		}
	}
	if !seen {
		t.Fatal("constructor never succeeded")
	}
	number := sxNumber(t, q, "7/5")
	h, e := m.MkNumeral(&number, 0, 0)
	sxMust(t, e)
	sxMust(t, q.Del(&number))
	str, e := m.MkString("graph", 0, 0)
	sxMust(t, e)
	root, e := m.MkComposite([]SexprHandle{h, str}, 0, 0)
	sxMust(t, e)
	sxRender(t, m, root, "(7/5 \"graph\")", nil)
	stats := m.Stats()
	charged := stats.OwnerBytes + stats.RecordBytes + stats.DirectoryBytes + stats.ScratchBytes
	// Inline numeral components do not grow borrowed numeric ownership here.
	// FormatInto has its own owner-retained workspace, measured separately.
	managerBytes := charged
	beforeClose := b.Used()
	sxMust(t, m.Close())
	if beforeClose-b.Used() != managerBytes {
		t.Fatal("manager graph accounting", beforeClose-b.Used(), managerBytes, stats)
	}
}
func TestSexprRegisteredEscapedDefaultPrintInputs(t *testing.T) {
	m, _, _ := sxOwner(t, arena.NewBudget(1<<24))
	for _, tc := range []struct{ in, want string }{{"\"hello\"\"world\"\n\n", "\"\\\"hello\\\"\\\"world\\\"\n\n\""}, {"\"hello\"\n\"world\"\n\n", "\"\\\"hello\\\"\n\\\"world\\\"\n\n\""}, {"\n\n\n", "\"\n\n\n\""}, {"\n", "\"\n\""}, {"", "\"\""}} {
		h, e := m.MkString(tc.in, 0, 0)
		sxMust(t, e)
		sxRender(t, m, h, tc.want, ErrSexprDisplayContinuation)
		sxMust(t, m.IncRef(h))
		sxMust(t, m.DecRef(h))
	}
}

func TestSexprAccessorsAndScalarIterator(t *testing.T) {
	m, s, q := sxOwner(t, arena.NewBudget(1<<24))
	num := sxNumber(t, q, "15")
	h, e := m.MkBVNumeral(&num, 4, 11, 12)
	sxMust(t, e)
	sxMust(t, q.Del(&num))
	if width, e := m.BVSize(h); e != nil || width != 4 {
		t.Fatal(width, e)
	}
	ok, e := m.IsBVNumeral(h)
	sxMust(t, e)
	if !ok {
		t.Fatal(ok)
	}
	ok, e = m.IsNumeral(h)
	sxMust(t, e)
	if ok {
		t.Fatal(ok)
	}
	symbol := internSymbol(t, s, ":key")
	key, e := m.MkKeyword(symbol, 0, 0)
	sxMust(t, e)
	got, e := m.Symbol(key)
	sxMust(t, e)
	if got != symbol {
		t.Fatal(got)
	}
	root, e := m.MkComposite([]SexprHandle{h, key}, 0, 0)
	sxMust(t, e)
	children := make([]SexprHandle, 2)
	count, e := m.ChildrenInto(root, children)
	sxMust(t, e)
	if count != 2 || children[0] != h || children[1] != key {
		t.Fatal(children, count)
	}
	v, e := m.NewRefVectorFrom(children)
	sxMust(t, e)
	defer v.Close()
	begin, e := v.Begin()
	sxMust(t, e)
	end, e := v.End()
	sxMust(t, e)
	for i := 0; i < 2; i++ {
		got, e := v.IteratorGet(begin)
		sxMust(t, e)
		if got != children[i] {
			t.Fatal(got, i)
		}
		begin, e = v.IteratorNext(begin)
		sxMust(t, e)
	}
	if begin != end {
		t.Fatal("end iterator", begin, end)
	}
	if _, e = v.IteratorGet(end); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	sxMust(t, v.PushBack(h))
	if _, e = v.IteratorGet(begin); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	reference, e := m.NewRef(root)
	sxMust(t, e)
	owner, e := m.RefManager(&reference)
	sxMust(t, e)
	if owner != m {
		t.Fatal("manager")
	}
	sxMust(t, m.CloseRef(&reference))
	// Symbols have no RC; raw numeral values are copied at factory construction.
	numeric, e := m.MkSymbol(usymbol.Numerical(math.MaxUint32), 0, 0)
	sxMust(t, e)
	sxRender(t, m, numeric, "k!-1", ErrSexprDisplayContinuation)
	null, e := m.MkSymbol(usymbol.Symbol{}, 0, 0)
	sxMust(t, e)
	sxRender(t, m, null, "null", ErrSexprDisplayContinuation)
}

func TestSexprBVGoScratchAndNumeralLifetime(t *testing.T) {
	m, _, q := sxOwner(t, arena.NewBudget(1<<24))
	m.output = make([]byte, 2048)
	zero := sxNumber(t, q, "0")
	widths := []uint32{60, 64, 128, 1025, 8}
	handles := make([]SexprHandle, len(widths))
	for i, width := range widths {
		var e error
		handles[i], e = m.MkBVNumeral(&zero, width, 0, 0)
		sxMust(t, e)
	}
	sxMust(t, q.Del(&zero))
	before := m.Config.Budget.Used()
	roots, e := q.Stats()
	sxMust(t, e)
	m.Config.Budget.FailAfter(0)
	for i, width := range widths {
		prefix, count := "#b", int(width)
		if width%4 == 0 {
			prefix, count = "#x", int(width/4)
		}
		got, err := m.DisplayAtom(handles[i])
		if want := prefix + strings.Repeat("0", count); err != nil || got != want {
			t.Fatal(width, got, err, want)
		}
		if m.Config.Budget.Used() != before || len(m.bvDigits) != 0 {
			t.Fatal("Go scratch changed numeric budget or retained logical digits", width)
		}
	}
	after, e := q.Stats()
	sxMust(t, e)
	if roots.LiveCells != after.LiveCells || roots.LivePayloads != after.LivePayloads {
		t.Fatal("temporary numeral leaked", roots, after)
	}
	if n := testing.AllocsPerRun(50, func() { _, e = m.DisplayAtom(handles[0]) }); n != 1 || e != nil {
		t.Fatal("warm display allocates only its returned string", n, e)
	}
	m.Config.Budget.FailAfter(-1)
}

func TestSexprGoTraversalScratchUnderBudgetRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, _, _ := sxOwner(t, budget)
	const count = 256
	children := make([]SexprHandle, count)
	for i := range children {
		var e error
		children[i], e = m.MkString("x", 0, 0)
		sxMust(t, e)
	}
	wide, e := m.MkComposite(children, 0, 0)
	sxMust(t, e)
	deep, e := m.MkString("y", 0, 0)
	sxMust(t, e)
	for i := 0; i < count; i++ {
		deep, e = m.MkComposite([]SexprHandle{deep}, 0, 0)
		sxMust(t, e)
	}
	root, e := m.MkComposite([]SexprHandle{wide, deep}, 0, 0)
	sxMust(t, e)
	sxMust(t, m.IncRef(root))
	before := budget.Used()
	budget.FailAfter(0)
	text, e := m.String(children[0])
	sxMust(t, e)
	if text != "x" {
		t.Fatal(text)
	}
	wantWide := "(" + strings.Repeat("\"x\" ", count-1) + "\"x\")"
	wantDeep := strings.Repeat("(", count) + "\"y\"" + strings.Repeat(")", count)
	sxRender(t, m, root, "("+wantWide+" "+wantDeep+")", nil)
	if budget.Used() != before {
		t.Fatal("Go output or traversal scratch charged the arena", budget.Used(), before)
	}
	sxMust(t, m.DecRef(root))
	if m.Stats().Live != 0 || len(m.deletion) != 0 {
		t.Fatal("wide/deep deletion left live records or pending work", m.Stats())
	}
	budget.FailAfter(-1)
}

func TestSexprVectorNativeGrowthUnderBudgetRefusal(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, _, _ := sxOwner(t, budget)
	h, e := m.MkString("retained", 0, 0)
	sxMust(t, e)
	sxMust(t, m.IncRef(h))
	before := budget.Used()
	budget.FailAfter(0)
	v, e := m.NewRefVector()
	sxMust(t, e)
	sxMust(t, v.ResizeFill(64, h))
	borrow, e := v.Element(0)
	sxMust(t, e)
	sxMust(t, v.PushBack(h))
	if _, e = v.GetElement(borrow); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("structural growth retained an element borrow", e)
	}
	clone, e := v.Clone()
	sxMust(t, e)
	moved, e := clone.Move()
	sxMust(t, e)
	if !clone.Empty() || moved.Size() != 65 {
		t.Fatal("move did not transfer owned handles", clone.Size(), moved.Size())
	}
	if refs, e := m.RefCount(h); e != nil || refs != 131 {
		t.Fatal("copy/move reference counts", refs, e)
	}
	sxMust(t, v.Reset())
	sxMust(t, v.Resize(3))
	for i := uint32(0); i < v.Size(); i++ {
		node, e := v.Get(i)
		sxMust(t, e)
		if !node.IsNull() {
			t.Fatal("resizing after reset exposed a released handle", i)
		}
	}
	sxMust(t, clone.Close())
	sxMust(t, moved.Finalize())
	sxMust(t, moved.Close())
	sxMust(t, v.Close())
	if refs, e := m.RefCount(h); e != nil || refs != 1 {
		t.Fatal("vector cleanup did not return the caller reference", refs, e)
	}
	if budget.Used() != before {
		t.Fatal("Go vector headers or backing charged the arena", budget.Used(), before)
	}
	sxMust(t, m.DecRef(h))
	budget.FailAfter(-1)
}

// Once a record page exists, child lists and text need no arena reservation.
// The factory must own its child list even if the caller reuses the input slice.
func TestSexprNativePayloadOwnership(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	_, symbols, rationals := sxOwner(t, budget)
	m, e := NewSexprManager(SexprConfig{Budget: budget, PageSize: 16, Symbols: symbols, Rationals: rationals})
	sxMust(t, e)
	defer func() { sxMust(t, m.Close()) }()
	left, e := m.MkString("left\x00tail", 0, 0)
	sxMust(t, e)
	before := budget.Used()
	budget.FailAfter(0)
	defer budget.FailAfter(-1)
	right, e := m.MkString("right", 0, 0)
	sxMust(t, e)
	input := []SexprHandle{left, right, left}
	root, e := m.MkComposite(input, 0, 0)
	sxMust(t, e)
	input[0] = right
	got := make([]SexprHandle, 3)
	n, e := m.ChildrenInto(root, got)
	sxMust(t, e)
	if n != 3 || got[0] != left || got[1] != right || got[2] != left {
		t.Fatal("child order or independent ownership", got, n)
	}
	got[0] = right
	if h, e := m.Child(root, 0); e != nil || h != left {
		t.Fatal("caller output aliases children", h, e)
	}
	if refs, e := m.RefCount(left); e != nil || refs != 2 {
		t.Fatal("duplicate child references", refs, e)
	}
	text, e := m.String(left)
	sxMust(t, e)
	if text != "left\x00tail" || budget.Used() != before {
		t.Fatal("native payload or accounting", text, budget.Used(), before)
	}
	rootRecord, e := m.record(root)
	sxMust(t, e)
	leftRecord, e := m.record(left)
	sxMust(t, e)
	sxMust(t, m.IncRef(root))
	sxMust(t, m.DecRef(root))
	if m.Stats().Live != 0 || rootRecord.children != nil || leftRecord.text != "" {
		t.Fatal("deletion retained payloads", m.Stats())
	}
	if _, e := m.Kind(left); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("deleted child remained valid", e)
	}
	sxMust(t, m.Close())
	if text != "left\x00tail" {
		t.Fatal("returned string lifetime", text)
	}
}
