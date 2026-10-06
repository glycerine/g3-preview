// Source: src/math/polynomial/polynomial.cpp:6577-6590;
// src/util/scoped_numeral.h. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"strconv"
	"testing"
)

func TestPolynomialFactorConstantActualHelperReceipt(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.CloseChecked()) }()
	five := NewInt(5)
	var receipt polynomialFactorConstantInvocationReceipt
	serviceMust(t, full.accumulateFactorConstantWithReceipt(fs, &five, &receipt))
	if fs.constant != NewInt(5) || receipt.invocation == 0 {
		t.Fatal("literal MUL/SET and actual invocation")
	}
	serviceMust(t, full.flipFactorSignWithReceipt(fs, &receipt))
	if fs.constant != NewInt(-5) || fs.localActive {
		t.Fatal("literal SET/NEG/SET and scoped DT")
	}
}

// A repeated SET into the existing cell reaches no allocation. It must not
// reserve an archive; the earlier reached allocation record remains charged.
func TestPolynomialFactorConstantReusedCellPreservesPriorAllocationHistory(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.CloseChecked()) }()
	var prior, next polynomialFactorConstantInvocationReceipt
	serviceMust(t, fs.setConstantWithReceipt(&full.core.integer.two64, &prior))
	old := fs.constant
	history, ok := fs.factorConstantDiagnostic(&prior)
	if !ok {
		t.Fatal("actual completed invocation")
	}
	full.core.integer.budget.FailAfter(0)
	e = fs.setConstantWithReceipt(&full.core.integer.two64, &next)
	full.core.integer.budget.FailAfter(-1)
	if e != nil || next.invocation == 0 || fs.constant != old {
		t.Fatal("source reused-cell SET must acquire no archive reservation", e)
	}
	after, ok := fs.factorConstantDiagnostic(&prior)
	if !ok || after.invocation != history.invocation || after.bodyCause != history.bodyCause || after.sourceCause != history.sourceCause {
		t.Fatal("prior actual tuple changed")
	}
	polynomialFactorsAudit(t, fs, nil, nil)
}

// Actual returning Go ErrorTransport during the reached SET allocating leaf.
func TestPolynomialFactorConstantTransportCannotRetireExecutingOwner(t *testing.T) {
	for _, operation := range []string{"source", "close", "cleanup", "fullSource", "fullClose", "coreSource", "coreClose", "integerSource", "integerClose", "rationalSource", "rationalClose", "typedCopy", "typedMul", "typedNeg", "publicSet"} {
		t.Run(operation, func(t *testing.T) {
			b := arena.NewBudget(1 << 27)
			var full *MultivariatePolynomialManager
			var fs *PolynomialFactors
			var q *RationalManager
			var callbackOut, callbackInput Int
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
						err = full.core.modular.copyWithFactorConstantReceipt(&callbackOut, &callbackInput, &fs.constantInvocation.direct)
					case "typedMul":
						err = full.core.modular.mulWithFactorConstantReceipt(&callbackInput, &callbackInput, &callbackOut, &fs.constantInvocation.direct)
					case "typedNeg":
						err = full.core.modular.negWithFactorConstantReceipt(&callbackOut, &fs.constantInvocation.direct)
					case "publicSet":
						err = fs.SetConstant(&callbackInput)
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
			pad := int64(99996) - rt.State().LocalSize - 8
			if pad <= 0 {
				t.Fatal("actual TLS frontier", rt.State())
			}
			held, e := rt.Allocate(uint64(pad))
			serviceMust(t, e)
			serviceMust(t, rt.SetMaxSize(1))
			active = true
			cause := fs.SetConstant(&a)
			active = false
			if !seen || !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual SET returning transport", seen, cause)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, nil, []*Int{&a, &callbackInput, &callbackOut, &fs.constant, &fs.local}, fs.header+uint64(cap(fs.constantHistory))*uint64(reflect.TypeOf(polynomialFactorConstantHistory{}).Size()), []memory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
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

type factorCallbackState struct {
	closing, sourceStarted    bool
	sourceStage               uint8
	constant                  Int
	invocation                uint64
	sourceCause, cleanupCause error
}

func rtStateForFactors(f *PolynomialFactors) factorCallbackState {
	return factorCallbackState{f.closing, f.sourceStarted, f.sourceStage, f.constant, f.constantInvocation.invocation, f.sourceCause, f.cleanupCause}
}

func TestPolynomialFactorConstantEveryActualReservationPrefix(t *testing.T) {
	for _, op := range []string{"set", "accumulate", "flip"} {
		t.Run(op, func(t *testing.T) {
			success := false
			for prefix := int64(0); prefix < 96; prefix++ {
				if success {
					break
				}
				t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
					full := polynomialPSCFullFixture(t)
					fs, e := NewPolynomialFactors(full)
					serviceMust(t, e)
					z := full.core.integer
					var a Int
					digits := make([]uint32, 80)
					digits[79] = 1
					serviceMust(t, z.SetLimbs(&a, digits))
					if op == "flip" {
						serviceMust(t, fs.SetConstant(&a))
					}
					var receipt polynomialFactorConstantInvocationReceipt
					z.budget.FailAfter(prefix)
					switch op {
					case "set":
						e = fs.setConstantWithReceipt(&a, &receipt)
					case "accumulate":
						e = full.accumulateFactorConstantWithReceipt(fs, &a, &receipt)
					case "flip":
						e = full.flipFactorSignWithReceipt(fs, &receipt)
					}
					polynomialFactorsAudit(t, fs, nil, []*Int{&a})
					if e == nil {
						success = true
						if op == "flip" {
							var expect Int
							z.budget.FailAfter(-1)
							serviceMust(t, z.Copy(&expect, &a))
							serviceMust(t, z.Neg(&expect))
							equal, err := z.Equal(&fs.constant, &expect)
							serviceMust(t, err)
							if !equal {
								t.Fatal("literal sign result")
							}
							serviceMust(t, z.Del(&expect))
						} else {
							equal, err := z.Equal(&fs.constant, &a)
							serviceMust(t, err)
							if !equal {
								t.Fatal("literal SET/MUL result")
							}
						}
					} else if !numericPhysicalReservationRefusal(e) {
						t.Fatal("actual leaf refusal", e)
					}
					if receipt.invocation != 0 && z.pendingCell.object != (memory.ObjectRef{}) && fs.constantInvocation.numeric.cell != z.pendingCell {
						t.Fatal("actual pending cell not captured by concrete helper")
					}
					z.budget.FailAfter(-1)
					serviceMust(t, fs.Cleanup())
					if receipt.invocation != 0 {
						h, ok := fs.factorConstantDiagnostic(&receipt)
						if !ok || h.bodyCause == nil && e != nil {
							t.Fatal("actual body history lost")
						}
					}
					serviceMust(t, fs.CloseChecked())
					serviceMust(t, z.Del(&a))
				})
			}
			if !success {
				t.Fatal("no real first-success horizon")
			}
		})
	}
}
