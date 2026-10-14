// Concrete owner/lifetime tests.
// Source: src/util/mpbq.h, src/util/scoped_numeral_vector.h.
package numeral

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func binaryCellsAudit(t *testing.T, m *BinaryRationalManager, values ...*BinaryRational) {
	t.Helper()
	roots := []*Int{&m.Tmp, &m.Tmp2, &m.AddMulTmp.Num, &m.SelectIntTmp1, &m.SelectIntTmp2, &m.SelectSmallTmp, &m.SelectSmallTmp1.Num, &m.SelectSmallTmp2.Num, &m.DivTmp1, &m.DivTmp2, &m.DivTmp3}
	for i := range m.Display.Values {
		roots = append(roots, &m.Display.Values[i])
	}
	for _, v := range values {
		roots = append(roots, &v.Num)
	}
	qopsAuditLiveOwnership(t, &RationalManager{Z: m.Z, Synchronized: true}, roots)
}
func binaryAudit(t *testing.T, m *BinaryRationalManager, values ...*BinaryRational) {
	binaryCellsAudit(t, m, values...)
	facadeAudit(t, m.Diagnostics)
}
func TestBinaryOwnerConstructorAndScopedScalar(t *testing.T) {
	m, b := binaryManager(t)
	baseline := b.Used()
	b.FailAfter(0)
	if _, e := NewBinaryRationalManager(m.Z, BinaryRationalConfig{Diagnostics: m.Diagnostics}); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline {
		t.Fatal("constructor refusal")
	}
	b.FailAfter(-1)
	n, e := NewBinaryRationalManager(m.Z, BinaryRationalConfig{Diagnostics: m.Diagnostics})
	serviceMust(t, e)
	if b.Used()-baseline != uint64(reflect.TypeOf(BinaryRationalManager{}).Size()) {
		t.Fatal("actual header")
	}
	n.Close()
	n.Close()
	if b.Used() != baseline {
		t.Fatal("close borrows")
	}
	var a BinaryRational
	serviceMust(t, m.Z.SetString(&a.Num, "123456789012345678901234567890"))
	a.KField = 7
	s, e := NewScopedBinary(m)
	serviceMust(t, e)
	serviceMust(t, s.Set(&a))
	cl, e := s.Clone()
	serviceMust(t, e)
	serviceMust(t, cl.Close())
	copyScoped := *s
	if _, e = copyScoped.Get(); !errors.Is(e, ErrScopedCopy) {
		t.Fatal(e)
	}
	serviceMust(t, s.Close())
	serviceMust(t, s.Close())
	serviceMust(t, m.Del(&a))
	if e = arena.ValidateRecordType(reflect.TypeOf(BinaryRational{})); e != nil {
		t.Fatal(e)
	}
}
func TestBinaryFailureImmediateRoots(t *testing.T) {
	for op := 0; op < 8; op++ {
		completed := false
		for prefix := int64(0); prefix < 4096; prefix++ {
			m, b := binaryManager(t)
			var a, c, out BinaryRational
			serviceMust(t, m.Z.SetString(&a.Num, "100000000000000000000000000000000000000000000000000000000000000000000000003"))
			a.KField = 137
			serviceMust(t, m.SetInt32Power(&c, 7, 2))
			serviceMust(t, m.SetInt32(&out, 13))
			b.FailAfter(prefix)
			var e error
			var converted OwnedRational
			switch op {
			case 0:
				e = m.Add(&a, &c, &out)
			case 1:
				e = m.Mul(&a, &c, &a)
			case 2:
				e = m.Power(&a, 3, &out)
			case 3:
				_, e = m.RootLower(&a, 3, &out)
			case 4:
				e = m.ApproxDiv(&a, &c, &out, 256, false)
			case 5:
				_, e = m.DecimalString(&a, 32)
			case 6:
				e = m.AddMul(&a, &c, &a, &out)
			case 7:
				converted, e = m.ToOwnedRational(m.Diagnostics, &a)
			}
			binaryCellsAudit(t, m, &a, &c, &out)
			facadeAudit(t, m.Diagnostics, &converted)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual physical refusal", op, prefix, e)
			}
			b.FailAfter(-1)
			if e != nil && converted.ownerID != 0 {
				t.Fatal("failed conversion published owner")
			}
			if converted.ownerID != 0 {
				serviceMust(t, m.Diagnostics.Del(&converted))
			}
			if e == nil {
				completed = true
				t.Logf("operation %d first successful reservation ordinal %d", op, prefix)
			}
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&c))
			serviceMust(t, m.Del(&out))
			m.Close()
			m.Z.Close()
			serviceMust(t, m.Diagnostics.Close())
			if b.Used() != 0 {
				t.Fatal("close graph", op, prefix, b.Used())
			}
			if completed {
				break
			}
		}
		if !completed {
			t.Fatal("no successful reservation prefix", op)
		}
	}
}

func TestBinaryScopedRoutes(t *testing.T) {
	m, _ := binaryManager(t)
	s, e := NewScopedBinary(m)
	serviceMust(t, e)
	defer s.Close()
	serviceMust(t, s.SetInt32(3))
	v := NewInt(2)
	for op := 0; op < 6; op++ {
		got, e := s.macroCompare(&v, uint8(op))
		serviceMust(t, e)
		want := []bool{false, true, false, false, true, true}[op]
		if got != want {
			t.Fatal(op, got)
		}
	}
	for op := 0; op < 3; op++ {
		r, e := s.macroValue(&v, uint8(op))
		serviceMust(t, e)
		want := []string{"5", "1", "6"}[op]
		binaryText(t, m, &r.num, want)
		serviceMust(t, r.Close())
	}
	a, e := s.Clone()
	serviceMust(t, e)
	serviceMust(t, s.AddAssign(a))
	serviceMust(t, s.SubAssignValue(&a.num))
	serviceMust(t, s.MulAssign(a))
	binaryText(t, m, &s.num, "9")
	serviceMust(t, a.Close())
	binaryAudit(t, m, &s.num)
	foreign, _ := binaryManager(t)
	fs, e := NewScopedBinary(foreign)
	serviceMust(t, e)
	defer fs.Close()
	if e = s.Assign(fs); !errors.Is(e, ErrScopedOwner) {
		t.Fatal(e)
	}
}
