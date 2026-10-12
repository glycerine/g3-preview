package numeral

import (
	"errors"
	"math"
	"testing"
)

func TestFixedPointPrevPowerOfTwoSourceIntegerWords(t *testing.T) {
	m, err := NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := v.Close(); err != nil {
			t.Fatal(err)
		}
		if err := m.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	for _, tc := range []struct {
		num, den int64
		want     uint32
	}{
		{-3, 1, 0}, {0, 1, 0}, {1, 2, math.MaxUint32},
		{1, 1, 0}, {3, 2, 0}, {2, 1, 1}, {3, 1, 1},
		{4, 1, 2}, {1023, 1, 9}, {1024, 1, 10},
	} {
		if err := m.SetRatio(v, tc.num, uint64(tc.den)); err != nil {
			t.Fatal(err)
		}
		got, err := m.PrevPowerOfTwo(v)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("prev power of two for %d/%d = %d, want %d", tc.num, tc.den, got, tc.want)
		}
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.PrevPowerOfTwo(v); !errors.Is(err, ErrFixedPointClosed) {
		t.Fatalf("closed numeral error = %v", err)
	}
}
