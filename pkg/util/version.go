// Source: src/util/version.h.in:2-7; src/util/version.h.cmake.in:2-8.
// Values: src/util/version.h:2-7.
package util

import utilversion "github.com/glycerine/g3/pkg/util/version"

// Compatibility constants for callers that have not moved to the leaf.
const (
	Z3MajorVersion   = utilversion.Major
	Z3MinorVersion   = utilversion.Minor
	Z3BuildNumber    = utilversion.Build
	Z3RevisionNumber = utilversion.Revision
	Z3FullVersion    = utilversion.Full
)
