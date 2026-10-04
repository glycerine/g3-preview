//go:build !g3debug

// Licensed under the MIT license.
package g3

import "testing"

func sxFractionalBVSourceProfile(t *testing.T, m *SexprManager, h SexprHandle) {
	// Source sexpr.cpp:138-145 uses raw numerator Rem/IDiv in release:
	// 1/2 yields digit1 and quotient0 before the atom continuation boundary.
	sxRender(t, m, h, "#x1", ErrSexprDisplayContinuation)
}
