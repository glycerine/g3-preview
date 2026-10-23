// Source: src/smt/theory_seq.cpp check_extensionality.
package smtcore

// checkSequenceExtensionality exposes one undecided interface equality between
// relevant symbolic sequence roots. The SAT case split either merges the
// roots or records their disequality before the next final-check pass.
func (t *SMTSequenceTheory) checkSequenceExtensionality() (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	variables := make([]*SMTEnode, 0, len(t.variables))
	for _, node := range t.variables {
		if node == nil || node.Root() != node || !state.isRelevant(node.Owner()) {
			continue
		}
		symbolic, variableErr := t.sequenceVariable(node.Owner())
		if variableErr != nil {
			return false, variableErr
		}
		if symbolic {
			variables = append(variables, node)
		}
	}
	for rightIndex := 1; rightIndex < len(variables); rightIndex++ {
		right := variables[rightIndex]
		rightSort, sortErr := state.manager.GetSort(right.Owner())
		if sortErr != nil {
			return false, sortErr
		}
		for leftIndex := 0; leftIndex < rightIndex; leftIndex++ {
			left := variables[leftIndex]
			leftSort, leftErr := state.manager.GetSort(left.Owner())
			if leftErr != nil {
				return false, leftErr
			}
			if leftSort != rightSort || left.Root() == right.Root() ||
				state.context.isDisequal(left, right) || t.exclusions.Contains(left.Owner(), right.Owner()) {
				continue
			}
			equality, makeErr := state.manager.MkEq(left.Owner(), right.Owner())
			if makeErr != nil {
				return false, makeErr
			}
			if makeErr = state.Internalize(equality, true); makeErr != nil {
				return false, makeErr
			}
			if makeErr = state.markRelevant(equality); makeErr != nil {
				return false, makeErr
			}
			return true, nil
		}
	}
	return false, nil
}
