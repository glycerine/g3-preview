// Source: src/util/mpzzp.h; src/util/mpzzp.cpp.
// Source author: Leonardo de Moura.
// Licensed under the MIT license.
package mpnum

func (m *IntegerModularManager) Reset(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Reset(a); e != nil {
		return e
	}
	return nil
}
func (m *IntegerModularManager) Del(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Del(a); e != nil {
		return e
	}
	return nil
}
func (m *IntegerModularManager) Neg(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Neg(a); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) Abs(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Abs(a); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) Inc(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.manager.Inc(a); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) Dec(a *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.manager.Dec(a); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) Add(a, b, c *Int) error {
	return m.addWithArithmeticReceipt(a, b, c, nil)
}
func (m *IntegerModularManager) addWithArithmeticReceipt(a, b, c *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.manager.addSubWithAllocationContext(a, b, c, false, integerAllocationInvocationContext{arithmetic: receipt}); e != nil {
		return e
	}
	return m.normalizeWithArithmeticReceipt(c, receipt)
}
func (m *IntegerModularManager) Sub(a, b, c *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.manager.Sub(a, b, c); e != nil {
		return e
	}
	return m.PNormalize(c)
}
func (m *IntegerModularManager) Mul(a, b, c *Int) error {
	return m.mulWithArithmeticReceipt(a, b, c, nil)
}
func (m *IntegerModularManager) mulWithArithmeticReceipt(a, b, c *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.mulWithSourceContext(a, b, c, integerAllocationInvocationContext{arithmetic: receipt})
}
func (m *IntegerModularManager) mulWithSourceContext(a, b, c *Int, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.manager.mulWithAllocationContext(a, b, c, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(c, context)
}
func (m *IntegerModularManager) AddMul(a, b, c, d *Int) error {
	return m.addMulWithArithmeticReceipt(a, b, c, d, nil)
}
func (m *IntegerModularManager) addMulWithArithmeticReceipt(a, b, c, d *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.assertNormalized(c); e != nil {
		return e
	}
	child := m.arithmeticAddMulReceipt(receipt)
	e := m.manager.addMulWithSourceReceipt(a, b, c, d, child, receipt)
	if child != nil && receipt.frame.sourceCause == nil {
		receipt.frame.sourceCause = m.manager.addMulInvocationSourceCause(child)
	}
	if e != nil {
		return e
	}
	return m.normalizeWithArithmeticReceipt(d, receipt)
}
func (m *IntegerModularManager) SubMul(a, b, c, d *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	if e := m.assertNormalized(c); e != nil {
		return e
	}
	if e := m.manager.SubMul(a, b, c, d); e != nil {
		return e
	}
	return m.PNormalize(d)
}
func (m *IntegerModularManager) IsZero(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsZero(a)
}
func (m *IntegerModularManager) IsOne(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsOne(a)
}
func (m *IntegerModularManager) IsPos(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsPos(a)
}
func (m *IntegerModularManager) IsNeg(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsNeg(a)
}
func (m *IntegerModularManager) IsNonPos(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsNonPos(a)
}
func (m *IntegerModularManager) IsNonNeg(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsNonNeg(a)
}
func (m *IntegerModularManager) IsMinusOne(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	return m.manager.IsMinusOne(a)
}
func (m *IntegerModularManager) Equal(a, b *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	if e := m.assertNormalized(b); e != nil {
		return false, e
	}
	return m.manager.Equal(a, b)
}
func (m *IntegerModularManager) Less(a, b *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	if e := m.assertNormalized(b); e != nil {
		return false, e
	}
	return m.manager.Less(a, b)
}
func (m *IntegerModularManager) LessEqual(a, b *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	if e := m.assertNormalized(b); e != nil {
		return false, e
	}
	return m.manager.LessEqual(a, b)
}
func (m *IntegerModularManager) Greater(a, b *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	if e := m.assertNormalized(b); e != nil {
		return false, e
	}
	return m.manager.Greater(a, b)
}
func (m *IntegerModularManager) GreaterEqual(a, b *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.assertNormalized(a); e != nil {
		return false, e
	}
	if e := m.assertNormalized(b); e != nil {
		return false, e
	}
	return m.manager.GreaterEqual(a, b)
}
func (m *IntegerModularManager) Swap(a, b *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.assertNormalized(a); e != nil {
		return e
	}
	if e := m.assertNormalized(b); e != nil {
		return e
	}
	return m.manager.Swap(a, b)
}
func (m *IntegerModularManager) SetInt32(a *Int, v int32) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.SetInt32(a, v); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) SetUint32(a *Int, v uint32) error {
	return m.setUint32WithSourceContext(a, v, integerAllocationInvocationContext{})
}
func (m *IntegerModularManager) setUint32WithSourceContext(a *Int, v uint32, context integerAllocationInvocationContext) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.setUint32WithAllocationContext(a, v, context); e != nil {
		return e
	}
	return m.normalizeWithSourceContext(a, context)
}
func (m *IntegerModularManager) SetInt64(a *Int, v int64) error {
	return m.setInt64WithArithmeticReceipt(a, v, nil)
}
func (m *IntegerModularManager) setInt64WithArithmeticReceipt(a *Int, v int64, receipt *polynomialArithmeticInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.setInt64WithAllocationContext(a, v, integerAllocationInvocationContext{arithmetic: receipt}); e != nil {
		return e
	}
	return m.normalizeWithArithmeticReceipt(a, receipt)
}
func (m *IntegerModularManager) SetUint64(a *Int, v uint64) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.SetUint64(a, v); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) SetString(a *Int, v string) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.SetString(a, v); e != nil {
		return e
	}
	return m.PNormalize(a)
}
func (m *IntegerModularManager) Copy(a, b *Int) error {
	return m.copyWithArithmeticReceipt(a, b, nil)
}
func (m *IntegerModularManager) copyWithArithmeticReceipt(a, b *Int, receipt *polynomialArithmeticInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.copyFromOwnerWithArithmeticReceipt(a, m.manager, b, receipt); e != nil {
		return e
	}
	return m.normalizeWithArithmeticReceipt(a, receipt)
}

// Exact target SET followed by source p_normalize, using the actual operand
// owner instead of a serializer or a temporary mirrored numeral backing.
func (m *IntegerModularManager) CopyFromOwner(a *Int, source *IntegerManager, b *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.copyFromOwnerCore(a, source, b)
}
func (m *IntegerModularManager) copyFromOwnerCore(a *Int, source *IntegerManager, b *Int) error {
	return m.copyFromOwnerWithAllocationReceipt(a, source, b, nil)
}
func (m *IntegerModularManager) copyFromOwnerWithAllocationReceipt(a *Int, source *IntegerManager, b *Int, receipt *integerCopyAllocationReceipt) error {
	return m.copyFromOwnerWithAllocationReceipts(a, source, b, receipt, nil)
}
func (m *IntegerModularManager) copyFromOwnerWithAllocationReceipts(a *Int, source *IntegerManager, b *Int, receipt *integerCopyAllocationReceipt, pn *integerPNormalizeInvocationReceipt) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.copyFromOwnerWithReceipt(a, source, b, receipt); e != nil {
		return e
	}
	return m.pNormalizeWithReceipt(a, pn)
}
func (m *IntegerModularManager) IsUint64(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return m.manager.IsUint64(a)
}
func (m *IntegerModularManager) IsUint64Mutable(a *Int) (bool, error) {
	if e := m.PNormalize(a); e != nil {
		return false, e
	}
	return m.manager.IsUint64(a)
}
func (m *IntegerModularManager) IsInt64(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return m.manager.IsInt64(a)
}
func (m *IntegerModularManager) IsInt64Mutable(a *Int) (bool, error) {
	if e := m.PNormalize(a); e != nil {
		return false, e
	}
	return m.manager.IsInt64(a)
}
func (m *IntegerModularManager) Uint64(a *Int) (uint64, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.Uint64(a)
}
func (m *IntegerModularManager) Uint64Mutable(a *Int) (uint64, error) {
	if e := m.PNormalize(a); e != nil {
		return 0, e
	}
	return m.manager.Uint64(a)
}
func (m *IntegerModularManager) Int64(a *Int) (int64, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.Int64(a)
}
func (m *IntegerModularManager) Int64Mutable(a *Int) (int64, error) {
	if e := m.PNormalize(a); e != nil {
		return 0, e
	}
	return m.manager.Int64(a)
}
func (m *IntegerModularManager) Float64Mutable(a *Int) (float64, error) {
	if e := m.PNormalize(a); e != nil {
		return 0, e
	}
	return m.manager.Float64(a)
}
func (m *IntegerModularManager) Hash(a *Int) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.Hash(a)
}
func (m *IntegerModularManager) PowerOfTwoMultiple(a *Int) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.PowerOfTwoMultiple(a)
}
func (m *IntegerModularManager) Log2(a *Int) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.Log2(a)
}
func (m *IntegerModularManager) MLog2(a *Int) (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.MLog2(a)
}
func (m *IntegerModularManager) Mul2k(a *Int, k uint32, out *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.Mul2k(a, k, out); e != nil {
		return e
	}
	return m.PNormalize(out)
}
func (m *IntegerModularManager) MachineDiv2k(a *Int, k uint32, out *Int) error {
	if e := m.check(); e != nil {
		return e
	}
	if e := m.manager.MachineDiv2k(a, k, out); e != nil {
		return e
	}
	return m.assertNormalized(out)
}
func (m *IntegerModularManager) IsSmall(a *Int) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	if e := m.manager.check(a); e != nil {
		return false, e
	}
	return a.IsSmall(), nil
}
func (m *IntegerModularManager) String(a *Int) (string, error) {
	if e := m.check(); e != nil {
		return "", e
	}
	if e := m.assertNormalized(a); e != nil {
		return "", e
	}
	return m.manager.String(a)
}

// FormatInto is the source display path, which does not assert normalization.
func (m *IntegerModularManager) FormatInto(a *Int, dst []byte) (int, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.manager.FormatInto(a, dst)
}
func (m *IntegerModularManager) Mul2kInPlace(a *Int, k uint32) error { return m.Mul2k(a, k, a) }
func (m *IntegerModularManager) MachineDiv2kInPlace(a *Int, k uint32) error {
	return m.MachineDiv2k(a, k, a)
}
