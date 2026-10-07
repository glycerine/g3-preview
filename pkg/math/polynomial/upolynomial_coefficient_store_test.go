// Licensed under the MIT license.
package polynomial

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialCoefficientStoreCounterAndColdLayout(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	z, e := util.NewIntegerManager(util.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	before := b.Used()
	s, e := newUPolynomialCoefficientStore(z, 17)
	if e != nil {
		t.Fatal(e)
	}
	st, e := s.Stats()
	if e != nil || st.Owners != 1 || st.TotalBytes != uint64(reflect.TypeOf(*s).Size()) || b.Used() != before+st.TotalBytes {
		t.Fatal(st, e, b.Used(), before)
	}
	s.owners = math.MaxUint32
	if e = s.retain(); !errors.Is(e, arena.ErrSlotLimit) || s.owners != math.MaxUint32 || b.Used() != before+st.TotalBytes {
		t.Fatal(e, s.owners)
	}
	s.owners = 1
	s.release()
	if b.Used() != before {
		t.Fatal(b.Used(), before)
	}
	if _, e = s.Stats(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	var zero UPolynomialCoefficientStore
	if _, e = zero.Stats(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
}
