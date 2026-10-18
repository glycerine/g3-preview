package polynomial

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Exercise copy_poly with independently owned big numeral cells. A small-only
// fixture would not distinguish a foreign owner from the receiving manager.
func TestAlgebraicCopyPolynomialFromOwner(t *testing.T) {
	source, sp := algebraicFixture(t)
	target, tp := algebraicFixture(t)
	want := []string{"-1267650600228229401496703205377", "7", "1361129467683753853853498429727072845825"}
	values := make([]unumeral.Int, len(want))
	for i := range values {
		serviceMust(t, source.config.Rationals.Integers().SetString(&values[i], want[i]))
	}
	poly, e := sp.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, sp.SetInts(poly, values))
	for i := range values {
		serviceMust(t, source.config.Rationals.Integers().Del(&values[i]))
	}
	v, e := sp.wholeView(poly)
	serviceMust(t, e)
	a, e := target.allocate(1)
	serviceMust(t, e)
	var dst Algebraic
	target.publish(&dst, a)
	t.Cleanup(func() { serviceMust(t, target.Del(&dst)) })
	l, e := target.location(&dst)
	serviceMust(t, e)
	r, e := target.load(l)
	serviceMust(t, e)
	serviceMust(t, target.copyPolynomialFromOwner(l, &r, sp, v))
	if r.size != uint32(len(want)) || l.coefficientInitialized != r.size {
		t.Fatalf("copy prefix size=%d initialized=%d", r.size, l.coefficientInitialized)
	}
	serviceMust(t, sp.DelPolynomial(&poly))
	out, e := tp.wholeView(r.poly)
	serviceMust(t, e)
	for i, text := range want {
		n, e := tp.read(out, uint32(i))
		serviceMust(t, e)
		got, e := target.config.Rationals.Integers().String(n)
		serviceMust(t, e)
		if got != text {
			t.Fatalf("coefficient %d: got %s want %s", i, got, text)
		}
	}
}

func TestAlgebraicCopyUsesRecordedPolynomialSize(t *testing.T) {
	c, p := algebraicFixture(t)
	src := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	var dst Algebraic
	t.Cleanup(func() {
		serviceMust(t, c.Del(&dst))
		serviceMust(t, c.Del(&src))
	})
	l, e := c.location(&src)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	// The algebraic cell captures its source extent independently of the Go
	// descriptor's backing extent. Extra backing must not extend copy_poly.
	serviceMust(t, p.SetInts(r.poly, []unumeral.Int{
		unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1), unumeral.NewInt(99),
	}))
	serviceMust(t, c.Copy(&dst, &src))
	dl, e := c.location(&dst)
	serviceMust(t, e)
	out, e := c.load(dl)
	serviceMust(t, e)
	v, e := p.wholeView(out.poly)
	serviceMust(t, e)
	if out.size != r.size || v.count != r.size || dl.coefficientInitialized != r.size {
		t.Fatalf("recorded=%d copied=%d backing=%d initialized=%d", r.size, out.size, v.count, dl.coefficientInitialized)
	}
	if out.flags != r.flags {
		t.Fatalf("flags changed: %x -> %x", r.flags, out.flags)
	}
}
