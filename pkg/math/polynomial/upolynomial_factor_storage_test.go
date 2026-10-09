package polynomial

import "testing"

func TestUPolynomialBerlekampNativePivotResize(t *testing.T) {
	var values []int32
	for _, n := range []uint32{0, 1, 2, 3, 4, 6, 70, 4097} {
		old := len(values)
		if e := upolyFactorInt32Resize(&values, n, -1); e != nil {
			t.Fatal(n, e)
		}
		if len(values) != int(n) {
			t.Fatal(n, len(values))
		}
		for i := range values {
			want := int32(-1)
			if i < old {
				want = int32(i*37 - 128)
			}
			if values[i] != want {
				t.Fatal(n, i, values[i], want)
			}
			values[i] = int32(i*37 - 128)
		}
	}
	if e := upolyFactorInt32Resize(&values, 3, -1); e != nil {
		t.Fatal(e)
	}
	if e := upolyFactorInt32Resize(&values, 70, -1); e != nil {
		t.Fatal(e)
	}
	for i, v := range values {
		want := int32(-1)
		if i < 3 {
			want = int32(i*37 - 128)
		}
		if v != want {
			t.Fatal(i, v, want)
		}
	}
}
