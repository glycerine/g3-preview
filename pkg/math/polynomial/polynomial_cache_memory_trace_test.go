package polynomial

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialCacheImpMemoryWrapperLifecycleAndResetRefusal(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	defer tags.Finalize()
	m.core.monomials.config.Trace = tags
	tags.Enable("memory")
	c, err := NewPolynomialCache(m)
	serviceMust(t, err)
	first := c.imp
	if first == nil {
		t.Fatal("constructor omitted private cache owner")
	}
	budget := m.core.integer.Budget()
	budget.FailAfter(0)
	err = c.Reset()
	budget.FailAfter(-1)
	if err == nil || !c.resetting || c.imp != nil {
		t.Fatal("refused reset did not leave the old imp released for retry", err)
	}
	serviceMust(t, c.CleanupReset())
	second := c.imp
	if second == nil || second == first || c.resetting {
		t.Fatal("retry did not install a distinct private cache owner")
	}
	serviceMust(t, c.Reset())
	third := c.imp
	if third == nil || third == second || third == first {
		t.Fatal("completed reset did not replace its private cache owner")
	}
	serviceMust(t, c.Close())
	got := output.String()
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		if got != "" {
			t.Fatal("erased memory trace emitted records", got)
		}
		return
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(polynomialCacheImpOwner{}).Size()), 10)
	parts := []string{
		"alloc " + fmt.Sprintf("%p", first) + " src/math/polynomial/polynomial_cache.cpp:207 imp " + size + "\n",
		"dealloc " + fmt.Sprintf("%p", first) + " src/math/polynomial/polynomial_cache.cpp:232\n",
		"alloc " + fmt.Sprintf("%p", second) + " src/math/polynomial/polynomial_cache.cpp:233 imp " + size + "\n",
		"dealloc " + fmt.Sprintf("%p", second) + " src/math/polynomial/polynomial_cache.cpp:232\n",
		"alloc " + fmt.Sprintf("%p", third) + " src/math/polynomial/polynomial_cache.cpp:233 imp " + size + "\n",
		"dealloc " + fmt.Sprintf("%p", third) + " src/math/polynomial/polynomial_cache.cpp:211\n",
	}
	last := -1
	for _, part := range parts {
		at := strings.Index(got, part)
		if at <= last || strings.Count(got, part) != 1 {
			t.Fatalf("private cache source wrapper order for %q in %q", part, got)
		}
		last = at
	}
	if strings.Count(got, "[memory] allocate ") != 3 || strings.Count(got, "[memory] deallocate ") != 3 {
		t.Fatal("refused allocation unexpectedly emitted a wrapper", got)
	}
}
