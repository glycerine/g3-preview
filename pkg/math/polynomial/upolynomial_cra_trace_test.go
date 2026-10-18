package polynomial

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialCRATraceRecordsAndStickyStream(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			m, _, _ := upolySourceOwner(t)
			a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{2})
			ao, e := m.descriptor(a)
			serviceMust(t, e)
			bo, e := m.descriptor(b)
			serviceMust(t, e)
			first, second := unumeral.NewInt(3), unumeral.NewInt(5)
			var text strings.Builder
			bad := factorDegreeRefusedTrace{cause: errors.New("CRA stream refusal")}
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			if refused {
				tags = diagnostic.NewTraceTagsWithWriter(&bad)
			}
			tags.Enable("CRA")
			serviceMust(t, m.modular.BindTrace(tags))
			defer m.modular.BindTrace(nil)
			serviceMust(t, m.combineImages(&ao.state, &first, &bo.state, &second))
			upolyGCDExpect(t, m, b, []int64{7})
			if second != unumeral.NewInt(15) {
				t.Fatal(second)
			}
			if tags.ShouldTrace("CRA") {
				if refused {
					if bad.calls != 1 || bad.flushes != 3 {
						t.Fatal("sticky writes/footer flushes", bad.calls, bad.flushes)
					}
				} else {
					for i, body := range []string{"inv1: 2, inv2: 2\n", "a1: 10, a2: 6\n", "lower: -7, upper: 7\n"} {
						line := []int{677, 682, 692}[i]
						want := fmt.Sprintf("-------- [CRA] CRA_combine_images src/math/polynomial/upolynomial.cpp:%d ---------\n%s------------------------------------------------\n", line, body)
						if !strings.Contains(text.String(), want) {
							t.Fatal("literal record", want, text.String())
						}
					}
				}
			} else if text.Len() != 0 || bad.calls != 0 {
				t.Fatal("erased trace executed")
			}
		})
	}
}

type craTraceReadGuard struct {
	t        *testing.T
	m        *UPolynomialManager
	query    umemory.ObjectRef
	raw      umemory.Block
	offset   uint64
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
	restored bool
	scope    *upolyCRATrace
	frontend bool
	refuse   bool
}

func (w *craTraceReadGuard) Write(p []byte) (int, error) {
	caption := ", inv2: "
	if w.frontend {
		caption = "a1: "
	}
	trigger := string(p) == caption
	if w.refuse {
		trigger = strings.HasPrefix(string(p), "--------")
	}
	if trigger && w.ref == 0 {
		tr := &w.m.craTrace
		if w.scope != nil {
			tr = w.scope
		}
		index := 1
		if w.frontend {
			index = 3
			if tr.values == nil || tr.values[index].IsSmall() {
				return len(p), nil
			}
		}
		if tr.values == nil || tr.declared != map[bool]uint8{false: 3, true: 5}[w.frontend] {
			w.t.Fatal("CRA inverse record crossed later NUM declarations")
		}
		z := w.m.config.Integer
		w.ref = tr.values[index].Cell
		if w.ref == 0 {
			w.t.Fatal("actual CRA inverse is not BIG")
		}
		w.original = z.Cell(w.ref)
		size, e := w.original.Backing.ReadWord(z.Allocator, 0)
		serviceMust(w.t, e)
		// A fresh query-only view has a readable original size and unreadable
		// capacity/limbs. Preserve genuine backing for owning cleanup.
		serviceMust(w.t, z.RuntimeOwner().WriteUint32(w.raw, w.offset, size))
		// Capacity and limb queries remain unreadable in this temporary view.
		cell := w.original
		cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = w.query, w.raw, w.offset
		z.Store(w.ref, cell)
		if w.refuse {
			return 0, errors.New("CRA first-write refusal")
		}
	}
	return len(p), nil
}
func (w *craTraceReadGuard) restore() {
	if w.ref != 0 && !w.restored {
		w.m.config.Integer.Store(w.ref, w.original)
		w.restored = true
	}
}
func TestUPolynomialCRATraceActualInsertionReadGuard(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) { testCRAInsertionReadGuard(t, refused) })
	}
}
func testCRAInsertionReadGuard(t *testing.T, refused bool) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace compiled out")
	}
	q, s := preparationPhysicalFixture(t)
	m := s.manager
	z := q.Integers()
	a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{2})
	ao, e := m.descriptor(a)
	serviceMust(t, e)
	bo, e := m.descriptor(b)
	serviceMust(t, e)
	var first unumeral.Int
	serviceMust(t, z.SetString(&first, "18446744073709551617"))
	defer z.Del(&first)
	second := unumeral.NewInt(3)
	allocator := z.SourceAllocator()
	query, e := allocator.Allocate(4096)
	serviceMust(t, e)
	raw, offset, e := allocator.RawLocation(query)
	serviceMust(t, e)

	w := craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset, refuse: refused}
	defer func() {
		w.restore()
		serviceMust(t, m.CleanupCRATrace())
		serviceMust(t, allocator.Deallocate(4096, query))
	}()
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("CRA")
	serviceMust(t, m.modular.BindTrace(tags))
	body := m.combineImages(&ao.state, &first, &bo.state, &second)
	serviceMust(t, m.modular.BindTrace(nil))
	tr := &m.craTrace
	if !errors.Is(body, umemory.ErrRuntimeUninitialized) || !tr.insertions[1].Active() || m.pendingCRATrace != tr || bo.state.size != 1 || m.values(&bo.state)[0] != unumeral.NewInt(2) || second != unumeral.NewInt(3) {
		t.Fatal("actual insertion read refusal/source frontier", body, w.restored, tr.pending())
	}
	if tr.sourceRemaining != 2 || tr.sourceCause == nil {
		t.Fatal("actual inverse destructor did not protect older NUM", tr.sourceRemaining)
	}
	if e := m.SourceFinalize(); e == nil {
		t.Fatal("unjoined insertion allowed older kernel retirement")
	}
	firstBody, firstSource, remaining := tr.bodyCause, tr.sourceCause, tr.sourceRemaining
	w.restore()
	// Reuse the natural character slot independently. The obsolete CRA
	// insertion owns neither this later generation nor its receipt.
	callerCell := z.Cell(first.Cell)
	callerSize, e := callerCell.Backing.ReadWord(z.Allocator, 0)
	serviceMust(t, e)
	serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, callerSize))
	projected := callerCell
	projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
	z.Store(first.Cell, projected)
	var independent unumeral.NumeralInsertionReceipt
	independentBody := z.InsertSourceIntegerTracked(&upolyDisplaySink{}, &first, &independent, nil)
	z.Store(first.Cell, callerCell)
	if !errors.Is(independentBody, umemory.ErrRuntimeUninitialized) || !independent.Active() {
		t.Fatal("independent insertion profile", independentBody)
	}
	character := *independent.Character
	serviceMust(t, m.CleanupCRATrace())
	if !independent.Active() || !reflect.DeepEqual(character, *independent.Character) {
		t.Fatal("CRA cleanup adopted independent character generation")
	}
	serviceMust(t, independent.Complete(z.Naturals(), false))
	serviceMust(t, independent.Complete(z.Naturals(), true))
	if tr.pending() || m.pendingCRATrace != nil || tr.bodyCause != firstBody || tr.sourceCause != firstSource || tr.sourceRemaining != remaining {
		t.Fatal("owning cleanup changed source history")
	}
	serviceMust(t, m.CleanupCRATrace())
}
func polynomialTraceEnabledForTest() bool {
	tags := diagnostic.NewTraceTagsWithWriter(&strings.Builder{})
	tags.Enable("CRA")
	return tags.ShouldTrace("CRA")
}

func TestUPolynomialCRATracePublicFactorSourceStop(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace compiled out")
	}
	q, s := preparationPhysicalFixture(t)
	m := s.manager
	z := q.Integers()
	fs, e := m.NewFactors()
	serviceMust(t, e)
	var input upolyVectorState
	var square, linear unumeral.Int
	serviceMust(t, z.SetUint64(&square, 9000000006000000001))
	serviceMust(t, z.SetUint64(&linear, 6000000002))
	serviceMust(t, m.set(&input, []unumeral.Int{square, linear, unumeral.NewInt(1)}))
	allocator := z.SourceAllocator()
	query, e := allocator.Allocate(4096)
	serviceMust(t, e)
	raw, offset, e := allocator.RawLocation(query)
	serviceMust(t, e)
	w := craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset, scope: &s.frontend.cra.trace, frontend: true}
	defer func() {
		w.restore()
		serviceMust(t, m.modular.BindTrace(nil))
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, fs.Close())
		serviceMust(t, m.reset(&input))
		m.freeBacking(&input)
		serviceMust(t, z.Del(&linear))
		serviceMust(t, z.Del(&square))
		serviceMust(t, allocator.Deallocate(4096, query))
	}()
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("CRA")
	serviceMust(t, m.modular.BindTrace(tags))
	_, body := s.Factor(m.stateView(&input), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, m.modular.BindTrace(nil))
	f := &s.frontend
	c := &f.cra
	if w.ref == 0 || !errors.Is(body, umemory.ErrRuntimeUninitialized) || !f.active || c.constructed != 4 || c.sourceCause == nil || f.modularGCD.constructed != 6 || f.modularGCD.sourceCause == nil || m.pendingCRATrace != &c.trace {
		t.Fatal("public CRA stopped source frontiers", body, c.constructed, f.modularGCD.constructed)
	}
	before, source := c.constructed, c.sourceCause
	serviceMust(t, m.CleanupCRATrace()) // Default cleanup cannot adopt frontend receipt.
	if c.constructed != before || !c.trace.pending() {
		t.Fatal("default cleanup adopted caller-owned insertion")
	}
	w.restore()
	firstBody, firstSource := f.bodyCause, f.sourceCause
	serviceMust(t, s.CleanupRetained())
	if c.trace.pending() || m.pendingCRATrace != nil || f.bodyCause != firstBody || f.sourceCause != firstSource || c.sourceCause != source {
		t.Fatal("owning caller cleanup changed source history")
	}
}

type craTraceBorrowProbe struct {
	t       *testing.T
	m       *UPolynomialManager
	text    strings.Builder
	records int
}

func (w *craTraceBorrowProbe) Write(p []byte) (int, error) {
	want := uint8(0)
	switch string(p) {
	case "inv1: ":
		want = 3
	case "a1: ":
		want = 5
	case "lower: ":
		want = 12
	}
	if want != 0 {
		w.records++
		if w.m.craTrace.declared != want {
			w.t.Fatal("record crossed scalar declaration frontier", want, w.m.craTrace.declared)
		}
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupCRATrace()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("trace callback acquired active owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialCRATraceSourceFrontiersAndBorrow(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a, b := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{2})
	ao, e := m.descriptor(a)
	serviceMust(t, e)
	bo, e := m.descriptor(b)
	serviceMust(t, e)
	first, second := unumeral.NewInt(3), unumeral.NewInt(5)
	w := craTraceBorrowProbe{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("CRA")
	serviceMust(t, m.modular.BindTrace(tags))
	defer m.modular.BindTrace(nil)
	serviceMust(t, m.combineImages(&ao.state, &first, &bo.state, &second))
	if tags.ShouldTrace("CRA") && w.records != 3 {
		t.Fatal("missing actual source records", w.records)
	}
	upolyGCDExpect(t, m, b, []int64{7})
}
