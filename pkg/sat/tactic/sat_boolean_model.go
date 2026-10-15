// Source: src/sat/tactic/sat_tactic.cpp:89-112.
package tactic

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/sat"
)

var ErrSATBooleanModelInterpreted = errors.New("sat: interpreted atoms require source sat2goal recovery")

// Original SAT-tactic model registration fragment. Its caller supplies the
// SAT model after a successful check. It does not perform a check, infer a
// result, evaluate formulas or replace the unported tactic/model converters.
func BuildBooleanModel(tr *GoalTranslator, values sat.Model) (md *model.ModelCore, err error) {
	if err = tr.open(); err != nil {
		return nil, err
	}
	interpreted, err := tr.HasInterpretedAtoms()
	if err != nil {
		return nil, err
	}
	if interpreted {
		return nil, ErrSATBooleanModelInterpreted
	}
	derived, err := model.NewSATBooleanModel(tr.manager)
	if err != nil {
		return nil, err
	}
	md = derived.Core()
	defer func() {
		if err != nil {
			if cleanup := md.Close(); cleanup != nil {
				err = errors.Join(err, cleanup)
			} else {
				md = nil
			}
		}
	}()
	for h, value := range tr.Atoms.All() {
		value, e := values.Read(sat.BoolVar(value))
		if e != nil {
			return md, e
		}
		var interpretation ast.ASTHandle
		switch value {
		case lbool.True:
			interpretation, e = tr.manager.MkTrue()
		case lbool.False:
			interpretation, e = tr.manager.MkFalse()
		default:
			continue
		}
		if e != nil {
			return md, e
		}
		declaration, e := tr.manager.GetDecl(h)
		if e != nil {
			return md, e
		}
		if e = md.RegisterConstant(declaration, interpretation); e != nil {
			return md, e
		}
	}
	return md, nil
}
