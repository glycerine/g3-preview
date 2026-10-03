// Copyright (c) 2011 Microsoft Corporation. Concrete owner/lifetime tests.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func binaryCellsAudit(t *testing.T, m *BinaryRationalManager, values ...*BinaryRational) {
	t.Helper()
	roots := []*Int{&m.tmp, &m.tmp2, &m.addMulTmp.num, &m.selectIntTmp1, &m.selectIntTmp2, &m.selectSmallTmp, &m.selectSmallTmp1.num, &m.selectSmallTmp2.num, &m.divTmp1, &m.divTmp2, &m.divTmp3}
	for _, v := range values {
		roots = append(roots, &v.num)
	}
	qopsAuditLiveOwnership(t, &RationalManager{z: m.z, synchronized: true}, roots)
}
func binaryAudit(t *testing.T, m *BinaryRationalManager, values ...*BinaryRational) {
	binaryCellsAudit(t, m, values...)
	facadeAudit(t, m.diagnostics)
}
func TestBinaryOwnerConstructorAndScopedVector(t *testing.T) {
	m, b := binaryManager(t)
	baseline := b.Used()
	b.FailAfter(0)
	if _, e := NewBinaryRationalManager(m.z, BinaryRationalConfig{Diagnostics: m.diagnostics}); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != baseline {
		t.Fatal("constructor refusal")
	}
	b.FailAfter(-1)
	n, e := NewBinaryRationalManager(m.z, BinaryRationalConfig{Diagnostics: m.diagnostics})
	serviceMust(t, e)
	if b.Used()-baseline != uint64(reflect.TypeOf(BinaryRationalManager{}).Size()) {
		t.Fatal("actual header")
	}
	n.Close()
	n.Close()
	if b.Used() != baseline {
		t.Fatal("close borrows")
	}
	v, e := NewScopedBinaryVector(m)
	serviceMust(t, e)
	defer v.Close()
	var a BinaryRational
	serviceMust(t, m.z.SetString(&a.num, "123456789012345678901234567890"))
	a.k = 7
	defer m.Del(&a)
	for i := 0; i < 8; i++ {
		serviceMust(t, v.Push(&a))
		want := uint32(2)
		for want < uint32(i+1) {
			want = (3*want + 1) / 2
		}
		if v.Capacity() != want {
			t.Fatal(i, v.Capacity(), want)
		}
	}
	roots := []*BinaryRational{&a}
	for i := range v.data {
		roots = append(roots, &v.data[i])
	}
	binaryAudit(t, m, roots...)
	serviceMust(t, v.Shrink(3))
	serviceMust(t, v.Pop())
	serviceMust(t, v.Resize(5))
	for i := 2; i < 5; i++ {
		if v.data[i].num.val != 0 || v.data[i].k != 0 {
			t.Fatal("new default")
		}
	}
	capBefore := v.Capacity()
	serviceMust(t, m.ResetVector(v))
	if v.Size() != 0 || v.Capacity() != capBefore {
		t.Fatal("retained backing")
	}
	binaryAudit(t, m, &a)
	copyOwner := *v
	if _, e = copyOwner.At(0); !errors.Is(e, ErrScopedCopy) {
		t.Fatal(e)
	}
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
	if e = arena.ValidateRecordType(reflect.TypeOf(BinaryRational{})); e != nil {
		t.Fatal(e)
	}
}
func TestBinaryFailureImmediateRoots(t *testing.T) {
	for op := 0; op < 8; op++ {
		completed := false
		for prefix := int64(0); prefix < 160; prefix++ {
			m, b := binaryManager(t)
			var a, c, out BinaryRational
			serviceMust(t, m.z.SetString(&a.num, "100000000000000000000000000000000000000000000000000000000000000000000000003"))
			a.k = 137
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
				converted, e = m.ToOwnedRational(m.diagnostics, &a)
			}
			b.FailAfter(-1)
			binaryCellsAudit(t, m, &a, &c, &out)
			facadeAudit(t, m.diagnostics, &converted)
			if e != nil && converted.ownerID != 0 {
				t.Fatal("failed conversion published owner")
			}
			if converted.ownerID != 0 {
				serviceMust(t, m.diagnostics.Del(&converted))
			}
			if e == nil {
				completed = true
			}
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&c))
			serviceMust(t, m.Del(&out))
			m.Close()
			m.z.Close()
			serviceMust(t, m.diagnostics.Close())
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

func TestBinaryVectorFailurePrefixesAndScopedRoutes(t *testing.T) {
	for prefix := int64(0); prefix < 12; prefix++ {
		m, b := binaryManager(t)
		v, e := NewScopedBinaryVector(m)
		serviceMust(t, e)
		var a BinaryRational
		serviceMust(t, m.z.SetString(&a.num, "123456789012345678901234567890123456789"))
		a.k = 9
		serviceMust(t, v.Push(&a))
		serviceMust(t, v.Push(&a))
		b.FailAfter(prefix)
		e = v.Push(&a)
		roots := []*BinaryRational{&a}
		for i := range v.data {
			roots = append(roots, &v.data[i])
		}
		binaryAudit(t, m, roots...)
		b.FailAfter(-1)
		if e != nil && v.Size() != 2 && v.Size() != 3 {
			t.Fatal(prefix, v.Size(), e)
		}
		if v.Size() == 3 && e != nil {
			// The vector publishes its default element before integer Copy;
			// Copy publishes sign and failed allocation identity before k.
			if v.data[2].num.cell != m.z.failedRef || v.data[2].num.val != a.num.val || v.data[2].k != 0 {
				t.Fatal("source integer-copy publication", prefix, v.data[2])
			}
			if x := m.z.check(&v.data[2].num); !errors.Is(x, arena.ErrInvalidSlot) {
				t.Fatal("failed numeral remains unreadable", x)
			}
		}
		serviceMust(t, v.Close())
		serviceMust(t, m.Del(&a))
		if e == nil {
			break
		}
	}
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
