//go:build !g3debug

// Licensed under the MIT license.
package util

const sourceGCDDebug = false

func gcdSourceRequire(condition bool) error                     { return nil }
func (m *IntegerManager) gcdSourceGreaterEqual(a, b *Int) error { return nil }
