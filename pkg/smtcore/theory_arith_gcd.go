// Source: src/smt/theory_arith_int.h gcd_test and ext_gcd_test. Eager and
// adaptive modes are excluded: their source flags have no retained setter.
package smtcore

import (
	"errors"
	"github.com/glycerine/g3/pkg/math/simplex"
	"sort"
	"strconv"
	"strings"
)

// Constants become bounds, so comparisons with the same linear
// coefficients share a source expression column even when RHSs differ.
func smtArithmeticCoefficientKey(form *smtLinearForm) string {
	columns := make([]uint32, 0, len(form.terms))
	for column := range form.terms {
		columns = append(columns, column)
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i] < columns[j] })
	var key strings.Builder
	for _, column := range columns {
		key.WriteString(strconv.FormatUint(uint64(column), 10))
		key.WriteByte(':')
		key.WriteString(form.terms[column].RatString())
		key.WriteByte(';')
	}
	return key.String()
}

func (t *SMTLinearArithmeticTheory) arithmeticIntegerMask() map[uint32]bool {
	integers := make(map[uint32]bool, len(t.simplexVariables)+len(t.atoms)+1)
	integers[0] = true
	for _, column := range t.simplexVariables {
		integers[column] = t.integer || t.integerColumns[column]
	}
	for i := range t.atoms {
		if t.atoms[i].supported {
			integers[t.nextSimplex+uint32(i)] = t.integralAtom(&t.atoms[i])
		}
	}
	return integers
}

// A failed deferred check uses a replay of current logical assertions;
// temporary candidate model values never justify a conflict.
func (t *SMTLinearArithmeticTheory) checkArithmeticGCD(solver *simplex.Solver) (bool, error) {
	if !t.gcdEnabled {
		return false, nil
	}
	result, checks, err := solver.ArithmeticGCD(t.arithmeticIntegerMask())
	t.statistics.gcdTests += checks
	if err != nil || result == simplex.ArithmeticGCDPass {
		return false, err
	}
	t.statistics.gcdConflicts++
	return true, nil
}

func (t *SMTLinearArithmeticTheory) arithmeticGCDConflict() (result error) {
	m := t.state.manager
	symbol, err := m.Symbols.Intern("gcd-test")
	if err != nil {
		return err
	}
	parameter, err := m.NewParameterSymbol(symbol)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, m.DelParameter(&parameter)) }()
	literals := make([]SMTLiteral, 0, len(t.assertions))
	for _, assertion := range t.assertions {
		literals = append(literals, smtLinearAssertionLiteral(&t.atoms[assertion.atom], assertion))
	}
	t.statistics.conflicts++
	return t.state.SetTheoryConflict(t.FamilyID(), literals, parameter)
}
