//go:build !g3debug

// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package mpnum

const sourceGCDDebug = false

func gcdSourceRequire(condition bool) error                     { return nil }
func (m *IntegerManager) gcdSourceGreaterEqual(a, b *Int) error { return nil }
