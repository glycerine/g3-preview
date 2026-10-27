package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicVectorBorrowPublishesReplacement(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	v, err := c.NewVector()
	serviceMust(t, err)
	defer func() { serviceMust(t, v.Close()) }()
	var value Algebraic
	one := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&value, &one))
	serviceMust(t, v.pushTake(&value))
	borrow, err := v.Borrow(0)
	serviceMust(t, err)
	serviceMust(t, c.Del(borrow))
	two := unumeral.NewInt(2)
	serviceMust(t, c.SetInteger(borrow, &two))
	actual, err := v.Get(0)
	serviceMust(t, err)
	if actual != *borrow {
		t.Fatal("replacement not published")
	}
	want := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&want, 2))
	defer func() { serviceMust(t, c.config.Rationals.Del(&want)) }()
	cmp, err := c.CompareRational(borrow, &want)
	serviceMust(t, err)
	if cmp != 0 {
		t.Fatal("replacement value")
	}
	if _, err = v.Borrow(1); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("bounds", err)
	}
	serviceMust(t, v.Close())
	if _, err = v.Borrow(0); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed borrow", err)
	}
}
