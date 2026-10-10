package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Added branch fixtures for inf_rational.cpp:80-111,169-179; no new source ENSURE credit.
func TestInfGuardZeroFirstDivisionProfiles(t *testing.T) {
	for _, divisorEPS := range []string{"0", "-1", "1"} {
		for _, upper := range []bool{false, true} {
			for _, eps := range []string{"0", "-1", "1"} {
				s, _ := serviceNew(t)
				a := infValue(t, s, 2, 1, eps)
				b := infValue(t, s, 0, 1, divisorEPS)
				var out OwnedInfRational
				var err error
				if upper {
					out, err = s.InfUpperDiv(&a, &b)
				} else {
					out, err = s.InfLowerDiv(&a, &b)
				}
				if SourceExtendedGCDDebug {
					if !errors.Is(err, ErrInfRationalDomain) || out.ownerID != 0 {
						t.Fatalf("debug assertion: %v %#v", err, out)
					}
				} else if eps == "0" {
					if !errors.Is(err, ErrDivideByZero) || out.ownerID != 0 {
						t.Fatalf("reached 0/0: %v %#v", err, out)
					}
				} else {
					serviceMust(t, err)
					sign := int32(1)
					if eps == "-1" {
						sign = -1
					}
					if out.First.Num.Val != 1 || out.First.Den.Val != 0 || out.Second.Num.Val != sign || out.Second.Den.Val != 0 {
						t.Fatalf("raw source infinity: %#v", out)
					}
					if _, e := s.InfString(&out); !errors.Is(e, ErrUninitializedRational) {
						t.Fatal(e)
					}
				}
				roots := []*OwnedInfRational{&a, &b}
				if out.ownerID != 0 {
					roots = append(roots, &out)
				}
				infAuditRoots(t, s, roots, nil)
				if out.ownerID != 0 {
					serviceMust(t, s.InfDel(&out))
				}
				serviceMust(t, s.InfDel(&a))
				serviceMust(t, s.InfDel(&b))
			}
		}
	}
}

func TestInfGuardRootProfilesAndUnreadFields(t *testing.T) {
	for _, upper := range []bool{false, true} {
		s, _ := serviceNew(t)
		v := infValue(t, s, -2, 1, "-3")
		var out OwnedInfRational
		var err error
		if upper {
			out, err = s.InfUpperRoot(&v, 0)
		} else {
			out, err = s.InfLowerRoot(&v, 0)
		}
		if SourceExtendedGCDDebug {
			if !errors.Is(err, ErrInfRationalDomain) {
				t.Fatal(err)
			}
		} else {
			serviceMust(t, err)
			if upper {
				infText(t, s, &out, "(-2 -e*3)")
			} else {
				infText(t, s, &out, "0")
			}
			serviceMust(t, s.InfDel(&out))
		}
		// Source lower root has no component reads. A raw positive 1/0 passes
		// its scalar debug assertion; upper root copies the actual raw fields.
		v.First.Num = Int{Val: 1}
		v.First.Den = Int{}
		out, err = s.InfLowerRoot(&v, 17)
		serviceMust(t, err)
		serviceMust(t, s.InfDel(&out))
		out, err = s.InfUpperRoot(&v, 17)
		serviceMust(t, err)
		if out.First.Num.Val != 1 || out.First.Den.Val != 0 {
			t.Fatal(out)
		}
		infAuditRoots(t, s, []*OwnedInfRational{&v, &out}, nil)
		serviceMust(t, s.InfDel(&out))
		serviceMust(t, s.InfDel(&v))
	}
}

// Unread fields are intentionally malformed scalar tokens; they own no live
// storage. Restore them before caller destruction, rather than repairing math.
func TestInfGuardUnreadDivisorAndLowerRoot(t *testing.T) {
	s, _ := serviceNew(t)
	a := infValue(t, s, 6, 1, "2")
	b := infValue(t, s, 3, 1, "0")
	saved := b.Second
	b.Second.Den = Int{Val: 1, Cell: s.Manager.Z.FailedRef}
	b.Second.Num = Int{Val: 0, Cell: s.Manager.Z.FailedRef}
	out, e := s.InfLowerDiv(&a, &b)
	serviceMust(t, e)
	infText(t, s, &out, "(2 +e*2/3)")
	serviceMust(t, s.InfDel(&out))
	b.First.Num = Int{Val: 1, Cell: s.Manager.Z.FailedRef}
	out, e = s.InfLowerRoot(&b, 2)
	serviceMust(t, e)
	infText(t, s, &out, "0")
	serviceMust(t, s.InfDel(&out))
	if _, e = s.InfUpperRoot(&b, 2); e == nil {
		t.Fatal("upper root must reach invalid FIRST copy")
	}
	b.First = NewRational()
	b.Second = saved
	infAuditRoots(t, s, []*OwnedInfRational{&a, &b}, nil)
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&b))
}

func TestInfGuardAllocationPrecedenceAndWarmRoots(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	defer func() {
		b.FailAfter(-1)
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	a := infValue(t, s, 1, 1, "0")
	zero := infValue(t, s, 0, 1, "0")
	serviceMust(t, s.Manager.SetString(&a.First, "123456789012345678901234567890123456789"))
	b.FailAfter(0)
	out, e := s.InfLowerDiv(&a, &zero)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrInfRationalDomain) {
			t.Fatal(e)
		}
	} else if !numericPhysicalReservationRefusal(e) {
		t.Fatalf("Copy allocation must precede unsafe zero division: %v", e)
	}
	if out.ownerID != 0 {
		t.Fatal(out)
	}
	infAuditRoots(t, s, []*OwnedInfRational{&a, &zero}, nil)
	b.FailAfter(-1)
	serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&zero))
	// All scalar constructor routes in this cluster are inline. Both root
	// results and division's rational(2) branch require no raw allocation.
	a = infValue(t, s, 2, 1, "2")
	d := infValue(t, s, 2, 1, "1")
	out, e = s.InfLowerDiv(&a, &d)
	serviceMust(t, e)
	serviceMust(t, s.InfDel(&out))
	before := numericWarmProfileSnapshot(t, s.Manager.Z, b)
	allocs := testing.AllocsPerRun(10, func() {
		r, err := s.InfLowerDiv(&a, &d)
		if err != nil {
			panic(err)
		}
		if err = s.InfDel(&r); err != nil {
			panic(err)
		}
		r, err = s.InfLowerRoot(&a, 0)
		if err != nil {
			panic(err)
		}
		if err = s.InfDel(&r); err != nil {
			panic(err)
		}
		r, err = s.InfUpperRoot(&a, 0)
		if err != nil {
			panic(err)
		}
		if err = s.InfDel(&r); err != nil {
			panic(err)
		}
	})
	numericWarmProfileAudit(t, s.Manager.Z, b, before, allocs, false)
	infAuditRoots(t, s, []*OwnedInfRational{&a, &d}, nil)
	serviceMust(t, s.InfDel(&a))
	serviceMust(t, s.InfDel(&d))
}
