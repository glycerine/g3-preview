// Source: src/sat/tactic/sat_tactic.cpp:89-112.
// Licensed under the MIT license. Source author: Leonardo de Moura (2011).
package g3

import "errors"

var ErrSATBooleanModelInterpreted = errors.New("sat: interpreted atoms require source sat2goal recovery")

// Original SAT-tactic model registration fragment. Its caller supplies the
// SAT model after a successful check. It does not perform a check, infer a
// result, evaluate formulas or replace the unported tactic/model converters.
func buildSATBooleanModel(tr *satGoalTranslator, values satModel) (md *modelCore, err error) {
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
	md, err = newModelCore(tr.manager)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if cleanup := md.Close(); cleanup != nil {
				err = errors.Join(err, cleanup)
			} else {
				md = nil
			}
		}
	}()
	rows := satSolverModelSpan{slice: values}
	for _, entry := range tr.atoms.cells {
		if !entry.used {
			continue
		}
		value, e := rows.read(satBoolVar(entry.value))
		if e != nil {
			return md, e
		}
		var interpretation ASTHandle
		switch value {
		case lTrue:
			interpretation, e = tr.manager.MkTrue()
		case lFalse:
			interpretation, e = tr.manager.MkFalse()
		default:
			continue
		}
		if e != nil {
			return md, e
		}
		declaration, e := tr.manager.GetDecl(entry.key)
		if e != nil {
			return md, e
		}
		if e = md.RegisterConstant(declaration, interpretation); e != nil {
			return md, e
		}
	}
	return md, nil
}
