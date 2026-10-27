// Source: src/util/lp/lp_solver.h:253-255 and retained primal/dual callers.
package lp

import (
	"fmt"
	"io"
	"testing"
)

func TestLPFloatMatrixStatisticsReportBoundaries(t *testing.T) {
	const report = "extended A[1,2]\n"
	for _, primal := range []bool{true, false} {
		for _, refused := range []error{nil, lpErrResidualWrite} {
			for cut := 0; cut <= len(report)+1; cut++ {
				t.Run(fmt.Sprintf("primal=%v/error=%v/cut=%d", primal, refused, cut), func(t *testing.T) {
					w := &lpValidationPrefixWriter{larConstraintPrefixWriter: larConstraintPrefixWriter{remaining: cut, err: refused}}
					if cut > len(report) {
						w.failAt = 1
					}
					var err error
					if primal {
						m := lpFloatPrimal(t)
						m.A = NewFloatStaticMatrix(1, 1)
						m.CoreRowsToExternal[0] = 9
						m.Constraints[9] = FloatConstraint{2, RelationEqual}
						m.CoreColumnsToExternal[0] = 7
						c := m.GetOrCreateColumnInfo(7)
						c.Index = 0
						c.Cost = 17
						m.ColumnScale = []float64{1}
						m.B = []float64{2}
						m.ColumnTypes = make([]ColumnType, 2)
						m.UpperBounds = make([]float64, 2)
						m.Settings.MessageOut = w
						err = m.PrepareStageOneTotalInf()
						if m.A.ColumnCount() != 2 || len(m.X) != 2 || m.Basis[0] != 1 || m.CoreSolver != nil {
							t.Fatal("primal preparation prefix", m)
						}
						wantCost := 0.0
						if cut == len(report) {
							wantCost = 17
						}
						if m.Costs[0] != wantCost {
							t.Fatal("cost setup crossed report refusal", cut, m.Costs)
						}
					} else {
						m := lpDualSingle(t)
						m.Settings.SimplexStrategy = StrategyLU
						lpStaticMust(t, m.FillFirstStageSolverFields())
						m.CopyBAsideAndSetToZeros()
						m.Settings.MessageOut = w
						err = m.Stage1()
						if cut == len(report) {
							if m.CoreSolver == nil || m.Status != StatusFeasible {
								t.Fatal("dual success", m.Status, err)
							}
						} else if m.CoreSolver != nil {
							t.Fatal("core published before report completed")
						}
						lpStaticMust(t, m.Close())
					}
					expected := report
					wantErr := error(nil)
					flushes := 1
					if cut < len(report) {
						expected = report[:cut]
						flushes = 0
						wantErr = refused
						if wantErr == nil {
							wantErr = io.ErrShortWrite
						}
					} else if cut > len(report) {
						wantErr = lpErrResidualWrite
					}
					if err != wantErr || w.output.String() != expected || w.flushes != flushes {
						t.Fatal(cut, err, wantErr, w.output.String(), w.flushes)
					}
				})
			}
		}
	}
}

func TestLPRationalMatrixStatisticsShortWrites(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	const report = "extended A[1,3]\n"
	for _, primal := range []bool{true, false} {
		for cut := 0; cut <= len(report)+1; cut++ {
			t.Run(fmt.Sprintf("primal=%v/cut=%d", primal, cut), func(t *testing.T) {
				h := newLPPairHarness(t)
				w := &lpValidationPrefixWriter{larConstraintPrefixWriter: larConstraintPrefixWriter{remaining: cut}}
				if cut > len(report) {
					w.failAt = 1
				}
				var err error
				if primal {
					m := lpRationalPrimalOuterLarge(t, h)
					lpRationalPrimalOuterPrepare(t, h, m)
					m.Settings.MessageOut = w
					err = m.PrepareStageOneTotalInf()
					if m.A.ColumnCount() != 3 || len(m.X.Data) != 3 || m.CoreSolver != nil {
						t.Fatal("primal preparation prefix", err)
					}
					zero, e := h.m.IsZero(&m.Costs.Data[0])
					h.must(e)
					if zero != (cut != len(report)) {
						t.Fatal("exact cost setup crossed report refusal", cut, zero)
					}
					h.must(m.Close())
				} else {
					m := lpRationalDualOuterLarge(t, h, "stage1")
					m.Settings.MessageOut = w
					err = m.Stage1()
					if cut == len(report) {
						if m.CoreSolver == nil || m.Status != StatusFeasible {
							t.Fatal("dual success", m.Status, err)
						}
					} else if m.CoreSolver != nil {
						t.Fatal("core published before report completed")
					}
					h.must(m.Close())
				}
				expected := report
				wantErr := error(nil)
				flushes := 1
				if cut < len(report) {
					expected = report[:cut]
					flushes = 0
					wantErr = io.ErrShortWrite
				} else if cut > len(report) {
					wantErr = lpErrResidualWrite
				}
				if err != wantErr || w.output.String() != expected || w.flushes != flushes {
					t.Fatal(cut, err, wantErr, w.output.String(), w.flushes)
				}
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}
