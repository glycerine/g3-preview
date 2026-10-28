package testoracle

// Source scenarios: src/test/check_assumptions.cpp and sat_user_scope.cpp.

import (
	"embed"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

//go:embed test_vectors/*.json
var fixtures embed.FS

type satFixture struct {
	Name        string      `json:"name"`
	Provenance  string      `json:"provenance"`
	Derivation  string      `json:"derivation"`
	Variables   int         `json:"variables"`
	Clauses     [][]Literal `json:"clauses"`
	Assumptions []Literal   `json:"assumptions"`
	SAT         bool        `json:"sat"`
}

func TestSATPermanentFixtures(t *testing.T) {
	data, err := fixtures.ReadFile("test_vectors/sat_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []satFixture
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Provenance == "" || tc.Derivation == "" {
				t.Fatal("missing oracle provenance")
			}
			cnf := CNF{Variables: tc.Variables, Clauses: tc.Clauses}
			result, err := Solve(cnf, tc.Assumptions)
			if err != nil {
				t.Fatal(err)
			}
			if result.Satisfiable != tc.SAT {
				t.Fatalf("sat=%v, want %v", result.Satisfiable, tc.SAT)
			}
			if result.Satisfiable {
				ok, err := ValidModel(cnf, tc.Assumptions, result.Model)
				if err != nil || !ok {
					t.Fatalf("invalid witness: %v %v", ok, err)
				}
			} else if result.Model != nil {
				t.Fatal("UNSAT must not carry a model")
			}
		})
	}
}

func TestSATBoundedExhaustion(t *testing.T) {
	// Use 16 unit clauses selecting the final enumerated assignment, then its blocker:
	// only all-true can satisfy the units, and the final clause excludes it.
	units := make([][]Literal, 0, 17)
	for v := 1; v <= 16; v++ {
		units = append(units, []Literal{Literal(v)})
	}
	result, err := Solve(CNF{Variables: 16, Clauses: units}, nil)
	if err != nil || !result.Satisfiable {
		t.Fatalf("bound witness %v %v", result, err)
	}
	for _, value := range result.Model {
		if !value {
			t.Fatal("units require all true")
		}
	}
	blocker := make([]Literal, 16)
	for v := range blocker {
		blocker[v] = -Literal(v + 1)
	}
	units = append(units, blocker)
	result, err = Solve(CNF{Variables: 16, Clauses: units}, nil)
	if err != nil || result.Satisfiable {
		t.Fatalf("bound UNSAT %v %v", result, err)
	}
}

func TestSATCheckedInput(t *testing.T) {
	cases := []struct {
		cnf         CNF
		assumptions []Literal
	}{
		{CNF{Variables: -1}, nil}, {CNF{Variables: 17}, nil},
		{CNF{Variables: 1, Clauses: [][]Literal{{0}}}, nil},
		{CNF{Variables: 1, Clauses: [][]Literal{{2}}}, nil},
		{CNF{Variables: 1, Clauses: [][]Literal{{-2}}}, nil},
		{CNF{Variables: 1, Clauses: [][]Literal{{-2147483648}}}, nil},
		{CNF{Variables: 1}, []Literal{0}}, {CNF{Variables: 1}, []Literal{2}},
		// Validation must inspect the whole input even when an early empty clause
		// already establishes UNSAT.
		{CNF{Variables: 1, Clauses: [][]Literal{{}, {2}}}, nil},
	}
	for i, tc := range cases {
		result, err := Solve(tc.cnf, tc.assumptions)
		if err == nil {
			t.Errorf("case %d accepted invalid input: %v", i, result)
		}
		var inputErr *InputError
		if !errors.As(err, &inputErr) || inputErr.Error() == "" {
			t.Errorf("case %d did not return a described InputError", i)
		}
		if result.Satisfiable || result.Model != nil {
			t.Error("invalid input returned a result")
		}
	}
	if _, err := ValidModel(CNF{Variables: 1}, nil, nil); err == nil {
		t.Error("short model accepted")
	}
	if _, err := ValidModel(CNF{Variables: 0}, nil, []bool{true}); err == nil {
		t.Error("long model accepted")
	}
	if _, err := ValidModel(CNF{Variables: 1, Clauses: [][]Literal{{2}}}, nil, []bool{true}); err == nil {
		t.Error("model validator accepted malformed CNF")
	}
}

func TestSATAllTwoVariableTruthTables(t *testing.T) {
	// All sixteen Boolean functions of two variables. A clause consisting of
	// literals opposite to one assignment excludes exactly that assignment.
	// This CNF construction derives expectations from a four-bit truth table;
	// witness checking below evaluates that table directly, independently of
	// Solve's and ValidModel's clause evaluators.
	for table := uint32(0); table < 16; table++ {
		var clauses [][]Literal
		for assignment := uint32(0); assignment < 4; assignment++ {
			if table&(1<<assignment) != 0 {
				continue
			}
			clause := []Literal{1, 2}
			for v := range clause {
				if assignment&(1<<v) != 0 {
					clause[v] = -clause[v]
				}
			}
			clauses = append(clauses, clause)
		}
		result, err := Solve(CNF{Variables: 2, Clauses: clauses}, nil)
		if err != nil || result.Satisfiable != (table != 0) {
			t.Fatalf("table %04b: %v %v", table, result, err)
		}
		if result.Satisfiable {
			assignment := uint32(0)
			if result.Model[0] {
				assignment |= 1
			}
			if result.Model[1] {
				assignment |= 2
			}
			if table&(1<<assignment) == 0 {
				t.Fatalf("table %04b rejects witness %v", table, result.Model)
			}
		}
	}
}

func TestSATModelValidationAndIsolation(t *testing.T) {
	cnf := CNF{Variables: 2, Clauses: [][]Literal{{1, 2}}}
	assumptions := []Literal{-1}
	wantClauses := [][]Literal{{1, 2}}
	wantAssumptions := []Literal{-1}
	result, err := Solve(cnf, assumptions)
	if err != nil || !result.Satisfiable {
		t.Fatal(result, err)
	}
	if !reflect.DeepEqual(result.Model, []bool{false, true}) {
		t.Fatal("assumption forces q", result.Model)
	}
	if !reflect.DeepEqual(cnf.Clauses, wantClauses) || !reflect.DeepEqual(assumptions, wantAssumptions) {
		t.Fatal("borrowed input mutated")
	}
	result.Model[0] = true
	if valid, err := ValidModel(cnf, assumptions, result.Model); err != nil || valid {
		t.Fatal("violated assumption accepted")
	}
	next, err := Solve(cnf, assumptions)
	if err != nil || next.Model[0] {
		t.Fatal("returned models share storage")
	}
	if valid, err := ValidModel(cnf, nil, []bool{false, false}); err != nil || valid {
		t.Fatal("violated clause accepted")
	}
}

type scopeFixture struct {
	Name       string `json:"name"`
	Derivation string `json:"derivation"`
	Variables  int    `json:"variables"`
	Steps      []struct {
		Op          string    `json:"op"`
		Clause      []Literal `json:"clause"`
		Assumptions []Literal `json:"assumptions"`
		SAT         *bool     `json:"sat"`
	} `json:"steps"`
}

func TestSATScopeScripts(t *testing.T) {
	data, err := fixtures.ReadFile("test_vectors/scope_scripts.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []scopeFixture
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Derivation == "" {
				t.Fatal("missing derivation")
			}
			// Test harness only: each frame owns copied clauses. Flatten the active
			// frames for a fresh exhaustive query. No production scope API is implied.
			frames := [][][]Literal{nil}
			for i, step := range tc.Steps {
				switch step.Op {
				case "push":
					frames = append(frames, nil)
				case "pop":
					if len(frames) == 1 {
						t.Fatalf("step %d pops base", i)
					}
					frames = frames[:len(frames)-1]
				case "add":
					frame := len(frames) - 1
					frames[frame] = append(frames[frame], append([]Literal(nil), step.Clause...))
				case "check":
					if step.SAT == nil {
						t.Fatalf("step %d missing expected status", i)
					}
					var active [][]Literal
					for _, frame := range frames {
						active = append(active, frame...)
					}
					result, err := Solve(CNF{Variables: tc.Variables, Clauses: active}, step.Assumptions)
					if err != nil || result.Satisfiable != *step.SAT {
						t.Fatalf("step %d: %v %v expected %v", i, result, err, *step.SAT)
					}
				default:
					t.Fatalf("unknown op %s", step.Op)
				}
			}
		})
	}
}
