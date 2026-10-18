package numeral

import (
	"math/big"
	"testing"
)

func TestIntegerCompareFromIndependentOwnersAndReceiverScratch(t *testing.T) {
	receiver, budget := mpzPublicationDirectOwner(t)
	left, _ := mpzPublicationDirectOwner(t)
	right, _ := mpzPublicationDirectOwner(t)
	values := []string{"-340282366920938463463374607431768211459", "-4294967296", "-2147483648", "-1", "0", "1", "2147483647", "4294967296", "340282366920938463463374607431768211459"}
	for _, x := range values {
		for _, y := range values {
			var a, b Int
			serviceMust(t, left.SetString(&a, x))
			serviceMust(t, right.SetString(&b, y))
			beforeA, beforeB := a, b
			used := budget.Used()
			budget.FailAfter(0)
			order, e := receiver.CompareFromOwners(left, &a, right, &b)
			budget.FailAfter(-1)
			serviceMust(t, e)
			ax, _ := new(big.Int).SetString(x, 10)
			by, _ := new(big.Int).SetString(y, 10)
			if order != ax.Cmp(by) || a != beforeA || b != beforeB || budget.Used() != used {
				t.Fatal("foreign comparison", x, y, order, ax.Cmp(by))
			}
			serviceMust(t, left.Del(&a))
			serviceMust(t, right.Del(&b))
		}
	}
}
