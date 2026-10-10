// Source: src/math/polynomial/polynomial.cpp:6577-6590.
// Invalid copied/stale Go receipt domains must refuse before the numerical body.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestPolynomialFactorConstantSuppliedReceiptRefusalBeforeBIGBody(t *testing.T) {
	for _, domain := range []string{"stale", "copiedFrame", "copiedOwner", "foreignReceiver", "callbackDepth", "nestedLeaf"} {
		for _, operation := range []string{"copy", "mul", "neg"} {
			t.Run(domain+"/"+operation, func(t *testing.T) {
				full := polynomialPSCFullFixture(t)
				fs, e := NewPolynomialFactors(full)
				serviceMust(t, e)
				z := full.core.integer
				var a, out unumeral.Int
				digits := make([]uint32, 80)
				digits[79] = 1
				digits[0] = 7
				serviceMust(t, z.SetLimbs(&a, digits))
				serviceMust(t, z.Copy(&out, &z.Two64))
				var prior, current polynomialFactorConstantInvocationReceipt
				three := unumeral.NewInt(3)
				serviceMust(t, fs.setConstantWithReceipt(&three, &prior))
				s, e := fs.beginFactorConstant(0, &current)
				serviceMust(t, e)
				r := current
				receiver := full.core.modular
				var foreign *unumeral.IntegerModularManager
				var nested polynomialNumeralOperationGuard
				extra := uint64(0)
				want := ErrPolynomialOwner
				switch domain {
				case "stale":
					r = prior
				case "copiedFrame":
					copied := *s
					r.frame = &copied
				case "copiedOwner":
					copied := *fs
					r.manager = &copied
				case "foreignReceiver":
					foreign, e = unumeral.NewIntegerModularManager(z)
					serviceMust(t, e)
					receiver = foreign
					extra = uint64(reflect.TypeOf(unumeral.IntegerModularManager{}).Size())
				case "callbackDepth":
					full.core.callbackDepth++
				case "nestedLeaf":
					serviceMust(t, nested.enter(z))
					want = ErrPolynomialPending
				}
				before := out
				value := integerString(t, z, &out)
				used := z.Budget().Used()
				state := rtStateForFactors(fs)
				numeric := s.numeric
				var cause error
				switch operation {
				case "copy":
					cause = numericIntegerModularManager_copyWithFactorConstantReceipt(receiver, &out, &a, &r)
				case "mul":
					cause = numericIntegerModularManager_mulWithFactorConstantReceipt(receiver, &a, &a, &out, &r)
				case "neg":
					cause = numericIntegerModularManager_negWithFactorConstantReceipt(receiver, &out, &r)
				}
				if cause != want || out != before || integerString(t, z, &out) != value || z.Budget().Used() != used || state != rtStateForFactors(fs) || (s.numeric).NumericAllocationRoots != (numeric).NumericAllocationRoots {
					t.Fatalf("invalid supplied receipt entered BIG numerical body: %v want %v", cause, want)
				}
				polynomialFactorsAuditExtra(t, fs, nil, []*unumeral.Int{&a, &out}, extra)
				full.core.callbackDepth = 0
				if nested.executing {
					nested.leave()
				}
				s.executing = false
				serviceMust(t, fs.finishFactorConstantSource())
				serviceMust(t, fs.cleanupFactorConstantInvocation(&current, true))
				if foreign != nil {
					serviceMust(t, foreign.Close())
				}
				serviceMust(t, z.Del(&out))
				serviceMust(t, z.Del(&a))
				serviceMust(t, fs.CloseChecked())
			})
		}
	}
}
