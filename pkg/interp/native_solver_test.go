package interp

import (
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestNativeSolverInterpolation(t *testing.T) {
	for _, arithmetic := range []bool{false, true} {
		name := "boolean"
		if arithmetic {
			name = "integer"
		}
		t.Run(name, func(t *testing.T) {
			m, w := interpRewritingManagerFixture(t)
			factory, err := smtcore.NewSMTSolverFactory(smtcore.SMTSolverConfig{Params: w.context, Globals: w.globals})
			interpMust(t, err)
			defer factory.Close()
			s, err := factory.New(m, nil, true, true, false, "")
			interpMust(t, err)
			defer s.Close()
			boolean, err := w.BoolType()
			interpMust(t, err)
			x, err := w.MakeVariable("native-p", boolean)
			interpMust(t, err)
			x = interpPin(t, m, x)
			a := x
			b, err := w.MkNot(x)
			interpMust(t, err)
			if arithmetic {
				integer, e := w.IntType()
				interpMust(t, e)
				x, e = w.MakeVariable("native-x", integer)
				interpMust(t, e)
				x = interpPin(t, m, x)
				zero, e := w.MakeIntText("0")
				interpMust(t, e)
				zero = interpPin(t, m, zero)
				two, e := w.MakeIntText("2")
				interpMust(t, e)
				two = interpPin(t, m, two)
				a, err = w.Make(OperatorLT, x, zero)
				interpMust(t, err)
				b, err = w.Make(OperatorGT, x, two)
				interpMust(t, err)
			}
			a = interpPin(t, m, a)
			b = interpPin(t, m, b)
			marked, err := w.Make(OperatorInterp, a)
			interpMust(t, err)
			marked = interpPin(t, m, marked)
			tree, err := w.MkAnd(marked, b)
			interpMust(t, err)
			tree = interpPin(t, m, tree)
			status, constraints, refs, mdl, err := SolveTree(w, s, tree, false, &RunConfig{SolverFactory: factory})
			interpMust(t, err)
			if status != lbool.False || len(refs) != 1 || mdl != nil {
				t.Fatalf("status %v, results %d, model %v", status, len(refs), mdl)
			}
			defer func() {
				for i := range refs {
					interpMust(t, m.DelRef(&refs[i]))
				}
			}()
			h, err := m.RefGet(refs[0])
			interpMust(t, err)
			check, err := factory.New(m, nil, true, false, false, "")
			interpMust(t, err)
			defer check.Close()
			valid, err := CheckTreeInterpolants(w, check, io.Discard, constraints, tree, []ast.ASTHandle{h})
			interpMust(t, err)
			if !valid {
				t.Fatal("invalid Craig interpolant")
			}
		})
	}
}
