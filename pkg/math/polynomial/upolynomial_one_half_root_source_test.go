package polynomial

// Source: src/math/polynomial/upolynomial.cpp:1298-1321.
// Qualified actual returning-transport/signed-host TLS destructor fixture.

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialHasOneHalfRootActualBodyAndNumeralDT(t *testing.T) {
	reached := false
	for offset := int64(12); offset < 500 && !reached; offset += 4 {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			var rt *umemory.Runtime
			var lane, held umemory.Block
			armed := false
			var transportBlocks uint32
			transport := func(f umemory.Failure) error {
				if !armed {
					return f
				}
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSNearMinimum(t, rt)
				var e error
				lane, e = rt.Allocate(8)
				serviceMust(t, e)
				delta := int64(-1000000)
				if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
					t.Fatal(e)
				}
				transportBlocks = rt.Stats().LiveBlocks
				return f
			}
			c, p, runtime := algebraicNegTLSFixture(t, transport)
			rt = runtime
			z := p.config.Integer
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			var big unumeral.Int
			serviceMust(t, z.SetLimbs(&big, []uint32{1, 2, 3, 4}))
			serviceMust(t, p.SetInts(poly, []unumeral.Int{big, big, unumeral.NewInt(1)}))
			serviceMust(t, z.Del(&big))
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			defer func() {
				armed = false
				serviceMust(t, rt.SetMaxSize(0))
				contextTLSFinish(t, rt, func() error { return p.cleanupOneHalfRoot(false) })
				for _, b := range []*umemory.Block{&lane, &held} {
					contextTLSFinish(t, rt, func() error {
						if *b == (umemory.Block{}) {
							return nil
						}
						return rt.Deallocate(b)
					})
				}
				contextTLSFinish(t, rt, func() error {
					if poly == (UPolynomial{}) {
						return nil
					}
					return p.DelPolynomial(&poly)
				})
				contextTLSFinish(t, rt, func() error { return p.cleanupOneHalfRoot(true) })
			}()
			held, e = rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - offset))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			armed = true
			_, body := p.HasOneHalfRoot(v)
			armed = false
			if lane == (umemory.Block{}) || rt.Stats().LiveBlocks >= transportBlocks {
				return
			}
			reached = true
			if !errors.Is(body, arena.ErrBounds) {
				t.Fatal("public scaled Horner discarded actual ak/r NUM destructor cause", body)
			}
			q := &p.oneHalfRoot
			if q.constructed != 2 || q.sourceRemaining != 2 || q.sourceCause == nil || q.bodyCause == nil {
				t.Fatal("actual youngest ak physical DT frontier", q.constructed, q.sourceRemaining, q.sourceCause)
			}
			if (q.values[1]).Cell == 0 || !z.IsLiveCellToken(uint64((q.values[1]).Cell)) || (!z.Cell((q.values[1]).Cell).Released && !z.Cell((q.values[1]).Cell).CleanupPending) {
				t.Fatal("ak descriptor does not name consumed physical source release")
			}
			algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&q.values[0], &q.values[1]}, []umemory.Block{lane, held})
			before := *q
			used, physical := z.Budget().Used(), rt.Stats()
			if e := p.SourceFinalize(); e != q.sourceCause {
				t.Fatal("older UPM crossed sticky ak source stop", e)
			}
			if e := q.finish(nil); !errors.Is(e, q.sourceCause) || q.sourceRemaining != before.sourceRemaining || z.Budget().Used() != used || rt.Stats() != physical {
				t.Fatal("source retry replayed stopped cursor", e)
			}
			if e := p.DelPolynomial(&poly); e != ErrUPolynomialCleanupPending {
				t.Fatal("input coefficient DT crossed younger ak", e)
			}
			r := p.oneHalfRootReceipt()
			old := *r.history
			firstOwn := p.cleanupOneHalfRoot(false)
			if !errors.Is(firstOwn, arena.ErrBounds) || q.ownCause != firstOwn {
				t.Fatal("first matched own cause lost", firstOwn)
			}
			contextTLSFinish(t, rt, func() error { return p.cleanupOneHalfRoot(false) })
			contextTLSFinish(t, rt, func() error {
				if poly == (UPolynomial{}) {
					return nil
				}
				return p.DelPolynomial(&poly)
			})
			contextTLSFinish(t, rt, func() error { return p.cleanupOneHalfRoot(true) })
			if r.history.bodyCause != old.bodyCause || r.history.sourceCause != old.sourceCause || !errors.Is(body, old.bodyCause) || !errors.Is(body, old.sourceCause) {
				t.Fatal("own ACK erased immutable original source/body")
			}
			if r.history.ownCause != firstOwn {
				t.Fatal("first own cause changed through retry")
			}
			finalHistory := *r.history
			linear := algebraicPolynomial(t, p, -1, 2)
			lv, e := p.wholeView(linear)
			serviceMust(t, e)
			z.Budget().FailAfter(0)
			yes, e := p.HasOneHalfRoot(lv)
			z.Budget().FailAfter(-1)
			if e != nil || !yes {
				t.Fatal("actual inline reuse after own cleanup", yes, e)
			}
			later := p.oneHalfRoot
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, false))
			serviceMust(t, p.cleanupOneHalfRootWithReceipt(&r, true))
			if *r.history != finalHistory || !reflect.DeepEqual(later, p.oneHalfRoot) || !errors.Is(body, finalHistory.sourceCause) || !errors.Is(body, finalHistory.bodyCause) {
				t.Fatal("old stopped body/source/own tuple changed on new actual invocation")
			}

		})
	}
	if !reached {
		t.Fatal("actual body failure followed by physical NUM release absent")
	}
}

func TestUPolynomialHasOneHalfRootActualModularOwnerCallback(t *testing.T) {
	var p *UPolynomialManager
	var foreign *unumeral.IntegerModularManager
	var current UPolynomial
	armed, reached := false, false
	transport := func(f umemory.Failure) error {
		if !armed {
			return f
		}
		armed = false
		reached = true
		closed, used, physical := p.modular.Closed(), p.config.Integer.Budget().Used(), p.config.Integer.RuntimeOwner().Stats()
		q := &p.oneHalfRoot
		before := *q
		r := p.oneHalfRootReceipt()
		for _, abort := range []bool{false, true} {
			if e := p.cleanupOneHalfRootWithReceipt(&r, abort); e != ErrUPolynomialCleanupPending {
				t.Error("executing real own child", e)
			}
		}
		if e := q.finish(nil); e != ErrUPolynomialCleanupPending {
			t.Error("executing source child", e)
		}
		if e := p.DelPolynomial(&current); e != ErrUPolynomialCleanupPending {
			t.Error("executing borrowed input DT", e)
		}
		bad := q.context()
		copied := *q
		bad.oneHalfRoot = &copied
		if (bad.roots(p.config.Integer)) != nil {
			t.Error("copied actual frame acquired roots")
		}
		bad = q.context()
		bad.oneHalfRootInvocation++
		if (bad.roots(p.config.Integer)) != nil {
			t.Error("stale context acquired roots")
		}
		if e := numericIntegerModularManager_copyWithOneHalfRootContext(p.modular, &q.values[0], &q.values[1], bad); e != ErrUPolynomialOwner {
			t.Error("stale typed SET changed actual NUM", e)
		}
		if e := numericIntegerModularManager_copyWithOneHalfRootContext(foreign, &q.values[0], &q.values[1], q.context()); e != ErrUPolynomialOwner {
			t.Error("foreign actual modular receiver acquired roots", e)
		}
		for _, end := range []func() error{p.SourceFinalize, p.Close, p.config.Integer.SourceFinalize, p.config.Integer.CloseChecked, p.config.Integer.Naturals().SourceFinalize, p.config.Integer.Naturals().CloseChecked} {
			if e := end(); e == nil {
				t.Error("executing related owner ended", e)
			}
		}
		if !reflect.DeepEqual(before, *q) || used != p.config.Integer.Budget().Used() || physical != p.config.Integer.RuntimeOwner().Stats() {
			t.Error("refused callback changed actual NUM/child/history/graph")
		}

		if e := p.modular.Close(); e != ErrPolynomialPending || closed != p.modular.Closed() || used != p.config.Integer.Budget().Used() || physical != p.config.Integer.RuntimeOwner().Stats() {
			t.Error("actual scaled Horner modular owner retired before return", e)
		}
		return f
	}
	c, actual, rt := algebraicNegTLSFixture(t, transport)
	p = actual
	z := p.config.Integer
	foreign, e := unumeral.NewIntegerModularManager(z)
	serviceMust(t, e)
	defer func() { serviceMust(t, foreign.Close()) }()
	poly := algebraicPolynomial(t, p, 1, 2147483647, 1)
	current = poly
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	held, e := rt.Allocate(uint64(int64(100000) - rt.State().LocalSize - 12))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	armed = true
	_, body := p.HasOneHalfRoot(v)
	armed = false
	if !reached || body == nil {
		t.Fatal("actual public allocating leaf absent", reached, body)
	}
	algebraicCellSourcePhysical(t, c, []UPolynomial{poly}, []*unumeral.Int{&p.oneHalfRoot.values[0], &p.oneHalfRoot.values[1]}, []umemory.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, p.cleanupOneHalfRoot(false))
	serviceMust(t, p.cleanupOneHalfRoot(true))
	serviceMust(t, rt.Deallocate(&held))
	_ = z
}
