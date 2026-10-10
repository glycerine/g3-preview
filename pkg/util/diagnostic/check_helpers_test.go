package diagnostic

import "testing"

func serviceMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func paramsMust(t *testing.T, e error) { serviceMust(t, e) }
