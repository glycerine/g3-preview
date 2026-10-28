package polynomial

// Source: src/math/polynomial/polynomial.cpp:6577-6590;
// src/util/scoped_numeral.h.

import (
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialFactorConstantActualHelperReceipt(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.CloseChecked()) }()
	five := unumeral.NewInt(5)
	var receipt polynomialFactorConstantInvocationReceipt
	serviceMust(t, full.accumulateFactorConstantWithReceipt(fs, &five, &receipt))
	if fs.constant != unumeral.NewInt(5) || receipt.invocation == 0 {
		t.Fatal("literal MUL/SET and actual invocation")
	}
	serviceMust(t, full.flipFactorSignWithReceipt(fs, &receipt))
	if fs.constant != unumeral.NewInt(-5) || fs.localActive {
		t.Fatal("literal SET/NEG/SET and scoped DT")
	}
}

// A repeated SET into the existing cell reaches no allocation. It must not
// reserve an archive; the earlier reached allocation record remains available.
func TestPolynomialFactorConstantReusedCellPreservesPriorAllocationHistory(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := NewPolynomialFactors(full)
	serviceMust(t, e)
	defer func() { serviceMust(t, fs.CloseChecked()) }()
	var prior, next polynomialFactorConstantInvocationReceipt
	serviceMust(t, fs.setConstantWithReceipt(&full.core.integer.Two64, &prior))
	old := fs.constant
	history, ok := fs.factorConstantDiagnostic(&prior)
	if !ok {
		t.Fatal("actual completed invocation")
	}
	full.core.integer.Budget().FailAfter(0)
	e = fs.setConstantWithReceipt(&full.core.integer.Two64, &next)
	full.core.integer.Budget().FailAfter(-1)
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
			var q *unumeral.RationalManager
			var callbackOut, callbackInput unumeral.Int
			active, seen := false, false
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
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
						err = numericIntegerModularManager_copyWithFactorConstantReceipt(full.core.modular, &callbackOut, &callbackInput, &fs.constantInvocation.direct)
					case "typedMul":
						err = numericIntegerModularManager_mulWithFactorConstantReceipt(full.core.modular, &callbackInput, &callbackInput, &callbackOut, &fs.constantInvocation.direct)
					case "typedNeg":
						err = numericIntegerModularManager_negWithFactorConstantReceipt(full.core.modular, &callbackOut, &fs.constantInvocation.direct)
					case "publicSet":
						err = fs.SetConstant(&callbackInput)
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
			q, e = unumeral.NewRationalManager(unumeral.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			fs, e = NewPolynomialFactors(full)
			serviceMust(t, e)
			var a unumeral.Int
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
			cause := fs.SetConstant(&a)
			active = false
			if !seen || !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual SET returning transport", seen, cause)
			}
			polynomialImpPhysicalAuditNamedRaw(t, full, nil, []*unumeral.Int{&a, &callbackInput, &callbackOut, &fs.constant, &fs.local}, fs.header, []umemory.Block{held})
			serviceMust(t, rt.SetMaxSize(0))
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

type factorCallbackState struct {
	closing, sourceStarted    bool
	sourceStage               uint8
	constant                  unumeral.Int
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
					var a unumeral.Int
					digits := make([]uint32, 80)
					digits[79] = 1
					serviceMust(t, z.SetLimbs(&a, digits))
					if op == "flip" {
						serviceMust(t, fs.SetConstant(&a))
					}
					var receipt polynomialFactorConstantInvocationReceipt
					z.Budget().FailAfter(prefix)
					switch op {
					case "set":
						e = fs.setConstantWithReceipt(&a, &receipt)
					case "accumulate":
						e = full.accumulateFactorConstantWithReceipt(fs, &a, &receipt)
					case "flip":
						e = full.flipFactorSignWithReceipt(fs, &receipt)
					}
					polynomialFactorsAudit(t, fs, nil, []*unumeral.Int{&a})
					if e == nil {
						success = true
						if op == "flip" {
							var expect unumeral.Int
							z.Budget().FailAfter(-1)
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
					if receipt.invocation != 0 && z.PendingAllocationObject() != (umemory.ObjectRef{}) && fs.constantInvocation.numeric.Cell != z.PendingCell {
						t.Fatal("actual pending cell not captured by concrete helper")
					}
					z.Budget().FailAfter(-1)
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
