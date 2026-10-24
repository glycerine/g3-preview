package ackermannization_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
)

func ackServices(t *testing.T, c *cmd_context.Context) model.ModelEvaluatorConfig {
	t.Helper()
	m, err := c.Manager()
	ackMust(t, err)
	g, err := c.GlobalParameters()
	ackMust(t, err)
	coop, err := cooperate.NewCooperation(m.Budget())
	ackMust(t, err)
	t.Cleanup(func() { ackMust(t, coop.Close()) })
	return model.ModelEvaluatorConfig{Context: g.Context, Globals: g, Memory: m.NumeralService().RuntimeOwner(), Cooperation: coop}
}

func TestAckermannModelAllocationFrontiers(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		completed := false
		for frontier := int64(0); frontier < 180 && !completed; frontier++ {
			t.Run(fmt.Sprintf("fixed=%v/%d", fixed, frontier), func(t *testing.T) {
				c := ackContext(t, `(set-logic ALL)(declare-const x Bool)
(declare-const a Bool)(declare-fun f (Bool) Bool)`)
				m, err := c.Manager()
				ackMust(t, err)
				services := ackServices(t, c)
				x, err := c.MakeApp("x", nil, nil, ast.ASTHandle{})
				ackMust(t, err)
				a, err := c.MakeApp("a", nil, nil, ast.ASTHandle{})
				ackMust(t, err)
				fx, err := c.MakeApp("f", []ast.ASTHandle{x}, nil, ast.ASTHandle{})
				ackMust(t, err)
				info, err := ackermannization.NewInfo(m)
				ackMust(t, err)
				ackMust(t, info.IncRef())
				t.Cleanup(func() { ackMust(t, info.DecRef()) })
				ackMust(t, info.SetAbstraction(fx, a))
				ackMust(t, info.Seal())
				md, err := model.NewModel(m)
				ackMust(t, err)
				var source, empty tactic.ModelRef
				ackMust(t, source.Set(md.Core()))
				t.Cleanup(func() { ackMust(t, source.Close()); ackMust(t, empty.Close()) })
				ad, err := m.GetDecl(a)
				ackMust(t, err)
				yes, err := m.MkTrue()
				ackMust(t, err)
				ackMust(t, md.Core().RegisterConstant(ad, yes))
				var converter *ackermannization.ModelConverter
				out := &source
				if fixed {
					converter, err = ackermannization.NewFixedModelConverter(m, info, md.Core(), services)
					out = &empty
				} else {
					converter, err = ackermannization.NewModelConverter(m, info, services)
				}
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, converter.Close()) })
				m.Budget().FailAfter(frontier)
				err = converter.ApplyModel(out, 0)
				m.Budget().FailAfter(-1)
				if err == nil {
					completed = true
					return
				}
				if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatalf("unexpected refusal: %v", err)
				}
			})
		}
		if !completed {
			t.Fatal("model allocation campaign never reached success")
		}
	}
}

// Source convert_constants deliberately revisits the growing source constant
// table: completion while evaluating arguments can insert more constants.
// Duplicate argument tuples keep the first entry even for inconsistent input.
func TestAckermannModelCompletionAndDuplicateEntries(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "fixed"}[fixed], func(t *testing.T) {
			c := ackContext(t, `(set-logic ALL)
(declare-const x Bool)(declare-const y Bool)
(declare-const a Bool)(declare-const b Bool)
(declare-fun f (Bool) Bool)(declare-fun h (Bool) Bool)`)
			m, err := c.Manager()
			ackMust(t, err)
			services := ackServices(t, c)
			app := func(name string, args ...ast.ASTHandle) ast.ASTHandle {
				h, err := c.MakeApp(name, args, nil, ast.ASTHandle{})
				ackMust(t, err)
				ackMust(t, m.IncRef(h))
				t.Cleanup(func() { ackMust(t, m.DecRef(h)) })
				return h
			}
			x, y, a, b := app("x"), app("y"), app("a"), app("b")
			fx, fy, hx := app("f", x), app("f", y), app("h", x)
			info, err := ackermannization.NewInfo(m)
			ackMust(t, err)
			ackMust(t, info.IncRef())
			t.Cleanup(func() { ackMust(t, info.DecRef()) })
			ackMust(t, info.SetAbstraction(fx, a))
			ackMust(t, info.SetAbstraction(fy, b))
			ackMust(t, info.Seal())
			md, err := model.NewModel(m)
			ackMust(t, err)
			var source tactic.ModelRef
			ackMust(t, source.Set(md.Core()))
			t.Cleanup(func() { ackMust(t, source.Close()) })
			yes, err := m.MkTrue()
			ackMust(t, err)
			no, err := m.MkFalse()
			ackMust(t, err)
			ad, err := m.GetDecl(a)
			ackMust(t, err)
			bd, err := m.GetDecl(b)
			ackMust(t, err)
			ackMust(t, md.Core().RegisterConstant(ad, yes))
			ackMust(t, md.Core().RegisterConstant(bd, no))
			hd, err := m.GetDecl(hx)
			ackMust(t, err)
			hi, err := model.NewFunctionInterpretation(m, 1)
			ackMust(t, err)
			ackMust(t, hi.SetElse(yes))
			ackMust(t, md.Core().RegisterFunction(hd, hi))
			var converter *ackermannization.ModelConverter
			if fixed {
				converter, err = ackermannization.NewFixedModelConverter(m, info, md.Core(), services)
			} else {
				converter, err = ackermannization.NewModelConverter(m, info, services)
			}
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, converter.Close()) })
			// Translate before completion mutates the source model. The fixed
			// converter translates its owned abstract model; the ordinary one
			// receives a separately translated model from its caller.
			to, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, to.Close()) })
			tr, err := ast.NewASTTranslation(m, to, true)
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, tr.Close()) })
			translated, err := converter.Translate(tr)
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, translated.Close()) })
			var translatedOut tactic.ModelRef
			t.Cleanup(func() { ackMust(t, translatedOut.Close()) })
			if !fixed {
				translatedModel, err := md.Translate(tr)
				ackMust(t, err)
				ackMust(t, translatedOut.Set(translatedModel.Core()))
			}
			ackMust(t, translated.ApplyModel(&translatedOut, 0))
			translatedResult, err := translatedOut.Get()
			ackMust(t, err)
			originalDecl, err := m.GetDecl(fx)
			ackMust(t, err)
			translatedDecl, err := tr.Translate(originalDecl)
			ackMust(t, err)
			translatedFunction, err := translatedResult.FunctionInterpretation(translatedDecl)
			ackMust(t, err)
			if translatedResult.Manager() != to || translatedFunction == nil || translatedFunction.NumEntries() != 1 {
				t.Fatal("translated converter did not reconstruct the target function")
			}
			out := &source
			if fixed {
				out = &tactic.ModelRef{}
				t.Cleanup(func() { ackMust(t, out.Close()) })
			}
			ackMust(t, converter.ApplyUnindexed(out))
			result, err := out.Get()
			ackMust(t, err)
			for _, term := range []ast.ASTHandle{x, y} {
				d, err := m.GetDecl(term)
				ackMust(t, err)
				value, err := result.ConstantInterpretation(d)
				ackMust(t, err)
				if value != no {
					t.Fatalf("completed argument = %v, want false", value)
				}
			}
			for _, d := range []ast.ASTHandle{ad, bd} {
				found, err := result.HasConstantInterpretation(d)
				ackMust(t, err)
				if found {
					t.Fatal("abstraction constant escaped")
				}
			}
			fd, err := m.GetDecl(fx)
			ackMust(t, err)
			fi, err := result.FunctionInterpretation(fd)
			ackMust(t, err)
			if fi == nil || fi.NumEntries() != 1 {
				t.Fatal("duplicate tuple was not merged")
			}
			entry, err := fi.FindEntry([]ast.ASTHandle{no})
			ackMust(t, err)
			if entry == nil {
				t.Fatal("missing reconstructed entry")
			}
			value, err := entry.Result()
			ackMust(t, err)
			if value != yes {
				t.Fatal("first duplicate entry was not retained")
			}
			otherwise, err := fi.Else()
			ackMust(t, err)
			if otherwise != no {
				t.Fatal("unexpected default value")
			}
			copied, err := result.FunctionInterpretation(hd)
			ackMust(t, err)
			if copied == nil || copied == hi {
				t.Fatal("existing function was not copied")
			}
			otherwise, err = copied.Else()
			ackMust(t, err)
			if otherwise != yes {
				t.Fatal("existing function lost its interpretation")
			}
		})
	}
}
