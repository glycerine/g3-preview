// Source: src/math/polynomial/upolynomial.h:401-423.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestPolynomialFactorToNumeralVectorTransportCannotRetireExecutingOwner(t *testing.T) {
	for _, operation := range []string{"fullSource", "fullClose", "coreSource", "coreClose", "integerSource", "integerClose", "rationalSource", "rationalClose", "publicSet", "typedSource", "typedOwn"} {
		t.Run(operation, func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			var full *MultivariatePolynomialManager
			var fs *PolynomialFactors
			var q *RationalManager
			var callbackOut, callbackInput Int
			var p Polynomial
			var output upolyVectorState
			active, seen := false, false
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
				if active {
					seen = true
					before := rtStateForFactors(fs)
					fullClosing, fullSource, fullStage := full.closing, full.sourceStarted, full.cleanupStage
					coreClosing, coreSource, coreStage := full.core.closing, full.core.sourceStarted, full.core.closeStage
					z := full.core.integer
					zClosed, zHeader, zStage, zStarted := z.closed, z.headerBytes, z.sourceFinalizeStage, z.sourceFinalizing
					qClosing, qStarted, qCause := q.closing, q.sourceFinalizing, q.cleanupCause
					used := b.Used()
					oldOut := callbackOut
					oldValue, inspectError := z.Compare(&callbackOut, &z.two64)
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
						err = full.upm.modular.copyWithSourceContext(&callbackOut, &callbackInput, integerAllocationInvocationContext{factorNumeralVector: &full.factorNumeralVector.direct, factorNumeralVectorInvocation: full.factorNumeralVector.invocation})
					case "typedSource":
						err = full.finishFactorNumeralVectorSource()
					case "typedOwn":
						err = full.cleanupFactorNumeralVector(&full.factorNumeralVector.direct, true)
					case "publicSet":
						err = full.toNumeralVectorWithReceipt(p, &output, nil)

					}
					newValue, inspectError := z.Compare(&callbackOut, &z.two64)
					serviceMust(t, inspectError)
					if err != ErrPolynomialPending || before != rtStateForFactors(fs) || used != b.Used() || full.closing != fullClosing || full.sourceStarted != fullSource || full.cleanupStage != fullStage || full.core.closing != coreClosing || full.core.sourceStarted != coreSource || full.core.closeStage != coreStage || z.closed != zClosed || z.headerBytes != zHeader || z.sourceFinalizeStage != zStage || z.sourceFinalizing != zStarted || q.closing != qClosing || q.sourceFinalizing != qStarted || q.cleanupCause != qCause || callbackOut != oldOut || newValue != oldValue {
						t.Errorf("executing factor owner changed: %v before=%+v after=%+v", err, before, rtStateForFactors(fs))
					}
				}
				return f
			}})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			q, e = NewRationalManager(RationalConfig{Integers: cfg})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			fs, e = NewPolynomialFactors(full)
			serviceMust(t, e)
			var a Int
			digits := make([]uint32, 80)
			digits[79] = 1
			serviceMust(t, q.z.SetLimbs(&a, digits))
			serviceMust(t, q.z.Copy(&callbackInput, &a))
			serviceMust(t, q.z.Copy(&callbackOut, &q.z.two64))
			var coefficient Int
			serviceMust(t, q.z.Copy(&coefficient, &a))
			p, e = full.core.MkConstantTake(&coefficient)
			serviceMust(t, e)
			serviceMust(t, full.core.IncRef(p))
			np, e := full.Normalize(p)
			serviceMust(t, e)
			serviceMust(t, full.core.IncRef(np))
			serviceMust(t, full.core.DecRef(np))
			pad := int64(99996) - rt.State().LocalSize - 8
			if pad <= 0 {
				t.Fatal("actual TLS frontier", rt.State())
			}
			held, e := rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			active = true
			cause := full.toNumeralVectorWithReceipt(p, &output, nil)
			active = false
			if !seen || !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual SET returning transport", seen, cause)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, []Polynomial{p}, []*Int{&a, &callbackInput, &callbackOut, &fs.constant, &fs.local}, fs.header+uint64(cap(fs.constantHistory))*uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size()), []memory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
			serviceMust(t, full.cleanupFactorNumeralVector(&full.factorNumeralVector.direct, false))
			serviceMust(t, full.upm.reset(&output))
			full.upm.freeBacking(&output)
			serviceMust(t, full.cleanupFactorNumeralVector(&full.factorNumeralVector.direct, true))
			serviceMust(t, full.core.DecRef(p))
			serviceMust(t, fs.CloseChecked())
			serviceMust(t, q.z.Del(&a))
			serviceMust(t, q.z.Del(&callbackOut))
			serviceMust(t, q.z.Del(&callbackInput))
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
