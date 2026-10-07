// Source: src/util/mpq.h/.cpp: caller observations of actual RAW invocations.
// Licensed under the MIT license.
package util

type RationalRawInvocationState struct {
	Active      [3]bool
	Generations [3]uint64
}

func (m *RationalManager) RawInvocationState() RationalRawInvocationState {
	r := &m.RawArithmetic
	return RationalRawInvocationState{[3]bool{r.NormActive, r.AddActive, r.PowerActive}, [3]uint64{r.normGeneration, r.addGeneration, r.PowerGeneration}}
}
func (m *RationalManager) CleanupRawInvocations(saved *[3]uint64) error {
	state := m.RawInvocationState()
	groups := [3]rationalRawGroup{rationalRawNormalize, rationalRawAddition, RationalRawPower}
	for i, g := range *saved {
		if g != 0 && g == state.Generations[i] {
			if e := m.CleanupRawGroup(groups[i]); e != nil {
				return e
			}
		}
		saved[i] = 0
	}
	return nil
}
