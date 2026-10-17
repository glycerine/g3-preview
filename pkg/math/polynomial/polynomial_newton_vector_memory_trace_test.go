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

func TestPolynomialNewtonVectorChildMemoryWrappers(t *testing.T) {
	m := polynomialCellFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.monomials.config.Trace = tags
	tags.Enable("memory")
	v, err := NewPolynomialNewtonVector(m)
	serviceMust(t, err)
	serviceMust(t, v.SetOwner(m))
	_, err = v.At(2)
	serviceMust(t, err)
	first := append([]*PolynomialNewtonInterpolator(nil), v.children...)
	_, err = v.At(2)
	serviceMust(t, err)
	if len(v.children) != 3 {
		t.Fatal("cache hit grew the source child vector")
	}
	serviceMust(t, v.Flush())
	_, err = v.At(0)
	serviceMust(t, err)
	last := v.children[0]
	serviceMust(t, v.Close())
	got := output.String()
	if !diagnostic.DebugBuild {
		if got != "" {
			t.Fatal("ordinary build emitted debug memory records", got)
		}
		return
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(PolynomialNewtonInterpolator{}).Size()), 10)
	ordered := make([]string, 0, 8)
	for _, child := range first {
		ordered = append(ordered, "alloc "+fmt.Sprintf("%p", child)+" src/math/polynomial/polynomial.cpp:2948 newton_interpolator "+size+"\n")
	}
	for _, child := range first {
		ordered = append(ordered, "dealloc "+fmt.Sprintf("%p", child)+" src/math/polynomial/polynomial.cpp:2936\n")
	}
	ordered = append(ordered,
		"alloc "+fmt.Sprintf("%p", last)+" src/math/polynomial/polynomial.cpp:2948 newton_interpolator "+size+"\n",
		"dealloc "+fmt.Sprintf("%p", last)+" src/math/polynomial/polynomial.cpp:2936\n")
	previous := -1
	for _, part := range ordered {
		at := strings.Index(got, part)
		if at <= previous || strings.Count(got, part) != 1 {
			t.Fatalf("source child allocation/deallocation order for %q in %q", part, got)
		}
		previous = at
	}
	if strings.Count(got, "[memory] allocate ") != 4 || strings.Count(got, "[memory] deallocate ") != 4 {
		t.Fatal("unexpected child memory wrapper count", got)
	}
}
