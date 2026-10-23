package smtcore

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTBitVectorTranslationOwnersModelsAndIndependentScopes(t *testing.T) {
	for _, sameManager := range []bool{false, true} {
		for _, configured := range []bool{false, true} {
			t.Run(fmt.Sprint(sameManager, "/", configured), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				target, targetCfg := m, cfg
				if !sameManager {
					target, targetCfg = smtModelGeneratorFixture(t)
				}
				params, err := cfg.Context.NewValue()
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&params)) })
				name := "auto_config"
				astTestMust(t, cfg.Context.SetBoolName(&params, &name, false))
				name = "bv.reflect"
				astTestMust(t, cfg.Context.SetBoolName(&params, &name, false))
				config := SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg, TargetConfig: func(manager *ast.ASTManager) (SMTSolverConfig, error) {
					if manager != target {
						return SMTSolverConfig{}, ast.ErrASTHandle
					}
					return SMTSolverConfig{Params: targetCfg.Context, ModelEvaluator: targetCfg}, nil
				}}
				s, err := NewSMTSolver(m, &params, "QF_BV", config)
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, s.Close()) })
				astTestMust(t, s.SetProduceModels(true))
				family, err := m.RegisterBitVectorPlugin()
				astTestMust(t, err)
				ten, err := m.BitVectorNumeralUint64(10, 4)
				astTestMust(t, err)
				sort, err := m.GetSort(ten)
				astTestMust(t, err)
				x, err := m.MkConst(astTestSymbol(t, m, "bv-clone-x"), sort)
				astTestMust(t, err)
				y, err := m.MkConst(astTestSymbol(t, m, "bv-clone-y"), sort)
				astTestMust(t, err)
				sum, err := m.MkBitVectorAdd(x, y)
				astTestMust(t, err)
				difference, err := m.MkBitVectorSub(x, y)
				astTestMust(t, err)
				four, err := m.BitVectorNumeralUint64(4, 4)
				astTestMust(t, err)
				for _, term := range []ast.ASTHandle{x, y, sum, difference, ten, four} {
					astTestPin(t, m, term)
				}
				for _, pair := range [][2]ast.ASTHandle{{sum, ten}, {difference, four}} {
					eq, err := m.MkEq(pair[0], pair[1])
					astTestMust(t, err)
					astTestMust(t, s.Assert(eq))
				}
				original := s.kernel.internalizer.theories[family].(*SMTBitVectorTheory)
				if configured {
					status, err := s.CheckSat(nil)
					astTestMust(t, err)
					if status != lbool.True || len(original.bits) == 0 {
						t.Fatal("symbolic source BV circuit was not solved", status)
					}
				}
				value, err := s.Translate(target, nil)
				astTestMust(t, err)
				translated := value.(*SMTSolver)
				t.Cleanup(func() { astTestMust(t, translated.Close()) })
				status, err := translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("copied symbolic BV circuit", status)
				}
				targetFamily, err := target.RegisterBitVectorPlugin()
				astTestMust(t, err)
				copied := translated.kernel.internalizer.theories[targetFamily].(*SMTBitVectorTheory)
				if copied == original || copied.SMTTheoryBase == original.SMTTheoryBase {
					t.Fatal("translation shared its source BV owner")
				}
				if configured {
					if copied.config != original.config || copied.bitBlasterParams != original.bitBlasterParams || translated.kernel.copiedBVOwner != &s.kernel.smtParams.bv || translated.kernel.copiedBitBlasterOwner != &s.kernel.smtParams.preprocess.bit {
						t.Fatal("BV mk_fresh did not preserve its two source parameter owners")
					}
					if copied.config.Reflect || copied.bitBlasterParams.ExtGates {
						t.Fatal("destination automatic setup overwrote source BV parameters")
					}
				} else if translated.kernel.copiedBVOwner != nil || copied.config.CC || !copied.config.Reflect || !copied.bitBlasterParams.ExtGates {
					t.Fatal("unconfigured placeholder suppressed destination BV setup")
				}
				targetX, targetY, targetSum := x, y, sum
				if !sameManager {
					translation, err := ast.NewASTTranslation(m, target, false)
					astTestMust(t, err)
					t.Cleanup(func() { astTestMust(t, translation.Close()) })
					targetX, err = translation.Translate(x)
					astTestMust(t, err)
					targetY, err = translation.Translate(y)
					astTestMust(t, err)
					targetSum, err = translation.Translate(sum)
					astTestMust(t, err)
				}
				modelCheck := func() {
					generated, err := translated.AcquireModel()
					astTestMust(t, err)
					values := [2]uint64{}
					for i, term := range []ast.ASTHandle{targetX, targetY} {
						decl, err := target.GetDecl(term)
						astTestMust(t, err)
						value, err := generated.ConstantInterpretation(decl)
						astTestMust(t, err)
						text, matched, err := target.BitVectorNumeralText(value)
						astTestMust(t, err)
						if !matched {
							t.Fatal("missing translated BV numeral")
						}
						values[i], err = strconv.ParseUint(text, 10, 64)
						astTestMust(t, err)
					}
					astTestMust(t, generated.DecRef())
					if (values[0]+values[1])&15 != 10 || (values[0]-values[1])&15 != 4 {
						t.Fatal("copied model violates modular arithmetic", values)
					}
				}
				modelCheck()
				oldStats := original.statistics
				astTestMust(t, translated.Push())
				eleven, err := target.BitVectorNumeralUint64(11, 4)
				astTestMust(t, err)
				conflict, err := target.MkEq(targetSum, eleven)
				astTestMust(t, err)
				astTestMust(t, translated.Assert(conflict))
				status, err = translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.False {
					t.Fatal("translated BV child contradiction", status)
				}
				astTestMust(t, translated.Pop(1))
				status, err = translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True || original.statistics != oldStats {
					t.Fatal("copied BV scopes affected their source", status)
				}
				modelCheck()
				if !configured {
					return
				}
				name = "bv.reflect"
				astTestMust(t, cfg.Context.SetBoolName(&params, &name, true))
				astTestMust(t, s.UpdateParams(&params))
				status, err = translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True || !copied.config.Reflect {
					t.Fatal("live source BV parameter update was lost", status)
				}
				astTestMust(t, translated.UpdateParams(nil))
				status, err = translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True || copied.config != original.config || copied.bitBlasterParams != original.bitBlasterParams {
					t.Fatal("destination update replaced borrowed BV configuration", status)
				}
				astTestMust(t, s.Close())
				status, err = translated.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True {
					t.Fatal("copied BV owner depended on the source solver lifetime", status)
				}
				modelCheck()
				value, err = translated.Translate(target, nil)
				astTestMust(t, err)
				again := value.(*SMTSolver)
				t.Cleanup(func() { astTestMust(t, again.Close()) })
				status, err = again.CheckSat(nil)
				astTestMust(t, err)
				if status != lbool.True || again.kernel.copiedBVOwner != translated.kernel.copiedBVOwner || again.kernel.copiedBitBlasterOwner != translated.kernel.copiedBitBlasterOwner {
					t.Fatal("chained translation lost its original BV parameter owners", status)
				}
			})
		}
	}
}
