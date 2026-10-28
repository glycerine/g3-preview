package version

// Source: src/util/version.h.in:2-7; src/util/version.h.cmake.in:2-8.
// Values: src/util/version.h:2-7.

import (
	"fmt"
	"testing"
)

func TestVersionConstantsMatchGeneratedSourceHeader(t *testing.T) {
	if Major != 4 || Minor != 7 || Build != 1 || Revision != 0 {
		t.Fatal("version tuple", Major, Minor, Build, Revision)
	}
	want := fmt.Sprintf("Z3 %d.%d.%d.%d", Major, Minor, Build, Revision)
	if Full != want {
		t.Fatal("full version", Full, want)
	}
}
