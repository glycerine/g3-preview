// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Pure scalar tests avoid a substitute resource/cooperation owner. Runtime
// storage/lifetimes use actual g3 owners in upolynomial_sequence_source_test.go.
func TestUPolynomialSequenceUnsignedVectorExtent(t *testing.T) {
	for _, f := range []struct {
		old  uint32
		want uint32
		ok   bool
	}{{0, 2, true}, {2, 3, true}, {3, 5, true}, {5, 8, true}, {536870910, 805306365, true}, {536870911, 805306367, true}, {715827883, 1073741825, false}} {
		n, oldBytes, newBytes, e := upolySequenceSourceCapacity(f.old)
		if f.ok {
			if e != nil || n != f.want {
				t.Fatal(f, n, oldBytes, newBytes, e)
			}
		} else if !errors.Is(e, arena.ErrBounds) {
			t.Fatal(f, n, oldBytes, newBytes, e)
		}
	}
	// Literal uint32 wrapped extent can pass the source comparison even though
	// its allocation would underrepresent the elements; separate Go guard rejects.
	n, oldBytes, newBytes, e := upolySequenceSourceCapacity(1073741824)
	if e != nil || n != 1610612736 || oldBytes != 8 || newBytes != 2147483656 {
		t.Fatal(n, oldBytes, newBytes, e)
	}
	if _, _, e = upolySequenceNextCapacity(1073741824, ^uint64(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("unsafe wrapped physical allocation", e)
	}
	if _, _, e = upolySequenceNextCapacity(2, 11); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("physical host extent", e)
	}
}
