package arena

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

func TestVectorLifetime(t *testing.T) {
	v, e := NewVectors64(VectorConfig{PageSize: 8, Descriptors: Config{PageSize: 2}})
	if e != nil {
		t.Fatal(e)
	}
	defer v.Close()
	empty, e := v.Allocate(0, 0)
	if e != nil || empty != (VectorRef{}) {
		t.Fatal(empty, e)
	}
	a, e := v.Allocate(3, 3)
	if e != nil {
		t.Fatal(e)
	}
	if c, _ := v.Cap(a); c != 4 {
		t.Fatal(c)
	}
	for i := uint32(0); i < 3; i++ {
		v.Write(a, i, uint64(i)+1)
	}
	b, e := v.Reserve(a, 17)
	if e != nil || b != a {
		t.Fatal(b, e)
	}
	for i := uint32(0); i < 3; i++ {
		x, e := v.Read(a, i)
		if e != nil || x != uint64(i)+1 {
			t.Fatal(x, e)
		}
	}
	v.Resize(a, 1)
	if c, _ := v.Cap(a); c != 32 {
		t.Fatal("shrink changed capacity", c)
	}
	v.Resize(a, 3)
	x, _ := v.Read(a, 2)
	if x != 0 {
		t.Fatal("new elements not initialized")
	}
	v.Free(a)
	if v.IsLive(a) {
		t.Fatal("freed live")
	}
	if _, e = v.Read(a, 0); !errors.Is(e, ErrInvalidSlot) {
		t.Fatal(e)
	}
	c, e := v.Allocate(1, 1)
	if e != nil {
		t.Fatal(e)
	}
	if c.Slot == a.Slot && c.Generation <= a.Generation {
		t.Fatal("generation reused")
	}
	v.Reset()
	if v.IsLive(c) {
		t.Fatal("reset token live")
	}
	v.TrimFreePages()
	if v.Stats().PayloadBytes != 0 {
		t.Fatal(v.Stats())
	}
}

func TestVectorNonOverlapPlateau(t *testing.T) {
	v, _ := NewWords(VectorConfig{PageSize: 8, Descriptors: Config{PageSize: 2}})
	defer v.Close()
	a, _ := v.Allocate(2, 2)
	b, _ := v.Allocate(2, 2)
	v.Write(a, 0, 9)
	v.Write(b, 0, 8)
	if x, _ := v.Read(a, 0); x != 9 {
		t.Fatal("overlap")
	}
	v.Free(a)
	v.Free(b)
	for cycle := 0; cycle < 2; cycle++ {
		for n := 0; n < 100; n++ {
			a, _ = v.Allocate(2, 2)
			a, _ = v.Resize(a, 13)
			a, _ = v.Resize(a, 1)
			v.Free(a)
		}
	}
	baseline := v.Stats()
	for n := 0; n < 1000; n++ {
		a, _ = v.Allocate(2, 2)
		a, _ = v.Resize(a, 13)
		a, _ = v.Resize(a, 1)
		v.Free(a)
	}
	if v.Stats() != baseline {
		t.Fatal("capacity did not plateau", baseline, v.Stats())
	}
}

func TestVectorGrowthFailure(t *testing.T) {
	b := NewBudget(1 << 20)
	v, _ := NewVectors64(VectorConfig{PageSize: 4, Descriptors: Config{PageSize: 2, Budget: b}})
	a, _ := v.Allocate(2, 2)
	v.Write(a, 0, 55)
	before := v.Stats()
	b.FailAfter(0)
	if _, e := v.Reserve(a, 32); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if v.Stats() != before {
		t.Fatal("growth failure changed storage")
	}
	if x, _ := v.Read(a, 0); x != 55 {
		t.Fatal(x)
	}
	if _, e := v.Allocate(32, 32); !errors.Is(e, ErrInjectedFailure) {
		t.Fatal(e)
	}
	if v.Stats().Live != 1 {
		t.Fatal("failed allocation leaked", v.Stats())
	}
	v.Close()
	if b.Used() != 0 {
		t.Fatal("budget leaked", b.Used())
	}
}

func TestVectorStateMachine(t *testing.T) {
	v, _ := NewVectors64(VectorConfig{PageSize: 4, Descriptors: Config{PageSize: 2}})
	defer v.Close()
	var refs [5]VectorRef
	var model [5][]uint64
	r := rand.New(rand.NewSource(812))
	for step := 0; step < 2000; step++ {
		i := r.Intn(5)
		switch r.Intn(5) {
		case 0:
			n := r.Intn(23)
			ref, e := v.Resize(refs[i], uint32(n))
			if e != nil {
				t.Fatal(e)
			}
			refs[i] = ref
			if n <= len(model[i]) {
				model[i] = model[i][:n]
			} else {
				model[i] = append(model[i], make([]uint64, n-len(model[i]))...)
			}
		case 1:
			x := uint64(step)
			ref, e := v.Append(refs[i], x)
			if e != nil {
				t.Fatal(e)
			}
			refs[i] = ref
			model[i] = append(model[i], x)
		case 2:
			if e := v.Free(refs[i]); e != nil {
				t.Fatal(e)
			}
			refs[i] = VectorRef{}
			model[i] = nil
		case 3:
			v.TrimFreePages()
		case 4:
			if len(model[i]) > 0 {
				j := r.Intn(len(model[i]))
				model[i][j] = uint64(step)
				if e := v.Write(refs[i], uint32(j), uint64(step)); e != nil {
					t.Fatal(e)
				}
			}
		}
		for k := range refs {
			n, e := v.Len(refs[k])
			if e != nil || int(n) != len(model[k]) {
				t.Fatal(step, k, n, e)
			}
			for j, x := range model[k] {
				y, e := v.Read(refs[k], uint32(j))
				if e != nil || y != x {
					t.Fatal(step, k, j, y, x, e)
				}
			}
		}
	}
}

func TestVectorWarmAllocations(t *testing.T) {
	v, _ := NewWords(VectorConfig{PageSize: 8, Descriptors: Config{PageSize: 2}})
	a, _ := v.Allocate(2, 2)
	v.Free(a)
	n := testing.AllocsPerRun(1000, func() {
		a, e := v.Allocate(2, 2)
		if e != nil {
			panic(e)
		}
		v.Write(a, 0, 3)
		v.Read(a, 0)
		v.Free(a)
	})
	if n != 0 {
		t.Fatal(n)
	}
}

// Port of finite ENSURE assertions in src/test/vector.cpp::tst1. Indexed
// traversal replaces a raw-pointer iterator; back/pop are Read/Resize.
func TestSourceVectorTst1(t *testing.T) {
	v, _ := NewVectors32Signed(VectorConfig{})
	defer v.Close()
	ref := VectorRef{}
	if n, _ := v.Len(ref); n != 0 {
		t.Fatal(n)
	}
	for i := uint32(0); i < 1000; i++ {
		var e error
		ref, e = v.Append(ref, int32(i+3))
		if e != nil {
			t.Fatal(e)
		}
		x, e := v.Read(ref, i)
		if e != nil || x != int32(i+3) {
			t.Fatal(i, x, e)
		}
		n, _ := v.Len(ref)
		c, _ := v.Cap(ref)
		if n == 0 || c < n {
			t.Fatal(n, c)
		}
	}
	for i := uint32(0); i < 1000; i++ {
		x, _ := v.Read(ref, i)
		if x != int32(i+3) {
			t.Fatal(i, x)
		}
	}
	for i := uint32(0); i < 1000; i++ {
		n, _ := v.Len(ref)
		x, _ := v.Read(ref, n-1)
		if n != 1000-i || x != int32(1000-i-1+3) {
			t.Fatal(n, x)
		}
		v.Resize(ref, n-1)
	}
	if n, _ := v.Len(ref); n != 0 {
		t.Fatal(n)
	}
	// The source's billion-element unbounded allocation loop is replaced with
	// a deterministic finite budget failure, preserving the allocation error.
	b := NewBudget(4096)
	bounded, _ := NewWords(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 1, Budget: b}})
	defer bounded.Close()
	if _, e := bounded.Allocate(1000000000, 1000000000); !errors.Is(e, ErrBudget) {
		t.Fatal(e)
	}
}

func TestVectorEmptyReservedAndBounds(t *testing.T) {
	v, _ := NewWords(VectorConfig{PageSize: 4})
	defer v.Close()
	a, e := v.Allocate(0, 3)
	if e != nil || a == (VectorRef{}) || !v.IsLive(a) {
		t.Fatal(a, e)
	}
	if n, _ := v.Len(a); n != 0 {
		t.Fatal(n)
	}
	if c, _ := v.Cap(a); c != 4 {
		t.Fatal(c)
	}
	if _, e = v.Read(a, 0); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	if _, e = v.Allocate(3, 2); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	if _, e = v.Reserve(a, ^uint32(0)); !errors.Is(e, ErrBounds) {
		t.Fatal(e)
	}
	v.Close()
	if _, e = v.Len(a); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
	if e = ValidateRecordType(reflect.TypeOf((*int)(nil))); !errors.Is(e, ErrPointerful) {
		t.Fatal(e)
	}
	if _, e = NewWords(VectorConfig{PageSize: 3}); !errors.Is(e, ErrConfig) {
		t.Fatal(e)
	}
}

func TestVectorBudgetPeakDirectory(t *testing.T) {
	probe, _ := NewWords(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 2}})
	a, _ := probe.Allocate(1, 1)
	before := probe.Stats()
	probe.Reserve(a, 4)
	after := probe.Stats()
	probe.Close()
	total := after.PayloadBytes + after.BlockMetadataBytes + after.DirectoryBytes + after.Descriptors.RecordBytes + after.Descriptors.MetadataBytes + after.Descriptors.DirectoryBytes
	budget := NewBudget(total)
	v, _ := NewWords(VectorConfig{PageSize: 2, Descriptors: Config{PageSize: 2, Budget: budget}})
	defer v.Close()
	ref, e := v.Allocate(1, 1)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = v.Reserve(ref, 4); !errors.Is(e, ErrBudget) {
		t.Fatal("missing old+new directory peak", e)
	}
	if v.Stats() != before {
		t.Fatal("peak failure changed state", before, v.Stats())
	}
}

func BenchmarkVectorWarm(b *testing.B) {
	v, _ := NewWords(VectorConfig{})
	ref, _ := v.Allocate(3, 4)
	v.Free(ref)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref, _ := v.Allocate(3, 4)
		v.Write(ref, 0, uint32(i))
		v.Read(ref, 0)
		v.Free(ref)
	}
}
