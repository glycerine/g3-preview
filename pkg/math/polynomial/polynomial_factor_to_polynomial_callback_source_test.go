// Source: src/math/polynomial/polynomial.cpp:6832-6841.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactorToPolynomialTransportCannotRetireExecutingOwner(t *testing.T) {
	for _, operation := range []string{"fullSource", "fullClose", "coreSource", "coreClose", "integerSource", "integerClose", "rationalSource", "rationalClose", "typedCopy", "publicSet", "typedSource", "typedOwn"} {
		t.Run(operation, func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			var full *MultivariatePolynomialManager
			var fs *PolynomialFactors
			var q *util.RationalManager
			var callbackOut, callbackInput util.Int
			active, seen := false, false
			rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
				if active {
					seen = true
					before := rtStateForFactors(fs)
					fullClosing, fullSource, fullStage := full.closing, full.sourceStarted, full.cleanupStage
					coreClosing, coreSource, coreStage := full.core.closing, full.core.sourceStarted, full.core.closeStage
					z := full.core.integer
					zClosed, zHeader, zStage, zStarted := z.Closed(), z.HeaderBytes, z.SourceFinalizeStage, z.SourceFinalizing
					qClosing, qStarted, qCause := q.Closing, q.SourceFinalizing, q.CleanupCauseField
					used := b.Used()
					oldOut := callbackOut
					oldValue, inspectError := z.Compare(&callbackOut, &z.Two64)
					serviceMust(t, inspectError)
					var err error
					switch operation {
					case "source":
						err = fs.SourceFinalize()
					case "close":
						err = fs.CloseChecked()
					case "cleanup":
						err = fs.Cleanup()
					case "fullSource":
						err = full.SourceFinalize()
					case "fullClose":
						err = full.CloseChecked()
					case "coreSource":
						err = full.core.SourceFinalizePrefix()
					case "coreClose":
						err = full.core.CloseChecked()
					case "integerSource":
						err = z.SourceFinalize()
					case "integerClose":
						err = z.CloseChecked()
					case "rationalSource":
						err = q.SourceFinalize()
					case "rationalClose":
						err = q.CloseChecked()
					case "typedCopy":
						err = numericIntegerModularManager_copyWithToPolynomialReceipt(full.core.modular, &callbackOut, &callbackInput, &full.toPolynomial.direct)
					case "typedSource":
						err = full.finishToPolynomial(nil)
					case "typedOwn":
						err = full.cleanupToPolynomialInvocation(&full.toPolynomial.direct, true)
					case "publicSet":
						_, err = full.ToPolynomial(1, []util.Int{callbackInput}, 0)

					}
					newValue, inspectError := z.Compare(&callbackOut, &z.Two64)
					serviceMust(t, inspectError)
					if err != ErrPolynomialPending || before != rtStateForFactors(fs) || used != b.Used() || full.closing != fullClosing || full.sourceStarted != fullSource || full.cleanupStage != fullStage || full.core.closing != coreClosing || full.core.sourceStarted != coreSource || full.core.closeStage != coreStage || z.Closed() != zClosed || z.HeaderBytes != zHeader || z.SourceFinalizeStage != zStage || z.SourceFinalizing != zStarted || q.Closing != qClosing || q.SourceFinalizing != qStarted || q.CleanupCauseField != qCause || callbackOut != oldOut || newValue != oldValue {
						t.Errorf("executing factor owner changed: %v before=%+v after=%+v", err, before, rtStateForFactors(fs))
					}
				}
				return f
			}})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			q, e = util.NewRationalManager(util.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			fs, e = NewPolynomialFactors(full)
			serviceMust(t, e)
			var a util.Int
			digits := make([]uint32, 80)
			digits[79] = 1
			serviceMust(t, q.Integers().SetLimbs(&a, digits))
			serviceMust(t, q.Integers().Copy(&callbackInput, &a))
			serviceMust(t, q.Integers().Copy(&callbackOut, &q.Integers().Two64))
			pad := int64(99996) - rt.State().LocalSize - 8
			if pad <= 0 {
				t.Fatal("actual TLS frontier", rt.State())
			}
			held, e := rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			active = true
			_, cause := full.ToPolynomial(1, []util.Int{a}, 0)
			active = false
			if !seen || !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual SET returning transport", seen, cause)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, nil, []*util.Int{&a, &callbackInput, &callbackOut, &fs.constant, &fs.local}, fs.header+uint64(cap(fs.constantHistory))*uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size()), []util.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
			serviceMust(t, full.CleanupToPolynomial())
			serviceMust(t, fs.CloseChecked())
			serviceMust(t, q.Integers().Del(&a))
			serviceMust(t, q.Integers().Del(&callbackOut))
			serviceMust(t, q.Integers().Del(&callbackInput))
			serviceMust(t, full.CloseChecked())
			serviceMust(t, q.CloseChecked())
			serviceMust(t, mm.CloseChecked())
			serviceMust(t, rt.Deallocate(&held))
			serviceMust(t, rt.Close())
			if b.Used() != 0 {
				t.Fatal("actual owner retained", b.Used())
			}
		})
	}
}
