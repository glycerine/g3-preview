package polynomial

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// These are direct diagnostic operands, independently supplied to a real
// admitted frontend. No valid GCD result or constructor frontier is fabricated.
func TestUPolynomialFrontendNormalizeTraceOperands(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			f, e := s.beginFactorFrontend()
			serviceMust(t, e)
			defer func() { f.bodyReturned(); serviceMust(t, f.finish(nil)); serviceMust(t, s.CleanupRetained()) }()
			var text strings.Builder
			failed := &linearTraceFailWriter{}
			var out io.Writer = &text
			if refused {
				out = failed
			}
			tags := diagnostic.NewTraceTagsWithWriter(out)
			tags.Enable("div_bug")
			serviceMust(t, m.modular.BindTrace(tags))
			defer func() { serviceMust(t, m.modular.BindTrace(nil)) }()
			old, g, x, tmp := unumeral.NewInt(3), unumeral.NewInt(2), unumeral.NewInt(1), unumeral.NewInt(2)
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			serviceMust(t, m.traceFrontendNormalization(f, &old, &g, &x, &tmp))
			if polynomialSourceDebug && tags.ShouldTrace("div_bug") {
				if refused {
					if failed.calls != 1 {
						t.Fatal("nonsticky failed stream", failed.calls)
					}
				} else if !strings.Contains(text.String(), "old(p[i]): 3, g: 2, p[i]: 1, tmp: 2\n") {
					t.Fatal(text.String())
				}
			} else if text.Len() != 0 || failed.calls != 0 {
				t.Fatal("erased DEBUG trace")
			}
			if f.scalarTracePending() {
				t.Fatal("completed SMALL format retained child")
			}
		})
	}
}

func TestUPolynomialFrontendNormalizeTraceErasureAndCondition(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	tags := diagnostic.NewTraceTagsWithWriter(nil)
	serviceMust(t, m.modular.BindTrace(tags))
	defer func() { serviceMust(t, m.modular.BindTrace(nil)) }()
	// Disabled CTRACE cannot inspect missing caller or operands.
	serviceMust(t, m.traceFrontendNormalization(nil, nil, nil, nil, nil))
	tags.Enable("div_bug")
	f, e := s.beginFactorFrontend()
	serviceMust(t, e)
	defer func() { f.bodyReturned(); serviceMust(t, f.finish(nil)); serviceMust(t, s.CleanupRetained()) }()
	one := unumeral.NewInt(1)
	// A false condition must not touch the unneeded divisor/coefficient.
	serviceMust(t, m.traceFrontendNormalization(f, &one, nil, nil, &one))
}

type frontendNormalizationReadGuard struct {
	craTraceReadGuard
	service *UPolynomialFactorizationService
	value   *unumeral.Int
	caption string
	refused bool
}

func (w *frontendNormalizationReadGuard) Write(p []byte) (int, error) {
	if string(p) == w.caption && w.ref == 0 {
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.service.Close(), w.service.CleanupRetained()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("trace callback acquired executing owner", e)
			}
		}
		z := w.m.config.Integer
		w.ref = w.value.Cell
		w.original = z.Cell(w.ref)
		size, e := w.original.Backing.ReadWord(z.Allocator, 0)
		serviceMust(w.t, e)
		serviceMust(w.t, z.RuntimeOwner().WriteUint32(w.raw, w.offset, size))
		projected := w.original
		projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = w.query, w.raw, w.offset
		z.Store(w.ref, projected)
		if w.refused {
			return 0, errors.New("outer normalization trace refused")
		}
	}
	return len(p), nil
}

func TestUPolynomialFrontendNormalizeTraceExactChildren(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("DEBUG trace erased")
	}
	for index, caption := range []string{"old(p[i]): ", ", g: ", ", p[i]: ", ", tmp: "} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(index, refused), func(t *testing.T) {
				q, s := preparationPhysicalFixture(t)
				m, z := s.manager, q.Integers()
				values := [4]unumeral.Int{unumeral.NewInt(3), unumeral.NewInt(2), unumeral.NewInt(1), unumeral.NewInt(2)}
				// Each selected independently supplied diagnostic operand is genuinely BIG.
				serviceMust(t, z.SetString(&values[index], "100000000000000000000000000000000000000000000000000000003"))
				t.Cleanup(func() {
					for i := range values {
						serviceMust(t, z.Del(&values[i]))
					}
				})
				f, e := s.beginFactorFrontend()
				serviceMust(t, e)
				allocator := z.SourceAllocator()
				query, e := allocator.Allocate(4096)
				serviceMust(t, e)
				raw, offset, e := allocator.RawLocation(query)
				serviceMust(t, e)
				w := frontendNormalizationReadGuard{craTraceReadGuard: craTraceReadGuard{t: t, m: m, query: query, raw: raw, offset: offset}, value: &values[index], caption: caption, refused: refused, service: s}
				t.Cleanup(func() {
					w.restore()
					serviceMust(t, s.CleanupRetained())
					serviceMust(t, allocator.Deallocate(4096, query))
				})
				tags := diagnostic.NewTraceTagsWithWriter(&w)
				tags.Enable("div_bug")
				serviceMust(t, m.modular.BindTrace(tags))
				t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
				body := m.traceFrontendNormalization(f, &values[0], &values[1], &values[2], &values[3])
				f.bodyReturned()
				body = f.finish(body)
				if !errors.Is(body, umemory.ErrRuntimeUninitialized) || w.ref == 0 || !f.active || !f.scalarTrace[index].Active() || m.pendingFactorScalarTrace != f {
					t.Fatal("actual character read/receipt", body, w.ref, f.active)
				}
				if f.sourceCause != nil || f.constructed != 0 {
					t.Fatal("unjoined body invented older source stop/frontier", f.sourceCause, f.constructed)
				}
				if f.history.scalarTrace[index].Character != f.scalarTrace[index].Character {
					t.Fatal("missing exact history")
				}
				if m.SourceFinalize() == nil {
					t.Fatal("source finalize crossed retained character child")
				}
				w.restore()
				// Accept a later independent formatting invocation before old cleanup;
				// cleanup owns only its old receipt, even when the numerical frame is reused.
				original := z.Cell(values[index].Cell)
				size, e := original.Backing.ReadWord(z.Allocator, 0)
				serviceMust(t, e)
				serviceMust(t, z.RuntimeOwner().WriteUint32(raw, offset, size))
				projected := original
				projected.Backing.Object, projected.Backing.Raw, projected.Backing.Offset = query, raw, offset
				z.Store(values[index].Cell, projected)
				var later unumeral.NumeralInsertionReceipt
				_, e = m.modular.SourceStringTracked(&values[index], &later)
				z.Store(values[index].Cell, original)
				t.Cleanup(func() { serviceMust(t, m.modular.CleanupSourceString(&later)) })
				if !errors.Is(e, umemory.ErrRuntimeUninitialized) || !later.Active() {
					t.Fatal("independent accepted child", e)
				}
				character := later.Character
				originalBody, originalSource := f.bodyCause, f.sourceCause
				serviceMust(t, s.CleanupRetained())
				if f.active || f.scalarTracePending() || m.pendingFactorScalarTrace != nil || f.bodyCause != originalBody || f.sourceCause != originalSource {
					t.Fatal("owning cleanup changed history")
				}
				if !later.Active() || later.Character != character {
					t.Fatal("old cleanup adopted independent child")
				}
				serviceMust(t, m.modular.CleanupSourceString(&later))
			})
		}
	}
}

func TestUPolynomialFrontendNormalizeTraceValidFactorization(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			_, s := preparationPhysicalFixture(t)
			m := s.manager
			p := upolySourceNew(t, m, []int32{-1, 0, 1})
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
			var text strings.Builder
			tags := diagnostic.NewTraceTagsWithWriter(&text)
			tags.Enable("div_bug")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			polynomialAssertionActionsFixture(t, "")
			if polynomialSourceDebug && disabled {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			ok, e := s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			serviceMust(t, e)
			if !ok || fs.count != 2 || s.frontend.vectorNormalize.invocation != 2 {
				t.Fatal("actual factor normalization", ok, fs.count)
			}
			if polynomialSourceDebug && s.frontend.vectorNormalize.peak != 3 {
				t.Fatal("DEBUG source product omitted")
			}
			if text.Len() != 0 || s.frontend.scalarTracePending() {
				t.Fatal("true division identity emitted conditional diagnostic")
			}
			// Independent exact coefficient oracle for (x-1)*(x+1).
			for i, want := range [2][2]int32{{-1, 1}, {1, 1}} {
				v, e := fs.FactorView(uint32(i))
				serviceMust(t, e)
				if v.count != 2 {
					t.Fatal("factor degree", v.count)
				}
				for j, n := range want {
					a, e := m.read(v, uint32(j))
					serviceMust(t, e)
					expected := unumeral.NewInt(n)
					equal, e := m.config.Integer.Equal(a, &expected)
					serviceMust(t, e)
					if !equal {
						t.Fatal("independent factor coefficient", i, j)
					}
				}
			}
		})
	}
}
