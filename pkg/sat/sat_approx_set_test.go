// Source: src/util/approx_set.h:43-158; src/sat/sat_types.h:115-119.
package sat

import "testing"

func TestSATApproximation(t *testing.T) {
	var a, b satApproxSet
	a.insert(1)
	a.insert(33)
	b.insert(1)
	if !a.mayContain(65) || a.mustNotContain(1) || !a.mustNotContain(2) || a != b {
		t.Fatal("source low-bit collision")
	}
	b.insert(2)
	if a.mustNotSubset(b) || !b.mustNotSubset(a) || !a.approxSubset(b) {
		t.Fatal("subset masks")
	}
	c := a
	c.union(b)
	if c != b {
		t.Fatal("union")
	}
	c.subtract(a)
	if c.mayContain(1) || !c.mayContain(2) {
		t.Fatal("difference")
	}
	c.intersect(a)
	if !c.empty() || !a.emptyIntersection(c) {
		t.Fatal("intersection")
	}
	a.reset()
	if !a.empty() {
		t.Fatal("reset")
	}
}
