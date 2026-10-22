package ast

import "errors"

// MkModelValue creates the source model-value constant with the given index.
func (m *ASTManager) MkModelValue(index uint32, sort ASTHandle) (result ASTHandle, err error) {
	p0, err := m.NewParameterUint(index)
	if err != nil {
		return ASTHandle{}, err
	}
	defer func() { err = errors.Join(err, m.DelParameter(&p0)) }()
	p1, err := m.NewParameterAST(sort)
	if err != nil {
		return ASTHandle{}, err
	}
	defer func() { err = errors.Join(err, m.DelParameter(&p1)) }()
	return m.MkFamilyApp(3, 0, []ASTParameter{p0, p1}, nil)
}
