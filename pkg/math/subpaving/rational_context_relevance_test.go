package subpaving

import "testing"

func TestRationalContextDefaultRelevantBoundSourceOrder(t *testing.T) {
	c, m, candidate, x, y := rationalClauseFixture(t)
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	check := func(variable uint32, lower, open, want bool, label string) {
		t.Helper()
		got, err := c.RelevantNewBound(root, variable, candidate, lower, open)
		if err != nil || got != want {
			value, _ := m.String(candidate)
			delta, _ := m.String(&c.relevanceScratch[2])
			cmp, cmpErr := m.Compare(candidate, &c.relevanceScratch[2])
			t.Fatalf("%s: candidate %s, threshold %s, compare %d/%v, got %v, want %v, err %v", label, value, delta, cmp, cmpErr, got, want, err)
		}
	}
	must(m.SetInt32(candidate, 1))
	must(c.PublishBound(root, x, candidate, true, false))
	check(x, true, false, false, "equal closed lower")
	check(x, true, true, false, "equal open lower is below positive epsilon")
	must(m.SetFraction(candidate, 21, 20))
	check(x, true, false, false, "epsilon equality does not improve")
	must(m.SetFraction(candidate, 53, 50))
	check(x, true, false, true, "strict epsilon improvement")
	must(m.SetInt32(candidate, 2))
	must(c.PublishBound(root, x, candidate, false, false))
	check(x, false, false, false, "equal closed upper")
	check(x, false, true, false, "equal open upper is below positive epsilon")
	must(m.SetFraction(candidate, 39, 20))
	check(x, false, false, false, "upper epsilon equality")
	must(m.SetFraction(candidate, 19, 10))
	check(x, false, false, true, "strict upper epsilon improvement")
	must(m.SetInt32(candidate, 0))
	check(x, false, false, true, "opposite lower conflict takes priority")
	must(m.SetInt32(candidate, 2))
	check(x, true, true, true, "open opposing tie is conflict before epsilon")
	must(m.SetInt64(candidate, 10_000_000_001))
	check(y, true, false, false, "new lower beyond default maximum")
	check(x, true, false, true, "opposite upper conflict before maximum")
	must(m.SetInt64(candidate, 10_000_000_000))
	check(y, true, false, true, "new lower at maximum")
	must(m.SetInt64(candidate, -10_000_000_001))
	check(y, false, false, false, "new upper below negative maximum")
	must(m.SetInt64(candidate, -10_000_000_000))
	check(y, false, false, true, "new upper at negative maximum")
	must(c.UpdateBoundPolicy(0, 1))
	must(m.SetInt32(candidate, 1))
	check(x, true, false, false, "zero epsilon equal closed lower")
	check(x, true, true, true, "zero epsilon equal open lower")
	must(m.SetFraction(candidate, 39, 20))
	check(x, false, false, true, "zero epsilon smaller upper")
	must(m.SetInt32(candidate, 11))
	check(y, true, false, false, "updated maximum bound")
	must(c.UpdateBoundPolicy(2, 2))
	must(m.SetFraction(candidate, 3, 2))
	check(x, true, false, false, "updated epsilon equality")
	must(m.SetInt32(candidate, 2))
	check(x, true, false, true, "updated epsilon strict improvement")
}
