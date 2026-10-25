package smtcore

import (
	"fmt"
	"testing"
)

func TestSMTDatatypeRestartAdvancesIndependentUtility(t *testing.T) {
	for _, used := range []bool{false, true} {
		t.Run(fmt.Sprint(used), func(t *testing.T) {
			m, dt, x, sort, ctors := smtDatatypeSplitFixture(t)
			state, err := dt.State()
			astTestMust(t, err)
			pluginChoice, found, err := m.DatatypeNonRecursiveConstructor(sort)
			astTestMust(t, err)
			if !found {
				t.Fatal("missing AST plugin preference")
			}
			if used {
				astTestMust(t, dt.split(x, sort))
			}
			owner := dt.utility
			variables, splits := dt.NumVariables(), dt.statistics.splits
			astTestMust(t, state.RestartTheories())
			if dt.utility == nil || used && dt.utility != owner || dt.NumVariables() != variables || dt.statistics.splits != splits {
				t.Fatal("restart replaced owners or reset theory state")
			}
			chosen, found, err := dt.utility.NonRecursiveConstructor(sort)
			astTestMust(t, err)
			expected := 2
			if used {
				expected = 0
			}
			if !found || chosen != ctors[expected] {
				t.Fatal("restart did not advance source constructor preference")
			}
			astTestMust(t, dt.split(x, sort))
			slots, err := dt.recognizerSlots(x)
			astTestMust(t, err)
			if slots[chosen].expression.IsNull() {
				t.Fatal("split ignored restarted preference")
			}
			after, found, err := m.DatatypeNonRecursiveConstructor(sort)
			astTestMust(t, err)
			if !found || after != pluginChoice {
				t.Fatal("theory restart reset another utility owner")
			}
		})
	}
}
