package lp

import "testing"

func TestLPSourceEnumsAndSpellings(t *testing.T) {
	columns := []string{"free_column", "low_bound", "upper_bound", "boxed", "fixed"}
	for i, want := range columns {
		got, err := ColumnTypeString(ColumnType(i))
		if err != nil || got != want {
			t.Fatal(i, got, err)
		}
	}
	if StrategyTableauRows != 0 || StrategyTableauCosts != 1 || StrategyLU != 2 || StrategyUndecided != 3 {
		t.Fatal("strategy source values")
	}
	statuses := []string{"UNKNOWN", "INFEASIBLE", "", "UNBOUNDED", "TENTATIVE_DUAL_UNBOUNDED", "DUAL_UNBOUNDED", "OPTIMAL", "FEASIBLE", "FLOATING_POINT_ERROR", "TIME_EXHAUSTED", "ITERATIONS_EXHAUSTED", "EMPTY", "UNSTABLE", ""}
	for i, want := range statuses {
		got, err := StatusString(Status(i))
		if want == "" {
			if got != "UNKNOWN" || (err == ErrAssertion) != sourceDebugTests {
				t.Fatal(i, got, err)
			}
		} else if got != want || err != nil {
			t.Fatal(i, got, err)
		}
	}
	for _, s := range []Status{StatusUnknown, StatusInfeasible, StatusUnbounded, StatusOptimal, StatusFeasible, StatusFloatingPointError, StatusTimeExhausted, StatusIterationsExhausted, StatusEmpty} {
		got, err := ParseStatus(statuses[s])
		if got != s || err != nil {
			t.Fatal(s, got, err)
		}
	}
}

func TestLPSourceStatusFallbackAndParserAsymmetry(t *testing.T) {
	for _, s := range []string{"", "unknown", " UNKNOWN", "UNKNOWN\x00", "TENTATIVE_UNBOUNDED", "TENTATIVE_DUAL_UNBOUNDED", "DUAL_UNBOUNDED", "UNSTABLE", "CANCELLED"} {
		got, err := ParseStatus(s)
		if got != StatusUnknown || (err == ErrAssertion) != sourceDebugTests {
			t.Fatal(s, got, err)
		}
	}
	for _, c := range []ColumnType{-1, 5} {
		got, err := ColumnTypeString(c)
		if got != "unknown" || (err == ErrAssertion) != sourceDebugTests {
			t.Fatal(c, got, err)
		}
	}
	got, err := StatusString(Status(-1))
	if got != "UNKNOWN" || (err == ErrAssertion) != sourceDebugTests {
		t.Fatal(got, err)
	}
}
