// Source: src/math/polynomial numeric invocation adapters.
// Licensed under the MIT license.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func numericIntegerManager_addMulWithSourceReceipt(m *util.IntegerManager, a, b, c, d *util.Int, receipt *util.IntegerAddMulInvocationReceipt, arithmetic *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext(((integerAllocationInvocationContext{}).withArithmeticInvocation(arithmetic)).utility(m)).AddMulTracked(a, b, c, d, receipt)
}

func numericRationalManager_setIntegerWithArithmeticReceipt(m *util.RationalManager, a *util.Rational, n *util.Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext(((integerAllocationInvocationContext{}).withArithmeticInvocation(receipt)).utility(m.Integers())).SetInteger(a, n)
}

func numericIntegerModularManager_addWithArithmeticReceipt(m *util.IntegerModularManager, a, b, c *util.Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext((numericIntegerModularManager_arithmeticAllocationContext(m, receipt)).utility(m.Integers())).Add(a, b, c)
}

func numericIntegerModularManager_mulWithArithmeticReceipt(m *util.IntegerModularManager, a, b, c *util.Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext((numericIntegerModularManager_arithmeticAllocationContext(m, receipt)).utility(m.Integers())).Mul(a, b, c)
}

func numericIntegerModularManager_addMulWithArithmeticReceipt(m *util.IntegerModularManager, a, b, c, d *util.Int, receipt *polynomialArithmeticInvocationReceipt) error {
	receipt = numericIntegerModularManager_arithmeticReceiptForReceiver(m, receipt)
	child := numericIntegerModularManager_arithmeticAddMulReceipt(m, receipt)
	e := m.WithAllocationContext((numericIntegerModularManager_arithmeticAllocationContext(m, receipt)).utility(m.Integers())).AddMulTracked(a, b, c, d, child)
	if child != nil && receipt.frame.sourceCause == nil {
		receipt.frame.sourceCause = m.Integers().AddMulInvocationSourceCause(child)
	}
	return e
}

func numericIntegerModularManager_setInt64WithArithmeticReceipt(m *util.IntegerModularManager, a *util.Int, v int64, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext((numericIntegerModularManager_arithmeticAllocationContext(m, receipt)).utility(m.Integers())).SetInt64(a, v)
}

func numericIntegerModularManager_copyWithArithmeticReceipt(m *util.IntegerModularManager, a, b *util.Int, receipt *polynomialArithmeticInvocationReceipt) error {
	return m.WithAllocationContext((numericIntegerModularManager_arithmeticAllocationContext(m, receipt)).utility(m.Integers())).Copy(a, b)
}
