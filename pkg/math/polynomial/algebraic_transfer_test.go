package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestAlgebraicSwapTransfersCellsWithoutNumericWork(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	var basic, zero Algebraic
	big := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&big, "123456789012345678901234567890123456789"))
	serviceMust(t, c.SetRational(&basic, &big))
	serviceMust(t, c.config.Rationals.Del(&big))
	defer func() {
		serviceMust(t, c.Reset(&zero))
		serviceMust(t, c.Reset(&basic))
		serviceMust(t, c.Reset(&root))
	}()
	originalRoot, originalBasic := root, basic
	before := c.config.Rationals.Budget().Used()
	checks := control.checks
	// An actual reservation refusal frontier must not be reached by Swap.
	c.config.Rationals.Budget().FailAfter(0)
	e := c.Swap(&root, &basic)
	c.config.Rationals.Budget().FailAfter(-1)
	serviceMust(t, e)
	if root != originalBasic || basic != originalRoot {
		t.Fatal("swap copied cells")
	}
	serviceMust(t, c.Swap(&basic, &zero))
	if basic != (Algebraic{}) || zero != originalRoot {
		t.Fatal("null swap lost ownership")
	}
	serviceMust(t, c.Swap(&zero, &zero))
	if zero != originalRoot || c.config.Rationals.Budget().Used() != before || control.checks != checks {
		t.Fatal("swap performed numerical work")
	}
	serviceMust(t, c.config.Rationals.SetString(&big, "123456789012345678901234567890123456789"))
	eq, e := c.EqualRational(&root, &big)
	serviceMust(t, e)
	serviceMust(t, c.config.Rationals.Del(&big))
	if !eq {
		t.Fatal("BIG rational lost its value")
	}
	algebraicExpectBetween(t, c, &zero, 1, 2)
	other, _, _ := algebraicCompareFixture(t)
	var foreign Algebraic
	one := unumeral.NewInt(1)
	serviceMust(t, other.SetInteger(&foreign, &one))
	oldForeign := foreign
	if e = c.Swap(&root, &foreign); !errors.Is(e, ErrAlgebraicOwner) || root != originalBasic || foreign != oldForeign {
		t.Fatal("invalid swap partially transferred ownership", e)
	}
	serviceMust(t, other.Reset(&foreign))
	serviceMust(t, c.Reset(&zero))
	if zero != (Algebraic{}) {
		t.Fatal("reset did not clear root cell")
	}
	serviceMust(t, c.Reset(&zero))
}
