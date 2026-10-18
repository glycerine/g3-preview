package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

type quadraticScalarReadGuard struct {
	craTraceReadGuard
	service *UPolynomialFactorizationService
	root    bool
	refused bool
}

func (w *quadraticScalarReadGuard) Write(p []byte) (int, error) {
	caption := "\nb: "
	if w.root {
		caption = "disc_sqrt: "
	}
	if string(p) == caption && w.ref == 0 {
		f := &w.service.frontend
		var n *unumeral.Int
		if w.root {
			if f.quadratic.constructed != 4 {
				w.t.Fatal("root trace declaration frontier", f.quadratic.constructed)
			}
			n = &f.quadratic.values[3]
		} else {
			if f.quadratic.constructed != 0 {
				w.t.Fatal("coefficient trace declared quadratic NUMs")
			}
			n = &w.m.values(&f.pp)[1]
		}
		if n.IsSmall() {
			w.t.Fatal("actual scalar trace operand not BIG")
		}
		z := w.m.config.Integer
		w.ref = n.Cell
		w.original = z.Cell(w.ref)
		size, e := w.original.Backing.ReadWord(z.Allocator, 0)
		serviceMust(w.t, e)
		serviceMust(w.t, z.RuntimeOwner().WriteUint32(w.raw, w.offset, size))
		cell := w.original
		cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = w.query, w.raw, w.offset
		z.Store(w.ref, cell)
		if w.refused {
			return 0, errors.New("outer scalar trace refused")
		}
	}
	return len(p), nil
}
func TestUPolynomialQuadraticScalarTraceActualReadStops(t *testing.T) {
	for _, root := range []bool{false, true} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(root, refused), func(t *testing.T) {
				if !polynomialTraceEnabledForTest() {
					t.Skip("trace erased")
				}
				q, s := preparationPhysicalFixture(t)
				m, z := s.manager, q.Integers()
				// (x+3000000001)*(x+1): BIG b and BIG discriminant root3000000000.
				var ns [3]unumeral.Int
				for i, v := range []string{"3000000001", "3000000002", "1"} {
					serviceMust(t, z.SetString(&ns[i], v))
				}
				p := upolySourceNew(t, m, nil)
				serviceMust(t, m.SetInts(p, ns[:]))
				for i := range ns {
					serviceMust(t, z.Del(&ns[i]))
				}
				fs, e := m.NewFactors()
				serviceMust(t, e)
				input, e := m.descriptor(p)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, fs.Close()) })
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := quadraticScalarReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, service: s, root: root, refused: refused}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, s.CleanupRetained())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("factor")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				_, body := s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
				f := &s.frontend
				index := 1
				want := uint8(0)
				if root {
					index = 0
					want = 4
				}
				if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !f.active || !f.scalarTrace[index].Active() || m.pendingFactorScalarTrace != f || f.quadratic.constructed != want || fs.count != 0 {
					t.Fatal("actual scalar formatting read stop", body, f.quadratic.constructed, want)
				}
				if f.history.scalarTrace[index].Character != f.scalarTrace[index].Character {
					t.Fatal("history lost exact scalar receipt")
				}
				if !errors.Is(f.sourceCause, umemory.ErrRuntimeUninitialized) {
					t.Fatal("source stop did not reach actual unreadable NUM destruction", f.sourceCause)
				}
				if !root && (f.constructed != 2 || f.sourceStoppedStage != 2 || f.sourceStoppedIndex != 1) {
					t.Fatal("body failure blocked older source destructors before actual b stop", f.constructed, f.sourceStoppedStage, f.sourceStoppedIndex)
				}
				source, originalBody := f.sourceCause, f.bodyCause
				if m.SourceFinalize() == nil {
					t.Fatal("source retirement crossed scalar child")
				}
				w.restore() // restore genuine backing before exact owning destruction

				// A later real to_string invocation can reuse the numerical formatting
				// frame. Old frontend cleanup has no authority over its accepted receipt.
				n := &m.values(&input.state)[1]
				original := z.Cell(n.Cell)
				size, e := original.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := original
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(n.Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				_, e = m.modular.SourceStringTracked(n, &later)
				z.Store(n.Cell, original)
				t.Cleanup(func() {
					if later.Active() {
						serviceMust(t, later.Complete(z.Naturals(), false))
						serviceMust(t, later.Complete(z.Naturals(), true))
					}
				})
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("later accepted scalar string", e)
				}
				character := later.Character
				serviceMust(t, s.CleanupRetained())
				if f.active || f.scalarTracePending() || m.pendingFactorScalarTrace != nil || f.sourceCause != source || f.bodyCause != originalBody {
					t.Fatal("scalar owning cleanup changed source history")
				}
				serviceMust(t, f.cleanupScalarTrace())
				if !later.Active() || later.Character != character {
					t.Fatal("stale scalar cleanup adopted later character invocation")
				}
				serviceMust(t, later.Complete(z.Naturals(), false))
				serviceMust(t, later.Complete(z.Naturals(), true))

			})
		}
	}
}
