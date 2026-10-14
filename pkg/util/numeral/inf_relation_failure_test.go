package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// A failed source component comparison has no boolean result to invert. The
// same-type and rational-mixed relations must both preserve that boundary.
func TestInfRelationsDoNotPublishComplementAfterComponentFailure(t *testing.T) {
	s, _ := serviceNew(t)
	check := func(name string, got bool, e error) {
		t.Helper()
		if got || !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatalf("%s: got %t, %v", name, got, e)
		}
	}

	r := serviceValue(t, s, "1")
	a, b := infApproxRaw(s, 1, 1, 0, 1), infApproxRaw(s, 1, 1, 0, 1)
	a.First.Num.Cell = s.Manager.Z.FailedRef
	for name, fn := range map[string]func(*OwnedInfRational, *OwnedInfRational) (bool, error){
		"not equal":     s.InfNotEqual,
		"less equal":    s.InfLessEqual,
		"greater equal": s.InfGreaterEqual,
	} {
		got, e := fn(&a, &b)
		check("inf "+name, got, e)
	}
	for name, fn := range map[string]func(*OwnedInfRational, *OwnedRational) (bool, error){
		"not equal":     s.InfNotEqualRational,
		"less equal":    s.InfLessEqualRational,
		"greater equal": s.InfGreaterEqualRational,
	} {
		got, e := fn(&a, &r)
		check("inf mixed "+name, got, e)
	}
	for name, fn := range map[string]func(*OwnedRational, *OwnedInfRational) (bool, error){
		"not equal":     s.InfRationalNotEqual,
		"less equal":    s.InfRationalLessEqual,
		"greater equal": s.InfRationalGreaterEqual,
	} {
		got, e := fn(&r, &a)
		check("rational inf "+name, got, e)
	}
	a.First.Num.Cell = 0
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))

	x, y := s.infEpsInfEmpty(), s.infEpsInfEmpty()
	x.infinity.Num.Cell = s.Manager.Z.FailedRef
	for name, fn := range map[string]func(*OwnedInfEpsInfRational, *OwnedInfEpsInfRational) (bool, error){
		"not equal":     s.InfEpsInfNotEqual,
		"less equal":    s.InfEpsInfLessEqual,
		"greater equal": s.InfEpsInfGreaterEqual,
	} {
		got, e := fn(&x, &y)
		check("inf eps "+name, got, e)
	}
	x.infinity.Num.Cell = 0
	r.Raw.Num.Cell = s.Manager.Z.FailedRef
	for name, fn := range map[string]func(*OwnedInfEpsInfRational, *OwnedRational) (bool, error){
		"not equal":     s.InfEpsInfNotEqualRational,
		"less equal":    s.InfEpsInfLessEqualRational,
		"greater equal": s.InfEpsInfGreaterEqualRational,
	} {
		got, e := fn(&x, &r)
		check("inf eps mixed "+name, got, e)
	}
	for name, fn := range map[string]func(*OwnedRational, *OwnedInfEpsInfRational) (bool, error){
		"not equal":     s.InfEpsInfRationalNotEqual,
		"less equal":    s.InfEpsInfRationalLessEqual,
		"greater equal": s.InfEpsInfRationalGreaterEqual,
	} {
		got, e := fn(&r, &x)
		check("rational inf eps "+name, got, e)
	}
	r.Raw.Num.Cell = 0
	serviceMust(t, s.InfEpsInfDel(&x))
	serviceMust(t, s.InfEpsInfDel(&y))
	serviceMust(t, s.Del(&r))
	infAuditRoots(t, s, nil, nil)
}
