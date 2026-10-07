// Package testoracle provides small independent mathematical oracles for tests.
// It must not import g3 or its production SAT algorithms.
package testoracle

import "fmt"

// Literal uses signed, one-based variable numbers. Positive means the variable,
// negative means its negation. Zero is invalid. This is an oracle representation,
// independent of the production solver's literal encoding.
type Literal int32

// CNF is a conjunction of disjunctive clauses over Variables Boolean variables.
// An empty clause is false; an empty conjunction is true. Inputs are borrowed
// for the duration of a query and are never modified or retained.
type CNF struct {
	Variables int
	Clauses   [][]Literal
}

// MaxVariables bounds complete truth-table enumeration to 65536 assignments.
const MaxVariables = 16

// Result is exact for valid bounded inputs: no unknown result is produced.
// A SAT model has Variables entries, with variable k at index k-1. Model belongs
// to the caller and is independent of future queries. UNSAT has a nil model.
type Result struct {
	Satisfiable bool
	Model       []bool
}

// InputError means the oracle cannot answer the supplied malformed or oversized
// query. It is distinct from UNSAT and must never be treated as solver unknown.
type InputError struct{ Reason string }

func (e *InputError) Error() string { return "SAT oracle: " + e.Reason }

func validate(cnf CNF, assumptions []Literal) error {
	if cnf.Variables < 0 || cnf.Variables > MaxVariables {
		return &InputError{fmt.Sprintf("variable count %d outside [0,%d]", cnf.Variables, MaxVariables)}
	}
	check := func(lit Literal) error {
		// Widen before negation so the minimum signed literal cannot overflow.
		v := int64(lit)
		if v < 0 {
			v = -v
		}
		if v == 0 || v > int64(cnf.Variables) {
			return &InputError{fmt.Sprintf("literal %d outside variable domain", lit)}
		}
		return nil
	}
	for _, clause := range cnf.Clauses {
		for _, lit := range clause {
			if err := check(lit); err != nil {
				return err
			}
		}
	}
	for _, lit := range assumptions {
		if err := check(lit); err != nil {
			return err
		}
	}
	return nil
}

func literalValue(lit Literal, assignment uint32) bool {
	v := int64(lit)
	if v < 0 {
		v = -v
	}
	value := assignment&(uint32(1)<<uint(v-1)) != 0
	if lit < 0 {
		return !value
	}
	return value
}

func satisfies(cnf CNF, assumptions []Literal, assignment uint32) bool {
	for _, lit := range assumptions {
		if !literalValue(lit, assignment) {
			return false
		}
	}
	for _, clause := range cnf.Clauses {
		holds := false
		for _, lit := range clause {
			if literalValue(lit, assignment) {
				holds = true
				break
			}
		}
		if !holds {
			return false
		}
	}
	return true
}

// Solve exhaustively enumerates every assignment in ascending binary order.
// Worst-case work is 2^Variables times the number of input literal occurrences
// plus assumptions. It has no propagation, learning, solver heuristics, random
// choices, or dependency on the source solver's algorithms. Assumptions apply
// only to this query. All input is checked before any semantic result is returned.
func Solve(cnf CNF, assumptions []Literal) (Result, error) {
	if err := validate(cnf, assumptions); err != nil {
		return Result{}, err
	}
	limit := uint32(1) << uint(cnf.Variables)
	for assignment := uint32(0); assignment < limit; assignment++ {
		if satisfies(cnf, assumptions, assignment) {
			model := make([]bool, cnf.Variables)
			for v := range model {
				model[v] = assignment&(uint32(1)<<uint(v)) != 0
			}
			return Result{Satisfiable: true, Model: model}, nil
		}
	}
	return Result{}, nil
}

// ValidModel checks a supplied total assignment against every clause and
// assumption. A well-formed counterexample returns false,nil; malformed input
// or a model with the wrong number of entries returns an explicit InputError.
func ValidModel(cnf CNF, assumptions []Literal, model []bool) (bool, error) {
	if err := validate(cnf, assumptions); err != nil {
		return false, err
	}
	if len(model) != cnf.Variables {
		return false, &InputError{fmt.Sprintf("model length %d differs from variable count %d", len(model), cnf.Variables)}
	}
	var assignment uint32
	for v, value := range model {
		if value {
			assignment |= uint32(1) << uint(v)
		}
	}
	return satisfies(cnf, assumptions, assignment), nil
}
