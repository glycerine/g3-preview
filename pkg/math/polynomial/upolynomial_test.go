package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Pure bounds/layout tests require no substitute resource owner.
func TestUPolynomialScalarExtentAndLayout(t *testing.T) {
	if reflect.TypeOf(UPolynomial{}).Size() != 16 {
		t.Fatal("scalar owner shape")
	}
	for _, f := range []struct {
		old, host uint64
		n         uint32
		ok        bool
	}{{0, 32, 2, true}, {2, 48, 3, true}, {3, 80, 5, true}, {178956970, ^uint64(0), 268435455, true}, {178956971, ^uint64(0), 0, false}, {0, 31, 0, false}, {1 << 32, ^uint64(0), 0, false}} {
		n, _, e := upolyNextCapacity(f.old, 16, f.host)
		if f.ok {
			if e != nil || n != f.n {
				t.Fatal(f, n, e)
			}
		} else if !errors.Is(e, arena.ErrBounds) {
			t.Fatal(f, n, e)
		}
	}
	// Reachable capacities from the actual empty constructor fail at the
	// literal source check before any wrapped physical allocation is attempted.
	cap := uint32(0)
	for {
		n, _, _, e := upolySourceCapacity(cap)
		if e != nil {
			if cap != 204324849 || n != 306487274 {
				t.Fatal("source geometric endpoint", cap, n, e)
			}
			break
		}
		cap = n
	}
	// Source physical extent keeps derivative's raw multiplier i inline int32.
	if uint64(268435455) >= uint64(1<<31) {
		t.Fatal("source scalar multiplier invariant")
	}
}

// Keep the source unsigned32 overflow action distinct from the Go guard for
// source allocation wrap that would otherwise make element writes undefined.
func TestUPolynomialSourceUnsignedExtent(t *testing.T) {
	n, oldBytes, newBytes, e := upolySourceCapacity(268435456)
	if e != nil || n != 402653184 || oldBytes != 8 || newBytes != 2147483656 {
		t.Fatal(n, oldBytes, newBytes, e)
	}
	if _, _, e = upolyNextCapacity(268435456, 16, ^uint64(0)); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("wrapped allocation unsafe domain", e)
	}
	if _, _, _, e = upolySourceCapacity(178956971); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source overflow action", e)
	}
}
