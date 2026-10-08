// Source: src/util/version.h.in:2-7; src/util/version.h.cmake.in:2-8.
// Values: src/util/version.h:2-7.
package util

import (
	"fmt"
	"testing"
)

func TestVersionConstantsMatchGeneratedSourceHeader(t *testing.T) {
	if Z3MajorVersion != 4 || Z3MinorVersion != 7 || Z3BuildNumber != 1 || Z3RevisionNumber != 0 {
		t.Fatal("version tuple", Z3MajorVersion, Z3MinorVersion, Z3BuildNumber, Z3RevisionNumber)
	}
	want := fmt.Sprintf("Z3 %d.%d.%d.%d", Z3MajorVersion, Z3MinorVersion, Z3BuildNumber, Z3RevisionNumber)
	if Z3FullVersion != want {
		t.Fatal("full version", Z3FullVersion, want)
	}
}
