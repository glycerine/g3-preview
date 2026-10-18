package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type euclidPointerProbe struct {
	strings.Builder
	t       *testing.T
	m       *UPolynomialManager
	a, b    UPolynomialView
	ap, bp  *unumeral.Int
	checked bool
	want    string
}

func (w *euclidPointerProbe) Write(p []byte) (int, error) {
	if !w.checked {
		w.checked = true
		values := w.m.values(&w.m.gcd2)
		w.want = fmt.Sprintf("-------- [upolynomial] euclid_gcd src/math/polynomial/upolynomial.cpp:833 ---------\nsz1: %d, p1: %p, sz2: %d, p2: %p\nB.size(): %d, B.c_ptr(): %p\n------------------------------------------------\n", w.a.count, w.ap, w.b.count, w.bp, w.m.gcd2.size, &values[0])
		if w.m.Close() == nil || w.m.SourceFinalize() == nil {
			w.t.Fatal("callback retired manager")
		}
	}
	return w.Builder.Write(p)
}
func TestUPolynomialEuclidPointerTrace(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{2, 3, 1})
	b := upolySourceNew(t, m, []int32{3, 4, 1})
	g := upolySourceNew(t, m, []int32{99})
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	out, e := m.descriptor(g)
	serviceMust(t, e)
	ap, e := m.read(av, 0)
	serviceMust(t, e)
	bp, e := m.read(bv, 0)
	serviceMust(t, e)
	probe := euclidPointerProbe{t: t, m: m, a: av, b: bv, ap: ap, bp: bp}
	tags := diagnostic.NewTraceTagsWithWriter(&probe)
	tags.Enable("upolynomial")
	serviceMust(t, m.modular.BindTrace(tags))
	defer m.modular.BindTrace(nil)
	serviceMust(t, m.euclidGCD(av, bv, &out.state))
	upolyGCDExpect(t, m, g, []int64{1, 1})
	if tags.ShouldTrace("upolynomial") {
		if !probe.checked || !strings.HasPrefix(probe.String(), probe.want) {
			t.Fatalf("source pointers: %s; want %s", probe.String(), probe.want)
		}
	} else if probe.checked {
		t.Fatal("erased trace evaluated")
	}
	// Empty input takes the earlier source return and never enters this record.
	before := probe.String()
	serviceMust(t, m.euclidGCD(av.WithCount(0), bv, &out.state))
	if probe.String() != before {
		t.Fatal("empty early return emitted record")
	}
}
