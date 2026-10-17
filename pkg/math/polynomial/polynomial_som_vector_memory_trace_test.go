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

func TestPolynomialSOMVectorPublishedChildMemoryWrappers(t *testing.T) {
	m := polynomialCellFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m.monomials.config.Trace = tags
	tags.Enable("memory")
	v, err := NewPolynomialSOMVector(m)
	serviceMust(t, err)
	serviceMust(t, v.SetOwner(m))
	_, err = v.At(2)
	serviceMust(t, err)
	children := append([]*PolynomialSOMBuffer(nil), v.children...)
	_, err = v.At(1)
	serviceMust(t, err)
	serviceMust(t, v.Reset())
	if len(v.children) != 3 {
		t.Fatal("reset discarded the reusable source child owners")
	}
	serviceMust(t, v.CloseChecked())
	got := output.String()
	if !diagnostic.DebugBuild {
		if got != "" {
			t.Fatal("ordinary build emitted debug memory records", got)
		}
		return
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(PolynomialSOMBuffer{}).Size()), 10)
	ordered := make([]string, 0, 6)
	for _, child := range children {
		ordered = append(ordered, "alloc "+fmt.Sprintf("%p", child)+" src/math/polynomial/polynomial.cpp:2166 som_buffer "+size+"\n")
	}
	for _, child := range children {
		ordered = append(ordered, "dealloc "+fmt.Sprintf("%p", child)+" src/math/polynomial/polynomial.cpp:2182\n")
	}
	previous := -1
	for _, part := range ordered {
		at := strings.Index(got, part)
		if at <= previous || strings.Count(got, part) != 1 {
			t.Fatalf("source child memory wrapper order for %q in %q", part, got)
		}
		previous = at
	}
	if strings.Count(got, "[memory] allocate ") != 3 || strings.Count(got, "[memory] deallocate ") != 3 {
		t.Fatal("unexpected published child memory wrapper count", got)
	}
}
