package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func upolyPhysicalBinaryVectorOwner(t *testing.T) (*unumeral.RationalManager, *unumeral.BinaryRationalManager, *BinaryRationalVector) {
	t.Helper()
	budget := arena.NewBudget(1 << 24)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = runtime
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 32}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	diagnostics, e := unumeral.NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	b, e := unumeral.NewBinaryRationalManager(q.Z, unumeral.BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	v, e := numericBinaryRationalManager_NewVector(b)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, v.Close())
		serviceMust(t, b.CloseChecked())
		serviceMust(t, diagnostics.Close())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, runtime.Close())
		if budget.Used() != 0 {
			t.Fatal("binary vector reset retained storage", budget.Used())
		}
	})
	return q, b, v
}

func TestUPolynomialBinaryVectorManagerResetPhysicalRetry(t *testing.T) {
	q, b, v := upolyPhysicalBinaryVectorOwner(t)
	var n unumeral.Int
	serviceMust(t, q.Z.SetLimbs(&n, []uint32{7, 0, 1}))
	a := unumeral.NewBinaryRational()
	serviceMust(t, b.SetIntegerPower(&a, &n, 3))
	serviceMust(t, q.Z.Del(&n))
	for i := 0; i < 3; i++ {
		serviceMust(t, upolyAppendBinaryFixture(v, &a))
	}
	serviceMust(t, b.Del(&a))
	ref := v.state.values[1].BorrowNumerator().Cell
	original := q.Z.Cell(ref)
	defer func() {
		if unumeral.ValidNumericRef(q.Z.Cells, ref) {
			q.Z.Store(ref, original)
		}
	}()
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	q.Z.Store(ref, bad)
	if e := v.resetManagerNumerals(); !errors.Is(e, arena.ErrBounds) || !v.state.managerResetPending || v.state.managerResetCursor != 1 || v.state.size != 3 {
		t.Fatal("source reset did not retain stopped second slot", e, v.state.managerResetCursor)
	}
	if v.state.values[0].BorrowNumerator().Cell != 0 || v.state.values[1].BorrowNumerator().Cell != ref || v.state.values[2].BorrowNumerator().Cell == 0 {
		t.Fatal("source ascending reset crossed stopped slot")
	}
	if _, e := v.Size(); !errors.Is(e, ErrBinaryVectorResetPending) {
		t.Fatal("pending reset allowed vector read", e)
	}
	if e := v.Reset(); !errors.Is(e, ErrBinaryVectorResetPending) {
		t.Fatal("pending reset allowed a second reset", e)
	}
	if e := v.Close(); !errors.Is(e, arena.ErrBounds) || v.state.managerResetCursor != 1 {
		t.Fatal("owning close lost stopped reset slot", e)
	}
	q.Z.Store(ref, original)
	serviceMust(t, v.CleanupManagerReset())
	if v.state.managerResetPending || v.state.managerResetCursor != 0 || v.state.size != 0 {
		t.Fatal("owning retry retained manager reset")
	}
	one := unumeral.NewBinaryInt(1)
	serviceMust(t, upolyAppendBinaryFixture(v, &one))
}

func TestUPolynomialBinaryVectorDropPhysicalRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target uint32
	}{
		{"reset", 0},
		{"shrink", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, b, v := upolyPhysicalBinaryVectorOwner(t)
			var n unumeral.Int
			serviceMust(t, q.Z.SetLimbs(&n, []uint32{7, 0, 1}))
			a := unumeral.NewBinaryRational()
			serviceMust(t, b.SetIntegerPower(&a, &n, 3))
			serviceMust(t, q.Z.Del(&n))
			for i := 0; i < 3; i++ {
				serviceMust(t, upolyAppendBinaryFixture(v, &a))
			}
			serviceMust(t, b.Del(&a))
			stopped := tc.target + 1
			ref := v.state.values[stopped].BorrowNumerator().Cell
			original := q.Z.Cell(ref)
			defer func() {
				if unumeral.ValidNumericRef(q.Z.Cells, ref) {
					q.Z.Store(ref, original)
				}
			}()
			bad := original
			bad.Backing.Requested = uint64(math.MaxUint32) + 1
			q.Z.Store(ref, bad)
			var e error
			if tc.target == 0 {
				e = v.Reset()
			} else {
				e = v.Shrink(tc.target)
			}
			if !errors.Is(e, arena.ErrBounds) || !v.state.dropPending || v.state.dropCursor != stopped || v.state.dropTo != tc.target || v.state.size != 3 {
				t.Fatal("source vector drop did not retain the stopped slot", e, v.state)
			}
			if v.state.values[tc.target].BorrowNumerator().Cell != 0 || v.state.values[stopped].BorrowNumerator().Cell != ref {
				t.Fatal("ascending deletion crossed stopped slot")
			}
			if _, e := v.Size(); !errors.Is(e, ErrBinaryVectorResetPending) {
				t.Fatal("pending drop allowed vector read", e)
			}
			if e := v.Close(); !errors.Is(e, arena.ErrBounds) || v.state.dropCursor != stopped {
				t.Fatal("owning close lost stopped drop slot", e)
			}
			q.Z.Store(ref, original)
			serviceMust(t, v.CleanupDrop())
			if v.state.dropPending || v.state.dropCursor != 0 || v.state.dropTo != 0 || v.state.size != tc.target {
				t.Fatal("owning retry retained vector drop", v.state)
			}
			one := unumeral.NewBinaryInt(1)
			serviceMust(t, upolyAppendBinaryFixture(v, &one))
		})
	}
}
