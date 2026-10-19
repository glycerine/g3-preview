// Raw views prove source field reads without claiming constructor ownership.
package polynomial

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func algebraicIntervalRawFields(t *testing.T, c *AlgebraicRootCore, parts int) Algebraic {
	t.Helper()
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for i, field := range []struct {
		offset uint64
		words  []uint32
	}{{24, []uint32{7, 0, 0}}, {36, []uint32{3}}, {40, []uint32{9, 0, 0}}, {52, []uint32{1}}} {
		if i >= parts {
			break
		}
		for j, word := range field.words {
			serviceMust(t, c.config.Allocator.WriteUint32(l.raw, field.offset+4*uint64(j), word))
		}
	}
	return a
}

func TestAlgebraicDisplayIntervalReadsOnlyReachedBoundFields(t *testing.T) {
	for parts := 0; parts <= 4; parts++ {
		t.Run(fmt.Sprint(parts), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicIntervalRawFields(t, c, parts)
			var out bytes.Buffer
			e := c.DisplayInterval(&out, &a)
			want := []string{"(", "(", "(7/2^3, ", "(7/2^3, ", "(7/2^3, 9/2)"}[parts]
			if parts < 4 && !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatalf("missing bound guard: %v", e)
			}
			if parts == 4 {
				serviceMust(t, e)
			}
			if out.String() != want {
				t.Fatalf("reached insertion prefix: got %q want %q", out.String(), want)
			}
			serviceMust(t, c.CleanupRetained())
		})
	}
}

func TestAlgebraicDisplayIntervalReadsBoundsAfterOpeningInsertion(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicIntervalRawFields(t, c, 4)
	l, e := c.location(&a)
	serviceMust(t, e)
	out := &algebraicDisplayCallbackWriter{callback: func() {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 24, 13))
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 40, 19))
	}}
	serviceMust(t, c.DisplayInterval(out, &a))
	if out.String() != "(13/2^3, 19/2)" {
		t.Fatalf("bounds fetched before opening insertion: %q", out.String())
	}
}
