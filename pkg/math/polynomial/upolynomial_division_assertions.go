package polynomial

// Source: upolynomial.cpp:404,409. Bind the actual receiving numeric owner
// only during a failed source assertion's debugger actions.
func (m *UPolynomialManager) scalarDivisionAssertionFailure(line int32, expression string) error {
	var borrow upolyFactorTraceBorrow
	if e := borrow.bindManagers([3]*UPolynomialManager{m}, m.modular.BorrowTraceTags()); e != nil {
		return e
	}
	defer borrow.unbind()
	return refinementSourceAssertionFailure(line, expression)
}
