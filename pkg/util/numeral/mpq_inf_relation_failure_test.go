package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPQInfComplementRelationsStopAtFailedFirstComponent(t *testing.T) {
	checks := []struct {
		name string
		call func(*MPQInfManager, *MPQInf, *MPQInf, *Rational) (bool, error)
	}{
		{"less equal", func(p *MPQInfManager, a, b *MPQInf, _ *Rational) (bool, error) { return p.Le(a, b) }},
		{"greater equal", func(p *MPQInfManager, a, b *MPQInf, _ *Rational) (bool, error) { return p.Ge(a, b) }},
		{"less equal rational", func(p *MPQInfManager, a, _ *MPQInf, r *Rational) (bool, error) { return p.LeRational(a, r) }},
		{"greater equal rational", func(p *MPQInfManager, a, _ *MPQInf, r *Rational) (bool, error) { return p.GeRational(a, r) }},
		{"less equal kind", func(p *MPQInfManager, a, _ *MPQInf, r *Rational) (bool, error) { return p.LeKind(a, r, MPQInfZero) }},
		{"greater equal kind", func(p *MPQInfManager, a, _ *MPQInf, r *Rational) (bool, error) { return p.GeKind(a, r, MPQInfZero) }},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := mpqInfFixture(t, false)
			a := mpqInfValue(t, p, "1", "0")
			b := mpqInfValue(t, p, "2", "0")
			r := Rational{Num: NewInt(2), Den: NewInt(1)}
			a.First.Num.Cell = p.m.Z.FailedRef
			got, e := tc.call(p, &a, &b, &r)
			if got || !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("failed comparison published a relation", got, e)
			}
			a.First.Num.Cell = 0
			serviceMust(t, p.CleanupValue(&a))
			serviceMust(t, p.CleanupValue(&b))
		})
	}
}
