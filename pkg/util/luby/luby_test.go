// Source: src/util/luby.cpp:21-31; src/util/luby.h:22-29.
package luby

import "testing"

func TestSourceSequence(t *testing.T) {
	want := []uint32{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8}
	for i, expected := range want {
		actual, err := Value(uint32(i + 1))
		if err != nil || actual != expected {
			t.Fatal(i+1, actual, expected, err)
		}
	}
}

func TestSourceUint32Wrap(t *testing.T) {
	actual, err := Value(^uint32(0))
	if err != nil || actual != 0 {
		t.Fatal(actual, err)
	}
}
