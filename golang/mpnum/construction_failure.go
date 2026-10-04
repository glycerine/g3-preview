// Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// ConstructionFailure is the already charged owner retaining real cleanup
// roots. CleanupCause is nil when cleanup is outstanding but has not refused.
type ConstructionFailure interface {
	error
	Unwrap() error
	Cleanup() error
	RuntimeOwner() *memory.Runtime
	OriginalCause() error
	CleanupCause() error
}
