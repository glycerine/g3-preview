//go:build !g3debug

package numeral

const sourceGCDDebug = false

func gcdSourceRequire(condition bool) error                     { return nil }
func (m *IntegerManager) gcdSourceGreaterEqual(a, b *Int) error { return nil }
func (m *IntegerManager) gcdSourceGreaterEqualAt(a, b *Int, line int32) error {
	return nil
}
