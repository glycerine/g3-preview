package polynomial

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialGCDSkeletonFailedConstructorAllocationRecord(t *testing.T) {
	m := polynomialCellDirectObjectFixture(t)
	p, _ := polynomialSkeletonInput(t, m)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m.monomials.config.Trace = tags
	m.integer.Budget().FailAfter(1) // header succeeds; first constructor reservation fails.
	s, err := newGCDPolynomialSkeleton(m, p, 0)
	if s == nil || err == nil || s.completed {
		t.Fatalf("expected failed constructor after allocation: %p %v", s, err)
	}
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, s.CloseChecked())
	serviceMust(t, m.DecRef(p))
	got := output.String()
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		if got != "" {
			t.Fatal("disabled memory tracing emitted records", got)
		}
		return
	}
	alloc := "alloc " + fmt.Sprintf("%p", s) + " src/math/polynomial/polynomial.cpp:4226 skeleton "
	if strings.Count(got, alloc) != 1 || strings.Contains(got, "dealloc "+fmt.Sprintf("%p", s)) {
		t.Fatalf("failed placement construction memory record: %q", got)
	}
}
