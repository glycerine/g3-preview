package bit_blaster

import (
	"testing"
)

func paramsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
