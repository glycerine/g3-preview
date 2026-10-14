package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Source hash expressions stop at a failed component read before combining
// fields. Restore each borrowed field before releasing the owning numeral.
func TestInfHashesStopBeforeCombiningFailedComponent(t *testing.T) {
	s, _ := serviceNew(t)
	invalid := ^NumericCellRef(0)

	plain := infValue(t, s, 5, 1, "3")
	firstCell := plain.First.Num.Cell
	plain.First.Num.Cell = invalid
	firstHash, firstErr := s.InfHash(&plain)
	plain.First.Num.Cell = firstCell
	if firstHash != 0 || !errors.Is(firstErr, arena.ErrInvalidSlot) {
		t.Fatal("inf_rational did not stop at failed first hash", firstHash, firstErr)
	}
	plainCell := plain.Second.Num.Cell
	plain.Second.Num.Cell = invalid
	plainHash, plainErr := s.InfHash(&plain)
	plain.Second.Num.Cell = plainCell
	if plainHash != 0 || !errors.Is(plainErr, arena.ErrInvalidSlot) {
		t.Fatal("inf_rational combined a failed second hash", plainHash, plainErr)
	}
	serviceMust(t, s.InfDel(&plain))

	integer := infIntValue(t, s, "5", 7)
	integerCell := integer.First.Num.Cell
	integer.First.Num.Cell = invalid
	integerHash, integerErr := s.InfIntHash(&integer)
	integer.First.Num.Cell = integerCell
	if integerHash != 0 || !errors.Is(integerErr, arena.ErrInvalidSlot) {
		t.Fatal("inf_int_rational combined a failed first hash", integerHash, integerErr)
	}
	serviceMust(t, s.InfIntDel(&integer))

	nested := infEpsValue(t, s, "1", "2", "3")
	nestedCell := nested.numeral.Second.Num.Cell
	nested.numeral.Second.Num.Cell = invalid
	nestedHash, nestedErr := s.InfEpsInfHash(&nested)
	nested.numeral.Second.Num.Cell = nestedCell
	if nestedHash != 0 || !errors.Is(nestedErr, arena.ErrInvalidSlot) {
		t.Fatal("inf_eps_rational combined a failed nested second hash", nestedHash, nestedErr)
	}
	serviceMust(t, s.InfEpsInfDel(&nested))
}
